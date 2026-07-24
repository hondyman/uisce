package semantic

import (
	"context"

	"github.com/google/uuid"
)

type ImpactedPage struct {
	PageID      uuid.UUID `json:"page_id"`
	PageName    string    `json:"page_name"`
	ImpactType  string    `json:"impact_type"` // field_removed, type_changed, etc.
	Description string    `json:"description"`
}

type RegressionReport struct {
	ChangeID       string         `json:"change_id"`
	ImpactedPages  []ImpactedPage `json:"impacted_pages"`
	TestsGenerated int            `json:"tests_generated"`
}

type RegressionDetector struct {
	// Lineage engine integration would go here
}

func NewRegressionDetector() *RegressionDetector {
	return &RegressionDetector{}
}

func (r *RegressionDetector) DetectImpact(ctx context.Context, changeType, changeTarget string) (*RegressionReport, error) {
	// Mock implementation
	// Real: Query lineage graph for dependent pages
	report := &RegressionReport{
		ChangeID:       changeTarget,
		ImpactedPages:  make([]ImpactedPage, 0),
		TestsGenerated: 0,
	}

	// Simulate finding impacted pages
	if changeType == "bo_field_change" {
		report.ImpactedPages = append(report.ImpactedPages, ImpactedPage{
			PageID:      uuid.New(),
			PageName:    "Positions Dashboard",
			ImpactType:  "field_type_changed",
			Description: "Field 'market_value' type changed from number to string",
		})
		report.TestsGenerated = 3
	}

	return report, nil
}
