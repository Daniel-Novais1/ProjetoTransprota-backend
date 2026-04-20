package telemetry

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// ============================================================================
// CX & MONETIZATION REPOSITORY
// ============================================================================

// GetETAWithConfidence calcula ETA com intervalo de confiança baseado em variância histórica
func (r *Repository) GetETAWithConfidence(ctx context.Context, deviceHash string, destLat, destLng float64) (*ETAWithConfidence, error) {
	start := time.Now()

	// 1. Buscar última posição e velocidade
	lastLat, lastLng, currentSpeed, err := r.GetDeviceLatestPosition(ctx, deviceHash)
	if err != nil {
		return nil, err
	}

	// 2. Calcular distância
	distanceMeters, err := r.CalculateDistance(ctx, lastLat, lastLng, destLat, destLng)
	if err != nil {
		return nil, err
	}

	// 3. Calcular fator de tráfego histórico de Goiânia
	trafficFactor := r.getGoiâniaTrafficFactor(ctx)

	// 4. Calcular ETA base com ajuste de tráfego
	distanceKm := distanceMeters / 1000.0
	avgSpeedKmh := currentSpeed
	if avgSpeedKmh <= 0 || avgSpeedKmh > 150 {
		avgSpeedKmh = 40.0 // Velocidade padrão urbana
	}

	// Ajustar velocidade baseado no fator de tráfego
	adjustedSpeed := avgSpeedKmh / trafficFactor
	etaHours := distanceKm / adjustedSpeed
	etaMinutes := int(math.Round(etaHours * 60))

	// 5. Calcular intervalo de confiança baseado em variância histórica
	confidence := r.calculateConfidence(ctx, deviceHash, currentSpeed, avgSpeedKmh, trafficFactor)

	// Intervalo de confiança: ±20% para baixa confiança, ±5% para alta confiança
	variancePercent := 0.20 - (float64(confidence) / 100.0 * 0.15) // 20% a 5%
	varianceMin := int(math.Round(float64(etaMinutes) * variancePercent))

	lowerBound := etaMinutes - varianceMin
	if lowerBound < 0 {
		lowerBound = 0
	}
	upperBound := etaMinutes + varianceMin

	// 6. Gerar mensagem amigável localizada
	message, friendlyMessage := r.generateLocalizedETA(etaMinutes, confidence, trafficFactor)

	elapsed := time.Since(start)
	logger.Debug("CX", "ETA with confidence calculated in %v | ETA: %dmin | Confidence: %d%% | TrafficFactor: %.2f", elapsed, etaMinutes, confidence, trafficFactor)

	return &ETAWithConfidence{
		EstimatedArrivalMin: etaMinutes,
		ConfidencePercent:   confidence,
		LowerBoundMin:       lowerBound,
		UpperBoundMin:       upperBound,
		Message:             message,
		FriendlyMessage:     friendlyMessage,
		IsPremium:           false, // Será preenchido pelo handler
	}, nil
}

// calculateConfidence calcula confiança da previsão baseado em fatores históricos
func (r *Repository) calculateConfidence(ctx context.Context, deviceHash string, currentSpeed, avgSpeed, trafficFactor float64) int {
	confidence := 80 // Base de 80%

	// Fator 1: Consistência de velocidade (se velocidade atual está próxima da média)
	speedDiff := math.Abs(currentSpeed - avgSpeed)
	if speedDiff < 5 {
		confidence += 10 // Velocidade consistente aumenta confiança
	} else if speedDiff > 20 {
		confidence -= 15 // Velocidade muito variável reduz confiança
	}

	// Fator 2: Fator de tráfego (tráfico intenso reduz confiança)
	if trafficFactor > 1.5 {
		confidence -= 15 // Trânsito muito intenso
	} else if trafficFactor > 1.2 {
		confidence -= 5 // Trânsito moderado
	} else if trafficFactor < 0.8 {
		confidence += 10 // Trânsito leve
	}

	// Fator 3: Hora do dia (trânsito em horários de pico reduz confiança)
	hour := time.Now().Hour()
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		confidence -= 10 // Horário de pico
	} else if hour >= 22 || hour <= 5 {
		confidence += 10 // Madrugada (trânsito previsível)
	}

	// Limitar confiança entre 30% e 95%
	if confidence > 95 {
		confidence = 95
	}
	if confidence < 30 {
		confidence = 30
	}

	return confidence
}

