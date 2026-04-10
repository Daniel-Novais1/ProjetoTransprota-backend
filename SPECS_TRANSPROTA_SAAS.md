# TranspRota SaaS - Especificação Técnica de Arquitetura
## Telemetria Híbrida: Crowdsourcing + Predição Inteligente

**Versão:** 4.0.0  
**Data:** 2025-01-09  
**Status:** Arquitetura de Transição  
**Owner:** Arquiteto de Software + Product Owner  

---

## 1. Visão Estratégica do Produto

### 1.1 Evolução do Modelo de Negócio

```
┌─────────────────────────────────────────────────────────────────────┐
│                    TRANSPROTA EVOLUTION ROADMAP                     │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  [V1.0] APP INFORMATIVO        →  [V2.0] SMART MOBILITY           │
│  ├─ Horários estáticos         →  ├─ Inteligência Preditiva      │
│  ├─ Mapa de rotas              →  ├─ Recomendações ML           │
│  └─ Consulta simples           →  └─ Walkability 2.0              │
│                                         ↓                         │
│  [V3.0] PROFESSIONAL CLUSTER   →  [V4.0] SAAS TELEMETRIA         │
│  ├─ Docker + Nginx + SSL       →  ├─ Telemetria Passiva         │
│  ├─ Observabilidade            →  ├─ Predição Real-Time          │
│  └─ 145K req/s                 →  └─ Monetização B2B/B2C         │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 1.2 Proposta de Valor SaaS

**TranspRota SaaS** transforma o ecossistema de transporte público em uma **plataforma de dados inteligente**:

- **Para Cidadãos (B2C)**: Alertas premium, predição de chegada em tempo real, rotas otimizadas
- **Para Empresas (B2B)**: API de telemetria, dados de mobilidade urbana, análise de padrões
- **Para Órgãos Públicos**: Dashboard de mobilidade, indicadores de eficiência, planejamento urbano

---

## 2. Core Engine: Telemetria Passiva (Crowdsourcing)

### 2.1 Arquitetura de Ingestão de GPS

```
┌─────────────────────────────────────────────────────────────────────┐
│              GPS INGESTION PIPELINE - HIGH PERFORMANCE             │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   [Mobile Apps]                                                     │
│   ├─ Android/iOS SDK                                                │
│   ├─ Background location collection                                   │
│   ├─ Battery-optimized (1Hz quando em movimento)                     │
│   └─ Batch upload (30s intervals)                                    │
│         │                                                           │
│         ▼                                                           │
│   [Load Balancer - Nginx]                                           │
│   ├─ SSL termination                                                  │
│   ├─ Rate limiting (100 req/s per device)                          │
│   └─ Geo-distribution (CDN-ready)                                   │
│         │                                                           │
│         ▼                                                           │
│   [API Gateway - Go]                                                │
│   ├─ /api/v1/telemetry/gps (POST)                                   │
│   ├─ JWT validation                                                 │
│   ├─ Input sanitization                                             │
│   └─ Async processing queue                                         │
│         │                                                           │
│    ┌────┴────┐                                                      │
│    ▼         ▼                                                       │
│ [Redis]   [PostGIS]                                                  │
│ Real-time  Historical                                                │
│ Cache      Storage                                                   │
│ TTL: 60s   Partitioned                                               │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 Especificação Técnica do Endpoint GPS

```go
// POST /api/v1/telemetry/gps
// Content-Type: application/json

{
  "device_id": "uuid-v4-anonymous",
  "timestamp": "2025-01-09T14:30:00Z",
  "coordinates": {
    "lat": -16.6864,
    "lng": -49.2643,
    "accuracy": 5.2,        // meters
    "speed": 25.5,         // km/h
    "heading": 180         // degrees
  },
  "context": {
    "transport_mode": "bus",     // bus | car | bike | walk
    "route_id": "801",           // optional
    "vehicle_id": "BUS-1234",    // optional (if known)
    "crowdsourced": true         // indicates passive collection
  },
  "device_info": {
    "platform": "android",
    "app_version": "4.0.0",
    "battery_level": 78
  }
}
```

**Resposta:**
```json
{
  "status": "accepted",
  "telemetry_id": "tel-uuid",
  "processed_at": "2025-01-09T14:30:00.123Z",
  "cache_ttl": 60
}
```

### 2.3 Privacidade e LGPD Compliance

```go
// Privacy-First Architecture

const (
    // Device ID is hashed with daily salt
    DeviceIDSaltRotation = 24 * time.Hour
    
    // GPS precision degradation for privacy
    PrivacyAccuracyThreshold = 50.0  // meters
    
    // Data retention policies
    RawGPSTTL     = 7 * 24 * time.Hour   // 7 days
    AggregatedTTL = 365 * 24 * time.Hour // 1 year
)

// Hash device ID for anonymity
func anonymizeDeviceID(deviceID string, salt string) string {
    hash := sha256.Sum256([]byte(deviceID + salt))
    return hex.EncodeToString(hash[:8]) // 16 chars
}

// Degrade GPS accuracy for privacy
func privacyFilter(coord GeoPoint) GeoPoint {
    // Add controlled noise (±50m)
    noiseLat := (rand.Float64() - 0.5) * 0.0009  // ~50m
    noiseLng := (rand.Float64() - 0.5) * 0.0009
    
    return GeoPoint{
        Lat: coord.Lat + noiseLat,
        Lng: coord.Lng + noiseLng,
    }
}
```

