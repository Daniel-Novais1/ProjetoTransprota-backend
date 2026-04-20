package telemetry

import (
	"context"
	"fmt"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/redis/go-redis/v9"
)

// BIAnalyticsService analisa métricas da frota e detecta anomalias
type BIAnalyticsService struct {
	redis          *redis.Client
	stop           chan struct{}
	baselineSpeed  float64 // Velocidade média de referência (km/h)
	analysisWindow time.Duration
	alertThreshold float64 // Porcentagem de redução para alerta (20%)
}

// NewBIAnalyticsService cria um novo serviço de BI analytics
func NewBIAnalyticsService(rdb *redis.Client) *BIAnalyticsService {
	return &BIAnalyticsService{
		redis:          rdb,
		stop:           make(chan struct{}),
		baselineSpeed:  40.0, // Velocidade média normal em área urbana
		analysisWindow: 10 * time.Minute,
		alertThreshold: 0.20, // 20% de redução
	}
}

// Start inicia o worker que roda a cada 10 minutos
func (b *BIAnalyticsService) Start() {
	logger.Info("BI", "Iniciando worker de BI Analytics (janela: %v)", b.analysisWindow)

	ticker := time.NewTicker(b.analysisWindow)

	go func() {
		// Executar análise imediatamente na inicialização
		b.analyzeFleetPerformance(context.Background())

		for {
			select {
			case <-ticker.C:
				b.analyzeFleetPerformance(context.Background())
			case <-b.stop:
				ticker.Stop()
				logger.Info("BI", "Worker de BI Analytics parado")
				return
			}
		}
	}()
}

// Stop para o worker
func (b *BIAnalyticsService) Stop() {
	close(b.stop)
}

// analyzeFleetPerformance analisa o desempenho da frota
func (b *BIAnalyticsService) analyzeFleetPerformance(ctx context.Context) {
	start := time.Now()
	logger.Info("BI", "Iniciando análise de desempenho da frota...")

	// 1. Obter velocidades atuais dos ônibus
	avgSpeed, busCount, err := b.getAverageFleetSpeed(ctx)
	if err != nil {
		logger.Error("BI", "Erro ao obter velocidade média: %v", err)
		return
	}

	if busCount == 0 {
		logger.Warn("BI", "Nenhum ônibus ativo para análise")
		return
	}

	// 2. Comparar com baseline
	speedReduction := (b.baselineSpeed - avgSpeed) / b.baselineSpeed

	// 3. Detectar engarrafamento
	if speedReduction >= b.alertThreshold {
		// ALERTA: Engarrafamento detectado
		logger.Warn("BI", "⚠️ ENGARRAFAMENTO DETECTADO!")
		logger.Warn("BI", "Velocidade média: %.1f km/h (baseline: %.1f km/h)", avgSpeed, b.baselineSpeed)
		logger.Warn("BI", "Redução: %.1f%%", speedReduction*100)
		logger.Warn("BI", "Ônibus monitorados: %d", busCount)

		// Publicar alerta no Redis
		b.publishCongestionAlert(ctx, avgSpeed, busCount, speedReduction)
	} else {
		logger.Info("BI", "Trânsito normal. Velocidade média: %.1f km/h (%d ônibus)", avgSpeed, busCount)
	}

	elapsed := time.Since(start)
	logger.Info("BI", "Análise concluída em %v", elapsed)
}

// getAverageFleetSpeed obtém a velocidade média da frota
func (b *BIAnalyticsService) getAverageFleetSpeed(ctx context.Context) (float64, int, error) {
	// Buscar todas as chaves de bus:*
	keys, err := b.redis.Keys(ctx, "bus:*:last_position").Result()
	if err != nil {
		return 0, 0, err
	}

	if len(keys) == 0 {
		return 0, 0, nil
	}

	var totalSpeed float64
	var busCount int

	// Iterar sobre as chaves e obter velocidade
	for _, key := range keys {
		data, err := b.redis.HGetAll(ctx, key).Result()
		if err != nil {
			continue
		}

		speedStr, ok := data["speed"]
		if !ok {
			continue
		}

		var speed float64
		if _, err := fmt.Sscanf(speedStr, "%f", &speed); err != nil {
			continue
		}

		// Ignorar velocidades zero ou inválidas
		if speed > 0 && speed < 120 {
			totalSpeed += speed
			busCount++
		}
	}

	if busCount == 0 {
		return 0, 0, nil
	}

	return totalSpeed / float64(busCount), busCount, nil
}

// publishCongestionAlert publica alerta de engarrafamento no Redis
func (b *BIAnalyticsService) publishCongestionAlert(ctx context.Context, avgSpeed float64, busCount int, reduction float64) {
	alertData := map[string]interface{}{
		"alert":           "CONGESTION_DETECTED",
		"avg_speed":       avgSpeed,
		"baseline":        b.baselineSpeed,
		"reduction":       reduction * 100,
		"bus_count":       busCount,
		"timestamp":       time.Now().Format(time.RFC3339),
		"analysis_window": b.analysisWindow.String(),
	}

	if err := b.redis.Publish(ctx, "bi_alerts", alertData).Err(); err != nil {
		logger.Error("BI", "Erro ao publicar alerta de engarrafamento: %v", err)
	} else {
		logger.Info("BI", "Alerta de engarrafamento publicado no canal 'bi_alerts'")
	}
}

// UpdateBaseline atualiza a velocidade de referência
func (b *BIAnalyticsService) UpdateBaseline(newBaseline float64) {
	b.baselineSpeed = newBaseline
	logger.Info("BI", "Baseline atualizado para %.1f km/h", newBaseline)
}
