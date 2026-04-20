package config

import (
	"fmt"
	"os"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/joho/godotenv"
)

// Config contém todas as configurações da aplicação
type Config struct {
	// Database
	DBUser     string
	DBPassword string
	DBName     string
	DBHost     string
	DBPort     string

	// Redis
	RedisAddr string

	// API
	APIKey string

	// Server
	ServerPort string
}

// Load carrega as configurações do arquivo .env e variáveis de ambiente
func Load() (*Config, error) {
	// Tentar carregar .env, mas não falhar se não existir
	if err := godotenv.Load(); err != nil {
		logger.Warn("Config", "Arquivo .env não encontrado, usando variáveis de ambiente")
	}

	cfg := &Config{
		DBUser:     getEnv("DB_USER", "admin"),
		DBPassword: getEnv("DB_PASSWORD", "admin123"),
		DBName:     getEnv("DB_NAME", "transprota"),
		DBHost:     getEnv("DB_HOST", "127.0.0.1"),
		DBPort:     getEnv("DB_PORT", "5432"),
		RedisAddr:  getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		APIKey:     getEnv("API_SECRET_KEY", ""),
		ServerPort: getEnv("SERVER_PORT", "8081"),
	}

	// Log das configurações carregadas (sem senhas)
	logger.Info("Config", "Configurações carregadas:")
	logger.Info("Config", "  PostgreSQL: %s@%s:%s/%s", cfg.DBUser, cfg.DBHost, cfg.DBPort, cfg.DBName)
	logger.Info("Config", "  Redis: %s", cfg.RedisAddr)
	logger.Info("Config", "  Server Port: %s", cfg.ServerPort)

	return cfg, nil
}

// GetPostgresDSN retorna a DSN para conexão com PostgreSQL
func (c *Config) GetPostgresDSN() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable host=%s port=%s",
		c.DBUser, c.DBPassword, c.DBName, c.DBHost, c.DBPort)
}

// getEnv retorna o valor da variável de ambiente ou o valor padrão
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
