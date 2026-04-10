package telemetry

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// SetupRoutes configura todas as rotas de telemetria
func SetupRoutes(r *gin.Engine, db *sql.DB, rdb *redis.Client) {
	controller := NewController(db, rdb)

	// POST /api/v1/telemetry/gps - Receber ping de telemetria GPS
	// Telemetria passiva do usuário com LGPD compliance
	r.POST("/api/v1/telemetry/gps", controller.ReceiveGPSPing)

	// GET /api/v1/telemetry/last-position/:device_hash - Última posição conhecida
	// Busca no Redis (cache 60s) ou fallback para PostgreSQL
	r.GET("/api/v1/telemetry/last-position/:device_hash", controller.GetLastPosition)

	// GET /api/v1/telemetry/latest - Última posição de todos os dispositivos
	// Retorna DISTINCT ON (device_hash) com ORDER BY device_hash, recorded_at DESC
	r.GET("/api/v1/telemetry/latest", controller.GetLatestPositions)
}