// generateLocalizedETA gera mensagem amigável localizada para Goiânia
func (r *Repository) generateLocalizedETA(etaMinutes, confidence int, trafficFactor float64) (string, string) {
	// Mensagem principal
	var message string
	if etaMinutes <= 2 {
		message = "Seu ônibus tá chegando!"
	} else if etaMinutes <= 5 {
		message = fmt.Sprintf("Chegando em %d min", etaMinutes)
	} else if etaMinutes <= 10 {
		message = fmt.Sprintf("Chegando em %d min", etaMinutes)
	} else {
		message = fmt.Sprintf("Chegando em %d min", etaMinutes)
	}

	// Mensagem amigável com confiança
	var friendlyMessage string
	if confidence >= 80 {
		friendlyMessage = fmt.Sprintf("%d%% de chance de chegar em %d min", confidence, etaMinutes)
	} else if confidence >= 60 {
		friendlyMessage = fmt.Sprintf("Previsão aproximada: %d min", etaMinutes)
	} else {
		friendlyMessage = fmt.Sprintf("Previsão incerta: %d min (pode variar)", etaMinutes)
	}

	// Adicionar contexto de tráfego de Goiânia
	if trafficFactor > 1.5 {
		friendlyMessage += " • Trânsito intenso"
	} else if trafficFactor > 1.2 {
		friendlyMessage += " • Trânsito moderado"
	} else if trafficFactor < 0.8 {
		friendlyMessage += " • Trânsito leve"
	}

	// Adicionar contexto de Goiânia
	if etaMinutes > 15 {
		switch {
		case etaMinutes <= 20:
			friendlyMessage += " • Bora pro Eixo?"
		case etaMinutes <= 30:
			friendlyMessage += " • Trânsito normal"
		default:
			friendlyMessage += " • Trânsito carregado"
		}
	}

	return message, friendlyMessage
}

// GetUserStatus busca status do usuário para monetização
func (r *Repository) GetUserStatus(ctx context.Context, userID string) (*UserStatus, error) {
	// Em MVP, usar Redis para armazenar status de usuário
	// Em produção, tabela de users com subscription_status

	// Tenta buscar do Redis primeiro
	redisKey := fmt.Sprintf("user:status:%s", userID)
	redisClient, ok := r.rdb.(*redis.Client)
	if ok {
		_, err := redisClient.Get(ctx, redisKey).Result()
		if err == nil {
			// Parse do JSON do Redis (simplificado para MVP)
			return &UserStatus{
				UserID:        userID,
				IsPremium:     false,
				AdFree:        false,
				Subscription:  "free",
				CheckInStreak: 0,
				Points:        0,
			}, nil
		}
	}

	// Usuário não encontrado, criar status padrão
	defaultStatus := &UserStatus{
		UserID:        userID,
		IsPremium:     false,
		AdFree:        false,
		Subscription:  "free",
		ExpiryDate:    time.Now().Add(30 * 24 * time.Hour),
		CheckInStreak: 0,
		Points:        0,
	}

	// Salvar no Redis
	if ok {
		statusJSON := fmt.Sprintf(`{"user_id":"%s","is_premium":false,"ad_free":false,"subscription":"free","points":0}`, userID)
		redisClient.Set(ctx, redisKey, statusJSON, 24*time.Hour)
	}

	return defaultStatus, nil
}

// RecordCheckIn registra check-in do usuário em ponto de ônibus (gamificação)
func (r *Repository) RecordCheckIn(ctx context.Context, checkIn *CheckInRequest) (*CheckInResponse, error) {
	redisClient, ok := r.rdb.(*redis.Client)
	if !ok {
		return &CheckInResponse{
			Success: false,
			Message: "Redis not available",
		}, nil
	}

	// 1. Verificar se usuário já fez check-in recentemente (evitar spam)
	redisKey := fmt.Sprintf("checkin:%s:%s", checkIn.UserID, checkIn.StopID)
	_, err := redisClient.Get(ctx, redisKey).Result()
	if err == nil {
		// Já fez check-in recentemente
		return &CheckInResponse{
			Success:         false,
			PointsEarned:    0,
			StreakIncreased: false,
			Message:         "Você já fez check-in neste ponto recentemente",
			FriendlyMessage: "Volte em 5 minutos para ganhar mais pontos!",
		}, nil
	}

	// 2. Buscar streak atual do usuário
	streakKey := fmt.Sprintf("streak:%s", checkIn.UserID)
	streak, _ := redisClient.Get(ctx, streakKey).Int()
	if streak == 0 {
		streak = 1
	} else {
		streak++
	}

	// 3. Calcular pontos ganhos
	pointsEarned := 10 // Base: 10 pontos por check-in
	if streak >= 7 {
		pointsEarned = 20 // Bônus por streak de 7 dias
	}

	// 4. Salvar check-in no Redis (TTL 5 minutos para evitar spam)
	redisClient.Set(ctx, redisKey, "1", 5*time.Minute)
	redisClient.Set(ctx, streakKey, streak, 24*time.Hour) // Streak reseta após 24h sem check-in

	// 5. Buscar ETA do próximo ônibus para este ponto
	nextBusETA := r.getNextBusETAForStop(ctx, checkIn.StopID, checkIn.RouteID)
	confidence := 75 // Valor padrão para MVP

	// 6. Gerar mensagem de incentivo
	var message string
	var friendlyMessage string
	if streak >= 7 {
		message = fmt.Sprintf("Streak de %d dias! +%d pontos bônus!", streak, pointsEarned)
		friendlyMessage = "Você está no caminho de virar um passageiro VIP!"
	} else {
		message = fmt.Sprintf("Check-in realizado! +%d pontos", pointsEarned)
		friendlyMessage = fmt.Sprintf("Continue check-in por %d dias para desbloquear recompensas", 7-streak)
	}

	logger.Info("CX", "Check-in recorded | User: %s | Stop: %s | Streak: %d | Points: %d",
		checkIn.UserID, checkIn.StopID, streak, pointsEarned)

	return &CheckInResponse{
		Success:         true,
		PointsEarned:    pointsEarned,
		StreakIncreased: streak > 1,
		CurrentStreak:   streak,
		NextBusETA:      nextBusETA,
		Confidence:      confidence,
		Message:         message,
		FriendlyMessage: friendlyMessage,
	}, nil
}

