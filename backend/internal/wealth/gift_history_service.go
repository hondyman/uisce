package wealth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/shopspring/decimal"
)

// GiftHistoryService handles gift tracking and exemption management
type GiftHistoryService struct {
	db      *pgxpool.Pool
	taxCalc *TaxCalculationService
}

// NewGiftHistoryService creates a new gift history service
func NewGiftHistoryService(db *pgxpool.Pool, taxCalc *TaxCalculationService) *GiftHistoryService {
	return &GiftHistoryService{
		db:      db,
		taxCalc: taxCalc,
	}
}

// RecordGiftInput represents input for recording a gift
type RecordGiftInput struct {
	FamilyID             string          `json:"family_id"`
	DonorMemberID        string          `json:"donor_member_id"`
	RecipientMemberID    *string         `json:"recipient_member_id,omitempty"`
	RecipientEntityID    *string         `json:"recipient_entity_id,omitempty"`
	GiftDate             time.Time       `json:"gift_date"`
	GiftType             string          `json:"gift_type"`
	AssetID              *string         `json:"asset_id,omitempty"`
	AssetDescription     string          `json:"asset_description"`
	FairMarketValue      decimal.Decimal `json:"fair_market_value"`
	ValuationMethod      string          `json:"valuation_method"`
	ValuationDiscountPct decimal.Decimal `json:"valuation_discount_pct"`
	SpousalSplitElection bool            `json:"spousal_split_election"`
	SpouseMemberID       *string         `json:"spouse_member_id,omitempty"`
	IsGenerationSkipping bool            `json:"is_generation_skipping"`
	GenerationSkipCount  *int            `json:"generation_skip_count,omitempty"`
	GiftPurpose          *string         `json:"gift_purpose,omitempty"`
	CreatedBy            *string         `json:"created_by,omitempty"`
}

// RecordGift records a new gift and calculates exemption usage
func (s *GiftHistoryService) RecordGift(ctx context.Context, input RecordGiftInput) (*GiftHistory, error) {
	giftID := uuid.New().String()
	now := time.Now()

	// Calculate net gift value after discount
	netGiftValue := input.FairMarketValue.Mul(
		decimal.NewFromInt(1).Sub(input.ValuationDiscountPct.Div(decimal.NewFromInt(100))),
	)

	// Get prior annual exclusion used this year for this donor
	annualExclusionUsed, err := s.getAnnualExclusionUsedThisYear(ctx, input.DonorMemberID, input.GiftDate.Year())
	if err != nil {
		return nil, err
	}

	// Get lifetime exemption used prior
	lifetimeExemptionUsed, err := s.getLifetimeExemptionUsed(ctx, input.FamilyID, input.DonorMemberID)
	if err != nil {
		return nil, err
	}

	// Calculate gift tax and exemption usage
	year := input.GiftDate.Year()
	giftTaxResult, err := s.taxCalc.CalculateGiftTax(
		ctx,
		netGiftValue,
		annualExclusionUsed,
		lifetimeExemptionUsed,
		input.SpousalSplitElection,
		&year,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate gift tax: %w", err)
	}

	// Calculate GST tax if generation-skipping
	var gstExemptionUsed decimal.Decimal
	if input.IsGenerationSkipping && input.GenerationSkipCount != nil && *input.GenerationSkipCount >= 2 {
		gstExemptionPrior, _ := s.getGSTExemptionUsed(ctx, input.FamilyID, input.DonorMemberID)
		year2 := input.GiftDate.Year()
		gstResult, err := s.taxCalc.CalculateGSTTax(
			ctx,
			netGiftValue,
			gstExemptionPrior,
			*input.GenerationSkipCount,
			&year2,
		)
		if err != nil {
			return nil, err
		}
		gstExemptionUsed = gstResult.GSTExemptionApplied
	}

	// Determine Form 709 due date (April 15 of following year)
	var form709DueDate *time.Time
	if giftTaxResult.RequiresForm709 {
		dueDate := time.Date(input.GiftDate.Year()+1, 4, 15, 0, 0, 0, 0, time.UTC)
		form709DueDate = &dueDate
	}

	// Insert gift record
	query := `
		INSERT INTO gift_history (
			gift_id, family_id, donor_member_id, recipient_member_id, recipient_entity_id,
			gift_date, gift_type, asset_id, asset_description,
			fair_market_value, valuation_method, valuation_discount_pct, net_gift_value,
			annual_exclusion_utilized, lifetime_exemption_utilized, gst_exemption_utilized,
			spousal_split_election, spouse_member_id,
			requires_gift_tax_return, form_709_due_date,
			is_generation_skipping, generation_skip_count,
			gift_purpose, created_at, updated_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
		RETURNING gift_id, family_id, donor_member_id, gift_date, net_gift_value,
			annual_exclusion_utilized, lifetime_exemption_utilized, requires_gift_tax_return
	`

	var gift GiftHistory
	err = s.db.QueryRow(ctx, query,
		giftID,
		input.FamilyID,
		input.DonorMemberID,
		input.RecipientMemberID,
		input.RecipientEntityID,
		input.GiftDate,
		input.GiftType,
		input.AssetID,
		input.AssetDescription,
		input.FairMarketValue,
		input.ValuationMethod,
		input.ValuationDiscountPct,
		netGiftValue,
		giftTaxResult.AnnualExclusionUtilized,
		giftTaxResult.LifetimeExemptionUtilized,
		gstExemptionUsed,
		input.SpousalSplitElection,
		input.SpouseMemberID,
		giftTaxResult.RequiresForm709,
		form709DueDate,
		input.IsGenerationSkipping,
		input.GenerationSkipCount,
		input.GiftPurpose,
		now,
		now,
		input.CreatedBy,
	).Scan(
		&gift.GiftID,
		&gift.FamilyID,
		&gift.DonorMemberID,
		&gift.GiftDate,
		&gift.NetGiftValue,
		&gift.AnnualExclusionUtilized,
		&gift.LifetimeExemptionUtilized,
		&gift.RequiresGiftTaxReturn,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to record gift: %w", err)
	}

	return &gift, nil
}

