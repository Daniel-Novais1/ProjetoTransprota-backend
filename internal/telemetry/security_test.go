package telemetry

import (
	"testing"
)

// ============================================================================
// SECURITY TESTS - SQL INJECTION
// ============================================================================

func TestValidateSQLInjection(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid input", "normal-text-123", false},
		{"Single quote", "text'or'1=1", true},
		{"Double quote", "text\"or\"1=1", true},
		{"Semicolon", "text; DROP TABLE", true},
		{"Comment", "text--comment", true},
		{"Union", "text UNION SELECT", true},
		{"Exec", "text EXEC xp_cmdshell", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateSQLInjection(tt.input)
			if result != tt.expected {
				t.Errorf("ValidateSQLInjection(%s) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}
