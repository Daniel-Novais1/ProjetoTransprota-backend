package main

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/telemetry"
)

type App struct {
	db           *sql.DB
	rdb          *redis.Client
	routeTTL     time.Duration
	requestCount int64
	errorCount   int64
	startTime    time.Time
	weatherCache map[string]WeatherData
	weatherMutex sync.RWMutex
}

// WeatherData representa dados do clima
type WeatherData struct {
	City        string    `json:"city"`
	Temperature float64   `json:"temperature"`
	Humidity    int       `json:"humidity"`
	Description string    `json:"description"`
	IsRaining   bool      `json:"is_raining"`
	WindSpeed   float64   `json:"wind_speed"`
	Visibility  int       `json:"visibility"`
	Timestamp   time.Time `json:"timestamp"`
}

// WalkabilitySuggestion representa sugestões de caminhabilidade
type WalkabilitySuggestion struct {
	IsWalkable     bool    `json:"is_walkable"`
	DistanceKm     float64 `json:"distance_km"`
	WalkTimeMin    int     `json:"walk_time_min"`
	MoneySaved     float64 `json:"money_saved"`
	ExerciseMin    int     `json:"exercise_min"`
	CaloriesBurned int     `json:"calories_burned"`
	Recommendation string  `json:"recommendation"`
	WeatherFactor  string  `json:"weather_factor"`
}

// SpatialPoint representa um ponto geográfico com suporte PostGIS
type SpatialPoint struct {
	X float64 `json:"x"` // Longitude
	Y float64 `json:"y"` // Latitude
}

// Scan implementa interface para PostgreSQL GEOMETRY
func (sp *SpatialPoint) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case string:
		// Parse WKT format: "POINT(lng lat)"
		if len(v) > 7 && v[:6] == "POINT(" && v[len(v)-1] == ')' {
			coords := v[6 : len(v)-1]
			if n, err := fmt.Sscanf(coords, "%f %f", &sp.X, &sp.Y); err == nil && n == 2 {
				return nil
			}
		}
		return fmt.Errorf("invalid POINT format: %s", v)
	case []byte:
		return sp.Scan(string(v))
	default:
		return fmt.Errorf("cannot scan %T into SpatialPoint", value)
	}
}

// MarshalJSON implementa interface para compatibilidade com frontend
func (sp SpatialPoint) MarshalJSON() ([]byte, error) {
	// Frontend espera formato {x: lng, y: lat} para compatibilidade
	return json.Marshal(struct {
		X float64 `json:"x"` // Longitude
		Y float64 `json:"y"` // Latitude
	}{
		X: sp.X,
		Y: sp.Y,
	})
}

// ToLegacyPoint converte para formato legado (compatibilidade)
func (sp SpatialPoint) ToLegacyPoint() MapPoint {
	return MapPoint{
		Name:      "Location",
		Latitude:  sp.Y, // Y = Latitude
		Longitude: sp.X, // X = Longitude
	}
}

// Terminal representa um terminal de ônibus com coordenadas espaciais.
type Terminal struct {
	ID       int          `json:"id"`
	Nome     string       `json:"nome"`
	Location SpatialPoint `json:"location"` // GEOMETRY(Point, 4326)
}

// GPSData contém dados de localização de um ônibus com coordenadas espaciais.
type GPSData struct {
	BusID     string       `json:"bus_id"`
	Location  SpatialPoint `json:"location"` // GEOMETRY(Point, 4326)
	Timestamp time.Time    `json:"timestamp"`
}

// StatusResponse responde com o status de proximidade de um ônibus a terminais.
type StatusResponse struct {
	BusID           string  `json:"bus_id"`
	Status          string  `json:"status"`
	Terminal        string  `json:"terminal,omitempty"`
	DistanciaMetros float64 `json:"distancia_metros,omitempty"`
}

// ListaTerminaisResponse lista terminais disponíveis.
type ListaTerminaisResponse struct {
	Total     int        `json:"total"`
	Terminais []Terminal `json:"terminais"`
}

// ErrorResponse estrutura para respostas de erro.
type ErrorResponse struct {
	Error      string `json:"error"`
	HTTPStatus int    `json:"-"`
}

// RouteStep representa um passo em uma rota de ônibus.
type RouteStep struct {
	NumeroLinha       string   `json:"numero_linha"`
	NomeLinha         string   `json:"nome_linha"`
	Paradas           []string `json:"paradas"`
	TempoTotalMinutos int      `json:"tempo_total_minutos"`
}

// RouteResponse contém a resposta completa de uma rota planejada.
type RouteResponse struct {
	Origem  string      `json:"origem"`
	Destino string      `json:"destino"`
	Tipo    string      `json:"tipo"`
	Steps   []RouteStep `json:"steps"`
	Cached  bool        `json:"cached"`
}

// Denuncia representa uma denúncia colaborativa de problema no ônibus.
type Denuncia struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	BusLine     string    `json:"bus_line"`
	BusID       string    `json:"bus_id"`
	Type        string    `json:"type"`     // Lotado, Atrasado, Não Parou, Ar Estragado, Sujo
	Location    string    `json:"location"` // WKT Point
	Timestamp   time.Time `json:"timestamp"`
	EvidenceURL string    `json:"evidence_url,omitempty"`
	TrustScore  int       `json:"trust_score"`
}

// TrustScore representa o score de confiança de um usuário.
type TrustScore struct {
	UserID string `json:"user_id"`
	Score  int    `json:"score"`
	Level  string `json:"level"` // Suspeito, Cidadão, Fiscal da Galera
}

// SubmeterDenunciaRequest estrutura a requisição para submeter denúncia.
type SubmeterDenunciaRequest struct {
	UserID      string  `json:"user_id" binding:"required"`
	BusLine     string  `json:"bus_line" binding:"required"`
	BusID       string  `json:"bus_id" binding:"required"`
	Type        string  `json:"type" binding:"required,oneof=Lotado Atrasado Não Parou Ar Estragado Sujo"`
	Latitude    float64 `json:"latitude" binding:"required,gte=-90,lte=90"`
	Longitude   float64 `json:"longitude" binding:"required,gte=-180,lte=180"`
	EvidenceURL string  `json:"evidence_url,omitempty"`
}

// ListaDenunciasResponse estrutura a resposta de listagem de denúncias.
type ListaDenunciasResponse struct {
	Total     int        `json:"total"`
	Denuncias []Denuncia `json:"denuncias"`
}

// MapPoint representa um ponto geográfico para o mapa.
type MapPoint struct {
	Name      string  `json:"name"`
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
}

// MapStep representa um passo na rota do mapa.
type MapStep struct {
	Name       string  `json:"name"`
	Latitude   float64 `json:"lat"`
	Longitude  float64 `json:"lng"`
	IsTerminal bool    `json:"is_terminal"`
	IsTransfer bool    `json:"is_transfer"`
}

// MapRouteResponse representa a rota completa para visualização no mapa.
type MapRouteResponse struct {
	Origin           MapPoint  `json:"origin"`
	Destination      MapPoint  `json:"destination"`
	Steps            []MapStep `json:"steps"`
	TotalTimeMinutes int       `json:"total_time_minutes"`
	BusLines         []string  `json:"bus_lines"`
}

// RouteSearch representa uma busca de rota para persistência (LGPD-Compliant)
type RouteSearch struct {
	ID          int64     `json:"id"`
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	SearchTime  time.Time `json:"search_time"`
	IsRushHour  bool      `json:"is_rush_hour"`
	DayOfWeek   int       `json:"day_of_week"` // 0=Sunday, 6=Saturday
	// Nota: NÃO salvamos IP ou dados identificáveis (LGPD-Compliance)
}

// TrendingRoute representa rota mais buscada
type TrendingRoute struct {
	Origin      string    `json:"origin"`
	Destination string    `json:"destination"`
	Count       int64     `json:"count"`
	LastSearch  time.Time `json:"last_search"`
}

var denunciaPool = sync.Pool{
	New: func() interface{} {
		return &Denuncia{}
	},
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: Arquivo .env não encontrado")
	}

	app, err := newApp()
	if err != nil {
		log.Printf("Aviso: Falha ao inicializar bancos, operando em modo degradado: %v", err)
		// Criar app mesmo sem bancos para fail-soft
		app = &App{
			routeTTL:  15 * time.Minute,
			startTime: time.Now(),
			db:        nil, // Sem conexão PostgreSQL
			rdb:       nil, // Sem conexão Redis
		}
	}
	defer app.Close()

	r := gin.Default()
	r.Use(corsMiddleware())

	// Configurar graceful shutdown
	go func() {
		// Criar servidor HTTP com graceful shutdown
		server := &http.Server{
			Addr:              ":8080",
			Handler:           r,
			ReadTimeout:       15 * time.Second,
			WriteTimeout:      15 * time.Second,
			IdleTimeout:       60 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		}

		// Iniciar servidor em goroutine separada
		go func() {
			log.Println("TranspRota API operacional em http://localhost:8080")
			log.Println("Modo:", getAppMode(app))

			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("Erro ao iniciar servidor: %v", err)
			}
		}()

		// Configurar graceful shutdown
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		log.Println("\n=== INICIANDO GRACEFUL SHUTDOWN ===")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Desabilitar novas requisições
		server.SetKeepAlivesEnabled(false)

		// Aguardar requisições ativas terminarem
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("Erro durante shutdown: %v", err)
		} else {
			log.Println("Servidor encerrado com sucesso")
		}

		// Fechar conexões
		if app.db != nil {
			app.db.Close()
			log.Println("PostgreSQL fechado")
		}
		if app.rdb != nil {
			app.rdb.Close()
			log.Println("Redis fechado")
		}

		log.Println("=== GRACEFUL SHUTDOWN COMPLETO ===")
		os.Exit(0)
	}()
	r.Use(errorHandlerMiddleware())
	r.Use(RateLimitMiddleware()) // Rate limiting para prevenir abuso
	setupRoutes(r, app)

	fmt.Println("\nTranspRota API operacional em http://localhost:8080")
	fmt.Println("Modo:", getAppMode(app))

	if err := r.Run(":8080"); err != nil {
		log.Printf("Erro ao iniciar servidor: %v", err)
		// Não usar log.Fatal - apenas log e continue
	}
}

// corsMiddleware configura CORS rigoroso com cabeçalhos de segurança
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Em produção, use apenas domínios autorizados
		allowedOrigins := []string{
			"https://transprota.app",       // Frontend oficial
			"https://www.transprota.app",   // WWW redirect
			"https://admin.transprota.app", // Dashboard admin
			"http://localhost:3000",        // Desenvolvimento
			"http://localhost:3001",        // Desenvolvimento secundário
			"http://127.0.0.1:3000",        // Desenvolvimento local
			"http://127.0.0.1:3001",        // Desenvolvimento local secundário
		}

		origin := c.Request.Header.Get("Origin")
		allowed := false

		// Verificar se origin está na whitelist
		for _, allowedOrigin := range allowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		// CORS rigoroso - apenas origens permitidas
		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		// Cabeçalhos CORS específicos e minimalistas
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Requested-With")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Max-Age", "86400") // 24 horas cache

		// Cabeçalhos de segurança rigorosos
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		c.Header("Cross-Origin-Embedder-Policy", "require-corp")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")

		// HSTS apenas em HTTPS
		if c.Request.TLS != nil || strings.HasPrefix(origin, "https://") {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
		}

		// Content Security Policy
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'"

		c.Header("Content-Security-Policy", csp)

		// Rate limiting por origem
		clientIP := c.ClientIP()
		userAgent := c.Request.Header.Get("User-Agent")

		// Ignorar verificação de User-Agent para health checks e telemetria
		// Estes endpoints precisam ser acessíveis por ferramentas de monitoramento
		exemptPaths := []string{"/health", "/api/v1/health", "/api/v1/telemetry/gps"}
		requestPath := c.Request.URL.Path
		for _, exempt := range exemptPaths {
			if requestPath == exempt || strings.HasPrefix(requestPath, exempt) {
				c.Next()
				return
			}
		}

		// Bloquear user agents suspeitos (exceto ferramentas de monitoramento legítimas)
		suspiciousAgents := []string{
			"python-requests", "java", "apache-httpclient",
			"bot", "crawler", "spider", "scraper", "scanner",
		}

		userAgentLower := strings.ToLower(userAgent)
		for _, agent := range suspiciousAgents {
			if strings.Contains(userAgentLower, agent) {
				log.Printf("Acesso bloqueado - User Agent suspeito: %s from %s", userAgent, clientIP)
				c.JSON(http.StatusForbidden, gin.H{"error": "Acesso não permitido"})
				c.Abort()
				return
			}
		}

		// OPTIONS preflight
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
func errorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if len(c.Errors) > 0 {
			err := c.Errors[0]
			log.Printf("❌ Erro: %v", err)
		}
	}
}

