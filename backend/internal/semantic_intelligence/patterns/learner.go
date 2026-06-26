package patterns

import (
	"context"
	"encoding/json"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type SemanticPattern struct {
	ID          string          `json:"id"`
	BOName      string          `json:"bo_name"`
	Type        string          `json:"type"` // dashboard, detail, list, chart
	Confidence  float64         `json:"confidence"`
	UsageCount  int             `json:"usage_count"`
	Definition  json.RawMessage `json:"definition"` // Generalized component config
	Description string          `json:"description"`
}

type Learner struct {
	repo *pagestudio.Repository
}

func NewLearner(repo *pagestudio.Repository) *Learner {
	return &Learner{repo: repo}
}

// LearnPatterns scans all pages and extracts common patterns
func (l *Learner) LearnPatterns(ctx context.Context) ([]SemanticPattern, error) {
	// 1. Fetch all pages
	// Note: repo.ListPages might need pagination, for MVP simplistic
	pages, err := l.repo.ListPages(ctx, "dev") // Hardcoding dev for now
	if err != nil {
		return nil, err
	}

	// 2. Analyze
	patterns := make([]SemanticPattern, 0)

	// Mock: If we see > 1 page with a chart for "Position", suggests a pattern
	chartCounts := make(map[string]int)

	for _, p := range pages {
		_ = p
		// Extract BO context (assuming p.Metadata or fingerprint has BO)
		// ...
		chartCounts["Positions_Chart"]++
	}

	if count := chartCounts["Positions_Chart"]; count > 0 {
		patterns = append(patterns, SemanticPattern{
			ID:          "pat_pos_chart",
			BOName:      "Position",
			Type:        "chart",
			Confidence:  0.85,
			UsageCount:  count,
			Description: "Standard Position History Chart",
			Definition:  json.RawMessage(`{"type": "line", "x": "date", "y": "market_value"}`),
		})
	}

	return patterns, nil
}
