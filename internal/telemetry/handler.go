package telemetry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/redis/go-redis/v9"
)

// Controller gerencia operações de telemetria
type Controller struct {
	repo      *Repository
	redis     *redis.Client
	antispoof *AntiSpoofingValidator
}

// NewController cria um novo controller
func NewController(db *sql.DB, rdb *redis.Client) *Controller {
	return &Controller{
		repo:      NewRepository(db, rdb),
		redis:     rdb,
		antispoof: NewAntiSpoofingValidator(rdb),
	}
}

// ReceiveGPSPing recebe dados de telemetria GPS do usuário.
//
// POST /api/v1/telemetry/gps
//
// Esta função valida o payload, anonimiza o device_id (LGPD), e processa
// o salvamento em background (non-blocking) para retornar 202 Accepted imediatamente.
//
// Processamento em background inclui:
//   - Salvamento no PostgreSQL/PostGIS
//   - Atualização do cache Redis (última posição)
//   - Verificação de geofencing
//   - Publicação no Redis Pub/Sub para WebSocket broadcast
//
// Parameters:
//   - ctx: Contexto Gin com a requisição HTTP
//
// Returns:
//   - 202 Accepted: Se o payload for válido e aceito para processamento
//   - 400 Bad Request: Se o JSON for inválido ou validação falhar
//   - 422 Unprocessable Entity: Se houver erros de validação nos campos
func (c *Controller) ReceiveGPSPing(ctx *gin.Context) {
	start := time.Now()

	var ping TelemetryPing
	// Usar ShouldBindBodyWith para evitar erro EOF (body pode ser lido uma vez)
	if err := ctx.ShouldBindBodyWith(&ping, binding.JSON); err != nil {
		c.logSecurityBlock(ctx, &ping, "Invalid JSON payload", err.Error())
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON payload",
			"details": err.Error(),
		})
		return
	}

	// Audit trail: log estruturado de evento de usuário (sem device_id raw por segurança)
	logger.Info("Telemetry", "GPS ping received | IP: %s | Lat: %.6f | Lng: %.6f | Speed: %.2f",
		ctx.ClientIP(), ping.Latitude, ping.Longitude, ping.Speed)

	// Validações de segurança
	validationErrors := c.validatePing(&ping)
	if len(validationErrors) > 0 {
		c.logSecurityBlock(ctx, &ping, fmt.Sprintf("Validation errors: %v", validationErrors), "")
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": "Validation failed",
		})
		return
	}

	// Anonimização LGPD: hash do device_id com sal diária
	deviceHash := c.anonymizeDeviceID(ping.DeviceID)
	logger.Info("Telemetry", "Device anonymized | DeviceID: %s -> DeviceHash: %s",
		ping.DeviceID, deviceHash)

	// Anti-spoofing: validar viagem no tempo (GPS jump)
	timestamp := time.Now()
	valid, reason := c.antispoof.ValidateGPSTravel(context.Background(), deviceHash, ping.Latitude, ping.Longitude, timestamp)
	if !valid {
		c.logSecurityBlock(ctx, &ping, "GPS time travel detected", reason)
		logger.Warn("Security", "GPS time travel blocked | Device: %s | Reason: %s", deviceHash, reason)
		ctx.JSON(http.StatusForbidden, gin.H{
			"error":  "GPS time travel detected",
			"reason": reason,
		})
		return
	}

	// Gerar telemetryID antes do processamento assíncrono para resposta imediata
	generatedID := fmt.Sprintf("tel-%d", time.Now().UnixNano())

	// Processar salvamento em background (non-blocking)
	go c.processGPSPingAsync(start, deviceHash, &ping)

	ctx.JSON(http.StatusAccepted, gin.H{
		"msg": "GPS ping received and queued for processing",
		"tid": generatedID,
		"dh":  deviceHash,
		"pms": time.Since(start).Milliseconds(),
	})

	// Log de performance
	LogAPILatency("POST /api/v1/telemetry/gps", time.Since(start))
}

// processGPSPingAsync processa o salvamento de GPS ping em background
func (c *Controller) processGPSPingAsync(start time.Time, deviceHash string, ping *TelemetryPing) {
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Salvar no PostgreSQL/PostGIS
	telemetryID, err := c.repo.SaveToDatabase(dbCtx, deviceHash, ping)
	if err != nil {
		logger.Error("Telemetry", "Database save failed | DeviceHash: %s | Error: %v",
			deviceHash, err)
		return
	}

	logger.Info("Telemetry", "GPS packet saved | ID: %s | DeviceHash: %s | Lat: %.6f | Lng: %.6f | Speed: %.2f | ProcessingTime: %v",
		telemetryID, deviceHash, ping.Latitude, ping.Longitude, ping.Speed, time.Since(start))

	// 2. Atualizar cache Redis (última posição conhecida)
	err = c.updateRedisCache(dbCtx, deviceHash, ping)
	if err != nil {
		logger.Warn("Telemetry", "Redis cache update failed | DeviceHash: %s | Error: %v",
			deviceHash, err)
	}

	// 3. Verificar geofencing
	c.checkGeofencing(dbCtx, deviceHash, ping.Latitude, ping.Longitude, ping.RouteID)

	// 4. Publicar no canal Redis para WebSocket
	c.publishBusUpdate(dbCtx, deviceHash, ping)
}