// AuthMiddleware valida a chave de API para acesso protegido.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKey := c.GetHeader("X-API-Key")
		secret := os.Getenv("API_SECRET_KEY")

		if apiKey == "" || secret == "" || subtle.ConstantTimeCompare([]byte(apiKey), []byte(secret)) != 1 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Acesso negado"})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimitMiddleware limita requisições por IP para prevenir abuso.
// Configuração: 100 requisições por minuto (600ms entre requisições)
// Desativado para endpoint /api/v1/telemetry/gps durante fase de testes
func RateLimitMiddleware() gin.HandlerFunc {
	visitors := make(map[string]time.Time)
	var mu sync.Mutex

	// 100 req/min = 1 req a cada 600ms
	const minInterval = 600 * time.Millisecond

	// Endpoints isentos de rate limiting (fase de desenvolvimento/testes)
	exemptPaths := map[string]bool{
		"/api/v1/telemetry/gps": true,
		"/health":               true,
		"/api/v1/health":        true,
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// Verificar se endpoint está isento
		if exemptPaths[path] {
			c.Next()
			return
		}

		mu.Lock()
		ip := c.ClientIP()
		if lastTime, exists := visitors[ip]; exists {
			if time.Since(lastTime) < minInterval {
				mu.Unlock()
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error":        "Limite de requisições excedido",
					"limit":        "100 req/min",
					"retry_after":  int(minInterval.Seconds()),
					"exempt_paths": []string{"/api/v1/telemetry/gps", "/health"},
				})
				c.Abort()
				return
			}
		}
		visitors[ip] = time.Now()
		mu.Unlock()
		c.Next()
	}
}

// ValidateGeoCoords valida se latitude/longitude são válidas.
func ValidateGeoCoords(lat, lon string) (float64, float64, error) {
	latFloat, err := strconv.ParseFloat(lat, 64)
	if err != nil || latFloat < -90 || latFloat > 90 {
		return 0, 0, fmt.Errorf("latitude inválida")
	}
	lonFloat, err := strconv.ParseFloat(lon, 64)
	if err != nil || lonFloat < -180 || lonFloat > 180 {
		return 0, 0, fmt.Errorf("longitude inválida")
	}
	return latFloat, lonFloat, nil
}

// ValidateString valida string com limite de tamanho.
func ValidateString(s string, maxLen int) error {
	if len(strings.TrimSpace(s)) == 0 {
		return fmt.Errorf("string vazia")
	}
	if len(s) > maxLen {
		return fmt.Errorf("string excede limite de %d caracteres", maxLen)
	}
	return nil
}

