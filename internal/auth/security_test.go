package auth

import (
	"testing"
	"time"
)

// ============================================================================
// SECURITY TESTS - JWT AND ANONYMIZATION
// ============================================================================

func TestVerifyJWTSecurity_ValidKey(t *testing.T) {
	// Chave segura de 64 bytes
	validKey := "this-is-a-very-secure-secret-key-with-at-least-32-bytes-for-hs256-algorithm"

	err := VerifyJWTSecurity(validKey)
	if err != nil {
		t.Errorf("Valid JWT key should pass security check: %v", err)
	}
}

func TestVerifyJWTSecurity_ShortKey(t *testing.T) {
	// Chave muito curta
	shortKey := "short"

	err := VerifyJWTSecurity(shortKey)
	if err == nil {
		t.Error("Short JWT key should fail security check")
	}

	t.Logf("Expected error: %v", err)
}

func TestVerifyJWTSecurity_DangerousDefault(t *testing.T) {
	// Chaves perigosas padrão
	dangerousKeys := []string{"secret", "password", "123456", "admin", "test"}

	for _, key := range dangerousKeys {
		err := VerifyJWTSecurity(key)
		if err == nil {
			t.Errorf("Dangerous key '%s' should fail security check", key)
		}
	}
}

func TestGenerateSecureSecret(t *testing.T) {
	secret, err := GenerateSecureSecret()
	if err != nil {
		t.Errorf("Failed to generate secure secret: %v", err)
	}

	// Verificar comprimento (64 bytes codificados em base64 = ~88 caracteres)
	if len(secret) < 80 {
		t.Errorf("Generated secret too short: %d characters", len(secret))
	}

	// Verificar se a chave passa na verificação de segurança
	err = VerifyJWTSecurity(secret)
	if err != nil {
		t.Errorf("Generated secret should pass security check: %v", err)
	}

	t.Logf("Generated secure secret: %s (length: %d)", secret, len(secret))
}

func TestVerifyDeviceHashSecurity_ValidHash(t *testing.T) {
	// Hash válido de 64 caracteres hex
	validHash := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"

	err := VerifyDeviceHashSecurity(validHash)
	if err != nil {
		t.Errorf("Valid device hash should pass security check: %v", err)
	}
}

func TestVerifyDeviceHashSecurity_InvalidLength(t *testing.T) {
	// Hash com comprimento incorreto
	invalidHash := "a1b2c3d4"

	err := VerifyDeviceHashSecurity(invalidHash)
	if err == nil {
		t.Error("Invalid length hash should fail security check")
	}

	t.Logf("Expected error: %v", err)
}

func TestVerifyDeviceHashSecurity_InvalidCharacters(t *testing.T) {
	// Hash com caracteres inválidos
	invalidHash := "g1h2i3j4k5l6m7n8o9p0q1r2s3t4u5v6w7x8y9z0a1b2c3d4e5f6g7h8i9j0k1l2"

	err := VerifyDeviceHashSecurity(invalidHash)
	if err == nil {
		t.Error("Hash with invalid characters should fail security check")
	}

	t.Logf("Expected error: %v", err)
}

func TestVerifyTokenExpiry_Valid(t *testing.T) {
	// Expiração válida (1 hora no futuro)
	validExpiry := time.Now().Add(1 * time.Hour)

	err := VerifyTokenExpiry(validExpiry)
	if err != nil {
		t.Errorf("Valid token expiry should pass: %v", err)
	}
}

func TestVerifyTokenExpiry_TooFar(t *testing.T) {
	// Expiração muito no futuro (48 horas)
	invalidExpiry := time.Now().Add(48 * time.Hour)

	err := VerifyTokenExpiry(invalidExpiry)
	if err == nil {
		t.Error("Token expiry too far in future should fail")
	}

	t.Logf("Expected error: %v", err)
}

func TestVerifyTokenExpiry_Expired(t *testing.T) {
	// Token já expirado (1 hora no passado)
	expiredExpiry := time.Now().Add(-1 * time.Hour)

	err := VerifyTokenExpiry(expiredExpiry)
	if err == nil {
		t.Error("Expired token should fail security check")
	}

	t.Logf("Expected error: %v", err)
}
