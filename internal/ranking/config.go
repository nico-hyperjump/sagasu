package ranking

// RankingConfig holds all configuration for the ranking system.
type RankingConfig struct {
	// Weights for different scoring components
	FilenameWeight  float64 `yaml:"filename_weight"`  // default: 1.5
	ContentWeight   float64 `yaml:"content_weight"`   // default: 1.0
	PathWeight      float64 `yaml:"path_weight"`      // default: 0.3
	MetadataWeight  float64 `yaml:"metadata_weight"`  // default: 0.4

	// Filename scoring values
	ExactFilenameScore       float64 `yaml:"exact_filename_score"`        // default: 100
	AllWordsInOrderScore     float64 `yaml:"all_words_in_order_score"`    // default: 90
	AllWordsAnyOrderScore    float64 `yaml:"all_words_any_order_score"`   // default: 80
	SubstringMatchScore      float64 `yaml:"substring_match_score"`       // default: 60
	PrefixMatchScore         float64 `yaml:"prefix_match_score"`          // default: 45
	MultipleOccurrenceBonus  float64 `yaml:"multiple_occurrence_bonus"`   // default: 10
	ExtensionMatchScore      float64 `yaml:"extension_match_score"`       // default: 60

	// Content scoring values
	PhraseMatchScore         float64 `yaml:"phrase_match_score"`          // default: 120
	HeaderMatchScore         float64 `yaml:"header_match_score"`          // default: 110
	AllWordsContentScore     float64 `yaml:"all_words_content_score"`     // default: 90
	ScatteredWordsScore      float64 `yaml:"scattered_words_score"`       // default: 70
	StemmingMatchScore       float64 `yaml:"stemming_match_score"`        // default: 55

	// Path scoring values
	PathExactMatchScore      float64 `yaml:"path_exact_match_score"`      // default: 40
	PathPartialMatchScore    float64 `yaml:"path_partial_match_score"`    // default: 30
	PathComponentBonus       float64 `yaml:"path_component_bonus"`        // default: 10

	// Metadata scoring values
	AuthorMatchScore         float64 `yaml:"author_match_score"`          // default: 45
	TagMatchScore            float64 `yaml:"tag_match_score"`             // default: 40
	OtherMetadataScore       float64 `yaml:"other_metadata_score"`        // default: 35

	// TF-IDF settings
	MaxTFIDFMultiplier       float64 `yaml:"max_tfidf_multiplier"`        // default: 2.0
	TFIDFEnabled             bool    `yaml:"tfidf_enabled"`               // default: true

	// Position-based scoring
	PositionBoostEnabled     bool    `yaml:"position_boost_enabled"`      // default: true
	PositionBoostThreshold   float64 `yaml:"position_boost_threshold"`    // default: 0.1 (first 10%)
	PositionBoostMultiplier  float64 `yaml:"position_boost_multiplier"`   // default: 1.3

	// Recency multiplier settings
	RecencyEnabled           bool    `yaml:"recency_enabled"`             // default: true
	Recency24hMultiplier     float64 `yaml:"recency_24h_multiplier"`      // default: 1.2
	RecencyWeekMultiplier    float64 `yaml:"recency_week_multiplier"`     // default: 1.1
	RecencyMonthMultiplier   float64 `yaml:"recency_month_multiplier"`    // default: 1.05

	// Query quality multipliers
	QueryQualityEnabled      bool    `yaml:"query_quality_enabled"`       // default: true
	PhraseMatchMultiplier    float64 `yaml:"phrase_match_multiplier"`     // default: 1.3
	AllWordsMultiplier       float64 `yaml:"all_words_multiplier"`        // default: 1.0
	PartialMatchMultiplier   float64 `yaml:"partial_match_multiplier"`    // default: 0.7

	// File size normalization
	FileSizeNormEnabled      bool    `yaml:"file_size_norm_enabled"`      // default: false
}

// DefaultRankingConfig returns the default ranking configuration.
func DefaultRankingConfig() *RankingConfig {
	return &RankingConfig{
		// Weights
		FilenameWeight:  1.5,
		ContentWeight:   1.0,
		PathWeight:      0.3,
		MetadataWeight:  0.4,

		// Filename scoring
		ExactFilenameScore:      100,
		AllWordsInOrderScore:    90,
		AllWordsAnyOrderScore:   80,
		SubstringMatchScore:     60,
		PrefixMatchScore:        45,
		MultipleOccurrenceBonus: 10,
		ExtensionMatchScore:     60,

		// Content scoring
		PhraseMatchScore:     120,
		HeaderMatchScore:     110,
		AllWordsContentScore: 90,
		ScatteredWordsScore:  70,
		StemmingMatchScore:   55,

		// Path scoring
		PathExactMatchScore:   40,
		PathPartialMatchScore: 30,
		PathComponentBonus:    10,

		// Metadata scoring
		AuthorMatchScore:   45,
		TagMatchScore:      40,
		OtherMetadataScore: 35,

		// TF-IDF
		MaxTFIDFMultiplier: 2.0,
		TFIDFEnabled:       true,

		// Position boost
		PositionBoostEnabled:    true,
		PositionBoostThreshold:  0.1,
		PositionBoostMultiplier: 1.3,

		// Recency
		RecencyEnabled:        true,
		Recency24hMultiplier:  1.2,
		RecencyWeekMultiplier: 1.1,
		RecencyMonthMultiplier: 1.05,

		// Query quality
		QueryQualityEnabled:    true,
		PhraseMatchMultiplier:  1.3,
		AllWordsMultiplier:     1.0,
		PartialMatchMultiplier: 0.7,

		// File size
		FileSizeNormEnabled: false,
	}
}

