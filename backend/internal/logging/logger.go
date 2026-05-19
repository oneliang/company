package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

var logger *slog.Logger

// Init initializes the logger with the given level and file.
func Init(level string, logFile string) {
	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{Level: lvl}

	// Output to stdout + file
	var writer io.Writer = os.Stdout
	if logFile != "" {
		// Create log directory if needed
		logDir := filepath.Dir(logFile)
		if err := os.MkdirAll(logDir, 0755); err == nil {
			f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				writer = io.MultiWriter(os.Stdout, f)
			}
		}
	}

	logger = slog.New(slog.NewTextHandler(writer, opts))
	slog.SetDefault(logger)
}

// Debug logs a debug message.
func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

// Info logs an info message.
func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

// Warn logs a warning message.
func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

// Error logs an error message.
func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}