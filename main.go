package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/config"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/server"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/telemetry"
)

func main() {
	logger.Info("Main", "Iniciando TranspRota Backend...")

	// 1. Carregar configuração
	logger.Info("Main", "Carregando configuração...")
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Main", "Falha ao carregar configuração: %v", err)
		os.Exit(1)
	}
	logger.Info("Main", "Configuração carregada")

	// 2. Inicializar bancos de dados
	logger.Info("Main", "Inicializando bancos de dados...")
	db, rdb, err := initDatabases(cfg)
	if err != nil {
		logger.Error("Main", "Falha ao inicializar bancos: %v", err)
		logger.Warn("Main", "Operando em modo degradado (sem bancos)")
		db = nil
		rdb = nil
	} else {
		logger.Info("Main", "Bancos de dados inicializados")
	}

	// 3. Iniciar workers em goroutines separadas
	if rdb != nil {
		logger.Info("Main", "Iniciando CleanupWorker...")
		cleanupWorker := telemetry.NewCleanupWorker(db, rdb)
		cleanupWorker.Start()
		logger.Info("Main", "CleanupWorker iniciado em background")
	}

	// 4. Criar e iniciar servidor
	logger.Info("Main", "Criando servidor HTTP...")
	srv := server.NewServer(cfg, db, rdb)
	srv.SetupRoutes()
	logger.Info("Main", "Servidor HTTP criado")

	// 5. Configurar graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt,    // SIGINT (Ctrl+C)
		syscall.SIGTERM, // SIGTERM (kill)
		syscall.SIGQUIT, // SIGQUIT (quit)
	)
	defer stop()

	// 6. Iniciar servidor em goroutine
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("Main", "Iniciando servidor na porta %s...", cfg.ServerPort)
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// 7. Aguardar sinal de shutdown
	<-ctx.Done()
	logger.Info("Main", "Recebido sinal de shutdown, iniciando graceful shutdown...")

	// 8. Criar contexto com timeout para shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 9. Fechar servidor HTTP
	logger.Info("Main", "Fechando servidor HTTP...")
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Main", "Erro ao fechar servidor: %v", err)
	}

	// 10. Fechar conexões de banco de dados
	logger.Info("Main", "Fechando conexões de banco de dados...")
	if db != nil {
		if err := db.Close(); err != nil {
			logger.Error("Main", "Erro ao fechar PostgreSQL: %v", err)
		} else {
			logger.Info("Main", "PostgreSQL fechado com sucesso")
		}
	}

	if rdb != nil {
		if err := rdb.Shutdown(shutdownCtx); err != nil {
			logger.Error("Main", "Erro ao fechar Redis: %v", err)
		} else {
			logger.Info("Main", "Redis fechado com sucesso")
		}
	}

	// 11. Verificar erro do servidor
	select {
	case err := <-serverErr:
		logger.Error("Main", "Erro do servidor: %v", err)
		os.Exit(1)
	default:
		logger.Info("Main", "Aplicação encerrada gracefulmente")
	}
}

// initDatabases inicializa PostgreSQL e Redis com retry
func initDatabases(cfg *config.Config) (*sql.DB, *redis.Client, error) {
	// PostgreSQL
	db, err := initPostgreSQL(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("falha ao conectar PostgreSQL: %w", err)
	}

	// Redis
	rdb, err := initRedis(cfg)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("falha ao conectar Redis: %w", err)
	}

	return db, rdb, nil
}

// initPostgreSQL inicializa conexão PostgreSQL com Exponential Backoff
func initPostgreSQL(cfg *config.Config) (*sql.DB, error) {
	var db *sql.DB
	var err error

	maxRetries := 5
	baseDelay := 1 * time.Second

	// Usar config em vez de hardcode
	dsn := cfg.GetPostgresDSN()

	for attempt := 1; attempt <= maxRetries; attempt++ {
		logger.Info("Main", "PostgreSQL Tentativa %d/%d de conexão (%s:%s)...", attempt, maxRetries, cfg.DBHost, cfg.DBPort)

		db, err = sql.Open("postgres", dsn)
		if err != nil {
			logger.Error("Main", "PostgreSQL Erro ao abrir conexão: %v", err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return nil, fmt.Errorf("falha ao conectar ao PostgreSQL após %d tentativas: %w", maxRetries, err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := db.PingContext(ctx); err != nil {
			logger.Error("Main", "PostgreSQL Erro no ping: %v", err)
			cancel()
			db.Close()
			if attempt < maxRetries {
				time.Sleep(baseDelay * time.Duration(attempt))
			}
			continue
		}
		cancel()

		// Configurações de pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		logger.Info("Main", "PostgreSQL Conectado com sucesso (tentativa %d)", attempt)
		return db, nil
	}

	return nil, fmt.Errorf("falha após %d tentativas: %w", maxRetries, err)
}

// initRedis inicializa conexão Redis com retry
func initRedis(cfg *config.Config) (*redis.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Usar config em vez de hardcode
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})

	logger.Info("Main", "Redis Tentativa de conexão (%s)...", cfg.RedisAddr)

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Error("Main", "Redis Erro no ping: %v", err)
		return nil, err
	}

	return rdb, nil
}
