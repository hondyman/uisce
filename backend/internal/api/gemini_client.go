package api

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

// GeminiClient wraps Google Gemini API for LLM operations
type GeminiClient struct {
	client *genai.Client
	model  string
}

// NewGeminiClient creates a new Gemini client
func NewGeminiClient(apiKey string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create gemini client: %w", err)
	}

	return &GeminiClient{
		client: client,
		// "gemini-pro" (and "gemini-1.5-flash") are retired on the current
		// Gemini API version (v1beta) - every caller (NL-to-SQL
		// planner/executor, AI page generation) silently fell back to its
		// deterministic path with a 404 logged, since none of them treat a
		// Gemini failure as fatal. Confirmed available via this project's
		// key's own ListModels response.
		model: "gemini-2.5-flash",
	}, nil
}

// GenerateSemanticQuery converts natural language to a SemanticQuery using Gemini
func (gc *GeminiClient) GenerateSemanticQuery(ctx context.Context, bundle *SemanticBundle, userPrompt string, mode string, region string) (*SemanticQuery, error) {
	if gc.client == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}

	// Build system prompt with bundle metadata and region guidance
	systemPrompt := buildPlannerSystemPrompt(bundle, mode)
	fullPrompt := systemPrompt + "\n\nRequest Region: " + region + "\n\nUser Query:\n" + userPrompt + "\n\nNOTE: The returned JSON MUST include a top-level \"region\" field equal to the Request Region."

	// Create model and set generation config
	model := gc.client.GenerativeModel(gc.model)
	model.SetTemperature(0.0) // Deterministic for reproducibility
	model.SetMaxOutputTokens(2000)

	// Call the API
	resp, err := model.GenerateContent(ctx, genai.Text(fullPrompt))
	if err != nil {
		return nil, fmt.Errorf("gemini API call failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response from gemini")
	}

	// Extract text from response
	var responseText string
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				responseText = string(text)
				break
			}
		}
		if responseText != "" {
			break
		}
	}

	if responseText == "" {
		return nil, fmt.Errorf("empty response from gemini")
	}

	// Extract JSON from markdown code blocks
	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		return nil, fmt.Errorf("failed to extract JSON from response: %s", responseText)
	}

	// Parse JSON into SemanticQuery
	var sq SemanticQuery
	if err := json.Unmarshal([]byte(jsonStr), &sq); err != nil {
		return nil, fmt.Errorf("failed to unmarshal semantic query: %w", err)
	}

	return &sq, nil
}

// GenerateSQL converts a SemanticQuery to SQL using Gemini
func (gc *GeminiClient) GenerateSQL(ctx context.Context, bundle *SemanticBundle, q *SemanticQuery) (string, error) {
	if gc.client == nil {
		return "", fmt.Errorf("gemini client not initialized")
	}

	// Build system prompt for SQL generation
	systemPrompt := buildExecutorSystemPrompt(bundle)

	// Convert query to JSON for passing to LLM
	queryJSON, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal query: %w", err)
	}

	fullPrompt := systemPrompt + "\n\nSemantic Query:\n" + string(queryJSON)

	// Create model and set generation config
	model := gc.client.GenerativeModel(gc.model)
	model.SetTemperature(0.0) // Deterministic for reproducibility
	model.SetMaxOutputTokens(2000)

	// Call the API
	resp, err := model.GenerateContent(ctx, genai.Text(fullPrompt))
	if err != nil {
		return "", fmt.Errorf("gemini API call failed: %w", err)
	}

	if len(resp.Candidates) == 0 {
		return "", fmt.Errorf("no response from gemini")
	}

	// Extract text from response
	var responseText string
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				responseText = string(text)
				break
			}
		}
		if responseText != "" {
			break
		}
	}

	if responseText == "" {
		return "", fmt.Errorf("empty response from gemini")
	}

	// Extract SQL from markdown code blocks
	sql := extractSQL(responseText)
	if sql == "" {
		return "", fmt.Errorf("failed to extract SQL from response: %s", responseText)
	}

	return sql, nil
}

