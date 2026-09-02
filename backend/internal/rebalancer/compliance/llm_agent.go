package compliance

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/uisce/backend/pkg/llm"
)

// IPSComplianceAgent uses an LLM to parse unstructured IPS constraints. Uses
// the centralized pkg/llm Gemini provider (deterministic, temperature 0)
// rather than its own SDK client.
type IPSComplianceAgent struct {
	provider *llm.GeminiProvider
}

func NewIPSComplianceAgent(ctx context.Context, apiKey string) (*IPSComplianceAgent, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}
	provider := llm.NewGeminiProvider(apiKey, "").WithGenerationConfig(0.0, 0.95, 40, 8192)
	return &IPSComplianceAgent{provider: provider}, nil
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

	responseText, err := a.provider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}
	if responseText == "" {
		return nil, fmt.Errorf("empty response from LLM")
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

// Close is a no-op: pkg/llm.GeminiProvider is a plain HTTP client with
// nothing to tear down. Kept for callers that defer a.Close().
func (a *IPSComplianceAgent) Close() {}
