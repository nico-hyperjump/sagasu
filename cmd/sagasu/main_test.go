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
			args:     []string{"invoice from microsoft", "-min-keyword-score", "0.5"},
			expected: []string{"-min-keyword-score", "0.5", "invoice from microsoft"},
		},
		{
			name:     "flags first returns unchanged",
			args:     []string{"-min-keyword-score", "0.5", "invoice from microsoft"},
			expected: []string{"-min-keyword-score", "0.5", "invoice from microsoft"},
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

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{"single word", []string{"hyperjump"}, "hyperjump"},
		{"multiple words", []string{"hyperjump", "profile"}, "hyperjump profile"},
		{"single quoted phrase", []string{"hyperjump profile"}, "hyperjump profile"},
		{"three words", []string{"machine", "learning", "algorithms"}, "machine learning algorithms"},
		{"empty args", []string{}, ""},
		{"blank args", []string{"  ", "  "}, ""},
		{"one space", []string{" "}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSearchQuery(tt.args)
			if got != tt.expected {
				t.Errorf("buildSearchQuery(%v) = %q, want %q", tt.args, got, tt.expected)
			}
		})
	}
}

func TestSearchConfigPathFromArgs(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		defaultPath string
		want   string
	}{
		{"no config flag", []string{"-limit", "5", "query"}, "/default.yaml", "/default.yaml"},
		{"-config present", []string{"-config", "/custom.yaml", "query"}, "/default.yaml", "/custom.yaml"},
		{"--config present", []string{"--config", "/other.yaml"}, "/default.yaml", "/other.yaml"},
		{"config at end", []string{"query", "-config", "/end.yaml"}, "/default.yaml", "/end.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := searchConfigPathFromArgs(tt.args, tt.defaultPath)
			if got != tt.want {
				t.Errorf("searchConfigPathFromArgs() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSearchMinScoreDefaultsFromConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `
search:
  default_min_keyword_score: 0.2
  default_min_semantic_score: 0.3
`
	if err := os.WriteFile(configPath, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	kw, sem := searchMinScoreDefaultsFromConfig(configPath)
	if kw != 0.2 || sem != 0.3 {
		t.Errorf("searchMinScoreDefaultsFromConfig() = %f, %f; want 0.2, 0.3", kw, sem)
	}
	// Missing file returns 0.49, 0.49
	kw2, sem2 := searchMinScoreDefaultsFromConfig(filepath.Join(dir, "nonexistent.yaml"))
	if kw2 != 0.49 || sem2 != 0.49 {
		t.Errorf("searchMinScoreDefaultsFromConfig(nonexistent) = %f, %f; want 0.49, 0.49", kw2, sem2)
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