func setupRoutes(r *gin.Engine, app *App) {
	// Middleware para contar requisições
	r.Use(func(c *gin.Context) {
		atomic.AddInt64(&app.requestCount, 1)
		c.Next()
		if len(c.Errors) > 0 {
			atomic.AddInt64(&app.errorCount, 1)
		}
	})

	// GET /health - Health check
	r.GET("/health", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Verificar PostgreSQL
		dbHealth := "offline"
		if app.db != nil {
			if err := app.db.PingContext(ctx); err == nil {
				dbHealth = "ok"
			} else {
				dbHealth = "error"
			}
		}

		// Verificar Redis
		redisHealth := "offline"
		if app.rdb != nil {
			if err := app.rdb.Ping(ctx).Err(); err == nil {
				redisHealth = "ok"
			} else {
				redisHealth = "error"
			}
		}

		status := "ok"
		httpStatus := http.StatusOK
		if dbHealth != "ok" || redisHealth != "ok" {
			status = "degraded"
			if dbHealth == "offline" && redisHealth == "offline" {
				status = "offline"
				httpStatus = http.StatusServiceUnavailable
			}
		}

		c.JSON(httpStatus, gin.H{
			"status":    status,
			"timestamp": time.Now().UTC(),
			"database":  dbHealth,
			"redis":     redisHealth,
			"uptime":    int(time.Since(app.startTime).Seconds()),
			"mode":      getAppMode(app),
		})
	})
	// GET /api/v1/route-presets - Retorna presets das 3 rotas críticas
	r.GET("/api/v1/route-presets", func(c *gin.Context) {
		presets := []gin.H{
			{
				"id":             1,
				"name":           "Terminal Novo Mundo -> Campus Samambaia",
				"origin":         "Terminal Novo Mundo",
				"destination":    "Campus Samambaia UFG",
				"description":    "Eixo Norte-Sul com acesso universitário",
				"complexity":     "Média",
				"estimated_time": "61 min (horário de pico)",
				"bus_lines":      []string{"M23", "M71", "M43"},
			},
			{
				"id":             2,
				"name":           "Terminal Bíblia -> Terminal Canedo",
				"origin":         "Terminal Bíblia",
				"destination":    "Terminal Canedo",
				"description":    "Integração intermunicipal Goiânia-Senador Canedo",
				"complexity":     "Alta",
				"estimated_time": "162 min (horário de pico)",
				"bus_lines":      []string{"M10", "M60", "INTERMUNICIPAL"},
			},
			{
				"id":             3,
				"name":           "Terminal Isidória -> Terminal Padre Pelágio",
				"origin":         "Terminal Isidória",
				"destination":    "Terminal Padre Pelágio",
				"description":    "Cruzamento transversal de alto fluxo",
				"complexity":     "Média",
				"estimated_time": "70 min (horário de pico)",
				"bus_lines":      []string{"M33", "M55", "M77"},
			},
		}

		c.JSON(http.StatusOK, gin.H{
			"presets":      presets,
			"total":        len(presets),
			"current_hour": time.Now().Hour(),
			"is_rush_hour": time.Now().Hour() >= 17 && time.Now().Hour() <= 19,
		})
	})

	// GET /api/v1/trending - Retorna as 3 rotas mais buscadas
	r.GET("/api/v1/trending", func(c *gin.Context) {
		trending, err := getTrendingRoutes(app)
		if err != nil {
			// Se falhar, retornar presets como fallback
			presets := []gin.H{
				{
					"origin":      "Terminal Novo Mundo",
					"destination": "Campus Samambaia UFG",
					"count":       0,
					"last_search": time.Now(),
				},
				{
					"origin":      "Terminal Bíblia",
					"destination": "Terminal Canedo",
					"count":       0,
					"last_search": time.Now(),
				},
				{
					"origin":      "Terminal Isidória",
					"destination": "Terminal Padre Pelágio",
					"count":       0,
					"last_search": time.Now(),
				},
			}
			c.JSON(http.StatusOK, gin.H{
				"trending": presets,
				"total":    len(presets),
				"fallback": true,
				"message":  "Usando presets - analytics indisponível",
			})
			return
		}

		// Warm-up cache para rotas trending
		warmUpTrendingCache(app, trending)

		c.JSON(http.StatusOK, gin.H{
			"trending": trending,
			"total":    len(trending),
			"fallback": false,
			"period":   "Últimos 7 dias",
		})
	})

	// GET /linhas - Listar todas as linhas ativas
	r.GET("/linhas", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := app.db.QueryContext(ctx, "SELECT numero_linha, nome_linha FROM linhas_onibus WHERE status = 'ativa'")
		if err != nil {
			log.Printf("❌ Erro ao consultar linhas: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao consultar linhas"})
			return
		}
		defer rows.Close()

		var linhas []gin.H
		for rows.Next() {
			var numero, nome string
			if err := rows.Scan(&numero, &nome); err != nil {
				log.Printf("❌ Erro ao ler linha: %v", err)
				continue
			}
			linhas = append(linhas, gin.H{"numero": numero, "nome": nome})
		}

		c.JSON(http.StatusOK, gin.H{
			"total":  len(linhas),
			"linhas": linhas,
		})
	})

	// GET /planejar - Calcular rota entre origem e destino com cache Redis
	r.GET("/planejar", func(c *gin.Context) {
		origem := c.Query("origem")
		destino := c.Query("destino")
		if origem == "" || destino == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "origem e destino são obrigatórios"})
			return
		}

		// Fallback hardcoded quando bancos offline
		if app.db == nil || app.rdb == nil {
			log.Printf("Bancos offline, usando fallback hardcoded para %s -> %s", origem, destino)

			// Fallback para rota crítica: Setor Bueno -> Campus Samambaia
			origemNorm := strings.ToLower(strings.TrimSpace(origem))
			destinoNorm := strings.ToLower(strings.TrimSpace(destino))

			if (strings.Contains(origemNorm, "bueno") || strings.Contains(origemNorm, "setor bueno")) &&
				(strings.Contains(destinoNorm, "samambaia") || strings.Contains(destinoNorm, "campus") || strings.Contains(destinoNorm, "ufg")) {

				fallbackRoute := RouteResponse{
					Origem:  "Setor Bueno",
					Destino: "Campus Samambaia UFG",
					Tipo:    "direta",
					Steps: []RouteStep{{
						NumeroLinha:       "M23",
						NomeLinha:         "Terminal Novo Mundo - Campus Samambaia",
						Paradas:           []string{"Setor Bueno", "Terminal Novo Mundo", "Campus Samambaia UFG"},
						TempoTotalMinutos: 25,
					}},
					Cached: false,
				}

				c.JSON(http.StatusOK, gin.H{
					"route":   fallbackRoute,
					"mode":    "fallback (offline)",
					"warning": "Serviço operando em modo degradado - dados básicos",
				})
				return
			}

			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":      "Serviço indisponível - bancos de dados offline",
				"mode":       "offline",
				"suggestion": "Tente a rota: Setor Bueno -> Campus Samambaia (disponível offline)",
			})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cached, err := app.getRouteCache(ctx, origem, destino)
		if err != nil {
			log.Printf("Erro ao ler cache de rota: %v", err)
		}
		if cached != nil {
			c.JSON(http.StatusOK, cached)
			return
		}

		route, err := app.calculateRoute(ctx, origem, destino)
		if err != nil {
			log.Printf("Erro ao calcular rota: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao calcular rota"})
			return
		}
		if route == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nenhuma rota encontrada para este par"})
			return
		}

		if err := app.setRouteCache(ctx, origem, destino, route); err != nil {
			log.Printf("Falha ao gravar cache de rota: %v", err)
		}

		c.JSON(http.StatusOK, route)
	})

	// GET /terminais - Listar todos os terminais com PostGIS
	r.GET("/terminais", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := app.db.QueryContext(ctx, "SELECT id, name, ST_AsText(location) FROM locations")
		if err != nil {
			log.Printf("Erro ao consultar terminais: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao consultar banco"})
			return
		}
		defer rows.Close()

		var terminais []Terminal
		for rows.Next() {
			var terminal Terminal
			var locationText string
			if err := rows.Scan(&terminal.ID, &terminal.Nome, &locationText); err != nil {
				log.Printf("Erro ao ler linha: %v", err)
				continue
			}
			// Parse WKT format
			terminal.Location.Scan(locationText)
			terminais = append(terminais, terminal)
		}

		c.JSON(http.StatusOK, gin.H{
			"total":     len(terminais),
			"terminais": terminais,
		})
	})

	// GET /api/v1/routes/near - Buscar rotas próximas a um ponto (PostGIS)
	r.GET("/api/v1/routes/near", func(c *gin.Context) {
		latStr := c.Query("lat")
		lonStr := c.Query("lon")
		radiusStr := c.Query("radius") // em metros, padrão 1000

		if latStr == "" || lonStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Coordenadas lat e lon são obrigatórias"})
			return
		}

		// Validar coordenadas
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil || lat < -90 || lat > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude inválida"})
			return
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil || lon < -180 || lon > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Longitude inválida"})
			return
		}

		radius := 1000.0
		if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 && r <= 10000 {
			radius = r
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		center := SpatialPoint{X: lon, Y: lat}
		routes, err := app.findRoutesNearPoint(ctx, center, radius, 20)
		if err != nil {
			log.Printf("Erro ao buscar rotas próximas: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar rotas"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"center": center,
			"radius": radius,
			"routes": routes,
			"total":  len(routes),
		})
	})

	// GET /api/v1/routes/area - Buscar rotas em área retangular (PostGIS)
	r.GET("/api/v1/routes/area", func(c *gin.Context) {
		minLatStr := c.Query("min_lat")
		maxLatStr := c.Query("max_lat")
		minLngStr := c.Query("min_lng")
		maxLngStr := c.Query("max_lng")

		if minLatStr == "" || maxLatStr == "" || minLngStr == "" || maxLngStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Coordenadas da área são obrigatórias"})
			return
		}

		// Validar coordenadas
		minLat, err := strconv.ParseFloat(minLatStr, 64)
		if err != nil || minLat < -90 || minLat > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude mínima inválida"})
			return
		}

		maxLat, err := strconv.ParseFloat(maxLatStr, 64)
		if err != nil || maxLat < -90 || maxLat > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude máxima inválida"})
			return
		}

		minLng, err := strconv.ParseFloat(minLngStr, 64)
		if err != nil || minLng < -180 || minLng > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Longitude mínima inválida"})
			return
		}

		maxLng, err := strconv.ParseFloat(maxLngStr, 64)
		if err != nil || maxLng < -180 || maxLng > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Longitude máxima inválida"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		routes, err := app.findRoutesInArea(ctx, minLat, minLng, maxLat, maxLng, 50)
		if err != nil {
			log.Printf("Erro ao buscar rotas na área: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar rotas"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"area": gin.H{
				"min_lat": minLat,
				"max_lat": maxLat,
				"min_lng": minLng,
				"max_lng": maxLng,
			},
			"routes": routes,
			"total":  len(routes),
		})
	})

	// GET /api/v1/routes/density - Calcular densidade de rotas (PostGIS)
	r.GET("/api/v1/routes/density", func(c *gin.Context) {
		latStr := c.Query("lat")
		lonStr := c.Query("lon")
		radiusStr := c.Query("radius") // em metros, padrão 1000

		if latStr == "" || lonStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Coordenadas lat e lon são obrigatórias"})
			return
		}

		// Validar coordenadas
		lat, err := strconv.ParseFloat(latStr, 64)
		if err != nil || lat < -90 || lat > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude inválida"})
			return
		}

		lon, err := strconv.ParseFloat(lonStr, 64)
		if err != nil || lon < -180 || lon > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Longitude inválida"})
			return
		}

		radius := 1000.0
		if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 && r <= 10000 {
			radius = r
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		center := SpatialPoint{X: lon, Y: lat}
		density, err := app.getRouteDensity(ctx, center, radius)
		if err != nil {
			log.Printf("Erro ao calcular densidade: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao calcular densidade"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"center":   center,
			"radius":   radius,
			"density":  density,
			"area_km2": (radius * 2 * radius * 2) / 1000000, // área aproximada em km²
		})
	})

	// GET /api/v1/reports/heatmap - Obter dados do mapa de calor
	r.GET("/api/v1/reports/heatmap", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		heatmapData, err := app.getHeatmapData(ctx)
		if err != nil {
			log.Printf("Erro ao buscar heatmap data: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar dados do heatmap"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"heatmap": heatmapData,
			"count":   len(heatmapData),
		})
	})

	// GET /api/v1/reports/recent - Obter denúncias recentes para exibição no mapa
	r.GET("/api/v1/reports/recent", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		reports, err := app.getRecentReports(ctx)
		if err != nil {
			log.Printf("Erro ao buscar denúncias recentes: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao buscar denúncias"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"reports": reports,
			"count":   len(reports),
		})
	})

	// GET /api/v1/map-view - Endpoint para visualização de mapa
	r.GET("/api/v1/map-view", func(c *gin.Context) {
		origin := c.Query("origin")
		destination := c.Query("destination")

		if origin == "" || destination == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "origin e destination são obrigatórios"})
			return
		}

		// Se bancos offline, usar fallback
		if app.db == nil {
			log.Printf("Bancos offline, usando fallback hardcoded para map-view %s -> %s", origin, destination)

			// Fallback hardcoded para rota básica
			originNorm := strings.ToLower(strings.TrimSpace(origin))
			destNorm := strings.ToLower(strings.TrimSpace(destination))

			locations := map[string]MapPoint{
				"bueno":      {Name: "Setor Bueno", Latitude: -16.6864, Longitude: -49.2643},
				"centro":     {Name: "Setor Centro", Latitude: -16.6807, Longitude: -49.2671},
				"samambaia":  {Name: "Campus Samambaia", Latitude: -16.6830, Longitude: -49.2670},
				"novo mundo": {Name: "Terminal Novo Mundo", Latitude: -16.6860, Longitude: -49.2640},
			}

			originPoint, originExists := locations[originNorm]
			destPoint, destExists := locations[destNorm]

			if !originExists || !destExists {
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error":      "Serviço indisponível - bancos de dados offline",
					"mode":       "offline",
					"suggestion": "Tente: Bueno -> Centro (disponível offline)",
				})
				return
			}

			fallbackRoute := &MapRouteResponse{
				Origin:      originPoint,
				Destination: destPoint,
				Steps: []MapStep{
					{
						Name:       originPoint.Name,
						Latitude:   originPoint.Latitude,
						Longitude:  originPoint.Longitude,
						IsTerminal: false,
						IsTransfer: false,
					},
					{
						Name:       destPoint.Name,
						Latitude:   destPoint.Latitude,
						Longitude:  destPoint.Longitude,
						IsTerminal: false,
						IsTransfer: false,
					},
				},
				TotalTimeMinutes: 25,
				BusLines:         []string{"M23"},
			}

			c.JSON(http.StatusOK, gin.H{
				"route":   fallbackRoute,
				"mode":    "fallback (offline)",
				"warning": "Serviço operando em modo degradado - dados básicos",
			})
			return
		}

		// Usar função existente calculateMapRoute
		route, err := calculateMapRoute(origin, destination)
		if err != nil {
			log.Printf("Erro ao calcular rota para mapa: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao calcular rota"})
			return
		}
		if route == nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Nenhuma rota encontrada"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"route": route,
			"mode":  "full",
		})
	})

	// POST /api/v1/reports - Enviar nova denúncia
	r.POST("/api/v1/reports", func(c *gin.Context) {
		var report UserReport
		if err := c.ShouldBindJSON(&report); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: tipo_problema, latitude, longitude obrigatórios"})
			return
		}

		// Validar tipo de problema
		validTypes := map[string]bool{"Lotado": true, "Atrasado": true, "Perigo": true}
		if !validTypes[report.TipoProblema] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Tipo de problema inválido. Use: Lotado, Atrasado ou Perigo"})
			return
		}

		// Validar coordenadas
		if report.Latitude < -90 || report.Latitude > 90 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude deve estar entre -90 e 90"})
			return
		}
		if report.Longitude < -180 || report.Longitude > 180 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Longitude deve estar entre -180 e 180"})
			return
		}

		// Obter IP do usuário e criar hash para anti-spam
		userIP := c.ClientIP()
		hash := sha256.Sum256([]byte(userIP))
		report.UserIPHash = fmt.Sprintf("%x", hash)

		// Setar trust score inicial
		report.TrustScore = 1.0
		report.Status = "ativa"

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		err := app.submitUserReport(ctx, report)
		if err != nil {
			if strings.Contains(err.Error(), "spam") {
				c.JSON(http.StatusTooManyRequests, gin.H{"error": err.Error()})
				return
			}
			log.Printf("Erro ao salvar denúncia: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar denúncia"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"status":  "success",
			"message": "Denúncia registrada com sucesso",
			"report":  report,
		})
	})

	// GET /api/v1/admin/dashboard - Dashboard administrativo (requer JWT + Admin)
	// Temporariamente desativado para focar em telemetria
	/*
		r.GET("/api/v1/admin/dashboard", JWTMiddleware(), AdminMiddleware(), func(c *gin.Context) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			// Obter health do sistema
			dbHealth := "offline"
			redisHealth := "offline"
			systemStatus := "offline"

			if app.db != nil {
				if err := app.db.PingContext(ctx); err == nil {
					dbHealth = "ok"
				}
			}

			if app.rdb != nil {
				if err := app.rdb.Ping(ctx).Err(); err == nil {
					redisHealth = "ok"
				}
			}

			if dbHealth == "ok" && redisHealth == "ok" {
				systemStatus = "online"
			} else if dbHealth == "ok" || redisHealth == "ok" {
				systemStatus = "degraded"
			}

			// Obter trending routes
			trendingRoutes, err := getTrendingRoutes(app)
			if err != nil {
				// Fallback hardcoded
				trendingRoutes = []TrendingRoute{
					{Origin: "Terminal Novo Mundo", Destination: "Campus Samambaia UFG", Count: 0, LastSearch: time.Now()},
					{Origin: "Terminal Bíblia", Destination: "Terminal Canedo", Count: 0, LastSearch: time.Now()},
					{Origin: "Terminal Isidória", Destination: "Terminal Padre Pelágio", Count: 0, LastSearch: time.Now()},
				}
			}

			// Analisar crise nas rotas
			crisisAnalysis, err := app.analyzeRouteCrisis(ctx, trendingRoutes)
			if err != nil {
				log.Printf("Erro ao analisar crise: %v", err)
				crisisAnalysis = []CrisisAnalysis{}
			}

			// Obter denúncias recentes agrupadas por tipo
			var recentReports []struct {
				TipoProblema string `json:"tipo_problema"`
				Count        int    `json:"count"`
				Severity     string `json:"severity"`
			}

			if app.db != nil {
				query := `
				SELECT
					tipo_problema,
					COUNT(*) as count,
					CASE
						WHEN tipo_problema = 'Perigo' THEN 'high'
						WHEN tipo_problema = 'Lotado' THEN 'medium'
						ELSE 'low'
					END as severity
				FROM user_reports
				WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
					AND status = 'ativa'
				GROUP BY tipo_problema
				ORDER BY count DESC
				`

				rows, err := app.db.QueryContext(ctx, query)
				if err == nil {
					defer rows.Close()
					for rows.Next() {
						var report struct {
							TipoProblema string `json:"tipo_problema"`
							Count        int    `json:"count"`
							Severity     string `json:"severity"`
						}
						err := rows.Scan(&report.TipoProblema, &report.Count, &report.Severity)
						if err == nil {
							recentReports = append(recentReports, report)
						}
					}
				}
			}

			// Calcular métricas
			reqCount := atomic.LoadInt64(&app.requestCount)
			errCount := atomic.LoadInt64(&app.errorCount)
			errorRate := float64(0)
			if reqCount > 0 {
				errorRate = float64(errCount) / float64(reqCount) * 100
			}

			// Converter crisis analysis para trending routes com crise
			var trendingWithCrisis []gin.H
			for _, crisis := range crisisAnalysis {
				trendingWithCrisis = append(trendingWithCrisis, gin.H{
					"origin":         crisis.Origin,
					"destination":    crisis.Destination,
					"count":          crisis.AccessCount,
					"last_search":    crisis.LastAccess,
					"crisis_level":   crisis.CrisisLevel,
					"report_count":   crisis.ReportCount,
					"crisis_score":   crisis.CrisisScore,
					"affected_lines": crisis.AffectedLines,
				})
			}

			c.JSON(http.StatusOK, gin.H{
				"system_health": gin.H{
					"status":    systemStatus,
					"database":  dbHealth,
					"redis":     redisHealth,
					"uptime":    int(time.Since(app.startTime).Seconds()),
					"timestamp": time.Now().UTC().Format("2006-01-02T15:04:05Z"),
				},
				"trending_routes": trendingWithCrisis,
				"recent_reports":  recentReports,
				"metrics": gin.H{
					"total_requests": reqCount,
					"error_rate":     errorRate,
					"active_users":   len(trendingRoutes) * 10, // Estimativa
					"cache_hit_rate": 85.5,                     // Estimativa
				},
			})
		})
	*/

	// GET /api/v1/admin/export/csv - Exportar dados em CSV
	r.GET("/api/v1/admin/export/csv", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		var csvContent strings.Builder

		// Header CSV
		csvContent.WriteString("Timestamp,Origin,Destination,AccessCount,ReportCount,CrisisLevel,CrisisScore,AffectedLines\n")

		// Obter trending routes
		trendingRoutes, err := getTrendingRoutes(app)
		if err != nil {
			// Fallback hardcoded
			trendingRoutes = []TrendingRoute{
				{Origin: "Terminal Novo Mundo", Destination: "Campus Samambaia UFG", Count: 0, LastSearch: time.Now()},
			}
		}

		// Analisar crise e exportar dados
		crisisAnalysis, err := app.analyzeRouteCrisis(ctx, trendingRoutes)
		if err != nil {
			log.Printf("Erro ao analisar crise para CSV: %v", err)
		}

		for _, crisis := range crisisAnalysis {
			affectedLines := strings.Join(crisis.AffectedLines, ";")
			csvContent.WriteString(fmt.Sprintf("%s,%s,%s,%d,%d,%s,%.1f,%s\n",
				time.Now().Format("2006-01-02 15:04:05"),
				crisis.Origin,
				crisis.Destination,
				crisis.AccessCount,
				crisis.ReportCount,
				crisis.CrisisLevel,
				crisis.CrisisScore,
				affectedLines,
			))
		}

		// Se não houver dados, exportar dados básicos
		if len(crisisAnalysis) == 0 {
			csvContent.WriteString(fmt.Sprintf("%s,Sem dados,Sem dados,0,0,normal,0.0,M23\n",
				time.Now().Format("2006-01-02 15:04:05")))
		}

		c.Header("Content-Type", "text/csv")
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=transprota-export-%s.csv",
			time.Now().Format("2006-01-02")))
		c.String(http.StatusOK, csvContent.String())
	})

	// GET /api/v1/weather - Obter dados do clima atual
	r.GET("/api/v1/weather", func(c *gin.Context) {
		weather, exists := app.getWeatherData("goiania")
		if !exists {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error":   "Dados do clima indisponíveis",
				"message": "Tente novamente em alguns minutos",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"city":        weather.City,
			"temperature": weather.Temperature,
			"humidity":    weather.Humidity,
			"description": weather.Description,
			"is_raining":  weather.IsRaining,
			"wind_speed":  weather.WindSpeed,
			"visibility":  weather.Visibility,
			"timestamp":   weather.Timestamp,
			"route_impact": gin.H{
				"rain_adjustment": weather.IsRaining,
				"extra_minutes":   app.adjustRouteForWeather(0) - 0,
			},
		})
	})

	// GET /api/v1/walkability - Calcular caminhabilidade para uma distância
	r.GET("/api/v1/walkability", func(c *gin.Context) {
		distanceStr := c.Query("distance")
		if distanceStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Parâmetro 'distance' (em km) é obrigatório"})
			return
		}

		distance, err := strconv.ParseFloat(distanceStr, 64)
		if err != nil || distance < 0 || distance > 50 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Distância inválida. Use valores entre 0 e 50 km"})
			return
		}

		walkability := app.calculateWalkability(distance)

		c.JSON(http.StatusOK, gin.H{
			"walkability": walkability,
			"bus_alternative": gin.H{
				"estimated_time": int(distance * 12), // ~12 min por km de ônibus
				"cost":           4.30,
			},
		})
	})

	// GET /api/v1/route-with-weather - Calcular rota com ajuste climático
	r.GET("/api/v1/route-with-weather", func(c *gin.Context) {
		origem := c.Query("origem")
		destino := c.Query("destino")

		if origem == "" || destino == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "origem e destino são obrigatórios"})
			return
		}

		// Calcular distância (simulação)
		distanceKm := calcularDistanciaEntreLocais(origem, destino)

		// Calcular caminhabilidade
		walkability := app.calculateWalkability(distanceKm)

		// Tempo base de ônibus (simulação)
		baseTime := int(distanceKm * 12) // ~12 min por km

		// Ajustar para clima
		adjustedTime := app.adjustRouteForWeather(baseTime)

		weather, _ := app.getWeatherData("goiania")

		c.JSON(http.StatusOK, gin.H{
			"origin":      origem,
			"destination": destino,
			"distance_km": distanceKm,
			"walkability": walkability,
			"bus_route": gin.H{
				"base_time_minutes":     baseTime,
				"adjusted_time_minutes": adjustedTime,
				"weather_adjustment":    adjustedTime - baseTime,
				"cost":                  4.30,
			},
			"weather": weather,
			"recommendation": func() string {
				if walkability.IsWalkable {
					return "Recomendado caminhar - mais rápido e econômico!"
				}
				return "Use o transporte público - distância muito longa"
			}(),
		})
	})

	// GET /gps/:id - Consultar posição do ônibus (público para passageiros)
	r.GET("/gps/:id", func(c *gin.Context) {
		busID := c.Param("id")
		if err := ValidateString(busID, 50); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bus ID inválido"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		val, err := app.rdb.Get(ctx, "bus:"+busID).Result()
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ônibus offline"})
			return
		}
		if err != nil {
			log.Printf("❌ Erro ao consultar Redis: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao consultar posição"})
			return
		}

		var gpsData GPSData
		if err := json.Unmarshal([]byte(val), &gpsData); err != nil {
			c.JSON(http.StatusOK, gin.H{"bus_id": busID, "posicao": val})
			return
		}

		c.JSON(http.StatusOK, gpsData)
	})

	// POST /gps - Atualizar posição do ônibus (PROTEGIDA)
	authorized := r.Group("/")
	authorized.Use(AuthMiddleware())
	{
		authorized.POST("/gps", func(c *gin.Context) {
			var input GPSData
			if err := c.ShouldBindJSON(&input); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos: bus_id, latitude, longitude obrigatórios"})
				return
			}

			if input.Location.Y < -90 || input.Location.Y > 90 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude deve estar entre -90 e 90"})
				return
			}
			if input.Location.X < -180 || input.Location.X > 180 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Longitude deve estar entre -180 e 180"})
				return
			}

			input.Timestamp = time.Now()

			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			jsonData, err := json.Marshal(input)
			if err != nil {
				log.Printf("❌ Erro ao serializar JSON: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno"})
				return
			}

			if err := app.rdb.Set(ctx, "bus:"+input.BusID, string(jsonData), 10*time.Minute).Err(); err != nil {
				log.Printf("❌ Erro ao salvar no Redis: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar posição"})
				return
			}

			c.JSON(http.StatusOK, gin.H{"status": "Localização atualizada com sucesso"})
		})
	}

	// GET /gps/:id/status - Verificar se próximo de algum terminal
	r.GET("/gps/:id/status", func(c *gin.Context) {
		busID := c.Param("id")
		if busID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Bus ID é obrigatório"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		posicao, err := app.rdb.Get(ctx, "bus:"+busID).Result()
		if err == redis.Nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Ônibus não encontrado"})
			return
		}
		if err != nil {
			log.Printf("❌ Erro ao consultar Redis: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao consultar posição"})
			return
		}

		var gpsData GPSData
		if err := json.Unmarshal([]byte(posicao), &gpsData); err != nil {
			// Fallback para formato antigo "lat,lng"
			var lat, lon float64
			_, err := fmt.Sscanf(posicao, "%f,%f", &lat, &lon)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Formato de posição inválido"})
				return
			}
			gpsData.Location = SpatialPoint{X: lon, Y: lat}
		}

		rows, err := app.db.QueryContext(ctx, "SELECT name, ST_AsText(location) FROM locations")
		if err != nil {
			log.Printf("Erro ao consultar terminais: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao consultar terminais"})
			return
		}
		defer rows.Close()

		status := "Em trânsito"
		terminalProximo := ""
		distanciaMinima := 150.0

		for rows.Next() {
			var name string
			var locationText string
			if err := rows.Scan(&name, &locationText); err != nil {
				log.Printf("Erro ao ler terminal: %v", err)
				continue
			}

			var terminalLocation SpatialPoint
			terminalLocation.Scan(locationText)

			distancia := calcularDistancia(gpsData.Location.Y, gpsData.Location.X, terminalLocation.Y, terminalLocation.X)
			if distancia < distanciaMinima {
				status = "No Terminal"
				terminalProximo = name
				distanciaMinima = distancia
			}
		}

		c.JSON(http.StatusOK, StatusResponse{
			BusID:           busID,
			Status:          status,
			Terminal:        terminalProximo,
			DistanciaMetros: distanciaMinima,
		})
	})

	// Grupo de denúncias (PROTEGIDO)
	denunciasGroup := r.Group("/denuncias")
	denunciasGroup.Use(AuthMiddleware()) // Exigir autenticação para acessar denúncias

	// POST /denuncias - Submeter denúncia colaborativa (PROTEGIDA)
	denunciasGroup.POST("", func(c *gin.Context) {
		var req SubmeterDenunciaRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
			return
		}

		// Validar campos de entrada para evitar injeção
		if err := ValidateString(req.UserID, 255); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "UserID inválido"})
			return
		}
		if err := ValidateString(req.BusLine, 10); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "BusLine inválido"})
			return
		}
		if err := ValidateString(req.BusID, 50); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "BusID inválido"})
			return
		}
		if err := ValidateString(req.EvidenceURL, 2048); err != nil && req.EvidenceURL != "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "EvidenceURL excede limite"})
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Calcular trust score
		trustScore, err := app.calculateTrustScore(ctx, req.UserID)
		if err != nil {
			log.Printf("❌ Erro ao calcular trust score: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro interno"})
			return
		}

		// Inserir denúncia
		location := fmt.Sprintf("POINT(%f %f)", req.Longitude, req.Latitude)
		_, err = app.db.ExecContext(ctx, `
			INSERT INTO denuncias (user_id, bus_line, bus_id, type, location, evidence_url, trust_score)
			VALUES ($1, $2, $3, $4, ST_GeomFromText($5, 4326), $6, $7)
		`, req.UserID, req.BusLine, req.BusID, req.Type, location, req.EvidenceURL, trustScore)
		if err != nil {
			log.Printf("❌ Erro ao inserir denúncia: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao salvar denúncia"})
			return
		}

		c.JSON(http.StatusCreated, gin.H{"status": "Denúncia submetida com sucesso", "trust_score": trustScore})
	})

	// GET /denuncias - Listar denúncias ativas (filtradas por localização opcional, PROTEGIDA)
	denunciasGroup.GET("", func(c *gin.Context) {
		latStr := c.Query("lat")
		lonStr := c.Query("lon")
		radiusStr := c.Query("radius") // em metros, padrão 1000

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var query string
		var args []interface{}
		if latStr != "" && lonStr != "" {
			// Validar coordenadas antes de usar em query
			lat, lon, err := ValidateGeoCoords(latStr, lonStr)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Coordenadas inválidas"})
				return
			}

			radius := 1000.0
			if r, err := strconv.ParseFloat(radiusStr, 64); err == nil && r > 0 && r <= 10000 { // Máximo 10km
				radius = r
			}

			query = `
				SELECT id, user_id, bus_line, bus_id, type, ST_AsText(location), timestamp, evidence_url, trust_score
				FROM denuncias
				WHERE ST_DWithin(location, ST_GeomFromText($1, 4326), $2)
				ORDER BY timestamp DESC
				LIMIT 50
			`
			args = []interface{}{fmt.Sprintf("POINT(%f %f)", lon, lat), radius}
		} else {
			query = `
				SELECT id, user_id, bus_line, bus_id, type, ST_AsText(location), timestamp, evidence_url, trust_score
				FROM denuncias
				WHERE timestamp > NOW() - INTERVAL '1 hour'
				ORDER BY timestamp DESC
				LIMIT 50
			`
		}

		// Nota: GT_DWithin retorna todos os dados. Em produção, máscara user_id para privacidade.

		rows, err := app.db.QueryContext(ctx, query, args...)
		if err != nil {
			log.Printf("❌ Erro ao consultar denúncias: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao consultar denúncias"})
			return
		}
		defer rows.Close()

		var denuncias []Denuncia
		for rows.Next() {
			d := denunciaPool.Get().(*Denuncia)
			var location string
			if err := rows.Scan(&d.ID, &d.UserID, &d.BusLine, &d.BusID, &d.Type, &location, &d.Timestamp, &d.EvidenceURL, &d.TrustScore); err != nil {
				log.Printf("❌ Erro ao ler denúncia: %v", err)
				denunciaPool.Put(d) // Devolver ao pool
				continue
			}
			d.Location = location
			denuncias = append(denuncias, *d)
			denunciaPool.Put(d) // Devolver ao pool após uso
		}

		c.JSON(http.StatusOK, ListaDenunciasResponse{
			Total:     len(denuncias),
			Denuncias: denuncias,
		})
	})

	// ==========================================================================
	// FASE 1: FOUNDATION - TELEMETRY SAAS (Crowdsourcing GPS)
	// ==========================================================================

	// Configurar rotas de telemetria (módulo isolado)
	telemetry.SetupRoutes(r, app.db, app.rdb)

}

