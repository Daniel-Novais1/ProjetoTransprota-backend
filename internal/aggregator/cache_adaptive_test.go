package aggregator

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
)

// ============================================================================
// UNIT TESTS - ADAPTIVE CACHE
// ============================================================================

func TestNewAdaptiveCache(t *testing.T) {
	cache := NewAdaptiveCache(nil)

	if cache == nil {
		t.Fatal("NewAdaptiveCache returned nil")
	}
}

func TestCacheData_WithoutRedis(t *testing.T) {
	cache := NewAdaptiveCache(nil)

	data := BusData{RouteID: "001", Latitude: -16.6869, Longitude: -49.2648}
	err := cache.CacheData(context.Background(), "test-key", data, ExternalAPICacheTTL)

	if err == nil {
		t.Error("Expected error when Redis is not available")
	}
}

func TestGetData_WithoutRedis(t *testing.T) {
	cache := NewAdaptiveCache(nil)

	var data BusData
	found, err := cache.GetData(context.Background(), "test-key", &data)

	if err == nil {
		t.Error("Expected error when Redis is not available")
	}
	if found {
		t.Error("Should not find data when Redis is not available")
	}
}

func TestCacheExternalAPIData(t *testing.T) {
	cache := NewAdaptiveCache(nil)

	data := BusData{RouteID: "001", Latitude: -16.6869, Longitude: -49.2648}
	err := cache.CacheExternalAPIData(context.Background(), SourceRMTC, "001", data)

	if err == nil {
		t.Error("Expected error when Redis is not available")
	}
}

func TestGetExternalAPIData_WithoutRedis(t *testing.T) {
	cache := NewAdaptiveCache(nil)

	var data BusData
	found, err := cache.GetExternalAPIData(context.Background(), SourceRMTC, "001", &data)

	if err == nil {
		t.Error("Expected error when Redis is not available")
	}
	if found {
		t.Error("Should not find data when Redis is not available")
	}
}

func TestInvalidateRoute_WithoutRedis(t *testing.T) {
	cache := NewAdaptiveCache(nil)

	err := cache.InvalidateRoute(context.Background(), "001")

	if err == nil {
		t.Error("Expected error when Redis is not available")
	}
}

func TestGetCacheStats_WithoutRedis(t *testing.T) {
	cache := NewAdaptiveCache(nil)

	stats, err := cache.GetCacheStats(context.Background())

	if err == nil {
		t.Error("Expected error when Redis is not available")
	}
	if stats != nil {
		t.Error("Should not return stats when Redis is not available")
	}
}

// Teste com mock de Redis (se necessário implementar)
func TestCacheData_WithMockRedis(t *testing.T) {
	t.Skip("Skipping Redis mock test - requires Redis instance or proper mock")

	// Este teste requer um mock de Redis ou Redis real
	// Por enquanto, apenas verificamos que a função não panic
	cache := NewAdaptiveCache(&redis.Client{})

	data := BusData{RouteID: "001", Latitude: -16.6869, Longitude: -49.2648}
	_ = cache.CacheData(context.Background(), "test-key", data, ExternalAPICacheTTL)
}