// logSecurityBlock registra bloqueio de segurança no audit log de forma assíncrona
func (c *Controller) logSecurityBlock(ctx *gin.Context, ping *TelemetryPing, reason, details string) {
	clientIP := ctx.ClientIP()
	payloadJSON, _ := json.Marshal(ping)

	actorID := "unknown"
	if userID, exists := ctx.Get("user_id"); exists {
		actorID = userID.(string)
	}

	go func() {
		auditCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		c.repo.LogAudit(auditCtx, "system", actorID, "SECURITY_BLOCK", "gps_telemetry", "",
			clientIP, ctx.GetHeader("User-Agent"), string(payloadJSON), "", reason)
	}()
}

// GetLatestPositions retorna a última posição conhecida de cada dispositivo.
//
// GET /api/v1/telemetry/latest
//
// Implementa estratégia Redis-First:
//  1. Tenta buscar do Redis primeiro (mais rápido)
//  2. Fallback para PostgreSQL se Redis falhar ou não tiver dados
//
// Sempre retorna 200 OK com array vazio [] se não houver dados,
// nunca retorna erro 500 para evitar quebrar o frontend.
func (c *Controller) GetLatestPositions(ctx *gin.Context) {
	start := time.Now()
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Verificar integridade do repositório
	if c.repo == nil {
		logger.Warn("Telemetry", "Repository is nil, returning empty positions")
		c.respondLatestPositions(ctx, []LatestPosition{}, "none", start)
		return
	}

	// Tenta Redis-First
	if positions, source := c.tryGetFromRedis(dbCtx, start); len(positions) > 0 {
		c.respondLatestPositions(ctx, positions, source, start)
		return
	}

	// Fallback para PostgreSQL
	positions, err := c.repo.GetAllLatestPositions(dbCtx)
	if err != nil {
		logger.Error("Telemetry", "Failed to get latest positions from DB: %v", err)
		c.respondLatestPositions(ctx, []LatestPosition{}, "none", start)
		return
	}

	c.respondLatestPositions(ctx, positions, "database", start)
}

// tryGetFromRedis tenta buscar posições do Redis
func (c *Controller) tryGetFromRedis(ctx context.Context, start time.Time) ([]LatestPosition, string) {
	if c.redis == nil {
		return nil, ""
	}

	deviceHashes, err := c.redis.SMembers(ctx, "telemetry:active_devices").Result()
	if err != nil || len(deviceHashes) == 0 {
		return nil, ""
	}

	positions, err := c.getPositionsFromRedis(ctx, deviceHashes)
	if err != nil || len(positions) == 0 {
		return nil, ""
	}

	logger.Info("Telemetry", "Retrieved %d positions from Redis in %v", len(positions), time.Since(start))
	return positions, "redis"
}

// respondLatestPositions envia resposta padronizada de posições
func (c *Controller) respondLatestPositions(ctx *gin.Context, positions []LatestPosition, source string, start time.Time) {
	if positions == nil {
		positions = []LatestPosition{}
	}

	response := gin.H{
		"n": len(positions),
		"p": positions,
		"s": source,
	}

	// Log colorido com JSON exato
	jsonData, _ := json.MarshalIndent(response, "", "  ")
	logger.Info("Telemetry", "\x1b[32m[JSON RESPONSE]\x1b[0m Retrieved %d latest positions from %s in %v\n%s", len(positions), source, time.Since(start), string(jsonData))

	ctx.JSON(http.StatusOK, response)
}

// getPositionsFromRedis busca posições de múltiplos devices do Redis
func (c *Controller) getPositionsFromRedis(ctx context.Context, deviceHashes []string) ([]LatestPosition, error) {
	var positions []LatestPosition

	for _, deviceHash := range deviceHashes {
		position, err := c.getFromRedisCache(ctx, deviceHash)
		if err != nil {
			continue // Pular devices sem cache
		}

		lat, ok1 := position["lat"].(float64)
		lng, ok2 := position["lng"].(float64)
		speed, _ := position["speed"].(float64)
		routeID, _ := position["route_id"].(string)
		recordedAtUnix, _ := position["recorded_at"].(float64)

		if !ok1 || !ok2 {
			continue
		}

		// Buscar status de trânsito para esta posição
		trafficStatus := c.getTrafficStatus(ctx, lat, lng)

		positions = append(positions, LatestPosition{
			DeviceHash: deviceHash,
			RouteID:    routeID,
			Speed:      speed,
			Location: Location{
				Lat: lat,
				Lng: lng,
			},
			RecordedAt:    time.Unix(int64(recordedAtUnix), 0),
			TrafficStatus: trafficStatus,
		})
	}

	return positions, nil
}