// extractJSON extracts JSON from markdown code blocks
func extractJSON(text string) string {
	// Try to extract from ```json ... ``` blocks
	jsonRe := regexp.MustCompile("(?s)```json\\s*(\\{[^`]*?\\})\\s*```")
	matches := jsonRe.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to extract from ``` ... ``` blocks (generic)
	genericRe := regexp.MustCompile("(?s)```\\s*(\\{[^`]*?\\})\\s*```")
	matches = genericRe.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}

	// Try to find JSON directly in the text
	text = strings.TrimSpace(text)
	if strings.HasPrefix(text, "{") && strings.HasSuffix(text, "}") {
		return text
	}

	return ""
}

// extractSQL extracts SQL from markdown code blocks
func extractSQL(text string) string {
	// Try to extract from ```sql ... ``` blocks
	sqlRe := regexp.MustCompile("(?s)```sql\\s*(SELECT[^`]*)```")
	matches := sqlRe.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try to extract from ``` ... ``` blocks (generic)
	genericRe := regexp.MustCompile("(?s)```\\s*(SELECT[^`]*)```")
	matches = genericRe.FindStringSubmatch(text)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Try to find SQL directly starting with SELECT
	lines := strings.Split(text, "\n")
	var sqlLines []string
	inSQL := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "SELECT") {
			inSQL = true
		}
		if inSQL {
			sqlLines = append(sqlLines, line)
			if strings.Contains(trimmed, ";") {
				break
			}
		}
	}

	if len(sqlLines) > 0 {
		return strings.TrimSpace(strings.Join(sqlLines, "\n"))
	}

	return ""
}

// PageGenerationField is one BO field made available to the model as
// grounding for page generation - just enough for it to judge the field mix
// (how many measures vs dimensions) without letting it invent field names
// the page could bind to, since actual data binding is resolved separately
// at render time (PageComponentRenderer.tsx fetches live BO terms), not
// from anything the model outputs here.
type PageGenerationField struct {
	Key         string
	DisplayName string
	DataType    string
	Role        string // DIMENSION, MEASURE, or CALCULATED
}

// RelatedBOSummary is one Business Object related to the page's primary BO
// (from the already-fixed catalog_edge relationship graph -
// GetBusinessObjectRelationships), offered to the model as a candidate to
// pull onto the page - e.g. an "Order" page might pull in "Order
// Allocation" or "Execution" sections, not just its own fields.
type RelatedBOSummary struct {
	BOKey            string
	DisplayName      string
	RelationshipType string
	Cardinality      string
	Fields           []PageGenerationField
}

