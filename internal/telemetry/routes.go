package telemetry

import (
	"database/sql"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/auth"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/infrastructure"
	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
)

// SetupRoutes configura todas as rotas de telemetria
func SetupRoutes(r *gin.Engine, db *sql.DB, rdb *redis.Client, jwtManager *auth.JWTManager) {
	logger.Info("Routes", "Criando Controller de telemetria...")
	controller := NewController(db, rdb)
	logger.Info("Routes", "Controller de telemetria criado")

	logger.Info("Routes", "Criando WebSocketHub...")
	wsHub := infrastructure.NewWebSocketHub(rdb)
	logger.Info("Routes", "WebSocketHub criado")

	// POST /api/v1/telemetry/gps - Receber ping de telemetria GPS
	// Telemetria passiva do usuário com LGPD compliance
	// Protegido com AuthMiddleware
	r.POST("/api/v1/telemetry/gps", auth.AuthMiddleware(jwtManager), controller.ReceiveGPSPing)

	// GET /api/v1/telemetry/last-position/:device_hash - Última posição conhecida
	// Busca no Redis (cache 60s) ou fallback para PostgreSQL
	r.GET("/api/v1/telemetry/last-position/:device_hash", controller.GetLastPosition)

	// GET /api/v1/telemetry/latest - Última posição de todos os dispositivos
	// Retorna DISTINCT ON (device_hash) com ORDER BY device_hash, recorded_at DESC
	r.GET("/api/v1/telemetry/latest", controller.GetLatestPositions)

	// GET /api/v1/telemetry/alerts - Alertas de geofencing
	// Retorna incidentes de veículos fora da rota ou parados fora do terminal
	r.GET("/api/v1/telemetry/alerts", controller.GetGeofenceAlerts)

	// GET /api/v1/telemetry/eta/:device_hash - Tempo estimado de chegada (ETA)
	// Calcula ETA baseado em distância e velocidade média
	r.GET("/api/v1/telemetry/eta/:device_hash", controller.CalculateETA)

	// GET /api/v1/telemetry/ws - WebSocket para atualizações em tempo real
	// Upgrade HTTP para WebSocket e faz broadcast de atualizações de GPS
	r.GET("/api/v1/telemetry/ws", func(c *gin.Context) {
		wsHub.HandleWebSocket(c.Writer, c.Request)
	})

	// GET /api/v1/analytics/fleet-health - Métricas de saúde da frota
	// Retorna eficiência, dwell time e outras métricas agregadas
	r.GET("/api/v1/analytics/fleet-health", controller.GetFleetHealth)

	// GET /api/v1/telemetry/history - Histórico de coordenadas de um dispositivo
	// Retorna trajetória em um intervalo de tempo com decimation automática
	// Protegido com AuthMiddleware
	r.GET("/api/v1/telemetry/history", auth.AuthMiddleware(jwtManager), controller.GetHistory)

	// GET /api/v1/telemetry/export - Exportar histórico em CSV
	// Retorna arquivo CSV para auditoria externa
	// Protegido com AuthMiddleware
	r.GET("/api/v1/telemetry/export", auth.AuthMiddleware(jwtManager), controller.ExportHistory)

	// GET /api/v1/telemetry/fleet-status - Status da frota para CCO
	// Retorna métricas gerenciais: ônibus ativos, alertas, velocidade média
	// Protegido com AuthMiddleware
	r.GET("/api/v1/telemetry/fleet-status", auth.AuthMiddleware(jwtManager), controller.GetFleetStatus)

	// ============================================================================
	// CX & MONETIZATION ROUTES
	// ============================================================================

	// GET /api/v1/telemetry/eta-confidence/:device_hash - ETA com confiança
	// Retorna ETA com intervalo de confiança (ex: "90% chance de chegar em 4 min")
	r.GET("/api/v1/telemetry/eta-confidence/:device_hash", controller.CalculateETAWithConfidence)

	// GET /api/v1/user/status - Status do usuário para monetização
	// Retorna is_premium, ad_free, points, check_in_streak
	r.GET("/api/v1/user/status", auth.AuthMiddleware(jwtManager), controller.GetUserStatus)

	// POST /api/v1/user/ad-free-toggle - Alternar modo ad-free (Premium)
	r.POST("/api/v1/user/ad-free-toggle", auth.AuthMiddleware(jwtManager), controller.ToggleAdFree)

	// POST /api/v1/gamification/check-in - Check-in em ponto de ônibus
	// Gamificação: ganha pontos por check-in, mantém streak
	r.POST("/api/v1/gamification/check-in", auth.AuthMiddleware(jwtManager), controller.RecordCheckIn)

	// POST /api/v1/gamification/occupancy - Reportar lotação do ônibus
	// Gamificação: ganha pontos por reportar lotação em tempo real
	r.POST("/api/v1/gamification/occupancy", auth.AuthMiddleware(jwtManager), controller.ReportOccupancy)

	// GET /api/v1/messages - Mensagens localizadas para Goiânia
	// Retorna mensagens amigáveis: "Bora pro Eixo?", "Seu ônibus tá chegando!"
	r.GET("/api/v1/messages", controller.GetLocalizedMessages)

	// GET /api/v1/ads/slots - Configuração de anúncios (AdSense)
	// Retorna slots de anúncio (vazio para Premium users)
	r.GET("/api/v1/ads/slots", controller.GetAdSlots)

	// ============================================================================
	// GEOLOCATION & COLLECTIVE INTELLIGENCE ROUTES
	// ============================================================================

	// Rotas GEO
	geoController := NewGeoController(controller.repo)
	r.POST("/api/v1/bus-update", geoController.ReceiveBusUpdate)
	r.GET("/api/v1/bus-locations", geoController.GetBusLocations)
	r.GET("/api/v1/buses/all", geoController.GetAllBuses)

	logger.Info("Routes", "Rotas de telemetria configuradas")
}