// checkGeofencing verifica se o dispositivo está dentro dos geofences permitidos
// Usa Redis para armazenar estado In/Out e evitar alertas duplicados
func (c *Controller) checkGeofencing(ctx context.Context, deviceHash string, lat, lng float64, routeID string) {
	// Buscar todos os geofences onde o ponto está dentro
	geofences, err := c.repo.CheckAllGeofences(ctx, lat, lng)
	if err != nil {
		logger.Info("Telemetry", " Failed to check geofences: %v", err)
		return
	}

	// Se não há geofences configurados, não fazer nada
	if len(geofences) == 0 {
		return
	}

	// Verificar cada geofence
	for _, gf := range geofences {
		// Chave do Redis para armazenar estado: geofence:state:{device_hash}:{geofence_id}
		redisKey := fmt.Sprintf("geofence:state:%s:%d", deviceHash, gf.ID)

		// Buscar estado anterior do Redis
		previousState, err := c.redis.Get(ctx, redisKey).Result()
		currentState := "In" // Está dentro do polígono

		if err == redis.Nil {
			// Primeira vez que detectamos este device neste geofence
			// Salvar estado atual no Redis
			c.redis.Set(ctx, redisKey, currentState, 5*time.Minute)
			logger.Info("Telemetry", " Device %s entrou no geofence %s (%s)", deviceHash[:8]+"...", gf.Nome, gf.Tipo)
			continue
		}

		// Se o estado mudou, disparar alerta
		if previousState != currentState {
			// Salvar novo estado no Redis
			c.redis.Set(ctx, redisKey, currentState, 5*time.Minute)

			// Criar alerta no banco
			err = c.repo.CreateGeofenceAlert(ctx, deviceHash, gf.ID, gf.Nome, currentState, lat, lng)
			if err != nil {
				logger.Info("Telemetry", " Failed to create alert: %v", err)
				continue
			}

			// Log de alerta
			if currentState == "Out" {
				logger.Info("Telemetry", " ⚠️ [ALERTA] Device %s SAIU do geofence %s (%s) - Lat: %.6f, Lng: %.6f",
					deviceHash[:8]+"...", gf.Nome, gf.Tipo, lat, lng)
			} else {
				logger.Info("Telemetry", " Device %s ENTROU no geofence %s (%s) - Lat: %.6f, Lng: %.6f",
					deviceHash[:8]+"...", gf.Nome, gf.Tipo, lat, lng)
			}
		}
	}

	// Verificar se device saiu de todos os geofences onde estava antes
}

// checkTrafficCongestion verifica se há congestionamento em geofences
// Se 3+ ônibus estiverem abaixo de 10km/h em um polígono, marca como CONGESTIONADA
func (c *Controller) checkTrafficCongestion(ctx context.Context) {
	// Lista de geofence IDs para monitorar (exemplo)
	geofenceIDs := []int64{1, 2} // IDs dos geofences criados na migration

	for _, gfID := range geofenceIDs {
		// Calcular velocidade média no polígono (últimos 15 min)
		avgSpeed, err := c.repo.GetAverageSpeedInPolygon(ctx, gfID, 15)
		if err != nil {
			logger.Warn("Telemetry", "Failed to get avg speed for geofence %d: %v", gfID, err)
			continue
		}

		// Contar quantos ônibus estão no polígono
		busCount, err := c.repo.GetBusesCountInPolygon(ctx, gfID, 15)
		if err != nil {
			logger.Warn("Telemetry", "Failed to count buses for geofence %d: %v", gfID, err)
			continue
		}

		// Chave do Redis para armazenar status de congestionamento
		redisKey := fmt.Sprintf("congestion:geofence:%d", gfID)

		// Lógica: 3+ ônibus < 10km/h = CONGESTIONADA
		if busCount >= 3 && avgSpeed > 0 && avgSpeed < 10.0 {
			// Marcar como CONGESTIONADA no Redis por 10 minutos
			c.redis.Set(ctx, redisKey, "congested", 10*time.Minute)
			logger.Warn("Telemetry", "⚠️ Geofence %d marcada como CONGESTIONADA (buses: %d, avg speed: %.1f km/h)",
				gfID, busCount, avgSpeed)
		} else if avgSpeed >= 20.0 {
			// Se velocidade recuperou para 20+ km/h, marcar como FLUIDO
			c.redis.Set(ctx, redisKey, "fluid", 10*time.Minute)
		}
	}
}

// getTrafficStatus busca o status de trânsito de uma posição
// Retorna 'fluido', 'moderado', 'lento' baseado no Redis
func (c *Controller) getTrafficStatus(ctx context.Context, lat, lng float64) string {
	// Buscar geofences onde o ponto está dentro
	geofences, err := c.repo.CheckAllGeofences(ctx, lat, lng)
	if err != nil || len(geofences) == 0 {
		return "desconhecido" // Sem geofences na área
	}

	// Verificar status de congestionamento nos geofences
	for _, gf := range geofences {
		redisKey := fmt.Sprintf("congestion:geofence:%d", gf.ID)
		status, err := c.redis.Get(ctx, redisKey).Result()
		if err == nil {
			// Status encontrado no Redis
			switch status {
			case "congested":
				return "lento"
			case "fluid":
				return "fluido"
			default:
				return "moderado"
			}
		}
	}

	// Se nenhum status encontrado, retorna moderado por padrão
	return "moderado"
}