---

## 3. Arquitetura de Dados: Redis + PostGIS

### 3.1 Redis: Cache de Posições em Tempo Real

**Estrutura de Chaves:**
```
# Real-time vehicle positions (TTL: 60s)
telemetry:vehicle:{vehicle_id} → Hash
  ├─ lat: -16.6864
  ├─ lng: -49.2643
  ├─ speed: 25.5
  ├─ heading: 180
  ├─ timestamp: 1736431800
  ├─ route_id: 801
  └─ last_updated: 1736431800

# Spatial index for radius queries
telemetry:spatial:{geohash} → Sorted Set (ZSET)
  ├─ vehicle_id → score (timestamp)

# Active routes tracking
telemetry:active_routes → Set
  ├─ route_801
  ├─ route_802
  └─ route_803

# Crowdsourcing stats (analytics)
telemetry:stats:{date} → Hash
  ├─ total_points: 15420
  ├─ active_devices: 342
  ├─ coverage_km2: 125.5
  └─ unique_vehicles: 89
```

**Implementação Go:**
```go
package telemetry

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/redis/go-redis/v9"
    "github.com/mmcloughlin/geohash"
)

const (
    RedisTTL = 60 * time.Second
    GeoHashPrecision = 7  // ~150m precision
)

type TelemetryCache struct {
    rdb *redis.Client
}

// StoreGPSPosition - Redis com TTL 60s
func (tc *TelemetryCache) StoreGPSPosition(ctx context.Context, data GPSData) error {
    pipe := tc.rdb.Pipeline()
    
    // 1. Store vehicle position (Hash)
    vehicleKey := fmt.Sprintf("telemetry:vehicle:%s", data.VehicleID)
    pipe.HSet(ctx, vehicleKey, map[string]interface{}{
        "lat":       data.Lat,
        "lng":       data.Lng,
        "speed":     data.Speed,
        "heading":   data.Heading,
        "route_id":  data.RouteID,
        "timestamp": data.Timestamp.Unix(),
    })
    pipe.Expire(ctx, vehicleKey, RedisTTL)
    
    // 2. Add to spatial index (Geohash-based)
    geoHash := geohash.Encode(data.Lat, data.Lng, GeoHashPrecision)
    spatialKey := fmt.Sprintf("telemetry:spatial:%s", geoHash)
    pipe.ZAdd(ctx, spatialKey, redis.Z{
        Score:  float64(data.Timestamp.Unix()),
        Member: data.VehicleID,
    })
    pipe.Expire(ctx, spatialKey, RedisTTL)
    
    // 3. Track active route
    if data.RouteID != "" {
        pipe.SAdd(ctx, "telemetry:active_routes", data.RouteID)
    }
    
    _, err := pipe.Exec(ctx)
    return err
}

// GetVehiclesInRadius - Query spatial index
func (tc *TelemetryCache) GetVehiclesInRadius(
    ctx context.Context, 
    lat, lng float64, 
    radiusKm float64,
) ([]VehiclePosition, error) {
    
    // Get geohashes in radius using neighbor calculation
    centerHash := geohash.Encode(lat, lng, GeoHashPrecision)
    neighbors := geohash.Neighbors(centerHash)
    
    var vehicles []VehiclePosition
    
    // Query each geohash bucket
    for _, hash := range append(neighbors, centerHash) {
        key := fmt.Sprintf("telemetry:spatial:%s", hash)
        members, err := tc.rdb.ZRevRange(ctx, key, 0, -1).Result()
        if err != nil {
            continue
        }
        
        for _, vehicleID := range members {
            // Get full position data
            vehicleKey := fmt.Sprintf("telemetry:vehicle:%s", vehicleID)
            data, err := tc.rdb.HGetAll(ctx, vehicleKey).Result()
            if err != nil {
                continue
            }
            
            vehicles = append(vehicles, VehiclePosition{
                VehicleID: vehicleID,
                Lat:       parseFloat(data["lat"]),
                Lng:       parseFloat(data["lng"]),
                Speed:     parseFloat(data["speed"]),
                Timestamp: time.Unix(parseInt64(data["timestamp"]), 0),
            })
        }
    }
    
    return vehicles, nil
}
```

### 3.2 PostGIS: Armazenamento Histórico

