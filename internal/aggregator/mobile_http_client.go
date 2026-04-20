package aggregator

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
)

// ============================================================================
// MOBILE HTTP CLIENT - SIMULA DISPOSITIVO MÓVEL
// ============================================================================

const (
	// Timeout padrão para requisições externas (2 segundos para failover)
	DefaultRequestTimeout = 2 * time.Second
	// TTL do cache para dados externos (30 segundos)
	ExternalAPICacheTTL = 30 * time.Second
)

// MobileHTTPClient simula requisições de um dispositivo móvel Android/iOS
type MobileHTTPClient struct {
	client      *http.Client
	userAgent   string
	deviceFingerprint string
}

// NewMobileHTTPClient cria um novo cliente HTTP simulando mobile
func NewMobileHTTPClient() *MobileHTTPClient {
	// Configurar transporte TLS para simular fingerprint de mobile
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			// Simular cipher suites comuns em mobile
			CipherSuites: []uint16{
				tls.TLS_AES_128_GCM_SHA256,
				tls.TLS_AES_256_GCM_SHA384,
				tls.TLS_CHACHA20_POLY1305_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			},
			// Habilitar session resumption
			ClientSessionCache: tls.NewLRUClientSessionCache(128),
		},
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   DefaultRequestTimeout,
	}

	return &MobileHTTPClient{
		client:    client,
		userAgent: "Mozilla/5.0 (Linux; Android 13; SM-G998B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
		deviceFingerprint: "android-13-chrome-120",
	}
}

// GetWithMobileHeaders realiza GET com headers de dispositivo móvel
func (m *MobileHTTPClient) GetWithMobileHeaders(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Headers simulando app móvel
	req.Header.Set("User-Agent", m.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Cache-Control", "max-age=0")
	
	// Headers específicos de app
	req.Header.Set("X-App-Version", "1.0.0")
	req.Header.Set("X-Device-ID", m.deviceFingerprint)
	req.Header.Set("X-Platform", "android")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	logger.Debug("Aggregator", "Mobile HTTP GET | URL: %s | User-Agent: %s", url, m.userAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		logger.Warn("Aggregator", "Mobile HTTP request failed | URL: %s | Error: %v", url, err)
		return nil, err
	}

	return resp, nil
}

// PostWithMobileHeaders realiza POST com headers de dispositivo móvel
func (m *MobileHTTPClient) PostWithMobileHeaders(ctx context.Context, url string, body string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}

	// Headers simulando app móvel
	req.Header.Set("User-Agent", m.userAgent)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "pt-BR,pt;q=0.9,en-US;q=0.8,en;q=0.7")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://app.transprota.com.br")
	req.Header.Set("Referer", "https://app.transprota.com.br/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("X-App-Version", "1.0.0")
	req.Header.Set("X-Device-ID", m.deviceFingerprint)
	req.Header.Set("X-Platform", "android")

	logger.Debug("Aggregator", "Mobile HTTP POST | URL: %s | User-Agent: %s", url, m.userAgent)

	resp, err := m.client.Do(req)
	if err != nil {
		logger.Warn("Aggregator", "Mobile HTTP POST failed | URL: %s | Error: %v", url, err)
		return nil, err
	}

	return resp, nil
}

// SetTimeout ajusta timeout do cliente
func (m *MobileHTTPClient) SetTimeout(timeout time.Duration) {
	m.client.Timeout = timeout
}