// publishBusUpdate publica atualização de GPS no Redis Pub/Sub para WebSocket broadcast
// Payload otimizado: apenas id, lat, lng, speed, traffic_status
func (c *Controller) publishBusUpdate(ctx context.Context, deviceHash string, ping *TelemetryPing) {
	// Validar sanidade dos dados GPS
	if !c.validateTelemetrySanity(ctx, deviceHash, ping) {
		return
	}

	// Buscar status de trânsito para a posição
	trafficStatus := c.getTrafficStatus(ctx, ping.Latitude, ping.Longitude)

	// Payload otimizado (JSON leve)
	payload := map[string]interface{}{
		"id":             deviceHash,
		"lat":            ping.Latitude,
		"lng":            ping.Longitude,
		"speed":          ping.Speed,
		"traffic_status": trafficStatus,
		"timestamp":      time.Now().Unix(),
	}

	// Serializar para JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		logger.Error("Telemetry", "Failed to marshal payload: %v", err)
		return
	}

	// Publicar no canal Redis
	err = c.redis.Publish(ctx, "bus_updates", jsonData).Err()
	if err != nil {
		logger.Error("Telemetry", "Failed to publish: %v", err)
		return
	}

	logger.Info("Telemetry", "Published update for device %s", deviceHash[:8]+"...")
}

// validateTelemetrySanity valida sanidade dos dados GPS e detecta anomalias
// Retorna true se válido, false se anômalo
func (c *Controller) validateTelemetrySanity(ctx context.Context, deviceHash string, ping *TelemetryPing) bool {
	// Filtro 1: Velocidade > 150km/h é anômalo (ônibus urbano não chega a essa velocidade)
	if ping.Speed > 150.0 {
		logger.Warn("Telemetry", "⚠️ Velocidade anômala detectada: %.1f km/h para device %s", ping.Speed, deviceHash[:8]+"...")

		// Registrar anomalia no audit log
		c.repo.LogAudit(ctx, "device", deviceHash, "anomaly_detected", "gps_telemetry", "",
			"", "", fmt.Sprintf(`{"reason": "speed_too_high", "speed": %.1f}`, ping.Speed), "", "")

		return false
	}

	// Filtro 2: Teletransporte impossível (distância/tempo)
	// Buscar última posição do dispositivo
	lastLat, lastLng, _, err := c.repo.GetDeviceLatestPosition(ctx, deviceHash)
	if err == nil {
		// Calcular distância entre posições
		distanceMeters, err := c.repo.CalculateDistance(ctx, lastLat, lastLng, ping.Latitude, ping.Longitude)
		if err == nil {
			// Se distância > 10km em 10 segundos, é impossível (teletransporte)
			if distanceMeters > 10000 { // 10km
				logger.Warn("Telemetry", "⚠️ Teletransporte detectado: %.2f km em 10s para device %s",
					distanceMeters/1000, deviceHash[:8]+"...")

				// Registrar anomalia no audit log
				c.repo.LogAudit(ctx, "device", deviceHash, "anomaly_detected", "gps_telemetry", "",
					"", "", fmt.Sprintf(`{"reason": "teleport", "distance_meters": %.2f}`, distanceMeters), "", "")

				return false
			}
		}
	}

	return true
}

// GetFleetHealth - GET /api/v1/analytics/fleet-health
// Retorna métricas de saúde da frota (eficiência, dwell time, etc.)
// Usa cache Redis com TTL de 15 minutos para pre-computing
func (c *Controller) GetFleetHealth(ctx *gin.Context) {
	// Parâmetro de horas (padrão: 24)
	hours := 24
	if hoursStr := ctx.Query("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 { // Máximo 7 dias
			hours = h
		}
	}

	// Context para operações
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Chave do Redis para cache
	redisKey := fmt.Sprintf("analytics:fleet_health:%dh", hours)

	// Tentar buscar do Redis primeiro
	cachedData, err := c.redis.Get(dbCtx, redisKey).Result()
	if err == redis.Nil {
		// Cache miss - recalcular
		metrics, err := c.repo.GetFleetHealth(dbCtx, hours)
		if err != nil {
			logger.Error("Analytics", "Failed to get fleet health: %v", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve fleet health metrics",
			})
			return
		}

		// Serializar e salvar no cache (TTL 15 minutos)
		jsonData, _ := json.Marshal(metrics)
		c.redis.Set(dbCtx, redisKey, jsonData, 15*time.Minute)

		logger.Info("Analytics", " Fleet health calculated and cached (hours: %d, devices: %d)", hours, len(metrics))

		ctx.JSON(http.StatusOK, gin.H{
			"hours":         hours,
			"count":         len(metrics),
			"source":        "database",
			"metrics":       metrics,
			"calculated_at": time.Now(),
		})
		return
	} else if err != nil {
		// Erro no Redis - fallback para banco
		logger.Info("Analytics", " Redis error, falling back to DB: %v", err)
		metrics, err := c.repo.GetFleetHealth(dbCtx, hours)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve fleet health metrics",
			})
			return
		}

		ctx.JSON(http.StatusOK, gin.H{
			"hours":         hours,
			"count":         len(metrics),
			"source":        "database",
			"metrics":       metrics,
			"calculated_at": time.Now(),
		})
		return
	}

	// Cache hit - deserializar e retornar
	var metrics []FleetHealthMetric
	if err := json.Unmarshal([]byte(cachedData), &metrics); err != nil {
		logger.Info("Analytics", " Failed to unmarshal cache, recalculating: %v", err)
		// Recalcular se deserialização falhar
		metrics, err := c.repo.GetFleetHealth(dbCtx, hours)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve fleet health metrics",
			})
			return
		}

		jsonData, _ := json.Marshal(metrics)
		c.redis.Set(dbCtx, redisKey, jsonData, 15*time.Minute)

		ctx.JSON(http.StatusOK, gin.H{
			"hours":         hours,
			"count":         len(metrics),
			"source":        "database",
			"metrics":       metrics,
			"calculated_at": time.Now(),
		})
		return
	}

	logger.Info("Analytics", " Fleet health from cache (hours: %d, devices: %d)", hours, len(metrics))

	ctx.JSON(http.StatusOK, gin.H{
		"hours":         hours,
		"count":         len(metrics),
		"source":        "cache",
		"metrics":       metrics,
		"calculated_at": time.Now(),
	})
}