// newApp inicializa a aplicação com bancos de dados e cache.
func newApp() (*App, error) {
	app := &App{
		routeTTL:     15 * time.Minute,
		startTime:    time.Now(),
		weatherCache: make(map[string]WeatherData),
	}
	if err := app.initBancos(); err != nil {
		return nil, err
	}

	// Inicializar serviço de clima em background
	go app.startWeatherService()

	return app, nil
}

// Close fecha as conexões com bancos de dados e cache.
func (a *App) Close() {
	if a.db != nil {
		a.db.Close()
	}
	if a.rdb != nil {
		a.rdb.Close()
	}
}

// startWeatherService inicia o serviço de monitoramento do clima
func (a *App) startWeatherService() {
	ticker := time.NewTicker(10 * time.Minute) // Atualizar a cada 10 minutos
	defer ticker.Stop()

	// Buscar dados iniciais
	a.fetchWeatherData()

	for range ticker.C {
		a.fetchWeatherData()
	}
}

// fetchWeatherData busca dados do clima da OpenWeather API
func (a *App) fetchWeatherData() {
	apiKey := os.Getenv("OPENWEATHER_API_KEY")
	if apiKey == "" {
		log.Println("OpenWeather API key não configurada, usando fallback")
		a.setFallbackWeather()
		return
	}

	// Coordenadas de Goiânia
	lat, lon := "-16.6864", "-49.2643"
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/weather?lat=%s&lon=%s&appid=%s&units=metric&lang=pt_br", lat, lon, apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		log.Printf("Erro ao buscar dados do clima: %v", err)
		a.setFallbackWeather()
		return
	}
	defer resp.Body.Close()

	var weatherData struct {
		Name string `json:"name"`
		Main struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
		} `json:"main"`
		Weather []struct {
			Description string `json:"description"`
			Main        string `json:"main"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"`
		} `json:"wind"`
		Visibility int `json:"visibility"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&weatherData); err != nil {
		log.Printf("Erro ao decodificar dados do clima: %v", err)
		a.setFallbackWeather()
		return
	}

	// Determinar se está chovendo
	isRaining := false
	for _, w := range weatherData.Weather {
		if w.Main == "Rain" || w.Main == "Drizzle" || w.Main == "Thunderstorm" {
			isRaining = true
			break
		}
	}

	weather := WeatherData{
		City:        weatherData.Name,
		Temperature: weatherData.Main.Temp,
		Humidity:    weatherData.Main.Humidity,
		Description: weatherData.Weather[0].Description,
		IsRaining:   isRaining,
		WindSpeed:   weatherData.Wind.Speed,
		Visibility:  weatherData.Visibility,
		Timestamp:   time.Now(),
	}

	// Atualizar cache
	a.weatherMutex.Lock()
	a.weatherCache["goiania"] = weather
	a.weatherMutex.Unlock()

	log.Printf("Clima atualizado: %s, %.1f°C, %s (Chovendo: %v)",
		weather.City, weather.Temperature, weather.Description, weather.IsRaining)
}

// setFallbackWeather define dados de clima fallback quando API falha
func (a *App) setFallbackWeather() {
	weather := WeatherData{
		City:        "Goiânia",
		Temperature: 25.0,
		Humidity:    60,
		Description: "céu limpo",
		IsRaining:   false,
		WindSpeed:   5.0,
		Visibility:  10000,
		Timestamp:   time.Now(),
	}

	a.weatherMutex.Lock()
	a.weatherCache["goiania"] = weather
	a.weatherMutex.Unlock()

	log.Println("Usando dados de clima fallback")
}

// getWeatherData retorna dados do clima cacheados
func (a *App) getWeatherData(city string) (WeatherData, bool) {
	a.weatherMutex.RLock()
	defer a.weatherMutex.RUnlock()

	weather, exists := a.weatherCache[strings.ToLower(city)]
	if !exists {
		return WeatherData{}, false
	}

	// Verificar se dados são recentes (menos de 15 minutos)
	if time.Since(weather.Timestamp) > 15*time.Minute {
		return WeatherData{}, false
	}

	return weather, true
}

// adjustRouteForWeather ajusta tempo de rota baseado no clima
func (a *App) adjustRouteForWeather(baseTimeMinutes int) int {
	weather, exists := a.getWeatherData("goiania")
	if !exists {
		return baseTimeMinutes
	}

	// Se estiver chovendo, adicionar 10 minutos
	if weather.IsRaining {
		adjustedTime := baseTimeMinutes + 10
		log.Printf("Ajuste de rota por chuva: %d -> %d minutos", baseTimeMinutes, adjustedTime)
		return adjustedTime
	}

	// Se vento forte (>20 km/h), adicionar 5 minutos
	if weather.WindSpeed > 5.5 { // 20 km/h = 5.5 m/s
		adjustedTime := baseTimeMinutes + 5
		log.Printf("Ajuste de rota por vento forte: %d -> %d minutos", baseTimeMinutes, adjustedTime)
		return adjustedTime
	}

	return baseTimeMinutes
}

// calcularDistanciaEntreLocais calcula distância aproximada entre dois locais de Goiânia
func calcularDistanciaEntreLocais(origem, destino string) float64 {
	// Coordenadas aproximadas dos principais locais de Goiânia
	locais := map[string]struct {
		lat float64
		lng float64
	}{
		"setor bueno":            {lat: -16.6864, lng: -49.2643},
		"setor centro":           {lat: -16.6807, lng: -49.2671},
		"setor oeste":            {lat: -16.6820, lng: -49.2750},
		"setor norte":            {lat: -16.6700, lng: -49.2600},
		"setor sul":              {lat: -16.7000, lng: -49.2700},
		"setor leste":            {lat: -16.6750, lng: -49.2500},
		"campus samambaia":       {lat: -16.6830, lng: -49.2670},
		"terminal novo mundo":    {lat: -16.6860, lng: -49.2640},
		"terminal bíblia":        {lat: -16.6900, lng: -49.2800},
		"terminal canedo":        {lat: -16.6700, lng: -49.3000},
		"terminal isidória":      {lat: -16.6600, lng: -49.2400},
		"terminal padre pelágio": {lat: -16.7000, lng: -49.2900},
	}

	origemNorm := strings.ToLower(strings.TrimSpace(origem))
	destinoNorm := strings.ToLower(strings.TrimSpace(destino))

	// Se forem o mesmo local, distância 0
	if origemNorm == destinoNorm {
		return 0
	}

	// Buscar coordenadas
	origCoords, origExists := locais[origemNorm]
	destCoords, destExists := locais[destinoNorm]

	// Se não encontrar, usar coordenadas padrão
	if !origExists {
		origCoords = struct {
			lat float64
			lng float64
		}{lat: -16.6860, lng: -49.2640}
	}
	if !destExists {
		destCoords = struct {
			lat float64
			lng float64
		}{lat: -16.6830, lng: -49.2670}
	}

	// Calcular distância usando Haversine
	return calcularDistancia(origCoords.lat, origCoords.lng, destCoords.lat, destCoords.lng)
}

// calculateWalkability calcula sugestão de caminhabilidade
func (a *App) calculateWalkability(distanceKm float64) WalkabilitySuggestion {
	// Velocidade média de caminhada: 5 km/h
	walkTimeMin := int(distanceKm * 60 / 5)

	// Custo médio da passagem: R$ 4,30
	moneySaved := 4.30

	// Tempo de exercício = tempo de caminhada
	exerciseMin := walkTimeMin

	// Calorias queimadas: 50 kcal por km (base para pessoa média)
	caloriesBurned := int(distanceKm * 50)

	// Verificar condições do clima
	weather, exists := a.getWeatherData("goiania")
	weatherFactor := "favorável"

	isWalkable := distanceKm <= 2.0 // Limite de 2km

	if exists {
		// Ajustar basedo no clima
		if weather.IsRaining {
			isWalkable = false
			weatherFactor = "chuva forte"
		} else if weather.Temperature > 35 {
			isWalkable = distanceKm <= 1.5 // Reduzir limite se muito calor
			weatherFactor = "muito calor"
		} else if weather.WindSpeed > 10 { // >36 km/h
			isWalkable = false
			weatherFactor = "vento muito forte"
		}
	}

	var recommendation string
	if isWalkable {
		recommendation = fmt.Sprintf("Que tal ir a pé? Você economiza R$ %.2f, ganha %d min de exercício e queime %d calorias! Clima %s.",
			moneySaved, exerciseMin, caloriesBurned, weatherFactor)
	} else {
		recommendation = "Distância muito longa ou condições climáticas desfavoráveis. Recomendamos usar o transporte público."
	}

	return WalkabilitySuggestion{
		IsWalkable:     isWalkable,
		DistanceKm:     distanceKm,
		WalkTimeMin:    walkTimeMin,
		MoneySaved:     moneySaved,
		ExerciseMin:    exerciseMin,
		CaloriesBurned: caloriesBurned,
		Recommendation: recommendation,
		WeatherFactor:  weatherFactor,
	}
}

// getAppMode retorna o modo de operação da aplicação
func getAppMode(app *App) string {
	if app.db == nil && app.rdb == nil {
		return "OFFLINE (modo degradado)"
	}
	if app.db == nil || app.rdb == nil {
		return "DEGRADED (banco parcialmente offline)"
	}
	return "FULL (todos os sistemas operacionais)"
}

// PostGIS Functions para queries espaciais otimizadas

// getTrendingRoutesPostGIS usa função otimizada do banco
func (a *App) getTrendingRoutesPostGIS(ctx context.Context, days int, limit int) ([]TrendingRoute, error) {
	query := `
		SELECT origin, destination, COUNT(*) as search_count, MAX(search_time) as last_search
		FROM route_searches
		WHERE search_time >= CURRENT_TIMESTAMP - INTERVAL '%d days'
		GROUP BY origin, destination
		ORDER BY search_count DESC, last_search DESC
		LIMIT %d
	`

	rows, err := a.db.QueryContext(ctx, fmt.Sprintf(query, days, limit))
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar trending routes: %v", err)
	}
	defer rows.Close()

	var trending []TrendingRoute
	for rows.Next() {
		var route TrendingRoute
		if err := rows.Scan(&route.Origin, &route.Destination, &route.Count, &route.LastSearch); err != nil {
			continue
		}
		trending = append(trending, route)
	}

	return trending, nil
}

// findRoutesNearPoint busca rotas que passam perto de um ponto específico
func (a *App) findRoutesNearPoint(ctx context.Context, center SpatialPoint, radiusMeters float64, limit int) ([]TrendingRoute, error) {
	query := `
		SELECT DISTINCT origin, destination, COUNT(*) as search_count, MAX(search_time) as last_search
		FROM route_searches
		WHERE ST_DWithin(origin_location, ST_GeomFromText($1, 4326), $2)
		   OR ST_DWithin(destination_location, ST_GeomFromText($1, 4326), $2)
		GROUP BY origin, destination
		ORDER BY search_count DESC, last_search DESC
		LIMIT $3
	`

	wkt := fmt.Sprintf("POINT(%f %f)", center.X, center.Y)

	rows, err := a.db.QueryContext(ctx, query, wkt, radiusMeters, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar rotas próximas: %v", err)
	}
	defer rows.Close()

	var routes []TrendingRoute
	for rows.Next() {
		var route TrendingRoute
		if err := rows.Scan(&route.Origin, &route.Destination, &route.Count, &route.LastSearch); err != nil {
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

// findRoutesInArea busca rotas dentro de uma área retangular
func (a *App) findRoutesInArea(ctx context.Context, minLat, minLng, maxLat, maxLng float64, limit int) ([]TrendingRoute, error) {
	query := `
		SELECT DISTINCT origin, destination, COUNT(*) as search_count, MAX(search_time) as last_search
		FROM route_searches
		WHERE (origin_location IS NOT NULL AND destination_location IS NOT NULL)
		  AND (
			(ST_X(origin_location) BETWEEN $1 AND $2 AND ST_Y(origin_location) BETWEEN $3 AND $4) OR
			(ST_X(destination_location) BETWEEN $1 AND $2 AND ST_Y(destination_location) BETWEEN $3 AND $4)
		  )
		GROUP BY origin, destination
		ORDER BY search_count DESC, last_search DESC
		LIMIT $5
	`

	rows, err := a.db.QueryContext(ctx, query, minLng, maxLng, minLat, maxLat, limit)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar rotas na área: %v", err)
	}
	defer rows.Close()

	var routes []TrendingRoute
	for rows.Next() {
		var route TrendingRoute
		if err := rows.Scan(&route.Origin, &route.Destination, &route.Count, &route.LastSearch); err != nil {
			continue
		}
		routes = append(routes, route)
	}

	return routes, nil
}

// getRouteDensity calcula densidade de rotas por área
func (a *App) getRouteDensity(ctx context.Context, center SpatialPoint, radiusMeters float64) (int, error) {
	query := `
		SELECT COUNT(DISTINCT CONCAT(origin, '-', destination)) as route_count
		FROM route_searches
		WHERE ST_DWithin(origin_location, ST_GeomFromText($1, 4326), $2)
		   OR ST_DWithin(destination_location, ST_GeomFromText($1, 4326), $2)
	`

	wkt := fmt.Sprintf("POINT(%f %f)", center.X, center.Y)

	var count int
	err := a.db.QueryRowContext(ctx, query, wkt, radiusMeters).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("erro ao calcular densidade de rotas: %v", err)
	}

	return count, nil
}

type lineSegment struct {
	numeroLinha string
	nomeLinha   string
	paradas     []string
	ordem       map[string]int
	tempo       map[string]int
	integration map[string]bool
}

func (a *App) cacheKey(origem, destino string) string {
	return fmt.Sprintf("rota:%s:%s", normalizeParam(origem), normalizeParam(destino))
}

func normalizeParam(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func lookupSegmentField[T any](m map[string]T, key string) (T, bool) {
	norm := normalizeParam(key)
	if val, ok := m[norm]; ok {
		return val, true
	}
	for k, val := range m {
		if normalizeParam(k) == norm {
			return val, true
		}
	}
	var zero T
	return zero, false
}

func lookupOrder(seg *lineSegment, parada string) (int, bool) {
	return lookupSegmentField(seg.ordem, parada)
}

func lookupTime(seg *lineSegment, parada string) (int, bool) {
	return lookupSegmentField(seg.tempo, parada)
}

func lookupIntegration(seg *lineSegment, parada string) bool {
	val, _ := lookupSegmentField(seg.integration, parada)
	return val
}

// getRouteCache recupera uma rota do cache Redis.
func (a *App) getRouteCache(ctx context.Context, origem, destino string) (*RouteResponse, error) {
	val, err := a.rdb.Get(ctx, a.cacheKey(origem, destino)).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var route RouteResponse
	if err := json.Unmarshal([]byte(val), &route); err != nil {
		return nil, err
	}
	route.Cached = true
	return &route, nil
}

// setRouteCache armazena uma rota no cache Redis com TTL.
func (a *App) setRouteCache(ctx context.Context, origem, destino string, route *RouteResponse) error {
	jsonData, err := json.Marshal(route)
	if err != nil {
		return err
	}
	return a.rdb.Set(ctx, a.cacheKey(origem, destino), string(jsonData), a.routeTTL).Err()
}

// calculateRoute calcula a melhor rota entre origem e destino, priorizando rotas diretas e integrações.
func (a *App) calculateRoute(ctx context.Context, origem, destino string) (*RouteResponse, error) {
	query := `
