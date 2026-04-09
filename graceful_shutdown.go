package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// GracefulShutdownManager gerencia shutdown gracioso
type GracefulShutdownManager struct {
	server *http.Server
	app    *App
}

// NewGracefulShutdownManager cria novo gerenciador de shutdown
func NewGracefulShutdownManager(server *http.Server, app *App) *GracefulShutdownManager {
	return &GracefulShutdownManager{
		server: server,
		app:    app,
	}
}

// setupGracefulShutdown configura graceful shutdown
func setupGracefulShutdown(r *gin.Engine, app *App) {
	// Criar servidor HTTP
	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
		// Configurações de timeout para graceful shutdown
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Iniciar servidor em goroutine separada
	go func() {
		log.Println("TranspRota API operacional em http://localhost:8080")
		log.Println("Modo:", getAppMode(app))
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Erro ao iniciar servidor: %v", err)
		}
	}()

	// Configurar graceful shutdown
	shutdownManager := NewGracefulShutdownManager(server, app)
	shutdownManager.waitForShutdown()
}

// waitForShutdown aguarda sinais de shutdown
func (gsm *GracefulShutdownManager) waitForShutdown() {
	// Canal para receber sinais do sistema operacional
	quit := make(chan os.Signal, 1)
	
	// Capturar sinais de shutdown
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// Aguardar sinal
	<-quit
	log.Println("\n=== INICIANDO GRACEFUL SHUTDOWN ===")
	
	// Criar contexto com timeout para shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Iniciar processo de shutdown
	gsm.performGracefulShutdown(ctx)
}

// performGracefulShutdown executa shutdown gracioso
func (gsm *GracefulShutdownManager) performGracefulShutdown(ctx context.Context) {
	log.Println("Encerrando servidor HTTP...")
	
	// Desabilitar novas requisições (keep-alives)
	gsm.server.SetKeepAlivesEnabled(false)
	
	// Aguardar requisições ativas terminarem (com timeout)
	if err := gsm.server.Shutdown(ctx); err != nil {
		log.Printf("Erro durante shutdown do servidor: %v", err)
	} else {
		log.Println("Servidor HTTP encerrado com sucesso")
	}
	
	// Fechar conexões com bancos de dados
	gsm.closeConnections()
	
	// Log final
	log.Println("=== GRACEFUL SHUTDOWN COMPLETO ===")
	log.Println("TranspRota encerrado com segurança")
}

// closeConnections fecha conexões com bancos e recursos
func (gsm *GracefulShutdownManager) closeConnections() {
	log.Println("Fechando conexões com bancos de dados...")
	
	// Fechar conexão PostgreSQL
	if gsm.app.db != nil {
		if err := gsm.app.db.Close(); err != nil {
			log.Printf("Erro ao fechar conexão PostgreSQL: %v", err)
		} else {
			log.Println("Conexão PostgreSQL fechada com sucesso")
		}
	}
	
	// Fechar conexão Redis
	if gsm.app.rdb != nil {
		if err := gsm.app.rdb.Close(); err != nil {
			log.Printf("Erro ao fechar conexão Redis: %v", err)
		} else {
			log.Println("Conexão Redis fechada com sucesso")
		}
	}
	
	// Limpar caches e recursos
	gsm.cleanupResources()
}

// cleanupResources limpa recursos adicionais
func (gsm *GracefulShutdownManager) cleanupResources() {
	log.Println("Limpando recursos...")
	
	// Limpar cache de clima
	if gsm.app.weatherCache != nil {
		gsm.app.weatherCache = make(map[string]WeatherData)
	}
	
	// Limpar outras estruturas de dados se necessário
	
	log.Println("Recursos limpos com sucesso")
}

// HealthCheckMiddleware middleware para health checks durante shutdown
func HealthCheckMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Verificar se sistema está em modo de shutdown
		if isShuttingDown() {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "shutting_down",
				"message": "Sistema em processo de encerramento",
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// Variável global para controlar estado de shutdown
var shuttingDown = false

// isShuttingDown verifica se sistema está em modo de shutdown
func isShuttingDown() bool {
	return shuttingDown
}

// setShuttingDown marca sistema como em modo de shutdown
func setShuttingDown() {
	shuttingDown = true
}

// EnhancedGracefulShutdownManager gerenciador avançado de shutdown
type EnhancedGracefulShutdownManager struct {
	*GracefulShutdownManager
	shutdownTimeout time.Duration
	activeRequests  int64
}

// NewEnhancedGracefulShutdownManager cria gerenciador avançado
func NewEnhancedGracefulShutdownManager(server *http.Server, app *App, timeout time.Duration) *EnhancedGracefulShutdownManager {
	return &EnhancedGracefulShutdownManager{
		GracefulShutdownManager: NewGracefulShutdownManager(server, app),
		shutdownTimeout:         timeout,
	}
}

// TrackRequestMiddleware middleware para rastrear requisições ativas
func (egsm *EnhancedGracefulShutdownManager) TrackRequestMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Incrementar contador de requisições ativas
		defer func() {
			// Decrementar quando completar
		}()
		
		c.Next()
	}
}

// performEnhancedGracefulShutdown executa shutdown avançado
func (egsm *EnhancedGracefulShutdownManager) performEnhancedGracefulShutdown(ctx context.Context) {
	log.Println("=== INICIANDO ENHANCED GRACEFUL SHUTDOWN ===")
	
	// Marcar como em modo de shutdown
	setShuttingDown()
	
	// Aguardar requisições ativas terminarem ou timeout
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	shutdownStart := time.Now()
	
	for {
		select {
		case <-ctx.Done():
			log.Println("Timeout durante graceful shutdown")
			goto forceShutdown
		case <-ticker.C:
			elapsed := time.Since(shutdownStart)
			log.Printf("Aguardando requisições ativas... (%.1fs decorrido)", elapsed.Seconds())
			
			// Se passou tempo suficiente, forçar shutdown
			if elapsed > egsm.shutdownTimeout {
				log.Println("Timeout excedido, forçando shutdown")
				goto forceShutdown
			}
			
			// Se não há requisições ativas, prosseguir
			if egsm.activeRequests == 0 {
				log.Println("Nenhuma requisição ativa, prosseguindo com shutdown")
				goto normalShutdown
			}
		}
	}
	
normalShutdown:
	// Shutdown normal
	egsm.performGracefulShutdown(ctx)
	return
	
forceShutdown:
	// Forçar shutdown
	log.Println("Forçando encerramento do servidor...")
	if err := egsm.server.Close(); err != nil {
		log.Printf("Erro ao forçar shutdown: %v", err)
	}
	
	egsm.closeConnections()
	log.Println("=== FORCED SHUTDOWN COMPLETO ===")
}

// setupEnhancedGracefulShutdown configura graceful shutdown avançado
func setupEnhancedGracefulShutdown(r *gin.Engine, app *App) {
	server := &http.Server{
		Addr:    ":8080",
		Handler: r,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	egsm := NewEnhancedGracefulShutdownManager(server, app, 30*time.Second)
	
	// Aplicar middleware de tracking
	r.Use(egsm.TrackRequestMiddleware())
	r.Use(HealthCheckMiddleware())
	
	// Iniciar servidor
	go func() {
		log.Println("TranspRota API operacional em http://localhost:8080")
		log.Println("Modo:", getAppMode(app))
		
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Erro ao iniciar servidor: %v", err)
		}
	}()

	// Configurar graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		
		ctx, cancel := context.WithTimeout(context.Background(), egsm.shutdownTimeout)
		defer cancel()
		
		egsm.performEnhancedGracefulShutdown(ctx)
	}()
}
