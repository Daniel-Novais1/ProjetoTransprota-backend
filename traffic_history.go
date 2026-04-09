package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TrafficRecord representa um registro de trânsito histórico
type TrafficRecord struct {
	ID          int       `json:"id"`
	RouteKey    string    `json:"route_key"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	DayOfWeek   int       `json:"day_of_week"`
	Hour        int       `json:"hour"`
	TimeMinutes int       `json:"time_minutes"`
	Weather     string    `json:"weather"`
	RecordedAt  time.Time `json:"recorded_at"`
	Confidence  float64   `json:"confidence"`
}

// TrafficPattern representa um padrão de trânsito identificado
type TrafficPattern struct {
	RouteKey       string    `json:"route_key"`
	Origin         string    `json:"origin"`
	Destination    string    `json:"destination"`
	DayOfWeek      int       `json:"day_of_week"`
	Hour           int       `json:"hour"`
	AverageTime    float64   `json:"average_time"`
	MinTime        int       `json:"min_time"`
	MaxTime        int       `json:"max_time"`
	StandardDev    float64   `json:"standard_deviation"`
	SampleCount    int       `json:"sample_count"`
	LastUpdated    time.Time `json:"last_updated"`
	PredictedDelay float64   `json:"predicted_delay"`
	Confidence     float64   `json:"confidence"`
}

// TrafficAnalysis representa uma análise completa de trânsito
type TrafficAnalysis struct {
	RouteKey       string           `json:"route_key"`
	CurrentPattern *TrafficPattern  `json:"current_pattern"`
	HistoricalData []TrafficRecord  `json:"historical_data"`
	Predictions    []TrafficPattern `json:"predictions"`
	Insights       []string         `json:"insights"`
}

// TrafficHistoryManager gerencia o histórico de trânsito
type TrafficHistoryManager struct {
	app      *App
	patterns map[string]*TrafficPattern
	mutex    sync.RWMutex
}

// NewTrafficHistoryManager cria um novo gerenciador de histórico
func NewTrafficHistoryManager(app *App) *TrafficHistoryManager {
	manager := &TrafficHistoryManager{
		app:      app,
		patterns: make(map[string]*TrafficPattern),
	}

	// Iniciar análise em background
	go manager.startTrafficAnalysis()

	return manager
}

// startTrafficAnalysis inicia análise contínua de trânsito
func (thm *TrafficHistoryManager) startTrafficAnalysis() {
	ticker := time.NewTicker(30 * time.Minute) // Analisar a cada 30 min
	defer ticker.Stop()

	for range ticker.C {
		thm.analyzeTrafficPatterns()
		thm.cleanupOldRecords()
	}
}

// RecordRouteTime registra o tempo de uma rota para análise histórica
func (thm *TrafficHistoryManager) RecordRouteTime(origin, destination string, timeMinutes int, weather string) {
	if thm.app.db == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		now := time.Now()
		routeKey := fmt.Sprintf("%s:%s", normalizeParam(origin), normalizeParam(destination))

		query := `
		INSERT INTO traffic_history (route_key, origin, destination, day_of_week, hour, time_minutes, weather, recorded_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`

		_, err := thm.app.db.ExecContext(ctx, query,
			routeKey, origin, destination,
			int(now.Weekday()), now.Hour(), timeMinutes, weather, now,
		)

		if err != nil {
			log.Printf("Erro ao registrar tempo de rota: %v", err)
		}
	}()
}

// analyzeTrafficPatterns analisa padrões de trânsito
func (thm *TrafficHistoryManager) analyzeTrafficPatterns() {
	if thm.app.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
	SELECT 
		route_key,
		origin,
		destination,
		day_of_week,
		hour,
		AVG(time_minutes) as avg_time,
		MIN(time_minutes) as min_time,
		MAX(time_minutes) as max_time,
		STDDEV(time_minutes) as std_dev,
		COUNT(*) as sample_count,
		MAX(recorded_at) as last_updated
	FROM traffic_history 
	WHERE recorded_at >= CURRENT_TIMESTAMP - INTERVAL '90 days'
	GROUP BY route_key, origin, destination, day_of_week, hour
	HAVING COUNT(*) >= 5 -- Mínimo de 5 amostras
	ORDER BY route_key, day_of_week, hour
	`

	rows, err := thm.app.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao analisar padrões de trânsito: %v", err)
		return
	}
	defer rows.Close()

	thm.mutex.Lock()
	defer thm.mutex.Unlock()

	newPatterns := make(map[string]*TrafficPattern)

	for rows.Next() {
		var pattern TrafficPattern
		err := rows.Scan(
			&pattern.RouteKey,
			&pattern.Origin,
			&pattern.Destination,
			&pattern.DayOfWeek,
			&pattern.Hour,
			&pattern.AverageTime,
			&pattern.MinTime,
			&pattern.MaxTime,
			&pattern.StandardDev,
			&pattern.SampleCount,
			&pattern.LastUpdated,
		)
		if err != nil {
			continue
		}

		// Calcular confiança baseada no tamanho da amostra e desvio padrão
		sampleScore := math.Min(1.0, float64(pattern.SampleCount)/30.0)
		consistencyScore := math.Max(0.0, 1.0-(pattern.StandardDev/pattern.AverageTime))
		pattern.Confidence = (sampleScore + consistencyScore) / 2.0

		// Calcular previsão de atraso baseada em tendências
		pattern.PredictedDelay = thm.calculatePredictedDelay(pattern)

		patternKey := fmt.Sprintf("%s:%d:%d", pattern.RouteKey, pattern.DayOfWeek, pattern.Hour)
		newPatterns[patternKey] = &pattern
	}

	thm.patterns = newPatterns
	log.Printf("Analisados %d padrões de trânsito", len(newPatterns))
}

