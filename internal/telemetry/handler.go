package telemetry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Controller gerencia operações de telemetria
type Controller struct {
	repo  *Repository
	redis *redis.Client
}

// NewController cria um novo controller
func NewController(db *sql.DB, rdb *redis.Client) *Controller {
	return &Controller{
		repo:  NewRepository(db),
		redis: rdb,
	}
}

// ReceiveGPSPing - POST /api/v1/telemetry/gps
// Recebe dados de telemetria GPS do usuário com validação e anonimização
func (c *Controller) ReceiveGPSPing(ctx *gin.Context) {
	start := time.Now()

	var ping TelemetryPing
	if err := ctx.ShouldBindJSON(&ping); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid JSON payload",
			"details": err.Error(),
		})
		return
	}

	// Validações de segurança
	validationErrors := c.validatePing(&ping)
	if len(validationErrors) > 0 {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":       "Validation failed",
			"validations": validationErrors,
		})
		return
	}

	// Anonimização LGPD: hash do device_id com sal diário
	deviceHash := c.anonymizeDeviceID(ping.DeviceID)

	// Context para operações de banco
	dbCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Salvar no PostgreSQL/PostGIS
	telemetryID, err := c.repo.SaveToDatabase(dbCtx, deviceHash, &ping)
	if err != nil {
		log.Printf("[Telemetry] Database error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to store telemetry data",
		})
		return
	}

	// 2. Atualizar cache Redis (última posição conhecida)
	err = c.updateRedisCache(dbCtx, deviceHash, &ping)
	if err != nil {
		// Log error but don't fail the request (cache is best-effort)
		log.Printf("[Telemetry] Redis cache error: %v", err)
	}

	// Calcular tempo de processamento
	processingTime := time.Since(start)

	// Log para analytics (async)
	go c.logTelemetryReceived(deviceHash, ping.RouteID, processingTime)

	// Resposta
	response := TelemetryResponse{
		Status:      "accepted",
		TelemetryID: telemetryID,
		DeviceHash:  deviceHash,
		Cached:      err == nil,
		ProcessedAt: time.Now(),
		TTL:         int(RedisLastPosTTL.Seconds()),
	}

	ctx.JSON(http.StatusCreated, response)
}

// GetLatestPositions - GET /api/v1/telemetry/latest
// Retorna a última posição conhecida de cada dispositivo (ônibus)
func (c *Controller) GetLatestPositions(ctx *gin.Context) {
	start := time.Now()

	// Context para operações de banco
	dbCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Buscar todas as últimas posições do banco
	positions, err := c.repo.GetAllLatestPositions(dbCtx)
	if err != nil {
		log.Printf("[Telemetry] Failed to get latest positions: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve latest positions",
		})
		return
	}

	// Log para analytics
	processingTime := time.Since(start)
	log.Printf("[Telemetry] Retrieved %d latest positions in %v", len(positions), processingTime)

	// Resposta
	ctx.JSON(http.StatusOK, gin.H{
		"count":     len(positions),
		"positions": positions,
	})
}

// GetLastPosition - GET /api/v1/telemetry/last-position/:device_hash
// Retorna a última posição conhecida de um dispositivo (do Redis ou DB)
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

// ============================================================================
// ANONIMIZAÇÃO LGPD
// ============================================================================

// anonymizeDeviceID cria hash SHA-256 com sal diário
func (c *Controller) anonymizeDeviceID(deviceID string) string {
	// Gerar sal baseado na data (rotação diária)
	salt := c.getDailySalt()

	// Criar hash SHA-256
	hasher := sha256.New()
	hasher.Write([]byte(deviceID + salt))
	hash := hex.EncodeToString(hasher.Sum(nil))

	// Retornar primeiros 32 caracteres (suficiente para unicidade)
	return hash[:32]
}

// getDailySalt retorna sal baseado na data atual
func (c *Controller) getDailySalt() string {
	// Formato: YYYY-MM-DD (muda a cada 24h)
	return time.Now().UTC().Format("2006-01-02")
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
	log.Printf("[Analytics] %s", string(jsonLog))
}

// ============================================================================
// UTILITÁRIOS
// ============================================================================

// GetActiveDevicesCount retorna número de dispositivos ativos (últimos 5 min)
func (c *Controller) GetActiveDevicesCount(ctx context.Context) (int64, error) {
	if c.redis == nil {
		return 0, fmt.Errorf("redis not available")
	}

	return c.redis.SCard(ctx, "telemetry:active_devices").Result()
}
