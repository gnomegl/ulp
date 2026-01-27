package freshness

import "time"

type Score struct {
	FreshnessScore      float64 `json:"freshness_score"`
	FreshnessCategory   string  `json:"freshness_category"`
	DuplicatePercentage float64 `json:"duplicate_percentage"`
	TotalLinesProcessed int     `json:"total_lines_processed"`
	ValidCredentials    int     `json:"valid_credentials"`
	DuplicatesRemoved   int     `json:"duplicates_removed"`
	AlgorithmVersion    string  `json:"scoring_algorithm_version"`
}

type Config struct {
	MinScore               float64
	MaxScore               float64
	DuplicateThresholds    []DuplicateThreshold
	SizeBonusThreshold     int
	SizeBonusAmount        float64
	SizeBonusMaxDuplicates float64
	AgePenaltyDays         int
	AgePenaltyMax          float64
}

type DuplicateThreshold struct {
	MaxPercent float64
	Score      float64
}

type Calculator interface {
	Calculate(totalLines, validLines, duplicateLines int, fileDate *time.Time, fileSizeBytes int64) *Score
	GetCategory(score float64) string
}

func DefaultConfig() *Config {
	return &Config{
		MinScore: 1.0,
		MaxScore: 5.0,
		DuplicateThresholds: []DuplicateThreshold{
			{MaxPercent: 0.05, Score: 5.0}, // < 5% duplicates: Score 5 (excellent)
			{MaxPercent: 0.15, Score: 4.0}, // 5-15% duplicates: Score 4 (good)
			{MaxPercent: 0.35, Score: 3.0}, // 15-35% duplicates: Score 3 (fair)
			{MaxPercent: 0.60, Score: 2.0}, // 35-60% duplicates: Score 2 (poor)
			{MaxPercent: 1.00, Score: 1.0}, // 60%+ duplicates: Score 1 (stale)
		},
		SizeBonusThreshold:     1000,
		SizeBonusAmount:        0.5,
		SizeBonusMaxDuplicates: 0.10,
		AgePenaltyDays:         30,
		AgePenaltyMax:          1.0,
	}
}