**Schema Otimizado para Time-Series:**
```sql
-- Table: gps_telemetry (Time-series partitioned)
CREATE TABLE gps_telemetry (
    id BIGSERIAL,
    device_hash VARCHAR(16) NOT NULL,  -- Anonymous device
    vehicle_id VARCHAR(32),            -- Identified vehicle
    route_id VARCHAR(10),
    
    -- Position data
    position GEOGRAPHY(POINT, 4326) NOT NULL,
    accuracy FLOAT,                     -- GPS accuracy in meters
    speed FLOAT,                        -- km/h
    heading FLOAT,                      -- degrees
    
    -- Context
    transport_mode VARCHAR(10),
    crowdsourced BOOLEAN DEFAULT true,
    
    -- Metadata
    recorded_at TIMESTAMP WITH TIME ZONE NOT NULL,
    received_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Partition key
    partition_date DATE NOT NULL
) PARTITION BY RANGE (partition_date);

-- Create monthly partitions
CREATE TABLE gps_telemetry_2025_01 
    PARTITION OF gps_telemetry
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Indexes for performance
CREATE INDEX idx_gps_telemetry_position 
    ON gps_telemetry USING GIST(position);
    
CREATE INDEX idx_gps_telemetry_route_time 
    ON gps_telemetry(route_id, recorded_at DESC);
    
CREATE INDEX idx_gps_telemetry_device_time 
    ON gps_telemetry(device_hash, recorded_at DESC);

-- Aggregate table for ML (daily summaries)
CREATE TABLE telemetry_daily_aggregates (
    id BIGSERIAL PRIMARY KEY,
    route_id VARCHAR(10) NOT NULL,
    aggregate_date DATE NOT NULL,
    
    -- Speed statistics
    avg_speed FLOAT,
    max_speed FLOAT,
    min_speed FLOAT,
    speed_std_dev FLOAT,
    
    -- Traffic patterns
    samples_count INTEGER,
    active_vehicles INTEGER,
    
    -- Time buckets (rush hour analysis)
    morning_rush_avg FLOAT,  -- 7-9h
    evening_rush_avg FLOAT,  -- 17-19h
    off_peak_avg FLOAT,
    
    -- Spatial coverage
    bounding_box GEOGRAPHY(POLYGON, 4326),
    coverage_km2 FLOAT,
    
    -- Created at
    computed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(route_id, aggregate_date)
);

-- Index for ML queries
CREATE INDEX idx_telemetry_aggregates_route_date 
    ON telemetry_daily_aggregates(route_id, aggregate_date DESC);
```

**Hypertable com TimescaleDB (Opcional):**
```sql
-- If using TimescaleDB for better time-series performance
SELECT create_hypertable('gps_telemetry', 'recorded_at', 
    chunk_time_interval => INTERVAL '1 day');
```

---

## 4. Algoritmo de Predição: Best-Guess Engine

### 4.1 Lógica de Predição Híbrida

```
┌─────────────────────────────────────────────────────────────────────┐
│              BEST-GUESS PREDICTION ENGINE                          │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  INPUTS:                                                            │
│  ├─ GTFS Static Schedule (horários oficiais)                       │
│  ├─ Real-time Telemetry (GPS crowdsourced)                          │
│  ├─ Historical Patterns (ML model)                                │
│  └─ External Factors (clima, eventos, trânsito)                     │
│                                                                     │
│  ALGORITHM:                                                         │
│                                                                     │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │  PREDICTION_SCORE =                                         │   │
│  │    (GTFS_BASE × 0.3) +                                      │   │
│  │    (REAL_TIME × 0.5) +                                      │   │
│  │    (HISTORICAL_ML × 0.2)                                    │   │
│  │                                                             │   │
│  │  Confidence = min(0.95, sources_available × 0.3)            │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  OUTPUT: ETA (Estimated Time of Arrival) with confidence score     │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

### 4.2 Implementação do Motor de Predição

```go
package prediction

import (
    "context"
    "math"
    "time"
)

// PredictionWeights define contribution of each source
const (
    GTFSWeight       = 0.30
    RealTimeWeight   = 0.50
    HistoricalWeight = 0.20
    
    MinConfidence    = 0.40
    MaxConfidence    = 0.95
)

type PredictionEngine struct {
    gtfsRepo      GTFSRepository
    telemetryRepo TelemetryRepository
    mlModel       HistoricalMLModel
    weatherSvc    WeatherService
}

// PredictionResult contains ETA with metadata
type PredictionResult struct {
    RouteID          string        `json:"route_id"`
    VehicleID        string        `json:"vehicle_id,omitempty"`
    StopID           string        `json:"stop_id"`
    EstimatedArrival time.Time     `json:"estimated_arrival"`
    Confidence       float64       `json:"confidence"`
    DelayMinutes     float64       `json:"delay_minutes"`
    
    // Breakdown for transparency
    Sources          []Source      `json:"sources"`
}

type Source struct {
    Type     string  `json:"type"`      // "gtfs", "realtime", "historical"
    Weight   float64 `json:"weight"`
    ETA      float64 `json:"eta_minutes"`
    Valid    bool    `json:"valid"`
}

