package activities

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"go.temporal.io/sdk/activity"
)

// ============================================================================
// AML SCREENING ACTIVITY (STEP 2)
// ============================================================================

// AMLScreeningInput represents input for AML screening activity
type AMLScreeningInput struct {
	ClientID             string
	ScreeningProvider    string // lexis_nexis, worldcheck, dow_jones, internal
	NetWorth             *float64
	CountryOfCitizenship *string
	TaxResidencyCountry  *string
	SourceOfFunds        *string
	PerformManualReview  bool
	RequiresDueDiligence bool
}

// AMLScreeningOutput represents result from AML screening activity
type AMLScreeningOutput struct {
	ScreeningID          string
	RiskScore            float64
	RiskLevel            string // low, medium, high, critical
	OverallStatus        string // clear, flagged, rejected
	WatchlistMatch       bool
	WatchlistMatches     []string
	SanctionsMatch       bool
	PEPMatch             bool
	PEPLevel             *string // low, medium, high
	HighNetWorthFlag     bool
	UnknownFundsFlag     bool
	RiskyCountriesFlag   bool
	RiskyCountries       []string
	ManualReviewRequired bool
	ManualReviewReason   string
	ApprovalRequired     bool
	ApprovalDeadline     time.Time
	ScreeningCompletedAt time.Time
}

// PerformAMLScreeningActivity performs comprehensive AML screening against watchlists,
// sanctions lists, PEP databases, and adverse media
// This is Step 2 of the onboarding workflow
func PerformAMLScreeningActivity(ctx context.Context, input AMLScreeningInput) (*AMLScreeningOutput, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting AML screening activity",
		"clientID", input.ClientID,
		"provider", input.ScreeningProvider)

	// Simulate external API calls based on provider
	var watchlistMatch bool
	var sanctionsMatch bool
	var pepMatch bool
	var pepLevel *string
	var riskCountries []string
	var watchlistMatches []string

	// Simulate provider-specific screening
	switch input.ScreeningProvider {
	case "lexis_nexis":
		watchlistMatch, sanctionsMatch, pepMatch, pepLevel, riskCountries, watchlistMatches = simulateLexisNexisScreening(input)
	case "worldcheck":
		watchlistMatch, sanctionsMatch, pepMatch, pepLevel, riskCountries, watchlistMatches = simulateWorldCheckScreening(input)
	case "dow_jones":
		watchlistMatch, sanctionsMatch, pepMatch, pepLevel, riskCountries, watchlistMatches = simulateDowJonesScreening(input)
	default:
		// Internal screening
		watchlistMatch, sanctionsMatch, pepMatch, pepLevel, riskCountries, watchlistMatches = performInternalAMLScreening(input)
	}

	// Determine flags
	unknownFundsFlag := input.SourceOfFunds == nil || *input.SourceOfFunds == "" || *input.SourceOfFunds == "unknown"
	highNetWorthFlag := input.NetWorth != nil && *input.NetWorth > 5000000 // $5M+
	riskyCountriesFlag := len(riskCountries) > 0

	// Compute risk score using weighted algorithm
	riskScore := computeAMLRiskScore(
		watchlistMatch,
		sanctionsMatch,
		pepMatch,
		highNetWorthFlag,
		unknownFundsFlag,
		riskyCountriesFlag,
	)

	// Determine risk level
	riskLevel := determineRiskLevel(riskScore)

	// Determine overall status and manual review requirement
	manualReviewRequired := riskScore >= 60 || input.PerformManualReview || input.RequiresDueDiligence
	var overallStatus string
	var manualReviewReason string

	if sanctionsMatch {
		overallStatus = "flagged"
		manualReviewReason = "Sanctions match detected - requires escalation"
		manualReviewRequired = true
	} else if riskScore >= 80 {
		overallStatus = "flagged"
		manualReviewReason = fmt.Sprintf("Critical risk score: %.0f - requires immediate review", riskScore)
		manualReviewRequired = true
	} else if manualReviewRequired {
		overallStatus = "flagged"
		manualReviewReason = "Manual review requested or due diligence required"
	} else {
		overallStatus = "clear"
		manualReviewReason = "Screening passed - cleared for next step"
	}

	// Set approval deadline (24-48 hours for manual review)
	approvalDeadline := time.Now()
	if riskScore >= 80 {
		approvalDeadline = approvalDeadline.Add(24 * time.Hour) // Critical: 24h
	} else {
		approvalDeadline = approvalDeadline.Add(48 * time.Hour) // Standard: 48h
	}

	output := &AMLScreeningOutput{
		ScreeningID:          fmt.Sprintf("scr_%s_%d", input.ClientID, time.Now().Unix()),
		RiskScore:            riskScore,
		RiskLevel:            riskLevel,
		OverallStatus:        overallStatus,
		WatchlistMatch:       watchlistMatch,
		WatchlistMatches:     watchlistMatches,
		SanctionsMatch:       sanctionsMatch,
		PEPMatch:             pepMatch,
		PEPLevel:             pepLevel,
		HighNetWorthFlag:     highNetWorthFlag,
		UnknownFundsFlag:     unknownFundsFlag,
		RiskyCountriesFlag:   riskyCountriesFlag,
		RiskyCountries:       riskCountries,
		ManualReviewRequired: manualReviewRequired,
		ManualReviewReason:   manualReviewReason,
		ApprovalRequired:     manualReviewRequired,
		ApprovalDeadline:     approvalDeadline,
		ScreeningCompletedAt: time.Now(),
	}

	logger.Info("AML screening completed",
		"clientID", input.ClientID,
		"riskScore", riskScore,
		"riskLevel", riskLevel,
		"manualReviewRequired", manualReviewRequired)

	return output, nil
}

