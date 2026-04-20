package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware protege rotas exigindo autenticação JWT
func AuthMiddleware(jwtManager *JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		// Verificar formato "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			// Diferenciar entre token expirado e inválido
			if errors.Is(err, ErrExpiredToken) {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "expired"})
			} else {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			}
			c.Abort()
			return
		}

		// Adicionar user_id ao contexto
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Next()
	}
}