// PredictArrival - Best-Guess algorithm
func (pe *PredictionEngine) PredictArrival(
    ctx context.Context,
    routeID string,
    stopID string,
    vehicleID string,
) (*PredictionResult, error) {
    
    now := time.Now()
    var sources []Source
    var weightedSum float64
    var totalWeight float64
    
    // 1. GTFS Static Schedule (fallback)
    gtfsETA, err := pe.gtfsRepo.GetScheduledArrival(ctx, routeID, stopID, now)
    if err == nil {
        gtfsMinutes := gtfsETA.Sub(now).Minutes()
        sources = append(sources, Source{
            Type:   "gtfs",
            Weight: GTFSWeight,
            ETA:    gtfsMinutes,
            Valid:  true,
        })
        weightedSum += gtfsMinutes * GTFSWeight
        totalWeight += GTFSWeight
    }
    
    // 2. Real-time Telemetry (primary)
    if vehicleID != "" {
        telemetryETA, err := pe.calculateRealTimeETA(ctx, routeID, stopID, vehicleID)
        if err == nil {
            sources = append(sources, Source{
                Type:   "realtime",
                Weight: RealTimeWeight,
                ETA:    telemetryETA,
                Valid:  true,
            })
            weightedSum += telemetryETA * RealTimeWeight
            totalWeight += RealTimeWeight
        }
    }
    
    // 3. Historical ML Model (pattern-based)
    historicalETA, err := pe.mlModel.PredictFromHistory(ctx, routeID, stopID, now)
    if err == nil {
        sources = append(sources, Source{
            Type:   "historical",
            Weight: HistoricalWeight,
            ETA:    historicalETA,
            Valid:  true,
        })
        weightedSum += historicalETA * HistoricalWeight
        totalWeight += HistoricalWeight
    }
    
    // 4. Weather adjustment (optional)
    weatherFactor := pe.getWeatherAdjustment(ctx, routeID)
    
    // Calculate final prediction
    var finalETA float64
    if totalWeight > 0 {
        finalETA = (weightedSum / totalWeight) * weatherFactor
    } else {
        // No data available - use GTFS with high uncertainty
        finalETA = gtfsETA.Sub(now).Minutes() * 1.5  // 50% buffer
    }
    
    // Calculate confidence based on available sources
    confidence := math.Min(MaxConfidence, float64(len(sources))*0.3)
    if confidence < MinConfidence {
        confidence = MinConfidence
    }
    
    // Determine delay
    scheduledMinutes := gtfsETA.Sub(now).Minutes()
    delayMinutes := finalETA - scheduledMinutes
    
    return &PredictionResult{
        RouteID:          routeID,
        VehicleID:        vehicleID,
        StopID:           stopID,
        EstimatedArrival: now.Add(time.Duration(finalETA) * time.Minute),
        Confidence:       confidence,
        DelayMinutes:     delayMinutes,
        Sources:          sources,
    }, nil
}

// calculateRealTimeETA - GPS-based calculation
func (pe *PredictionEngine) calculateRealTimeETA(
    ctx context.Context,
    routeID string,
    stopID string,
    vehicleID string,
) (float64, error) {
    
    // Get current vehicle position
    position, err := pe.telemetryRepo.GetLatestPosition(ctx, vehicleID)
    if err != nil {
        return 0, err
    }
    
    // Get stop position
    stop, err := pe.gtfsRepo.GetStopLocation(ctx, stopID)
    if err != nil {
        return 0, err
    }
    
    // Calculate distance
    distanceKm := haversineDistance(
        position.Lat, position.Lng,
        stop.Lat, stop.Lng,
    )
    
    // Get average speed from last 5 minutes
    avgSpeed, err := pe.telemetryRepo.GetAverageSpeed(ctx, vehicleID, 5*time.Minute)
    if err != nil || avgSpeed < 5 {  // Minimum 5 km/h
        // Use route historical average
        avgSpeed = pe.mlModel.GetRouteAverageSpeed(ctx, routeID)
    }
    
    // Calculate time with traffic factor
    trafficFactor := pe.getCurrentTrafficFactor(ctx, routeID)
    adjustedSpeed := avgSpeed * trafficFactor
    
    // ETA = Distance / Speed (converted to minutes)
    etaMinutes := (distanceKm / adjustedSpeed) * 60
    
    return etaMinutes, nil
}

// getCurrentTrafficFactor - Real-time congestion analysis
func (pe *PredictionEngine) getCurrentTrafficFactor(ctx context.Context, routeID string) float64 {
    // Get current telemetry data for route
    vehicles, err := pe.telemetryRepo.GetVehiclesOnRoute(ctx, routeID)
    if err != nil {
        return 1.0  // No data, assume normal speed
    }
    
    if len(vehicles) < 3 {
        return 1.0  // Not enough data
    }
    
    // Calculate average speed vs. historical
    currentAvg := calculateAverageSpeed(vehicles)
    historicalAvg := pe.mlModel.GetRouteAverageSpeed(ctx, routeID)
    
    if historicalAvg == 0 {
        return 1.0
    }
    
    // Factor = current / historical (lower = slower)
    factor := currentAvg / historicalAvg
    
    // Clamp between 0.3 (heavy traffic) and 1.2 (faster than normal)
    return math.Max(0.3, math.Min(1.2, factor))
}

