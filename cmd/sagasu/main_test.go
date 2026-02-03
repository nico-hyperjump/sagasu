package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSearchArgsReorder(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "flags after query are moved first",
			args:     []string{"invoice from microsoft", "-min-score", "0.5"},
			expected: []string{"-min-score", "0.5", "invoice from microsoft"},
		},
		{
			name:     "flags first returns unchanged",
			args:     []string{"-min-score", "0.5", "invoice from microsoft"},
			expected: []string{"-min-score", "0.5", "invoice from microsoft"},
		},
		{
			name:     "query only returns unchanged",
			args:     []string{"invoice from microsoft"},
			expected: []string{"invoice from microsoft"},
		},
		{
			name:     "empty args returns unchanged",
			args:     []string{},
			expected: []string{},
		},
		{
			name:     "multiple positionals then flags",
			args:     []string{"one", "two", "-limit", "5"},
			expected: []string{"-limit", "5", "one", "two"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchArgsReorder(tt.args)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("searchArgsReorder() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestLoadConfig_prefersCwdConfigWhenDefaultPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
debug: true
server:
  host: "localhost"
  port: 8080
storage:
  database_path: "test.db"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origWd) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, resolved, err := loadConfig(defaultConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	// On macOS, cwd can be /private/var/... while configPath from t.TempDir() is /var/...; compare canonical paths.
	resolvedCanon, _ := filepath.EvalSymlinks(resolved)
	configPathCanon, _ := filepath.EvalSymlinks(configPath)
	if resolvedCanon != configPathCanon {
		t.Errorf("resolved path = %s (canon %s), want %s (canon %s)", resolved, resolvedCanon, configPath, configPathCanon)
	}
	if !cfg.Debug {
		t.Error("debug should be true from cwd config.yaml")
	}
}

func TestLoadConfig_usesExplicitPath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
server:
  host: "127.0.0.1"
  port: 9000
storage:
  database_path: "test.db"
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, resolved, err := loadConfig(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if resolved != configPath {
		t.Errorf("resolved path = %s, want %s", resolved, configPath)
	}
	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 9000 {
		t.Errorf("unexpected server config: %+v", cfg.Server)
	}
}
