package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// UserPattern representa um padrão de comportamento do usuário
type UserPattern struct {
	UserID      string    `json:"user_id"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	DayOfWeek   int       `json:"day_of_week"` // 0=Sunday, 1=Monday, ...
	Hour        int       `json:"hour"`        // 0-23
	Minute      int       `json:"minute"`      // 0-59
	Frequency   int       `json:"frequency"`   // Quantas vezes ocorreu
	LastSeen    time.Time `json:"last_seen"`
	Confidence  float64   `json:"confidence"` // 0.0-1.0
}

// RouteRecommendation representa uma recomendação de rota
type RouteRecommendation struct {
	UserID       string             `json:"user_id"`
	Route        *MapRouteResponse  `json:"route"`
	Reason       string             `json:"reason"`
	Confidence   float64            `json:"confidence"`
	IsPredictive bool               `json:"is_predictive"`
	Weather      WeatherData        `json:"weather"`
	Alternatives []RouteAlternative `json:"alternatives"`
}

// RouteAlternative representa uma rota alternativa
type RouteAlternative struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	TimeMinutes int     `json:"time_minutes"`
	Cost        float64 `json:"cost"`
	Confidence  float64 `json:"confidence"`
}

// RecommendationEngine é o motor de recomendação personalizada
type RecommendationEngine struct {
	app           *App
	patterns      map[string][]UserPattern // userID -> patterns
	patternsMutex sync.RWMutex
	cache         map[string]*RouteRecommendation
	cacheMutex    sync.RWMutex
}

// NewRecommendationEngine cria uma nova instância do motor de recomendação
func NewRecommendationEngine(app *App) *RecommendationEngine {
	engine := &RecommendationEngine{
		app:      app,
		patterns: make(map[string][]UserPattern),
		cache:    make(map[string]*RouteRecommendation),
	}

	// Iniciar análise de padrões em background
	go engine.startPatternAnalysis()

	return engine
}

// startPatternAnalysis inicia análise contínua de padrões
func (re *RecommendationEngine) startPatternAnalysis() {
	ticker := time.NewTicker(1 * time.Hour) // Analisar a cada hora
	defer ticker.Stop()

	for range ticker.C {
		re.analyzeUserPatterns()
		re.preloadPredictiveRoutes()
	}
}

// analyzeUserPatterns analisa padrões de comportamento dos usuários
func (re *RecommendationEngine) analyzeUserPatterns() {
	if re.app.db == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
	SELECT 
		user_id,
		origin,
		destination,
		EXTRACT(DOW FROM search_time)::int as day_of_week,
		EXTRACT(HOUR FROM search_time)::int as hour,
		EXTRACT(MINUTE FROM search_time)::int as minute,
		COUNT(*) as frequency,
		MAX(search_time) as last_seen
	FROM route_searches 
	WHERE search_time >= CURRENT_TIMESTAMP - INTERVAL '30 days'
	GROUP BY user_id, origin, destination, day_of_week, hour, minute
	HAVING COUNT(*) >= 3 -- Apenas padrões recorrentes
	ORDER BY frequency DESC, last_seen DESC
`

	rows, err := re.app.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao analisar padrões: %v", err)
		return
	}
	defer rows.Close()

	re.patternsMutex.Lock()
	defer re.patternsMutex.Unlock()

	newPatterns := make(map[string][]UserPattern)

	for rows.Next() {
		var pattern UserPattern
		err := rows.Scan(
			&pattern.UserID,
			&pattern.Origin,
			&pattern.Destination,
			&pattern.DayOfWeek,
			&pattern.Hour,
			&pattern.Minute,
			&pattern.Frequency,
			&pattern.LastSeen,
		)
		if err != nil {
			continue
		}

		// Calcular confiança baseada na frequência e recência
		daysSinceLastSeen := time.Since(pattern.LastSeen).Hours() / 24
		recencyScore := math.Max(0, 1.0-(daysSinceLastSeen/30.0))
		frequencyScore := math.Min(1.0, float64(pattern.Frequency)/10.0)
		pattern.Confidence = (recencyScore + frequencyScore) / 2.0

		newPatterns[pattern.UserID] = append(newPatterns[pattern.UserID], pattern)
	}

	re.patterns = newPatterns
	log.Printf("Analisados %d padrões de usuários", len(newPatterns))
}

// preloadPredictiveRoutes pré-carrega rotas baseadas em padrões
func (re *RecommendationEngine) preloadPredictiveRoutes() {
	re.patternsMutex.RLock()
	defer re.patternsMutex.RUnlock()

	now := time.Now()
	currentDayOfWeek := int(now.Weekday())
	currentHour := now.Hour()
	currentMinute := now.Minute()

	for userID, patterns := range re.patterns {
		for _, pattern := range patterns {
			// Verificar se o padrão corresponde ao horário atual (±15 minutos)
			if pattern.DayOfWeek == currentDayOfWeek &&
				pattern.Hour == currentHour &&
				math.Abs(float64(pattern.Minute-currentMinute)) <= 15 &&
				pattern.Confidence > 0.7 {

				// Pré-carregar rota no cache
				re.preloadRouteForUser(userID, pattern)
			}
		}
	}
}

