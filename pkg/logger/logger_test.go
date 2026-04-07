package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"db-mcp/internal/config"
)

func TestNewLogger(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	log := NewLogger(cfg)
	assert.NotNil(t, log)
	assert.Equal(t, "debug", log.format)
	assert.Equal(t, slog.LevelDebug, log.level)
}

func TestNewLogger_TextFormat(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}

	log := NewLogger(cfg)
	assert.NotNil(t, log)
	assert.Equal(t, "text", log.format)
}

func TestNewLogger_DefaultLevel(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "unknown",
		Format: "json",
	}

	log := NewLogger(cfg)
	assert.NotNil(t, log)
	assert.Equal(t, slog.LevelInfo, log.level)
}

func TestLogger_Debug(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}

	log := NewLogger(cfg)

	// Should not panic
	log.Debug("debug message")
	log.Debug("debug with key", "key", "value")
}

func TestLogger_Info(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	log := NewLogger(cfg)

	// Should not panic
	log.Info("info message")
	log.Info("info with key", "key", "value")
}

func TestLogger_Warn(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "warn",
		Format: "json",
		Output: "stdout",
	}

	log := NewLogger(cfg)

	// Should not panic
	log.Warn("warning message")
	log.Warn("warning with key", "key", "value")
}

func TestLogger_Error(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "error",
		Format: "json",
		Output: "stdout",
	}

	log := NewLogger(cfg)

	// Should not panic
	log.Error("error message")
	log.Error("error with key", "key", "value")
}

func TestLogger_With(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	log := NewLogger(cfg)
	logWith := log.With("key", "value")

	assert.NotNil(t, logWith)
	assert.Equal(t, log.level, logWith.level)
	assert.Equal(t, log.format, logWith.format)
}

func TestLogger_WithMultiple(t *testing.T) {
	cfg := &config.LogConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	log := NewLogger(cfg)
	logWith := log.With("key1", "value1", "key2", "value2")

	assert.NotNil(t, logWith)
}

func TestLogger_Level(t *testing.T) {
	testCases := []struct {
		level   string
		slogLevel slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug},
		{"INFO", slog.LevelInfo},
		{"WARN", slog.LevelWarn},
		{"ERROR", slog.LevelError},
	}

	for _, tc := range testCases {
		t.Run(tc.level, func(t *testing.T) {
			cfg := &config.LogConfig{
				Level:  tc.level,
				Format: "json",
			}
			log := NewLogger(cfg)
			assert.Equal(t, tc.slogLevel, log.Level())
		})
	}
}

func TestParseLevel(t *testing.T) {
	testCases := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"unknown", slog.LevelInfo},
		{"", slog.LevelInfo},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got := parseLevel(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestLogFields(t *testing.T) {
	fields := LogFields{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	assert.Equal(t, "value1", fields["key1"])
	assert.Equal(t, 123, fields["key2"])
	assert.Equal(t, true, fields["key3"])
}
