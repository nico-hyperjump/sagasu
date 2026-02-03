package utils

import "go.uber.org/zap"

// NewProductionLogger returns a production zap logger, or a no-op logger on error.
func NewProductionLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}

// NewLogger returns a zap logger. When debug is true, uses development config
// (human-readable, debug level); otherwise uses production config (JSON, info level).
func NewLogger(debug bool) (*zap.Logger, error) {
	if debug {
		return zap.NewDevelopment()
	}
	return zap.NewProduction()
}
