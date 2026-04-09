package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// AlertType representa diferentes tipos de alertas
type AlertType string

const (
	AlertStorm     AlertType = "storm"
	AlertHeavyRain AlertType = "heavy_rain"
	AlertHeatWave  AlertType = "heat_wave"
	AlertFlood     AlertType = "flood"
	AlertSystem    AlertType = "system"
	AlertTraffic   AlertType = "traffic"
)

// AlertSeverity representa a severidade do alerta
type AlertSeverity string

const (
	SeverityLow      AlertSeverity = "low"
	SeverityMedium   AlertSeverity = "medium"
	SeverityHigh     AlertSeverity = "high"
	SeverityCritical AlertSeverity = "critical"
)

// WebhookPayload representa o payload de um webhook
type WebhookPayload struct {
	ID        string        `json:"id"`
	Type      AlertType     `json:"type"`
	Severity  AlertSeverity `json:"severity"`
	Title     string        `json:"title"`
	Message   string        `json:"message"`
	Data      interface{}   `json:"data"`
	Timestamp time.Time     `json:"timestamp"`
	Location  string        `json:"location"`
	Impact    string        `json:"impact"`
	Actions   []AlertAction `json:"actions"`
}

// AlertAction representa uma ação que pode ser tomada
type AlertAction struct {
	Type        string `json:"type"`
	Label       string `json:"label"`
	URL         string `json:"url,omitempty"`
	Method      string `json:"method,omitempty"`
	Description string `json:"description"`
}

// WebhookSubscription representa uma assinatura de webhook
type WebhookSubscription struct {
	ID          string            `json:"id"`
	URL         string            `json:"url"`
	Events      []AlertType       `json:"events"`
	Secret      string            `json:"secret"`
	Headers     map[string]string `json:"headers"`
	Active      bool              `json:"active"`
	CreatedAt   time.Time         `json:"created_at"`
	LastTrigger time.Time         `json:"last_trigger"`
}

// AlertManager gerencia todos os alertas e webhooks
type AlertManager struct {
	app           *App
	subscriptions map[string]*WebhookSubscription
	subMutex      sync.RWMutex
	alertHistory  []WebhookPayload
	alertMutex    sync.RWMutex
}

// NewAlertManager cria um novo gerenciador de alertas
func NewAlertManager(app *App) *AlertManager {
	manager := &AlertManager{
		app:           app,
		subscriptions: make(map[string]*WebhookSubscription),
		alertHistory:  make([]WebhookPayload, 0),
	}

	// Iniciar monitoramento de alertas
	go manager.startAlertMonitoring()

	return manager
}

// startAlertMonitoring inicia monitoramento contínuo de alertas
func (am *AlertManager) startAlertMonitoring() {
	ticker := time.NewTicker(5 * time.Minute) // Verificar a cada 5 min
	defer ticker.Stop()

	for range ticker.C {
		am.checkWeatherAlerts()
		am.checkSystemAlerts()
		am.checkTrafficAlerts()
	}
}

// checkWeatherAlerts verifica alertas climáticos
func (am *AlertManager) checkWeatherAlerts() {
	weather, exists := am.app.getWeatherData("goiania")
	if !exists {
		return
	}

	// Alerta de tempestade
	if weather.Description == "tempestade" || strings.Contains(strings.ToLower(weather.Description), "storm") {
		alert := WebhookPayload{
			ID:        generateAlertID(),
			Type:      AlertStorm,
			Severity:  SeverityHigh,
			Title:     "Alerta de Tempestade em Goiânia",
			Message:   fmt.Sprintf("Tempestade detectada! Temperatura: %.1f°C, Umidade: %.1f%%", weather.Temperature, weather.Humidity),
			Data:      weather,
			Timestamp: time.Now(),
			Location:  "Goiânia, GO",
			Impact:    "Aumento de 10min em todas as rotas. Transporte público pode ser afetado.",
			Actions: []AlertAction{
				{
					Type:        "view_routes",
					Label:       "Ver Rotas Afetadas",
					Description: "Visualizar rotas com ajuste climático",
				},
				{
					Type:        "weather_details",
					Label:       "Detalhes do Clima",
					Description: "Ver informações detalhadas do clima",
				},
			},
		}
		am.triggerAlert(alert)
	}

	// Alerta de chuva forte
	if weather.Humidity > 85 && weather.Temperature > 20 {
		alert := WebhookPayload{
			ID:        generateAlertID(),
			Type:      AlertHeavyRain,
			Severity:  SeverityMedium,
			Title:     "Chuva Forte em Goiânia",
			Message:   fmt.Sprintf("Chuva forte detectada! Umidade: %.1f%%", weather.Humidity),
			Data:      weather,
			Timestamp: time.Now(),
			Location:  "Goiânia, GO",
			Impact:    "Possível aumento de 5min nos tempos de viagem.",
			Actions: []AlertAction{
				{
					Type:        "adjust_routes",
					Label:       "Ajustar Rotas",
					Description: "Recalcular rotas com clima",
				},
			},
		}
		am.triggerAlert(alert)
	}

	// Alerta de onda de calor
	if weather.Temperature > 35 {
		alert := WebhookPayload{
			ID:        generateAlertID(),
			Type:      AlertHeatWave,
			Severity:  SeverityMedium,
			Title:     "Onda de Calor em Goiânia",
			Message:   fmt.Sprintf("Temperatura elevada! %.1f°C", weather.Temperature),
			Data:      weather,
			Timestamp: time.Now(),
			Location:  "Goiânia, GO",
			Impact:    "Recomendado usar transporte com ar condicionado. Hidrate-se!",
			Actions: []AlertAction{
				{
					Type:        "shelter_routes",
					Label:       "Rotas com Abrigo",
					Description: "Ver rotas com pontos de espera cobertos",
				},
			},
		}
		am.triggerAlert(alert)
	}
}

