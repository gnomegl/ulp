package freshness

import (
	"testing"
	"time"
)

func TestCalculateFreshnessScore(t *testing.T) {
	calc := NewDefaultCalculator()

	tests := []struct {
		name             string
		totalLines       int
		validLines       int
		duplicateLines   int
		fileDate         *time.Time
		expectedCategory string
		expectedMinScore float64
		expectedMaxScore float64
	}{
		{
			name:             "Excellent - very low duplicates",
			totalLines:       1000,
			validLines:       980,
			duplicateLines:   20, // 2% duplicates
			expectedCategory: "excellent",
			expectedMinScore: 4.5,
			expectedMaxScore: 5.0,
		},
		{
			name:             "Good - moderate duplicates",
			totalLines:       1000,
			validLines:       900,
			duplicateLines:   100, // 10% duplicates
			expectedCategory: "good",
			expectedMinScore: 3.5,
			expectedMaxScore: 4.5,
		},
		{
			name:             "Fair - higher duplicates",
			totalLines:       1000,
			validLines:       750,
			duplicateLines:   250, // 25% duplicates
			expectedCategory: "fair",
			expectedMinScore: 2.5,
			expectedMaxScore: 3.5,
		},
		{
			name:             "Poor - high duplicates",
			totalLines:       1000,
			validLines:       500,
			duplicateLines:   500, // 50% duplicates
			expectedCategory: "poor",
			expectedMinScore: 1.5,
			expectedMaxScore: 2.5,
		},
		{
			name:             "Stale - very high duplicates",
			totalLines:       1000,
			validLines:       300,
			duplicateLines:   700, // 70% duplicates
			expectedCategory: "stale",
			expectedMinScore: 1.0,
			expectedMaxScore: 1.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calc.Calculate(tt.totalLines, tt.validLines, tt.duplicateLines, tt.fileDate, 0)

			if score.FreshnessCategory != tt.expectedCategory {
				t.Errorf("Expected category %s, got %s", tt.expectedCategory, score.FreshnessCategory)
			}

			if score.FreshnessScore < tt.expectedMinScore || score.FreshnessScore > tt.expectedMaxScore {
				t.Errorf("Expected score between %f and %f, got %f",
					tt.expectedMinScore, tt.expectedMaxScore, score.FreshnessScore)
			}

			expectedDuplicatePercentage := float64(tt.duplicateLines) / float64(tt.totalLines)
			if score.DuplicatePercentage != expectedDuplicatePercentage {
				t.Errorf("Expected duplicate percentage %f, got %f",
					expectedDuplicatePercentage, score.DuplicatePercentage)
			}
		})
	}
}

func TestAgePenalty(t *testing.T) {
	calc := NewDefaultCalculator()

	// Test with old file (should have penalty)
	oldDate := time.Now().AddDate(0, 0, -60) // 60 days ago
	score := calc.Calculate(1000, 950, 50, &oldDate, 0)

	// Test with recent file (should have no penalty)
	recentDate := time.Now().AddDate(0, 0, -10) // 10 days ago
	scoreRecent := calc.Calculate(1000, 950, 50, &recentDate, 0)

	if score.FreshnessScore >= scoreRecent.FreshnessScore {
		t.Errorf("Expected old file to have lower score due to age penalty. Old: %f, Recent: %f",
			score.FreshnessScore, scoreRecent.FreshnessScore)
	}
}

func TestSizeBonus(t *testing.T) {
	calc := NewDefaultCalculator()

	// Large file with low duplicates (should get bonus) - base score 4, +0.5 bonus = 4.5
	scoreLarge := calc.Calculate(2000, 1800, 200, nil, 0) // 10% duplicates, large file

	// Small file with same duplicate percentage (no bonus) - base score 4, no bonus = 4.0
	scoreSmall := calc.Calculate(500, 450, 50, nil, 0) // 10% duplicates, small file

	if scoreLarge.FreshnessScore <= scoreSmall.FreshnessScore {
		t.Errorf("Expected large file to have higher score due to size bonus. Large: %f, Small: %f",
			scoreLarge.FreshnessScore, scoreSmall.FreshnessScore)
	}
}
