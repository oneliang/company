package logging

import (
	"log/slog"
)

// AuraLogAdapter implements aura's logger.Log interface using slog.
// This allows aura SDK logs to be output through company's logging system.
type AuraLogAdapter struct {
	logger *slog.Logger
	module string
}

// NewAuraLogAdapter creates an adapter that wraps company's slog logger.
func NewAuraLogAdapter(module string) *AuraLogAdapter {
	return &AuraLogAdapter{
		logger: slog.Default().With("module", module),
		module: module,
	}
}

// Debug logs a debug message with key-value pairs.
func (a *AuraLogAdapter) Debug(msg string, keyValues ...any) {
	a.logger.Debug(msg, keyValues...)
}

// Info logs an info message with key-value pairs.
func (a *AuraLogAdapter) Info(msg string, keyValues ...any) {
	a.logger.Info(msg, keyValues...)
}

// Warn logs a warning message with key-value pairs.
func (a *AuraLogAdapter) Warn(msg string, keyValues ...any) {
	a.logger.Warn(msg, keyValues...)
}

// Error logs an error message with key-value pairs.
func (a *AuraLogAdapter) Error(msg string, keyValues ...any) {
	a.logger.Error(msg, keyValues...)
}