// preloadRouteForUser pré-carrega uma rota específica para um usuário
func (re *RecommendationEngine) preloadRouteForUser(userID string, pattern UserPattern) {
	route, err := calculateMapRoute(pattern.Origin, pattern.Destination)
	if err != nil {
		return
	}

	// Ajustar rota com clima atual
	weather, exists := re.app.getWeatherData("goiania")
	if exists {
		route.TotalTimeMinutes = re.app.adjustRouteForWeather(route.TotalTimeMinutes)
	}

	recommendation := &RouteRecommendation{
		UserID:       userID,
		Route:        route,
		Reason:       "Rota previa carregada baseada no seu histórico",
		Confidence:   pattern.Confidence,
		IsPredictive: true,
		Weather:      weather,
		Alternatives: re.generateAlternatives(pattern.Origin, pattern.Destination),
	}

	cacheKey := fmt.Sprintf("recommendation:%s:%s:%s", userID, pattern.Origin, pattern.Destination)

	re.cacheMutex.Lock()
	re.cache[cacheKey] = recommendation
	re.cacheMutex.Unlock()

	// Salvar no Redis com TTL mais longo
	if re.app.rdb != nil {
		jsonData, _ := json.Marshal(recommendation)
		re.app.rdb.Set(context.Background(), cacheKey, string(jsonData), 2*time.Hour)
	}

	log.Printf("Rota pré-carregada para usuário %s: %s -> %s", userID, pattern.Origin, pattern.Destination)
}

// generateAlternatives gera rotas alternativas
func (re *RecommendationEngine) generateAlternatives(origin, destination string) []RouteAlternative {
	alternatives := []RouteAlternative{
		{
			Name:        "Rota Rápida",
			Description: "Menor tempo de viagem",
			TimeMinutes: 25,
			Cost:        4.30,
			Confidence:  0.8,
		},
		{
			Name:        "Rota Econômica",
			Description: "Menor custo",
			TimeMinutes: 35,
			Cost:        3.20,
			Confidence:  0.7,
		},
		{
			Name:        "Rota Direta",
			Description: "Menos transferências",
			TimeMinutes: 30,
			Cost:        4.30,
			Confidence:  0.9,
		},
	}

	// Ordenar por confiança
	sort.Slice(alternatives, func(i, j int) bool {
		return alternatives[i].Confidence > alternatives[j].Confidence
	})

	return alternatives
}

// GetUserRecommendations obtém recomendações para um usuário
func (re *RecommendationEngine) GetUserRecommendations(userID, origin, destination string) (*RouteRecommendation, error) {
	// Verificar cache primeiro
	cacheKey := fmt.Sprintf("recommendation:%s:%s:%s", userID, origin, destination)

	re.cacheMutex.RLock()
	if cached, exists := re.cache[cacheKey]; exists {
		re.cacheMutex.RUnlock()
		return cached, nil
	}
	re.cacheMutex.RUnlock()

	// Gerar recomendação em tempo real
	recommendation := re.generateRealtimeRecommendation(userID, origin, destination)

	// Salvar no cache
	re.cacheMutex.Lock()
	re.cache[cacheKey] = recommendation
	re.cacheMutex.Unlock()

	return recommendation, nil
}

// generateRealtimeRecommendation gera recomendação em tempo real
func (re *RecommendationEngine) generateRealtimeRecommendation(userID, origin, destination string) *RouteRecommendation {
	route, err := calculateMapRoute(origin, destination)
	if err != nil {
		// Fallback
		route = &MapRouteResponse{
			Origin:           MapPoint{Name: origin},
			Destination:      MapPoint{Name: destination},
			TotalTimeMinutes: 30,
			BusLines:         []string{"M23", "M71"},
		}
	}

	// Ajustar com clima
	weather, exists := re.app.getWeatherData("goiania")
	if exists {
		route.TotalTimeMinutes = re.app.adjustRouteForWeather(route.TotalTimeMinutes)
	}

	// Verificar se existe padrão similar
	re.patternsMutex.RLock()
	var confidence float64 = 0.5
	var reason string = "Rota calculada em tempo real"

	if patterns, exists := re.patterns[userID]; exists {
		for _, pattern := range patterns {
			// Verificar se origem/destino são similares
			if re.isSimilarRoute(pattern.Origin, origin) && re.isSimilarRoute(pattern.Destination, destination) {
				confidence = pattern.Confidence
				reason = fmt.Sprintf("Baseado no seu histórico (%d buscas similares)", pattern.Frequency)
				break
			}
		}
	}
	re.patternsMutex.RUnlock()

	return &RouteRecommendation{
		UserID:       userID,
		Route:        route,
		Reason:       reason,
		Confidence:   confidence,
		IsPredictive: false,
		Weather:      weather,
		Alternatives: re.generateAlternatives(origin, destination),
	}
}