// ============================================================================
// EXTERNAL PROVIDER SIMULATION
// ============================================================================

// simulateLexisNexisScreening simulates Lexis Nexis API response
func simulateLexisNexisScreening(input AMLScreeningInput) (bool, bool, bool, *string, []string, []string) {
	// Simulate API call to Lexis Nexis
	// 70% no match, 20% watchlist, 5% sanctions, 5% PEP
	roll := rand.Intn(100)

	watchlistMatch := false
	sanctionsMatch := false
	pepMatch := false
	var pepLevel *string
	var riskCountries []string
	var watchlistMatches []string

	if roll < 70 {
		// Clear
	} else if roll < 90 {
		// Watchlist match
		watchlistMatch = true
		watchlistMatches = []string{"OFAC_SDN_List", "UN_Security_Council_List"}
	} else if roll < 95 {
		// Sanctions
		sanctionsMatch = true
		watchlistMatches = []string{"EU_Sanctions_List"}
	} else {
		// PEP match
		pepMatch = true
		level := "medium"
		pepLevel = &level
	}

	// Check for risky countries
	if input.CountryOfCitizenship != nil {
		riskyList := []string{"KP", "IR", "SY", "CU"}
		for _, rc := range riskyList {
			if *input.CountryOfCitizenship == rc {
				riskCountries = []string{*input.CountryOfCitizenship}
				break
			}
		}
	}

	return watchlistMatch, sanctionsMatch, pepMatch, pepLevel, riskCountries, watchlistMatches
}

// simulateWorldCheckScreening simulates World Check API response
func simulateWorldCheckScreening(input AMLScreeningInput) (bool, bool, bool, *string, []string, []string) {
	// Similar to Lexis Nexis with different hit rates
	roll := rand.Intn(100)

	watchlistMatch := false
	sanctionsMatch := false
	pepMatch := false
	var pepLevel *string
	var riskCountries []string
	var watchlistMatches []string

	if roll < 75 {
		// Clear
	} else if roll < 85 {
		// Watchlist match
		watchlistMatch = true
		watchlistMatches = []string{"WC_Main_List"}
	} else if roll < 92 {
		// Sanctions
		sanctionsMatch = true
		watchlistMatches = []string{"UK_Sanctions_List"}
	} else {
		// PEP match
		pepMatch = true
		level := "high"
		pepLevel = &level
	}

	// Check country risk
	if input.TaxResidencyCountry != nil {
		riskyList := []string{"KP", "IR"}
		for _, rc := range riskyList {
			if *input.TaxResidencyCountry == rc {
				riskCountries = []string{*input.TaxResidencyCountry}
				break
			}
		}
	}

	return watchlistMatch, sanctionsMatch, pepMatch, pepLevel, riskCountries, watchlistMatches
}

// simulateDowJonesScreening simulates Dow Jones Risk & Compliance API response
func simulateDowJonesScreening(input AMLScreeningInput) (bool, bool, bool, *string, []string, []string) {
	roll := rand.Intn(100)

	watchlistMatch := false
	sanctionsMatch := false
	pepMatch := false
	var pepLevel *string
	var riskCountries []string
	var watchlistMatches []string

	if roll < 65 {
		// Clear
	} else if roll < 80 {
		// Watchlist match
		watchlistMatch = true
		watchlistMatches = []string{"DJ_Watchlist"}
	} else if roll < 90 {
		// Sanctions + Adverse Media
		sanctionsMatch = true
		watchlistMatches = []string{"US_Sanctions_List"}
	} else {
		// PEP + Adverse Media
		pepMatch = true
		level := "low"
		pepLevel = &level
	}

	return watchlistMatch, sanctionsMatch, pepMatch, pepLevel, riskCountries, watchlistMatches
}

