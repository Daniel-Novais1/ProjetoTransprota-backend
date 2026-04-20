package aggregator

import (
	"net/http"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/gin-gonic/gin"
)

// ============================================================================
// AGGREGATOR HTTP HANDLERS
// ============================================================================

// Controller gerencia handlers do agregador
type Controller struct {
	worker *AggregatorWorker
}

// NewController cria um novo controller
func NewController(worker *AggregatorWorker) *Controller {
	return &Controller{worker: worker}
}

// GetBusData retorna dados de ônibus com failover inteligente
// GET /api/v1/aggregator/bus/:route_id
func (c *Controller) GetBusData(ctx *gin.Context) {
	start := time.Now()
	
	routeID := ctx.Param("route_id")
	if routeID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "route_id is required"})
		return
	}
	
	logger.Info("Aggregator", "Bus data requested | Route: %s | IP: %s", routeID, ctx.ClientIP())
	
	// Buscar dados com failover
	data, err := c.worker.failoverSelector.GetBusData(ctx.Request.Context(), routeID)
	if err != nil {
		logger.Error("Aggregator", "Failed to get bus data | Route: %s | Error: %v", routeID, err)
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Failed to get bus data",
			"route": routeID,
		})
		return
	}
	
	ctx.JSON(http.StatusOK, gin.H{
		"route_id":   data.RouteID,
		"vehicle_id": data.VehicleID,
		"lat":        data.Latitude,
		"lng":        data.Longitude,
		"heading":    data.Heading,
		"speed":      data.Speed,
		"occupancy":  data.Occupancy,
		"source":     data.Source,
		"confidence": data.Confidence,
		"updated_at": data.LastUpdate,
		"pms":        time.Since(start).Milliseconds(),
	})
}

// RefreshRoute força refresh de uma rota
// POST /api/v1/aggregator/refresh/:route_id
func (c *Controller) RefreshRoute(ctx *gin.Context) {
	start := time.Now()
	
	routeID := ctx.Param("route_id")
	if routeID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "route_id is required"})
		return
	}
	
	logger.Info("Aggregator", "Route refresh requested | Route: %s | IP: %s", routeID, ctx.ClientIP())
	
	data, err := c.worker.RefreshRoute(ctx.Request.Context(), routeID)
	if err != nil {
		logger.Error("Aggregator", "Failed to refresh route | Route: %s | Error: %v", routeID, err)
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Failed to refresh route",
			"route": routeID,
		})
		return
	}
	
	ctx.JSON(http.StatusOK, gin.H{
		"route_id":   data.RouteID,
		"vehicle_id": data.VehicleID,
		"lat":        data.Latitude,
		"lng":        data.Longitude,
		"heading":    data.Heading,
		"speed":      data.Speed,
		"occupancy":  data.Occupancy,
		"source":     data.Source,
		"confidence": data.Confidence,
		"updated_at": data.LastUpdate,
		"pms":        time.Since(start).Milliseconds(),
	})
}

// GetStats retorna estatísticas do agregador
// GET /api/v1/aggregator/stats
func (c *Controller) GetStats(ctx *gin.Context) {
	start := time.Now()
	
	stats, err := c.worker.GetStats(ctx.Request.Context())
	if err != nil {
		logger.Error("Aggregator", "Failed to get stats | Error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get stats",
		})
		return
	}
	
	ctx.JSON(http.StatusOK, gin.H{
		"cache": gin.H{
			"external_keys": stats.CacheStats.ExternalDataKeys,
			"ttl_seconds":   int(stats.CacheStats.TTL.Seconds()),
		},
		"health": stats.Health,
		"interval_seconds": int(stats.Interval.Seconds()),
		"pms":   time.Since(start).Milliseconds(),
	})
}

// HealthCheck retorna saúde do agregador
// GET /api/v1/aggregator/health
func (c *Controller) HealthCheck(ctx *gin.Context) {
	health := c.worker.failoverSelector.HealthCheck(ctx.Request.Context())
	
	status := http.StatusOK
	for _, healthy := range health {
		if !healthy {
			status = http.StatusServiceUnavailable
			break
		}
	}
	
	ctx.JSON(status, gin.H{
		"status": status,
		"health": health,
	})
}