// GetGeofenceAlerts - GET /api/v1/telemetry/alerts
// Retorna os últimos incidentes de geofencing (ônibus fora da rota ou parados fora do terminal)
func (c *Controller) GetGeofenceAlerts(ctx *gin.Context) {
	start := time.Now()

	// Context para operações de banco
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Parâmetro de limite (padrão: 50)
	limit := 50
	if limitStr := ctx.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 500 {
			limit = l
		}
	}

	// Buscar alertas recentes
	alerts, err := c.repo.GetRecentGeofenceAlerts(dbCtx, limit)
	if err != nil {
		logger.Error("Telemetry", "Failed to get geofence alerts: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve geofence alerts",
		})
		return
	}

	// Log para analytics
	processingTime := time.Since(start)
	logger.Info("Telemetry", "Retrieved %d geofence alerts in %v", len(alerts), processingTime)

	// Resposta
	ctx.JSON(http.StatusOK, gin.H{
		"count":  len(alerts),
		"alerts": alerts,
	})
}

// CalculateETA calcula o tempo estimado de chegada (ETA) para um destino.
//
// GET /api/v1/telemetry/eta/:device_hash
//
// Calcula ETA baseado em distância (PostGIS ST_Distance) e velocidade média recente.
// Valida device_hash e coordenadas de destino.
//
// Parameters:
//   - ctx: Contexto Gin com a requisição HTTP
//   - device_hash: Hash SHA-256 do dispositivo (path parameter)
//   - lat: Latitude do destino (query parameter)
//   - lng: Longitude do destino (query parameter)
//
// Returns:
//   - 200 OK: Com ETA em minutos
//   - 400 Bad Request: Se parâmetros forem inválidos
func (c *Controller) CalculateETA(ctx *gin.Context) {
	deviceHash := ctx.Param("device_hash")

	// Parâmetros de destino
	destLatStr := ctx.Query("lat")
	destLngStr := ctx.Query("lng")

	if destLatStr == "" || destLngStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing destination coordinates (lat, lng)",
		})
		return
	}

	destLat, err := strconv.ParseFloat(destLatStr, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid latitude",
		})
		return
	}

	destLng, err := strconv.ParseFloat(destLngStr, 64)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid longitude",
		})
		return
	}

	// Context para operações de banco
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Buscar última posição do dispositivo
	lastLat, lastLng, currentSpeed, err := c.repo.GetDeviceLatestPosition(dbCtx, deviceHash)
	if err != nil {
		logger.Error("Telemetry", "Failed to get device position: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get device position",
		})
		return
	}

	// Calcular distância até o destino (em metros)
	distanceMeters, err := c.repo.CalculateDistance(dbCtx, lastLat, lastLng, destLat, destLng)
	if err != nil {
		logger.Error("Telemetry", "Failed to calculate distance: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to calculate ETA",
		})
		return
	}

	// Determinar velocidade média (usa velocidade atual ou padrão de 40km/h)
	avgSpeedKmh := currentSpeed
	if avgSpeedKmh <= 0 || avgSpeedKmh > 150 {
		avgSpeedKmh = 40.0 // Velocidade padrão de via urbana
	}

	// Calcular ETA em minutos
	// Fórmula: Tempo (horas) = Distância (km) / Velocidade (km/h)
	// Tempo (minutos) = Tempo (horas) * 60
	distanceKm := distanceMeters / 1000.0
	etaHours := distanceKm / avgSpeedKmh
	etaMinutes := etaHours * 60

	// Log para analytics
	logger.Info("Telemetry", "ETA calculated for device %s: %.2f km at %.1f km/h = %.1f min",
		deviceHash[:8]+"...", distanceKm, avgSpeedKmh, etaMinutes)

	// Resposta
	ctx.JSON(http.StatusOK, gin.H{
		"device_hash": deviceHash,
		"destination": gin.H{
			"lat": destLat,
			"lng": destLng,
		},
		"current_position": gin.H{
			"lat": lastLat,
			"lng": lastLng,
		},
		"distance_meters": distanceMeters,
		"distance_km":     distanceKm,
		"avg_speed_kmh":   avgSpeedKmh,
		"eta_minutes":     etaMinutes,
		"eta_seconds":     etaMinutes * 60,
		"calculated_at":   time.Now(),
	})
}

