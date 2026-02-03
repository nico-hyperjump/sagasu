// Package config provides configuration loading and structs for the Sagasu server.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the application.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Storage  StorageConfig  `yaml:"storage"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	Search   SearchConfig   `yaml:"search"`
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
	DefaultLimit          int     `yaml:"default_limit"`
	MaxLimit              int     `yaml:"max_limit"`
	DefaultKeywordWeight  float64 `yaml:"default_keyword_weight"`
	DefaultSemanticWeight float64 `yaml:"default_semantic_weight"`
	ChunkSize             int     `yaml:"chunk_size"`
	ChunkOverlap          int     `yaml:"chunk_overlap"`
	TopKCandidates        int     `yaml:"top_k_candidates"`
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

	cfg.Storage.DatabasePath = expandPath(cfg.Storage.DatabasePath)
	cfg.Storage.BleveIndexPath = expandPath(cfg.Storage.BleveIndexPath)
	cfg.Storage.FAISSIndexPath = expandPath(cfg.Storage.FAISSIndexPath)
	cfg.Embedding.ModelPath = expandPath(cfg.Embedding.ModelPath)

	return &cfg, nil
}

// expandPath converts a path to absolute using the home directory if it is not already absolute.
func expandPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, path)
	}
	return path
}
