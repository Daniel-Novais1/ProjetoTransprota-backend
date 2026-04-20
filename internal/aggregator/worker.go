package aggregator

import (
	"context"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/telemetry"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// AGGREGATOR WORKER - WORKER PARA REQUISIÇÕES EXTERNAS
// ============================================================================

// AggregatorWorker gerencia worker de agregação de APIs externas
type AggregatorWorker struct {
	failoverSelector *FailoverSelector
	cache            *AdaptiveCache
	telemetryRepo    *telemetry.Repository
	rdb              *redis.Client
	interval         time.Duration
	stopChan         chan struct{}
}

// NewAggregatorWorker cria um novo worker de agregação
func NewAggregatorWorker(rdb *redis.Client, telemetryRepo *telemetry.Repository, interval time.Duration) *AggregatorWorker {
	return &AggregatorWorker{
		failoverSelector: NewFailoverSelector(rdb, telemetryRepo),
		cache:            NewAdaptiveCache(rdb),
		telemetryRepo:    telemetryRepo,
		rdb:              rdb,
		interval:         interval,
		stopChan:         make(chan struct{}),
	}
}

// Start inicia o worker em background
func (aw *AggregatorWorker) Start(ctx context.Context) {
	logger.Info("Aggregator", "Starting Aggregator Worker | Interval: %v", aw.interval)
	
	ticker := time.NewTicker(aw.interval)
	defer ticker.Stop()
	
	// Primeira execução imediata
	aw.runOnce(ctx)
	
	for {
		select {
		case <-ctx.Done():
			logger.Info("Aggregator", "Aggregator Worker stopped")
			return
		case <-ticker.C:
			aw.runOnce(ctx)
		case <-aw.stopChan:
			logger.Info("Aggregator", "Aggregator Worker stopped via stop channel")
			return
		}
	}
}

// Stop para o worker
func (aw *AggregatorWorker) Stop() {
	close(aw.stopChan)
}

// runOnce executa uma única iteração do worker
func (aw *AggregatorWorker) runOnce(ctx context.Context) {
	start := time.Now()
	
	logger.Debug("Aggregator", "Running aggregation cycle")
	
	// Lista de rotas para buscar (pode ser configurável ou dinâmica)
	routeIDs := []string{"001", "002", "003", "004", "005"}
	
	// Buscar dados de múltiplas rotas em paralelo
	results, err := aw.failoverSelector.GetMultipleBusData(ctx, routeIDs)
	if err != nil {
		logger.Error("Aggregator", "Failed to aggregate bus data | Error: %v", err)
		return
	}
	
	// Logar resultados
	for routeID, data := range results {
		logger.Info("Aggregator", "Bus data aggregated | Route: %s | Source: %s | Lat: %.6f | Lng: %.6f | Confidence: %.2f",
			routeID, data.Source, data.Latitude, data.Longitude, data.Confidence)
	}
	
	elapsed := time.Since(start)
	logger.Info("Aggregator", "Aggregation cycle completed | Routes: %d | Time: %v", len(results), elapsed)
	
	// Log de performance se demorar muito
	if elapsed > 5*time.Second {
		logger.Warn("Aggregator", "Slow aggregation cycle | Time: %v", elapsed)
	}
}

// RefreshRoute força refresh de uma rota específica
func (aw *AggregatorWorker) RefreshRoute(ctx context.Context, routeID string) (*BusData, error) {
	logger.Info("Aggregator", "Force refresh route | Route: %s", routeID)
	
	// Invalidar cache da rota
	if err := aw.cache.InvalidateRoute(ctx, routeID); err != nil {
		logger.Warn("Aggregator", "Failed to invalidate cache | Route: %s | Error: %v", routeID, err)
	}
	
	// Buscar dados frescos
	return aw.failoverSelector.GetBusData(ctx, routeID)
}

// GetStats retorna estatísticas do worker
func (aw *AggregatorWorker) GetStats(ctx context.Context) (*WorkerStats, error) {
	cacheStats, err := aw.cache.GetCacheStats(ctx)
	if err != nil {
		return nil, err
	}
	
	health := aw.failoverSelector.HealthCheck(ctx)
	
	return &WorkerStats{
		CacheStats: cacheStats,
		Health:     health,
		Interval:   aw.interval,
	}, nil
}

// WorkerStats estatísticas do worker
type WorkerStats struct {
	CacheStats *CacheStats
	Health     map[DataSource]bool
	Interval   time.Duration
}
