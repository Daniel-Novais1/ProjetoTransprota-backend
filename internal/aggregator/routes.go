package aggregator

import (
	"context"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/telemetry"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// AGGREGATOR ROUTES
// ============================================================================

// RegisterRoutes registra rotas do agregador
func RegisterRoutes(router *gin.Engine, rdb *redis.Client, telemetryRepo *telemetry.Repository) {
	// Criar worker
	worker := NewAggregatorWorker(rdb, telemetryRepo, 30*time.Second)

	// Iniciar worker em background
	go worker.Start(context.Background())

	// Criar controller
	controller := NewController(worker)

	// Grupo de rotas do agregador
	aggregatorGroup := router.Group("/api/v1/aggregator")
	{
		// GET /api/v1/aggregator/bus/:route_id - Dados de ônibus com failover
		aggregatorGroup.GET("/bus/:route_id", controller.GetBusData)

		// POST /api/v1/aggregator/refresh/:route_id - Forçar refresh de rota
		aggregatorGroup.POST("/refresh/:route_id", controller.RefreshRoute)

		// GET /api/v1/aggregator/stats - Estatísticas do agregador
		aggregatorGroup.GET("/stats", controller.GetStats)

		// GET /api/v1/aggregator/health - Health check
		aggregatorGroup.GET("/health", controller.HealthCheck)
	}
}