// PageGenerationSection is one widget the model wants placed on the
// generated page, and which Business Object it should be bound to.
// BOKey == "" means the page's own primary BO; any other value must match
// one of the RelatedBOSummary.BOKey values offered in the prompt - the
// model can't invent a BO to bind to, only choose among ones actually
// related to the primary. Field binding within a BO is still auto-resolved
// at render time (PageComponentRenderer.tsx), not chosen here.
type PageGenerationSection struct {
	BOKey string `json:"boKey"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

// PageGenerationSpec is the JSON shape asked of the model.
type PageGenerationSpec struct {
	Title    string `json:"title"`
	PageKind string `json:"pageKind"`
	FilterBar []PageGenerationSection `json:"filterBar,omitempty"`
	// LayoutTemplate names one of allowedPageGenerationTemplates - the
	// section-based body layout (frontend/src/pages/page-studio/
	// layoutTemplates.ts) the generated sections are distributed into, in
	// order, one per section. Restricted to plain section templates (no
	// side panels) for this first pass - see PageStudioListPage.tsx's
	// "Generate with AI" dialog comment.
	LayoutTemplate string                  `json:"layoutTemplate"`
	Sections       []PageGenerationSection `json:"sections"`
}

// allowedPageGenerationWidgetTypes are the only component types the page
// designer's palette actually renders as data-bound widgets (see
// COMPONENT_TO_WIDGET_TYPE and the Table/Form special cases in
// PageComponentRenderer.tsx). Anything else the model returns is dropped
// rather than trusted, since an unknown component type renders as an inert
// "Component Preview" placeholder box.
var allowedPageGenerationWidgetTypes = map[string]bool{
	"KPIGroup":  true,
	"LineChart": true,
	"Table":     true,
	"Slicer":    true,
	"Form":      true,
}

// allowedPageGenerationTemplates mirrors the plain section-based ids in
// layoutTemplates.ts (single-column, two-column, three-column,
// dashboard-grid). master-detail and two-column-side-panel are deliberately
// excluded - those bundle a side Panel as part of the template shape, and
// AI-driven layout is scoped to body sections only for this first pass.
var allowedPageGenerationTemplates = map[string]bool{
	"single-column":  true,
	"two-column":     true,
	"three-column":   true,
	"dashboard-grid": true,
	"master-detail":  true,
}

var allowedPageKinds = map[string]bool{
	"list":          true,
	"detail":        true,
	"master-detail": true,
	"dashboard":     true,
}

// GeneratePageSpec asks Gemini to pick a small section mix and page title
// for a Business Object, grounded in that BO's real fields AND its real
// related Business Objects (relatedBOs, from the catalog relationship
// graph) so the model can decide e.g. "this is an Order page, pull in
// Order Allocation and Execution as their own sections" instead of only
// ever describing the primary BO's own fields. It does not choose field
// bindings itself - see PageGenerationSection - so a wrong or missing field
// name in the model's reasoning can't corrupt the generated page, and it
// can't bind to a BO that isn't actually related (validated against
// relatedBOs below).
func (gc *GeminiClient) GeneratePageSpec(ctx context.Context, boName, boKey, description, pageKind string, fields []PageGenerationField, relatedBOs []RelatedBOSummary) (*PageGenerationSpec, error) {
	if gc.client == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}

	prompt := buildPageGenerationPrompt(boName, boKey, description, pageKind, fields, relatedBOs)

	model := gc.client.GenerativeModel(gc.model)
	model.SetTemperature(0.2)
	// 500, then 1200, both still truncated real responses mid-JSON -
	// gemini-2.5-flash spends part of MaxOutputTokens on internal
	// "thinking" tokens before it ever writes the visible JSON, so the
	// visible-text budget is smaller than the number itself suggests.
	model.SetMaxOutputTokens(4000)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("gemini API call failed: %w", err)
	}
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no response from gemini")
	}

	var responseText string
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if text, ok := part.(genai.Text); ok {
				responseText = string(text)
				break
			}
		}
		if responseText != "" {
			break
		}
	}
	if responseText == "" {
		return nil, fmt.Errorf("empty response from gemini")
	}

	jsonStr := extractJSON(responseText)
	if jsonStr == "" {
		return nil, fmt.Errorf("failed to extract JSON from response: %s", responseText)
	}

	var spec PageGenerationSpec
	if err := json.Unmarshal([]byte(jsonStr), &spec); err != nil {
		return nil, fmt.Errorf("failed to unmarshal page spec: %w", err)
	}

	// Every boKey the model is allowed to bind to: "" (primary) plus each
	// offered related BO. Anything else - a hallucinated or misspelled key -
	// gets remapped to the primary rather than trusted, since an unknown
	// boKey would leave a section with no resolvable data source.
	validBOKeys := map[string]bool{"": true}
	for _, r := range relatedBOs {
		validBOKeys[r.BOKey] = true
	}

	// Filter to section types the designer can actually render, and cap the
	// count - a runaway or malformed response shouldn't be able to hand the
	// caller an unbounded or unrenderable section list.
	filtered := make([]PageGenerationSection, 0, len(spec.Sections))
	for _, s := range spec.Sections {
		if !allowedPageGenerationWidgetTypes[s.Type] {
			continue
		}
		if !validBOKeys[s.BOKey] {
			s.BOKey = ""
		}
		filtered = append(filtered, s)
		if len(filtered) == 6 {
			break
		}
	}
	spec.Sections = filtered
	if len(spec.Sections) == 0 {
		return nil, fmt.Errorf("gemini returned no usable sections")
	}
	if spec.Title == "" {
		spec.Title = boName
	}
	if !allowedPageKinds[spec.PageKind] {
		if allowedPageKinds[pageKind] {
			spec.PageKind = pageKind
		} else {
			spec.PageKind = "dashboard"
		}
	}
	if !allowedPageGenerationTemplates[spec.LayoutTemplate] {
		spec.LayoutTemplate = "single-column"
	}
	filteredBar := make([]PageGenerationSection, 0, len(spec.FilterBar))
	for _, s := range spec.FilterBar {
		if s.Type != "Slicer" {
			continue
		}
		if !validBOKeys[s.BOKey] {
			s.BOKey = ""
		}
		filteredBar = append(filteredBar, s)
		if len(filteredBar) == 4 {
			break
		}
	}
	spec.FilterBar = filteredBar

	return &spec, nil
}

func buildPageGenerationPrompt(boName, boKey, description, pageKind string, fields []PageGenerationField, relatedBOs []RelatedBOSummary) string {
	if pageKind == "" {
		pageKind = "dashboard"
	}
	prompt := "You are designing one page for a governed Business Object application (Salesforce Lightning / PeopleSoft analog), not a marketing site.\n\n"
	prompt += "You MUST output only valid JSON in a markdown code block: ```json {...}```\n"
	prompt += "The JSON shape is exactly: {\"title\": string, \"pageKind\": string, \"layoutTemplate\": string, \"filterBar\": [{\"boKey\": string, \"type\": \"Slicer\", \"title\": string}], \"sections\": [{\"boKey\": string, \"type\": string, \"title\": string}]}\n"
	prompt += "\"pageKind\" MUST be one of: \"list\", \"detail\", \"master-detail\", \"dashboard\".\n"
	prompt += "\"layoutTemplate\" MUST be one of: \"single-column\", \"two-column\", \"three-column\", \"dashboard-grid\", \"master-detail\".\n"
	prompt += "\"type\" MUST be one of: \"KPIGroup\", \"LineChart\", \"Table\", \"Slicer\", \"Form\".\n"
	prompt += fmt.Sprintf("\"boKey\" MUST be either \"\" (the primary Business Object %q) or one of the related keys listed below — never invent a Business Object or field name.\n", boKey)
	prompt += "Cardinality rules: a one-valued object uses Form (or Table on a list page); a 1:N related object uses Table. KPIGroup/LineChart only if that object has MEASURE or CALCULATED fields.\n"
	prompt += "list: one primary Table, optional Slicers in filterBar, no Form. detail: one primary Form plus 0-2 related Tables. master-detail: primary Table then Form (and optional child Tables), layoutTemplate master-detail. dashboard: KPI/Chart/Table mix.\n"
	prompt += "Pick 1 to 6 body sections. Prefer 0-2 related Business Objects (children like allocations/executions, not every inbound FK). filterBar is optional and Slicer-only.\n"
	prompt += "Do not emit Save/Delete/Create widgets. Formatting-only; CRUD lives on the Business Object.\n"
	prompt += "Never include any text before or after the JSON block.\n\n"
	prompt += fmt.Sprintf("Requested pageKind: %s\n", pageKind)
	prompt += fmt.Sprintf("Primary Business Object: %s (key: %s)\n", boName, boKey)
	if description != "" {
		prompt += fmt.Sprintf("User's request: %s\n", description)
	}
	prompt += "\nPrimary Business Object's available fields:\n"
	for _, f := range fields {
		prompt += fmt.Sprintf("  - %s (%s, %s)\n", f.DisplayName, f.DataType, f.Role)
	}
	if len(relatedBOs) > 0 {
		prompt += "\nRelated Business Objects you may optionally pull in as their own sections:\n"
		for _, r := range relatedBOs {
			prompt += fmt.Sprintf("  Business Object %q (key: %s) - relationship: %s, cardinality: %s\n", r.DisplayName, r.BOKey, r.RelationshipType, r.Cardinality)
			for _, f := range r.Fields {
				prompt += fmt.Sprintf("    - %s (%s, %s)\n", f.DisplayName, f.DataType, f.Role)
			}
		}
	}
	return prompt
}

// Close closes the Gemini client connection
func (gc *GeminiClient) Close() error {
	if gc.client != nil {
		return gc.client.Close()
	}
	return nil
}

// buildPlannerSystemPrompt creates the system prompt for the planner LLM
func buildPlannerSystemPrompt(bundle *SemanticBundle, mode string) string {
	// Use the golden prompt from the existing codebase
	// For now, provide a reasonable default
	prompt := "You are an expert SQL query planner. Your job is to convert natural language questions into structured semantic queries.\n\n"
	prompt += "CRITICAL RULES FOR SEMANTIC QUERY GENERATION:\n"
	prompt += "1. You MUST output only valid JSON in a markdown code block: ```json {...}```\n"
	prompt += "2. Never include any text before or after the JSON block\n"
	prompt += "3. All field references MUST use their semantic (display) names from the bundle, NOT physical column names\n"
	prompt += "4. Always use the exact field names as they appear in the bundle metadata\n"
	prompt += "5. Include ALL fields mentioned in the user's question\n"
	prompt += "6. Use \"EXPLORATORY\" mode only for inferred fields\n"
	prompt += "7. REGION: You MUST include a top-level string field `region` in the output JSON and it MUST equal the Request Region provided in the prompt. If the requested region is not available for the tenant, return an error object instead.\n"
	if mode == "strict" {
		prompt += "7. STRICT MODE: Do not infer any fields. Only include fields explicitly mentioned in the query.\n"
	}

	prompt += "\nSEMANTIC BUNDLE METADATA:\n"
	prompt += fmt.Sprintf("Business Object: %s\n", bundle.BusinessObjectName)
	prompt += fmt.Sprintf("Version: %s\n", bundle.Version)
	prompt += fmt.Sprintf("Driving Table: %s\n\n", bundle.DrivingTable)

	// Add fields to the prompt
	prompt += "Available Fields:\n"
	for _, field := range bundle.Fields {
		prompt += fmt.Sprintf("  - %s (display: %s): %s [%s.%s]\n", field.Name, field.DisplayName, field.SemanticTerm, field.Physical.Table, field.Physical.Column)
	}

	return prompt
}

// buildExecutorSystemPrompt creates the system prompt for the executor LLM
func buildExecutorSystemPrompt(bundle *SemanticBundle) string {
	prompt := "You are an expert SQL generator. Your job is to convert semantic queries into correct, parameterized SQL.\n\n"
	prompt += "CRITICAL RULES FOR SQL GENERATION:\n"
	prompt += "1. You MUST output only valid SQL in a markdown code block: ```sql SELECT ...```\n"
	prompt += "2. Never include any text before or after the SQL block\n"
	prompt += "3. Use physical column names and table names from the bundle metadata\n"
	prompt += "4. Use proper JOINs to connect tables from different entities as specified in the relationships\n"
	prompt += "5. Apply LIMIT and ORDER BY clauses as specified\n"
	prompt += "6. Preserve all WHERE clause filters from the semantic query\n\n"
	prompt += "SEMANTIC BUNDLE METADATA:\n"
	prompt += fmt.Sprintf("Business Object: %s\n", bundle.BusinessObjectName)
	prompt += fmt.Sprintf("Version: %s\n", bundle.Version)
	prompt += fmt.Sprintf("Driving Table: %s\n\n", bundle.DrivingTable)

	// Add physical mapping info
	prompt += "Available Fields (Semantic -> Physical):\n"
	for _, field := range bundle.Fields {
		prompt += fmt.Sprintf("  - %s -> %s.%s (%s)\n", field.Name, field.Physical.Table, field.Physical.Column, field.SemanticTerm)
	}

	if len(bundle.Relationships) > 0 {
		prompt += "\nRelationships (for JOINs):\n"
		for _, rel := range bundle.Relationships {
			prompt += fmt.Sprintf("  - %s (source: %s.%s -> target: %s.%s)\n", rel.JoinType, bundle.DrivingTable, rel.SourceColumn, rel.TargetTable, rel.TargetColumn)
		}
	}

	return prompt
}
