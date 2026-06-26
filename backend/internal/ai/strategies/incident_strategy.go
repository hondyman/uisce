package strategies

import (
	"context"
	"encoding/json"
	"fmt"
)

type IncidentPayload struct {
	ClusterID        string          `json:"cluster_id"`
	ErrorPattern     string          `json:"error_pattern"`
	AffectedJobs     []string        `json:"affected_jobs"`
	SemanticBindings json.RawMessage `json:"semantic_bindings"`
}

type IncidentNarrative struct {
	Title                     string `json:"title"`
	RootCause                 string `json:"rootCause"`
	BlastRadius               string `json:"blastRadius"`
	RecommendedFix            string `json:"recommendedFix"`
	SuggestedChangeSetSummary string `json:"suggestedChangeSetSummary"`
}

type IncidentStrategy struct{}

func (s *IncidentStrategy) BuildPrompt(payload json.RawMessage) (string, string) {
	var p IncidentPayload
	_ = json.Unmarshal(payload, &p)

	system := `You are an expert SRE and data operations analyst.
Your task is to generate an incident narrative in strict JSON format.
You must:
1. Identify the most likely root cause.
2. Describe the blast radius.
3. Recommend a fix.
4. Propose a ChangeSet summary.
5. Return ONLY valid JSON.`

	user := fmt.Sprintf(`
Error Pattern: %s
Affected Jobs: %v
Semantic Bindings: %s

Generate an Incident Narrative JSON object.
`, p.ErrorPattern, p.AffectedJobs, p.SemanticBindings)

	return system, user
}

func (s *IncidentStrategy) Validate(raw string) bool {
	var out IncidentNarrative
	clean := cleanJSON(raw)
	return json.Unmarshal([]byte(clean), &out) == nil && out.Title != ""
}

func (s *IncidentStrategy) Parse(raw string) (any, error) {
	var out IncidentNarrative
	clean := cleanJSON(raw)
	if err := json.Unmarshal([]byte(clean), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *IncidentStrategy) Apply(ctx context.Context, output any) error {
	// Logic to attach narrative to the Exception Cluster record would go here
	return nil
}