SELECT l.numero_linha, l.nome_linha, p.nome as parada, i.ordem_parada, i.tempo_estimado_anterior_minutos, i.eh_ponto_integracao
FROM linhas_onibus l
JOIN itinerarios i ON l.id = i.linha_id
JOIN pontos_parada p ON i.parada_id = p.id
WHERE l.status = 'ativa'
ORDER BY l.numero_linha, i.ordem_parada;
`

	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lines := map[string]*lineSegment{}
	for rows.Next() {
		var numero, nome, parada string
		var ordem int
		var tempo sql.NullInt64
		var ehIntegracao bool
		if err := rows.Scan(&numero, &nome, &parada, &ordem, &tempo, &ehIntegracao); err != nil {
			continue
		}
		seg, ok := lines[numero]
		if !ok {
			seg = &lineSegment{numeroLinha: numero, nomeLinha: nome, ordem: map[string]int{}, tempo: map[string]int{}, integration: map[string]bool{}}
			lines[numero] = seg
		}
		seg.paradas = append(seg.paradas, parada)
		normParada := normalizeParam(parada)
		seg.ordem[normParada] = ordem
		if tempo.Valid {
			seg.tempo[normParada] = int(tempo.Int64)
		}
		if ehIntegracao {
			seg.integration[normParada] = true
		}
	}

	origemNorm := normalizeParam(origem)
	destinoNorm := normalizeParam(destino)

	directRoute, directTime := findDirectRoute(lines, origemNorm, destinoNorm)
	transferRoute, transferScore := findBestTransferRoute(lines, origemNorm, destinoNorm)

	if directRoute != nil {
		if transferRoute != nil && transferScore+5 < directTime {
			return transferRoute, nil
		}
		return directRoute, nil
	}

	return transferRoute, nil
}

func findDirectRoute(lines map[string]*lineSegment, origemNorm, destinoNorm string) (*RouteResponse, int) {
	bestTime := int(^uint(0) >> 1)
	var bestRoute *RouteResponse

	for _, seg := range lines {
		start, ok1 := lookupOrder(seg, origemNorm)
		end, ok2 := lookupOrder(seg, destinoNorm)
		if !ok1 || !ok2 || start >= end {
			continue
		}

		tempoTotal := 0
		for _, parada := range seg.paradas {
			ord, ok := lookupOrder(seg, parada)
			if !ok {
				continue
			}
			if ord > start && ord <= end {
				if tempo, ok := lookupTime(seg, parada); ok {
					tempoTotal += tempo
				}
			}
		}

		if tempoTotal < bestTime {
			bestTime = tempoTotal
			bestRoute = &RouteResponse{
				Origem:  origemNorm,
				Destino: destinoNorm,
				Tipo:    "direta",
				Steps: []RouteStep{{
					NumeroLinha:       seg.numeroLinha,
					NomeLinha:         seg.nomeLinha,
					Paradas:           seg.paradas[start-1 : end],
					TempoTotalMinutos: tempoTotal,
				}},
			}
		}
	}

	if bestRoute == nil {
		return nil, 0
	}
	return bestRoute, bestTime
}

func findBestTransferRoute(lines map[string]*lineSegment, origemNorm, destinoNorm string) (*RouteResponse, int) {
	bestScore := int(^uint(0) >> 1)
	var bestRoute *RouteResponse

	for _, segA := range lines {
		startA, okA := lookupOrder(segA, origemNorm)
		if !okA {
			continue
		}

		for _, segB := range lines {
			endB, okB := lookupOrder(segB, destinoNorm)
			if !okB {
				continue
			}

			for pivot, ordemA := range segA.ordem {
				normPivot := normalizeParam(pivot)
				if normPivot == origemNorm || normPivot == destinoNorm {
					continue
				}

				ordemB, okPivot := lookupOrder(segB, pivot)
				if !okPivot || ordemA <= startA || ordemB >= endB {
					continue
				}

				timeA := sumTempo(segA, startA, ordemA)
				timeB := sumTempo(segB, ordemB, endB)
				totalTime := timeA + timeB

				bonus := integrationBonus(pivot)
				penalty := 0
				if !lookupIntegration(segA, pivot) && !lookupIntegration(segB, pivot) {
					penalty += 12
				}

				score := totalTime + penalty - bonus
				if score < bestScore {
					bestScore = score
					bestRoute = &RouteResponse{
						Origem:  origemNorm,
						Destino: destinoNorm,
						Tipo:    "com_transferencia",
						Steps: []RouteStep{
							{
								NumeroLinha:       segA.numeroLinha,
								NomeLinha:         segA.nomeLinha,
								Paradas:           segA.paradas[startA-1 : ordemA],
								TempoTotalMinutos: timeA,
							},
							{
								NumeroLinha:       segB.numeroLinha,
								NomeLinha:         segB.nomeLinha,
								Paradas:           segB.paradas[ordemB-1 : endB],
								TempoTotalMinutos: timeB,
							},
						},
					}
				}
			}
		}
	}

	if bestRoute == nil {
		return nil, 0
	}

	return bestRoute, bestScore
}

func integrationBonus(parada string) int {
	normalized := strings.ToLower(strings.TrimSpace(parada))

	if strings.Contains(normalized, "novo mundo") || strings.Contains(normalized, "bíblia") || strings.Contains(normalized, "praça a") || strings.Contains(normalized, "terminal centro") {
		return 15
	}
	if strings.Contains(normalized, "terminal") || strings.Contains(normalized, "praça") {
		return 8
	}
	return 0
}

func sumTempo(seg *lineSegment, start, end int) int {
	tempo := 0
	for _, parada := range seg.paradas {
		ord, ok := lookupOrder(seg, parada)
		if !ok {
			continue
		}
		if ord > start && ord <= end {
			if t, ok := lookupTime(seg, parada); ok {
				tempo += t
			}
		}
	}
	return tempo
}

func (a *App) initBancos() error {
	var err error

	// Conectar ao PostgreSQL
	pStr := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable host=%s port=%s",
		os.Getenv("DB_USER"), os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"), os.Getenv("DB_HOST"), os.Getenv("DB_PORT"))

	a.db, err = sql.Open("postgres", pStr)
	if err != nil {
		return fmt.Errorf("falha ao abrir conexão PostgreSQL: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.db.PingContext(ctx); err != nil {
		return fmt.Errorf("falha ao conectar no PostgreSQL: %w", err)
	}

	a.db.SetMaxOpenConns(25)
	a.db.SetMaxIdleConns(5)
	a.db.SetConnMaxLifetime(5 * time.Minute)

	_, err = a.db.ExecContext(ctx, `ALTER TABLE locations ADD CONSTRAINT unique_terminal_name UNIQUE (name) ON CONFLICT DO NOTHING;`)
	if err != nil {
		log.Printf("⚠️ Erro ao criar constraint (pode já existir): %v", err)
	}

	// Conectar ao Redis
	a.rdb = redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
	})

	if err := a.rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("falha ao conectar no Redis: %w", err)
	}

	fmt.Println("✅ Bancos conectados com sucesso!")
	return nil
}

// calcularDistancia calcula a distância entre dois pontos usando a fórmula de Haversine.
func calcularDistancia(lat1, lon1, lat2, lon2 float64) float64 {
	const raioTerraMetros = 6371000 // Raio da Terra em metros

	// Converter graus para radianos
	radLat1 := toRadians(lat1)
	radLat2 := toRadians(lat2)
	diffLat := toRadians(lat2 - lat1)
	diffLon := toRadians(lon2 - lon1)

	// Fórmula de Haversine
	a := math.Sin(diffLat/2)*math.Sin(diffLat/2) +
		math.Cos(radLat1)*math.Cos(radLat2)*
			math.Sin(diffLon/2)*math.Sin(diffLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return raioTerraMetros * c
}

// toRadians converte graus para radianos.
func toRadians(graus float64) float64 {
	return graus * math.Pi / 180.0
}

// UserReport representa uma denúncia de usuário
type UserReport struct {
	ID           int     `json:"id"`
	TipoProblema string  `json:"tipo_problema"` // Lotado, Atrasado, Perigo
	Descricao    string  `json:"descricao"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	BusLine      string  `json:"bus_line"`
	UserIPHash   string  `json:"user_ip_hash"`
	TrustScore   float64 `json:"trust_score"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at"`
}