// GetGiftHistory retrieves gift history for a family
func (s *GiftHistoryService) GetGiftHistory(ctx context.Context, familyID string) ([]GiftHistory, error) {
	query := `
		SELECT gift_id, family_id, donor_member_id, recipient_member_id, recipient_entity_id,
			gift_date, gift_type, asset_description, fair_market_value, net_gift_value,
			annual_exclusion_utilized, lifetime_exemption_utilized, gst_exemption_utilized,
			spousal_split_election, requires_gift_tax_return, form_709_filed, form_709_due_date,
			is_generation_skipping, generation_skip_count, created_at
		FROM gift_history
		WHERE family_id = $1 AND deleted_at IS NULL
		ORDER BY gift_date DESC
	`

	rows, err := s.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, fmt.Errorf("failed to query gift history: %w", err)
	}
	defer rows.Close()

	gifts := []GiftHistory{}
	for rows.Next() {
		var g GiftHistory
		err := rows.Scan(
			&g.GiftID,
			&g.FamilyID,
			&g.DonorMemberID,
			&g.RecipientMemberID,
			&g.RecipientEntityID,
			&g.GiftDate,
			&g.GiftType,
			&g.AssetDescription,
			&g.FairMarketValue,
			&g.NetGiftValue,
			&g.AnnualExclusionUtilized,
			&g.LifetimeExemptionUtilized,
			&g.GSTExemptionUtilized,
			&g.SpousalSplitElection,
			&g.RequiresGiftTaxReturn,
			&g.Form709Filed,
			&g.Form709DueDate,
			&g.IsGenerationSkipping,
			&g.GenerationSkipCount,
			&g.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		gifts = append(gifts, g)
	}

	return gifts, nil
}

