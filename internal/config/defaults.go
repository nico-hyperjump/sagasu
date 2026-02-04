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
		cfg.Search.KeywordTitleBoost = 3.0 // reduced from 10.0 for smarter multi-term ranking
	}
	if cfg.Search.KeywordPhraseBoost == 0 {
		cfg.Search.KeywordPhraseBoost = 1.5 // boost for adjacent query terms (phrase match)
	}
	if cfg.Search.DefaultMinKeywordScore == 0 {
		cfg.Search.DefaultMinKeywordScore = 0 // disabled - let ranking handle relevance
	}
	if cfg.Search.DefaultMinSemanticScore == 0 {
		cfg.Search.DefaultMinSemanticScore = 0.05
	}
	if cfg.Watch.Extensions == nil {
		cfg.Watch.Extensions = []string{".txt", ".md", ".rst", ".pdf", ".docx", ".xlsx", ".pptx", ".odp", ".ods"}
	}
	// Recursive defaults to true when unset (nil).
	if len(cfg.Watch.Directories) > 0 && cfg.Watch.Recursive == nil {
		t := true
		cfg.Watch.Recursive = &t
	}

	// Apply ranking defaults
	applyRankingDefaults(&cfg.Ranking)

	// Apply vector defaults
	applyVectorDefaults(&cfg.Vector)
}

// applyVectorDefaults sets default values for vector configuration.
func applyVectorDefaults(cfg *VectorConfig) {
	if cfg.IndexType == "" {
		cfg.IndexType = "memory" // Default to in-memory index
	}
	// MaxVectors defaults to 0 (unlimited)
}

// applyRankingDefaults sets default values for ranking configuration.
func applyRankingDefaults(cfg *RankingConfig) {
	// Weights
	if cfg.FilenameWeight == 0 {
		cfg.FilenameWeight = 1.5
	}
	if cfg.ContentWeight == 0 {
		cfg.ContentWeight = 1.0
	}
	if cfg.PathWeight == 0 {
		cfg.PathWeight = 0.3
	}
	if cfg.MetadataWeight == 0 {
		cfg.MetadataWeight = 0.4
	}

	// Filename scoring
	if cfg.ExactFilenameScore == 0 {
		cfg.ExactFilenameScore = 100
	}
	if cfg.AllWordsInOrderScore == 0 {
		cfg.AllWordsInOrderScore = 90
	}
	if cfg.AllWordsAnyOrderScore == 0 {
		cfg.AllWordsAnyOrderScore = 80
	}
	if cfg.SubstringMatchScore == 0 {
		cfg.SubstringMatchScore = 60
	}
	if cfg.PrefixMatchScore == 0 {
		cfg.PrefixMatchScore = 45
	}
	if cfg.MultipleOccurrenceBonus == 0 {
		cfg.MultipleOccurrenceBonus = 10
	}
	if cfg.ExtensionMatchScore == 0 {
		cfg.ExtensionMatchScore = 60
	}

	// Content scoring
	if cfg.PhraseMatchScore == 0 {
		cfg.PhraseMatchScore = 120
	}
	if cfg.HeaderMatchScore == 0 {
		cfg.HeaderMatchScore = 110
	}
	if cfg.AllWordsContentScore == 0 {
		cfg.AllWordsContentScore = 90
	}
	if cfg.ScatteredWordsScore == 0 {
		cfg.ScatteredWordsScore = 70
	}
	if cfg.StemmingMatchScore == 0 {
		cfg.StemmingMatchScore = 55
	}

	// Path scoring
	if cfg.PathExactMatchScore == 0 {
		cfg.PathExactMatchScore = 40
	}
	if cfg.PathPartialMatchScore == 0 {
		cfg.PathPartialMatchScore = 30
	}
	if cfg.PathComponentBonus == 0 {
		cfg.PathComponentBonus = 10
	}

	// Metadata scoring
	if cfg.AuthorMatchScore == 0 {
		cfg.AuthorMatchScore = 45
	}
	if cfg.TagMatchScore == 0 {
		cfg.TagMatchScore = 40
	}
	if cfg.OtherMetadataScore == 0 {
		cfg.OtherMetadataScore = 35
	}

	// TF-IDF
	if cfg.MaxTFIDFMultiplier == 0 {
		cfg.MaxTFIDFMultiplier = 2.0
	}
	// TFIDFEnabled defaults to true (handled separately since it's a bool)

	// Position boost
	if cfg.PositionBoostThreshold == 0 {
		cfg.PositionBoostThreshold = 0.1
	}
	if cfg.PositionBoostMultiplier == 0 {
		cfg.PositionBoostMultiplier = 1.3
	}

	// Recency
	if cfg.Recency24hMultiplier == 0 {
		cfg.Recency24hMultiplier = 1.2
	}
	if cfg.RecencyWeekMultiplier == 0 {
		cfg.RecencyWeekMultiplier = 1.1
	}
	if cfg.RecencyMonthMultiplier == 0 {
		cfg.RecencyMonthMultiplier = 1.05
	}

	// Query quality
	if cfg.PhraseMatchMultiplier == 0 {
		cfg.PhraseMatchMultiplier = 1.3
	}
	if cfg.AllWordsMultiplier == 0 {
		cfg.AllWordsMultiplier = 1.0
	}
	if cfg.PartialMatchMultiplier == 0 {
		cfg.PartialMatchMultiplier = 0.7
	}
}
