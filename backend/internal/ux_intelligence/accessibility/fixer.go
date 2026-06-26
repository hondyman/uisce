package accessibility

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type ViolationSeverity string

const (
	SeverityCritical ViolationSeverity = "critical"
	SeverityHigh     ViolationSeverity = "high"
	SeverityMedium   ViolationSeverity = "medium"
	SeverityLow      ViolationSeverity = "low"
)

type AccessibilityViolation struct {
	ID          uuid.UUID         `json:"id"`
	ComponentID string            `json:"component_id"`
	Type        string            `json:"type"` // missing_aria, low_contrast, no_alt_text, etc.
	Severity    ViolationSeverity `json:"severity"`
	Description string            `json:"description"`
	WCAG        string            `json:"wcag"` // e.g., "WCAG 2.1 AA 1.4.3"
	CanAutoFix  bool              `json:"can_auto_fix"`
}

type AccessibilityFix struct {
	ViolationID uuid.UUID         `json:"violation_id"`
	FixType     string            `json:"fix_type"`
	Changes     map[string]string `json:"changes"` // attribute -> value
}

type AccessibilityReport struct {
	PageID     uuid.UUID                `json:"page_id"`
	Violations []AccessibilityViolation `json:"violations"`
	Score      int                      `json:"score"`      // 0-100
	WCAGLevel  string                   `json:"wcag_level"` // A, AA, AAA
}

type AccessibilityFixer struct{}

func NewAccessibilityFixer() *AccessibilityFixer {
	return &AccessibilityFixer{}
}

func (f *AccessibilityFixer) Analyze(ctx context.Context, pageID uuid.UUID) (*AccessibilityReport, error) {
	// Mock: Generate accessibility report
	// Real: Analyze page structure, components, colors, interactions

	violations := []AccessibilityViolation{
		{
			ID:          uuid.New(),
			ComponentID: "kpi_card_1",
			Type:        "missing_aria_label",
			Severity:    SeverityHigh,
			Description: "KPI card missing ARIA label for screen readers",
			WCAG:        "WCAG 2.1 AA 4.1.2",
			CanAutoFix:  true,
		},
		{
			ID:          uuid.New(),
			ComponentID: "chart_1",
			Type:        "low_contrast",
			Severity:    SeverityMedium,
			Description: "Chart text has insufficient color contrast (2.8:1, requires 4.5:1)",
			WCAG:        "WCAG 2.1 AA 1.4.3",
			CanAutoFix:  true,
		},
		{
			ID:          uuid.New(),
			ComponentID: "button_submit",
			Type:        "missing_keyboard_nav",
			Severity:    SeverityHigh,
			Description: "Button not accessible via keyboard navigation",
			WCAG:        "WCAG 2.1 AA 2.1.1",
			CanAutoFix:  true,
		},
	}

	report := &AccessibilityReport{
		PageID:     pageID,
		Violations: violations,
		Score:      72,
		WCAGLevel:  "AA (partial)",
	}

	return report, nil
}

func (f *AccessibilityFixer) GenerateFixes(ctx context.Context, violations []AccessibilityViolation) ([]AccessibilityFix, error) {
	fixes := make([]AccessibilityFix, 0)

	for _, v := range violations {
		if !v.CanAutoFix {
			continue
		}

		fix := AccessibilityFix{
			ViolationID: v.ID,
			Changes:     make(map[string]string),
		}

		switch v.Type {
		case "missing_aria_label":
			fix.FixType = "add_aria_label"
			fix.Changes["aria-label"] = fmt.Sprintf("Key Performance Indicator for %s", v.ComponentID)
			fix.Changes["role"] = "region"
		case "low_contrast":
			fix.FixType = "adjust_color_contrast"
			fix.Changes["color"] = "#1a1a1a" // High contrast color
		case "missing_keyboard_nav":
			fix.FixType = "add_keyboard_support"
			fix.Changes["tabindex"] = "0"
			fix.Changes["role"] = "button"
		}

		fixes = append(fixes, fix)
	}

	return fixes, nil
}