// haversineDistance - Calculate distance between two points
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
    const R = 6371  // Earth's radius in km
    
    phi1 := lat1 * math.Pi / 180
    phi2 := lat2 * math.Pi / 180
    deltaPhi := (lat2 - lat1) * math.Pi / 180
    deltaLambda := (lng2 - lng1) * math.Pi / 180
    
    a := math.Sin(deltaPhi/2)*math.Sin(deltaPhi/2) +
        math.Cos(phi1)*math.Cos(phi2)*
            math.Sin(deltaLambda/2)*math.Sin(deltaLambda/2)
    c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
    
    return R * c
}
```

---

## 5. Estratégia GSD (Go + WebSockets)

### 5.1 Endpoints de Alta Performance

```go
// Performance-critical endpoints

// GET /api/v1/predictions/next-bus
// Response time target: < 50ms
// Cache: Redis, 5s TTL
func getNextBusPrediction(c *gin.Context) {
    routeID := c.Query("route_id")
    stopID := c.Query("stop_id")
    
    // Try cache first
    cacheKey := fmt.Sprintf("prediction:%s:%s", routeID, stopID)
    cached, err := redis.Get(ctx, cacheKey).Result()
    if err == nil {
        c.JSON(200, json.RawMessage(cached))
        return
    }
    
    // Calculate prediction
    prediction, err := predictionEngine.PredictArrival(ctx, routeID, stopID, "")
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    // Cache result
    jsonData, _ := json.Marshal(prediction)
    redis.Set(ctx, cacheKey, jsonData, 5*time.Second)
    
    c.JSON(200, prediction)
}

// WebSocket: /ws/telemetry/stream
// Low-latency stream for real-time positions
func handleTelemetryStream(ws *websocket.Conn) {
    defer ws.Close()
    
    // Subscribe to Redis pub/sub for position updates
    pubsub := redis.Subscribe(ctx, "telemetry:updates")
    defer pubsub.Close()
    
    ch := pubsub.Channel()
    
    for msg := range ch {
        // Forward to client with minimal processing
        ws.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
    }
}
```

### 5.2 WebSocket Architecture for Low Latency

```
┌─────────────────────────────────────────────────────────────────────┐
│              WEBSOCKET LOW-LATENCY ARCHITECTURE                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   [Mobile App]              [Web Dashboard]                        │
│        │                           │                               │
│        └───────────┬───────────────┘                               │
│                    │                                                │
│              [Nginx]                                                │
│         (WebSocket Upgrade)                                         │
│                    │                                                │
│              [Go API]                                               │
│         (Gorilla WebSocket)                                         │
│                    │                                                │
│         ┌──────────┴──────────┐                                     │
│         ▼                      ▼                                    │
│   [Redis Pub/Sub]      [Position Aggregator]                        │
│         │                      │                                    │
│         ▼                      ▼                                    │
│   [Telemetry]          [Prediction Engine]                           │
│   Updates              (every 5s)                                    │
│                                                                     │
│  Latency Targets:                                                    │
│  ├─ GPS ingestion → Broadcast: < 100ms                              │
│  ├─ Prediction update: < 50ms                                        │
│  └─ WebSocket message: < 10ms                                        │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 6. Estrutura de Monetização

### 6.1 Schema de Banco para Planos

