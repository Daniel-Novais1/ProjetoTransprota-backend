package telemetry

import "time"

// ============================================================================
// TELEMETRY SAAS - FASE 1: FOUNDATION
// Telemetria Passiva (Crowdsourcing) com LGPD Compliance
// ============================================================================

const (
	// Redis TTL para última posição conhecida (10 minutos para cache ativo de alta frequência)
	RedisLastPosTTL = 10 * time.Minute

	// Sal rotação para hash de device (LGPD)
	DeviceHashSaltRotation = 24 * time.Hour

	// Redis TTL para consultas de conformidade (5 minutos para dados frescos)
	ComplianceCacheTTL = 5 * time.Minute

	// Chaves de cache para consultas de conformidade
	CacheKeyFleetStatus = "compliance:fleet_status"
	CacheKeyHistory     = "compliance:history:"

	// Limites de validação
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
	DeviceID string `json:"deviceId" binding:"required"`

	// Timestamp de quando o dado foi coletado no dispositivo
	RecordedAt time.Time `json:"recordedAt" binding:"required"`

	// Coordenadas geográficas (padronizado: lat, lng)
	Latitude  float64 `json:"lat" binding:"required"`
	Longitude float64 `json:"lng" binding:"required"`

	// Métricas de movimento
	Speed    float64 `json:"speed,omitempty"`    // km/h
	Heading  float64 `json:"heading,omitempty"`  // graus (0-360)
	Accuracy float64 `json:"accuracy,omitempty"` // metros

	// Contexto do transporte
	TransportMode string `json:"transportMode,omitempty"` // bus | car | bike | walk | metro
	RouteID       string `json:"routeId,omitempty"`       // código da rota (opcional)

	// Metadados do dispositivo
	BatteryLevel int    `json:"batteryLevel,omitempty"` // 0-100
	Platform     string `json:"platform,omitempty"`     // android | ios
	AppVersion   string `json:"appVersion,omitempty"`   // ex: "4.0.0"
}

// TelemetryResponse resposta do endpoint
type TelemetryResponse struct {
	Status      string    `json:"status"`
	TelemetryID string    `json:"telemetryId"`
	DeviceHash  string    `json:"deviceHash"` // Hash retornado para confirmação
	Cached      bool      `json:"cached"`
	ProcessedAt time.Time `json:"processedAt"`
	TTL         int       `json:"ttlSeconds"`
}

// ValidationError erro de validação detalhado
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// LatestPosition representa a posição mais recente de um dispositivo
// Usado em /api/v1/telemetry/latest
type LatestPosition struct {
	DeviceHash    string    `json:"deviceHash"`
	RouteID       string    `json:"routeId"`
	Speed         float64   `json:"speed"`
	Location      Location  `json:"location"` // lat/lng
	RecordedAt    time.Time `json:"recordedAt"`
	Platform      string    `json:"platform,omitempty"`   // android | ios
	AppVersion    string    `json:"appVersion,omitempty"` // ex: "4.0.0"
	TrafficStatus string    `json:"trafficStatus"`        // 'fluido', 'moderado', 'lento', 'desconhecido'
}

// Location representa coordenadas geográficas
type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// Geofence representa uma cerca eletrônica para monitoramento de veículos
type Geofence struct {
	ID        int64     `json:"id"`
	Nome      string    `json:"nome"`
	Tipo      string    `json:"tipo"`    // 'Terminal' ou 'Rota'
	Polygon   string    `json:"polygon"` // WKT POLYGON
	CreatedAt time.Time `json:"createdAt"`
}

// GeofenceAlert representa um alerta de geofencing disparado
type GeofenceAlert struct {
	ID           int64     `json:"id"`
	DeviceHash   string    `json:"deviceHash"`
	GeofenceID   int64     `json:"geofenceId"`
	GeofenceNome string    `json:"geofenceNome"`
	Estado       string    `json:"estado"` // 'In' ou 'Out'
	Lat          float64   `json:"lat"`
	Lng          float64   `json:"lng"`
	OcorridoEm   time.Time `json:"ocorridoEm"`
}
