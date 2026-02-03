package utils

import (
	"testing"
)

func TestNewLogger(t *testing.T) {
	t.Run("debug mode returns development logger", func(t *testing.T) {
		logger, err := NewLogger(true)
		if err != nil {
			t.Fatalf("NewLogger(true) error: %v", err)
		}
		if logger == nil {
			t.Fatal("NewLogger(true) returned nil logger")
		}
		_ = logger.Sync()
	})

	t.Run("production mode returns production logger", func(t *testing.T) {
		logger, err := NewLogger(false)
		if err != nil {
			t.Fatalf("NewLogger(false) error: %v", err)
		}
		if logger == nil {
			t.Fatal("NewLogger(false) returned nil logger")
		}
		_ = logger.Sync()
	})
}