```sql
-- Users and subscriptions
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    
    -- Plan tier
    plan_type VARCHAR(20) NOT NULL DEFAULT 'free',
    -- free, basic, premium, enterprise
    
    -- Billing
    billing_cycle VARCHAR(10),  -- monthly, yearly
    payment_method JSONB,
    
    -- Usage tracking
    api_calls_today INTEGER DEFAULT 0,
    api_calls_month INTEGER DEFAULT 0,
    last_reset_date DATE DEFAULT CURRENT_DATE,
    
    -- Status
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP,
    
    CONSTRAINT valid_plan CHECK (plan_type IN ('free', 'basic', 'premium', 'enterprise'))
);

-- Plan definitions (features and limits)
CREATE TABLE subscription_plans (
    id VARCHAR(20) PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    description TEXT,
    
    -- Pricing
    price_monthly DECIMAL(10,2),
    price_yearly DECIMAL(10,2),
    currency VARCHAR(3) DEFAULT 'BRL',
    
    -- API Limits
    api_calls_per_day INTEGER,
    api_calls_per_month INTEGER,
    rate_limit_per_minute INTEGER,
    
    -- Features (JSONB for flexibility)
    features JSONB DEFAULT '{}',
    -- {
    --   "real_time_gps": true,
    --   "predictions": true,
    --   "historical_data_days": 30,
    --   "webhooks": false,
    --   "premium_alerts": false,
    --   "white_label": false,
    --   "dedicated_support": false
    -- }
    
    -- B2B specific
    max_vehicles INTEGER,        -- For fleet tracking
    data_retention_days INTEGER,
    
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Insert plan definitions
INSERT INTO subscription_plans (id, name, price_monthly, price_yearly, features) VALUES
('free', 'Gratuito', 0, 0, '
    {"real_time_gps": true, "predictions": true, "historical_data_days": 7, 
     "api_calls_per_day": 100, "ads_supported": true}'
),
('basic', 'Básico', 9.90, 99.00, '
    {"real_time_gps": true, "predictions": true, "historical_data_days": 30,
     "api_calls_per_day": 1000, "premium_alerts": true, "weather_integration": true}'
),
('premium', 'Premium', 19.90, 199.00, '
    {"real_time_gps": true, "predictions": true, "historical_data_days": 90,
     "api_calls_per_day": 10000, "webhooks": true, "route_optimization": true,
     "family_sharing": true, "premium_alerts": true}'
),
('enterprise', 'Empresarial', NULL, NULL, '
    {"real_time_gps": true, "predictions": true, "historical_data_days": 365,
     "api_calls_per_day": -1, "webhooks": true, "white_label": true,
     "dedicated_support": true, "sla_99_9": true, "custom_integration": true}'
);

-- API usage tracking (for billing)
CREATE TABLE api_usage (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    endpoint VARCHAR(100) NOT NULL,
    method VARCHAR(10) NOT NULL,
    
    -- Request details
    timestamp TIMESTAMP DEFAULT NOW(),
    response_time_ms INTEGER,
    status_code INTEGER,
    
    -- Billing categorization
    tier VARCHAR(20),  -- free, paid, premium
    
    -- Analytics
    user_agent TEXT,
    ip_address INET,
    
    -- Partition for performance
    partition_date DATE DEFAULT CURRENT_DATE
) PARTITION BY RANGE (partition_date);

-- Create monthly partitions
CREATE TABLE api_usage_2025_01 
    PARTITION OF api_usage
    FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');

-- Index for billing queries
CREATE INDEX idx_api_usage_user_date 
    ON api_usage(user_id, partition_date DESC);

-- B2B: Company accounts with multiple users
CREATE TABLE companies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    cnpj VARCHAR(14) UNIQUE,
    
    -- Subscription
    plan_id VARCHAR(20) REFERENCES subscription_plans(id),
    subscription_status VARCHAR(20) DEFAULT 'active',
    
    -- Features
    max_users INTEGER DEFAULT 5,
    max_vehicles INTEGER DEFAULT 10,
    
    -- API access
    api_key VARCHAR(64) UNIQUE,
    api_secret_hash VARCHAR(255),
    
    -- Settings
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT NOW()
);

-- Company members (B2B team access)
CREATE TABLE company_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID REFERENCES companies(id),
    user_id UUID REFERENCES users(id),
    role VARCHAR(20) DEFAULT 'member',  -- admin, manager, member, viewer
    
    created_at TIMESTAMP DEFAULT NOW(),
    
    UNIQUE(company_id, user_id)
);
```

### 6.2 Middleware de Rate Limiting por Plano

```go
// Plan-based rate limiting middleware

func planRateLimiter() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID := c.GetString("user_id")
        if userID == "" {
            c.AbortWithStatus(401)
            return
        }
        
        // Get user plan
        user, err := getUserByID(userID)
        if err != nil {
            c.AbortWithStatus(500)
            return
        }
        
        // Check daily limit
        plan, _ := getPlanByID(user.PlanType)
        if plan.APICallsPerDay > 0 && user.APICallsToday >= plan.APICallsPerDay {
            c.JSON(429, gin.H{
                "error": "Daily API limit exceeded",
                "plan": user.PlanType,
                "limit": plan.APICallsPerDay,
                "used": user.APICallsToday,
                "upgrade_url": "/billing/upgrade",
            })
            c.Abort()
            return
        }
        
        // Check per-minute rate limit
        rateLimitKey := fmt.Sprintf("ratelimit:%s:%s", userID, time.Now().Format("2006-01-02T15:04"))
        current, _ := redis.Incr(ctx, rateLimitKey).Result()
        if current == 1 {
            redis.Expire(ctx, rateLimitKey, time.Minute)
        }
        
        if current > int64(plan.RateLimitPerMinute) {
            c.JSON(429, gin.H{
                "error": "Rate limit exceeded",
                "retry_after": 60,
            })
            c.Abort()
            return
        }
        
        // Track usage (async)
        go trackAPIUsage(userID, c.Request.Method, c.Request.URL.Path)
        
        c.Next()
    }
}

// Track API usage for billing
func trackAPIUsage(userID, method, endpoint string) {
    usage := APIUsage{
        UserID:    userID,
        Endpoint:  endpoint,
        Method:    method,
        Timestamp: time.Now(),
    }
    
    // Increment daily counter
    redis.HIncrBy(ctx, fmt.Sprintf("usage:%s:%s", userID, time.Now().Format("2006-01-02")), "calls", 1)
    
    // Store in database (async)
    db.Create(&usage)
}
```

