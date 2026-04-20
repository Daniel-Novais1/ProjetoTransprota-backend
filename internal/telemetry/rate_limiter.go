package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimiterMiddleware limita requisições por dispositivo usando Redis
// Limite: 1 requisição a cada 5 segundos por device_id
func RateLimiterMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		// Extrair device_id do JSON payload (se for endpoint de GPS)
		if c.Request.URL.Path == "/api/v1/telemetry/gps" {
			var body map[string]interface{}
			if err := c.ShouldBindJSON(&body); err == nil {
				if deviceID, ok := body["device_id"].(string); ok {
					// Chave do Redis para rate limiting
					redisKey := fmt.Sprintf("ratelimit:device:%s", deviceID)

					// Verificar se há requisição recente
					lastRequest, err := rdb.Get(ctx, redisKey).Result()
					if err == redis.Nil {
						// Primeira requisição ou expirou - permitir
						rdb.Set(ctx, redisKey, time.Now().Unix(), 5*time.Second)
						c.Next()
						return
					} else if err != nil {
						// Erro no Redis - log mas permitir (fail-open)
						logger.Error("RateLimiter", "Redis error, allowing request: %v", err)
						c.Next()
						return
					}

					// Converter timestamp
					lastRequestTime, _ := strconv.ParseInt(lastRequest, 10, 64)
					timeSinceLastRequest := time.Since(time.Unix(lastRequestTime, 0))

					// Se passou menos de 5 segundos, bloquear
					if timeSinceLastRequest < 5*time.Second {
						logger.Warn("RateLimiter", "Rate limit excedido para device %s (última req: %v atrás)",
							deviceID[:8]+"...", timeSinceLastRequest)

						c.JSON(http.StatusTooManyRequests, gin.H{
							"error":       "Rate limit exceeded",
							"message":     "Only 1 request per 5 seconds allowed per device",
							"retry_after": 5*time.Second - timeSinceLastRequest,
						})
						c.Abort()
						return
					}

					// Atualizar timestamp
					rdb.Set(ctx, redisKey, time.Now().Unix(), 5*time.Second)
				}
			}
		}

		c.Next()
	}
}

// IPRateLimiterMiddleware limita requisições por IP (proteção geral contra DoS)
// Limite: 100 requisições por minuto por IP
func IPRateLimiterMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.Background()

		// Extrair IP do cliente
		clientIP := c.ClientIP()
		if clientIP == "" {
			clientIP = "unknown"
		}

		// Chave do Redis para rate limiting por IP
		redisKey := fmt.Sprintf("ratelimit:ip:%s", clientIP)

		// Usar Redis INCR para contador
		count, err := rdb.Incr(ctx, redisKey).Result()
		if err == redis.Nil {
			// Primeira requisição - definir TTL de 1 minuto
			rdb.Set(ctx, redisKey, 1, 1*time.Minute)
		} else if err != nil {
			// Erro no Redis - permitir (fail-open)
			logger.Error("RateLimiter", "Redis error, allowing request: %v", err)
			c.Next()
			return
		}

		// Se contador > 100, bloquear
		if count > 100 {
			logger.Warn("RateLimiter", "IP rate limit excedido para %s (count: %d)", clientIP, count)

			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests from this IP",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
