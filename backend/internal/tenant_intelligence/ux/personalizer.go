package ux

import (
	"context"
)

type PersonalizationType string

const (
	PersonalizationTheme      PersonalizationType = "theme"
	PersonalizationLayout     PersonalizationType = "layout"
	PersonalizationNavigation PersonalizationType = "navigation"
)

type Personalization struct {
	Type        PersonalizationType `json:"type"`
	Title       string              `json:"title"`
	Description string              `json:"description"`
	Rationale   string              `json:"rationale"`
	Preview     string              `json:"preview,omitempty"`
}

type UXPersonalizer struct{}

func NewUXPersonalizer() *UXPersonalizer {
	return &UXPersonalizer{}
}

func (up *UXPersonalizer) Personalize(ctx context.Context, tenantID string) ([]Personalization, error) {
	// Mock: Generate UX personalizations
	// Real: Analyze tenant brand, locale, usage patterns, industry

	personalizations := []Personalization{
		{
			Type:        PersonalizationTheme,
			Title:       "Brand-Aligned Theme",
			Description: "Generate theme variant with tenant brand colors and accessible contrast",
			Rationale:   "Tenant has strong brand palette (primary: #0066cc, secondary: #6c757d). Generated theme maintains WCAG AA contrast.",
		},
		{
			Type:        PersonalizationTheme,
			Title:       "CJK Typography Adjustment",
			Description: "Adjust typography for Chinese/Japanese/Korean locale",
			Rationale:   "Tenant locale is 'ja'. Optimized font stack and line-height for CJK characters.",
		},
		{
			Type:        PersonalizationLayout,
			Title:       "Simplified Dashboard Layout",
			Description: "Reduce KPI cluster from 8 to 4 primary metrics",
			Rationale:   "Tenant only uses 4 of 8 KPIs. Simplified layout improves focus and reduces cognitive load.",
		},
		{
			Type:        PersonalizationLayout,
			Title:       "Mobile-Optimized Table Layout",
			Description: "Switch to card-based layout for tables on mobile devices",
			Rationale:   "45% of tenant users access from mobile. Card layout improves mobile UX.",
		},
		{
			Type:        PersonalizationNavigation,
			Title:       "Hide Risk Section",
			Description: "Hide Risk section from main navigation",
			Rationale:   "Risk section has only 12 views in last 30 days. Hiding reduces nav clutter.",
		},
		{
			Type:        PersonalizationNavigation,
			Title:       "Promote Trade Approval to Top-Level",
			Description: "Move Trade Approval workflow to top-level navigation",
			Rationale:   "Trade Approval is used heavily (234 executions in last 7 days). Promoting improves accessibility.",
		},
	}

	return personalizations, nil
}
