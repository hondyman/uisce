package ai

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/pkg/llm"
)

// RuleTemplate is an AI-generated reusable rule pattern derived from usage clusters
type RuleTemplate struct {
	ID               uuid.UUID              `json:"id"`
	TenantID         uuid.UUID              `json:"tenant_id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	RuleType         string                 `json:"rule_type"`
	ParameterSchema  map[string]interface{} `json:"parameter_schema"`
	ExampleCondition map[string]interface{} `json:"example_condition"`
	Confidence       float64                `json:"confidence"`
	UsageCount       int                    `json:"usage_count"`
	SemanticTerms    []string               `json:"semantic_terms"`
	Tags             []string               `json:"tags"`
	GeneratedAt      time.Time              `json:"generated_at"`
}

// RuleTemplateSuggestion wraps a template with recommendation context
type RuleTemplateSuggestion struct {
	Template   RuleTemplate `json:"template"`
	Rationale  string       `json:"rationale"`
	AppliesTo  []string     `json:"applies_to"` // business object names
	Confidence float64      `json:"confidence"`
}

// ruleCluster groups similar rules for template extraction
type ruleCluster struct {
	ruleType      string
	conditions    []map[string]interface{}
	usageCount    int
	semanticTerms []string
}

// RuleTemplateEngine generates AI-assisted rule templates by analysing existing rules
type RuleTemplateEngine struct {
	db          *sql.DB
	llmProvider llm.LLMProvider
}

func NewRuleTemplateEngine(db *sql.DB, llmProvider llm.LLMProvider) *RuleTemplateEngine {
	return &RuleTemplateEngine{db: db, llmProvider: llmProvider}
}

// SuggestTemplates analyses historical rule patterns and returns reusable templates
func (e *RuleTemplateEngine) SuggestTemplates(ctx context.Context, tenantID uuid.UUID) ([]RuleTemplateSuggestion, error) {
	// 1. Load existing rules for the tenant
	patterns, err := e.loadRulePatterns(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load rule patterns: %w", err)
	}
	if len(patterns) == 0 {
		return e.generateDefaultTemplates(ctx, tenantID), nil
	}

	// 2. Cluster by rule_type
	clusters := e.clusterPatterns(patterns)

	// 3. Generate a template per cluster
	suggestions := make([]RuleTemplateSuggestion, 0, len(clusters))
	for _, cluster := range clusters {
		tmpl := e.extractTemplate(tenantID, cluster)

		// 4. Optionally enrich name/description via LLM
		if e.llmProvider != nil {
			e.enrichTemplate(ctx, &tmpl)
		}

		suggestions = append(suggestions, RuleTemplateSuggestion{
			Template:   tmpl,
			Rationale:  fmt.Sprintf("Derived from %d similar rules with %.0f%% confidence.", cluster.usageCount, tmpl.Confidence*100),
			AppliesTo:  cluster.semanticTerms,
			Confidence: tmpl.Confidence,
		})
	}

	// 5. Sort by confidence descending
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	return suggestions, nil
}

// loadRulePatterns fetches existing rule conditions from the ai_training_data store
func (e *RuleTemplateEngine) loadRulePatterns(ctx context.Context, tenantID uuid.UUID) ([]map[string]interface{}, error) {
	rows, err := e.db.QueryContext(ctx, `
		SELECT input FROM edm.ai_training_data
		WHERE tenant_id = $1 AND source IN ('rule_creation', 'user_feedback')
		ORDER BY created_at DESC LIMIT 200
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var patterns []map[string]interface{}
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			continue
		}
		var p map[string]interface{}
		if err := json.Unmarshal(raw, &p); err == nil {
			patterns = append(patterns, p)
		}
	}
	return patterns, nil
}

// clusterPatterns groups rule patterns by their rule_type field
func (e *RuleTemplateEngine) clusterPatterns(patterns []map[string]interface{}) []ruleCluster {
	byType := map[string]*ruleCluster{}
	for _, p := range patterns {
		rt, _ := p["rule_type"].(string)
		if rt == "" {
			rt = "general"
		}
		if _, ok := byType[rt]; !ok {
			byType[rt] = &ruleCluster{ruleType: rt}
		}
		c := byType[rt]
		c.usageCount++
		if cond, ok := p["condition"].(map[string]interface{}); ok {
			c.conditions = append(c.conditions, cond)
		}
		if term, ok := p["semantic_term"].(string); ok && term != "" {
			c.semanticTerms = append(c.semanticTerms, term)
		}
	}
	result := make([]ruleCluster, 0, len(byType))
	for _, c := range byType {
		result = append(result, *c)
	}
	return result
}

