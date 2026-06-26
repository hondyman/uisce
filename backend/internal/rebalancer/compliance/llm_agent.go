package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// IPSComplianceAgent uses an LLM to parse unstructured IPS constraints
type IPSComplianceAgent struct {
	client *genai.Client
	model  *genai.GenerativeModel
}

func NewIPSComplianceAgent(ctx context.Context, apiKey string) (*IPSComplianceAgent, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	model := client.GenerativeModel("gemini-1.5-pro")
	model.SetTemperature(0.0) // Deterministic output

	return &IPSComplianceAgent{
		client: client,
		model:  model,
	}, nil
}

type ComplianceResult struct {
	Compliant       bool   `json:"compliant"`
	ViolationReason string `json:"violation_reason,omitempty"`
}

// CheckCompliance verifies if a company profile meets the IPS constraints
func (a *IPSComplianceAgent) CheckCompliance(ctx context.Context, ipsText string, companyProfile string) (*ComplianceResult, error) {
	prompt := fmt.Sprintf(`
You are a strict Compliance Officer. Analyze the following company profile against the client's Investment Policy Statement (IPS) constraints.

IPS Constraints:
%s

Company Profile:
%s

Determine if investing in this company violates the IPS.
Respond with a JSON object in the following format:
{
  "compliant": boolean,
  "violation_reason": "string (only if compliant is false)"
}
`, ipsText, companyProfile)

	resp, err := a.model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	if len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil {
		return nil, fmt.Errorf("empty response from LLM")
	}

	// Extract text from response
	var responseText string
	for _, part := range resp.Candidates[0].Content.Parts {
		if txt, ok := part.(genai.Text); ok {
			responseText += string(txt)
		}
	}

	// Clean up markdown code blocks if present
	responseText = strings.TrimPrefix(responseText, "```json")
	responseText = strings.TrimPrefix(responseText, "```")
	responseText = strings.TrimSuffix(responseText, "```")
	responseText = strings.TrimSpace(responseText)

	var result ComplianceResult
	if err := json.Unmarshal([]byte(responseText), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &result, nil
}

func (a *IPSComplianceAgent) Close() {
	a.client.Close()
}