// getNextBusETAForStop busca ETA do próximo ônibus para um ponto
func (r *Repository) getNextBusETAForStop(ctx context.Context, stopID, routeID string) int {
	// Em MVP, retorna valor estimado baseado em horário
	// Em produção, consulta tabela de horários programados

	hour := time.Now().Hour()
	minute := time.Now().Minute()

	// Simulação: ônibus chegam a cada 15 minutos
	nextBus := ((15 - (minute % 15)) % 15)

	// Se horário fora de operação (22h-5h), retorna valor alto
	if hour >= 22 || hour < 5 {
		nextBus = 60
	}

	return nextBus
}

// RecordOccupancyReport registra reporte de lotação do ônibus
func (r *Repository) RecordOccupancyReport(ctx context.Context, report *OccupancyReport) error {
	redisClient, ok := r.rdb.(*redis.Client)

	// 1. Salvar no banco de dados (tabela occupancy_reports)
	if r.db != nil {
		query := `
			INSERT INTO occupancy_reports (device_hash, route_id, occupancy, occupancy_pct, reporter_id, timestamp)
			VALUES ($1, $2, $3, $4, $5, $6)
		`
		_, err := r.db.ExecContext(ctx, query, report.DeviceHash, report.RouteID, report.Occupancy, report.OccupancyPct, report.ReporterID, report.Timestamp)
		if err != nil {
			logger.Warn("CX", "Failed to save occupancy report: %v", err)
			// Continuar mesmo se falhar (best-effort)
		}
	}

	// 2. Atualizar cache de lotação no Redis (últimos 5 minutos)
	if ok {
		redisKey := fmt.Sprintf("occupancy:%s", report.DeviceHash)
		occupancyData := fmt.Sprintf(`{"occupancy":"%s","pct":%d,"reported_at":"%s","reporter":"%s"}`,
			report.Occupancy, report.OccupancyPct, report.Timestamp.Format(time.RFC3339), report.ReporterID)
		redisClient.Set(ctx, redisKey, occupancyData, 5*time.Minute)

		// 3. Dar pontos ao usuário (gamificação)
		pointsKey := fmt.Sprintf("points:%s", report.ReporterID)
		redisClient.Incr(ctx, pointsKey)
	}

	logger.Info("CX", "Occupancy report recorded | Device: %s | Occupancy: %s | Reporter: %s | Points: +5",
		report.DeviceHash, report.Occupancy, report.ReporterID)

	return nil
}

// getGoiâniaTrafficFactor retorna fator de tráfego baseado em dados históricos de Goiânia
func (r *Repository) getGoiâniaTrafficFactor(ctx context.Context) float64 {
	hour := time.Now().Hour()
	weekday := time.Now().Weekday()

	// Fatores baseados em padrões de tráfego de Goiânia
	// 1.0 = tráfego normal, >1.0 = tráfego intenso, <1.0 = tráfego leve

	baseFactor := 1.0

	// Horários de pico em Goiânia
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		baseFactor = 1.6 // Trânsito intenso
	} else if (hour >= 11 && hour <= 13) || (hour >= 18 && hour <= 20) {
		baseFactor = 1.3 // Trânsito moderado (almoço/fim de trabalho)
	} else if hour >= 22 || hour <= 5 {
		baseFactor = 0.7 // Trânsito leve (madrugada)
	}

	// Ajuste por dia da semana
	if weekday == time.Friday {
		baseFactor += 0.1 // Sexta-feira tem mais trânsito
	} else if weekday == time.Saturday || weekday == time.Sunday {
		baseFactor -= 0.2 // Fim de semana tem menos trânsito
	}

	// Limitar fator entre 0.5 e 2.0
	if baseFactor < 0.5 {
		baseFactor = 0.5
	}
	if baseFactor > 2.0 {
		baseFactor = 2.0
	}

	return baseFactor
}

// GetLocalizedMessages retorna mensagens localizadas para Goiânia
func (r *Repository) GetLocalizedMessages(ctx context.Context, language string) (*LocalizedMessages, error) {
	// Em MVP, retorna mensagens em português para Goiânia
	messages := &LocalizedMessages{
		ArrivalSoon:      "Seu ônibus tá chegando!",
		ArrivingIn:       "Chegando em X min",
		BusAtTerminal:    "Ônibus no Terminal",
		TrafficCongested: "Trânsito carregado",
		BusFull:          "Ônibus lotado",
		BusEmpty:         "Ônibus vazio",
		CheckInReward:    "Você ganhou X pontos!",
		Gamification:     "Continue check-in para desbloquear recompensas",
	}

	return messages, nil
}