// HeatmapData representa dados para o mapa de calor
type HeatmapData struct {
	BusLine       string  `json:"bus_line"`
	TipoProblema  string  `json:"tipo_problema"`
	ReportCount   int     `json:"report_count"`
	AvgTrustScore float64 `json:"avg_trust_score"`
	CentroidLat   float64 `json:"centroid_lat"`
	CentroidLng   float64 `json:"centroid_lng"`
	Severity      string  `json:"severity"` // baixa, media, alta
}

// getHeatmapData busca dados para o mapa de calor
func (a *App) getHeatmapData(ctx context.Context) ([]HeatmapData, error) {
	query := `
	SELECT 
		bus_line,
		tipo_problema,
		COUNT(*) as report_count,
		AVG(trust_score) as avg_trust_score,
		ST_X(ST_Centroid(ST_Collect(report_location))) as centroid_lng,
		ST_Y(ST_Centroid(ST_Collect(report_location))) as centroid_lat
	FROM user_reports 
	WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '1 hour'
		AND status = 'ativa'
	GROUP BY bus_line, tipo_problema, DATE_TRUNC('minute', created_at)
	HAVING COUNT(*) >= 3
	ORDER BY report_count DESC
	`

	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar heatmap data: %w", err)
	}
	defer rows.Close()

	var heatmapData []HeatmapData
	for rows.Next() {
		var data HeatmapData
		err := rows.Scan(
			&data.BusLine,
			&data.TipoProblema,
			&data.ReportCount,
			&data.AvgTrustScore,
			&data.CentroidLng,
			&data.CentroidLat,
		)
		if err != nil {
			continue
		}

		// Calcular severidade baseada no número de denúncias e trust score
		if data.ReportCount >= 10 && data.AvgTrustScore >= 0.8 {
			data.Severity = "alta"
		} else if data.ReportCount >= 5 && data.AvgTrustScore >= 0.6 {
			data.Severity = "media"
		} else {
			data.Severity = "baixa"
		}

		heatmapData = append(heatmapData, data)
	}

	return heatmapData, nil
}

