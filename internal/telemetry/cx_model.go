package telemetry

import "time"

// ============================================================================
// CX & MONETIZATION MODELS
// ============================================================================

// ETAWithConfidence representa ETA com intervalo de confiança para o usuário
type ETAWithConfidence struct {
	EstimatedArrivalMin int     `json:"estimated_arrival_min"` // Tempo estimado em minutos
	ConfidencePercent   int     `json:"confidence_percent"`     // 0-100: confiança da previsão
	LowerBoundMin       int     `json:"lower_bound_min"`        // Pessimista: mínimo de minutos
	UpperBoundMin       int     `json:"upper_bound_min"`        // Otimista: máximo de minutos
	Message             string  `json:"message"`                // Mensagem amigável localizada
	FriendlyMessage    string  `json:"friendly_message"`       // "Bora pro Eixo?"
	IsPremium           bool    `json:"is_premium"`             // Status Premium do usuário
}

// UserStatus representa status do usuário para monetização
type UserStatus struct {
	UserID         string    `json:"user_id"`
	IsPremium      bool      `json:"is_premium"`
	AdFree         bool      `json:"ad_free"`           // Toggle ad-free
	Subscription  string    `json:"subscription"`      // "free", "premium"
	ExpiryDate     time.Time `json:"expiry_date"`
	CheckInStreak  int       `json:"check_in_streak"`  // Gamificação: sequência de check-ins
	LastCheckIn    time.Time `json:"last_check_in"`
	Points         int       `json:"points"`            // Pontos de gamificação
}

// CheckInRequest representa check-in do usuário em um ponto de ônibus
type CheckInRequest struct {
	UserID      string  `json:"user_id"`
	StopID      string  `json:"stop_id"`       // ID do ponto de ônibus
	RouteID     string  `json:"route_id"`      // ID da linha
	DeviceHash  string  `json:"device_hash"`   // Para evitar múltiplos check-ins
	IsOnBus     bool    `json:"is_on_bus"`     // Se está no ônibus
}

// CheckInResponse representa resposta do check-in
type CheckInResponse struct {
	Success          bool      `json:"success"`
	PointsEarned     int       `json:"points_earned"`
	StreakIncreased  bool      `json:"streak_increased"`
	CurrentStreak    int       `json:"current_streak"`
	NextBusETA       int       `json:"next_bus_eta_min"` // Tempo até o próximo ônibus
	Confidence       int       `json:"confidence"`        // Confiança da previsão
	Message          string    `json:"message"`           // Mensagem de incentivo
	FriendlyMessage  string    `json:"friendly_message"`
}

// OccupancyReport representa reporte de lotação do ônibus
type OccupancyReport struct {
	DeviceHash   string  `json:"device_hash"`
	RouteID      string  `json:"route_id"`
	Occupancy    string  `json:"occupancy"`    // "empty", "low", "medium", "high", "full"
	OccupancyPct int     `json:"occupancy_pct"` // 0-100
	Timestamp    time.Time `json:"timestamp"`
	ReporterID   string  `json:"reporter_id"`   // User ID do reporte
	PointsEarned int     `json:"points_earned"` // Pontos por reportar
}

// LocalizedMessages contém mensagens localizadas para Goiânia
type LocalizedMessages struct {
	ArrivalSoon      string `json:"arrival_soon"`       // "Seu ônibus tá chegando!"
	ArrivingIn       string `json:"arriving_in"`        // "Chegando em X min"
	BusAtTerminal    string `json:"bus_at_terminal"`    // "Ônibus no Terminal"
	TrafficCongested string `json:"traffic_congested"`  // "Trânsito carregado"
	BusFull          string `json:"bus_full"`           // "Ônibus lotado"
	BusEmpty         string `json:"bus_empty"`          // "Ônibus vazio"
	CheckInReward    string `json:"check_in_reward"`     // "Você ganhou X pontos!"
	Gamification     string `json:"gamification"`       // "Continue check-in para desbloquear recompensas"
}

// MobileOptimizedResponse é uma resposta otimizada para mobile com payloads mínimos
type MobileOptimizedResponse struct {
	Data interface{} `json:"d"` // Minificado: "d" ao invés de "data"
	Meta struct {
		LatencyMs int64  `json:"l"` // Latência em ms
		Cached    bool   `json:"c"` // Se veio do cache
		Version   string `json:"v"` // Versão da API
	} `json:"m"`
}

// AdSlot representa configuração de slot de anúncio
type AdSlot struct {
	Position string `json:"position"` // "top", "bottom", "between"
	Enabled  bool   `json:"enabled"`  // Se está habilitado (false para Premium)
	Size     string `json:"size"`     // "banner", "native"
}
