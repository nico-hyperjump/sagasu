package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  host: "127.0.0.1"
  port: 9000
storage:
  database_path: "test.db"
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 9000 {
		t.Errorf("unexpected server config: %+v", cfg.Server)
	}
	if cfg.Storage.DatabasePath == "" {
		t.Error("database_path should be set")
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := &Config{}
	ApplyDefaults(cfg)
	if cfg.Server.Host != "localhost" {
		t.Errorf("default host: got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("default port: got %d", cfg.Server.Port)
	}
	if cfg.Search.DefaultLimit != 10 {
		t.Errorf("default limit: got %d", cfg.Search.DefaultLimit)
	}
}
