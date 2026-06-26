package billing

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

type Service struct {
	DB     *sqlx.DB
	hasura HasuraClient
}

func NewService(db *sqlx.DB) *Service {
	return &Service{DB: db}
}

func NewServiceWithHasura(db *sqlx.DB, hasura HasuraClient) *Service {
	return &Service{DB: db, hasura: hasura}
}

// CreateFeeSchedule creates a new fee schedule
func (s *Service) CreateFeeSchedule(ctx context.Context, schedule *FeeSchedule) error {
	if schedule.ScheduleID == uuid.Nil {
		schedule.ScheduleID = uuid.New()
	}
	schedule.CreatedAt = time.Now()
	schedule.UpdatedAt = time.Now()

	err := s.createFeeScheduleRecord(ctx, schedule)
	if err != nil {
		return fmt.Errorf("failed to create fee schedule: %w", err)
	}
	return nil
}

// AssignFeeSchedule assigns a fee schedule to a client
func (s *Service) AssignFeeSchedule(ctx context.Context, assignment *ClientFeeAssignment) error {
	if assignment.AssignmentID == uuid.Nil {
		assignment.AssignmentID = uuid.New()
	}
	assignment.CreatedAt = time.Now()
	assignment.UpdatedAt = time.Now()

	err := s.assignFeeScheduleRecord(ctx, assignment)
	if err != nil {
		return fmt.Errorf("failed to assign fee schedule: %w", err)
	}
	return nil
}

