package telemetry

import (
	"context"
	"strings"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// PERFORMANCE LOGGING SERVICE
// ============================================================================

// PerformanceMetrics armazena métricas de performance
type PerformanceMetrics struct {
	QueryCount     int64
	QueryTotalTime time.Duration
	RedisMemory    string
	RedisKeys      int64
	SlowQueries    []SlowQuery
}

type SlowQuery struct {
	Query     string
	Duration  time.Duration
	Timestamp time.Time
}

var perfMetrics = &PerformanceMetrics{
	SlowQueries: make([]SlowQuery, 0),
}

// LogQueryPerformance registra performance de query SQL/PostGIS
func LogQueryPerformance(query string, duration time.Duration, ctx context.Context) {
	perfMetrics.QueryCount++
	perfMetrics.QueryTotalTime += duration

	// Logar queries lentas (> 100ms)
	if duration > 100*time.Millisecond {
		slowQuery := SlowQuery{
			Query:     query,
			Duration:  duration,
			Timestamp: time.Now(),
		}
		perfMetrics.SlowQueries = append(perfMetrics.SlowQueries, slowQuery)

		logger.Warn("Performance", "Slow query detected | Duration: %v | Query: %s", duration, query)
	}

	// Logar performance a cada 100 queries
	if perfMetrics.QueryCount%100 == 0 {
		avgDuration := perfMetrics.QueryTotalTime / time.Duration(perfMetrics.QueryCount)
		logger.Info("Performance", "Query metrics | Count: %d | Avg: %v | Total: %v",
			perfMetrics.QueryCount, avgDuration, perfMetrics.QueryTotalTime)
	}
}

// LogRedisPerformance registra performance de operações Redis
func LogRedisMetrics(rdb interface{}, ctx context.Context) {
	redisClient, ok := rdb.(*redis.Client)
	if !ok {
		return
	}

	// Buscar info do Redis
	info, err := redisClient.Info(ctx, "memory").Result()
	if err != nil {
		logger.Warn("Performance", "Failed to get Redis memory info: %v", err)
		return
	}

	// Buscar número de chaves
	keysCount, err := redisClient.DBSize(ctx).Result()
	if err != nil {
		logger.Warn("Performance", "Failed to get Redis keys count: %v", err)
		return
	}

	perfMetrics.RedisKeys = keysCount

	// Parse memory info (simplificado)
	var memoryUsed string
	for _, line := range splitLines(info) {
		if strings.Contains(line, "used_memory_human:") {
			memoryUsed = splitColon(line)[1]
		}
	}

	perfMetrics.RedisMemory = memoryUsed

	logger.Info("Performance", "Redis metrics | Memory: %s | Keys: %d", memoryUsed, keysCount)
}

// GetPerformanceMetrics retorna métricas atuais
func GetPerformanceMetrics() *PerformanceMetrics {
	return perfMetrics
}

// ResetPerformanceMetrics reseta métricas (chamado diariamente)
func ResetPerformanceMetrics() {
	perfMetrics = &PerformanceMetrics{
		SlowQueries: make([]SlowQuery, 0),
	}
	logger.Info("Performance", "Performance metrics reset")
}

// MonitorPerformance monitora performance em background
func MonitorPerformance(ctx context.Context, rdb interface{}) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			LogRedisMetrics(rdb, ctx)
		}
	}
}

// Funções auxiliares
func splitLines(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func splitColon(s string) []string {
	parts := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

// LogAPILatency registra latência de endpoint
func LogAPILatency(endpoint string, duration time.Duration) {
	// Alertar se latência > 100ms (objetivo CX)
	if duration > 100*time.Millisecond {
		logger.Warn("Performance", "High latency detected | Endpoint: %s | Duration: %v",
			endpoint, duration)
	} else {
		logger.Debug("Performance", "API latency | Endpoint: %s | Duration: %v",
			endpoint, duration)
	}
}
