package telemetry

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

const (
	MaxSpeedKmh = 300.0 // Velocidade máxima física (300 km/h = 83.3 m/s)
	MaxDistanceFor5s = 416.0 // 83.3 m/s * 5s = ~416 metros
	SuspiciousTTL = 24 * time.Hour
)

type DeviceHistory struct {
	LastLat      float64
	LastLng      float64
	LastTime     time.Time
	IsSuspicious bool
}

type AntiSpoofingValidator struct {
	rdb   *redis.Client
	cache map[string]*DeviceHistory
	mu    sync.RWMutex
}

func NewAntiSpoofingValidator(rdb *redis.Client) *AntiSpoofingValidator {
	return &AntiSpoofingValidator{
		rdb:   rdb,
		cache: make(map[string]*DeviceHistory),
	}
}

func (asv *AntiSpoofingValidator) ValidateGPSTravel(ctx context.Context, deviceHash string, lat, lng float64, timestamp time.Time) (bool, string) {
	asv.mu.Lock()
	defer asv.mu.Unlock()

	history, exists := asv.cache[deviceHash]
	
	if !exists {
		history = &DeviceHistory{
			LastLat:  lat,
			LastLng:  lng,
			LastTime: timestamp,
		}
		asv.cache[deviceHash] = history
		asv.persistHistory(ctx, deviceHash, history)
		return true, "First GPS point recorded"
	}

	if history.IsSuspicious {
		return false, "Device marked as suspicious - GPS blocked"
	}

	timeElapsed := timestamp.Sub(history.LastTime).Seconds()
	
	if timeElapsed <= 0 {
		timeElapsed = 1
	}

	distance := asv.calculateDistance(history.LastLat, history.LastLng, lat, lng)
	speed := (distance / 1000.0) / (timeElapsed / 3600.0)

	if speed > MaxSpeedKmh {
		history.IsSuspicious = true
		asv.cache[deviceHash] = history
		asv.persistHistory(ctx, deviceHash, history)
		
		logger.Warn("AntiSpoofing", "GPS time travel detected | Device: %s | Speed: %.2f km/h | Distance: %.2f m | Time: %.2f s",
			deviceHash, speed, distance, timeElapsed)
		
		return false, fmt.Sprintf("GPS time travel detected: %.2f km/h is impossible", speed)
	}

	if timeElapsed <= 5 && distance > MaxDistanceFor5s {
		history.IsSuspicious = true
		asv.cache[deviceHash] = history
		asv.persistHistory(ctx, deviceHash, history)
		
		logger.Warn("AntiSpoofing", "GPS jump detected | Device: %s | Distance: %.2f m | Time: %.2f s",
			deviceHash, distance, timeElapsed)
		
		return false, fmt.Sprintf("GPS jump detected: %.2f m in %.2f s is impossible", distance, timeElapsed)
	}

	history.LastLat = lat
	history.LastLng = lng
	history.LastTime = timestamp
	asv.cache[deviceHash] = history
	asv.persistHistory(ctx, deviceHash, history)

	return true, "GPS validated"
}

func (asv *AntiSpoofingValidator) calculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371000.0 // Raio da Terra em metros
	
	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

func (asv *AntiSpoofingValidator) persistHistory(ctx context.Context, deviceHash string, history *DeviceHistory) {
	if asv.rdb == nil {
		return
	}

	key := fmt.Sprintf("antispoof:%s", deviceHash)
	data := fmt.Sprintf("%f|%f|%d|%t", history.LastLat, history.LastLng, history.LastTime.Unix(), history.IsSuspicious)
	
	err := asv.rdb.Set(ctx, key, data, SuspiciousTTL).Err()
	if err != nil {
		logger.Warn("AntiSpoofing", "Failed to persist history: %v", err)
	}
}

func (asv *AntiSpoofingValidator) loadHistory(ctx context.Context, deviceHash string) (*DeviceHistory, error) {
	if asv.rdb == nil {
		return nil, fmt.Errorf("redis not available")
	}

	key := fmt.Sprintf("antispoof:%s", deviceHash)
	data, err := asv.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var lat, lng float64
	var timestamp int64
	var isSuspicious bool
	
	_, err = fmt.Sscanf(data, "%f|%f|%d|%t", &lat, &lng, &timestamp, &isSuspicious)
	if err != nil {
		return nil, err
	}

	return &DeviceHistory{
		LastLat:      lat,
		LastLng:      lng,
		LastTime:     time.Unix(timestamp, 0),
		IsSuspicious: isSuspicious,
	}, nil
}

func (asv *AntiSpoofingValidator) MarkSuspicious(ctx context.Context, deviceHash string, reason string) {
	asv.mu.Lock()
	defer asv.mu.Unlock()

	history, exists := asv.cache[deviceHash]
	if !exists {
		history = &DeviceHistory{}
	}
	
	history.IsSuspicious = true
	asv.cache[deviceHash] = history
	asv.persistHistory(ctx, deviceHash, history)

	logger.Warn("AntiSpoofing", "Device marked as suspicious | Device: %s | Reason: %s", deviceHash, reason)
}

func (asv *AntiSpoofingValidator) IsDeviceSuspicious(ctx context.Context, deviceHash string) bool {
	asv.mu.RLock()
	defer asv.mu.RUnlock()

	history, exists := asv.cache[deviceHash]
	if exists && history.IsSuspicious {
		return true
	}

	loaded, err := asv.loadHistory(ctx, deviceHash)
	if err == nil && loaded.IsSuspicious {
		asv.cache[deviceHash] = loaded
		return true
	}

	return false
}

func (asv *AntiSpoofingValidator) ClearSuspicious(ctx context.Context, deviceHash string) {
	asv.mu.Lock()
	defer asv.mu.Unlock()

	delete(asv.cache, deviceHash)
	
	if asv.rdb != nil {
		key := fmt.Sprintf("antispoof:%s", deviceHash)
		asv.rdb.Del(ctx, key)
	}

	logger.Info("AntiSpoofing", "Suspicious flag cleared | Device: %s", deviceHash)
}
