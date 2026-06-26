package pii

import (
	"context"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/pagestudio"
)

type RiskLevel string

const (
	RiskHigh   RiskLevel = "high"   // SSN, Account Number
	RiskMedium RiskLevel = "medium" // Email, Phone
	RiskLow    RiskLevel = "low"    // Zip, City
	RiskNone   RiskLevel = "none"
)

type PiiZone struct {
	ComponentID string    `json:"component_id"`
	Field       string    `json:"field"`
	Risk        RiskLevel `json:"risk"`
	Reason      string    `json:"reason"`
}

type Heatmap struct {
	PageID string    `json:"page_id"`
	Zones  []PiiZone `json:"zones"`
}

type Scanner struct{}

func NewScanner() *Scanner {
	return &Scanner{}
}

func (s *Scanner) ScanPage(ctx context.Context, page *pagestudio.CorePage) (*Heatmap, error) {
	heatmap := &Heatmap{
		PageID: page.ID.String(),
		Zones:  make([]PiiZone, 0),
	}

	compStr := string(page.Components)

	// Mock PII detection using simple string matching
	if strings.Contains(compStr, "ssn") || strings.Contains(compStr, "social_security") {
		heatmap.Zones = append(heatmap.Zones, PiiZone{
			ComponentID: "unknown", // In real impl, would parse component tree
			Field:       "ssn",
			Risk:        RiskHigh,
			Reason:      "Field 'ssn' is classified as High Risk PII.",
		})
	}

	if strings.Contains(compStr, "email") {
		heatmap.Zones = append(heatmap.Zones, PiiZone{
			ComponentID: "unknown",
			Field:       "email",
			Risk:        RiskMedium,
			Reason:      "Field 'email' is classified as Medium Risk PII.",
		})
	}

	return heatmap, nil
}
