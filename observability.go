package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics representa métricas do sistema
type Metrics struct {
	Timestamp           time.Time      `json:"timestamp"`
	InstanceID          string         `json:"instance_id"`
	RequestsTotal       int64          `json:"requests_total"`
	RequestsPerSecond   float64        `json:"requests_per_second"`
	ResponseTime        float64        `json:"response_time_ms"`
	ErrorRate           float64        `json:"error_rate_percent"`
	CPUUsage            float64        `json:"cpu_usage_percent"`
	MemoryUsage         float64        `json:"memory_usage_mb"`
	GoRoutines          int            `json:"goroutines"`
	RedisHitRate        float64        `json:"redis_hit_rate_percent"`
	DatabaseConnections int            `json:"database_connections"`
	ClusterStatus       ClusterMetrics `json:"cluster_status"`
}

// ClusterMetrics representa métricas do cluster
type ClusterMetrics struct {
	TotalNodes    int          `json:"total_nodes"`
	HealthyNodes  int          `json:"healthy_nodes"`
	LoadBalancer  string       `json:"load_balancer"`
	FailoverCount int64        `json:"failover_count"`
	LastFailover  time.Time    `json:"last_failover"`
	NodeStatus    []NodeStatus `json:"node_status"`
}

// NodeStatus representa status de um nó do cluster
type NodeStatus struct {
	NodeID       string    `json:"node_id"`
	Status       string    `json:"status"`
	LastSeen     time.Time `json:"last_seen"`
	Requests     int64     `json:"requests"`
	ResponseTime float64   `json:"response_time_ms"`
}

// PrometheusMetrics representa métricas no formato Prometheus
type PrometheusMetrics struct {
	Metrics []PrometheusMetric `json:"metrics"`
}

// PrometheusMetric representa uma métrica Prometheus
type PrometheusMetric struct {
	Name   string            `json:"name"`
	Type   string            `json:"type"`
	Help   string            `json:"help"`
	Values []PrometheusValue `json:"values"`
}

// PrometheusValue representa um valor de métrica Prometheus
type PrometheusValue struct {
	Labels    map[string]string `json:"labels"`
	Value     float64           `json:"value"`
	Timestamp int64             `json:"timestamp"`
}

// ObservabilityManager gerencia observabilidade
type ObservabilityManager struct {
	instanceID      string
	startTime       time.Time
	requestCount    int64
	errorCount      int64
	responseTimeSum int64
	lastSecondCount int64
	lastSecondTime  time.Time
	failoverCount   int64
}

// NewObservabilityManager cria novo gerenciador de observabilidade
func NewObservabilityManager() *ObservabilityManager {
	instanceID := "transprota-api"
	if id := os.Getenv("INSTANCE_ID"); id != "" {
		instanceID = id
	}

	return &ObservabilityManager{
		instanceID:     instanceID,
		startTime:      time.Now(),
		lastSecondTime: time.Now(),
	}
}

// RecordRequest registra uma requisição
func (om *ObservabilityManager) RecordRequest(responseTime time.Duration, isError bool) {
	atomic.AddInt64(&om.requestCount, 1)
	atomic.AddInt64(&om.responseTimeSum, int64(responseTime.Milliseconds()))

	if isError {
		atomic.AddInt64(&om.errorCount, 1)
	}
}

// RecordFailover registra um failover
func (om *ObservabilityManager) RecordFailover() {
	atomic.AddInt64(&om.failoverCount, 1)
}

// GetMetrics retorna métricas atuais
func (om *ObservabilityManager) GetMetrics(app *App) *Metrics {
	now := time.Now()

	// Calcular requisições por segundo
	var rps float64
	atomic.StoreInt64(&om.lastSecondCount, om.requestCount)
	rps = float64(om.requestCount) / time.Since(om.startTime).Seconds()

	// Calcular tempo médio de resposta
	var avgResponseTime float64
	if om.requestCount > 0 {
		avgResponseTime = float64(om.responseTimeSum) / float64(om.requestCount)
	}

	// Calcular taxa de erro
	var errorRate float64
	if om.requestCount > 0 {
		errorRate = float64(om.errorCount) / float64(om.requestCount) * 100
	}

	// Obter uso de memória
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsageMB := float64(m.Alloc) / 1024 / 1024

	// Obter status do Redis
	var redisHitRate float64
	if app.rdb != nil {
		// Simulação - em produção usar INFO do Redis
		redisHitRate = 85.5
	}

	// Obter status do cluster
	clusterMetrics := om.getClusterMetrics()

	return &Metrics{
		Timestamp:           now,
		InstanceID:          om.instanceID,
		RequestsTotal:       om.requestCount,
		RequestsPerSecond:   rps,
		ResponseTime:        avgResponseTime,
		ErrorRate:           errorRate,
		CPUUsage:            0, // Implementar com runtime/debug se necessário
		MemoryUsage:         memoryUsageMB,
		GoRoutines:          runtime.NumGoroutine(),
		RedisHitRate:        redisHitRate,
		DatabaseConnections: 10, // Simulação
		ClusterStatus:       clusterMetrics,
	}
}