// GetLastPosition retorna a última posição conhecida de um dispositivo específico.
//
// GET /api/v1/telemetry/last-position/:device_hash
//
// Busca do Redis primeiro (cache), fallback para PostgreSQL.
// Valida que device_hash é hexadecimal válido.
//
// Parameters:
//   - ctx: Contexto Gin com a requisição HTTP
//   - device_hash: Hash SHA-256 do dispositivo (path parameter)
//
// Returns:
//   - 200 OK: Com a última posição
//   - 400 Bad Request: Se device_hash for inválido
func (c *Controller) GetLastPosition(ctx *gin.Context) {
	deviceHash := ctx.Param("device_hash")

	// Validar formato do hash
	if len(deviceHash) != 32 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device hash format",
		})
		return
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 1. Tentar buscar do Redis (mais rápido)
	position, err := c.getFromRedisCache(dbCtx, deviceHash)
	if err == nil && position != nil {
		ctx.JSON(http.StatusOK, gin.H{
			"source":   "redis",
			"position": position,
			"cached":   true,
		})
		return
	}

	// 2. Fallback para PostgreSQL
	position, err = c.repo.GetFromDatabase(dbCtx, deviceHash)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "No position found for device",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"source":   "database",
		"position": position,
		"cached":   false,
	})
}

// ============================================================================
// VALIDAÇÕES DE SEGURANÇA
// ============================================================================

func (c *Controller) validatePing(ping *TelemetryPing) []ValidationError {
	var errors []ValidationError

	// 1. Validação de velocidade urbana
	if ping.Speed > MaxUrbanSpeedKmh {
		errors = append(errors, ValidationError{
			Field:   "speed",
			Value:   fmt.Sprintf("%.1f", ping.Speed),
			Message: fmt.Sprintf("Speed exceeds urban limit of %.0f km/h", MaxUrbanSpeedKmh),
		})
	}

	// 2. Validação de velocidade negativa
	if ping.Speed < 0 {
		errors = append(errors, ValidationError{
			Field:   "speed",
			Value:   fmt.Sprintf("%.1f", ping.Speed),
			Message: "Speed cannot be negative",
		})
	}

	// 3. Validação de heading (0-360 graus)
	if ping.Heading < 0 || ping.Heading > 360 {
		errors = append(errors, ValidationError{
			Field:   "heading",
			Value:   fmt.Sprintf("%.1f", ping.Heading),
			Message: "Heading must be between 0 and 360 degrees",
		})
	}

	// 4. Validação de precisão GPS
	if ping.Accuracy > MaxAccuracyMeters {
		errors = append(errors, ValidationError{
			Field:   "accuracy",
			Value:   fmt.Sprintf("%.1f", ping.Accuracy),
			Message: fmt.Sprintf("GPS accuracy too poor (> %.0f m)", MaxAccuracyMeters),
		})
	}

	if ping.Accuracy > 0 && ping.Accuracy < MinAccuracyMeters {
		errors = append(errors, ValidationError{
			Field:   "accuracy",
			Value:   fmt.Sprintf("%.1f", ping.Accuracy),
			Message: fmt.Sprintf("GPS accuracy unrealistically low (< %.0f m)", MinAccuracyMeters),
		})
	}

	// 5. Validação de coordenadas (Goiania bounding box aproximado)
	// Latitude: -16.5 a -16.8, Longitude: -49.1 a -49.4
	if ping.Latitude < -17.0 || ping.Latitude > -15.0 {
		errors = append(errors, ValidationError{
			Field:   "lat",
			Value:   fmt.Sprintf("%.6f", ping.Latitude),
			Message: "Latitude outside valid range for Goiania region",
		})
	}

	if ping.Longitude < -50.0 || ping.Longitude > -48.0 {
		errors = append(errors, ValidationError{
			Field:   "lng",
			Value:   fmt.Sprintf("%.6f", ping.Longitude),
			Message: "Longitude outside valid range for Goiania region",
		})
	}

	// 6. Validação de timestamp (não pode ser no futuro)
	if ping.RecordedAt.After(time.Now().Add(1 * time.Minute)) {
		errors = append(errors, ValidationError{
			Field:   "recorded_at",
			Value:   ping.RecordedAt.Format(time.RFC3339),
			Message: "Timestamp cannot be in the future",
		})
	}

	// 7. Validação de timestamp (não pode ser muito antigo)
	if ping.RecordedAt.Before(time.Now().Add(-24 * time.Hour)) {
		errors = append(errors, ValidationError{
			Field:   "recorded_at",
			Value:   ping.RecordedAt.Format(time.RFC3339),
			Message: "Timestamp too old (> 24 hours)",
		})
	}

	// 8. Validação de modo de transporte
	validModes := map[string]bool{
		"bus": true, "car": true, "bike": true,
		"walk": true, "metro": true, "": true,
	}
	if !validModes[ping.TransportMode] {
		errors = append(errors, ValidationError{
			Field:   "transport_mode",
			Value:   ping.TransportMode,
			Message: "Invalid transport mode. Must be: bus, car, bike, walk, metro",
		})
	}

	// 9. Validação de bateria
	if ping.BatteryLevel > MaxBatteryLevel || ping.BatteryLevel < MinBatteryLevel {
		errors = append(errors, ValidationError{
			Field:   "battery_level",
			Value:   fmt.Sprintf("%d", ping.BatteryLevel),
			Message: "Battery level must be between 0 and 100",
		})
	}

	return errors
}

// ANONIMIZAÇÃO LGPD
// ============================================================================