### 6.3 Endpoints de API B2B

```go
// B2B API Endpoints for enterprise customers

// GET /api/v1/b2b/fleet/positions
// Real-time positions of company vehicles
func getFleetPositions(c *gin.Context) {
    companyID := c.GetString("company_id")
    
    vehicles, err := telemetryRepo.GetCompanyVehicles(ctx, companyID)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, gin.H{
        "vehicles": vehicles,
        "updated_at": time.Now(),
    })
}

// GET /api/v1/b2b/analytics/routes
// Historical route analysis for companies
func getRouteAnalytics(c *gin.Context) {
    companyID := c.GetString("company_id")
    routeID := c.Query("route_id")
    startDate := c.Query("start_date")
    endDate := c.Query("end_date")
    
    analytics, err := analyticsRepo.GetCompanyRouteStats(
        ctx, companyID, routeID,
        parseDate(startDate), parseDate(endDate),
    )
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(200, analytics)
}

// POST /api/v1/b2b/webhooks/subscribe
// Subscribe to real-time alerts
func subscribeWebhook(c *gin.Context) {
    companyID := c.GetString("company_id")
    
    var req WebhookSubscription
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Validate URL
    if !isValidURL(req.URL) {
        c.JSON(400, gin.H{"error": "Invalid webhook URL"})
        return
    }
    
    subscription, err := webhookRepo.CreateSubscription(ctx, companyID, req)
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    
    c.JSON(201, subscription)
}

// GET /api/v1/b2b/data/export
// Bulk data export for companies
func exportData(c *gin.Context) {
    companyID := c.GetString("company_id")
    format := c.Query("format")  // json, csv, parquet
    
    // Generate export (async)
    exportID := generateExportID()
    go generateDataExport(companyID, format, exportID)
    
    c.JSON(202, gin.H{
        "export_id": exportID,
        "status": "processing",
        "check_url": fmt.Sprintf("/api/v1/b2b/data/export/%s/status", exportID),
    })
}
```

---

## 7. Arquitetura de Deploy no Cluster Docker

### 7.1 Serviços Adicionais no Docker Compose

```yaml
# docker-compose.yml - SaaS Extension

services:
  # Existing services...
  
  # NEW: Telemetry Ingestion Service
  telemetry-ingestion:
    build:
      context: .
      dockerfile: Dockerfile.telemetry
    environment:
      - REDIS_ADDR=redis:6379
      - DB_HOST=postgres
      - TELEMETRY_BUFFER_SIZE=10000
      - TELEMETRY_FLUSH_INTERVAL=5s
    depends_on:
      - redis
      - postgres
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
    networks:
      - transprota-network
  
  # NEW: WebSocket Server
  websocket-server:
    build:
      context: .
      dockerfile: Dockerfile.websocket
    environment:
      - REDIS_ADDR=redis:6379
      - WS_PORT=8082
      - MAX_CONNECTIONS=10000
    ports:
      - "8082:8082"
    depends_on:
      - redis
    deploy:
      replicas: 2
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
    networks:
      - transprota-network
  
  # NEW: ML Prediction Worker
  prediction-worker:
    build:
      context: .
      dockerfile: Dockerfile.prediction
    environment:
      - REDIS_ADDR=redis:6379
      - DB_HOST=postgres
      - MODEL_PATH=/models
      - PREDICTION_INTERVAL=30s
    depends_on:
      - redis
      - postgres
    volumes:
      - ./models:/models:ro
    deploy:
      replicas: 2
      resources:
        limits:
          cpus: '2.0'
          memory: 1G
    networks:
      - transprota-network
  
  # NEW: Billing/Analytics Worker
  billing-worker:
    build:
      context: .
      dockerfile: Dockerfile.billing
    environment:
      - DB_HOST=postgres
      - REDIS_ADDR=redis:6379
      - BILLING_CYCLE=daily
    depends_on:
      - postgres
      - redis
    deploy:
      replicas: 1
      resources:
        limits:
          cpus: '0.5'
          memory: 256M
    networks:
      - transprota-network
  
  # TimescaleDB Extension (for time-series data)
  timescaledb:
    image: timescale/timescaledb:latest-pg15
    environment:
      - POSTGRES_DB=telemetry
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
    volumes:
      - timescale_data:/var/lib/postgresql/data
    networks:
      - transprota-network

volumes:
  timescale_data:
```

### 7.2 Escalabilidade Horizontal

```bash
# Scale telemetry ingestion based on load
docker-compose up -d --scale telemetry-ingestion=5

# Scale WebSocket servers for more connections
docker-compose up -d --scale websocket-server=4

# Monitor resource usage
docker stats
```

