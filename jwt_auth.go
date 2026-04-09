package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTClaims representa os claims do token JWT
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// LoginRequest representa requisição de login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse representa resposta de login
type LoginResponse struct {
	Token     string   `json:"token"`
	ExpiresIn int      `json:"expires_in"`
	User      UserInfo `json:"user"`
}

// UserInfo representa informações do usuário
type UserInfo struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

// JWTManager gerencia tokens JWT
type JWTManager struct {
	secretKey string
	issuer    string
}

// NewJWTManager cria um novo gerenciador JWT
func NewJWTManager() *JWTManager {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		secretKey = "transprota-secret-key-change-in-production"
	}
	return &JWTManager{
		secretKey: secretKey,
		issuer:    "transprota-api",
	}
}

// GenerateToken gera um novo token JWT
func (j *JWTManager) GenerateToken(userID, username, role string) (string, error) {
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}

// ValidateToken valida um token JWT
func (j *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	// Remover "Bearer " se presente
	tokenString = strings.TrimPrefix(tokenString, "Bearer ")

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("método de assinatura inválido: %v", token.Header["alg"])
		}
		return []byte(j.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("token inválido")
}

// RefreshToken renova um token JWT
func (j *JWTManager) RefreshToken(claims *JWTClaims) (string, error) {
	// Criar novos claims com mesmo usuário mas nova expiração
	newClaims := JWTClaims{
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    j.issuer,
			Subject:   claims.UserID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	return token.SignedString([]byte(j.secretKey))
}

// validateCredentials valida credenciais do usuário
func validateCredentials(username, password string) (*UserInfo, error) {
	// Credenciais admin (em produção usar banco de dados)
	adminUsername := os.Getenv("ADMIN_USERNAME")
	if adminUsername == "" {
		adminUsername = "admin"
	}
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123"
	}

	if username == adminUsername && password == adminPassword {
		return &UserInfo{
			ID:       "admin",
			Username: username,
			Role:     "admin",
		}, nil
	}

	// Para demonstração, permitir usuário demo
	if username == "demo" && password == "demo123" {
		return &UserInfo{
			ID:       "demo",
			Username: username,
			Role:     "user",
		}, nil
	}

	return nil, fmt.Errorf("credenciais inválidas")
}

// setupJWTRoutes configura rotas de autenticação JWT
func setupJWTRoutes(r *gin.Engine) {
	jwtManager := NewJWTManager()

	// POST /api/v1/auth/login - Login
	r.POST("/api/v1/auth/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
			return
		}

		user, err := validateCredentials(req.Username, req.Password)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciais inválidas"})
			return
		}

		token, err := jwtManager.GenerateToken(user.ID, user.Username, user.Role)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar token"})
			return
		}

		c.JSON(http.StatusOK, LoginResponse{
			Token:     token,
			ExpiresIn: 86400, // 24 horas em segundos
			User:      *user,
		})
	})

	// POST /api/v1/auth/refresh - Refresh token
	r.POST("/api/v1/auth/refresh", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token não fornecido"})
			return
		}

		claims, err := jwtManager.ValidateToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			return
		}

		// Verificar se token está próximo de expirar (opcional)
		if time.Until(claims.ExpiresAt.Time) > 1*time.Hour {
			// Token ainda válido por mais de 1 hora, não precisa refresh
			c.JSON(http.StatusOK, gin.H{
				"message":    "Token ainda válido",
				"expires_in": int(time.Until(claims.ExpiresAt.Time).Seconds()),
			})
			return
		}

		newToken, err := jwtManager.RefreshToken(claims)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao renovar token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token":      newToken,
			"expires_in": 86400,
		})
	})

	// GET /api/v1/auth/me - Informações do usuário atual
	r.GET("/api/v1/auth/me", JWTMiddleware(), func(c *gin.Context) {
		claims, exists := c.Get("jwt_claims")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Claims não encontrados"})
			return
		}

		jwtClaims, ok := claims.(*JWTClaims)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro nos claims"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"user": UserInfo{
				ID:       jwtClaims.UserID,
				Username: jwtClaims.Username,
				Role:     jwtClaims.Role,
			},
		})
	})

	// POST /api/v1/auth/logout - Logout (cliente-side)
	r.POST("/api/v1/auth/logout", JWTMiddleware(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Logout realizado com sucesso"})
	})
}

// JWTMiddleware cria middleware de autenticação JWT
func JWTMiddleware() gin.HandlerFunc {
	jwtManager := NewJWTManager()

	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token não fornecido"})
			c.Abort()
			return
		}

		claims, err := jwtManager.ValidateToken(authHeader)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token inválido"})
			c.Abort()
			return
		}

		// Verificar se token não expirou
		if time.Now().After(claims.ExpiresAt.Time) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token expirado"})
			c.Abort()
			return
		}

		// Adicionar claims ao contexto
		c.Set("jwt_claims", claims)
		c.Set("user_id", claims.UserID)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// AdminMiddleware cria middleware para verificar se usuário é admin
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("user_role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Usuário não autenticado"})
			c.Abort()
			return
		}

		role, ok := userRole.(string)
		if !ok || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Acesso negado - privilégios insuficientes"})
			c.Abort()
			return
		}

		c.Next()
	}
}
