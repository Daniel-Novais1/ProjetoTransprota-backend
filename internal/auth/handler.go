package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler gerencia endpoints de autenticação
type Handler struct {
	jwtManager *JWTManager
}

// NewHandler cria um novo handler de autenticação
func NewHandler(jwtManager *JWTManager) *Handler {
	return &Handler{
		jwtManager: jwtManager,
	}
}

// LoginRequest representa a requisição de login
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse representa a resposta de login
type LoginResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"userId"`
	Username string `json:"username"`
}

// Login autentica um usuário e retorna um token JWT
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Mock authentication - em produção use hash real
	if req.Username != "admin" || req.Password != "admin123" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Gerar token JWT
	token, err := h.jwtManager.GenerateToken("admin", req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		Token:    token,
		UserID:   "admin",
		Username: req.Username,
	})
}
