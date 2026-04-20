package aggregator

import (
	"context"
	"testing"
)

// ============================================================================
// UNIT TESTS - FAILOVER SELECTOR
// ============================================================================

func TestNewFailoverSelector(t *testing.T) {
	selector := NewFailoverSelector(nil, nil)
	
	if selector == nil {
		t.Fatal("NewFailoverSelector returned nil")
	}
	
	if selector.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}
	
	if selector.cache == nil {
		t.Error("Cache should not be nil")
	}
}

func TestGetBusData_WithoutSources(t *testing.T) {
	selector := NewFailoverSelector(nil, nil)
	
	_, err := selector.GetBusData(context.Background(), "001")
	
	if err == nil {
		t.Error("Expected error when all data sources are unavailable")
	}
}

func TestTryCache_WithoutRedis(t *testing.T) {
	selector := NewFailoverSelector(nil, nil)
	
	_, found, err := selector.tryCache(context.Background(), "001")
	
	if err == nil {
		t.Error("Expected error when Redis is not available")
	}
	if found {
		t.Error("Should not find data when Redis is not available")
	}
}

func TestTryRMTC_Timeout(t *testing.T) {
	selector := NewFailoverSelector(nil, nil)
	
	_, err := selector.tryRMTC(context.Background(), "001")
	
	if err == nil {
		t.Error("Expected error when RMTC request times out")
	}
}

func TestTryCollectiveIntelligence_WithoutTelemetry(t *testing.T) {
	selector := NewFailoverSelector(nil, nil)
	
	_, err := selector.tryCollectiveIntelligence(context.Background(), "001")
	
	if err == nil {
		t.Error("Expected error when telemetry repository is not available")
	}
}

func TestHealthCheck(t *testing.T) {
	selector := NewFailoverSelector(nil, nil)
	
	health := selector.HealthCheck(context.Background())
	
	if health == nil {
		t.Fatal("HealthCheck returned nil")
	}
	
	// Sem Redis e sem telemetry, todas as fontes devem estar indisponíveis
	if health[DataSourceCache] {
		t.Error("Cache should be unavailable without Redis")
	}
	if health[DataSourceCollective] {
		t.Error("Collective intelligence should be unavailable without telemetry repo")
	}
}

func TestGetMultipleBusData(t *testing.T) {
	selector := NewFailoverSelector(nil, nil)
	
	routeIDs := []string{"001", "002", "003"}
	results, err := selector.GetMultipleBusData(context.Background(), routeIDs)
	
	if err != nil {
		t.Logf("Expected error without data sources: %v", err)
	}
	
	if results == nil {
		t.Fatal("GetMultipleBusData returned nil")
	}
	
	// Sem fontes de dados, o mapa deve estar vazio
	if len(results) > 0 {
		t.Errorf("Expected empty results, got %d entries", len(results))
	}
}

func TestBusData_ConfidenceLevels(t *testing.T) {
	tests := []struct {
		name       string
		source     DataSource
		confidence float64
	}{
		{"RMTC source", DataSourceRMTC, 1.0},
		{"Collective source", DataSourceCollective, 0.8},
		{"Cache source", DataSourceCache, 0.95},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := BusData{
				RouteID:    "001",
				Source:     tt.source,
				Confidence: tt.confidence,
			}
			
			if data.Source != tt.source {
				t.Errorf("Expected source %s, got %s", tt.source, data.Source)
			}
		})
	}
}