// getClusterMetrics obtém métricas do cluster
func (om *ObservabilityManager) getClusterMetrics() ClusterMetrics {
	// Simulação - em produção consultar outros nós via API
	return ClusterMetrics{
		TotalNodes:    2,
		HealthyNodes:  2,
		LoadBalancer:  "nginx",
		FailoverCount: om.failoverCount,
		LastFailover:  time.Now().Add(-1 * time.Hour), // Simulação
		NodeStatus: []NodeStatus{
			{
				NodeID:       "api-1",
				Status:       "healthy",
				LastSeen:     time.Now().Add(-10 * time.Second),
				Requests:     atomic.LoadInt64(&om.requestCount),
				ResponseTime: 18.7,
			},
			{
				NodeID:       "api-2",
				Status:       "healthy",
				LastSeen:     time.Now().Add(-5 * time.Second),
				Requests:     50000, // Simulação
				ResponseTime: 19.2,
			},
		},
	}
}

// GetPrometheusMetrics retorna métricas no formato Prometheus
func (om *ObservabilityManager) GetPrometheusMetrics(app *App) *PrometheusMetrics {
	metrics := om.GetMetrics(app)

	promMetrics := &PrometheusMetrics{
		Metrics: []PrometheusMetric{
			{
				Name: "transprota_requests_total",
				Type: "counter",
				Help: "Total number of requests",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"instance": metrics.InstanceID,
							"method":   "GET",
							"status":   "200",
						},
						Value:     float64(metrics.RequestsTotal),
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
			{
				Name: "transprota_requests_per_second",
				Type: "gauge",
				Help: "Requests per second",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"instance": metrics.InstanceID,
						},
						Value:     metrics.RequestsPerSecond,
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
			{
				Name: "transprota_response_time_ms",
				Type: "histogram",
				Help: "Response time in milliseconds",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"instance": metrics.InstanceID,
							"quantile": "0.95",
						},
						Value:     metrics.ResponseTime,
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
			{
				Name: "transprota_error_rate_percent",
				Type: "gauge",
				Help: "Error rate percentage",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"instance": metrics.InstanceID,
						},
						Value:     metrics.ErrorRate,
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
			{
				Name: "transprota_memory_usage_mb",
				Type: "gauge",
				Help: "Memory usage in MB",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"instance": metrics.InstanceID,
						},
						Value:     metrics.MemoryUsage,
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
			{
				Name: "transprota_goroutines",
				Type: "gauge",
				Help: "Number of goroutines",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"instance": metrics.InstanceID,
						},
						Value:     float64(metrics.GoRoutines),
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
			{
				Name: "transprota_redis_hit_rate_percent",
				Type: "gauge",
				Help: "Redis cache hit rate percentage",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"instance": metrics.InstanceID,
						},
						Value:     metrics.RedisHitRate,
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
			{
				Name: "transprota_cluster_nodes_total",
				Type: "gauge",
				Help: "Total number of cluster nodes",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"cluster": "transprota",
						},
						Value:     float64(metrics.ClusterStatus.TotalNodes),
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
			{
				Name: "transprota_cluster_nodes_healthy",
				Type: "gauge",
				Help: "Number of healthy cluster nodes",
				Values: []PrometheusValue{
					{
						Labels: map[string]string{
							"cluster": "transprota",
						},
						Value:     float64(metrics.ClusterStatus.HealthyNodes),
						Timestamp: metrics.Timestamp.Unix(),
					},
				},
			},
		},
	}

	return promMetrics
}