// checkSystemAlerts verifica alertas do sistema
func (am *AlertManager) checkSystemAlerts() {
	// Verificar disponibilidade do banco de dados
	if am.app.db == nil {
		alert := WebhookPayload{
			ID:       generateAlertID(),
			Type:     AlertSystem,
			Severity: SeverityHigh,
			Title:    "Banco de Dados Offline",
			Message:  "Banco de dados principal está indisponível. Operando em modo degradado.",
			Data: map[string]interface{}{
				"mode":     "offline",
				"fallback": true,
			},
			Timestamp: time.Now(),
			Location:  "Sistema",
			Impact:    "Funcionalidades limitadas. Cache e fallback ativos.",
			Actions: []AlertAction{
				{
					Type:        "system_status",
					Label:       "Ver Status",
					Description: "Verificar status do sistema",
				},
			},
		}
		am.triggerAlert(alert)
	}

	// Verificar disponibilidade do Redis
	if am.app.rdb == nil {
		alert := WebhookPayload{
			ID:       generateAlertID(),
			Type:     AlertSystem,
			Severity: SeverityMedium,
			Title:    "Cache Offline",
			Message:  "Redis está indisponível. Performance pode ser afetada.",
			Data: map[string]interface{}{
				"cache_status": "offline",
			},
			Timestamp: time.Now(),
			Location:  "Sistema",
			Impact:    "Tempo de resposta aumentado. Funcionalidades preservadas.",
			Actions: []AlertAction{
				{
					Type:        "check_cache",
					Label:       "Verificar Cache",
					Description: "Verificar status do cache",
				},
			},
		}
		am.triggerAlert(alert)
	}
}

// checkTrafficAlerts verifica alertas de trânsito
func (am *AlertManager) checkTrafficAlerts() {
	// Simulação de análise de trânsito
	now := time.Now()
	hour := now.Hour()

	// Horários de pico conhecidos em Goiânia
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		alert := WebhookPayload{
			ID:       generateAlertID(),
			Type:     AlertTraffic,
			Severity: SeverityMedium,
			Title:    "Horário de Pico Detectado",
			Message:  fmt.Sprintf("Trânsito intenso esperado para %d:00h", hour),
			Data: map[string]interface{}{
				"hour":         hour,
				"is_rush_hour": true,
			},
			Timestamp: time.Now(),
			Location:  "Goiânia, GO",
			Impact:    "Aumento de 15-20% nos tempos de viagem. Considere alternativas.",
			Actions: []AlertAction{
				{
					Type:        "alternative_routes",
					Label:       "Rotas Alternativas",
					Description: "Ver opções de rota alternativas",
				},
				{
					Type:        "walkability_check",
					Label:       "Ver Caminhabilidade",
					Description: "Verificar se vale a pena andar",
				},
			},
		}
		am.triggerAlert(alert)
	}
}