// isSimilarRoute verifica se duas rotas são similares
func (re *RecommendationEngine) isSimilarRoute(route1, route2 string) bool {
	// Normalizar strings
	r1 := normalizeParam(route1)
	r2 := normalizeParam(route2)

	// Verificar correspondência exata ou parcial
	return r1 == r2 ||
		(len(r1) > 3 && len(r2) > 3 &&
			(contains(r1, r2) || contains(r2, r1)))
}

// contains verifica se uma string contém outra
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOf(s, substr) >= 0))
}

// indexOf encontra a primeira ocorrência de uma substring
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// RecordUserSearch registra uma busca de usuário para análise de padrões
func (re *RecommendationEngine) RecordUserSearch(userID, origin, destination string) {
	if re.app.db == nil {
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Obter coordenadas
		originCoords := getLocationCoordinates(origin)
		destCoords := getLocationCoordinates(destination)

		query := `
		INSERT INTO route_searches (user_id, origin, destination, origin_location, destination_location, search_time)
		VALUES ($1, $2, $3, ST_GeomFromText($4, 4326), ST_GeomFromText($5, 4326), $6)
		ON CONFLICT (user_id, origin, destination, search_time) DO NOTHING
		`

		_, err := re.app.db.ExecContext(ctx, query,
			userID, origin, destination,
			fmt.Sprintf("POINT(%f %f)", originCoords.X, originCoords.Y),
			fmt.Sprintf("POINT(%f %f)", destCoords.X, destCoords.Y),
			time.Now(),
		)
		if err != nil {
			log.Printf("Erro ao registrar busca: %v", err)
		}
	}()
}

// GetPredictiveRoutes obtém rotas preditivas para o usuário atual
func (re *RecommendationEngine) GetPredictiveRoutes(userID string) []*RouteRecommendation {
	re.patternsMutex.RLock()
	defer re.patternsMutex.RUnlock()

	var recommendations []*RouteRecommendation
	now := time.Now()
	currentDayOfWeek := int(now.Weekday())
	currentHour := now.Hour()

	if patterns, exists := re.patterns[userID]; exists {
		for _, pattern := range patterns {
			// Verificar se é um padrão para o dia/horário atual
			if pattern.DayOfWeek == currentDayOfWeek && pattern.Hour == currentHour && pattern.Confidence > 0.6 {
				cacheKey := fmt.Sprintf("recommendation:%s:%s:%s", userID, pattern.Origin, pattern.Destination)

				re.cacheMutex.RLock()
				if cached, exists := re.cache[cacheKey]; exists {
					recommendations = append(recommendations, cached)
				}
				re.cacheMutex.RUnlock()
			}
		}
	}

	return recommendations
}

// setupRecommendationRoutes configura as rotas do motor de recomendação
func setupRecommendationRoutes(r *gin.Engine, app *App) {
	engine := NewRecommendationEngine(app)

	// GET /api/v1/recommendations - Obter recomendações personalizadas
	r.GET("/api/v1/recommendations", func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = "anonymous"
		}

		origin := c.Query("origin")
		destination := c.Query("destination")

		if origin == "" || destination == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "origin e destination são obrigatórios"})
			return
		}

		// Registrar busca para análise de padrões
		engine.RecordUserSearch(userID, origin, destination)

		// Obter recomendações
		recommendation, err := engine.GetUserRecommendations(userID, origin, destination)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao gerar recomendação"})
			return
		}

		c.JSON(http.StatusOK, recommendation)
	})

	// GET /api/v1/predictive-routes - Obter rotas preditivas
	r.GET("/api/v1/predictive-routes", func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		if userID == "" {
			userID = "anonymous"
		}

		recommendations := engine.GetPredictiveRoutes(userID)
		c.JSON(http.StatusOK, gin.H{
			"recommendations": recommendations,
			"count":           len(recommendations),
		})
	})

	// POST /api/v1/feedback - Registrar feedback sobre recomendação
	r.POST("/api/v1/feedback", func(c *gin.Context) {
		var feedback struct {
			UserID      string `json:"user_id"`
			Origin      string `json:"origin"`
			Destination string `json:"destination"`
			Rating      int    `json:"rating"` // 1-5
			Comments    string `json:"comments"`
		}

		if err := c.ShouldBindJSON(&feedback); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
			return
		}

		log.Printf("Feedback recebido: User=%s, Rating=%d, %s -> %s",
			feedback.UserID, feedback.Rating, feedback.Origin, feedback.Destination)

		c.JSON(http.StatusOK, gin.H{"message": "Feedback registrado com sucesso"})
	})
}
