package telemetry

import "time"

// ============================================================================
// GEOLOCATION & COLLECTIVE INTELLIGENCE MODELS
// ============================================================================

// BusUpdate representa uma atualização de posição de ônibus comunitário
// Enviada por usuários que estão em ônibus reais
type BusUpdate struct {
	UserID     string    `json:"user_id"`     // ID do usuário colaborador
	DeviceHash string    `json:"device_hash"` // Hash do dispositivo (LGPD)
	Latitude   float64   `json:"lat"`         // Latitude em graus decimais
	Longitude  float64   `json:"lng"`         // Longitude em graus decimais
	RouteID    string    `json:"route_id"`    // ID da linha (ex: "001", "EIXO")
	Speed      float64   `json:"speed"`       // Velocidade em km/h
	Heading    float64   `json:"heading"`     // Direção em graus (0-360)
	IsOnBus    bool      `json:"is_on_bus"`   // Confirmação se está no ônibus
	Occupancy  string    `json:"occupancy"`   // "empty", "low", "medium", "high", "full"
	TerminalID string    `json:"terminal_id"` // Terminal mais próximo
	Timestamp  time.Time `json:"timestamp"`   // Timestamp da atualização
}

// BusLocation representa a posição de um ônibus comunitário ativo
// Retornada para o frontend
type BusLocation struct {
	DeviceHash   string    `json:"dh"`   // Device hash (minificado)
	RouteID      string    `json:"rid"`  // Route ID (minificado)
	Latitude     float64   `json:"lat"`  // Latitude
	Longitude    float64   `json:"lng"`  // Longitude
	Speed        float64   `json:"spd"`  // Speed km/h
	Heading      float64   `json:"hdg"`  // Heading degrees
	Occupancy    string    `json:"occ"`  // Occupancy level
	LastUpdate   time.Time `json:"lu"`   // Last update timestamp
	Confidence   int       `json:"conf"` // Confidence score (0-100)
	IsVerified   bool      `json:"vrf"`  // Se foi verificado por múltiplos usuários
	Contributors int       `json:"cnt"`  // Número de usuários contribuindo
}

// DensityLog representa log de densidade de terminais
// Usado para relatórios de lucro e AdSense
type DensityLog struct {
	ID           int64     `json:"id"`
	TerminalID   string    `json:"terminal_id"`
	TerminalName string    `json:"terminal_name"`
	UserCount    int       `json:"user_count"` // Número de usuários ativos
	BusCount     int       `json:"bus_count"`  // Número de ônibus ativos
	AvgSpeed     float64   `json:"avg_speed"`  // Velocidade média
	Timestamp    time.Time `json:"timestamp"`
	Hour         int       `json:"hour"`        // Hora do dia (0-23)
	DayOfWeek    int       `json:"day_of_week"` // Dia da semana (0-6)
}

// TerminalDensity representa densidade atual de um terminal
// Retornada para analytics
type TerminalDensity struct {
	TerminalID   string  `json:"tid"`
	TerminalName string  `json:"tnm"`
	UserCount    int     `json:"ucnt"`
	BusCount     int     `json:"bcnt"`
	AvgSpeed     float64 `json:"aspd"`
	Trend        string  `json:"trnd"` // "increasing", "decreasing", "stable"
}

// GeoRadiusQuery representa consulta por raio geográfico
type GeoRadiusQuery struct {
	CenterLat    float64 `json:"center_lat"`
	CenterLng    float64 `json:"center_lng"`
	RadiusMeters int     `json:"radius_meters"`
	RouteFilter  string  `json:"route_filter"` // Opcional: filtrar por linha
}

// GeoRadiusResponse representa resposta de consulta geográfica
type GeoRadiusResponse struct {
	Count        int           `json:"cnt"`
	Buses        []BusLocation `json:"buses"`
	CenterLat    float64       `json:"clat"`
	CenterLng    float64       `json:"clng"`
	RadiusMeters int           `json:"radm"`
}