// CalculateClientFee calculates the fee for a client for a given period
func (s *Service) CalculateClientFee(ctx context.Context, clientID uuid.UUID, periodStart, periodEnd time.Time) (*FeeCalculation, error) {
	// 1. Get active assignment
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetActiveFeeAssignment($clientId: uuid!, $start: timestamptz!, $end: timestamptz!) {
	//   client_fee_assignments(where: {
	//     client_id: {_eq: $clientId},
	//     effective_date: {_lte: $end},
	//     _or: [{end_date: {_is_null: true}}, {end_date: {_gte: $start}}]
	//   }, order_by: {effective_date: desc}, limit: 1) {
	//     assignment_id
	//     client_id
	//     schedule_id
	//     effective_date
	//     end_date
	//     custom_discount_pct
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var assignment ClientFeeAssignment
	err := s.DB.GetContext(ctx, &assignment, `
		SELECT * FROM client_fee_assignments 
		WHERE client_id = $1 
		  AND effective_date <= $2 
		  AND (end_date IS NULL OR end_date >= $3)
		ORDER BY effective_date DESC LIMIT 1
	`, clientID, periodEnd, periodStart)
	if err != nil {
		return nil, fmt.Errorf("no active fee assignment found: %w", err)
	}

	// 2. Get schedule
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetFeeSchedule($scheduleId: uuid!) {
	//   fee_schedules_by_pk(schedule_id: $scheduleId) {
	//     schedule_id
	//     schedule_name
	//     fee_type
	//     tier_structure
	//     flat_fee_amount
	//     min_fee
	//     max_fee
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var schedule FeeSchedule
	err = s.DB.GetContext(ctx, &schedule, "SELECT * FROM fee_schedules WHERE schedule_id = $1", assignment.ScheduleID)
	if err != nil {
		return nil, fmt.Errorf("fee schedule not found: %w", err)
	}

	// 3. Calculate Average Daily Balance (Mocked for now)
	// In real system, query daily_balances table
	adb := 1000000.0 // Mock $1M

	// 4. Calculate AUM Fee
	aumFee := 0.0
	if schedule.FeeType == FeeTypeAUMTiered || schedule.FeeType == FeeTypeHybrid {
		// Parse tiers
		type Tier struct {
			Min  float64 `json:"min"`
			Max  float64 `json:"max"`
			Rate float64 `json:"rate"`
		}
		var tiers []Tier
		if len(schedule.TierStructure) > 0 {
			if err := json.Unmarshal(schedule.TierStructure, &tiers); err != nil {
				return nil, fmt.Errorf("invalid tier structure: %w", err)
			}
		}

		remaining := adb
		for _, tier := range tiers {
			if remaining <= 0 {
				break
			}
			tierSize := tier.Max - tier.Min
			amountInTier := math.Min(remaining, tierSize)
			// Annual rate to period rate (assuming monthly for now)
			periodRate := tier.Rate / 12.0
			aumFee += amountInTier * periodRate
			remaining -= amountInTier
		}
	}

	// 5. Apply Discounts
	discount := 0.0
	if assignment.CustomDiscountPct != nil {
		discount = aumFee * (*assignment.CustomDiscountPct)
		aumFee -= discount
	}

	// 6. Construct Result
	calc := &FeeCalculation{
		CalculationID:   uuid.New(),
		ClientID:        clientID,
		PeriodStart:     periodStart,
		PeriodEnd:       periodEnd,
		AvgDailyBalance: adb,
		AUMFee:          aumFee,
		DiscountAmount:  discount,
		TotalFee:        aumFee, // + Performance Fee (omitted for brevity)
		Status:          "DRAFT",
		CreatedAt:       time.Now(),
	}

	// 7. Save Calculation
	err = s.saveFeeCalculationRecord(ctx, calc)
	if err != nil {
		return nil, fmt.Errorf("failed to save calculation: %w", err)
	}

	return calc, nil
}

// ============================================================================
// HASURA-FIRST HELPERS
// ============================================================================

// createFeeScheduleRecord inserts a fee schedule
// Hasura-first with SQL fallback
func (s *Service) createFeeScheduleRecord(ctx context.Context, schedule *FeeSchedule) error {
	// Note: Hasura implementation would need proper type mapping
	// Skipping Hasura for this complex structure with many fields
	// and using SQL fallback directly

	// SQL fallback
	query := `
		INSERT INTO fee_schedules (
			schedule_id, schedule_name, fee_type, tier_structure,
			performance_hurdle_rate, performance_fee_rate, high_water_mark_enabled,
			billing_frequency, billing_advance_or_arrears, minimum_quarterly_fee, minimum_annual_fee,
			exclude_cash_from_aum, exclude_alternatives_from_aum, created_at, updated_at
		) VALUES (
			:schedule_id, :schedule_name, :fee_type, :tier_structure,
			:performance_hurdle_rate, :performance_fee_rate, :high_water_mark_enabled,
			:billing_frequency, :billing_advance_or_arrears, :minimum_quarterly_fee, :minimum_annual_fee,
			:exclude_cash_from_aum, :exclude_alternatives_from_aum, :created_at, :updated_at
		)`

	_, err := s.DB.NamedExecContext(ctx, query, schedule)
	return err
}

// assignFeeScheduleRecord assigns a fee schedule to a client
// Hasura-first with SQL fallback
func (s *Service) assignFeeScheduleRecord(ctx context.Context, assignment *ClientFeeAssignment) error {
	// Note: Hasura implementation would need proper type mapping
	// Skipping Hasura for this complex structure with many fields
	// and using SQL fallback directly

	// SQL fallback
	query := `
		INSERT INTO client_fee_assignments (
			assignment_id, client_id, account_id, schedule_id, effective_date, end_date,
			custom_discount_pct, custom_minimum_fee, invoice_contact_email,
			payment_method, billing_day_of_month, created_at, updated_at
		) VALUES (
			:assignment_id, :client_id, :account_id, :schedule_id, :effective_date, :end_date,
			:custom_discount_pct, :custom_minimum_fee, :invoice_contact_email,
			:payment_method, :billing_day_of_month, :created_at, :updated_at
		)`

	_, err := s.DB.NamedExecContext(ctx, query, assignment)
	return err
}

// saveFeeCalculationRecord saves a fee calculation
// Hasura-first with SQL fallback
func (s *Service) saveFeeCalculationRecord(ctx context.Context, calc *FeeCalculation) error {
	// Note: Hasura implementation would need proper type mapping
	// Skipping Hasura for this complex structure with many fields
	// and using SQL fallback directly

	// SQL fallback
	query := `
		INSERT INTO fee_calculations (
			calculation_id, client_id, billing_period_start, billing_period_end,
			average_daily_balance, aum_based_fee, performance_fee,
			prior_period_adjustment, discount_amount, minimum_fee_adjustment,
			total_fee, calculation_status, created_at
		) VALUES (
			:calculation_id, :client_id, :billing_period_start, :billing_period_end,
			:average_daily_balance, :aum_based_fee, :performance_fee,
			:prior_period_adjustment, :discount_amount, :minimum_fee_adjustment,
			:total_fee, :calculation_status, :created_at
		)`

	_, err := s.DB.NamedExecContext(ctx, query, calc)
	return err
}
