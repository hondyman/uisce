package catalog

import (
	"context"
	"encoding/json"
	"fmt"
)

// LLMClient interface (mockable)
type LLMClient interface {
	Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}

// BusinessTermGenerator uses AI to generate business terms from technical metadata
type BusinessTermGenerator struct {
	llm LLMClient
}

// NewBusinessTermGenerator creates a new generator
func NewBusinessTermGenerator(llm LLMClient) *BusinessTermGenerator {
	return &BusinessTermGenerator{llm: llm}
}

// GenerationInput contains all data needed for generation
type GenerationInput struct {
	TechnicalColumns []TechnicalColumn  `json:"technical_columns"`
	SemanticTerms    []SemanticTerm     `json:"semantic_terms"`
	DataProfile      DataProfile        `json:"data_profile"`
	ComplianceHints  ComplianceHints    `json:"compliance_hints"`
	Hierarchy        FinancialHierarchy `json:"financial_hierarchy"`
}

type TechnicalColumn struct {
	Table        string   `json:"table"`
	Column       string   `json:"column"`
	DataType     string   `json:"dataType"`
	SampleValues []string `json:"sampleValues"`
	Nullable     bool     `json:"nullable"`
}

type SemanticTerm struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

type DataProfile struct {
	DistinctCount int      `json:"distinctCount"`
	NullRatio     float64  `json:"nullRatio"`
	Patterns      []string `json:"patterns"`
	ExampleValues []string `json:"exampleValues"`
}

type ComplianceHints struct {
	PIICandidate          bool     `json:"piiCandidate"`
	SensitivityIndicators []string `json:"sensitivityIndicators"`
}

type FinancialHierarchy struct {
	Level1Options  []string            `json:"level1Options"`
	Level2Options  map[string][]string `json:"level2Options"`
	Level3Examples map[string][]string `json:"level3Examples"`
}

// BusinessTerm represents the generated business term (matching the JSON schema)
type BusinessTermDraft struct {
	BusinessTermID      string            `json:"businessTermId"`
	Name                string            `json:"name"`
	Definition          string            `json:"definition"`
	PIIFlag             bool              `json:"piiFlag"`
	Sensitivity         string            `json:"sensitivity"` // LOW, MEDIUM, HIGH
	Residency           string            `json:"residency"`
	Hierarchy           HierarchyPosition `json:"hierarchy"`
	SourceSemanticTerms []string          `json:"sourceSemanticTerms"`
	SourceColumns       []string          `json:"sourceColumns"`
	Tags                []string          `json:"tags"`
}

type HierarchyPosition struct {
	Level1 string `json:"level1"`
	Level2 string `json:"level2"`
	Level3 string `json:"level3"`
}

// Generate proposes a single business term from inputs with retry logic
func (g *BusinessTermGenerator) Generate(ctx context.Context, input GenerationInput) (*BusinessTermDraft, error) {
	prompt := g.buildPrompt(input)
	systemPrompt := `
You are an expert financial data steward and metadata architect.
Return ONLY valid JSON matching the BusinessTerm schema.
`

	var lastErr error
	for i := 0; i < 3; i++ {
		response, err := g.llm.Generate(ctx, systemPrompt, prompt)
		if err != nil {
			lastErr = err
			continue
		}

		var draft BusinessTermDraft
		if jsonErr := json.Unmarshal([]byte(response), &draft); jsonErr != nil {
			lastErr = fmt.Errorf("invalid JSON from LLM: %w", jsonErr)
			prompt = g.addJsonCorrectionHint(prompt, response, jsonErr)
			continue
		}

		return &draft, nil
	}

	return nil, fmt.Errorf("failed after retries: %w", lastErr)
}

func (g *BusinessTermGenerator) buildPrompt(input GenerationInput) string {
	return fmt.Sprintf(`
Technical Columns:
%s

Semantic Terms:
%s

Data Profile:
%s

Compliance Hints:
%s

Financial Hierarchy:
%s

Generate a single Business Term JSON object that matches the schema exactly.
Return ONLY the JSON object with no commentary.
`,
		marshalOrEmpty(input.TechnicalColumns),
		marshalOrEmpty(input.SemanticTerms),
		marshalOrEmpty(input.DataProfile),
		marshalOrEmpty(input.ComplianceHints),
		marshalOrEmpty(input.Hierarchy),
	)
}

func (g *BusinessTermGenerator) addJsonCorrectionHint(prompt, raw string, err error) string {
	return prompt + "\n\nThe previous JSON was invalid:\n" +
		raw + "\nError:\n" + err.Error() +
		"\nPlease return corrected JSON only."
}

func marshalOrEmpty(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(b)
}
