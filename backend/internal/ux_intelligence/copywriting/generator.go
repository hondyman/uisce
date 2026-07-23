package copywriting

import (
	"context"
)

type CopyTone string

const (
	ToneFormal     CopyTone = "formal"
	ToneFriendly   CopyTone = "friendly"
	ToneConcise    CopyTone = "concise"
	ToneEnterprise CopyTone = "enterprise"
)

type CopyRequest struct {
	FieldName     string   `json:"field_name"`
	FieldType     string   `json:"field_type"`
	ComponentType string   `json:"component_type"`
	PageContext   string   `json:"page_context"`
	Locale        string   `json:"locale"`
	Tone          CopyTone `json:"tone"`
}

type GeneratedCopy struct {
	Label          string `json:"label"`
	Tooltip        string `json:"tooltip,omitempty"`
	Description    string `json:"description,omitempty"`
	Placeholder    string `json:"placeholder,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	SuccessMessage string `json:"success_message,omitempty"`
	EmptyState     string `json:"empty_state,omitempty"`
}

type CopyGenerator struct {
	// LLM integration would go here
}

func NewCopyGenerator() *CopyGenerator {
	return &CopyGenerator{}
}

func (g *CopyGenerator) Generate(ctx context.Context, req *CopyRequest) (*GeneratedCopy, error) {
	// Mock: Generate UX copy
	// Real: Call LLM with semantic context

	copy := &GeneratedCopy{}

	// Generate based on locale
	switch req.Locale {
	case "en":
		copy.Label = formatLabel(req.FieldName, "en")
		copy.Tooltip = "The current market value in US Dollars"
		copy.Description = "This field shows the total market value of the position"
	case "es":
		copy.Label = formatLabel(req.FieldName, "es")
		copy.Tooltip = "El valor de mercado actual en dólares estadounidenses"
		copy.Description = "Este campo muestra el valor de mercado total de la posición"
	case "fr":
		copy.Label = formatLabel(req.FieldName, "fr")
		copy.Tooltip = "La valeur marchande actuelle en dollars américains"
		copy.Description = "Ce champ affiche la valeur marchande totale de la position"
	case "ja":
		copy.Label = "時価総額 (USD)"
		copy.Tooltip = "米ドル建ての現在の時価総額"
		copy.Description = "このフィールドはポジションの総時価総額を表示します"
	default:
		copy.Label = formatLabel(req.FieldName, "en")
	}

	// Add accessibility-friendly phrasing
	if req.ComponentType == "kpi" {
		copy.Description = "Key Performance Indicator: " + copy.Description
	}

	return copy, nil
}

func formatLabel(fieldName, locale string) string {
	// Simple formatting - real implementation would use NLP
	switch locale {
	case "es":
		if fieldName == "market_value_usd" {
			return "Valor de Mercado (USD)"
		}
	case "fr":
		if fieldName == "market_value_usd" {
			return "Valeur Marchande (USD)"
		}
	default:
		if fieldName == "market_value_usd" {
			return "Market Value (USD)"
		}
	}
	return fieldName
}