// extractTemplate creates a RuleTemplate from a cluster
func (e *RuleTemplateEngine) extractTemplate(tenantID uuid.UUID, cluster ruleCluster) RuleTemplate {
	// Build a parameter schema from the most common condition keys
	paramSchema := map[string]interface{}{}
	fieldFreq := map[string]int{}
	for _, cond := range cluster.conditions {
		for k := range cond {
			fieldFreq[k]++
		}
	}
	for field, freq := range fieldFreq {
		if freq >= 2 { // appears in at least 2 rules
			paramSchema[field] = map[string]interface{}{"type": "string", "description": field}
		}
	}

	// Example condition: pick the first one present
	var exampleCond map[string]interface{}
	if len(cluster.conditions) > 0 {
		exampleCond = cluster.conditions[0]
	}

	// Deduplicate semantic terms
	termSet := map[string]struct{}{}
	for _, t := range cluster.semanticTerms {
		termSet[t] = struct{}{}
	}
	terms := make([]string, 0, len(termSet))
	for t := range termSet {
		terms = append(terms, t)
	}

	confidence := float64(cluster.usageCount) / 20.0 // saturates at 20 uses
	if confidence > 0.95 {
		confidence = 0.95
	}

	return RuleTemplate{
		ID:               uuid.New(),
		TenantID:         tenantID,
		Name:             fmt.Sprintf("%s Rule Template", cluster.ruleType),
		Description:      fmt.Sprintf("Auto-generated template based on %d similar %s rules.", cluster.usageCount, cluster.ruleType),
		RuleType:         cluster.ruleType,
		ParameterSchema:  paramSchema,
		ExampleCondition: exampleCond,
		Confidence:       confidence,
		UsageCount:       cluster.usageCount,
		SemanticTerms:    terms,
		GeneratedAt:      time.Now(),
	}
}

// enrichTemplate calls the LLM to produce a better name and description
func (e *RuleTemplateEngine) enrichTemplate(ctx context.Context, tmpl *RuleTemplate) {
	condJSON, _ := json.Marshal(tmpl.ExampleCondition)
	prompt := fmt.Sprintf(`You are a business rules expert. Given a rule template:
Rule type: %s
Example condition: %s
Usage count: %d
Semantic terms: %v

Respond with a short JSON object with keys "name" (max 60 chars) and "description" (max 200 chars).
Make both professional and precise for enterprise use. Return only valid JSON.`,
		tmpl.RuleType, string(condJSON), tmpl.UsageCount, tmpl.SemanticTerms)

	resp, err := e.llmProvider.GenerateResponse(ctx, prompt)
	if err != nil {
		return
	}
	var enriched struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal([]byte(cleanJSON(resp)), &enriched); err == nil {
		if enriched.Name != "" {
			tmpl.Name = enriched.Name
		}
		if enriched.Description != "" {
			tmpl.Description = enriched.Description
		}
	}
}

// generateDefaultTemplates returns sensible built-in templates when no history exists
func (e *RuleTemplateEngine) generateDefaultTemplates(ctx context.Context, tenantID uuid.UUID) []RuleTemplateSuggestion {
	defaults := []RuleTemplate{
		{
			ID: uuid.New(), TenantID: tenantID,
			Name:        "Mandatory Field Validation",
			Description: "Ensures a required field is present and non-empty on all records.",
			RuleType:    "MANDATORY",
			ParameterSchema: map[string]interface{}{
				"field_path": map[string]interface{}{"type": "string", "description": "Field to validate"},
			},
			ExampleCondition: map[string]interface{}{"field_path": "$.entity.field", "operator": "IS_NOT_NULL"},
			Confidence:       0.90, UsageCount: 0, GeneratedAt: time.Now(),
		},
		{
			ID: uuid.New(), TenantID: tenantID,
			Name:        "Value Range Constraint",
			Description: "Validates that a numeric field falls within an acceptable range.",
			RuleType:    "RANGE",
			ParameterSchema: map[string]interface{}{
				"field_path": map[string]interface{}{"type": "string"},
				"min":        map[string]interface{}{"type": "number"},
				"max":        map[string]interface{}{"type": "number"},
			},
			ExampleCondition: map[string]interface{}{"field_path": "$.entity.amount", "min": 0, "max": 1000000},
			Confidence:       0.85, UsageCount: 0, GeneratedAt: time.Now(),
		},
		{
			ID: uuid.New(), TenantID: tenantID,
			Name:        "Referential Integrity Check",
			Description: "Ensures a foreign key field references a valid entity in another business object.",
			RuleType:    "REFERENTIAL",
			ParameterSchema: map[string]interface{}{
				"source_field":  map[string]interface{}{"type": "string"},
				"target_object": map[string]interface{}{"type": "string"},
			},
			ExampleCondition: map[string]interface{}{"source_field": "$.entity.customer_id", "target_object": "Customer"},
			Confidence:       0.80, UsageCount: 0, GeneratedAt: time.Now(),
		},
	}

	suggestions := make([]RuleTemplateSuggestion, len(defaults))
	for i, t := range defaults {
		suggestions[i] = RuleTemplateSuggestion{
			Template:   t,
			Rationale:  "Built-in template — no tenant history available yet.",
			Confidence: t.Confidence,
		}
	}
	return suggestions
}