// anonymizeDeviceID cria hash SHA-256 com sal diária
func (c *Controller) anonymizeDeviceID(deviceID string) string {
	salt := c.getDailySalt()
	h := sha256.New()
	h.Write([]byte(salt + deviceID))
	return hex.EncodeToString(h.Sum(nil))
}

// getDailySalt retorna sal baseado na data atual
func (c *Controller) getDailySalt() string {
	// Formato: YYYY-MM-DD (muda a cada 24h)
	return time.Now().UTC().Format("2006-01-02")
}

// isValidHex verifica se uma string contém apenas caracteres hexadecimais
func isValidHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// sanitizeDeviceID sanitiza device_id removendo caracteres perigosos
func sanitizeDeviceID(deviceID string) string {
	// Permitir apenas alfanumérico, hífens e underscores
	var sanitized strings.Builder
	for _, r := range deviceID {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			sanitized.WriteRune(r)
		}
	}
	return sanitized.String()
}

// ============================================================================
// OPERAÇÕES REDIS
// ============================================================================

func (c *Controller) updateRedisCache(ctx context.Context, deviceHash string, ping *TelemetryPing) error {
	if c.redis == nil {
		return fmt.Errorf("redis not available")
	}

	// Chave: last_pos:{device_hash}
	key := fmt.Sprintf("last_pos:%s", deviceHash)

	// Criar JSON da posição
	positionData := map[string]interface{}{
		"lat":            ping.Latitude,
		"lng":            ping.Longitude,
		"speed":          ping.Speed,
		"heading":        ping.Heading,
		"accuracy":       ping.Accuracy,
		"transport_mode": ping.TransportMode,
		"route_id":       ping.RouteID,
		"battery_level":  ping.BatteryLevel,
		"recorded_at":    ping.RecordedAt.Unix(),
		"cached_at":      time.Now().Unix(),
	}

	jsonData, err := json.Marshal(positionData)
	if err != nil {
		return fmt.Errorf("failed to marshal position: %w", err)
	}

	// Salvar com TTL de 60 segundos
	err = c.redis.Set(ctx, key, jsonData, RedisLastPosTTL).Err()
	if err != nil {
		return fmt.Errorf("failed to set redis key: %w", err)
	}

	// Adicionar à lista de devices ativos (Set para rápida verificação)
	// TTL maior (5 minutos) para manter lista de dispositivos recentes
	c.redis.SAdd(ctx, "telemetry:active_devices", deviceHash)
	c.redis.Expire(ctx, "telemetry:active_devices", 5*time.Minute)

	return nil
}

func (c *Controller) getFromRedisCache(ctx context.Context, deviceHash string) (map[string]interface{}, error) {
	if c.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	key := fmt.Sprintf("last_pos:%s", deviceHash)

	data, err := c.redis.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("position not in cache")
	}
	if err != nil {
		return nil, err
	}

	var position map[string]interface{}
	err = json.Unmarshal([]byte(data), &position)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal position: %w", err)
	}

	return position, nil
}

// ============================================================================
// LOGGING E ANALYTICS
// ============================================================================

func (c *Controller) logTelemetryReceived(deviceHash, routeID string, processingTime time.Duration) {
	// Log estruturado para analytics
	logData := map[string]interface{}{
		"event":         "telemetry_received",
		"device_hash":   deviceHash[:8] + "...", // Parcial para privacidade
		"route_id":      routeID,
		"processing_ms": processingTime.Milliseconds(),
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	jsonLog, _ := json.Marshal(logData)
	logger.Info("Analytics", " %s", string(jsonLog))
}

// UTILITÁRIOS
// ============================================================================

// GetActiveDevicesCount retorna número de dispositivos ativos (últimos 5 min)
func (c *Controller) GetActiveDevicesCount(ctx *gin.Context) (int64, error) {
	if c.redis == nil {
		return 0, fmt.Errorf("redis not available")
	}

	return c.redis.SCard(ctx, "telemetry:active_devices").Result()
}

// GetHistory retorna o histórico de coordenadas de um dispositivo em um intervalo de tempo.
//
// GET /api/v1/telemetry/history
//
// Requer JWT authentication.
// Valida device_id e intervalo temporal (máximo 7 dias).
//
// Parameters:
//   - ctx: Contexto Gin com a requisição HTTP
//   - device_id: ID do dispositivo (query parameter)
//   - start: Timestamp inicial RFC3339 (query parameter, opcional)
//   - end: Timestamp final RFC3339 (query parameter, opcional)
//
// Returns:
//   - 200 OK: Com histórico de posições
//   - 400 Bad Request: Se parâmetros forem inválidos
//   - 401 Unauthorized: Se JWT não for fornecido ou inválido
func (c *Controller) GetHistory(ctx *gin.Context) {
	start := time.Now()

	// Parâmetros de query
	deviceID := ctx.Query("device_id")
	if deviceID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameter: device_id",
		})
		return
	}

	// Validar e sanitizar device_id (máximo 255 caracteres)
	if len(deviceID) > 255 {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "device_id too long (max 255 characters)",
		})
		return
	}

	// Sanitizar: remover caracteres perigosos
	deviceID = sanitizeDeviceID(deviceID)
	if deviceID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid device_id format",
		})
		return
	}

	startStr := ctx.Query("start")
	endStr := ctx.Query("end")

	// Validar timestamps
	var startTime, endTime time.Time
	var err error

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start time format (use RFC3339)",
			})
			return
		}
	} else {
		// Padrão: 1 hora atrás
		startTime = time.Now().Add(-1 * time.Hour)
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid end time format (use RFC3339)",
			})
			return
		}
	} else {
		// Padrão: agora
		endTime = time.Now()
	}

	// Validar intervalo máximo (7 dias)
	if endTime.Sub(startTime) > 7*24*time.Hour {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Time range cannot exceed 7 days",
		})
		return
	}

	// Context para operações de banco
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Buscar histórico do banco
	points, err := c.repo.GetHistory(dbCtx, deviceID, startTime, endTime)
	if err != nil {
		logger.Error("Telemetry", "Failed to get history: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve history",
		})
		return
	}

	// Log para analytics
	processingTime := time.Since(start)
	logger.Info("Telemetry", "Retrieved %d history points for device %s in %v", len(points), deviceID, processingTime)

	// Resposta
	ctx.JSON(http.StatusOK, gin.H{
		"device_id":    deviceID,
		"start":        startTime.Format(time.RFC3339),
		"end":          endTime.Format(time.RFC3339),
		"count":        len(points),
		"points":       points,
		"decimated":    len(points) > 1000,
		"retrieved_at": time.Now(),
	})
}

