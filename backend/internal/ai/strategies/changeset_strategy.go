package strategies

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type ChangeSetPayload struct {
	ObjectType       string          `json:"object_type"`
	OldValue         json.RawMessage `json:"old_value"`
	NewValue         json.RawMessage `json:"new_value"`
	SemanticImpact   json.RawMessage `json:"semantic_impact"`
	ComplianceImpact json.RawMessage `json:"compliance_impact"`
}

type ChangeSetAISummary struct {
	Title              string   `json:"title"`
	Summary            string   `json:"summary"`
	RiskScore          float64  `json:"riskScore"`
	RiskLevel          string   `json:"riskLevel"`
	SuggestedReviewers []string `json:"suggestedReviewers"`
}

type ChangeSetStrategy struct{}

func (s *ChangeSetStrategy) BuildPrompt(payload json.RawMessage) (string, string) {
	var p ChangeSetPayload
	_ = json.Unmarshal(payload, &p)

	system := `You are an expert metadata governance analyst.
Your task is to generate a ChangeSet summary in strict JSON format.
You must:
1. Write a concise title.
2. Summarize the change in business-friendly language.
3. Assign a risk score (0.0–1.0).
4. Assign a risk level (LOW/MEDIUM/HIGH).
5. Suggest reviewers (roles, not individuals).
6. Return ONLY valid JSON matching the schema.`

	user := fmt.Sprintf(`
Object Type: %s
Old Value: %s
New Value: %s
Semantic Impact: %s
Compliance Impact: %s

Generate a ChangeSet summary JSON object with title, summary, riskScore, riskLevel, and suggestedReviewers.
`, p.ObjectType, p.OldValue, p.NewValue, p.SemanticImpact, p.ComplianceImpact)

	return system, user
}

func (s *ChangeSetStrategy) Validate(raw string) bool {
	// Basic JSON check
	var out ChangeSetAISummary
	clean := cleanJSON(raw)
	return json.Unmarshal([]byte(clean), &out) == nil && out.Title != ""
}

func (s *ChangeSetStrategy) Parse(raw string) (any, error) {
	var out ChangeSetAISummary
	clean := cleanJSON(raw)
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ChangeSetStrategy) Apply(ctx context.Context, output any) error {
	// In a real implementation, this might update the ChangeSet record directly
	// or trigger a notification. For now, the output stored in ai_requests is sufficient
	// for the UI to poll and display.
	return nil
}

// Helper to remove markdown fences
func cleanJSON(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	return s
}
