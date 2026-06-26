package strategies

import (
	"context"
	"encoding/json"
	"fmt"
)

type RiskAssessmentPayload struct {
	ChangeSummary      string          `json:"change_summary"`
	ImpactAnalysis     json.RawMessage `json:"impact_analysis"`
	HistoricalFailures json.RawMessage `json:"historical_failures"`
}

type ChangeSetRiskAssessment struct {
	RiskScore            float64  `json:"riskScore"`
	RiskLevel            string   `json:"riskLevel"`
	Justification        string   `json:"justification"`
	RiskDrivers          []string `json:"riskDrivers"`
	RecommendedApprovers []string `json:"recommendedApprovers"`
}

type RiskStrategy struct{}

func (s *RiskStrategy) BuildPrompt(payload json.RawMessage) (string, string) {
	var p RiskAssessmentPayload
	_ = json.Unmarshal(payload, &p)

	system := `You are an expert metadata governance analyst.
Evaluate risk and return JSON.
You must:
1. Assign riskScore (0.0-1.0).
2. Assign riskLevel (LOW/MEDIUM/HIGH).
3. concise justification.
4. top risk drivers.
5. recommended approver roles.`

	user := fmt.Sprintf(`
Change Summary: %s
Impact Analysis: %s
Historical Failures: %s

Return ChangeSetRiskAssessment JSON.
`, p.ChangeSummary, p.ImpactAnalysis, p.HistoricalFailures)

	return system, user
}

func (s *RiskStrategy) Validate(raw string) bool {
	var out ChangeSetRiskAssessment
	clean := cleanJSON(raw)
	return json.Unmarshal([]byte(clean), &out) == nil && out.RiskLevel != ""
}

func (s *RiskStrategy) Parse(raw string) (any, error) {
	var out ChangeSetRiskAssessment
	clean := cleanJSON(raw)
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RiskStrategy) Apply(ctx context.Context, output any) error {
	return nil
}
