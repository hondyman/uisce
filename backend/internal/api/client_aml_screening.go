package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ============================================================================
// AML SCREENING TYPES & CONSTANTS
// ============================================================================

// AMLScreeningResult represents comprehensive AML screening data
type AMLScreeningResult struct {
	ID                    string          `json:"id"`
	TenantID              string          `json:"tenant_id"`
	DatasourceID          string          `json:"datasource_id"`
	ClientID              string          `json:"client_id"`
	ScreeningDate         time.Time       `json:"screening_date"`
	ScreeningProvider     string          `json:"screening_provider"` // lexis_nexis, worldcheck, internal, dow_jones
	ScreeningStatus       string          `json:"screening_status"`   // pending, in_progress, completed, failed
	RiskScore             float64         `json:"risk_score"`         // 0-100
	RiskLevel             string          `json:"risk_level"`         // low, medium, high, critical
	OverallStatus         string          `json:"overall_status"`     // clear, flagged, rejected
	WatchlistMatch        bool            `json:"watchlist_match"`
	WatchlistMatches      []string        `json:"watchlist_matches,omitempty"`
	SanctionsMatch        bool            `json:"sanctions_match"`
	SanctionsDetails      *string         `json:"sanctions_details,omitempty"`
	PEPMatch              bool            `json:"pep_match"`
	PEPDetails            *string         `json:"pep_details,omitempty"`
	PEPLevel              *string         `json:"pep_level,omitempty"` // low, medium, high
	HighNetWorthFlag      bool            `json:"high_net_worth_flag"`
	UnknownFundsFlag      bool            `json:"unknown_funds_flag"`
	RiskyCountriesFlag    bool            `json:"risky_countries_flag"`
	RiskyCountries        []string        `json:"risky_countries,omitempty"`
	SourceOfFundsVerified bool            `json:"source_of_funds_verified"`
	BeneficialOwnerFlags  bool            `json:"beneficial_owner_flags"`
	BeneficialOwners      []string        `json:"beneficial_owners,omitempty"`
	AdverseMediaFlag      bool            `json:"adverse_media_flag"`
	AdverseMediaDetails   *string         `json:"adverse_media_details,omitempty"`
	ManualReviewRequired  bool            `json:"manual_review_required"`
	ManualReviewReason    *string         `json:"manual_review_reason,omitempty"`
	ApprovedBy            *string         `json:"approved_by,omitempty"`
	ApprovedAt            *time.Time      `json:"approved_at,omitempty"`
	RejectedBy            *string         `json:"rejected_by,omitempty"`
	RejectedAt            *time.Time      `json:"rejected_at,omitempty"`
	RejectionReason       *string         `json:"rejection_reason,omitempty"`
	RawAPIResponse        json.RawMessage `json:"raw_api_response,omitempty"`
	ComplianceNotes       *string         `json:"compliance_notes,omitempty"`
	CreatedBy             string          `json:"created_by"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// AMLScreeningRequest represents a request to perform AML screening
type AMLScreeningRequest struct {
	ClientID             string  `json:"client_id" binding:"required"`
	ScreeningProvider    string  `json:"screening_provider" binding:"required"` // lexis_nexis, worldcheck, internal, dow_jones
	PerformManualReview  bool    `json:"perform_manual_review"`
	RequiresDueDiligence bool    `json:"requires_due_diligence"`
	DueDiligenceReason   *string `json:"due_diligence_reason"`
}

// AMLRiskScoreInput contains data for risk score calculation
type AMLRiskScoreInput struct {
	NetWorth             *float64
	AnnualIncome         *float64
	CountryOfCitizenship *string
	TaxResidencyCountry  *string
	SourceOfFunds        *string
	WatchlistMatch       bool
	SanctionsMatch       bool
	PEPStatus            bool
	AdverseMedia         bool
	HighRiskCountry      bool
	UnknownSourceOfFunds bool
}

// AMLScreeningReviewRequest represents approval/rejection of AML screening
type AMLScreeningReviewRequest struct {
	ScreeningID     string  `json:"screening_id" binding:"required"`
	ApprovalStatus  string  `json:"approval_status" binding:"required"` // approved, rejected
	ComplianceNotes string  `json:"compliance_notes"`
	RejectionReason *string `json:"rejection_reason"`
}

// AMLScreeningService handles AML screening operations
type AMLScreeningService struct {
	db *sqlx.DB
}

// NewAMLScreeningService creates a new AML screening service
func NewAMLScreeningService(db *sqlx.DB) *AMLScreeningService {
	return &AMLScreeningService{db: db}
}

// ============================================================================
// AML RISK SCORING ENGINE
// ============================================================================

// ComputeAMLRiskScore calculates comprehensive AML risk score using weighted algorithm
// Based on FATF and FinCEN guidelines
func (s *AMLScreeningService) ComputeAMLRiskScore(input AMLRiskScoreInput) (float64, string) {
	score := 0.0

	// 1. WATCHLIST MATCHING (Max: 50 points)
	// Critical for compliance - immediate red flag
	if input.WatchlistMatch {
		score += 50.0
	}

	// 2. SANCTIONS (Max: 40 points)
	// Very high risk - strong indicator of illicit activity
	if input.SanctionsMatch {
		score += 40.0
	}

	// 3. PEP (POLITICALLY EXPOSED PERSON) (Max: 25 points)
	// Medium-high risk - requires escalation but not automatic rejection
	if input.PEPStatus {
		score += 25.0
	}

	// 4. HIGH NET WORTH (Max: 20 points)
	// Medium risk - higher net worth = higher transaction volume = higher risk
	if input.NetWorth != nil && *input.NetWorth > 10000000 { // > $10M
		score += 20.0
	} else if input.NetWorth != nil && *input.NetWorth > 5000000 { // > $5M
		score += 10.0
	}

	// 5. UNKNOWN SOURCE OF FUNDS (Max: 25 points)
	// Critical for high-net-worth clients
	if input.UnknownSourceOfFunds {
		if input.NetWorth != nil && *input.NetWorth > 5000000 {
			score += 25.0 // Higher impact for wealthy clients
		} else {
			score += 15.0
		}
	}

	// 6. HIGH-RISK COUNTRIES (Max: 30 points)
	// Based on FATF Grey/Black lists
	if input.HighRiskCountry {
		score += 30.0
	}

	// 7. ADVERSE MEDIA / NEGATIVE NEWS (Max: 20 points)
	// Indicates potential reputation or legal risk
	if input.AdverseMedia {
		score += 20.0
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	// Determine risk level
	var riskLevel string
	switch {
	case score >= 80:
		riskLevel = "critical"
	case score >= 60:
		riskLevel = "high"
	case score >= 40:
		riskLevel = "medium"
	case score >= 20:
		riskLevel = "medium_low"
	default:
		riskLevel = "low"
	}

	return score, riskLevel
}

// ============================================================================
// DATABASE OPERATIONS
// ============================================================================

// CreateAMLScreening creates a new AML screening record
func (s *AMLScreeningService) CreateAMLScreening(
	ctx context.Context,
	tenantID, datasourceID, userID string,
	req *AMLScreeningRequest,
	client *Client,
) (*AMLScreeningResult, error) {

	// Step 1: Compute initial risk score
	riskInput := AMLRiskScoreInput{
		NetWorth:             client.NetWorth,
		AnnualIncome:         client.AnnualIncome,
		CountryOfCitizenship: client.CountryOfCitizenship,
		TaxResidencyCountry:  client.TaxResidencyCountry,
	}

	riskScore, riskLevel := s.ComputeAMLRiskScore(riskInput)

	// Step 2: Determine if manual review is required
	manualReviewRequired := req.PerformManualReview || riskScore >= 60

	screening := &AMLScreeningResult{
		TenantID:             tenantID,
		DatasourceID:         datasourceID,
		ClientID:             req.ClientID,
		ScreeningDate:        time.Now(),
		ScreeningProvider:    req.ScreeningProvider,
		ScreeningStatus:      "pending",
		RiskScore:            riskScore,
		RiskLevel:            riskLevel,
		OverallStatus:        "pending",
		ManualReviewRequired: manualReviewRequired,
		ManualReviewReason:   req.DueDiligenceReason,
		CreatedBy:            userID,
	}

	query := `
		INSERT INTO kyc_aml_results (
			tenant_id, datasource_id, client_id, screening_date, screening_provider,
			screening_status, risk_score, risk_level, overall_status,
			watchlist_match, sanctions_match, pep_match,
			high_net_worth_flag, unknown_funds_flag, risky_countries_flag,
			source_of_funds_verified, beneficial_owner_flags, adverse_media_flag,
			manual_review_required, manual_review_reason, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23
		) RETURNING id, created_at, updated_at
	`

	err := s.db.QueryRowxContext(ctx, query,
		screening.TenantID, screening.DatasourceID, screening.ClientID,
		screening.ScreeningDate, screening.ScreeningProvider,
		screening.ScreeningStatus, screening.RiskScore, screening.RiskLevel,
		screening.OverallStatus, screening.WatchlistMatch, screening.SanctionsMatch,
		screening.PEPMatch, screening.HighNetWorthFlag, screening.UnknownFundsFlag,
		screening.RiskyCountriesFlag, screening.SourceOfFundsVerified,
		screening.BeneficialOwnerFlags, screening.AdverseMediaFlag,
		screening.ManualReviewRequired, screening.ManualReviewReason,
		screening.CreatedBy, time.Now(), time.Now(),
	).StructScan(screening)

	if err != nil {
		return nil, fmt.Errorf("failed to create AML screening: %w", err)
	}

	return screening, nil
}

// GetAMLScreening retrieves an AML screening result by ID
func (s *AMLScreeningService) GetAMLScreening(ctx context.Context, tenantID, screeningID string) (*AMLScreeningResult, error) {
	screening := &AMLScreeningResult{}
	query := `
		SELECT * FROM kyc_aml_results
		WHERE id = $1 AND tenant_id = $2
	`
	err := s.db.QueryRowxContext(ctx, query, screeningID, tenantID).StructScan(screening)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("screening not found")
		}
		return nil, fmt.Errorf("failed to get AML screening: %w", err)
	}
	return screening, nil
}

// GetLatestClientAMLScreening retrieves the most recent AML screening for a client
func (s *AMLScreeningService) GetLatestClientAMLScreening(ctx context.Context, clientID string) (*AMLScreeningResult, error) {
	screening := &AMLScreeningResult{}
	query := `
		SELECT * FROM kyc_aml_results
		WHERE client_id = $1
		ORDER BY screening_date DESC
		LIMIT 1
	`
	err := s.db.QueryRowxContext(ctx, query, clientID).StructScan(screening)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no screening found for client")
		}
		return nil, fmt.Errorf("failed to get AML screening: %w", err)
	}
	return screening, nil
}

// UpdateAMLScreeningStatus updates the screening status and overall status
func (s *AMLScreeningService) UpdateAMLScreeningStatus(
	ctx context.Context,
	screeningID, tenantID, userID string,
	approvalStatus string,
	complianceNotes string,
	rejectionReason *string,
) (*AMLScreeningResult, error) {

	var overallStatus string
	switch approvalStatus {
	case "approved":
		overallStatus = "clear"
	case "rejected":
		overallStatus = "rejected"
	default:
		return nil, fmt.Errorf("invalid approval status: %s", approvalStatus)
	}

	query := `
		UPDATE kyc_aml_results
		SET screening_status = $1, overall_status = $2, approved_by = $3, approved_at = $4,
		    rejected_by = CASE WHEN $1 = 'rejected' THEN $3 ELSE NULL END,
		    rejected_at = CASE WHEN $1 = 'rejected' THEN $4 ELSE NULL END,
		    rejection_reason = $5, compliance_notes = $6, updated_at = $4
		WHERE id = $7 AND tenant_id = $8
		RETURNING *
	`

	screening := &AMLScreeningResult{}
	now := time.Now()
	err := s.db.QueryRowxContext(ctx, query,
		"completed", overallStatus, userID, now,
		rejectionReason, complianceNotes,
		screeningID, tenantID,
	).StructScan(screening)

	if err != nil {
		return nil, fmt.Errorf("failed to update AML screening: %w", err)
	}

	return screening, nil
}

// GetClientAMLScreeningHistory retrieves all AML screenings for a client
func (s *AMLScreeningService) GetClientAMLScreeningHistory(
	ctx context.Context,
	clientID string,
	limit int,
) ([]*AMLScreeningResult, error) {
	if limit == 0 {
		limit = 10
	}

	screenings := []*AMLScreeningResult{}
	query := `
		SELECT * FROM kyc_aml_results
		WHERE client_id = $1
		ORDER BY screening_date DESC
		LIMIT $2
	`
	err := s.db.SelectContext(ctx, &screenings, query, clientID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get AML screening history: %w", err)
	}

	return screenings, nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// IsAMLClearanceRequired determines if AML clearance is needed before account creation
func (s *AMLScreeningService) IsAMLClearanceRequired(screening *AMLScreeningResult) bool {
	// Account creation blocked if:
	// 1. Screening status not completed
	// 2. Overall status is rejected
	// 3. Manual review required but not completed
	// 4. Risk level is critical
	if screening.ScreeningStatus != "completed" {
		return true
	}
	if screening.OverallStatus == "rejected" {
		return true
	}
	if screening.ManualReviewRequired && screening.ScreeningStatus != "completed" {
		return true
	}
	if screening.RiskLevel == "critical" && screening.OverallStatus != "clear" {
		return true
	}
	return false
}

// GetAMLComplianceFlag returns human-readable compliance status
func (s *AMLScreeningService) GetAMLComplianceFlag(screening *AMLScreeningResult) string {
	if screening.OverallStatus == "rejected" {
		return "COMPLIANCE_FAILURE"
	}
	if screening.RiskLevel == "critical" {
		return "REQUIRES_ESCALATION"
	}
	if screening.ManualReviewRequired && screening.ScreeningStatus == "pending" {
		return "PENDING_MANUAL_REVIEW"
	}
	if screening.OverallStatus == "clear" {
		return "COMPLIANCE_PASS"
	}
	return "UNKNOWN"
}
