package strategies

import (
	"context"
	"encoding/json"
	"fmt"
)

type DriftPayload struct {
	SemanticTermID   string          `json:"semantic_term_id"`
	DriftSignals     json.RawMessage `json:"drift_signals"`
	DownstreamImpact json.RawMessage `json:"downstream_impact"`
}

type SemanticDriftRemediation struct {
	DriftSummary              string   `json:"driftSummary"`
	RootCause                 string   `json:"rootCause"`
	AffectedObjects           []string `json:"affectedObjects"`
	RecommendedFix            string   `json:"recommendedFix"`
	SuggestedChangeSetSummary string   `json:"suggestedChangeSetSummary"`
}

type DriftStrategy struct{}

func (s *DriftStrategy) BuildPrompt(payload json.RawMessage) (string, string) {
	var p DriftPayload
	_ = json.Unmarshal(payload, &p)

	system := `You are an expert semantic architect.
Analyze semantic drift and propose remediation.
Return JSON with driftSummary, rootCause, affectedObjects, proposedFix.`

	user := fmt.Sprintf(`
Term ID: %s
Drift Signals: %s
Impact: %s

Return SemanticDriftRemediation JSON.
`, p.SemanticTermID, p.DriftSignals, p.DownstreamImpact)

	return system, user
}

func (s *DriftStrategy) Validate(raw string) bool {
	var out SemanticDriftRemediation
	clean := cleanJSON(raw)
	return json.Unmarshal([]byte(clean), &out) == nil && out.DriftSummary != ""
}

func (s *DriftStrategy) Parse(raw string) (any, error) {
	var out SemanticDriftRemediation
	clean := cleanJSON(raw)
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *DriftStrategy) Apply(ctx context.Context, output any) error {
	return nil
}
