package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

const (
	StreamUnifierTTL         = 30 * time.Second
	MinConfidenceForPriority = 70 // 0.7 * 100
)

type StreamSource string

const (
	StreamSourceRMTC       StreamSource = "rmtc"
	StreamSourceCollective StreamSource = "collective"
	StreamSourceUnified    StreamSource = "unified"
)

type UnifiedBusData struct {
	RouteID       string         `json:"route_id"`
	VehicleID     string         `json:"vehicle_id"`
	Latitude      float64        `json:"latitude"`
	Longitude     float64        `json:"longitude"`
	Heading       float64        `json:"heading"`
	Speed         float64        `json:"speed"`
	Occupancy     string         `json:"occupancy"`
	LastUpdate    time.Time      `json:"last_update"`
	PrimarySource StreamSource   `json:"primary_source"`
	Confidence    float64        `json:"confidence"`
	Contributors  int            `json:"contributors"`
	ETA           *ETAPrediction `json:"eta,omitempty"`
}

type ETAPrediction struct {
	Minutes       int     `json:"minutes"`
	DistanceKM    float64 `json:"distance_km"`
	TrafficFactor float64 `json:"traffic_factor"`
	Confidence    float64 `json:"confidence"`
}

type StreamUnifier struct {
	rdb   *redis.Client
	cache map[string]*UnifiedBusData
	mu    sync.RWMutex
}

func NewStreamUnifier(rdb *redis.Client) *StreamUnifier {
	return &StreamUnifier{
		rdb:   rdb,
		cache: make(map[string]*UnifiedBusData),
	}
}

func (su *StreamUnifier) UnifyData(ctx context.Context, routeID string, source StreamSource, data *BusLocation) (*UnifiedBusData, error) {
	su.mu.Lock()
	defer su.mu.Unlock()

	key := fmt.Sprintf("unified:%s", routeID)

	existing, exists := su.cache[key]

	unified := &UnifiedBusData{
		RouteID:      routeID,
		VehicleID:    data.DeviceHash,
		Latitude:     data.Latitude,
		Longitude:    data.Longitude,
		Heading:      data.Heading,
		Speed:        data.Speed,
		Occupancy:    data.Occupancy,
		LastUpdate:   time.Now(),
		Confidence:   float64(data.Confidence) / 100.0,
		Contributors: data.Contributors,
	}

	if source == StreamSourceCollective && data.Contributors >= 3 && data.Confidence >= MinConfidenceForPriority {
		unified.PrimarySource = StreamSourceCollective
		unified.Confidence = 0.9
		logger.Info("StreamUnifier", "Collective data prioritized | Route: %s | Contributors: %d", routeID, data.Contributors)
	} else if source == StreamSourceRMTC {
		unified.PrimarySource = StreamSourceRMTC
		unified.Confidence = 1.0
		logger.Info("StreamUnifier", "RMTC data used | Route: %s", routeID)
	} else if exists {
		unified.PrimarySource = existing.PrimarySource
		unified.Confidence = existing.Confidence
	}

	if exists && unified.PrimarySource == StreamSourceCollective {
		unified.Contributors = existing.Contributors
	}

	su.cache[key] = unified

	if su.rdb != nil {
		su.cacheUnifiedData(ctx, key, unified)
	}

	return unified, nil
}

func (su *StreamUnifier) GetUnifiedData(ctx context.Context, routeID string) (*UnifiedBusData, error) {
	su.mu.RLock()
	defer su.mu.RUnlock()

	key := fmt.Sprintf("unified:%s", routeID)

	if data, exists := su.cache[key]; exists {
		return data, nil
	}

	if su.rdb != nil {
		return su.getCachedUnifiedData(ctx, key)
	}

	return nil, fmt.Errorf("no unified data available for route %s", routeID)
}

func (su *StreamUnifier) GetAllUnifiedData(ctx context.Context) (map[string]*UnifiedBusData, error) {
	su.mu.RLock()
	defer su.mu.RUnlock()

	result := make(map[string]*UnifiedBusData)
	for k, v := range su.cache {
		result[k] = v
	}

	return result, nil
}

func (su *StreamUnifier) cacheUnifiedData(ctx context.Context, key string, data *UnifiedBusData) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Warn("StreamUnifier", "Failed to marshal unified data: %v", err)
		return
	}
	err = su.rdb.Set(ctx, key, jsonData, StreamUnifierTTL).Err()
	if err != nil {
		logger.Warn("StreamUnifier", "Failed to cache unified data: %v", err)
	}
}

func (su *StreamUnifier) getCachedUnifiedData(ctx context.Context, key string) (*UnifiedBusData, error) {
	data, err := su.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var unified UnifiedBusData
	err = json.Unmarshal([]byte(data), &unified)
	if err != nil {
		return nil, err
	}

	su.cache[key] = &unified
	return &unified, nil
}

func (su *StreamUnifier) InvalidateRoute(routeID string) {
	su.mu.Lock()
	defer su.mu.Unlock()

	key := fmt.Sprintf("unified:%s", routeID)
	delete(su.cache, key)

	if su.rdb != nil {
		su.rdb.Del(context.Background(), key)
	}
}

func (su *StreamUnifier) BroadcastToWebSocket(ctx context.Context, data *UnifiedBusData) error {
	if su.rdb == nil {
		return fmt.Errorf("redis not available")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		logger.Error("StreamUnifier", "Failed to marshal broadcast data: %v", err)
		return err
	}

	channel := "unified_bus_updates"
	err = su.rdb.Publish(ctx, channel, jsonData).Err()
	if err != nil {
		logger.Error("StreamUnifier", "Failed to broadcast to WebSocket: %v", err)
		return err
	}

	logger.Debug("StreamUnifier", "Broadcasted to WebSocket | Route: %s | Source: %s", data.RouteID, data.PrimarySource)
	return nil
}
