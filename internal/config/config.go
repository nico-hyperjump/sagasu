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
	Debug     bool            `yaml:"debug"`
	Server    ServerConfig    `yaml:"server"`
	Storage   StorageConfig   `yaml:"storage"`
	Embedding EmbeddingConfig `yaml:"embedding"`
	Search    SearchConfig    `yaml:"search"`
	Watch     WatchConfig     `yaml:"watch"`
	Ranking   RankingConfig   `yaml:"ranking"`
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
	DefaultLimit               int     `yaml:"default_limit"`
	MaxLimit                   int     `yaml:"max_limit"`
	DefaultKeywordEnabled      bool    `yaml:"default_keyword_enabled"`
	DefaultSemanticEnabled     bool    `yaml:"default_semantic_enabled"`
	DefaultMinKeywordScore     float64 `yaml:"default_min_keyword_score"`
	DefaultMinSemanticScore    float64 `yaml:"default_min_semantic_score"`
	ChunkSize                  int     `yaml:"chunk_size"`
	ChunkOverlap               int     `yaml:"chunk_overlap"`
	TopKCandidates             int     `yaml:"top_k_candidates"`
	KeywordTitleBoost          float64 `yaml:"keyword_title_boost"`
	KeywordPhraseBoost         float64 `yaml:"keyword_phrase_boost"`
	// RankingEnabled enables the new content-aware ranking system.
	RankingEnabled             bool    `yaml:"ranking_enabled"`
}

// RankingConfig holds content-aware ranking settings.
type RankingConfig struct {
	// Weights for different scoring components
	FilenameWeight  float64 `yaml:"filename_weight"`
	ContentWeight   float64 `yaml:"content_weight"`
	PathWeight      float64 `yaml:"path_weight"`
	MetadataWeight  float64 `yaml:"metadata_weight"`

	// Filename scoring values
	ExactFilenameScore       float64 `yaml:"exact_filename_score"`
	AllWordsInOrderScore     float64 `yaml:"all_words_in_order_score"`
	AllWordsAnyOrderScore    float64 `yaml:"all_words_any_order_score"`
	SubstringMatchScore      float64 `yaml:"substring_match_score"`
	PrefixMatchScore         float64 `yaml:"prefix_match_score"`
	MultipleOccurrenceBonus  float64 `yaml:"multiple_occurrence_bonus"`
	ExtensionMatchScore      float64 `yaml:"extension_match_score"`

	// Content scoring values
	PhraseMatchScore         float64 `yaml:"phrase_match_score"`
	HeaderMatchScore         float64 `yaml:"header_match_score"`
	AllWordsContentScore     float64 `yaml:"all_words_content_score"`
	ScatteredWordsScore      float64 `yaml:"scattered_words_score"`
	StemmingMatchScore       float64 `yaml:"stemming_match_score"`

	// Path scoring values
	PathExactMatchScore      float64 `yaml:"path_exact_match_score"`
	PathPartialMatchScore    float64 `yaml:"path_partial_match_score"`
	PathComponentBonus       float64 `yaml:"path_component_bonus"`

	// Metadata scoring values
	AuthorMatchScore         float64 `yaml:"author_match_score"`
	TagMatchScore            float64 `yaml:"tag_match_score"`
	OtherMetadataScore       float64 `yaml:"other_metadata_score"`

	// TF-IDF settings
	MaxTFIDFMultiplier       float64 `yaml:"max_tfidf_multiplier"`
	TFIDFEnabled             bool    `yaml:"tfidf_enabled"`

	// Position-based scoring
	PositionBoostEnabled     bool    `yaml:"position_boost_enabled"`
	PositionBoostThreshold   float64 `yaml:"position_boost_threshold"`
	PositionBoostMultiplier  float64 `yaml:"position_boost_multiplier"`

	// Recency multiplier settings
	RecencyEnabled           bool    `yaml:"recency_enabled"`
	Recency24hMultiplier     float64 `yaml:"recency_24h_multiplier"`
	RecencyWeekMultiplier    float64 `yaml:"recency_week_multiplier"`
	RecencyMonthMultiplier   float64 `yaml:"recency_month_multiplier"`

	// Query quality multipliers
	QueryQualityEnabled      bool    `yaml:"query_quality_enabled"`
	PhraseMatchMultiplier    float64 `yaml:"phrase_match_multiplier"`
	AllWordsMultiplier       float64 `yaml:"all_words_multiplier"`
	PartialMatchMultiplier   float64 `yaml:"partial_match_multiplier"`

	// File size normalization
	FileSizeNormEnabled      bool    `yaml:"file_size_norm_enabled"`
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
