package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
	"github.com/gin-gonic/gin"
)

// ============================================================================
// CX & MONETIZATION HANDLERS
// ============================================================================

// CalculateETAWithConfidence calcula ETA com intervalo de confiança
// GET /api/v1/telemetry/eta-confidence/:device_hash?lat=&lng=
func (c *Controller) CalculateETAWithConfidence(ctx *gin.Context) {
	start := time.Now()

	// Parâmetros
	deviceHash := ctx.Param("device_hash")
	destLatStr := ctx.Query("lat")
	destLngStr := ctx.Query("lng")

	// Validação básica
	if deviceHash == "" || destLatStr == "" || destLngStr == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters"})
		return
	}

	// Parse coordinates (validação simplificada)
	var destLat, destLng float64
	if _, err := fmt.Sscanf(destLatStr, "%f", &destLat); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid latitude"})
		return
	}
	if _, err := fmt.Sscanf(destLngStr, "%f", &destLng); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid longitude"})
		return
	}

	// Contexto com timeout para operações rápidas
	dbCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond) // 50ms timeout para CX
	defer cancel()

	// Calcular ETA com confiança
	eta, err := c.repo.GetETAWithConfidence(dbCtx, deviceHash, destLat, destLng)
	if err != nil {
		logger.Error("CX", "Failed to calculate ETA with confidence: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to calculate ETA"})
		return
	}

	// Buscar status Premium do usuário (se user_id disponível no contexto)
	userID := "anonymous"
	if uid, exists := ctx.Get("user_id"); exists {
		userID = uid.(string)
		userStatus, err := c.repo.GetUserStatus(dbCtx, userID)
		if err == nil {
			eta.IsPremium = userStatus.IsPremium
		}
	}

	// Resposta otimizada para mobile (payload mínimo)
	elapsedMs := time.Since(start).Milliseconds()
	response := gin.H{
		"eta_min":      eta.EstimatedArrivalMin,
		"confidence":   eta.ConfidencePercent,
		"lower_bound":  eta.LowerBoundMin,
		"upper_bound":  eta.UpperBoundMin,
		"message":      eta.Message,
		"friendly_msg": eta.FriendlyMessage,
		"is_premium":   eta.IsPremium,
		"latency_ms":   elapsedMs,
	}

	ctx.JSON(http.StatusOK, response)
}

// GetUserStatus retorna status do usuário para monetização
// GET /api/v1/user/status
func (c *Controller) GetUserStatus(ctx *gin.Context) {
	// Extrair user_id do contexto JWT
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	status, err := c.repo.GetUserStatus(dbCtx, userID.(string))
	if err != nil {
		logger.Error("CX", "Failed to get user status: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user status"})
		return
	}

	ctx.JSON(http.StatusOK, status)
}

// ToggleAdFree alterna modo ad-free para usuários Premium
// POST /api/v1/user/ad-free-toggle
func (c *Controller) ToggleAdFree(ctx *gin.Context) {
	userID, exists := ctx.Get("user_id")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Em MVP, apenas log e retorna status mock
	logger.Info("CX", "Ad-free toggle requested | User: %s", userID)

	ctx.JSON(http.StatusOK, gin.H{
		"ad_free": true,
		"message": "Modo ad-free ativado (Premium)",
	})
}

// RecordCheckIn registra check-in do usuário (gamificação)
// POST /api/v1/gamification/check-in
func (c *Controller) RecordCheckIn(ctx *gin.Context) {
	var checkIn CheckInRequest

	if err := ctx.ShouldBindJSON(&checkIn); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	response, err := c.repo.RecordCheckIn(dbCtx, &checkIn)
	if err != nil {
		logger.Error("CX", "Failed to record check-in: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record check-in"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// ReportOccupancy registra lotação do ônibus (gamificação)
// POST /api/v1/gamification/occupancy
func (c *Controller) ReportOccupancy(ctx *gin.Context) {
	var report OccupancyReport

	if err := ctx.ShouldBindJSON(&report); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Set timestamp
	report.Timestamp = time.Now()

	dbCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	if err := c.repo.RecordOccupancyReport(dbCtx, &report); err != nil {
		logger.Error("CX", "Failed to record occupancy: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to record occupancy"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"success":       true,
		"points_earned": 5,
		"message":       "Obrigado por reportar! +5 pontos",
	})
}

// GetLocalizedMessages retorna mensagens localizadas
// GET /api/v1/messages?lang=pt-BR
func (c *Controller) GetLocalizedMessages(ctx *gin.Context) {
	language := ctx.Query("lang")
	if language == "" {
		language = "pt-BR" // Padrão para Goiânia
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	messages, err := c.repo.GetLocalizedMessages(dbCtx, language)
	if err != nil {
		logger.Error("CX", "Failed to get localized messages: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get messages"})
		return
	}

	ctx.JSON(http.StatusOK, messages)
}

// GetAdSlots retorna configuração de anúncios (para Premium users, retorna vazio)
// GET /api/v1/ads/slots
func (c *Controller) GetAdSlots(ctx *gin.Context) {
	// Verificar se usuário é Premium
	userID, exists := ctx.Get("user_id")
	if !exists {
		// Usuário anônimo: retorna slots de anúncio
		ctx.JSON(http.StatusOK, gin.H{
			"slots": []AdSlot{
				{Position: "top", Enabled: true, Size: "banner"},
				{Position: "bottom", Enabled: true, Size: "banner"},
			},
		})
		return
	}

	dbCtx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	status, err := c.repo.GetUserStatus(dbCtx, userID.(string))
	if err != nil || !status.IsPremium || !status.AdFree {
		// Usuário Free ou Ad-Free desativado
		ctx.JSON(http.StatusOK, gin.H{
			"slots": []AdSlot{
				{Position: "top", Enabled: true, Size: "banner"},
				{Position: "bottom", Enabled: true, Size: "banner"},
			},
		})
		return
	}

	// Usuário Premium com Ad-Free ativado
	ctx.JSON(http.StatusOK, gin.H{
		"slots": []AdSlot{}, // Vazio para Premium
	})
}