// performInternalAMLScreening performs rule-based AML screening
func performInternalAMLScreening(input AMLScreeningInput) (bool, bool, bool, *string, []string, []string) {
	// Internal rule-based screening (no watchlist)
	watchlistMatch := false
	sanctionsMatch := false
	pepMatch := false
	var pepLevel *string

	// Just check country risk
	var riskCountries []string
	if input.CountryOfCitizenship != nil {
		riskyList := []string{"KP", "IR", "SY", "CU", "VE"}
		for _, rc := range riskyList {
			if *input.CountryOfCitizenship == rc {
				riskCountries = []string{*input.CountryOfCitizenship}
				break
			}
		}
	}

	return watchlistMatch, sanctionsMatch, pepMatch, pepLevel, riskCountries, []string{}
}

// ============================================================================
// RISK SCORING
// ============================================================================

// computeAMLRiskScore calculates comprehensive AML risk score using weighted algorithm
// Based on FATF and FinCEN guidelines (matching backend implementation)
func computeAMLRiskScore(
	watchlistMatch, sanctionsMatch, pepMatch, highNetWorth, unknownFunds, riskyCountries bool,
) float64 {
	score := 0.0

	// 1. WATCHLIST MATCHING (Max: 50 points)
	if watchlistMatch {
		score += 50.0
	}

	// 2. SANCTIONS (Max: 40 points)
	if sanctionsMatch {
		score += 40.0
	}

	// 3. PEP (Max: 25 points)
	if pepMatch {
		score += 25.0
	}

	// 4. HIGH NET WORTH (Max: 20 points)
	if highNetWorth {
		score += 20.0
	}

	// 5. UNKNOWN SOURCE OF FUNDS (Max: 25 points)
	if unknownFunds {
		if highNetWorth {
			score += 25.0
		} else {
			score += 15.0
		}
	}

	// 6. HIGH-RISK COUNTRIES (Max: 30 points)
	if riskyCountries {
		score += 30.0
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// determineRiskLevel converts risk score to risk level
func determineRiskLevel(score float64) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	case score >= 20:
		return "medium_low"
	default:
		return "low"
	}
}

// ============================================================================
// ESCALATION ACTIVITY FOR CRITICAL AML FINDINGS
// ============================================================================

// EscalateAMLFindingsActivity escalates critical AML findings to management
// Called when risk score >= 80 or sanctions match
func EscalateAMLFindingsActivity(ctx context.Context, clientID string, riskScore float64, reason string) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Escalating AML findings",
		"clientID", clientID,
		"riskScore", riskScore,
		"reason", reason)

	// In real implementation:
	// 1. Create escalation ticket in compliance system
	// 2. Route to AML Manager with priority based on risk
	// 3. Set escalation deadline (24h for critical)
	// 4. Send notification to compliance team

	escalationResult := map[string]interface{}{
		"escalation_id":     fmt.Sprintf("esc_%s_%d", clientID, time.Now().Unix()),
		"escalated_at":      time.Now(),
		"escalation_level":  "management",
		"escalation_reason": reason,
		"review_deadline":   time.Now().Add(24 * time.Hour),
		"status":            "escalated",
	}

	logger.Info("AML findings escalated", "escalationID", escalationResult["escalation_id"])
	return escalationResult, nil
}

// SendAMLApprovalNotificationActivity notifies compliance officers of pending reviews
func SendAMLApprovalNotificationActivity(ctx context.Context, clientID string, screeningID string, deadline time.Time) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending AML approval notification",
		"clientID", clientID,
		"screeningID", screeningID,
		"deadline", deadline)

	// In real implementation:
	// 1. Send email/SMS to assigned compliance officer
	// 2. Create task in task management system
	// 3. Update compliance dashboard

	notificationResult := map[string]interface{}{
		"notification_sent":   true,
		"notification_method": "email",
		"recipient_role":      "ComplianceOfficer",
		"deadline":            deadline,
		"status":              "pending_review",
	}

	logger.Info("Notification sent")
	return notificationResult, nil
}

// RecordAMLScreeningAuditActivity records screening activity for compliance audit trail
func RecordAMLScreeningAuditActivity(ctx context.Context, clientID string, screeningOutput *AMLScreeningOutput) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Recording AML screening audit trail",
		"clientID", clientID,
		"screeningID", screeningOutput.ScreeningID)

	// Convert to JSON for audit trail
	auditEntry := map[string]interface{}{
		"event_type":       "aml_screening_completed",
		"client_id":        clientID,
		"screening_id":     screeningOutput.ScreeningID,
		"risk_score":       screeningOutput.RiskScore,
		"risk_level":       screeningOutput.RiskLevel,
		"overall_status":   screeningOutput.OverallStatus,
		"timestamp":        time.Now(),
		"screening_output": screeningOutput,
	}

	auditJSON, _ := json.Marshal(auditEntry)
	logger.Info("Audit trail recorded", "audit", string(auditJSON))

	return map[string]interface{}{
		"audit_id":    fmt.Sprintf("audit_%d", time.Now().Unix()),
		"recorded_at": time.Now(),
		"status":      "recorded",
	}, nil
}
