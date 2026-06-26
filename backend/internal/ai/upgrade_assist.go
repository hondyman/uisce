package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/rules"
)

type UpgradeAssistRequest struct {
	TenantID   string
	CoreKey    string
	OldCore    rules.CoreValidationRule
	NewCore    rules.CoreValidationRule
	TenantRule rules.TenantValidationRule
}

type UpgradeSuggestion struct {
	NewExtensionSrc string
	Summary         string
}

type RefactorSuggestion struct {
	NewRules     []map[string]interface{} `json:"newRules"`     // conditionJson + metadata
	RuleRewrites map[string]string        `json:"ruleRewrites"` // oldRuleID -> newRuleKey or explanation
}

type UpgradeAssistService struct {
	llm LLMClient
}

func NewUpgradeAssistService(llm LLMClient) *UpgradeAssistService {
	return &UpgradeAssistService{llm: llm}
}

func (s *UpgradeAssistService) SuggestUpgrade(
	ctx context.Context,
	req UpgradeAssistRequest,
) (*UpgradeSuggestion, error) {
	// Build payload describing the delta
	payload := map[string]interface{}{
		"oldCore": map[string]interface{}{
			"key":     req.OldCore.RuleKey,
			"version": req.OldCore.Version,
			"src":     req.OldCore.ConditionSrc,
		},
		"newCore": map[string]interface{}{
			"key":     req.NewCore.RuleKey,
			"version": req.NewCore.Version,
			"src":     req.NewCore.ConditionSrc,
		},
		"tenantRule": map[string]interface{}{
			"inheritMode":  req.TenantRule.InheritMode,
			"conditionSrc": req.TenantRule.ConditionSrc,
		},
		"instructions": "Propose a new extension compatible with newCore; return JSON {\"newExtensionSrc\": \"...\", \"summary\": \"...\"}.",
	}
	buf, _ := json.Marshal(payload)
	prompt := fmt.Sprintf("Upgrade Assistant Task:\n%s", string(buf))

	out, err := s.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Clean output (simple workaround for code blocks)
	cleanOut := strings.Trim(out, " `\n")
	if strings.HasPrefix(cleanOut, "json") {
		cleanOut = cleanOut[4:]
	}

	var raw struct {
		NewExtensionSrc string `json:"newExtensionSrc"`
		Summary         string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(cleanOut), &raw); err != nil {
		return nil, fmt.Errorf("invalid LLM JSON: %w", err)
	}

	return &UpgradeSuggestion{
		NewExtensionSrc: raw.NewExtensionSrc,
		Summary:         raw.Summary,
	}, nil
}

func (s *UpgradeAssistService) SuggestRefactor(
	ctx context.Context,
	tenantID string,
	rulesCluster []rules.TenantValidationRule,
) (*RefactorSuggestion, error) {
	// Build payload
	payload := map[string]interface{}{
		"rules":        rulesCluster, // stripped down if needed
		"instructions": "Identify redundancies and propose a smaller set of rules. Return JSON {\"newRules\": [...], \"ruleRewrites\": {\"oldRuleId\": \"description or new rule key\"}}.",
	}
	buf, _ := json.Marshal(payload)
	prompt := fmt.Sprintf("Refactor Assistant Task:\n%s", string(buf))

	out, err := s.llm.Complete(ctx, prompt)
	if err != nil {
		return nil, err
	}

	cleanOut := strings.Trim(out, " `\n")
	if strings.HasPrefix(cleanOut, "json") {
		cleanOut = cleanOut[4:]
	}

	var ref RefactorSuggestion
	if err := json.Unmarshal([]byte(cleanOut), &ref); err != nil {
		return nil, err
	}
	return &ref, nil
}