// calculatePredictedDelay calcula atraso previsto baseado em tendências
func (thm *TrafficHistoryManager) calculatePredictedDelay(pattern TrafficPattern) float64 {
	// Lógica simples: se desvio padrão > 20% da média, prever atraso
	if pattern.StandardDev > pattern.AverageTime*0.2 {
		return pattern.StandardDev * 0.5 // Prever 50% do desvio como atraso
	}
	return 0
}

// cleanupOldRecords remove registros muito antigos
func (thm *TrafficHistoryManager) cleanupOldRecords() {
	if thm.app.db == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		query := `DELETE FROM traffic_history WHERE recorded_at < CURRENT_TIMESTAMP - INTERVAL '180 days'`
		_, err := thm.app.db.ExecContext(ctx, query)
		if err != nil {
			log.Printf("Erro ao limpar registros antigos: %v", err)
		}
	}()
}

// GetTrafficAnalysis obtém análise completa de trânsito para uma rota
func (thm *TrafficHistoryManager) GetTrafficAnalysis(origin, destination string) (*TrafficAnalysis, error) {
	routeKey := fmt.Sprintf("%s:%s", normalizeParam(origin), normalizeParam(destination))

	analysis := &TrafficAnalysis{
		RouteKey: routeKey,
		Insights: []string{},
	}

	// Obter padrão atual
	now := time.Now()
	currentPatternKey := fmt.Sprintf("%s:%d:%d", routeKey, int(now.Weekday()), now.Hour())

	thm.mutex.RLock()
	if pattern, exists := thm.patterns[currentPatternKey]; exists {
		analysis.CurrentPattern = pattern
		analysis.Insights = append(analysis.Insights,
			fmt.Sprintf("Tempo médio histórico: %.1f minutos", pattern.AverageTime),
			fmt.Sprintf("Confiabilidade: %.1f%%", pattern.Confidence*100),
		)

		if pattern.PredictedDelay > 0 {
			analysis.Insights = append(analysis.Insights,
				fmt.Sprintf("Previsão de atraso: %.1f minutos", pattern.PredictedDelay),
			)
		}
	}
	thm.mutex.RUnlock()

	// Obter dados históricos recentes
	if thm.app.db != nil {
		historical, err := thm.getHistoricalData(routeKey, 50)
		if err == nil {
			analysis.HistoricalData = historical
		}
	}

	// Gerar insights adicionais
	analysis.Insights = append(analysis.Insights, thm.generateInsights(analysis)...)

	return analysis, nil
}

