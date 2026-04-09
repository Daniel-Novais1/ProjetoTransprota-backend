package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// GeoLocation representa informações de localização geográfica
type GeoLocation struct {
	Country     string  `json:"country"`
	CountryCode string  `json:"country_code"`
	Region      string  `json:"region"`
	City        string  `json:"city"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	ISP         string  `json:"isp"`
	IsVPN       bool    `json:"is_vpn"`
	IsBrazil    bool    `json:"is_brazil"`
	IsGoias     bool    `json:"is_goias"`
}

// GeoRateLimitConfig representa configuração de rate limiting por região
type GeoRateLimitConfig struct {
	DefaultLimit      int           `json:"default_limit"`       // Requisições por minuto
	BrazilLimit       int           `json:"brazil_limit"`        // Para IPs do Brasil
	GoiasLimit        int           `json:"goias_limit"`         // Para IPs de Goiás
	VPNAllowed        bool          `json:"vpn_allowed"`         // Se permite VPNs
	VPNDedicatedLimit int           `json:"vpn_dedicated_limit"` // Limite para VPNs autorizadas
	BlockedCountries  []string      `json:"blocked_countries"`   // Países bloqueados
	AllowedRegions    []string      `json:"allowed_regions"`     // Regiões permitidas
	WhitelistIPs      []string      `json:"whitelist_ips"`       // IPs autorizados
	BlacklistIPs      []string      `json:"blacklist_ips"`       // IPs bloqueados
	CacheTTL          time.Duration `json:"cache_ttl"`           // TTL do cache de geolocalização
}

// GeoRateLimiter gerencia rate limiting geográfico
type GeoRateLimiter struct {
	config        GeoRateLimitConfig
	locationCache map[string]*GeoLocation
	cacheMutex    sync.RWMutex
	rateLimiters  map[string]*RateLimiter
	limiterMutex  sync.RWMutex
}

// RateLimiter representa um rate limiter por IP
type RateLimiter struct {
	requests    []time.Time
	maxRequests int
	window      time.Duration
	mutex       sync.Mutex
}

// NewGeoRateLimiter cria um novo rate limiter geográfico
func NewGeoRateLimiter() *GeoRateLimiter {
	config := GeoRateLimitConfig{
		DefaultLimit:      30,  // 30 req/min para estrangeiros
		BrazilLimit:       60,  // 60 req/min para Brasil
		GoiasLimit:        100, // 100 req/min para Goiás
		VPNAllowed:        false,
		VPNDedicatedLimit: 200,
		BlockedCountries:  []string{"CN", "RU", "KP", "IR"},       // China, Rússia, Coreia do Norte, Irã
		AllowedRegions:    []string{"BR", "US", "AR", "UY", "PY"}, // Brasil e vizinhos
		WhitelistIPs:      []string{},
		BlacklistIPs:      []string{},
		CacheTTL:          24 * time.Hour,
	}

	return &GeoRateLimiter{
		config:        config,
		locationCache: make(map[string]*GeoLocation),
		rateLimiters:  make(map[string]*RateLimiter),
	}
}

// GeoRateLimitMiddleware cria middleware de rate limiting geográfico
func (grl *GeoRateLimiter) GeoRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := getClientIP(c)

		// Verificar blacklist
		if grl.isIPBlacklisted(clientIP) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Acesso bloqueado por política de segurança",
				"code":  "GEO_BLOCKED",
			})
			c.Abort()
			return
		}

		// Verificar whitelist
		if grl.isIPWhitelisted(clientIP) {
			c.Next()
			return
		}

		// Obter localização geográfica
		location, err := grl.getGeoLocation(clientIP)
		if err != nil {
			log.Printf("Erro ao obter geolocalização para IP %s: %v", clientIP, err)
			// Em caso de erro, aplicar rate limit padrão
			if !grl.checkRateLimit(clientIP, grl.config.DefaultLimit) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Rate limit excedido",
					"code":  "RATE_LIMIT_EXCEEDED",
					"limit": grl.config.DefaultLimit,
				})
				c.Abort()
				return
			}
			c.Next()
			return
		}

		// Verificar se país está bloqueado
		if grl.isCountryBlocked(location.CountryCode) {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Acesso bloqueado para sua região",
				"code":    "COUNTRY_BLOCKED",
				"country": location.Country,
			})
			c.Abort()
			return
		}

		// Verificar VPN
		if location.IsVPN && !grl.config.VPNAllowed {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Acesso via VPN não permitido",
				"code":  "VPN_BLOCKED",
			})
			c.Abort()
			return
		}

		// Determinar limite de rate baseado na localização e suspeita
		limit := grl.getExtremeRateLimit(clientIP)

		// Verificar rate limit
		if !grl.checkRateLimit(clientIP, limit) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":    "Rate limit excedido",
				"code":     "RATE_LIMIT_EXCEEDED",
				"limit":    limit,
				"location": location.Country,
			})
			c.Abort()
			return
		}

		// Adicionar headers de informação
		c.Header("X-Geo-Country", location.Country)
		c.Header("X-Geo-Region", location.Region)
		c.Header("X-Rate-Limit-Limit", fmt.Sprintf("%d", limit))

		c.Next()
	}
}

// getClientIP obtém o IP real do cliente
func getClientIP(c *gin.Context) string {
	// Tentar obter de headers primeiro (para proxies)
	if xForwardedFor := c.GetHeader("X-Forwarded-For"); xForwardedFor != "" {
		// Pega o primeiro IP da lista
		ips := strings.Split(xForwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}

	if xRealIP := c.GetHeader("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	return c.ClientIP()
}

// isIPBlacklisted verifica se IP está na blacklist
func (grl *GeoRateLimiter) isIPBlacklisted(ip string) bool {
	for _, blacklisted := range grl.config.BlacklistIPs {
		if ip == blacklisted {
			return true
		}
	}
	return false
}

// isIPWhitelisted verifica se IP está na whitelist
func (grl *GeoRateLimiter) isIPWhitelisted(ip string) bool {
	for _, whitelisted := range grl.config.WhitelistIPs {
		if ip == whitelisted {
			return true
		}
	}
	return false
}

// getGeoLocation obtém localização geográfica do IP
func (grl *GeoRateLimiter) getGeoLocation(ip string) (*GeoLocation, error) {
	// Verificar cache primeiro
	grl.cacheMutex.RLock()
	if cached, exists := grl.locationCache[ip]; exists {
		grl.cacheMutex.RUnlock()
		return cached, nil
	}
	grl.cacheMutex.RUnlock()

	// Obter localização (simulação - em produção usar API real)
	location, err := grl.lookupGeoLocation(ip)
	if err != nil {
		return nil, err
	}

	// Salvar no cache
	grl.cacheMutex.Lock()
	grl.locationCache[ip] = location
	grl.cacheMutex.Unlock()

	return location, nil
}

// lookupGeoLocation consulta API de geolocalização (simulação)
func (grl *GeoRateLimiter) lookupGeoLocation(ip string) (*GeoLocation, error) {
	// Simulação de lookup de geolocalização
	// Em produção, usar APIs como MaxMind GeoIP2, IP-API, etc.

	// IPs locais sempre são do Brasil
	if ip == "127.0.0.1" || ip == "::1" || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "10.") {
		return &GeoLocation{
			Country:     "Brasil",
			CountryCode: "BR",
			Region:      "Goiás",
			City:        "Goiânia",
			Latitude:    -16.6864,
			Longitude:   -49.2643,
			ISP:         "Local",
			IsVPN:       false,
			IsBrazil:    true,
			IsGoias:     true,
		}, nil
	}

	// Simulação para alguns ranges de IP brasileiros
	if grl.isBrazilianIP(ip) {
		location := &GeoLocation{
			Country:     "Brasil",
			CountryCode: "BR",
			Region:      "Goiás",
			City:        "Goiânia",
			Latitude:    -16.6864,
			Longitude:   -49.2643,
			ISP:         "Brasil Telecom",
			IsVPN:       false,
			IsBrazil:    true,
			IsGoias:     true,
		}

		// Verificar se é de outros estados
		if !grl.isGoiasIP(ip) {
			location.Region = "São Paulo"
			location.City = "São Paulo"
			location.IsGoias = false
			location.Latitude = -23.5505
			location.Longitude = -46.6333
		}

		return location, nil
	}

	// Simulação para IPs estrangeiros
	return &GeoLocation{
		Country:     "United States",
		CountryCode: "US",
		Region:      "California",
		City:        "San Francisco",
		Latitude:    37.7749,
		Longitude:   -122.4194,
		ISP:         "Foreign ISP",
		IsVPN:       false,
		IsBrazil:    false,
		IsGoias:     false,
	}, nil
}

// isBrazilianIP verifica se IP é brasileiro (simulação)
func (grl *GeoRateLimiter) isBrazilianIP(ip string) bool {
	// Simulação de ranges brasileiros
	brazilianRanges := []string{
		"177.", "179.", "186.", "187.", "189.", "200.", "201.",
	}

	for _, range_ := range brazilianRanges {
		if strings.HasPrefix(ip, range_) {
			return true
		}
	}

	return false
}

// isGoiasIP verifica se IP é de Goiás (simulação)
func (grl *GeoRateLimiter) isGoiasIP(ip string) bool {
	// Simulação simplificada - alguns ranges de Goiás
	goiasRanges := []string{
		"177.66.", "179.106.", "186.192.", "187.95.",
	}

	for _, range_ := range goiasRanges {
		if strings.HasPrefix(ip, range_) {
			return true
		}
	}
	return false
}

// isSuspiciousIP verifica se IP é suspeito e deve ter limitação extrema
func (grl *GeoRateLimiter) isSuspiciousIP(ip string) bool {
	// IPs de data centers conhecidos (simulação)
	suspiciousRanges := []string{
		"8.8.8.",      // Google DNS
		"8.8.4.",      // Google DNS
		"1.1.1.",      // Cloudflare DNS
		"208.67.222.", // OpenDNS
		"208.67.220.", // OpenDNS
		"9.9.9.",      // Quad9 DNS
	}

	for _, range_ := range suspiciousRanges {
		if strings.HasPrefix(ip, range_) {
			return true
		}
	}

	// Verificar se é IP de VPN/proxy conhecido (simulação)
	vpnRanges := []string{
		"172.16.", // RFC1918
		"172.17.", // Docker
		"172.18.", // Docker
		"172.19.", // Docker
		"172.20.", // Docker
		"172.21.", // Docker
		"172.22.", // Docker
		"172.23.", // Docker
		"172.24.", // Docker
		"172.25.", // Docker
		"172.26.", // Docker
		"172.27.", // Docker
		"172.28.", // Docker
		"172.29.", // Docker
		"172.30.", // Docker
		"172.31.", // Docker
	}

	for _, range_ := range vpnRanges {
		if strings.HasPrefix(ip, range_) {
			return true
		}
	}

	return false
}

// getExtremeRateLimit obtém limite extremo para IPs suspeitos
func (grl *GeoRateLimiter) getExtremeRateLimit(ip string) int {
	if grl.isSuspiciousIP(ip) {
		return 1 // 1 requisição por minuto para IPs suspeitos
	}

	location, err := grl.getGeoLocation(ip)
	if err != nil {
		return 5 // Limite padrão baixo se não conseguir localizar
	}

	// IPs fora do Brasil ou muito distantes
	if !location.IsBrazil {
		return 2 // 2 req/min para estrangeiros
	}

	// IPs brasileiros mas fora de Goiás
	if location.IsBrazil && !location.IsGoias {
		return 10 // 10 req/min para outros estados brasileiros
	}

	// Verificar se é IP de data center brasileiro
	if strings.Contains(strings.ToLower(location.ISP), "datacenter") ||
		strings.Contains(strings.ToLower(location.ISP), "cloud") ||
		strings.Contains(strings.ToLower(location.ISP), "hosting") {
		return 3 // 3 req/min para data centers brasileiros
	}

	return grl.config.GoiasLimit // Limite normal para Goiás
}

// isCountryBlocked verifica se país está bloqueado
func (grl *GeoRateLimiter) isCountryBlocked(countryCode string) bool {
	for _, blocked := range grl.config.BlockedCountries {
		if countryCode == blocked {
			return true
		}
	}
	return false
}

// getRateLimitForLocation determina limite de rate baseado na localização
func (grl *GeoRateLimiter) getRateLimitForLocation(location *GeoLocation) int {
	if location.IsGoias {
		return grl.config.GoiasLimit
	}

	if location.IsBrazil {
		return grl.config.BrazilLimit
	}

	if location.IsVPN {
		return grl.config.VPNDedicatedLimit
	}

	return grl.config.DefaultLimit
}

// checkRateLimit verifica se IP excedeu o rate limit
func (grl *GeoRateLimiter) checkRateLimit(ip string, maxRequests int) bool {
	grl.limiterMutex.Lock()
	defer grl.limiterMutex.Unlock()

	limiter, exists := grl.rateLimiters[ip]
	if !exists {
		limiter = &RateLimiter{
			maxRequests: maxRequests,
			window:      time.Minute,
			requests:    make([]time.Time, 0),
		}
		grl.rateLimiters[ip] = limiter
	}

	return limiter.Allow()
}

// Allow verifica se permite requisição
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()

	// Remover requisições antigas (fora da janela)
	validRequests := make([]time.Time, 0)
	for _, req := range rl.requests {
		if now.Sub(req) <= rl.window {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// Verificar se pode fazer nova requisição
	if len(rl.requests) < rl.maxRequests {
		rl.requests = append(rl.requests, now)
		return true
	}

	return false
}

// GetGeoStats obtém estatísticas geográficas
func (grl *GeoRateLimiter) GetGeoStats() map[string]interface{} {
	grl.cacheMutex.RLock()
	grl.limiterMutex.RLock()
	defer grl.cacheMutex.RUnlock()
	defer grl.limiterMutex.RUnlock()

	stats := map[string]interface{}{
		"cached_locations": len(grl.locationCache),
		"active_limiters":  len(grl.rateLimiters),
		"config":           grl.config,
	}

	// Contar por país
	countryCount := make(map[string]int)
	for _, location := range grl.locationCache {
		countryCount[location.CountryCode]++
	}
	stats["countries"] = countryCount

	return stats
}

// UpdateConfig atualiza configuração do rate limiter
func (grl *GeoRateLimiter) UpdateConfig(newConfig GeoRateLimitConfig) {
	grl.cacheMutex.Lock()
	defer grl.cacheMutex.Unlock()

	grl.config = newConfig
	log.Printf("Configuração de rate limiting geográfico atualizada")
}

// ClearCache limpa cache de localizações
func (grl *GeoRateLimiter) ClearCache() {
	grl.cacheMutex.Lock()
	defer grl.cacheMutex.Unlock()

	grl.locationCache = make(map[string]*GeoLocation)
	log.Printf("Cache de geolocalização limpo")
}

// setupGeoRateLimitRoutes configura rotas do rate limiting geográfico
func setupGeoRateLimitRoutes(r *gin.Engine) {
	geoLimiter := NewGeoRateLimiter()

	// Aplicar middleware global
	r.Use(geoLimiter.GeoRateLimitMiddleware())

	// GET /api/v1/geo/stats - Estatísticas geográficas
	r.GET("/api/v1/geo/stats", func(c *gin.Context) {
		stats := geoLimiter.GetGeoStats()
		c.JSON(http.StatusOK, stats)
	})

	// POST /api/v1/geo/config - Atualizar configuração (admin only)
	r.POST("/api/v1/geo/config", func(c *gin.Context) {
		var config GeoRateLimitConfig
		if err := c.ShouldBindJSON(&config); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Dados inválidos"})
			return
		}

		geoLimiter.UpdateConfig(config)
		c.JSON(http.StatusOK, gin.H{"message": "Configuração atualizada com sucesso"})
	})

	// DELETE /api/v1/geo/cache - Limpar cache (admin only)
	r.DELETE("/api/v1/geo/cache", func(c *gin.Context) {
		geoLimiter.ClearCache()
		c.JSON(http.StatusOK, gin.H{"message": "Cache limpo com sucesso"})
	})

	// GET /api/v1/geo/location-test - Testar geolocalização
	r.GET("/api/v1/geo/location-test", func(c *gin.Context) {
		clientIP := getClientIP(c)
		location, err := geoLimiter.getGeoLocation(clientIP)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Erro ao obter localização",
				"ip":    clientIP,
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"ip":         clientIP,
			"location":   location,
			"rate_limit": geoLimiter.getRateLimitForLocation(location),
		})
	})
}