// ExportHistory exporta o histórico de posições de um dispositivo em formato CSV.
//
// GET /api/v1/telemetry/export
//
// Requer JWT authentication.
// Valida device_id e intervalo temporal (máximo 7 dias).
// Retorna arquivo CSV com cabeçalho e linhas de dados.
//
// Parameters:
//   - ctx: Contexto Gin com a requisição HTTP
//   - device_id: ID do dispositivo (query parameter)
//   - start: Timestamp inicial RFC3339 (query parameter, opcional)
//   - end: Timestamp final RFC3339 (query parameter, opcional)
//
// Returns:
//   - 200 OK: Com arquivo CSV anexado
//   - 400 Bad Request: Se parâmetros forem inválidos
//   - 401 Unauthorized: Se JWT não for fornecido ou inválido
func (c *Controller) ExportHistory(ctx *gin.Context) {
	start := time.Now()

	// Parâmetros de query
	deviceID := ctx.Query("device_id")
	if deviceID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Missing required parameter: device_id",
		})
		return
	}

	startStr := ctx.Query("start")
	endStr := ctx.Query("end")

	// Validar timestamps
	var startTime, endTime time.Time
	var err error

	if startStr != "" {
		startTime, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid start time format (use RFC3339)",
			})
			return
		}
	} else {
		// Padrão: 24 horas atrás
		startTime = time.Now().Add(-24 * time.Hour)
	}

	if endStr != "" {
		endTime, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid end time format (use RFC3339)",
			})
			return
		}
	} else {
		// Padrão: agora
		endTime = time.Now()
	}

	// Validar intervalo máximo (7 dias)
	if endTime.Sub(startTime) > 7*24*time.Hour {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "Time range cannot exceed 7 days",
		})
		return
	}

	// Context para operações de banco
	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Buscar histórico do banco
	points, err := c.repo.GetHistory(dbCtx, deviceID, startTime, endTime)
	if err != nil {
		logger.Error("Telemetry", "Failed to get history for export: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve history",
		})
		return
	}

	// Gerar CSV
	csvBuilder := &strings.Builder{}
	csvBuilder.WriteString("device_id,lat,lng,speed,recorded_at\n")

	for _, point := range points {
		csvBuilder.WriteString(fmt.Sprintf("%s,%.6f,%.6f,%.2f,%s\n",
			deviceID,
			point.Lat,
			point.Lng,
			point.Speed,
			point.RecordedAt.Format(time.RFC3339),
		))
	}

	// Configurar headers para download CSV
	ctx.Header("Content-Type", "text/csv")
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s_history_%s.csv",
		deviceID,
		time.Now().Format("20060102_150405")))

	// Log para analytics
	processingTime := time.Since(start)
	logger.Info("Telemetry", "Exported %d history points for device %s in %v", len(points), deviceID, processingTime)

	// Retornar CSV
	ctx.String(http.StatusOK, csvBuilder.String())
}

// GetFleetStatus retorna métricas gerenciais da frota para o Centro de Controle Operacional.
//
// GET /api/v1/telemetry/fleet-status
//
// Requer JWT authentication.
// Calcula métricas de eficiência, tempo em movimento, dwell time em terminais.
//
// Parameters:
//   - ctx: Contexto Gin com a requisição HTTP
//
// Returns:
//   - 200 OK: Com métricas da frota
//   - 401 Unauthorized: Se JWT não for fornecido ou inválido
func (c *Controller) GetFleetStatus(ctx *gin.Context) {
	start := time.Now()

	// Context para operações de banco
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Buscar status da frota
	status, err := c.repo.GetFleetStatus(dbCtx)
	if err != nil {
		logger.Error("Telemetry", "Failed to get fleet status: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve fleet status",
		})
		return
	}

	// Log para analytics
	processingTime := time.Since(start)
	logger.Info("Telemetry", "Fleet status retrieved in %v", processingTime)

	// Resposta
	ctx.JSON(http.StatusOK, status)
}
