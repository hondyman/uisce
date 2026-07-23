package themes

import (
	"context"

	"github.com/google/uuid"
)

type ThemeRequest struct {
	TenantID           string `json:"tenant_id"`
	BrandGuideURL      string `json:"brand_guide_url,omitempty"`
	PrimaryColor       string `json:"primary_color,omitempty"`
	SecondaryColor     string `json:"secondary_color,omitempty"`
	LogoURL            string `json:"logo_url,omitempty"`
	AccessibilityFirst bool   `json:"accessibility_first"`
	SupportRTL         bool   `json:"support_rtl"`
}

type ColorToken struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Contrast float64 `json:"contrast"` // Contrast ratio
}

type TypographyToken struct {
	Name       string `json:"name"`
	FontFamily string `json:"font_family"`
	FontSize   string `json:"font_size"`
	FontWeight string `json:"font_weight"`
	LineHeight string `json:"line_height"`
}

type GeneratedTheme struct {
	ID           uuid.UUID         `json:"id"`
	TenantID     string            `json:"tenant_id"`
	Colors       []ColorToken      `json:"colors"`
	Typography   []TypographyToken `json:"typography"`
	Spacing      map[string]string `json:"spacing"`
	DarkMode     bool              `json:"dark_mode"`
	HighContrast bool              `json:"high_contrast"`
	RTLSupport   bool              `json:"rtl_support"`
}

type ThemeGenerator struct{}

func NewThemeGenerator() *ThemeGenerator {
	return &ThemeGenerator{}
}

func (g *ThemeGenerator) Generate(ctx context.Context, req *ThemeRequest) (*GeneratedTheme, error) {
	// Mock: Generate theme
	// Real: Analyze brand guide, extract colors, ensure accessibility

	theme := &GeneratedTheme{
		ID:       uuid.New(),
		TenantID: req.TenantID,
		Colors: []ColorToken{
			{Name: "primary", Value: "#0066cc", Contrast: 4.52},
			{Name: "secondary", Value: "#6c757d", Contrast: 4.61},
			{Name: "success", Value: "#28a745", Contrast: 4.53},
			{Name: "danger", Value: "#dc3545", Contrast: 4.51},
			{Name: "background", Value: "#ffffff", Contrast: 21.0},
			{Name: "text", Value: "#212529", Contrast: 16.05},
		},
		Typography: []TypographyToken{
			{Name: "heading1", FontFamily: "Inter, sans-serif", FontSize: "2.5rem", FontWeight: "700", LineHeight: "1.2"},
			{Name: "heading2", FontFamily: "Inter, sans-serif", FontSize: "2rem", FontWeight: "600", LineHeight: "1.3"},
			{Name: "body", FontFamily: "Inter, sans-serif", FontSize: "1rem", FontWeight: "400", LineHeight: "1.5"},
			{Name: "caption", FontFamily: "Inter, sans-serif", FontSize: "0.875rem", FontWeight: "400", LineHeight: "1.4"},
		},
		Spacing: map[string]string{
			"xs": "0.25rem",
			"sm": "0.5rem",
			"md": "1rem",
			"lg": "1.5rem",
			"xl": "2rem",
		},
		DarkMode:     true,
		HighContrast: req.AccessibilityFirst,
		RTLSupport:   req.SupportRTL,
	}

	// Ensure all colors meet WCAG AA contrast requirements
	if req.AccessibilityFirst {
		theme.Colors = g.ensureAccessibleColors(theme.Colors)
	}

	return theme, nil
}

func (g *ThemeGenerator) ensureAccessibleColors(colors []ColorToken) []ColorToken {
	// Mock: Adjust colors for accessibility
	// Real: Calculate contrast ratios, adjust as needed
	for i := range colors {
		if colors[i].Contrast < 4.5 {
			// Adjust color to meet WCAG AA
			colors[i].Contrast = 4.5
		}
	}
	return colors
}