// formatPrometheusMetrics formata métricas no formato Prometheus
func (pm *PrometheusMetrics) FormatPrometheus() string {
	var builder strings.Builder

	for _, metric := range pm.Metrics {
		builder.WriteString(fmt.Sprintf("# HELP %s %s\n", metric.Name, metric.Help))
		builder.WriteString(fmt.Sprintf("# TYPE %s %s\n", metric.Name, metric.Type))

		for _, value := range metric.Values {
			labels := ""
			if len(value.Labels) > 0 {
				var labelPairs []string
				for k, v := range value.Labels {
					labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, k, v))
				}
				labels = fmt.Sprintf("{%s}", strings.Join(labelPairs, ","))
			}

			builder.WriteString(fmt.Sprintf("%s%s %f %d\n", metric.Name, labels, value.Value, value.Timestamp))
		}

		builder.WriteString("\n")
	}

	return builder.String()
}

// MetricsMiddleware middleware para coletar métricas
func (om *ObservabilityManager) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Processar requisição
		c.Next()

		// Registrar métricas
		responseTime := time.Since(start)
		isError := c.Writer.Status() >= 400

		om.RecordRequest(responseTime, isError)
	}
}

// setupObservabilityRoutes configura rotas de observabilidade
func setupObservabilityRoutes(r *gin.Engine, app *App) {
	obsManager := NewObservabilityManager()

	// Aplicar middleware de métricas globalmente
	r.Use(obsManager.MetricsMiddleware())

	// GET /api/v1/metrics - Métricas em formato JSON
	r.GET("/api/v1/metrics", func(c *gin.Context) {
		metrics := obsManager.GetMetrics(app)
		c.JSON(http.StatusOK, metrics)
	})

	// GET /metrics - Métricas no formato Prometheus
	r.GET("/metrics", func(c *gin.Context) {
		promMetrics := obsManager.GetPrometheusMetrics(app)

		// Aceitar text/plain para Prometheus
		if c.GetHeader("Accept") == "text/plain" || c.Query("format") == "prometheus" {
			c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
			c.String(http.StatusOK, promMetrics.FormatPrometheus())
		} else {
			c.JSON(http.StatusOK, promMetrics)
		}
	})

	// GET /api/v1/health - Health check detalhado
	r.GET("/api/v1/health", func(c *gin.Context) {
		metrics := obsManager.GetMetrics(app)

		status := "healthy"
		if metrics.ErrorRate > 5.0 || metrics.ResponseTime > 100 {
			status = "degraded"
		}
		if metrics.ErrorRate > 20.0 || metrics.ResponseTime > 500 {
			status = "unhealthy"
		}

		health := gin.H{
			"status":    status,
			"timestamp": metrics.Timestamp,
			"uptime":    time.Since(obsManager.startTime).String(),
			"instance":  metrics.InstanceID,
			"checks": gin.H{
				"database": func() string {
					if app.db != nil {
						return "connected"
					}
					return "disconnected"
				}(),
				"redis": func() string {
					if app.rdb != nil {
						return "connected"
					}
					return "disconnected"
				}(),
				"memory": gin.H{
					"usage_mb": metrics.MemoryUsage,
					"status": func() string {
						if metrics.MemoryUsage > 500 {
							return "high"
						}
						return "normal"
					}(),
				},
				"goroutines": gin.H{
					"count": metrics.GoRoutines,
					"status": func() string {
						if metrics.GoRoutines > 1000 {
							return "high"
						}
						return "normal"
					}(),
				},
			},
			"metrics": gin.H{
				"requests_total":      metrics.RequestsTotal,
				"requests_per_second": metrics.RequestsPerSecond,
				"response_time_ms":    metrics.ResponseTime,
				"error_rate_percent":  metrics.ErrorRate,
				"redis_hit_rate":      metrics.RedisHitRate,
			},
		}

		statusCode := http.StatusOK
		if status == "degraded" {
			statusCode = http.StatusServiceUnavailable
		} else if status == "unhealthy" {
			statusCode = http.StatusInternalServerError
		}

		c.JSON(statusCode, health)
	})

	// GET /api/v1/cluster/status - Status detalhado do cluster
	r.GET("/api/v1/cluster/status", func(c *gin.Context) {
		metrics := obsManager.GetMetrics(app)
		c.JSON(http.StatusOK, gin.H{
			"cluster":          metrics.ClusterStatus,
			"current_instance": metrics.InstanceID,
			"timestamp":        metrics.Timestamp,
		})
	})
}
