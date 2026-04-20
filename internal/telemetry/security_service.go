package telemetry

import (
	"strings"

	"github.com/Daniel-Novais1/ProjetoTransprota-backend/internal/logger"
)

// ============================================================================
// SECURITY SERVICE - SQL INJECTION VALIDATION
// ============================================================================

// ValidateSQLInjection verifica se há tentativas de injeção SQL em strings
func ValidateSQLInjection(input string) bool {
	// Padrões comuns de injeção SQL
	sqlInjectionPatterns := []string{
		"'",
		"\"",
		";",
		"--",
		"/*",
		"*/",
		"xp_",
		"exec",
		"union",
		"select",
		"insert",
		"update",
		"delete",
		"drop",
		"alter",
		"create",
		"truncate",
	}

	inputLower := strings.ToLower(input)
	for _, pattern := range sqlInjectionPatterns {
		if strings.Contains(inputLower, pattern) {
			logger.Warn("Security", "SQL injection attempt detected | Pattern: %s", pattern)
			return true
		}
	}

	return false
}