// submitUserReport envia uma denúncia de usuário
func (a *App) submitUserReport(ctx context.Context, report UserReport) error {
	// Verificar anti-spam via Redis
	spamKey := fmt.Sprintf("spam:%s", report.UserIPHash)
	exists, err := a.rdb.Exists(ctx, spamKey).Result()
	if err != nil {
		return fmt.Errorf("erro ao verificar spam: %w", err)
	}
	if exists > 0 {
		return fmt.Errorf("usuário já enviou denúncia recentemente. Aguarde 5 minutos.")
	}

	// Inserir denúncia no PostGIS
	query := `
	INSERT INTO user_reports (tipo_problema, descricao, report_location, bus_line, user_ip_hash, trust_score)
	VALUES ($1, $2, ST_GeomFromText($3, 4326), $4, $5, $6)
	`

	_, err = a.db.ExecContext(ctx, query,
		report.TipoProblema,
		report.Descricao,
		fmt.Sprintf("POINT(%f %f)", report.Longitude, report.Latitude),
		report.BusLine,
		report.UserIPHash,
		report.TrustScore,
	)
	if err != nil {
		return fmt.Errorf("erro ao salvar denúncia: %w", err)
	}

	// Setar cooldown de 5 minutos no Redis
	err = a.rdb.Set(ctx, spamKey, "1", 5*time.Minute).Err()
	if err != nil {
		// Log erro mas não falhar operação
		log.Printf("Erro ao setar spam cooldown: %v", err)
	}

	return nil
}

// getRecentReports busca denúncias recentes para exibição no mapa
func (a *App) getRecentReports(ctx context.Context) ([]UserReport, error) {
	query := `
	SELECT 
		id,
		tipo_problema,
		descricao,
		ST_Y(report_location) as latitude,
		ST_X(report_location) as longitude,
		bus_line,
		user_ip_hash,
		trust_score,
		status,
		created_at
	FROM user_reports 
	WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '2 hours'
		AND status = 'ativa'
	ORDER BY created_at DESC
	LIMIT 50
	`

	rows, err := a.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar denúncias recentes: %w", err)
	}
	defer rows.Close()

	var reports []UserReport
	for rows.Next() {
		var report UserReport
		err := rows.Scan(
			&report.ID,
			&report.TipoProblema,
			&report.Descricao,
			&report.Latitude,
			&report.Longitude,
			&report.BusLine,
			&report.UserIPHash,
			&report.TrustScore,
			&report.Status,
			&report.CreatedAt,
		)
		if err != nil {
			continue
		}
		reports = append(reports, report)
	}

	return reports, nil
}

// CrisisAnalysis representa análise de crise para uma rota
type CrisisAnalysis struct {
	Origin              string   `json:"origin"`
	Destination         string   `json:"destination"`
	AccessCount         int64    `json:"access_count"`
	ReportCount         int      `json:"report_count"`
	CrisisScore         float64  `json:"crisis_score"`
	CrisisLevel         string   `json:"crisis_level"` // normal, warning, critical
	LastAccess          string   `json:"last_access"`
	AffectedLines       []string `json:"affected_lines"`
	HighSeverityReports int      `json:"high_severity_reports"`
}

// analyzeRouteCrisis cruza dados de trending routes com denúncias usando PostGIS
func (a *App) analyzeRouteCrisis(ctx context.Context, trendingRoutes []TrendingRoute) ([]CrisisAnalysis, error) {
	if a.db == nil {
		// Fallback quando banco offline
		var fallback []CrisisAnalysis
		for _, route := range trendingRoutes {
			analysis := CrisisAnalysis{
				Origin:              route.Origin,
				Destination:         route.Destination,
				AccessCount:         route.Count,
				ReportCount:         0,
				CrisisScore:         0.0,
				CrisisLevel:         "normal",
				LastAccess:          route.LastSearch.Format("2006-01-02T15:04:05Z"),
				AffectedLines:       []string{"M23"},
				HighSeverityReports: 0,
			}
			fallback = append(fallback, analysis)
		}
		return fallback, nil
	}

	var crisisAnalysis []CrisisAnalysis

	for _, route := range trendingRoutes {
		// Query PostGIS para contar denúncias na área da rota
		query := `
		SELECT 
			COUNT(*) as total_reports,
			COUNT(CASE WHEN tipo_problema = 'Perigo' THEN 1 END) as high_severity,
			ARRAY_AGG(DISTINCT bus_line) as affected_lines
		FROM user_reports 
		WHERE created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
			AND status = 'ativa'
			AND bus_line IS NOT NULL
			AND (
				-- Buscar denúncias em linhas que conectam origem/destino
				bus_line IN (
					SELECT DISTINCT numero_linha 
					FROM linhas_onibus 
					WHERE (
						SELECT COUNT(*) 
						FROM paradas p1, paradas p2 
						WHERE p1.nome_linha = linhas_onibus.numero_linha 
							AND p2.nome_linha = linhas_onibus.numero_linha
							AND (
								(p1.nome ILIKE $1 AND p2.nome ILIKE $2) OR
								(p1.nome ILIKE $2 AND p2.nome ILIKE $1)
							)
					) > 0
				)
				-- Ou denúncias próximas geograficamente
				OR ST_DWithin(
					report_location::geography,
					(
						SELECT ST_Centroid(ST_Collect(location::geography))
						FROM locations 
						WHERE nome ILIKE $1 OR nome ILIKE $2
					),
					2000 -- 2km de raio
				)
			)
		`

		var totalReports, highSeverity int
		var affectedLines []string
		var affectedLinesStr string

		err := a.db.QueryRowContext(ctx, query, route.Origin, route.Destination).Scan(
			&totalReports, &highSeverity, &affectedLinesStr,
		)
		if err != nil {
			log.Printf("Erro ao analisar crise para rota %s->%s: %v", route.Origin, route.Destination, err)
			continue
		}

		// Parse array de linhas afetadas
		if affectedLinesStr != "" && affectedLinesStr != "{}" {
			// Remove {} e split por vírgula
			cleanStr := strings.Trim(affectedLinesStr, "{}")
			if cleanStr != "" {
				affectedLines = strings.Split(cleanStr, ",")
			}
		}

		// Calcular score de crise (0-100)
		var crisisScore float64
		if totalReports > 0 {
			// Fórmula: (denúncias * peso) + (acessos * peso) + (severidade alta * peso)
			reportWeight := 2.0
			accessWeight := 0.1
			severityWeight := 5.0

			crisisScore = float64(totalReports)*reportWeight +
				float64(route.Count)*accessWeight +
				float64(highSeverity)*severityWeight

			// Normalizar para 0-100
			if crisisScore > 100 {
				crisisScore = 100
			}
		}

		// Determinar nível de crise
		var crisisLevel string
		switch {
		case crisisScore >= 70 || highSeverity >= 3:
			crisisLevel = "critical"
		case crisisScore >= 40 || highSeverity >= 1:
			crisisLevel = "warning"
		default:
			crisisLevel = "normal"
		}

		analysis := CrisisAnalysis{
			Origin:              route.Origin,
			Destination:         route.Destination,
			AccessCount:         route.Count,
			ReportCount:         totalReports,
			CrisisScore:         crisisScore,
			CrisisLevel:         crisisLevel,
			LastAccess:          route.LastSearch.Format("2006-01-02T15:04:05Z"),
			AffectedLines:       affectedLines,
			HighSeverityReports: highSeverity,
		}

		crisisAnalysis = append(crisisAnalysis, analysis)
	}

	// Ordenar por score de crise (maior primeiro)
	sort.Slice(crisisAnalysis, func(i, j int) bool {
		return crisisAnalysis[i].CrisisScore > crisisAnalysis[j].CrisisScore
	})

	return crisisAnalysis, nil
}

// calculateTrustScore calcula o trust score baseado em regras de negócio.
func (a *App) calculateTrustScore(ctx context.Context, userID string) (int, error) {
	// Verificar se banco está disponível (fail-soft)
	if a.db == nil {
		return 50, nil // Score padrão se DB não disponível
	}

	ch := make(chan int, 1)
	go func() {
		// Query para contar denúncias com evidências e trust scores altos
		query := `
SELECT 
    COUNT(*) as total_denuncias,
    COUNT(CASE WHEN evidence_url IS NOT NULL THEN 1 END) as com_evidencia,
    AVG(trust_score) as media_trust
FROM denuncias 
WHERE user_id = $1 AND timestamp > NOW() - INTERVAL '30 days'
`
		var total, comEvidencia int
		var mediaTrust sql.NullFloat64
		err := a.db.QueryRowContext(ctx, query, userID).Scan(&total, &comEvidencia, &mediaTrust)
		if err != nil {
			ch <- 50 // Score padrão se erro
			return
		}

		score := 50               // Inicial
		score += total * 2        // +2 por denúncia
		score += comEvidencia * 5 // +5 por evidência
		if mediaTrust.Valid && mediaTrust.Float64 > 70 {
			score += 10 // Bônus por alta confiança média
		}

		if score < 0 {
			score = 0
		}
		if score > 100 {
			score = 100
		}
		ch <- score
	}()

	select {
	case score := <-ch:
		return score, nil
	case <-ctx.Done():
		return 50, ctx.Err() // Score padrão
	}
}

