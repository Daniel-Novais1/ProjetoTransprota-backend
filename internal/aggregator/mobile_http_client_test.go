package aggregator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// ============================================================================
// UNIT TESTS - MOBILE HTTP CLIENT
// ============================================================================

func TestNewMobileHTTPClient(t *testing.T) {
	client := NewMobileHTTPClient()
	
	if client == nil {
		t.Fatal("NewMobileHTTPClient returned nil")
	}
	
	if client.userAgent == "" {
		t.Error("User-Agent should not be empty")
	}
	
	if client.deviceFingerprint == "" {
		t.Error("Device fingerprint should not be empty")
	}
	
	if client.client.Timeout != DefaultRequestTimeout {
		t.Errorf("Expected timeout %v, got %v", DefaultRequestTimeout, client.client.Timeout)
	}
}

func TestGetWithMobileHeaders(t *testing.T) {
	// Criar servidor de teste
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verificar headers mobile
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent header missing")
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Accept header incorrect")
		}
		if r.Header.Get("X-Platform") != "android" {
			t.Error("X-Platform header incorrect")
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	
	client := NewMobileHTTPClient()
	resp, err := client.GetWithMobileHeaders(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("GetWithMobileHeaders failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestGetWithMobileHeaders_Timeout(t *testing.T) {
	// Criar servidor lento
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(3 * time.Second) // Mais que o timeout de 2s
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewMobileHTTPClient()
	client.SetTimeout(1 * time.Second)
	
	_, err := client.GetWithMobileHeaders(context.Background(), server.URL)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestPostWithMobileHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type header incorrect")
		}
		if r.Header.Get("X-Platform") != "android" {
			t.Error("X-Platform header incorrect")
		}
		
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := NewMobileHTTPClient()
	resp, err := client.PostWithMobileHeaders(context.Background(), server.URL, "")
	if err != nil {
		t.Fatalf("PostWithMobileHeaders failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestSetTimeout(t *testing.T) {
	client := NewMobileHTTPClient()
	
	newTimeout := 5 * time.Second
	client.SetTimeout(newTimeout)
	
	if client.client.Timeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, client.client.Timeout)
	}
}
