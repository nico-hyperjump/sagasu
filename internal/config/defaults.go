package config

// ApplyDefaults sets default values for any zero values in cfg.
func ApplyDefaults(cfg *Config) {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "localhost"
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Storage.DatabasePath == "" {
		cfg.Storage.DatabasePath = "/usr/local/var/sagasu/data/db/documents.db"
	}
	if cfg.Storage.BleveIndexPath == "" {
		cfg.Storage.BleveIndexPath = "/usr/local/var/sagasu/data/indices/bleve"
	}
	if cfg.Storage.FAISSIndexPath == "" {
		cfg.Storage.FAISSIndexPath = "/usr/local/var/sagasu/data/indices/faiss"
	}
	if cfg.Embedding.ModelPath == "" {
		cfg.Embedding.ModelPath = "/usr/local/var/sagasu/data/models/all-MiniLM-L6-v2.onnx"
	}
	if cfg.Embedding.Dimensions == 0 {
		cfg.Embedding.Dimensions = 384
	}
	if cfg.Embedding.MaxTokens == 0 {
		cfg.Embedding.MaxTokens = 256
	}
	if cfg.Embedding.CacheSize == 0 {
		cfg.Embedding.CacheSize = 10000
	}
	if cfg.Search.DefaultLimit == 0 {
		cfg.Search.DefaultLimit = 10
	}
	if cfg.Search.MaxLimit == 0 {
		cfg.Search.MaxLimit = 100
	}
	if !cfg.Search.DefaultKeywordEnabled && !cfg.Search.DefaultSemanticEnabled {
		cfg.Search.DefaultKeywordEnabled = true
		cfg.Search.DefaultSemanticEnabled = true
	}
	if cfg.Search.ChunkSize == 0 {
		cfg.Search.ChunkSize = 512
	}
	if cfg.Search.ChunkOverlap == 0 {
		cfg.Search.ChunkOverlap = 50
	}
	if cfg.Search.TopKCandidates == 0 {
		cfg.Search.TopKCandidates = 100
	}
	if cfg.Search.KeywordTitleBoost == 0 {
		cfg.Search.KeywordTitleBoost = 10.0
	}
	if cfg.Search.DefaultMinKeywordScore == 0 {
		cfg.Search.DefaultMinKeywordScore = 0.49
	}
	if cfg.Search.DefaultMinSemanticScore == 0 {
		cfg.Search.DefaultMinSemanticScore = 0.49
	}
	if cfg.Watch.Extensions == nil {
		cfg.Watch.Extensions = []string{".txt", ".md", ".rst", ".pdf", ".docx", ".xlsx", ".pptx", ".odp", ".ods"}
	}
	// Recursive defaults to true when unset (nil).
	if len(cfg.Watch.Directories) > 0 && cfg.Watch.Recursive == nil {
		t := true
		cfg.Watch.Recursive = &t
	}
}
