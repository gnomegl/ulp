package freshness

import (
	"math"
	"time"
)

type DefaultCalculator struct {
	config *Config
}

func NewDefaultCalculator() *DefaultCalculator {
	return &DefaultCalculator{
		config: DefaultConfig(),
	}
}

func NewCalculatorWithConfig(config *Config) *DefaultCalculator {
	return &DefaultCalculator{
		config: config,
	}
}

func (c *DefaultCalculator) Calculate(totalLines, validLines, duplicateLines int, fileDate *time.Time, fileSizeBytes int64) *Score {
	// Calculate duplicate percentage
	duplicatePercentage := 0.0
	if totalLines > 0 {
		duplicatePercentage = float64(duplicateLines) / float64(totalLines)
	}

	// Get base score from duplicate percentage
	score := c.getBaseScoreFromDuplicates(duplicatePercentage)

	// Apply size bonus for large files with very low duplicates
	if validLines >= c.config.SizeBonusThreshold && duplicatePercentage <= c.config.SizeBonusMaxDuplicates {
		score += c.config.SizeBonusAmount
	}

	// Apply age penalty if date is available
	if fileDate != nil {
		agePenalty := c.calculateAgePenalty(*fileDate)
		score -= agePenalty
	}

	// Clamp score to valid range
	score = math.Max(c.config.MinScore, math.Min(c.config.MaxScore, score))

	// Round to 1 decimal place
	score = math.Round(score*10) / 10

	return &Score{
		FreshnessScore:      score,
		FreshnessCategory:   c.GetCategory(score),
		DuplicatePercentage: duplicatePercentage,
		TotalLinesProcessed: totalLines,
		ValidCredentials:    validLines,
		DuplicatesRemoved:   duplicateLines,
		AlgorithmVersion:    "1.0",
	}
}

func (c *DefaultCalculator) getBaseScoreFromDuplicates(duplicatePercentage float64) float64 {
	for _, threshold := range c.config.DuplicateThresholds {
		if duplicatePercentage < threshold.MaxPercent {
			return threshold.Score
		}
	}
	// If we get here, return the lowest score
	return 1.0
}

func (c *DefaultCalculator) calculateAgePenalty(fileDate time.Time) float64 {
	currentTime := time.Now()
	ageDays := currentTime.Sub(fileDate).Hours() / 24

	penaltyThreshold := float64(c.config.AgePenaltyDays)
	maxPenalty := c.config.AgePenaltyMax

	if ageDays <= penaltyThreshold {
		return 0.0
	}

	// Linear penalty after threshold
	penalty := (ageDays - penaltyThreshold) / (365 - penaltyThreshold) * maxPenalty

	if penalty > maxPenalty {
		return maxPenalty
	}
	return penalty
}

func (c *DefaultCalculator) GetCategory(score float64) string {
	if score >= 4.5 {
		return "excellent"
	} else if score >= 3.5 {
		return "good"
	} else if score >= 2.5 {
		return "fair"
	} else if score >= 1.5 {
		return "poor"
	} else {
		return "stale"
	}
}
