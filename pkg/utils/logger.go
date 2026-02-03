package utils

import "go.uber.org/zap"

// NewProductionLogger returns a production zap logger, or a no-op logger on error.
func NewProductionLogger() (*zap.Logger, error) {
	return zap.NewProduction()
}