// triggerAlert dispara um alerta para todos os assinantes
func (am *AlertManager) triggerAlert(alert WebhookPayload) {
	am.alertMutex.Lock()
	am.alertHistory = append(am.alertHistory, alert)
	// Manter apenas os últimos 100 alertas no histórico
	if len(am.alertHistory) > 100 {
		am.alertHistory = am.alertHistory[1:]
	}
	am.alertMutex.Unlock()

	am.subMutex.RLock()
	subscriptions := make([]*WebhookSubscription, 0, len(am.subscriptions))
	for _, sub := range am.subscriptions {
		if sub.Active && am.shouldTriggerSubscription(sub, alert.Type) {
			subscriptions = append(subscriptions, sub)
		}
	}
	am.subMutex.RUnlock()

	// Disparar webhooks em paralelo
	var wg sync.WaitGroup
	for _, sub := range subscriptions {
		wg.Add(1)
		go func(subscription *WebhookSubscription) {
			defer wg.Done()
			am.sendWebhook(subscription, alert)
		}(sub)
	}
	wg.Wait()

	log.Printf("Alerta disparado: %s - %s", alert.Type, alert.Title)
}

// shouldTriggerSubscription verifica se o alerta deve ser disparado para a assinatura
func (am *AlertManager) shouldTriggerSubscription(sub *WebhookSubscription, alertType AlertType) bool {
	for _, event := range sub.Events {
		if event == alertType || event == AlertType("") { // Assinatura para todos os eventos
			return true
		}
	}
	return false
}

// sendWebhook envia o webhook para uma URL
func (am *AlertManager) sendWebhook(sub *WebhookSubscription, alert WebhookPayload) {
	jsonData, err := json.Marshal(alert)
	if err != nil {
		log.Printf("Erro ao serializar alerta: %v", err)
		return
	}

	req, err := http.NewRequest("POST", sub.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Erro ao criar requisição webhook: %v", err)
		return
	}

	// Configurar headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TranspRota-Webhook/1.0")
	req.Header.Set("X-TranspRota-Alert-ID", alert.ID)
	req.Header.Set("X-TranspRota-Alert-Type", string(alert.Type))
	req.Header.Set("X-TranspRota-Alert-Severity", string(alert.Severity))

	// Adicionar headers customizados
	for key, value := range sub.Headers {
		req.Header.Set(key, value)
	}

	// Adicionar assinatura se secret foi configurado
	if sub.Secret != "" {
		signature := generateSignature(jsonData, sub.Secret)
		req.Header.Set("X-TranspRota-Signature", signature)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Erro ao enviar webhook para %s: %v", sub.URL, err)
		return
	}
	defer resp.Body.Close()

	// Atualizar último disparo
	am.subMutex.Lock()
	sub.LastTrigger = time.Now()
	am.subMutex.Unlock()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Webhook enviado com sucesso para %s (status: %d)", sub.URL, resp.StatusCode)
	} else {
		log.Printf("Webhook falhou para %s (status: %d)", sub.URL, resp.StatusCode)
	}
}

// generateSignature gera assinatura HMAC para webhook
func generateSignature(data []byte, secret string) string {
	// Simplificado - em produção usar HMAC-SHA256
	return fmt.Sprintf("sha256=%x", data) // Placeholder
}

// generateAlertID gera um ID único para o alerta
func generateAlertID() string {
	return fmt.Sprintf("alert_%d", time.Now().UnixNano())
}

// SubscribeWebhook adiciona uma nova assinatura de webhook
func (am *AlertManager) SubscribeWebhook(url string, events []AlertType, secret string, headers map[string]string) (*WebhookSubscription, error) {
	sub := &WebhookSubscription{
		ID:        fmt.Sprintf("webhook_%d", time.Now().UnixNano()),
		URL:       url,
		Events:    events,
		Secret:    secret,
		Headers:   headers,
		Active:    true,
		CreatedAt: time.Now(),
	}

	am.subMutex.Lock()
	am.subscriptions[sub.ID] = sub
	am.subMutex.Unlock()

	log.Printf("Webhook assinado: %s para eventos %v", url, events)
	return sub, nil
}

// UnsubscribeWebhook remove uma assinatura de webhook
func (am *AlertManager) UnsubscribeWebhook(id string) error {
	am.subMutex.Lock()
	defer am.subMutex.Unlock()

	if _, exists := am.subscriptions[id]; !exists {
		return fmt.Errorf("webhook não encontrado")
	}

	delete(am.subscriptions, id)
	log.Printf("Webhook removido: %s", id)
	return nil
}