// GetPendingForm709Filings retrieves gifts requiring Form 709 not yet filed
func (s *GiftHistoryService) GetPendingForm709Filings(ctx context.Context, familyID string) ([]GiftHistory, error) {
	query := `
		SELECT gift_id, family_id, donor_member_id, gift_date, net_gift_value,
			lifetime_exemption_utilized, form_709_due_date
		FROM gift_history
		WHERE family_id = $1
			AND requires_gift_tax_return = TRUE
			AND form_709_filed = FALSE
			AND deleted_at IS NULL
		ORDER BY form_709_due_date ASC
	`

	rows, err := s.db.Query(ctx, query, familyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	gifts := []GiftHistory{}
	for rows.Next() {
		var g GiftHistory
		err := rows.Scan(
			&g.GiftID,
			&g.FamilyID,
			&g.DonorMemberID,
			&g.GiftDate,
			&g.NetGiftValue,
			&g.LifetimeExemptionUtilized,
			&g.Form709DueDate,
		)
		if err != nil {
			return nil, err
		}
		gifts = append(gifts, g)
	}

	return gifts, nil
}

// MarkForm709Filed marks a gift's Form 709 as filed
func (s *GiftHistoryService) MarkForm709Filed(ctx context.Context, giftID string, filingDate time.Time, documentID *string) error {
	query := `
		UPDATE gift_history
		SET form_709_filed = TRUE,
			form_709_filing_date = $1,
			form_709_document_id = $2,
			updated_at = NOW()
		WHERE gift_id = $3
	`

	_, err := s.db.Exec(ctx, query, filingDate, documentID, giftID)
	if err != nil {
		return fmt.Errorf("failed to mark Form 709 filed: %w", err)
	}

	return nil
}

// GetExemptionSummary calculates exemption usage for a donor
func (s *GiftHistoryService) GetExemptionSummary(ctx context.Context, familyID string, memberID string) (*ExemptionSummary, error) {
	// Get current year's annual exclusion usage
	currentYear := time.Now().Year()
	annualUsed, err := s.getAnnualExclusionUsedThisYear(ctx, memberID, currentYear)
	if err != nil {
		return nil, err
	}

	// Get lifetime exemption used
	lifetimeUsed, err := s.getLifetimeExemptionUsed(ctx, familyID, memberID)
	if err != nil {
		return nil, err
	}

	// Get GST exemption used
	gstUsed, err := s.getGSTExemptionUsed(ctx, familyID, memberID)
	if err != nil {
		return nil, err
	}

	// Get current exemption limits from tax jurisdiction
	juris, err := s.taxCalc.getTaxJurisdiction(ctx, "US-FEDERAL", &currentYear)
	if err != nil {
		return nil, err
	}

	annualRemaining := juris.AnnualGiftExclusion.Sub(annualUsed)
	lifetimeRemaining := juris.LifetimeGiftExemption.Sub(lifetimeUsed)
	gstRemaining := juris.GSTTaxExemption.Sub(gstUsed)

	return &ExemptionSummary{
		AnnualExclusionUsed:        annualUsed,
		AnnualExclusionRemaining:   annualRemaining,
		AnnualExclusionLimit:       *juris.AnnualGiftExclusion,
		LifetimeExemptionUsed:      lifetimeUsed,
		LifetimeExemptionRemaining: lifetimeRemaining,
		LifetimeExemptionLimit:     *juris.LifetimeGiftExemption,
		GSTExemptionUsed:           gstUsed,
		GSTExemptionRemaining:      gstRemaining,
		GSTExemptionLimit:          *juris.GSTTaxExemption,
	}, nil
}

// ExemptionSummary represents current exemption usage
type ExemptionSummary struct {
	AnnualExclusionUsed        decimal.Decimal `json:"annual_exclusion_used"`
	AnnualExclusionRemaining   decimal.Decimal `json:"annual_exclusion_remaining"`
	AnnualExclusionLimit       decimal.Decimal `json:"annual_exclusion_limit"`
	LifetimeExemptionUsed      decimal.Decimal `json:"lifetime_exemption_used"`
	LifetimeExemptionRemaining decimal.Decimal `json:"lifetime_exemption_remaining"`
	LifetimeExemptionLimit     decimal.Decimal `json:"lifetime_exemption_limit"`
	GSTExemptionUsed           decimal.Decimal `json:"gst_exemption_used"`
	GSTExemptionRemaining      decimal.Decimal `json:"gst_exemption_remaining"`
	GSTExemptionLimit          decimal.Decimal `json:"gst_exemption_limit"`
}

// Helper functions

func (s *GiftHistoryService) getAnnualExclusionUsedThisYear(ctx context.Context, memberID string, year int) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(annual_exclusion_utilized), 0)
		FROM gift_history
		WHERE donor_member_id = $1
			AND EXTRACT(YEAR FROM gift_date) = $2
			AND deleted_at IS NULL
	`

	var used decimal.Decimal
	err := s.db.QueryRow(ctx, query, memberID, year).Scan(&used)
	if err != nil {
		return decimal.Zero, err
	}

	return used, nil
}

func (s *GiftHistoryService) getLifetimeExemptionUsed(ctx context.Context, familyID string, memberID string) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(lifetime_exemption_utilized), 0)
		FROM gift_history
		WHERE family_id = $1
			AND donor_member_id = $2
			AND deleted_at IS NULL
	`

	var used decimal.Decimal
	err := s.db.QueryRow(ctx, query, familyID, memberID).Scan(&used)
	if err != nil {
		return decimal.Zero, err
	}

	return used, nil
}

func (s *GiftHistoryService) getGSTExemptionUsed(ctx context.Context, familyID string, memberID string) (decimal.Decimal, error) {
	query := `
		SELECT COALESCE(SUM(gst_exemption_utilized), 0)
		FROM gift_history
		WHERE family_id = $1
			AND donor_member_id = $2
			AND deleted_at IS NULL
	`

	var used decimal.Decimal
	err := s.db.QueryRow(ctx, query, familyID, memberID).Scan(&used)
	if err != nil {
		return decimal.Zero, err
	}

	return used, nil
}
