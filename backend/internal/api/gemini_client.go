package api

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hondyman/uisce/backend/pkg/llm"
)

// GeminiClient wraps the centralized pkg/llm Gemini provider for the
// semantic-query planner/executor. All Gemini calls in the backend go
// through pkg/llm.GeminiProvider so the model and API key live in one place;
// this type just supplies the planner/executor-specific prompts and
// deterministic generation settings.
type GeminiClient struct {
	provider *llm.GeminiProvider
}

// NewGeminiClient creates a new Gemini client. modelName is optional — pass
// "" to use pkg/llm's centralized default model.
func NewGeminiClient(apiKey string, modelName ...string) (*GeminiClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY is required")
	}
	model := ""
	if len(modelName) > 0 {
		model = modelName[0]
	}
	provider := llm.NewGeminiProvider(apiKey, model).WithGenerationConfig(0.0, 0.95, 40, 2000)
	return &GeminiClient{provider: provider}, nil
}

// GenerateSemanticQuery converts natural language to a SemanticQuery using Gemini
func (gc *GeminiClient) GenerateSemanticQuery(ctx context.Context, bundle *SemanticBundle, userPrompt string, mode string, region string) (*SemanticQuery, error) {
	if gc.provider == nil {
		return nil, fmt.Errorf("gemini client not initialized")
	}

	// Build system prompt with bundle metadata and region guidance
	systemPrompt := buildPlannerSystemPrompt(bundle, mode)
	fullPrompt := systemPrompt + "\n\nRequest Region: " + region + "\n\nUser Query:\n" + userPrompt + "\n\nNOTE: The returned JSON MUST include a top-level \"region\" field equal to the Request Region."

	responseText, err := gc.provider.GenerateResponse(ctx, fullPrompt)
	if err != nil {
		return nil, fmt.Errorf("gemini API call failed: %w", err)
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
	if gc.provider == nil {
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

	responseText, err := gc.provider.GenerateResponse(ctx, fullPrompt)
	if err != nil {
		return "", fmt.Errorf("gemini API call failed: %w", err)
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

// Close is a no-op: pkg/llm.GeminiProvider is a plain HTTP client with
// nothing to tear down. Kept for callers that defer gc.Close().
func (gc *GeminiClient) Close() error {
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
