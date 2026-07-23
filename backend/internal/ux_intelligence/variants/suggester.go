package variants

import (
	"context"

	"github.com/google/uuid"
)

type VariantType string

const (
	VariantTypeLayout        VariantType = "layout"
	VariantTypeVisualization VariantType = "visualization"
	VariantTypeInteraction   VariantType = "interaction"
	VariantTypeMobile        VariantType = "mobile"
	VariantTypeAccessibility VariantType = "accessibility"
)

type ComponentVariant struct {
	ID          uuid.UUID   `json:"id"`
	Type        VariantType `json:"type"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Confidence  float64     `json:"confidence"`
	Preview     string      `json:"preview,omitempty"`
	Rationale   string      `json:"rationale"`
}

type VariantRequest struct {
	ComponentID   string `json:"component_id"`
	ComponentType string `json:"component_type"`
	DataType      string `json:"data_type"`
	Cardinality   int    `json:"cardinality"`
	DeviceContext string `json:"device_context"`
}

type VariantSuggester struct{}

func NewVariantSuggester() *VariantSuggester {
	return &VariantSuggester{}
}

func (s *VariantSuggester) Suggest(ctx context.Context, req *VariantRequest) ([]ComponentVariant, error) {
	variants := make([]ComponentVariant, 0)

	// Mock: Generate variant suggestions
	// Real: Analyze data type, cardinality, context, SLO pressure

	if req.ComponentType == "table" {
		variants = append(variants, ComponentVariant{
			ID:          uuid.New(),
			Type:        VariantTypeVisualization,
			Title:       "Convert to Bar Chart",
			Description: "This table could be a bar chart for better readability",
			Confidence:  0.78,
			Rationale:   "Data has low cardinality (< 20 rows) and numeric values suitable for visualization",
		})

		variants = append(variants, ComponentVariant{
			ID:          uuid.New(),
			Type:        VariantTypeMobile,
			Title:       "Card-Based Mobile Layout",
			Description: "Optimize for mobile with card-based layout",
			Confidence:  0.85,
			Rationale:   "Table has many columns that would be difficult to view on mobile",
		})
	}

	if req.ComponentType == "kpi_cluster" {
		variants = append(variants, ComponentVariant{
			ID:          uuid.New(),
			Type:        VariantTypeLayout,
			Title:       "Simplified KPI Layout",
			Description: "Reduce KPI cluster to 3 primary metrics",
			Confidence:  0.72,
			Rationale:   "Current cluster has 8 KPIs which may overwhelm users",
		})

		variants = append(variants, ComponentVariant{
			ID:          uuid.New(),
			Type:        VariantTypeAccessibility,
			Title:       "High-Contrast KPI Cards",
			Description: "Use high-contrast colors for better accessibility",
			Confidence:  0.90,
			Rationale:   "Current color scheme has low contrast ratios",
		})
	}

	return variants, nil
}