// GetAlertHistory obtém o histórico de alertas
func (am *AlertManager) GetAlertHistory(limit int) []WebhookPayload {
	am.alertMutex.RLock()
	defer am.alertMutex.RUnlock()

	if limit <= 0 || limit > len(am.alertHistory) {
		limit = len(am.alertHistory)
	}

	// Retornar os alertas mais recentes
	start := len(am.alertHistory) - limit
	if start < 0 {
		start = 0
	}

	result := make([]WebhookPayload, limit)
	copy(result, am.alertHistory[start:])
	return result
}

// GetActiveWebhooks obtém todas as assinaturas ativas
func (am *AlertManager) GetActiveWebhooks() []*WebhookSubscription {
	am.subMutex.RLock()
	defer am.subMutex.RUnlock()

	webhooks := make([]*WebhookSubscription, 0, len(am.subscriptions))
	for _, sub := range am.subscriptions {
		if sub.Active {
			webhooks = append(webhooks, sub)
		}
	}

	return webhooks
}

// setupWebhookRoutes configura as rotas do sistema de webhooks
func setupWebhookRoutes(r *gin.Engine, app *App) {
	alertManager := NewAlertManager(app)

	// POST /api/v1/webhooks/subscribe - Assinar webhook
	r.POST("/api/v1/webhooks/subscribe", func(c *gin.Context) {
		var req struct {
			URL     string            `json:"url" binding:"required"`
			Events  []AlertType       `json:"events"`
			Secret  string            `json:"secret"`
			Headers map[string]string `json:"headers"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
			return
		}

		subscription, err := alertManager.SubscribeWebhook(req.URL, req.Events, req.Secret, req.Headers)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Erro ao assinar webhook"})
			return
		}

		c.JSON(http.StatusCreated, subscription)
	})

	// DELETE /api/v1/webhooks/:id/unsubscribe - Cancelar assinatura
	r.DELETE("/api/v1/webhooks/:id/unsubscribe", func(c *gin.Context) {
		id := c.Param("id")

		err := alertManager.UnsubscribeWebhook(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Webhook removido com sucesso"})
	})

	// GET /api/v1/webhooks - Listar webhooks ativos
	r.GET("/api/v1/webhooks", func(c *gin.Context) {
		webhooks := alertManager.GetActiveWebhooks()
		c.JSON(http.StatusOK, gin.H{
			"webhooks": webhooks,
			"count":    len(webhooks),
		})
	})

	// GET /api/v1/alerts/history - Histórico de alertas
	r.GET("/api/v1/alerts/history", func(c *gin.Context) {
		limit := 50 // padrão
		if l := c.Query("limit"); l != "" {
			if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
				limit = parsed
			}
		}

		alerts := alertManager.GetAlertHistory(limit)
		c.JSON(http.StatusOK, gin.H{
			"alerts": alerts,
			"count":  len(alerts),
		})
	})

	// POST /api/v1/alerts/test - Testar envio de alerta
	r.POST("/api/v1/alerts/test", func(c *gin.Context) {
		testAlert := WebhookPayload{
			ID:        generateAlertID(),
			Type:      AlertSystem,
			Severity:  SeverityLow,
			Title:     "Alerta de Teste",
			Message:   "Este é um alerta de teste do sistema TranspRota",
			Data:      map[string]interface{}{"test": true},
			Timestamp: time.Now(),
			Location:  "Sistema",
			Impact:    "Nenhum impacto - apenas teste",
			Actions: []AlertAction{
				{
					Type:        "test_action",
					Label:       "Ação de Teste",
					Description: "Esta é uma ação de teste",
				},
			},
		}

		alertManager.triggerAlert(testAlert)

		c.JSON(http.StatusOK, gin.H{
			"message":  "Alerta de teste disparado com sucesso",
			"alert_id": testAlert.ID,
		})
	})

	// GET /api/v1/alerts/active - Alertas ativos recentes
	r.GET("/api/v1/alerts/active", func(c *gin.Context) {
		// Alertas considerados "ativos" se foram disparados nos últimos 30 minutos
		recentAlerts := []WebhookPayload{}
		thirtyMinutesAgo := time.Now().Add(-30 * time.Minute)

		alerts := alertManager.GetAlertHistory(20)
		for _, alert := range alerts {
			if alert.Timestamp.After(thirtyMinutesAgo) {
				recentAlerts = append(recentAlerts, alert)
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"active_alerts": recentAlerts,
			"count":         len(recentAlerts),
		})
	})
}

// parseInt converte string para int com fallback
func parseInt(s string) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
