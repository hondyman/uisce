package strategies

import (
	"context"
	"encoding/json"
	"fmt"
)

type SLOPayload struct {
	JobID          string          `json:"job_id"`
	HistoricalRuns json.RawMessage `json:"historical_runs"`
	SLOForecast    json.RawMessage `json:"slo_forecast"`
	Contention     json.RawMessage `json:"contention"`
}

type SLOSchedulingRecommendation struct {
	SLORiskSummary      string `json:"sloRiskSummary"`
	RecommendedWindow   string `json:"recommendedWindow"`
	PriorityAdjustment  string `json:"priorityAdjustment"`
	ContentionAvoidance string `json:"contentionAvoidance"`
	Justification       string `json:"justification"`
}

type SLOStrategy struct{}

func (s *SLOStrategy) BuildPrompt(payload json.RawMessage) (string, string) {
	var p SLOPayload
	_ = json.Unmarshal(payload, &p)

	system := `You are an expert scheduler and SLO engineer.
Recommend scheduling parameters based on SLO pressure.
Return JSON with sloRiskSummary, recommendedWindow, priorityAdjustment.`

	user := fmt.Sprintf(`
Job ID: %s
Runs: %s
Forecast: %s
Contention: %s

Return SLOSchedulingRecommendation JSON.
`, p.JobID, p.HistoricalRuns, p.SLOForecast, p.Contention)

	return system, user
}

func (s *SLOStrategy) Validate(raw string) bool {
	var out SLOSchedulingRecommendation
	clean := cleanJSON(raw)
	return json.Unmarshal([]byte(clean), &out) == nil && out.SLORiskSummary != ""
}

func (s *SLOStrategy) Parse(raw string) (any, error) {
	var out SLOSchedulingRecommendation
	clean := cleanJSON(raw)
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SLOStrategy) Apply(ctx context.Context, output any) error {
	return nil
}
