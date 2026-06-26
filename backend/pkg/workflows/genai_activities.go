package workflows

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/hondyman/semlayer/backend/pkg/llm"
)

type GenAIActivities struct {
	ConfigService *llm.LLMConfigService
}

type GenerateContentOutput struct {
	Content string `json:"content"`
}

func (a *GenAIActivities) ActivityGenerateContent(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (*GenerateContentOutput, error) {
	// 1. Get Config from args
	promptTemplate, _ := config["promptTemplate"].(string)
	systemInstruction, _ := config["systemInstruction"].(string)
	modelOverride, _ := config["modelOverride"].(string)

	if promptTemplate == "" {
		return nil, fmt.Errorf("promptTemplate is required in config")
	}

	// 2. Get LLM System Config
	cfg, err := a.ConfigService.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to get LLM config: %w", err)
	}

	// 3. Hydrate Prompt using state
	tmpl, err := template.New("prompt").Parse(promptTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse prompt template: %w", err)
	}

	var promptBuf bytes.Buffer
	if err := tmpl.Execute(&promptBuf, state); err != nil {
		return nil, fmt.Errorf("failed to execute prompt template: %w", err)
	}
	promptStr := promptBuf.String()

	if systemInstruction != "" {
		promptStr = fmt.Sprintf("System: %s\n\nUser: %s", systemInstruction, promptStr)
	}

	// 4. Resolve Provider
	model := cfg.Model
	if modelOverride != "" {
		model = modelOverride
	}

	// Create provider (using API key from config if available)
	provider := llm.NewGeminiProvider(cfg.APIKey, model)

	// 5. Generate
	resp, err := provider.GenerateResponse(ctx, promptStr)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	return &GenerateContentOutput{Content: resp}, nil
}
