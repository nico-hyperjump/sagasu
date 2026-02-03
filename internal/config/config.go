// Package config provides configuration loading and structs for the Sagasu server.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application.
type Config struct {
	Debug   bool           `yaml:"debug"`
	Server  ServerConfig   `yaml:"server"`
	Storage StorageConfig  `yaml:"storage"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	Search  SearchConfig   `yaml:"search"`
	Watch   WatchConfig    `yaml:"watch"`
}

// WatchConfig holds directory watch settings.
type WatchConfig struct {
	Directories []string `yaml:"directories"`
	Extensions  []string `yaml:"extensions"`
	Recursive   *bool    `yaml:"recursive"`
}

// Recursive returns whether to watch recursively; defaults to true when unset.
func (w *WatchConfig) RecursiveOrDefault() bool {
	if w.Recursive != nil {
		return *w.Recursive
	}
	return true
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// StorageConfig holds paths for database and indices.
type StorageConfig struct {
	DatabasePath   string `yaml:"database_path"`
	BleveIndexPath string `yaml:"bleve_index_path"`
	FAISSIndexPath string `yaml:"faiss_index_path"`
}

// EmbeddingConfig holds ONNX embedder settings.
type EmbeddingConfig struct {
	ModelPath       string `yaml:"model_path"`
	Dimensions      int    `yaml:"dimensions"`
	MaxTokens       int    `yaml:"max_tokens"`
	UseQuantization bool   `yaml:"use_quantization"`
	CacheSize       int    `yaml:"cache_size"`
}

// SearchConfig holds search and chunking settings.
type SearchConfig struct {
	DefaultLimit           int     `yaml:"default_limit"`
	MaxLimit               int     `yaml:"max_limit"`
	DefaultKeywordEnabled  bool    `yaml:"default_keyword_enabled"`
	DefaultSemanticEnabled bool    `yaml:"default_semantic_enabled"`
	ChunkSize              int     `yaml:"chunk_size"`
	ChunkOverlap           int     `yaml:"chunk_overlap"`
	TopKCandidates         int     `yaml:"top_k_candidates"`
	KeywordTitleBoost      float64 `yaml:"keyword_title_boost"`
}

// Load reads and parses the config file at path, expands paths, and applies defaults.
// Returns an error if the file cannot be read or parsed.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	ApplyDefaults(&cfg)

	configDir := filepath.Dir(path)
	cfg.Storage.DatabasePath = expandPath(cfg.Storage.DatabasePath, configDir)
	cfg.Storage.BleveIndexPath = expandPath(cfg.Storage.BleveIndexPath, configDir)
	cfg.Storage.FAISSIndexPath = expandPath(cfg.Storage.FAISSIndexPath, configDir)
	cfg.Embedding.ModelPath = expandPath(cfg.Embedding.ModelPath, configDir)
	for i := range cfg.Watch.Directories {
		cfg.Watch.Directories[i] = expandPath(cfg.Watch.Directories[i], configDir)
	}

	return &cfg, nil
}

// Save writes the config to path. Used for persisting watch directory add/remove.
func Save(path string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}
	return nil
}

// expandPath converts a path to absolute. Paths starting with "./" are relative to configDir;
// other relative paths are relative to the home directory.
func expandPath(path string, configDir string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if strings.HasPrefix(path, "./") || path == "." {
		return filepath.Join(configDir, path)
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, path)
	}
	return path
}
