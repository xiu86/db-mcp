package logger_test

import (
	"testing"

	"log/slog"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/config"
	"db-mcp/pkg/logger"
)

func TestNewLogger_JSONFormat(t *testing.T) {
	cfg := &config.LogConfig{Level: "info", Format: "json", Output: "stdout"}
	appLogger := logger.NewLogger(cfg)
	assert.NotNil(t, appLogger)
}

func TestNewLogger_TextFormat(t *testing.T) {
	cfg := &config.LogConfig{Level: "info", Format: "text", Output: "stdout"}
	appLogger := logger.NewLogger(cfg)
	assert.NotNil(t, appLogger)
	assert.Equal(t, "text", cfg.Format)
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"warning", slog.LevelWarn},
		{"unknown", slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			cfg := &config.LogConfig{Level: tt.input}
			appLogger := logger.NewLogger(cfg)
			// Use string comparison to avoid level comparison issues
			var actualLevel string
			switch appLogger.Level() {
			case slog.LevelDebug:
				actualLevel = "debug"
			case slog.LevelInfo:
				actualLevel = "info"
			case slog.LevelWarn:
				actualLevel = "warn"
			case slog.LevelError:
				actualLevel = "error"
			}
			var expectedLevel string
			switch tt.expected {
			case slog.LevelDebug:
				expectedLevel = "debug"
			case slog.LevelInfo:
				expectedLevel = "info"
			case slog.LevelWarn:
				expectedLevel = "warn"
			case slog.LevelError:
				expectedLevel = "error"
			}
			assert.Equal(t, expectedLevel, actualLevel)
		})
	}
}

func TestLogger_With(t *testing.T) {
	cfg := &config.LogConfig{Level: "info", Format: "json"}
	appLogger := logger.NewLogger(cfg)
	childLogger := appLogger.With("service", "test")
	assert.NotNil(t, childLogger)
}

func TestLogger_LogMethods(t *testing.T) {
	cfg := &config.LogConfig{Level: "info", Format: "json"}
	appLogger := logger.NewLogger(cfg)

	// These should not panic
	appLogger.Debug("debug message")
	appLogger.Info("info message")
	appLogger.Warn("warn message")
	appLogger.Error("error message")
}