// getHistoricalData obtém dados históricos do banco
func (thm *TrafficHistoryManager) getHistoricalData(routeKey string, limit int) ([]TrafficRecord, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
	SELECT id, route_key, origin, destination, day_of_week, hour, time_minutes, weather, recorded_at
	FROM traffic_history 
	WHERE route_key = $1
	ORDER BY recorded_at DESC
	LIMIT $2
	`

	rows, err := thm.app.db.QueryContext(ctx, query, routeKey, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []TrafficRecord
	for rows.Next() {
		var record TrafficRecord
		err := rows.Scan(
			&record.ID,
			&record.RouteKey,
			&record.Origin,
			&record.Destination,
			&record.DayOfWeek,
			&record.Hour,
			&record.TimeMinutes,
			&record.Weather,
			&record.RecordedAt,
		)
		if err != nil {
			continue
		}
		records = append(records, record)
	}

	return records, nil
}

// generateInsights gera insights baseados nos dados
func (thm *TrafficHistoryManager) generateInsights(analysis *TrafficAnalysis) []string {
	insights := []string{}

	// Análise de padrões semanais
	if len(analysis.HistoricalData) > 10 {
		weekdayTimes := make(map[int][]int)
		for _, record := range analysis.HistoricalData {
			weekdayTimes[record.DayOfWeek] = append(weekdayTimes[record.DayOfWeek], record.TimeMinutes)
		}

		// Encontrar dia mais lento
		var slowestDay int
		var slowestAvg float64
		for day, times := range weekdayTimes {
			if len(times) > 0 {
				avg := average(times)
				if avg > slowestAvg {
					slowestAvg = avg
					slowestDay = day
				}
			}
		}

		if slowestDay >= 0 && slowestDay <= 6 {
			dayNames := []string{"Domingo", "Segunda", "Terça", "Quarta", "Quinta", "Sexta", "Sábado"}
			insights = append(insights, fmt.Sprintf("Dia mais lento: %s (%.1f min)", dayNames[slowestDay], slowestAvg))
		}
	}

	// Análise de impacto do clima
	if len(analysis.HistoricalData) > 5 {
		rainyTimes := []int{}
		sunnyTimes := []int{}

		for _, record := range analysis.HistoricalData {
			if record.Weather == "chuva" || record.Weather == "tempestade" {
				rainyTimes = append(rainyTimes, record.TimeMinutes)
			} else if record.Weather == "céu limpo" || record.Weather == "parcialmente nublado" {
				sunnyTimes = append(sunnyTimes, record.TimeMinutes)
			}
		}

		if len(rainyTimes) > 0 && len(sunnyTimes) > 0 {
			rainyAvg := average(rainyTimes)
			sunnyAvg := average(sunnyTimes)
			if rainyAvg > sunnyAvg*1.1 {
				insights = append(insights, fmt.Sprintf("Chuva aumenta tempo em %.1f%%", ((rainyAvg/sunnyAvg)-1)*100))
			}
		}
	}

	return insights
}

// average calcula média de um slice de inteiros
func average(numbers []int) float64 {
	if len(numbers) == 0 {
		return 0
	}
	sum := 0
	for _, n := range numbers {
		sum += n
	}
	return float64(sum) / float64(len(numbers))
}

// GetPredictedTime obtém tempo previsto para uma rota
func (thm *TrafficHistoryManager) GetPredictedTime(origin, destination string) (int, float64) {
	routeKey := fmt.Sprintf("%s:%s", normalizeParam(origin), normalizeParam(destination))
	now := time.Now()
	patternKey := fmt.Sprintf("%s:%d:%d", routeKey, int(now.Weekday()), now.Hour())

	thm.mutex.RLock()
	pattern, exists := thm.patterns[patternKey]
	thm.mutex.RUnlock()

	if !exists {
		// Fallback: tempo base estimado
		distance := calcularDistanciaEntreLocais(origin, destination)
		return int(distance * 12), 0.5 // 12 min por km, confiança média
	}

	predictedTime := int(pattern.AverageTime + pattern.PredictedDelay)
	return predictedTime, pattern.Confidence
}

// GetTrafficTrends obtém tendências de trânsito para múltiplas rotas
func (thm *TrafficHistoryManager) GetTrafficTrends() map[string][]TrafficPattern {
	thm.mutex.RLock()
	defer thm.mutex.RUnlock()

	// Agrupar padrões por rota
	trends := make(map[string][]TrafficPattern)
	for _, pattern := range thm.patterns {
		trends[pattern.RouteKey] = append(trends[pattern.RouteKey], *pattern)
	}

	// Ordenar por horário
	for route := range trends {
		sort.Slice(trends[route], func(i, j int) bool {
			if trends[route][i].DayOfWeek != trends[route][j].DayOfWeek {
				return trends[route][i].DayOfWeek < trends[route][j].DayOfWeek
			}
			return trends[route][i].Hour < trends[route][j].Hour
		})
	}

	return trends
}

// setupTrafficHistoryRoutes configura as rotas do módulo de histórico
func setupTrafficHistoryRoutes(r *gin.Engine, app *App) {
	manager := NewTrafficHistoryManager(app)

	// GET /api/v1/traffic/analysis - Análise de trânsito para uma rota
	r.GET("/api/v1/traffic/analysis", func(c *gin.Context) {
		origin := c.Query("origin")
		destination := c.Query("destination")

		if origin == "" || destination == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "origin e destination são obrigatórios"})
			return
		}

		analysis, err := manager.GetTrafficAnalysis(origin, destination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao analisar trânsito"})
			return
		}

		c.JSON(http.StatusOK, analysis)
	})

	// GET /api/v1/traffic/predict - Prever tempo de rota
	r.GET("/api/v1/traffic/predict", func(c *gin.Context) {
		origin := c.Query("origin")
		destination := c.Query("destination")

		if origin == "" || destination == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "origin e destination são obrigatórios"})
			return
		}

		predictedTime, confidence := manager.GetPredictedTime(origin, destination)

		c.JSON(http.StatusOK, gin.H{
			"origin":         origin,
			"destination":    destination,
			"predicted_time": predictedTime,
			"confidence":     confidence,
			"analysis": fmt.Sprintf("Tempo previsto: %d minutos (confiança: %.1f%%)",
				predictedTime, confidence*100),
		})
	})

	// GET /api/v1/traffic/trends - Tendências de trânsito
	r.GET("/api/v1/traffic/trends", func(c *gin.Context) {
		trends := manager.GetTrafficTrends()

		// Limitar a 10 rotas mais populares
		limited := make(map[string][]TrafficPattern)
		count := 0
		for route, patterns := range trends {
			if count >= 10 {
				break
			}
			limited[route] = patterns
			count++
		}

		c.JSON(http.StatusOK, gin.H{
			"trends":          limited,
			"total_routes":    len(trends),
			"analyzed_routes": len(limited),
		})
	})

	// POST /api/v1/traffic/record - Registrar tempo de rota (interno)
	r.POST("/api/v1/traffic/record", func(c *gin.Context) {
		var record struct {
			Origin      string `json:"origin"`
			Destination string `json:"destination"`
			TimeMinutes int    `json:"time_minutes"`
			Weather     string `json:"weather"`
		}

		if err := c.ShouldBindJSON(&record); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
			return
		}

		manager.RecordRouteTime(record.Origin, record.Destination, record.TimeMinutes, record.Weather)

		c.JSON(http.StatusOK, gin.H{"message": "Tempo de rota registrado com sucesso"})
	})
}