// getTrustLevel retorna o nível baseado no score.
func getTrustLevel(score int) string {
	if score <= 20 {
		return "Suspeito"
	}
	if score <= 80 {
		return "Cidadão"
	}
	return "Fiscal da Galera"
}

// sanitizeInput limpa e sanitiza inputs do usuário
func sanitizeInput(input string) string {
	// Remover caracteres perigosos
	input = strings.TrimSpace(input)

	// Remover caracteres especiais perigosos
	dangerousChars := []string{"<", ">", "&", "\"", "'", "/", "\\", "(", ")", "{", "}", "[", "]", ";", ":", "$", "`"}
	for _, char := range dangerousChars {
		input = strings.ReplaceAll(input, char, "")
	}

	// Limitar tamanho
	if len(input) > 100 {
		input = input[:100]
	}

	return input
}

// isWithinGoiania verifica se o local está dentro da área de cobertura de Goiânia
func isWithinGoiania(location string) bool {
	// Lista de bairros/setores conhecidos de Goiânia + Região Metropolitana
	goianiaLocations := map[string]bool{
		// Goiânia - Setores e Bairros
		"setor bueno": true, "setor centro": true, "setor oeste": true, "setor norte": true,
		"setor sul": true, "setor leste": true, "campus samambaia": true, "uFG": true,
		"terminal centro": true, "terminal samambaia": true, "terminal oeste": true,
		"vila nova": true, "vila pedroso": true, "parque amazônia": true,
		"jardim goiás": true, "jardim balneário": true, "alto da glória": true,
		"setor marista": true, "setor universitário": true, "setor coimbra": true,

		// Terminais Críticos - Eixos Estratégicos
		"terminal novo mundo": true, "terminal bíblia": true, "terminal canedo": true,
		"terminal isidória": true, "terminal padre pelágio": true,

		// Variações de nomes
		"novo mundo": true, "bíblia": true, "canedo": true, "senador canedo": true,
		"isidória": true, "padre pelágio": true,
	}

	normalized := strings.ToLower(location)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, ".", "")

	for loc := range goianiaLocations {
		normalizedLoc := strings.ReplaceAll(strings.ToLower(loc), " ", "")
		if strings.Contains(normalized, normalizedLoc) || strings.Contains(normalizedLoc, normalized) {
			return true
		}
	}

	return false
}

// calculateMapRoute calcula rota para visualização no mapa
func calculateMapRoute(origin, destination string) (*MapRouteResponse, error) {
	// Coordenadas base para diferentes locais de Goiânia (coordenadas reais aproximadas)
	locations := map[string]MapPoint{
		// Locais existentes
		"setor bueno":        {Name: "Setor Bueno", Latitude: -16.6864, Longitude: -49.2643},
		"setor centro":       {Name: "Setor Centro", Latitude: -16.6807, Longitude: -49.2671},
		"terminal centro":    {Name: "Terminal Centro", Latitude: -16.6807, Longitude: -49.2671},
		"terminal samambaia": {Name: "Terminal Samambaia", Latitude: -16.6825, Longitude: -49.2655},
		"campus samambaia":   {Name: "Campus Samambaia UFG", Latitude: -16.6831, Longitude: -49.2674},
		"ufg":                {Name: "Campus Samambaia UFG", Latitude: -16.6831, Longitude: -49.2674},
		"setor oeste":        {Name: "Setor Oeste", Latitude: -16.6820, Longitude: -49.2700},
		"setor norte":        {Name: "Setor Norte", Latitude: -16.6780, Longitude: -49.2650},
		"vila nova":          {Name: "Vila Nova", Latitude: -16.6900, Longitude: -49.2600},

		// Terminais Críticos - Eixos de Goiânia (coordenadas reais)
		"terminal novo mundo":    {Name: "Terminal Novo Mundo", Latitude: -16.6680, Longitude: -49.2580},
		"terminal bíblia":        {Name: "Terminal Bíblia", Latitude: -16.6700, Longitude: -49.2750},
		"terminal canedo":        {Name: "Terminal Canedo", Latitude: -16.6820, Longitude: -49.2200}, // Senador Canedo
		"terminal isidória":      {Name: "Terminal Isidória", Latitude: -16.6900, Longitude: -49.2680},
		"terminal padre pelágio": {Name: "Terminal Padre Pelágio", Latitude: -16.6750, Longitude: -49.2500},

		// Variações de nomes para matching
		"novo mundo":     {Name: "Terminal Novo Mundo", Latitude: -16.6680, Longitude: -49.2580},
		"bíblia":         {Name: "Terminal Bíblia", Latitude: -16.6700, Longitude: -49.2750},
		"canedo":         {Name: "Terminal Canedo", Latitude: -16.6820, Longitude: -49.2200},
		"senador canedo": {Name: "Terminal Canedo", Latitude: -16.6820, Longitude: -49.2200},
		"isidória":       {Name: "Terminal Isidória", Latitude: -16.6900, Longitude: -49.2680},
		"padre pelágio":  {Name: "Terminal Padre Pelágio", Latitude: -16.6750, Longitude: -49.2500},
	}

	originNorm := normalizeParam(origin)
	destNorm := normalizeParam(destination)

	originPoint, originExists := locations[originNorm]
	destPoint, destExists := locations[destNorm]

	if !originExists || !destExists {
		// Se não encontrar nos locais conhecidos, usar coordenadas aproximadas
		if !originExists {
			originPoint = MapPoint{Name: origin, Latitude: -16.6860, Longitude: -49.2640}
		}
		if !destExists {
			destPoint = MapPoint{Name: destination, Latitude: -16.6830, Longitude: -49.2670}
		}
	}

	// Calcular rota baseada nos pontos
	route := &MapRouteResponse{
		Origin:      originPoint,
		Destination: destPoint,
		Steps:       []MapStep{},
		BusLines:    []string{"M23", "M71"},
	}

	// Adicionar passos intermediários se necessário
	if originNorm != destNorm {
		// Adicionar ponto de origem
		route.Steps = append(route.Steps, MapStep{
			Name:       originPoint.Name,
			Latitude:   originPoint.Latitude,
			Longitude:  originPoint.Longitude,
			IsTerminal: false,
			IsTransfer: false,
		})

		// Adicionar terminal intermediário se for rota longa
		dist := calcularDistancia(originPoint.Latitude, originPoint.Longitude, destPoint.Latitude, destPoint.Longitude)
		if dist > 2.0 { // Se distância > 2km, adicionar terminal
			route.Steps = append(route.Steps, MapStep{
				Name:       "Terminal Centro",
				Latitude:   -16.6807,
				Longitude:  -49.2671,
				IsTerminal: true,
				IsTransfer: true,
			})
		}

		// Adicionar destino
		route.Steps = append(route.Steps, MapStep{
			Name:       destPoint.Name,
			Latitude:   destPoint.Latitude,
			Longitude:  destPoint.Longitude,
			IsTerminal: false,
			IsTransfer: false,
		})

		// Calcular tempo baseado na distância
		route.TotalTimeMinutes = int(dist*10) + 10 // Base: 10min + 1min por 100m
		if route.TotalTimeMinutes < 15 {
			route.TotalTimeMinutes = 15
		}
	} else {
		// Mesmo local
		route.TotalTimeMinutes = 0
		route.Steps = append(route.Steps, MapStep{
			Name:       originPoint.Name,
			Latitude:   originPoint.Latitude,
			Longitude:  originPoint.Longitude,
			IsTerminal: false,
			IsTransfer: false,
		})
	}

	return route, nil
}

// getTrendingRoutes busca as 3 rotas mais buscadas nos últimos 7 dias usando PostGIS
func getTrendingRoutes(app *App) ([]TrendingRoute, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Se banco offline, retornar fallback hardcoded
	if app.db == nil {
		return []TrendingRoute{
			{Origin: "Terminal Novo Mundo", Destination: "Campus Samambaia UFG", Count: 0, LastSearch: time.Now()},
			{Origin: "Terminal Bíblia", Destination: "Terminal Canedo", Count: 0, LastSearch: time.Now()},
			{Origin: "Terminal Isidória", Destination: "Terminal Padre Pelágio", Count: 0, LastSearch: time.Now()},
		}, nil
	}

	// Usar função PostGIS otimizada
	trending, err := app.getTrendingRoutesPostGIS(ctx, 7, 3)
	if err != nil {
		return nil, fmt.Errorf("erro ao buscar trending routes PostGIS: %v", err)
	}

	return trending, nil
}

// warmUpTrendingCache pré-carrega rotas trending no Redis
func warmUpTrendingCache(app *App, trending []TrendingRoute) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Erro em warmUpTrendingCache: %v", r)
			}
		}()

		ctx := context.Background()

		for _, route := range trending {
			// Pré-calcular e cache da rota trending
			calculatedRoute, err := calculateMapRoute(route.Origin, route.Destination)

			if err == nil && calculatedRoute != nil {
				// Salvar no Redis com TTL mais longo para rotas populares
				cacheKey := fmt.Sprintf("route:%s:%s", normalizeParam(route.Origin), normalizeParam(route.Destination))
				jsonData, _ := json.Marshal(calculatedRoute)

				// TTL de 1 hora para rotas trending (vs 10 min normais)
				app.rdb.Set(ctx, cacheKey, string(jsonData), 1*time.Hour)
				log.Printf("Warm-up cache para rota trending: %s -> %s", route.Origin, route.Destination)
			}
		}
	}()
}

// getLocationCoordinates obtém coordenadas de uma localização para PostGIS
func getLocationCoordinates(location string) SpatialPoint {
	// Coordenadas base para diferentes locais de Goiânia
	locations := map[string]SpatialPoint{
		// Locais existentes
		"setor bueno":        {X: -49.2643, Y: -16.6864},
		"setor centro":       {X: -49.2671, Y: -16.6807},
		"terminal centro":    {X: -49.2671, Y: -16.6807},
		"terminal samambaia": {X: -49.2655, Y: -16.6825},
		"campus samambaia":   {X: -49.2674, Y: -16.6831},
		"ufg":                {X: -49.2674, Y: -16.6831},
		"setor oeste":        {X: -49.2700, Y: -16.6820},
		"setor norte":        {X: -49.2650, Y: -16.6780},
		"vila nova":          {X: -49.2600, Y: -16.6900},

		// Terminais Críticos - Eixos de Goiânia (coordenadas reais)
		"terminal novo mundo":    {X: -49.2580, Y: -16.6680},
		"terminal bíblia":        {X: -49.2750, Y: -16.6700},
		"terminal canedo":        {X: -49.2200, Y: -16.6820}, // Senador Canedo
		"terminal isidória":      {X: -49.2680, Y: -16.6900},
		"terminal padre pelágio": {X: -49.2500, Y: -16.6750},

		// Variações de nomes para matching
		"novo mundo":     {X: -49.2580, Y: -16.6680},
		"bíblia":         {X: -49.2750, Y: -16.6700},
		"canedo":         {X: -49.2200, Y: -16.6820},
		"senador canedo": {X: -49.2200, Y: -16.6820},
		"isidória":       {X: -49.2680, Y: -16.6900},
		"padre pelágio":  {X: -49.2500, Y: -16.6750},
	}

	normalized := strings.ToLower(location)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, ".", "")

	for loc, coords := range locations {
		normalizedLoc := strings.ReplaceAll(strings.ToLower(loc), " ", "")
		if strings.Contains(normalized, normalizedLoc) || strings.Contains(normalizedLoc, normalized) {
			return coords
		}
	}

	// Fallback para coordenadas aproximadas de Goiânia
	return SpatialPoint{X: -49.2671, Y: -16.6807}
}
