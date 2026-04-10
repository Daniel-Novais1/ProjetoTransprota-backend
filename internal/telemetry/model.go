package telemetry

import "time"

// ============================================================================
// TELEMETRY SAAS - FASE 1: FOUNDATION
// Telemetria Passiva (Crowdsourcing) com LGPD Compliance
// ============================================================================

const (
	// Redis TTL para última posição conhecida
	RedisLastPosTTL = 60 * time.Second

	// Sal rotação para hash de device (LGPD)
	DeviceHashSaltRotation = 24 * time.Hour

	// Limites de validação (HACKER Squad)
	MaxUrbanSpeedKmh  = 120.0 // Velocidade máxima plausível em área urbana
	MaxAccuracyMeters = 100.0 // Precisão GPS máxima aceitável
	MinAccuracyMeters = 1.0   // Precisão mínima (evita dados falsos)
	MaxBatteryLevel   = 100   // Nível máximo de bateria
	MinBatteryLevel   = 0     // Nível mínimo de bateria
)

// ============================================================================
// STRUCTS
// ============================================================================

// TelemetryPing representa um ping de telemetria GPS do usuário
type TelemetryPing struct {
	// Identificação do dispositivo (será anonimizado)
	DeviceID string `json:"device_id" binding:"required"`

	// Timestamp de quando o dado foi coletado no dispositivo
	RecordedAt time.Time `json:"recorded_at" binding:"required"`

	// Coordenadas geográficas
	Latitude  float64 `json:"lat" binding:"required"`
	Longitude float64 `json:"lng" binding:"required"`

	// Métricas de movimento
	Speed    float64 `json:"speed,omitempty"`    // km/h
	Heading  float64 `json:"heading,omitempty"`  // graus (0-360)
	Accuracy float64 `json:"accuracy,omitempty"` // metros

	// Contexto do transporte
	TransportMode string `json:"transport_mode,omitempty"` // bus | car | bike | walk | metro
	RouteID       string `json:"route_id,omitempty"`       // código da rota (opcional)

	// Metadados do dispositivo
	BatteryLevel int    `json:"battery_level,omitempty"` // 0-100
	Platform     string `json:"platform,omitempty"`      // android | ios
	AppVersion   string `json:"app_version,omitempty"`   // ex: "4.0.0"
}

// TelemetryResponse resposta do endpoint
type TelemetryResponse struct {
	Status      string    `json:"status"`
	TelemetryID string    `json:"telemetry_id"`
	DeviceHash  string    `json:"device_hash"` // Hash retornado para confirmação
	Cached      bool      `json:"cached"`
	ProcessedAt time.Time `json:"processed_at"`
	TTL         int       `json:"ttl_seconds"`
}

// ValidationError erro de validação detalhado
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// LatestPosition representa a última posição conhecida de um dispositivo
type LatestPosition struct {
	DeviceHash string    `json:"device_hash"`
	RouteID    string    `json:"route_id,omitempty"`
	Speed      float64   `json:"speed"`
	Location   Location  `json:"location"` // lat/lng
	RecordedAt time.Time `json:"recorded_at"`
}

// Location representa coordenadas geográficas
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
