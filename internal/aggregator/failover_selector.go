package aggregator

import (
	"context"
	"fmt"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/telemetry"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// FAILOVER SELECTOR - PRIORIZA RMTC, FALHA PARA INTELIGÊNCIA COLETIVA
// ============================================================================

const (
	// Timeout para requisição à API externa (2 segundos)
	ExternalAPITimeout = 2 * time.Second
	// Source RMTC (API oficial)
	SourceRMTC = "rmtc"
	// Source Collective Intelligence (dados dos usuários)
	SourceCollective = "collective"
)

// DataSource representa origem dos dados
type DataSource string

const (
	DataSourceRMTC       DataSource = "rmtc"
	DataSourceCollective DataSource = "collective"
	DataSourceCache      DataSource = "cache"
)

// BusData representa dados de ônibus
type BusData struct {
	RouteID    string     `json:"route_id"`
	VehicleID  string     `json:"vehicle_id"`
	Latitude   float64    `json:"latitude"`
	Longitude  float64    `json:"longitude"`
	Heading    float64    `json:"heading"`
	Speed      float64    `json:"speed"`
	Occupancy  string     `json:"occupancy"`
	LastUpdate time.Time  `json:"last_update"`
	Source     DataSource `json:"source"`
	Confidence float64    `json:"confidence"` // 0.0 a 1.0
}

// FailoverSelector gerencia seleção de fonte de dados com failover
type FailoverSelector struct {
	httpClient    *MobileHTTPClient
	cache         *AdaptiveCache
	telemetryRepo *telemetry.Repository
	rdb           *redis.Client
}

// NewFailoverSelector cria um novo seletor de failover
func NewFailoverSelector(rdb *redis.Client, telemetryRepo *telemetry.Repository) *FailoverSelector {
	return &FailoverSelector{
		httpClient:    NewMobileHTTPClient(),
		cache:         NewAdaptiveCache(rdb),
		telemetryRepo: telemetryRepo,
		rdb:           rdb,
	}
}

// GetBusData busca dados de ônibus com failover inteligente
// Prioridade: Cache -> RMTC -> Inteligência Coletiva
func (fs *FailoverSelector) GetBusData(ctx context.Context, routeID string) (*BusData, error) {
	start := time.Now()

	// 1. Tentar buscar do cache (mais rápido)
	cachedData, found, err := fs.tryCache(ctx, routeID)
	if err == nil && found {
		cachedData.Source = DataSourceCache
		cachedData.Confidence = 0.95 // Alta confiança para cache
		logger.Info("Aggregator", "Data from cache | Route: %s | Source: %s | Time: %v", routeID, cachedData.Source, time.Since(start))
		return cachedData, nil
	}

	// 2. Tentar API externa (RMTC) com timeout de 2s
	rmtcData, err := fs.tryRMTC(ctx, routeID)
	if err == nil {
		// Sucesso: cachear e retornar
		fs.cache.CacheExternalAPIData(ctx, SourceRMTC, routeID, rmtcData)
		rmtcData.Source = DataSourceRMTC
		rmtcData.Confidence = 1.0 // Máxima confiança para RMTC
		logger.Info("Aggregator", "Data from RMTC | Route: %s | Time: %v", routeID, time.Since(start))
		return rmtcData, nil
	}

	logger.Warn("Aggregator", "RMTC failed | Route: %s | Error: %v", routeID, err)

	// 3. Failover para Inteligência Coletiva
	collectiveData, err := fs.tryCollectiveIntelligence(ctx, routeID)
	if err == nil {
		collectiveData.Source = DataSourceCollective
		collectiveData.Confidence = 0.8 // Confiança média para dados coletivos
		logger.Info("Aggregator", "Data from Collective Intelligence | Route: %s | Time: %v", routeID, time.Since(start))
		return collectiveData, nil
	}

	logger.Error("Aggregator", "All data sources failed | Route: %s | Total time: %v", routeID, time.Since(start))
	return nil, fmt.Errorf("all data sources failed for route %s", routeID)
}

// tryCache tenta buscar dados do cache
func (fs *FailoverSelector) tryCache(ctx context.Context, routeID string) (*BusData, bool, error) {
	var data BusData
	found, err := fs.cache.GetExternalAPIData(ctx, SourceRMTC, routeID, &data)
	if err != nil {
		return nil, false, err
	}
	return &data, found, nil
}

// tryRMTC tenta buscar dados da API RMTC
func (fs *FailoverSelector) tryRMTC(ctx context.Context, routeID string) (*BusData, error) {
	// Criar contexto com timeout de 2 segundos
	ctxWithTimeout, cancel := context.WithTimeout(ctx, ExternalAPITimeout)
	defer cancel()

	// URL simulada da API RMTC (substituir pela URL real)
	url := fmt.Sprintf("https://api.rmtc.goiania.gov.br/v1/vehicles/%s", routeID)

	// Realizar requisição com headers mobile
	resp, err := fs.httpClient.GetWithMobileHeaders(ctxWithTimeout, url)
	if err != nil {
		return nil, fmt.Errorf("RMTC request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("RMTC returned status %d", resp.StatusCode)
	}

	// Normalizar resposta usando Normalizer
	normalizer := NewNormalizer()
	normalized, err := normalizer.NormalizeFromRMTCReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize RMTC response: %w", err)
	}

	// Converter para BusData
	return &BusData{
		RouteID:    normalized.TID,
		VehicleID:  fmt.Sprintf("rmtc-%s", normalized.TID),
		Latitude:   normalized.Lat,
		Longitude:  normalized.Long,
		Heading:    normalized.Hdg,
		Speed:      normalized.Spd,
		Occupancy:  normalized.Ocp,
		LastUpdate: time.Now(),
		Source:     DataSourceRMTC,
		Confidence: normalized.Cnf,
	}, nil
}

// tryCollectiveIntelligence tenta buscar dados da Inteligência Coletiva
func (fs *FailoverSelector) tryCollectiveIntelligence(ctx context.Context, routeID string) (*BusData, error) {
	if fs.telemetryRepo == nil {
		return nil, fmt.Errorf("telemetry repository not available")
	}

	// Buscar última posição de ônibus da rota via Redis GEO
	query := &telemetry.GeoRadiusQuery{
		CenterLat:    -16.6869, // Centro de Goiânia
		CenterLng:    -49.2648,
		RadiusMeters: 20000, // 20km para cobrir toda cidade
		RouteFilter:  routeID,
	}

	response, err := fs.telemetryRepo.GetBusLocationsByRadius(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("collective intelligence query failed: %w", err)
	}

	if len(response.Buses) == 0 {
		return nil, fmt.Errorf("no collective data available for route %s", routeID)
	}

	// Retornar o ônibus mais recente da rota
	bus := response.Buses[0]

	return &BusData{
		RouteID:    bus.RouteID,
		VehicleID:  bus.DeviceHash,
		Latitude:   bus.Latitude,
		Longitude:  bus.Longitude,
		Heading:    bus.Heading,
		Speed:      bus.Speed,
		Occupancy:  bus.Occupancy,
		LastUpdate: time.Now(),
		Source:     DataSourceCollective,
		Confidence: 0.8,
	}, nil
}

// GetMultipleBusData busca dados de múltiplos ônibus em paralelo
func (fs *FailoverSelector) GetMultipleBusData(ctx context.Context, routeIDs []string) (map[string]*BusData, error) {
	results := make(map[string]*BusData)
	errChan := make(chan error, len(routeIDs))

	// Buscar em paralelo
	for _, routeID := range routeIDs {
		go func(rid string) {
			data, err := fs.GetBusData(ctx, rid)
			if err != nil {
				errChan <- err
				return
			}
			results[rid] = data
			errChan <- nil
		}(routeID)
	}

	// Coletar resultados
	for i := 0; i < len(routeIDs); i++ {
		if err := <-errChan; err != nil {
			logger.Warn("Aggregator", "Failed to get bus data | Error: %v", err)
		}
	}

	return results, nil
}

// HealthCheck verifica saúde das fontes de dados
func (fs *FailoverSelector) HealthCheck(ctx context.Context) map[DataSource]bool {
	health := make(map[DataSource]bool)

	// Verificar Redis
	health[DataSourceCache] = fs.rdb != nil

	// Verificar RMTC (timeout curto)
	ctxTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	_, err := fs.tryRMTC(ctxTimeout, "001") // Testar com rota 001
	health[DataSourceRMTC] = err == nil

	// Verificar Inteligência Coletiva
	health[DataSourceCollective] = fs.telemetryRepo != nil

	logger.Info("Aggregator", "Health check | Cache: %v | RMTC: %v | Collective: %v",
		health[DataSourceCache], health[DataSourceRMTC], health[DataSourceCollective])

	return health
}