// ApplyDefaults fills in zero values with defaults.
func (c *RankingConfig) ApplyDefaults() {
	defaults := DefaultRankingConfig()

	if c.FilenameWeight == 0 {
		c.FilenameWeight = defaults.FilenameWeight
	}
	if c.ContentWeight == 0 {
		c.ContentWeight = defaults.ContentWeight
	}
	if c.PathWeight == 0 {
		c.PathWeight = defaults.PathWeight
	}
	if c.MetadataWeight == 0 {
		c.MetadataWeight = defaults.MetadataWeight
	}

	// Filename scoring
	if c.ExactFilenameScore == 0 {
		c.ExactFilenameScore = defaults.ExactFilenameScore
	}
	if c.AllWordsInOrderScore == 0 {
		c.AllWordsInOrderScore = defaults.AllWordsInOrderScore
	}
	if c.AllWordsAnyOrderScore == 0 {
		c.AllWordsAnyOrderScore = defaults.AllWordsAnyOrderScore
	}
	if c.SubstringMatchScore == 0 {
		c.SubstringMatchScore = defaults.SubstringMatchScore
	}
	if c.PrefixMatchScore == 0 {
		c.PrefixMatchScore = defaults.PrefixMatchScore
	}
	if c.MultipleOccurrenceBonus == 0 {
		c.MultipleOccurrenceBonus = defaults.MultipleOccurrenceBonus
	}
	if c.ExtensionMatchScore == 0 {
		c.ExtensionMatchScore = defaults.ExtensionMatchScore
	}

	// Content scoring
	if c.PhraseMatchScore == 0 {
		c.PhraseMatchScore = defaults.PhraseMatchScore
	}
	if c.HeaderMatchScore == 0 {
		c.HeaderMatchScore = defaults.HeaderMatchScore
	}
	if c.AllWordsContentScore == 0 {
		c.AllWordsContentScore = defaults.AllWordsContentScore
	}
	if c.ScatteredWordsScore == 0 {
		c.ScatteredWordsScore = defaults.ScatteredWordsScore
	}
	if c.StemmingMatchScore == 0 {
		c.StemmingMatchScore = defaults.StemmingMatchScore
	}

	// Path scoring
	if c.PathExactMatchScore == 0 {
		c.PathExactMatchScore = defaults.PathExactMatchScore
	}
	if c.PathPartialMatchScore == 0 {
		c.PathPartialMatchScore = defaults.PathPartialMatchScore
	}
	if c.PathComponentBonus == 0 {
		c.PathComponentBonus = defaults.PathComponentBonus
	}

	// Metadata scoring
	if c.AuthorMatchScore == 0 {
		c.AuthorMatchScore = defaults.AuthorMatchScore
	}
	if c.TagMatchScore == 0 {
		c.TagMatchScore = defaults.TagMatchScore
	}
	if c.OtherMetadataScore == 0 {
		c.OtherMetadataScore = defaults.OtherMetadataScore
	}

	// TF-IDF
	if c.MaxTFIDFMultiplier == 0 {
		c.MaxTFIDFMultiplier = defaults.MaxTFIDFMultiplier
	}

	// Position boost
	if c.PositionBoostThreshold == 0 {
		c.PositionBoostThreshold = defaults.PositionBoostThreshold
	}
	if c.PositionBoostMultiplier == 0 {
		c.PositionBoostMultiplier = defaults.PositionBoostMultiplier
	}

	// Recency
	if c.Recency24hMultiplier == 0 {
		c.Recency24hMultiplier = defaults.Recency24hMultiplier
	}
	if c.RecencyWeekMultiplier == 0 {
		c.RecencyWeekMultiplier = defaults.RecencyWeekMultiplier
	}
	if c.RecencyMonthMultiplier == 0 {
		c.RecencyMonthMultiplier = defaults.RecencyMonthMultiplier
	}

	// Query quality
	if c.PhraseMatchMultiplier == 0 {
		c.PhraseMatchMultiplier = defaults.PhraseMatchMultiplier
	}
	if c.AllWordsMultiplier == 0 {
		c.AllWordsMultiplier = defaults.AllWordsMultiplier
	}
	if c.PartialMatchMultiplier == 0 {
		c.PartialMatchMultiplier = defaults.PartialMatchMultiplier
	}
}
