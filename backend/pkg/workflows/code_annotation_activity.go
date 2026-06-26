package workflows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/hondyman/semlayer/backend/pkg/migration"
	"go.temporal.io/sdk/activity"
)

// CodeAnnotationActivities handles LLM-based code analysis
type CodeAnnotationActivities struct {
	ConfigService *llm.LLMConfigService
	ASTParser     *migration.ASTParserService
}

func NewCodeAnnotationActivities(cfgSvc *llm.LLMConfigService) *CodeAnnotationActivities {
	return &CodeAnnotationActivities{
		ConfigService: cfgSvc,
		ASTParser:     migration.NewASTParserService(),
	}
}

// AnnotationResult contains the LLM's business intent analysis
type AnnotationResult struct {
	NodeID      string  `json:"nodeId"`
	Type        string  `json:"type"`        // "condition", "action", "validation", "transformation"
	Description string  `json:"description"` // Human-readable business intent
	Confidence  float64 `json:"confidence"`  // 0-1 confidence score
}

// BusinessRuleIntent is the aggregated output of code analysis
type BusinessRuleIntent struct {
	RuleID         string             `json:"ruleId"`
	Summary        string             `json:"summary"`
	Preconditions  []AnnotationResult `json:"preconditions"`
	Actions        []AnnotationResult `json:"actions"`
	Postconditions []AnnotationResult `json:"postconditions,omitempty"`
}

// ActivityAnnotateCode parses code and annotates AST nodes with business intent
func (a *CodeAnnotationActivities) ActivityAnnotateCode(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting code annotation activity")

	// Extract inputs
	sourceCode, _ := state["sourceCode"].(string)
	language, _ := state["language"].(string)
	if language == "" {
		language = "java" // Default
	}

	// 1. Parse code into AST
	parsed, err := a.ASTParser.Parse(sourceCode, language)
	if err != nil {
		return nil, fmt.Errorf("failed to parse code: %w", err)
	}

	// 2. Get LLM config
	cfg, err := a.ConfigService.Get()
	if err != nil {
		return nil, fmt.Errorf("LLM config unavailable: %w", err)
	}
	provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)

	// 3. Annotate AST nodes
	annotations, err := a.annotateNodes(ctx, provider, parsed.RootNode, sourceCode)
	if err != nil {
		return nil, fmt.Errorf("annotation failed: %w", err)
	}

	// 4. Aggregate into business rule
	rule := a.aggregateAnnotations(annotations)

	// 5. Return results
	astJSON, _ := json.Marshal(parsed)
	ruleJSON, _ := json.Marshal(rule)

	return map[string]interface{}{
		"ast_json":         string(astJSON),
		"extracted_intent": string(ruleJSON),
		"annotation_count": len(annotations),
	}, nil
}

func (a *CodeAnnotationActivities) annotateNodes(ctx context.Context, provider *llm.GeminiProvider, node *migration.ASTNode, sourceCode string) ([]AnnotationResult, error) {
	var results []AnnotationResult

	if node == nil {
		return results, nil
	}

	// Only annotate interesting nodes
	if node.Type == "if" || node.Type == "function" || node.Type == "call" {
		annotation, err := a.annotateNode(ctx, provider, node, sourceCode)
		if err == nil && annotation != nil {
			results = append(results, *annotation)
		}
	}

	// Recurse into children
	for _, child := range node.Children {
		childResults, _ := a.annotateNodes(ctx, provider, &child, sourceCode)
		results = append(results, childResults...)
	}

	return results, nil
}

func (a *CodeAnnotationActivities) annotateNode(ctx context.Context, provider *llm.GeminiProvider, node *migration.ASTNode, sourceCode string) (*AnnotationResult, error) {
	prompt := fmt.Sprintf(`Analyze this code structure and describe its business purpose in one sentence.

Node Type: %s
Node Name: %s
Condition: %s
Lines: %d-%d

Respond with JSON: {"type": "condition|action|validation|transformation", "description": "...", "confidence": 0.0-1.0}`,
		node.Type, node.Name, node.Condition, node.LineStart, node.LineEnd)

	response, err := provider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	var result AnnotationResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Fallback to simple parsing
		result = AnnotationResult{
			NodeID:      node.ID,
			Type:        node.Type,
			Description: response,
			Confidence:  0.5,
		}
	}
	result.NodeID = node.ID

	return &result, nil
}

func (a *CodeAnnotationActivities) aggregateAnnotations(annotations []AnnotationResult) *BusinessRuleIntent {
	rule := &BusinessRuleIntent{
		RuleID: fmt.Sprintf("rule_%d", len(annotations)),
	}

	summaryParts := []string{}
	for _, ann := range annotations {
		switch ann.Type {
		case "condition":
			rule.Preconditions = append(rule.Preconditions, ann)
		case "action", "transformation":
			rule.Actions = append(rule.Actions, ann)
		case "validation":
			rule.Postconditions = append(rule.Postconditions, ann)
		default:
			rule.Actions = append(rule.Actions, ann)
		}
		if ann.Description != "" {
			summaryParts = append(summaryParts, ann.Description)
		}
	}

	if len(summaryParts) > 0 {
		rule.Summary = fmt.Sprintf("Business Rule: %s", summaryParts[0])
		if len(summaryParts) > 1 {
			rule.Summary += fmt.Sprintf(" (and %d more steps)", len(summaryParts)-1)
		}
	}

	return rule
}
