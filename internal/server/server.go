package server

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/auth"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/config"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/telemetry"
	"github.com/gin-contrib/cors"
	gzipadapter "github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Server representa o servidor HTTP
type Server struct {
	cfg        *config.Config
	db         *sql.DB
	rdb        *redis.Client
	router     *gin.Engine
	httpServer *http.Server
	jwtManager *auth.JWTManager
	rateLimit  *RateLimiter
}

// RateLimiter implementa rate limiting simples em memória
type RateLimiter struct {
	mu     sync.RWMutex
	ips    map[string][]time.Time
	limit  int
	window time.Duration
}

// NewRateLimiter cria um novo rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		ips:    make(map[string][]time.Time),
		limit:  limit,
		window: window,
	}
}

// NewServer cria uma nova instância do servidor
func NewServer(cfg *config.Config, db *sql.DB, rdb *redis.Client) *Server {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// Inicializar JWT Manager com chave secreta
	jwtManager := auth.NewJWTManager(cfg.APIKey)
	if cfg.APIKey == "" {
		jwtManager = auth.NewJWTManager("transprota-secret-key-2024")
	}

	return &Server{
		cfg:        cfg,
		db:         db,
		rdb:        rdb,
		router:     r,
		jwtManager: jwtManager,
		rateLimit:  NewRateLimiter(500, time.Minute), // 500 req/min por IP
	}
}

// Allow verifica se o IP está dentro do limite
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	timestamps, exists := rl.ips[ip]

	if !exists {
		rl.ips[ip] = []time.Time{now}
		return true
	}

	// Remover timestamps antigos (fora da janela)
	var validTimestamps []time.Time
	for _, ts := range timestamps {
		if now.Sub(ts) < rl.window {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	rl.ips[ip] = validTimestamps

	// Verificar se excedeu o limite
	if len(validTimestamps) >= rl.limit {
		logger.Warn("Server", "IP %s excedeu limite (%d/%d)", ip, len(validTimestamps), rl.limit)
		return false
	}

	// Adicionar timestamp atual
	rl.ips[ip] = append(validTimestamps, now)
	return true
}

// SetupRoutes configura todas as rotas da aplicação
func (s *Server) SetupRoutes() {
	// Middleware de compressão gzip
	s.router.Use(gzipadapter.Gzip(gzipadapter.DefaultCompression))

	// Middleware de CORS - Liberação total para desenvolvimento
	s.router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	// Middleware de timeout para garantir <100ms TTFB
	s.router.Use(func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	})

	// Middleware de tratamento de erros
	s.router.Use(s.errorHandlerMiddleware())

	// Middleware de Rate Limiting
	s.router.Use(s.rateLimitMiddleware())

	// Health Check
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/api/v1/health", s.healthCheck)

	// Rotas de Telemetria
	telemetry.SetupRoutes(s.router, s.db, s.rdb, s.jwtManager)
}

// Start inicia o servidor HTTP
func (s *Server) Start() error {
	// Configurar servidor HTTP
	s.httpServer = &http.Server{
		Addr:              ":" + s.cfg.ServerPort,
		Handler:           s.router,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("Server", "Servidor iniciado na porta %s", s.cfg.ServerPort)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server", "Erro ao iniciar servidor: %v", err)
		}
	}()

	// Aguardar sinal de shutdown
	<-quit
	logger.Info("Server", "Recebido sinal de shutdown, iniciando graceful shutdown...")

	// Fechar conexões com bancos de dados antes de fechar o servidor
	if err := s.Close(); err != nil {
		logger.Error("Server", "Erro ao fechar conexões: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server", "Erro durante graceful shutdown: %v", err)
		return err
	}
	return nil
}

// Shutdown realiza graceful shutdown do servidor HTTP
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("Server", "Iniciando graceful shutdown...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}
	logger.Info("Server", "Servidor HTTP fechado gracefulmente")
	return nil
}

// Close fecha conexões e limpa recursos (deprecated, usar Shutdown)
func (s *Server) Close() error {
	var errs []error

	if s.db != nil {
		if err := s.db.Close(); err != nil {
			errs = append(errs, fmt.Errorf("db close: %w", err))
		}
	}

	if s.rdb != nil {
		if err := s.rdb.Close(); err != nil {
			logger.Error("Server", "Erro ao fechar conexão Redis: %v", err)
			return fmt.Errorf("erro ao fechar conexão Redis: %w", err)
		}
		logger.Info("Server", "Conexão Redis fechada")
	}

	return nil
}

// corsMiddleware configura CORS
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// errorHandlerMiddleware trata erros
func (s *Server) errorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors[0]
			logger.Error("Server", "Erro: %v", err)

			if !c.Writer.Written() {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "Internal server error",
					"details": err.Error(),
				})
			}
		}
	}
}

// rateLimitMiddleware implementa rate limiting simples (100 req/min por IP)
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Bypass rate limit para healthcheck
		if c.Request.URL.Path == "/health" || c.Request.URL.Path == "/api/v1/health" {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		if !s.rateLimit.Allow(clientIP) {
			logger.Warn("Server", "Bloqueando requisição de %s (limite excedido)", clientIP)
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate limit exceeded",
				"message": "Too many requests. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// healthCheck verifica a saúde do sistema
func (s *Server) healthCheck(c *gin.Context) {
	status := gin.H{
		"timestamp": time.Now().Unix(),
	}

	allHealthy := true

	// Verificar PostgreSQL
	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := s.db.PingContext(ctx); err != nil {
			status["postgres"] = "down"
			status["postgres_error"] = err.Error()
			allHealthy = false
		} else {
			status["postgres"] = "up"
		}
	} else {
		status["postgres"] = "not configured"
		allHealthy = false
	}

	// Verificar Redis
	if s.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		if err := s.rdb.Ping(ctx).Err(); err != nil {
			status["redis"] = "down"
			status["redis_error"] = err.Error()
			allHealthy = false
		} else {
			status["redis"] = "up"
		}
	} else {
		status["redis"] = "not configured"
		allHealthy = false
	}

	// Status do servidor (se endpoint responde, servidor está up)
	status["server"] = "up"

	// Se todos os serviços estiverem saudáveis, retorna 200
	if allHealthy {
		status["status"] = "healthy"
		status["message"] = "Backend Perfeito para Integração"
		c.JSON(http.StatusOK, status)
	} else {
		status["status"] = "unhealthy"
		status["message"] = "Alguns serviços não estão disponíveis"
		c.JSON(http.StatusServiceUnavailable, status)
	}
}
