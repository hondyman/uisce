package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hondyman/semlayer/backend/pkg/llm"
	"github.com/hondyman/semlayer/backend/pkg/migration"
	"github.com/jmoiron/sqlx"
	"go.temporal.io/sdk/activity"
)

// ConfigGenerationActivities handles RAG-powered Titan config generation
type ConfigGenerationActivities struct {
	DB               *sqlx.DB
	ConfigService    *llm.LLMConfigService
	EmbeddingService *migration.EmbeddingService
}

func NewConfigGenerationActivities(db *sqlx.DB, cfgSvc *llm.LLMConfigService) *ConfigGenerationActivities {
	return &ConfigGenerationActivities{
		DB:               db,
		ConfigService:    cfgSvc,
		EmbeddingService: migration.NewEmbeddingService(db.DB, cfgSvc),
	}
}

// GeneratedConfig represents the output of config generation
type GeneratedConfig struct {
	DAG         json.RawMessage `json:"dag"`
	Rego        string          `json:"rego,omitempty"`
	Explanation string          `json:"explanation"`
	Context     []string        `json:"contextUsed"` // IDs of knowledge base items used
}

// ActivityGenerateConfig takes business intent and generates Titan configuration
func (a *ConfigGenerationActivities) ActivityGenerateConfig(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting config generation activity")

	// 1. Extract business intent from state
	intentJSON, _ := state["extracted_intent"].(string)
	if intentJSON == "" {
		return nil, fmt.Errorf("no extracted_intent in state")
	}

	var intent BusinessRuleIntent
	if err := json.Unmarshal([]byte(intentJSON), &intent); err != nil {
		return nil, fmt.Errorf("failed to parse intent: %w", err)
	}

	// 2. Retrieve relevant context from Knowledge Base
	ragContext, err := a.retrieveContext(ctx, intent)
	if err != nil {
		logger.Warn("RAG retrieval failed, proceeding without context", "error", err)
	}

	// 3. Generate configuration using LLM
	cfg, err := a.ConfigService.Get()
	if err != nil {
		return nil, fmt.Errorf("LLM config unavailable: %w", err)
	}
	provider := llm.NewGeminiProvider(cfg.APIKey, cfg.Model)

	generated, err := a.generateWithRAG(ctx, provider, intent, ragContext)
	if err != nil {
		return nil, fmt.Errorf("generation failed: %w", err)
	}

	// 4. Return results
	dagJSON, _ := json.Marshal(generated.DAG)
	contextJSON, _ := json.Marshal(generated.Context)

	return map[string]interface{}{
		"generated_dag":  string(dagJSON),
		"generated_rego": generated.Rego,
		"explanation":    generated.Explanation,
		"rag_context":    string(contextJSON),
	}, nil
}

func (a *ConfigGenerationActivities) retrieveContext(ctx context.Context, intent BusinessRuleIntent) ([]migration.KnowledgeItem, error) {
	// Build search query from intent
	searchTerms := []string{intent.Summary}
	for _, pre := range intent.Preconditions {
		searchTerms = append(searchTerms, pre.Description)
	}
	for _, act := range intent.Actions {
		searchTerms = append(searchTerms, act.Description)
	}
	queryText := strings.Join(searchTerms, " ")

	// Retrieve from all categories
	categories := []string{"component_schema", "example", "data_object"}
	return a.EmbeddingService.RetrieveSimilar(ctx, queryText, 5, categories)
}

func (a *ConfigGenerationActivities) generateWithRAG(ctx context.Context, provider *llm.GeminiProvider, intent BusinessRuleIntent, context []migration.KnowledgeItem) (*GeneratedConfig, error) {
	// Build augmented prompt
	prompt := a.buildAugmentedPrompt(intent, context)

	response, err := provider.GenerateResponse(ctx, prompt)
	if err != nil {
		return nil, err
	}

	// Parse response
	return a.parseGeneratedConfig(response, context)
}

func (a *ConfigGenerationActivities) buildAugmentedPrompt(intent BusinessRuleIntent, context []migration.KnowledgeItem) string {
	var sb strings.Builder

	sb.WriteString("### INSTRUCTION\n")
	sb.WriteString("Translate the following business rule into a Titan DAG JSON configuration.\n")
	sb.WriteString("Also generate OPA Rego policy if validation rules are needed.\n\n")

	sb.WriteString("### BUSINESS RULE\n")
	sb.WriteString(fmt.Sprintf("Summary: %s\n", intent.Summary))

	if len(intent.Preconditions) > 0 {
		sb.WriteString("Preconditions:\n")
		for _, pre := range intent.Preconditions {
			sb.WriteString(fmt.Sprintf("  - %s\n", pre.Description))
		}
	}

	if len(intent.Actions) > 0 {
		sb.WriteString("Actions:\n")
		for _, act := range intent.Actions {
			sb.WriteString(fmt.Sprintf("  - %s\n", act.Description))
		}
	}

	sb.WriteString("\n### RETRIEVED CONTEXT (from Titan Knowledge Base)\n")
	for _, item := range context {
		contentJSON, _ := json.MarshalIndent(item.Content, "", "  ")
		sb.WriteString(fmt.Sprintf("\n# %s (%s)\n", item.Name, item.Category))
		if item.Description != "" {
			sb.WriteString(fmt.Sprintf("Description: %s\n", item.Description))
		}
		sb.WriteString(fmt.Sprintf("Schema/Example:\n%s\n", string(contentJSON)))
	}

	sb.WriteString("\n### OUTPUT FORMAT\n")
	sb.WriteString(`Respond with JSON only:
{
  "dag": { ... Titan DAG definition ... },
  "rego": "package ...\n...", // Optional OPA policy
  "explanation": "Brief explanation of the generated configuration"
}`)

	return sb.String()
}

func (a *ConfigGenerationActivities) parseGeneratedConfig(response string, context []migration.KnowledgeItem) (*GeneratedConfig, error) {
	// Clean up response
	cleaned := strings.TrimSpace(response)
	cleaned = strings.TrimPrefix(cleaned, "```json")
	cleaned = strings.TrimPrefix(cleaned, "```")
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	var parsed struct {
		DAG         json.RawMessage `json:"dag"`
		Rego        string          `json:"rego"`
		Explanation string          `json:"explanation"`
	}

	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		// Return raw response as explanation if parsing fails
		return &GeneratedConfig{
			DAG:         json.RawMessage(`{"error": "parsing_failed"}`),
			Explanation: response,
		}, nil
	}

	// Collect context IDs
	contextIDs := make([]string, len(context))
	for i, item := range context {
		contextIDs[i] = item.ID
	}

	return &GeneratedConfig{
		DAG:         parsed.DAG,
		Rego:        parsed.Rego,
		Explanation: parsed.Explanation,
		Context:     contextIDs,
	}, nil
}
