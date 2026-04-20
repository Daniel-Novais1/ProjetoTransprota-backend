package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// ADAPTIVE CACHE - CACHE PARA DADOS DE API EXTERNA
// ============================================================================

// AdaptiveCache gerencia cache adaptativo para dados de APIs externas
type AdaptiveCache struct {
	rdb *redis.Client
}

// NewAdaptiveCache cria um novo cache adaptativo
func NewAdaptiveCache(rdb *redis.Client) *AdaptiveCache {
	return &AdaptiveCache{rdb: rdb}
}

// CacheData armazena dados no cache com TTL adaptativo
func (ac *AdaptiveCache) CacheData(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	if ac.rdb == nil {
		logger.Warn("Aggregator", "Redis not available, skipping cache | Key: %s", key)
		return fmt.Errorf("redis not available")
	}

	// Serializar dados
	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Error("Aggregator", "Failed to marshal cache data | Key: %s | Error: %v", key, err)
		return err
	}

	// Armazenar no Redis com TTL
	err = ac.rdb.Set(ctx, key, jsonData, ttl).Err()
	if err != nil {
		logger.Error("Aggregator", "Failed to cache data | Key: %s | Error: %v", key, err)
		return err
	}

	logger.Debug("Aggregator", "Data cached | Key: %s | TTL: %v | Size: %d bytes", key, ttl, len(jsonData))
	return nil
}

// GetData recupera dados do cache
func (ac *AdaptiveCache) GetData(ctx context.Context, key string, dest interface{}) (bool, error) {
	if ac.rdb == nil {
		return false, fmt.Errorf("redis not available")
	}

	// Buscar do Redis
	data, err := ac.rdb.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			logger.Debug("Aggregator", "Cache miss | Key: %s", key)
			return false, nil
		}
		logger.Error("Aggregator", "Failed to get cached data | Key: %s | Error: %v", key, err)
		return false, err
	}

	// Deserializar dados
	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		logger.Error("Aggregator", "Failed to unmarshal cached data | Key: %s | Error: %v", key, err)
		return false, err
	}

	logger.Debug("Aggregator", "Cache hit | Key: %s", key)
	return true, nil
}

// CacheExternalAPIData cacheia dados de API externa com TTL padrão de 30s
func (ac *AdaptiveCache) CacheExternalAPIData(ctx context.Context, source string, routeID string, data interface{}) error {
	key := fmt.Sprintf("external:%s:route:%s", source, routeID)
	return ac.CacheData(ctx, key, data, ExternalAPICacheTTL)
}

// GetExternalAPIData recupera dados de API externa do cache
func (ac *AdaptiveCache) GetExternalAPIData(ctx context.Context, source string, routeID string, dest interface{}) (bool, error) {
	key := fmt.Sprintf("external:%s:route:%s", source, routeID)
	return ac.GetData(ctx, key, dest)
}

// InvalidateRoute invalida cache de uma rota específica
func (ac *AdaptiveCache) InvalidateRoute(ctx context.Context, routeID string) error {
	if ac.rdb == nil {
		return fmt.Errorf("redis not available")
	}

	// Buscar todas as chaves que correspondem à rota
	iter := ac.rdb.Scan(ctx, 0, fmt.Sprintf("external:*:route:%s", routeID), 0).Iterator()
	keys := make([]string, 0)
	
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		logger.Error("Aggregator", "Failed to scan cache keys | Route: %s | Error: %v", routeID, err)
		return err
	}

	// Deletar todas as chaves encontradas
	if len(keys) > 0 {
		err := ac.rdb.Del(ctx, keys...).Err()
		if err != nil {
			logger.Error("Aggregator", "Failed to invalidate cache | Route: %s | Error: %v", routeID, err)
			return err
		}
		logger.Info("Aggregator", "Cache invalidated | Route: %s | Keys: %d", routeID, len(keys))
	}

	return nil
}

// GetCacheStats retorna estatísticas do cache
func (ac *AdaptiveCache) GetCacheStats(ctx context.Context) (*CacheStats, error) {
	if ac.rdb == nil {
		return nil, fmt.Errorf("redis not available")
	}

	// Contar chaves de cache externo
	iter := ac.rdb.Scan(ctx, 0, "external:*", 0).Iterator()
	count := 0
	
	for iter.Next(ctx) {
		count++
	}
	
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return &CacheStats{
		ExternalDataKeys: count,
		TTL:              ExternalAPICacheTTL,
	}, nil
}

// CacheStats estatísticas do cache
type CacheStats struct {
	ExternalDataKeys int
	TTL              time.Duration
}
