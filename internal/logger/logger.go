package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

// LogLevel representa o nível de log
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel = INFO
	logger       *log.Logger
	infoPrefix   = "[INFO] "
	warnPrefix   = "[WARN] "
	errorPrefix  = "[ERROR] "
	debugPrefix  = "[DEBUG] "
)

func init() {
	// Configurar logger com timestamp
	logger = log.New(os.Stdout, "", 0)
}

// SetLevel define o nível mínimo de log
func SetLevel(level LogLevel) {
	currentLevel = level
}

// logMessage formata e imprime a mensagem de log
func logMessage(level LogLevel, module, message string, args ...interface{}) {
	if level < currentLevel {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := ""

	switch level {
	case DEBUG:
		levelStr = debugPrefix
	case INFO:
		levelStr = infoPrefix
	case WARN:
		levelStr = warnPrefix
	case ERROR:
		levelStr = errorPrefix
	}

	formattedMsg := fmt.Sprintf(message, args...)
	log.Printf("[%s] [%s] [%s] %s", timestamp, levelStr, module, formattedMsg)
}

// Debug loga mensagem de nível DEBUG
func Debug(module, message string, args ...interface{}) {
	logMessage(DEBUG, module, message, args...)
}

// Info log de nível INFO
func Info(module, message string, args ...interface{}) {
	logMessage(INFO, module, message, args...)
}

// Warn log de nível WARN
func Warn(module, message string, args ...interface{}) {
	logMessage(WARN, module, message, args...)
}

// Error loga mensagem de nível ERROR
func Error(module, message string, args ...interface{}) {
	logMessage(ERROR, module, message, args...)
}
