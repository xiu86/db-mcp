package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"db-mcp/internal/config"
)

type Logger struct {
	level  slog.Level
	format string
	logger *slog.Logger
}

type LogFields map[string]interface{}

func NewLogger(cfg *config.LogConfig) *Logger {
	out := resolveOutput(cfg.Output)
	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.Level),
	}

	switch cfg.Format {
	case "json":
		handler = slog.NewJSONHandler(out, opts)
	default:
		handler = slog.NewTextHandler(out, opts)
	}

	return &Logger{
		level:  parseLevel(cfg.Level),
		format: cfg.Format,
		logger: slog.New(handler),
	}
}

func resolveOutput(output string) io.Writer {
	switch strings.ToLower(strings.TrimSpace(output)) {
	case "", "stdout":
		return os.Stdout
	case "stderr":
		return os.Stderr
	default:
		return os.Stdout
	}
}

func (l *Logger) Debug(msg string, fields ...interface{}) {
	l.logger.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...interface{}) {
	l.logger.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...interface{}) {
	l.logger.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...interface{}) {
	l.logger.Error(msg, fields...)
}

func (l *Logger) With(fields ...interface{}) *Logger {
	return &Logger{
		level:  l.level,
		format: l.format,
		logger: l.logger.With(fields...),
	}
}

func (l *Logger) Level() slog.Level {
	return l.level
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info":
		return slog.LevelInfo
	default:
		return slog.LevelInfo
	}
}
