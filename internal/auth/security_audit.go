package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"
)

// ============================================================================
// SECURITY AUDIT - JWT AND ANONYMIZATION
// ============================================================================

// VerifyJWTSecurity verifica se a configuração JWT é segura
func VerifyJWTSecurity(secretKey string) error {
	// 1. Verificar comprimento da chave (mínimo 32 bytes para HS256)
	if len(secretKey) < 32 {
		return fmt.Errorf("JWT secret key too short: %d bytes (min: 32 bytes)", len(secretKey))
	}
	
	// 2. Verificar se a chave não é um valor padrão perigoso
	dangerousKeys := []string{"secret", "password", "123456", "admin", "test"}
	for _, dangerous := range dangerousKeys {
		if secretKey == dangerous {
			return fmt.Errorf("JWT secret key is a dangerous default value: %s", secretKey)
		}
	}
	
	return nil
}

// GenerateSecureSecret gera uma chave JWT segura (64 bytes)
func GenerateSecureSecret() (string, error) {
	secret := make([]byte, 64)
	_, err := rand.Read(secret)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure secret: %w", err)
	}
	return base64.StdEncoding.EncodeToString(secret), nil
}

// VerifyDeviceHashSecurity verifica se o hash do dispositivo é seguro
func VerifyDeviceHashSecurity(deviceHash string) error {
	// 1. Verificar comprimento (deve ser 64 caracteres hex)
	if len(deviceHash) != 64 {
		return fmt.Errorf("device hash invalid length: %d (expected: 64)", len(deviceHash))
	}
	
	// 2. Verificar se é hexadecimal válido
	for _, c := range deviceHash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("device hash contains invalid characters: %c", c)
		}
	}
	
	return nil
}

// VerifyTokenExpiry verifica se o tempo de expiração do token é seguro
func VerifyTokenExpiry(expiresAt time.Time) error {
	// Token deve expirar em no máximo 24 horas
	maxExpiry := time.Now().Add(24 * time.Hour)
	
	if expiresAt.After(maxExpiry) {
		return fmt.Errorf("token expiry too far in future: %v (max: 24h from now)", expiresAt)
	}
	
	// Token não deve expirar no passado
	if expiresAt.Before(time.Now()) {
		return fmt.Errorf("token already expired: %v", expiresAt)
	}
	
	return nil
}
