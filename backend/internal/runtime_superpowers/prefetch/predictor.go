package prefetch

import (
	"context"
	"sort"
)

type TransitionEvidence struct {
	FromPageID string `json:"from_page_id"`
	ToPageID   string `json:"to_page_id"`
}

type Prediction struct {
	TargetType string  `json:"target_type"` // page, api
	TargetID   string  `json:"target_id"`
	Confidence float64 `json:"confidence"`
}

type Predictor struct {
	// In-memory Markov chain for MVP
	// Map[FromPageID] -> Map[ToPageID] -> Count
	transitions map[string]map[string]int
}

func NewPredictor() *Predictor {
	return &Predictor{
		transitions: make(map[string]map[string]int),
	}
}

func (p *Predictor) RecordTransition(ctx context.Context, from, to string) error {
	if p.transitions[from] == nil {
		p.transitions[from] = make(map[string]int)
	}
	p.transitions[from][to]++
	return nil
}

func (p *Predictor) PredictNext(ctx context.Context, currentPageID string) ([]Prediction, error) {
	counts, ok := p.transitions[currentPageID]
	if !ok {
		return []Prediction{}, nil
	}

	total := 0
	for _, count := range counts {
		total += count
	}

	predictions := make([]Prediction, 0)
	for next, count := range counts {
		confidence := float64(count) / float64(total)
		if confidence > 0.1 { // Threshold
			predictions = append(predictions, Prediction{
				TargetType: "page",
				TargetID:   next,
				Confidence: confidence,
			})
		}
	}

	// Sort by confidence
	sort.Slice(predictions, func(i, j int) bool {
		return predictions[i].Confidence > predictions[j].Confidence
	})

	return predictions, nil
}
