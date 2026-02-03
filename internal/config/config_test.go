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
	if cfg.Debug {
		t.Error("debug should default to false when unset")
	}
}

func TestLoad_debugTrue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
debug: true
server:
  host: "localhost"
  port: 8080
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
	if !cfg.Debug {
		t.Error("debug should be true when set in config")
	}
}

func TestLoad_expandPathDotSlashRelativeToConfigDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  host: "localhost"
  port: 8080
storage:
  database_path: "./data/db/documents.db"
watch:
  directories: ["./dev/sample"]
`
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	wantDB := filepath.Join(dir, "data", "db", "documents.db")
	if cfg.Storage.DatabasePath != wantDB {
		t.Errorf("database_path = %s, want %s", cfg.Storage.DatabasePath, wantDB)
	}
	if len(cfg.Watch.Directories) != 1 {
		t.Fatalf("watch directories: got %d", len(cfg.Watch.Directories))
	}
	wantWatch := filepath.Join(dir, "dev", "sample")
	if cfg.Watch.Directories[0] != wantWatch {
		t.Errorf("watch directory = %s, want %s", cfg.Watch.Directories[0], wantWatch)
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
	if !cfg.Search.DefaultKeywordEnabled || !cfg.Search.DefaultSemanticEnabled {
		t.Errorf("when both search enabled flags are false, both should default to true; got keyword=%v semantic=%v",
			cfg.Search.DefaultKeywordEnabled, cfg.Search.DefaultSemanticEnabled)
	}
	if cfg.Search.KeywordTitleBoost != 10.0 {
		t.Errorf("default keyword_title_boost: got %f, want 10.0", cfg.Search.KeywordTitleBoost)
	}
	if cfg.Watch.Extensions == nil {
		t.Error("watch extensions should be set by default")
	}
	if len(cfg.Watch.Extensions) != 9 || cfg.Watch.Extensions[0] != ".txt" {
		t.Errorf("watch extensions: got %v", cfg.Watch.Extensions)
	}
	if cfg.Watch.Extensions[6] != ".pptx" || cfg.Watch.Extensions[7] != ".odp" || cfg.Watch.Extensions[8] != ".ods" {
		t.Errorf("watch extensions should include .pptx, .odp, .ods: got %v", cfg.Watch.Extensions)
	}
}

func TestApplyDefaults_WatchRecursiveWhenDirectoriesSet(t *testing.T) {
	cfg := &Config{Watch: WatchConfig{Directories: []string{"/tmp/docs"}}}
	ApplyDefaults(cfg)
	if cfg.Watch.Recursive == nil || !*cfg.Watch.Recursive {
		t.Error("recursive should default to true when directories are set")
	}
}

func TestWatchConfig_RecursiveOrDefault(t *testing.T) {
	t.Run("nil_returns_true", func(t *testing.T) {
		w := &WatchConfig{}
		if got := w.RecursiveOrDefault(); !got {
			t.Errorf("RecursiveOrDefault() = %v, want true", got)
		}
	})
	t.Run("true_returns_true", func(t *testing.T) {
		v := true
		w := &WatchConfig{Recursive: &v}
		if got := w.RecursiveOrDefault(); !got {
			t.Errorf("RecursiveOrDefault() = %v, want true", got)
		}
	})
	t.Run("false_returns_false", func(t *testing.T) {
		f := false
		w := &WatchConfig{Recursive: &f}
		if got := w.RecursiveOrDefault(); got {
			t.Errorf("RecursiveOrDefault() = %v, want false", got)
		}
	})
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "saved.yaml")
	cfg := &Config{
		Server:  ServerConfig{Host: "localhost", Port: 9090},
		Storage: StorageConfig{DatabasePath: "/tmp/db"},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Server.Port != 9090 {
		t.Errorf("loaded port: got %d", loaded.Server.Port)
	}
}
