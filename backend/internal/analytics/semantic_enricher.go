package analytics

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/jmoiron/sqlx"
)

// SemanticEnrichment holds the enriched properties for a semantic term
type SemanticEnrichment struct {
	UIComponent        string                 `json:"ui_component"`
	UIProps            map[string]interface{} `json:"ui_props,omitempty"`
	ValidationRules    []ValidationRule       `json:"validation_rules,omitempty"`
	DisplayHints       map[string]interface{} `json:"display_hints,omitempty"`
	WealthDomain       string                 `json:"wealth_domain,omitempty"`
	BOSubtypeHint      string                 `json:"bo_subtype_hint,omitempty"`
	ConstraintTemplate map[string]interface{} `json:"constraint_template,omitempty"`
}

// ValidationRule represents a single validation rule
type ValidationRule struct {
	Type     string      `json:"type"`
	Field    string      `json:"field,omitempty"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value,omitempty"`
	Severity string      `json:"severity"`
	Message  string      `json:"message,omitempty"`
}

// CubeDefinition represents column metadata for enrichment inference
type CubeDefinition struct {
	Name         string `json:"name"`
	Table        string `json:"table"`
	Column       string `json:"column"`
	DataType     string `json:"data_type"`
	SemanticType string `json:"semantic_type"`
	IsNullable   bool   `json:"is_nullable"`
	IsForeignKey bool   `json:"is_foreign_key"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// SemanticEnricher handles semantic term enrichment
type SemanticEnricher struct {
	db          *sqlx.DB
	llmProvider interface{}
}

// NewSemanticEnricher creates a new enricher instance
func NewSemanticEnricher(db *sqlx.DB, llmProvider interface{}) *SemanticEnricher {
	return &SemanticEnricher{
		db:          db,
		llmProvider: llmProvider,
	}
}

// EnrichSemanticNode enriches a semantic term node with UI, validation, and domain hints
func (e *SemanticEnricher) EnrichSemanticNode(ctx context.Context, nodeID uuid.UUID, cubeDef CubeDefinition) (*SemanticEnrichment, error) {
	logger := logging.GetLogger().Sugar()
	logger.Infof("Enriching semantic node %s for column %s.%s", nodeID, cubeDef.Table, cubeDef.Column)

	enrichment := &SemanticEnrichment{
		UIComponent: "text_input", // Default
		UIProps:     make(map[string]interface{}),
	}

	// 1. Infer UI Component
	enrichment.UIComponent, enrichment.UIProps = e.inferUIComponent(cubeDef)

	// 2. Infer Validation Rules
	enrichment.ValidationRules = e.inferValidationRules(cubeDef)

	// 3. Infer Display Hints
	enrichment.DisplayHints = e.inferDisplayHints(cubeDef)

	// 4. Infer Wealth Domain
	enrichment.WealthDomain = e.inferWealthDomain(cubeDef)

	// 5. Infer BO Subtype Hint
	enrichment.BOSubtypeHint = e.inferBOSubtypeHint(cubeDef)

	// 6. Infer Constraint Template
	enrichment.ConstraintTemplate = e.inferConstraintTemplate(cubeDef)

	// Update node properties in DB
	if nodeID != uuid.Nil {
		if err := e.updateNodeProperties(ctx, nodeID, enrichment); err != nil {
			logger.Warnf("Failed to update node properties: %v", err)
			// Continue - return enrichment even if DB update fails
		}
	}

	return enrichment, nil
}

// inferUIComponent determines the appropriate UI component based on column patterns
func (e *SemanticEnricher) inferUIComponent(def CubeDefinition) (string, map[string]interface{}) {
	upperCol := strings.ToUpper(def.Column)
	upperName := strings.ToUpper(def.Name)
	lowerType := strings.ToLower(def.DataType)
	props := make(map[string]interface{})

	// Currency/Amount patterns
	if strings.Contains(upperCol, "_AMT") || strings.Contains(upperCol, "_AMOUNT") ||
		strings.Contains(upperCol, "_BALANCE") || strings.Contains(upperCol, "_PRICE") ||
		strings.Contains(upperCol, "_COST") || strings.Contains(upperCol, "_VALUE") ||
		strings.Contains(upperCol, "_TOTAL") || strings.Contains(upperName, "AMOUNT") {
		props["precision"] = 2
		props["symbol"] = "$"
		return "currency_input", props
	}

	// Percentage/Rate patterns
	if strings.Contains(upperCol, "_PCT") || strings.Contains(upperCol, "_PERCENT") ||
		strings.Contains(upperCol, "_RATE") || strings.Contains(upperCol, "_RATIO") {
		props["precision"] = 4
		props["suffix"] = "%"
		props["min"] = 0
		props["max"] = 100
		return "percentage_input", props
	}

	// Foreign Key / Lookup patterns
	if def.IsForeignKey || strings.HasSuffix(upperCol, "_ID") && !def.IsPrimaryKey {
		// Infer target BO from column name
		targetBO := strings.TrimSuffix(strings.ToLower(def.Column), "_id")
		props["target_bo"] = targetBO
		props["searchable"] = true
		return "lookup_dropdown", props
	}

	// Date/Time patterns
	if strings.Contains(lowerType, "date") || strings.Contains(lowerType, "timestamp") ||
		strings.Contains(lowerType, "time") {
		if strings.Contains(lowerType, "timestamp") || strings.Contains(upperCol, "_TS") {
			props["format"] = "datetime"
			props["showTime"] = true
			return "datetime_picker", props
		}
		props["format"] = "date"
		return "date_picker", props
	}

	// Boolean patterns
	if strings.Contains(lowerType, "bool") || strings.HasPrefix(upperCol, "IS_") ||
		strings.HasPrefix(upperCol, "HAS_") || strings.HasSuffix(upperCol, "_FLAG") {
		return "switch", props
	}

	// Numeric patterns (non-currency)
	if strings.Contains(lowerType, "int") || strings.Contains(lowerType, "numeric") ||
		strings.Contains(lowerType, "decimal") || strings.Contains(lowerType, "float") {
		if strings.Contains(upperCol, "QTY") || strings.Contains(upperCol, "QUANTITY") ||
			strings.Contains(upperCol, "COUNT") || strings.Contains(upperCol, "CNT") {
			props["precision"] = 0
			props["min"] = 0
			return "number_input", props
		}
		props["precision"] = 2
		return "number_input", props
	}

	// Text area for descriptions/notes
	if strings.Contains(upperCol, "DESC") || strings.Contains(upperCol, "DESCRIPTION") ||
		strings.Contains(upperCol, "NOTES") || strings.Contains(upperCol, "COMMENT") {
		props["rows"] = 3
		props["maxLength"] = 2000
		return "textarea", props
	}

	// Email pattern
	if strings.Contains(upperCol, "EMAIL") {
		props["inputType"] = "email"
		return "email_input", props
	}

	// Phone pattern
	if strings.Contains(upperCol, "PHONE") || strings.Contains(upperCol, "TEL") {
		props["inputType"] = "tel"
		return "phone_input", props
	}

	// Status/Enum patterns
	if strings.Contains(upperCol, "STATUS") || strings.Contains(upperCol, "STATE") ||
		strings.Contains(upperCol, "_TYPE") || strings.Contains(upperCol, "_CD") ||
		strings.Contains(upperCol, "_CODE") {
		return "select", props
	}

	// Default to text input
	return "text_input", props
}

// inferValidationRules generates validation rules based on column constraints and patterns
func (e *SemanticEnricher) inferValidationRules(def CubeDefinition) []ValidationRule {
	var rules []ValidationRule
	upperCol := strings.ToUpper(def.Column)

	// NOT NULL -> required
	if !def.IsNullable {
		rules = append(rules, ValidationRule{
			Type:     "condition",
			Field:    def.Column,
			Operator: "isNotEmpty",
			Severity: "error",
			Message:  def.Column + " is required",
		})
	}

	// Currency/Amount -> greater than 0
	if strings.Contains(upperCol, "_AMT") || strings.Contains(upperCol, "_AMOUNT") ||
		strings.Contains(upperCol, "_BALANCE") || strings.Contains(upperCol, "_PRICE") {
		rules = append(rules, ValidationRule{
			Type:     "condition",
			Field:    def.Column,
			Operator: "gt",
			Value:    0,
			Severity: "warning",
			Message:  def.Column + " should be greater than 0",
		})
	}

	// Percentage -> 0-100 range
	if strings.Contains(upperCol, "_PCT") || strings.Contains(upperCol, "_PERCENT") {
		rules = append(rules, ValidationRule{
			Type:     "condition",
			Field:    def.Column,
			Operator: "between",
			Value:    []float64{0, 100},
			Severity: "error",
			Message:  def.Column + " must be between 0 and 100",
		})
	}

	// Rate/Ratio -> 0-1 range
	if strings.Contains(upperCol, "_RATE") || strings.Contains(upperCol, "_RATIO") {
		rules = append(rules, ValidationRule{
			Type:     "condition",
			Field:    def.Column,
			Operator: "between",
			Value:    []float64{0, 1},
			Severity: "error",
			Message:  def.Column + " must be between 0 and 1",
		})
	}

	// Email format
	if strings.Contains(upperCol, "EMAIL") {
		rules = append(rules, ValidationRule{
			Type:     "condition",
			Field:    def.Column,
			Operator: "matchesPattern",
			Value:    "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
			Severity: "error",
			Message:  "Invalid email format",
		})
	}

	return rules
}

// inferDisplayHints generates display hints for UI rendering
func (e *SemanticEnricher) inferDisplayHints(def CubeDefinition) map[string]interface{} {
	hints := make(map[string]interface{})
	upperCol := strings.ToUpper(def.Column)
	upperTable := strings.ToUpper(def.Table)

	// Section inference based on column patterns
	switch {
	case strings.Contains(upperCol, "CREATED") || strings.Contains(upperCol, "UPDATED") ||
		strings.Contains(upperCol, "MODIFIED") || strings.Contains(upperCol, "_BY"):
		hints["section"] = "Audit"
		hints["order"] = 99 // Last
	case strings.Contains(upperCol, "NAME") || strings.Contains(upperCol, "TITLE"):
		hints["section"] = "General"
		hints["order"] = 1
	case strings.Contains(upperCol, "DESC") || strings.Contains(upperCol, "NOTES"):
		hints["section"] = "General"
		hints["order"] = 10
	case strings.Contains(upperCol, "_AMT") || strings.Contains(upperCol, "_BALANCE") ||
		strings.Contains(upperCol, "_VALUE"):
		hints["section"] = "Financial"
		hints["order"] = 5
	case def.IsPrimaryKey:
		hints["section"] = "System"
		hints["hidden"] = true
	}

	// Badge inference
	if strings.Contains(upperCol, "RISK") {
		hints["badge"] = "risk"
	} else if strings.Contains(upperCol, "STATUS") {
		hints["badge"] = "status"
	}

	// Table-based section override
	if strings.Contains(upperTable, "PORTFOLIO") {
		if hints["section"] == nil {
			hints["section"] = "Portfolio"
		}
	} else if strings.Contains(upperTable, "ACCOUNT") {
		if hints["section"] == nil {
			hints["section"] = "Account"
		}
	}

	return hints
}

// inferWealthDomain determines the wealth management domain for the column
func (e *SemanticEnricher) inferWealthDomain(def CubeDefinition) string {
	upperCol := strings.ToUpper(def.Column)
	upperTable := strings.ToUpper(def.Table)
	combined := upperTable + "_" + upperCol

	switch {
	case strings.Contains(combined, "PORTFOLIO") || strings.Contains(combined, "HOLDING") ||
		strings.Contains(combined, "POSITION") || strings.Contains(combined, "ASSET"):
		return "portfolio"
	case strings.Contains(combined, "PROPOSAL") || strings.Contains(combined, "TRADE") ||
		strings.Contains(combined, "ORDER"):
		return "proposal"
	case strings.Contains(combined, "MODEL") || strings.Contains(combined, "STRATEGY") ||
		strings.Contains(combined, "ALLOCATION"):
		return "model"
	case strings.Contains(combined, "SECURITY") || strings.Contains(combined, "TICKER") ||
		strings.Contains(combined, "CUSIP") || strings.Contains(combined, "ISIN"):
		return "security"
	case strings.Contains(combined, "ACCOUNT") || strings.Contains(combined, "CLIENT"):
		return "account"
	case strings.Contains(combined, "TAX") || strings.Contains(combined, "LOT"):
		return "tax"
	default:
		return "general"
	}
}

// inferBOSubtypeHint suggests a BO subtype based on patterns
func (e *SemanticEnricher) inferBOSubtypeHint(def CubeDefinition) string {
	upperTable := strings.ToUpper(def.Table)

	switch {
	case strings.Contains(upperTable, "TAXABLE"):
		return "taxable"
	case strings.Contains(upperTable, "RETIREMENT") || strings.Contains(upperTable, "IRA"):
		return "retirement"
	case strings.Contains(upperTable, "TRUST"):
		return "trust"
	default:
		return ""
	}
}

// inferConstraintTemplate generates constraint templates for wealth-specific rules
func (e *SemanticEnricher) inferConstraintTemplate(def CubeDefinition) map[string]interface{} {
	upperCol := strings.ToUpper(def.Column)
	template := make(map[string]interface{})

	// Allocation percentage constraints
	if strings.Contains(upperCol, "ALLOCATION") || strings.Contains(upperCol, "_PCT") {
		template["maxpct"] = 1.0
		template["minpct"] = 0.0
		template["sumTo100"] = true
	}

	// Concentration limits
	if strings.Contains(upperCol, "CONCENTRATION") {
		template["maxpct"] = 0.10 // 10% default concentration limit
	}

	// Position limits
	if strings.Contains(upperCol, "POSITION") && strings.Contains(upperCol, "SIZE") {
		template["maxPositionSize"] = 1000000
	}

	return template
}

// updateNodeProperties updates the catalog_node properties with enrichment
func (e *SemanticEnricher) updateNodeProperties(ctx context.Context, nodeID uuid.UUID, enrichment *SemanticEnrichment) error {
	logger := logging.GetLogger().Sugar()

	propsJSON, err := json.Marshal(enrichment)
	if err != nil {
		return err
	}

	query := `
		UPDATE catalog_node 
		SET properties = properties || $1::jsonb,
		    updated_at = NOW()
		WHERE id = $2
	`

	result, err := e.db.ExecContext(ctx, query, propsJSON, nodeID)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	logger.Infof("Updated %d node(s) with enrichment for nodeID=%s", rows, nodeID)

	return nil
}

// EnrichFromColumnData creates enrichment directly from column metadata (for wizard use)
func (e *SemanticEnricher) EnrichFromColumnData(columnName, tableName, dataType string, isNullable, isForeignKey, isPrimaryKey bool) *SemanticEnrichment {
	cubeDef := CubeDefinition{
		Name:         columnName,
		Table:        tableName,
		Column:       columnName,
		DataType:     dataType,
		IsNullable:   isNullable,
		IsForeignKey: isForeignKey,
		IsPrimaryKey: isPrimaryKey,
	}

	// Skip DB update - just return enrichment
	enrichment := &SemanticEnrichment{
		UIComponent: "text_input",
		UIProps:     make(map[string]interface{}),
	}

	enrichment.UIComponent, enrichment.UIProps = e.inferUIComponent(cubeDef)
	enrichment.ValidationRules = e.inferValidationRules(cubeDef)
	enrichment.DisplayHints = e.inferDisplayHints(cubeDef)
	enrichment.WealthDomain = e.inferWealthDomain(cubeDef)
	enrichment.BOSubtypeHint = e.inferBOSubtypeHint(cubeDef)
	enrichment.ConstraintTemplate = e.inferConstraintTemplate(cubeDef)

	return enrichment
}

// AIEnrichmentRequest represents a request for AI-powered enrichment
type AIEnrichmentRequest struct {
	NodeKey      string         `json:"node_key"`
	CubeDef      CubeDefinition `json:"cube_def"`
	SemanticType string         `json:"semantic_type,omitempty"`
}

// AIEnrichmentResponse represents the AI suggestion response
type AIEnrichmentResponse struct {
	UIComponent        string                 `json:"ui_component,omitempty"`
	UIProps            map[string]interface{} `json:"ui_props,omitempty"`
	ValidationRules    []ValidationRule       `json:"validation_rules,omitempty"`
	DisplayHints       map[string]interface{} `json:"display_hints,omitempty"`
	WealthDomain       string                 `json:"wealth_domain,omitempty"`
	BOSubtypeHint      string                 `json:"bo_subtype_hint,omitempty"`
	ConstraintTemplate map[string]interface{} `json:"constraint_template,omitempty"`
	Reasoning          string                 `json:"reasoning,omitempty"`
}

// EnrichWithAI uses LLM to enhance enrichment suggestions beyond pattern-based inference
func (e *SemanticEnricher) EnrichWithAI(ctx context.Context, cubeDef CubeDefinition, baseEnrichment *SemanticEnrichment) (*SemanticEnrichment, error) {
	logger := logging.GetLogger().Sugar()

	if e.llmProvider == nil {
		logger.Warn("LLM provider not available, returning base enrichment")
		return baseEnrichment, nil
	}

	// Type assert to LLM provider interface
	llmProvider, ok := e.llmProvider.(interface {
		GenerateContent(context.Context, string) (string, error)
	})
	if !ok {
		logger.Warn("Invalid LLM provider type, returning base enrichment")
		return baseEnrichment, nil
	}

	prompt := e.buildAIEnrichmentPrompt(cubeDef, baseEnrichment)

	result, err := llmProvider.GenerateContent(ctx, prompt)
	if err != nil {
		logger.Warnf("AI enrichment failed: %v, returning base enrichment", err)
		return baseEnrichment, nil
	}

	// Parse AI response
	aiResponse, err := e.parseAIEnrichmentResponse(result)
	if err != nil {
		logger.Warnf("Failed to parse AI response: %v, returning base enrichment", err)
		return baseEnrichment, nil
	}

	// Merge AI suggestions with base enrichment (AI takes precedence)
	merged := e.mergeEnrichments(baseEnrichment, aiResponse)
	logger.Infof("AI enrichment successful for %s.%s", cubeDef.Table, cubeDef.Column)

	return merged, nil
}

// buildAIEnrichmentPrompt creates the prompt for AI enrichment
func (e *SemanticEnricher) buildAIEnrichmentPrompt(cubeDef CubeDefinition, baseEnrichment *SemanticEnrichment) string {
	baseJSON, _ := json.Marshal(baseEnrichment)

	return `You are a wealth management semantic layer expert. Given the following database column metadata, suggest enrichment properties for UI rendering, validation, and business logic.

Column Metadata:
- Column Name: ` + cubeDef.Column + `
- Table Name: ` + cubeDef.Table + `
- Data Type: ` + cubeDef.DataType + `
- Semantic Type: ` + cubeDef.SemanticType + `
- Is Nullable: ` + boolToStr(cubeDef.IsNullable) + `
- Is Foreign Key: ` + boolToStr(cubeDef.IsForeignKey) + `
- Is Primary Key: ` + boolToStr(cubeDef.IsPrimaryKey) + `

Current Pattern-Based Enrichment:
` + string(baseJSON) + `

Please enhance or correct the enrichment with wealth management domain knowledge. Consider:
1. UI Component: Best component for this data (currency_input, percentage_input, lookup_dropdown, date_picker, etc.)
2. UI Props: Component-specific settings (precision, symbol, min/max, format)
3. Validation Rules: Business rules with operators (isNotEmpty, gt, lt, between, matchesPattern)
4. Display Hints: Section grouping, display order, badges (risk, status)
5. Wealth Domain: portfolio, proposal, model, security, account, tax, or general
6. BO Subtype Hint: taxable, retirement, trust, custodial, etc.
7. Constraint Template: Wealth-specific constraints (maxpct, concentration limits)

Return ONLY valid JSON matching this structure (no markdown):
{
  "ui_component": "currency_input",
  "ui_props": {"precision": 2, "symbol": "$"},
  "validation_rules": [{"type": "condition", "field": "column", "operator": "gt", "value": 0, "severity": "error"}],
  "display_hints": {"section": "Financial", "order": 5},
  "wealth_domain": "portfolio",
  "bo_subtype_hint": "",
  "constraint_template": {},
  "reasoning": "Brief explanation of suggestions"
}`
}

// parseAIEnrichmentResponse parses the JSON response from AI
func (e *SemanticEnricher) parseAIEnrichmentResponse(result string) (*AIEnrichmentResponse, error) {
	// Clean up potential markdown formatting
	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	result = strings.TrimSpace(result)

	var response AIEnrichmentResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// mergeEnrichments merges AI suggestions with base enrichment
func (e *SemanticEnricher) mergeEnrichments(base *SemanticEnrichment, ai *AIEnrichmentResponse) *SemanticEnrichment {
	merged := &SemanticEnrichment{
		UIComponent:        base.UIComponent,
		UIProps:            base.UIProps,
		ValidationRules:    base.ValidationRules,
		DisplayHints:       base.DisplayHints,
		WealthDomain:       base.WealthDomain,
		BOSubtypeHint:      base.BOSubtypeHint,
		ConstraintTemplate: base.ConstraintTemplate,
	}

	// Override with AI suggestions if non-empty
	if ai.UIComponent != "" {
		merged.UIComponent = ai.UIComponent
	}
	if len(ai.UIProps) > 0 {
		if merged.UIProps == nil {
			merged.UIProps = make(map[string]interface{})
		}
		for k, v := range ai.UIProps {
			merged.UIProps[k] = v
		}
	}
	if len(ai.ValidationRules) > 0 {
		merged.ValidationRules = ai.ValidationRules
	}
	if len(ai.DisplayHints) > 0 {
		if merged.DisplayHints == nil {
			merged.DisplayHints = make(map[string]interface{})
		}
		for k, v := range ai.DisplayHints {
			merged.DisplayHints[k] = v
		}
	}
	if ai.WealthDomain != "" {
		merged.WealthDomain = ai.WealthDomain
	}
	if ai.BOSubtypeHint != "" {
		merged.BOSubtypeHint = ai.BOSubtypeHint
	}
	if len(ai.ConstraintTemplate) > 0 {
		if merged.ConstraintTemplate == nil {
			merged.ConstraintTemplate = make(map[string]interface{})
		}
		for k, v := range ai.ConstraintTemplate {
			merged.ConstraintTemplate[k] = v
		}
	}

	return merged
}

// boolToStr converts bool to string for prompt building
func boolToStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// EnrichWithAIFromColumnData convenience method for wizard use
func (e *SemanticEnricher) EnrichWithAIFromColumnData(ctx context.Context, columnName, tableName, dataType, semanticType string, isNullable, isForeignKey, isPrimaryKey bool) (*SemanticEnrichment, error) {
	cubeDef := CubeDefinition{
		Name:         columnName,
		Table:        tableName,
		Column:       columnName,
		DataType:     dataType,
		SemanticType: semanticType,
		IsNullable:   isNullable,
		IsForeignKey: isForeignKey,
		IsPrimaryKey: isPrimaryKey,
	}

	// First get pattern-based enrichment
	baseEnrichment := e.EnrichFromColumnData(columnName, tableName, dataType, isNullable, isForeignKey, isPrimaryKey)

	// Then enhance with AI
	return e.EnrichWithAI(ctx, cubeDef, baseEnrichment)
}
