package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/rules"
	"github.com/hondyman/semlayer/backend/internal/validation"
)

// Interfaces

type LLMClient interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

type MetadataRepository interface {
	GetEntityDefinition(ctx context.Context, tenantID, entity string) (map[string]interface{}, []string, error)
}

// Request/Response

type SuggestRequest struct {
	TenantID      string
	Entity        string
	Intent        string
	TargetField   *string
	ContextRuleID *string
}

type SuggestResponse struct {
	ConditionJSON         map[string]interface{}
	Description           string
	Severity              string
	InheritModeSuggestion string
	ConflictsWith         []string
	TestFailureRate       float64
	RuntimeOK             bool
}

// Service

type SuggestService struct {
	llm      LLMClient
	ruleRepo rules.RuleRepository
	metaRepo MetadataRepository
	testSvc  *validation.TestService
}

func NewSuggestService(
	llm LLMClient,
	ruleRepo rules.RuleRepository,
	metaRepo MetadataRepository,
	testSvc *validation.TestService,
) *SuggestService {
	return &SuggestService{
		llm:      llm,
		ruleRepo: ruleRepo,
		metaRepo: metaRepo,
		testSvc:  testSvc,
	}
}

func (s *SuggestService) SuggestRule(ctx context.Context, req SuggestRequest) (*SuggestResponse, error) {
	// 1. Load metadata (Stubbed prompt context for now)
	// entityDef, fields, err := s.metaRepo.GetEntityDefinition(ctx, req.TenantID, req.Entity)
	// ...

	// 2. Build structured prompt
	prompt := fmt.Sprintf(`
You are a Rule expert. User intent: "%s" for entity "%s".
Target field: %v.
Generate a valid rule condition JSON.
Return JSON:
{
  "conditionJson": null,
  "description": "...",
  "severity": "warning",
  "inheritModeSuggestion": "CUSTOM"
}
`, req.Intent, req.Entity, req.TargetField)

	// 3. Call LLM
	out, err := s.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// 4. Parse LLM response
	var raw struct {
		ConditionJSON map[string]interface{} `json:"conditionJson"`
		Description   string                 `json:"description"`
		Severity      string                 `json:"severity"`
		InheritMode   string                 `json:"inheritModeSuggestion"`
	}

	// Mock parsing logic in case LLM returns markdown code blocks
	cleanOut := out
	// ... strip markdown ...

	if err := json.Unmarshal([]byte(cleanOut), &raw); err != nil {
		return nil, fmt.Errorf("invalid LLM JSON: %w", err)
	}

	// 5. Normalization & Validation
	if raw.ConditionJSON != nil {
		if err := validation.ValidateConditionTree(raw.ConditionJSON); err != nil {
			return nil, fmt.Errorf("invalid condition tree: %w", err)
		}
	}

	// 6. Conflict detection (Stub)
	conflicts := []string{}

	// 7. Run quick test (StarlarkSrc is empty)
	testRate, runtimeOK := s.testSvc.SmokeTestCondition(ctx, req.TenantID, req.Entity, raw.ConditionJSON)

	return &SuggestResponse{
		ConditionJSON:         raw.ConditionJSON,
		Description:           raw.Description,
		Severity:              raw.Severity,
		InheritModeSuggestion: raw.InheritMode,
		ConflictsWith:         conflicts,
		TestFailureRate:       testRate,
		RuntimeOK:             runtimeOK,
	}, nil
}

// Mock LLM Client (for verify step or if no real client)

type MockLLMClient struct{}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
	// Simple mock response based on prompt or static
	return `{
  "conditionJson": {"type":"condition","field": "amount", "operator": ">=", "value": 1000},
  "description": "Limit amount to 1000",
  "severity": "error",
  "inheritModeSuggestion": "CUSTOM"
}`, nil
}