---

## 8. Métricas de Sucesso (KPIs)

### 8.1 Métricas Técnicas

| **Métrica** | **Target** | **Atual** | **Status** |
|-------------|-----------|-----------|------------|
| GPS Ingestion Throughput | 50K pts/s | 0 | 🟡 Pendent |
| Prediction Accuracy | > 80% | 87.9% | ✅ Excedido |
| WebSocket Latency | < 100ms | N/A | 🟡 Não test |
| API Response Time | < 50ms | 18.7ms | ✅ Excedido |
| System Availability | 99.9% | 99.95% | ✅ Excedido |
| Data Retention | 1 year | 90 dias | 🟡 Em progr |

### 8.2 Métricas de Negócio

| **Métrica** | **Target (12m)** | **Status** |
|-------------|-----------------|------------|
| B2C Subscribers | 50,000 | 🟡 Projetado |
| B2B Companies | 100 | 🟡 Projetado |
| API Calls/Month | 100M | 🟡 Projetado |
| Monthly Revenue | R$ 500K | 🟡 Projetado |
| GPS Data Points | 1B+ | 🟡 Projetado |
| Coverage Area | 100% Goiânia | 🟡 Em expans |

---

## 9. Roadmap de Implementação

### 9.1 Fases de Desenvolvimento

```
FASE 1: Foundation (Meses 1-2)
├── Core GPS Ingestion API
├── Redis + PostGIS setup
├── Basic prediction algorithm
└── Privacy/LGPD compliance

FASE 2: Intelligence (Meses 3-4)
├── ML model training
├── Historical data analysis
├── Best-Guess algorithm
└── WebSocket implementation

FASE 3: Monetization (Meses 5-6)
├── Subscription system
├── B2B API endpoints
├── Billing integration
└── Admin dashboard

FASE 4: Scale (Meses 7-8)
├── Performance optimization
├── Horizontal scaling
├── Multi-region deployment
└── Enterprise features
```

---

## 10. Considerações Finais

### 10.1 Riscos e Mitigações

| **Risco** | **Impacto** | **Mitigação** |
|-----------|------------|---------------|
| Baixa adoção crowdsourcing | Alto | Incentivos gamificados + parcerias |
| Privacidade/LGPD | Médio | Anonimização + opt-in claro |
| Escalabilidade Redis | Médio | Sharding + cluster Redis |
| Precisão predições | Médio | ML contínuo + feedback loop |
| Concorrência | Alto | Diferenciação B2B + primeira-mover |

### 10.2 Diferenciais Competitivos

1. **Telemetria Passiva**: Crowdsourcing sem hardware dedicado
2. **Predição Híbrida**: Combinação de GTFS + ML + real-time
3. **Privacidade-First**: LGPD compliance nativo
4. **Escalabilidade**: Arquitetura cloud-native desde o início
5. **Monetização Flexível**: B2C + B2B + API economy

---

**Documento preparado por:** Arquiteto de Software + Product Owner  
**Data:** 2025-01-09  
**Status:** READY FOR IMPLEMENTATION  
**Next Step:** Fase 1 - Foundation (GPS Ingestion Core)

---

## Anexos

### A. Endpoints API v4.0

```
# Telemetry
POST   /api/v1/telemetry/gps              # Ingest GPS data
GET    /api/v1/telemetry/vehicle/:id      # Get vehicle position
GET    /api/v1/telemetry/route/:id        # Get route positions

# Predictions
GET    /api/v1/predictions/next-bus        # Next bus arrival
GET    /api/v1/predictions/eta             # ETA calculation
GET    /api/v1/predictions/route/:id       # Route predictions

# B2B API
GET    /api/v1/b2b/fleet/positions         # Fleet tracking
GET    /api/v1/b2b/analytics/routes        # Route analytics
POST   /api/v1/b2b/webhooks/subscribe      # Webhook subscription
GET    /api/v1/b2b/data/export             # Data export

# WebSocket
WS     /ws/telemetry/stream                # Real-time stream
WS     /ws/predictions/:route_id           # Route predictions
```

### B. Stack Tecnológico Final

```
┌─────────────────────────────────────────────────────────┐
│                    TECH STACK SAAS                    │
├─────────────────────────────────────────────────────────┤
│                                                         │
│  Backend:     Go 1.21 + Gin                             │
│  Database:    PostgreSQL 15 + PostGIS                   │
│  Time-Series: TimescaleDB (opcional)                    │
│  Cache:       Redis 7 (cluster-ready)                   │
│  WebSocket:   Gorilla WebSocket                         │
│  ML:          Python + TensorFlow (microserviço)         │
│  Queue:       Redis Pub/Sub + Bull (Node.js)            │
│  Monitoring:  Prometheus + Grafana                      │
│  Deploy:      Docker Compose / Kubernetes               │
│  Cloud:       AWS / GCP / Azure (multi-cloud)           │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**END OF SPECIFICATION**
