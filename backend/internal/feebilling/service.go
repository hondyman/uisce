package feebilling

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

// Service provides fee billing operations
type Service interface {
	// Fee Schedules
	CreateFeeSchedule(ctx context.Context, input CreateFeeScheduleInput) (*FeeSchedule, error)
	GetFeeSchedule(ctx context.Context, scheduleID uuid.UUID) (*FeeSchedule, error)
	ListFeeSchedules(ctx context.Context, activeOnly bool) ([]*FeeSchedule, error)

	// Client Assignments
	AssignFeeSchedule(ctx context.Context, input AssignFeeScheduleInput) (*ClientFeeAssignment, error)
	GetClientAssignment(ctx context.Context, clientID uuid.UUID) (*ClientFeeAssignment, error)

	// Fee Calculations
	CalculateFees(ctx context.Context, clientID uuid.UUID, periodStart, periodEnd time.Time) (*FeeCalculation, error)
	ApproveFeeCalculation(ctx context.Context, calculationID uuid.UUID, approvedBy uuid.UUID) error
	ListPendingApprovals(ctx context.Context) ([]*FeeCalculation, error)

	// High Water Marks
	GetHighWaterMark(ctx context.Context, clientID uuid.UUID, accountID *uuid.UUID) (*HighWaterMark, error)
	UpdateHighWaterMark(ctx context.Context, clientID uuid.UUID, accountID *uuid.UUID, newValue float64) error
}

type service struct {
	db           *sqlx.DB
	hasuraClient HasuraClient
}

func NewService(db *sqlx.DB) Service {
	return &service{db: db}
}

// NewServiceWithHasura creates a new fee billing service with Hasura support
func NewServiceWithHasura(db *sqlx.DB, hasuraClient HasuraClient) Service {
	return &service{db: db, hasuraClient: hasuraClient}
}

type CreateFeeScheduleInput struct {
	ScheduleName               string
	Description                *string
	FeeType                    FeeType
	TierStructure              []FeeTier
	FlatAUMRate                *float64
	PerformanceHurdleRate      *float64
	PerformanceFeeRate         *float64
	HighWaterMarkEnabled       bool
	BillingFrequency           BillingFrequency
	MinimumQuarterlyFee        *float64
	ExcludeCashFromAUM         bool
	ExcludeAlternativesFromAUM bool
}

func (s *service) CreateFeeSchedule(ctx context.Context, input CreateFeeScheduleInput) (*FeeSchedule, error) {
	tierJSON, _ := json.Marshal(input.TierStructure)

	schedule := &FeeSchedule{
		ScheduleID:                 uuid.New(),
		ScheduleName:               input.ScheduleName,
		Description:                input.Description,
		FeeType:                    input.FeeType,
		TierStructure:              tierJSON,
		FlatAUMRate:                input.FlatAUMRate,
		PerformanceHurdleRate:      input.PerformanceHurdleRate,
		PerformanceFeeRate:         input.PerformanceFeeRate,
		HighWaterMarkEnabled:       input.HighWaterMarkEnabled,
		BillingFrequency:           input.BillingFrequency,
		MinimumQuarterlyFee:        input.MinimumQuarterlyFee,
		ExcludeCashFromAUM:         input.ExcludeCashFromAUM,
		ExcludeAlternativesFromAUM: input.ExcludeAlternativesFromAUM,
		IsActive:                   true,
		CreatedAt:                  time.Now(),
		UpdatedAt:                  time.Now(),
	}

	err := s.createFeeScheduleRecord(ctx, schedule)
	if err != nil {
		return nil, fmt.Errorf("failed to create fee schedule: %w", err)
	}

	return schedule, nil
}

func (s *service) GetFeeSchedule(ctx context.Context, scheduleID uuid.UUID) (*FeeSchedule, error) {
	schedule, err := s.getFeeScheduleRecord(ctx, scheduleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("fee schedule not found: %s", scheduleID)
		}
		return nil, err
	}
	return schedule, nil
}

func (s *service) ListFeeSchedules(ctx context.Context, activeOnly bool) ([]*FeeSchedule, error) {
	return s.listFeeSchedulesRecords(ctx, activeOnly)
}

type AssignFeeScheduleInput struct {
	ClientID          uuid.UUID
	AccountID         *uuid.UUID
	ScheduleID        uuid.UUID
	EffectiveDate     time.Time
	CustomDiscountPct *float64
}

func (s *service) AssignFeeSchedule(ctx context.Context, input AssignFeeScheduleInput) (*ClientFeeAssignment, error) {
	assignment := &ClientFeeAssignment{
		AssignmentID:      uuid.New(),
		ClientID:          input.ClientID,
		AccountID:         input.AccountID,
		ScheduleID:        input.ScheduleID,
		EffectiveDate:     input.EffectiveDate,
		CustomDiscountPct: input.CustomDiscountPct,
		IsActive:          true,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := s.assignFeeScheduleRecord(ctx, assignment)
	if err != nil {
		return nil, fmt.Errorf("failed to assign fee schedule: %w", err)
	}

	return assignment, nil
}

func (s *service) GetClientAssignment(ctx context.Context, clientID uuid.UUID) (*ClientFeeAssignment, error) {
	assignment, err := s.getClientAssignmentRecord(ctx, clientID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active fee assignment for client: %s", clientID)
		}
		return nil, err
	}
	return assignment, nil
}

// CalculateFees calculates fees for a client for a given period
func (s *service) CalculateFees(ctx context.Context, clientID uuid.UUID, periodStart, periodEnd time.Time) (*FeeCalculation, error) {
	// Get client's fee assignment
	assignment, err := s.GetClientAssignment(ctx, clientID)
	if err != nil {
		return nil, err
	}

	// Get fee schedule
	schedule, err := s.GetFeeSchedule(ctx, assignment.ScheduleID)
	if err != nil {
		return nil, err
	}

	// Calculate average daily balance (simplified - would query portfolio holdings in production)
	adb := s.calculateAverageDailyBalance(ctx, clientID, periodStart, periodEnd)

	calc := &FeeCalculation{
		CalculationID:        uuid.New(),
		ClientID:             clientID,
		AssignmentID:         &assignment.AssignmentID,
		BillingPeriodStart:   periodStart,
		BillingPeriodEnd:     periodEnd,
		BillingFrequency:     schedule.BillingFrequency,
		AverageDailyBalance:  adb,
		AUMCalculationMethod: "AVERAGE_DAILY",
		CalculationStatus:    StatusDraft,
		CreatedAt:            time.Now(),
		UpdatedAt:            time.Now(),
	}

	// Calculate AUM-based fee
	if schedule.FeeType == FeeTypeAUMTiered || schedule.FeeType == FeeTypeHybrid {
		var tiers []FeeTier
		json.Unmarshal(schedule.TierStructure, &tiers)

		aumFee := 0.0
		for _, tier := range tiers {
			if adb > tier.Min {
				tierAmount := math.Min(adb-tier.Min, tier.Max-tier.Min)
				aumFee += tierAmount * tier.Rate
			}
		}
		calc.AUMBasedFee = aumFee
	} else if schedule.FeeType == FeeTypeAUMFlat && schedule.FlatAUMRate != nil {
		calc.AUMBasedFee = adb * *schedule.FlatAUMRate
	}

	// Apply minimum fee if applicable
	if schedule.MinimumQuarterlyFee != nil && calc.AUMBasedFee < *schedule.MinimumQuarterlyFee {
		calc.MinimumFeeAdjustment = *schedule.MinimumQuarterlyFee - calc.AUMBasedFee
		calc.AUMBasedFee = *schedule.MinimumQuarterlyFee
	}

	// Apply discount if applicable
	if assignment.CustomDiscountPct != nil {
		calc.DiscountAmount = calc.AUMBasedFee * (*assignment.CustomDiscountPct / 100)
	}

	// Calculate totals
	calc.GrossFee = calc.AUMBasedFee + calc.PerformanceFee + calc.PlanningFee + calc.HourlyFees + calc.OtherFees
	calc.NetFee = calc.GrossFee - calc.DiscountAmount + calc.PriorPeriodAdjustment
	calc.TaxableAmount = calc.NetFee

	// Save calculation
	err = s.saveFeeCalculationRecord(ctx, calc)
	if err != nil {
		return nil, fmt.Errorf("failed to save fee calculation: %w", err)
	}

	return calc, nil
}

func (s *service) calculateAverageDailyBalance(ctx context.Context, clientID uuid.UUID, start, end time.Time) float64 {
	// Simplified - in production would query daily portfolio values
	return 5000000.0 // $5M placeholder
}

func (s *service) ApproveFeeCalculation(ctx context.Context, calculationID uuid.UUID, approvedBy uuid.UUID) error {
	return s.approveFeeCalculationRecord(ctx, calculationID, approvedBy)
}

func (s *service) ListPendingApprovals(ctx context.Context) ([]*FeeCalculation, error) {
	return s.listPendingApprovalsRecords(ctx)
}

func (s *service) GetHighWaterMark(ctx context.Context, clientID uuid.UUID, accountID *uuid.UUID) (*HighWaterMark, error) {
	return s.getHighWaterMarkRecord(ctx, clientID, accountID)
}

func (s *service) UpdateHighWaterMark(ctx context.Context, clientID uuid.UUID, accountID *uuid.UUID, newValue float64) error {
	existing, _ := s.GetHighWaterMark(ctx, clientID, accountID)

	if existing == nil {
		// Create new
		hwm := &HighWaterMark{
			HWMID:                uuid.New(),
			ClientID:             clientID,
			AccountID:            accountID,
			CurrentHighWaterMark: newValue,
			HWMDate:              time.Now(),
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
		}
		return s.createHighWaterMarkRecord(ctx, hwm)
	}

	// Update if new value exceeds current HWM
	if newValue > existing.CurrentHighWaterMark {
		return s.updateHighWaterMarkRecord(ctx, existing.HWMID, newValue)
	}

	return nil
}

// Helper methods for Hasura integration - SQL fallback for complex operations

func (s *service) createFeeScheduleRecord(ctx context.Context, schedule *FeeSchedule) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for complex struct with JSON fields
	query := `
		INSERT INTO fee_schedules (
schedule_id, schedule_name, fee_type, billing_frequency,
tiers, performance_fee_config, minimum_fee, calculation_rules,
created_by, is_active, created_at, updated_at
) VALUES (
:schedule_id, :schedule_name, :fee_type, :billing_frequency,
:tiers, :performance_fee_config, :minimum_fee, :calculation_rules,
:created_by, :is_active, :created_at, :updated_at
)
	`
	_, err := s.db.NamedExecContext(ctx, query, schedule)
	return err
}

func (s *service) getFeeScheduleRecord(ctx context.Context, scheduleID uuid.UUID) (*FeeSchedule, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetFeeSchedule($id: uuid!) {
	//   fee_schedules_by_pk(schedule_id: $id) {
	//     schedule_id schedule_name fee_type billing_frequency tiers
	//     performance_fee_config minimum_fee calculation_rules is_active
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var schedule FeeSchedule
	query := `SELECT * FROM fee_schedules WHERE schedule_id = $1`
	err := s.db.GetContext(ctx, &schedule, query, scheduleID)
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (s *service) listFeeSchedulesRecords(ctx context.Context, activeOnly bool) ([]*FeeSchedule, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query ListFeeSchedules($activeOnly: Boolean!) {
	//   fee_schedules(where: {is_active: {_eq: $activeOnly}}, order_by: {schedule_name: asc}) {
	//     schedule_id schedule_name fee_type billing_frequency is_active
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var schedules []*FeeSchedule
	query := `SELECT * FROM fee_schedules`
	if activeOnly {
		query += ` WHERE is_active = TRUE`
	}
	query += ` ORDER BY schedule_name`
	err := s.db.SelectContext(ctx, &schedules, query)
	return schedules, err
}

func (s *service) assignFeeScheduleRecord(ctx context.Context, assignment *ClientFeeAssignment) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for complex struct with JSONB fields
	query := `
		INSERT INTO client_fee_assignments (
assignment_id, client_id, schedule_id, effective_date, end_date,
custom_discount, custom_minimum, billing_day, is_active, assigned_by, created_at, updated_at
) VALUES (
:assignment_id, :client_id, :schedule_id, :effective_date, :end_date,
:custom_discount, :custom_minimum, :billing_day, :is_active, :assigned_by, :created_at, :updated_at
)
	`
	_, err := s.db.NamedExecContext(ctx, query, assignment)
	return err
}

func (s *service) getClientAssignmentRecord(ctx context.Context, clientID uuid.UUID) (*ClientFeeAssignment, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetActiveClientAssignment($clientId: uuid!, $now: timestamptz!) {
	//   client_fee_assignments(where: {
	//     client_id: {_eq: $clientId}, is_active: {_eq: true},
	//     effective_date: {_lte: $now},
	//     _or: [{end_date: {_is_null: true}}, {end_date: {_gte: $now}}]
	//   }, order_by: {effective_date: desc}, limit: 1) {
	//     assignment_id client_id schedule_id effective_date custom_discount
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var assignment ClientFeeAssignment
	query := `
		SELECT * FROM client_fee_assignments
		WHERE client_id = $1 AND is_active = TRUE
		  AND effective_date <= NOW()
		  AND (end_date IS NULL OR end_date >= NOW())
		ORDER BY effective_date DESC
		LIMIT 1
	`
	err := s.db.GetContext(ctx, &assignment, query, clientID)
	if err != nil {
		return nil, err
	}
	return &assignment, nil
}

func (s *service) saveFeeCalculationRecord(ctx context.Context, calc *FeeCalculation) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback for very complex struct with many fields
	query := `
		INSERT INTO fee_calculations (
calculation_id, client_id, assignment_id, billing_period_start, billing_period_end,
billing_frequency, average_daily_balance, aum_calculation_method, aum_based_fee,
performance_fee, planning_fee, hourly_fees, other_fees,
prior_period_adjustment, discount_amount, minimum_fee_adjustment,
gross_fee, net_fee, taxable_amount, calculation_status,
created_at, updated_at
) VALUES (
:calculation_id, :client_id, :assignment_id, :billing_period_start, :billing_period_end,
:billing_frequency, :average_daily_balance, :aum_calculation_method, :aum_based_fee,
:performance_fee, :planning_fee, :hourly_fees, :other_fees,
:prior_period_adjustment, :discount_amount, :minimum_fee_adjustment,
:gross_fee, :net_fee, :taxable_amount, :calculation_status,
:created_at, :updated_at
)
	`
	_, err := s.db.NamedExecContext(ctx, query, calc)
	return err
}

func (s *service) approveFeeCalculationRecord(ctx context.Context, calculationID uuid.UUID, approvedBy uuid.UUID) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation ApproveFeeCalculation($id: uuid!, $approvedBy: uuid!, $now: timestamptz!) {
	//   update_fee_calculations_by_pk(
	//     pk_columns: {calculation_id: $id},
	//     _set: {calculation_status: "APPROVED", approved_by: $approvedBy, approved_at: $now, updated_at: $now}
	//   ) { calculation_id calculation_status }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE fee_calculations
		SET calculation_status = $1, approved_by = $2, approved_at = $3, updated_at = $4
		WHERE calculation_id = $5
	`
	now := time.Now()
	_, err := s.db.ExecContext(ctx, query, StatusApproved, approvedBy, now, now, calculationID)
	return err
}

func (s *service) listPendingApprovalsRecords(ctx context.Context) ([]*FeeCalculation, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query ListPendingApprovals {
	//   fee_calc_pending_approval(order_by: {created_at: asc}) {
	//     calculation_id client_id gross_fee net_fee calculation_status created_at
	//   }
	// }
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var calcs []*FeeCalculation
	query := `SELECT * FROM fee_calc_pending_approval ORDER BY created_at`
	err := s.db.SelectContext(ctx, &calcs, query)
	return calcs, err
}

func (s *service) getHighWaterMarkRecord(ctx context.Context, clientID uuid.UUID, accountID *uuid.UUID) (*HighWaterMark, error) {
	// TODO: Replace SQL with Hasura GraphQL query:
	// query GetHighWaterMark($clientId: uuid!, $accountId: uuid) {
	//   high_water_marks(where: {client_id: {_eq: $clientId}, account_id: {_eq: $accountId}}) {
	//     hwm_id current_high_water_mark previous_high_water_mark hwm_date
	//   }
	// }
	// Note: Hasura handles NULL equality with _eq operator
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	var hwm HighWaterMark
	query := `SELECT * FROM high_water_marks WHERE client_id = $1 AND account_id IS NOT DISTINCT FROM $2`
	err := s.db.GetContext(ctx, &hwm, query, clientID, accountID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if err == sql.ErrNoRows {
		return nil, nil // No high water mark yet
	}
	return &hwm, nil
}

func (s *service) createHighWaterMarkRecord(ctx context.Context, hwm *HighWaterMark) error {
	// TODO: Use HasuraClient for INSERT when available
	// For now, use SQL fallback
	query := `
		INSERT INTO high_water_marks (hwm_id, client_id, account_id, current_high_water_mark, hwm_date, created_at, updated_at)
		VALUES (:hwm_id, :client_id, :account_id, :current_high_water_mark, :hwm_date, :created_at, :updated_at)
	`
	_, err := s.db.NamedExecContext(ctx, query, hwm)
	return err
}

func (s *service) updateHighWaterMarkRecord(ctx context.Context, hwmID uuid.UUID, newValue float64) error {
	// TODO: Replace SQL with Hasura GraphQL mutation:
	// mutation UpdateHighWaterMark($id: uuid!, $newValue: numeric!, $now: timestamptz!) {
	//   update_high_water_marks_by_pk(
	//     pk_columns: {hwm_id: $id},
	//     _set: {previous_high_water_mark: current_high_water_mark, current_high_water_mark: $newValue, hwm_date: $now, updated_at: $now}
	//   ) { hwm_id current_high_water_mark }
	// }
	// Note: Need to fetch current value first or use database function for atomic swap
	// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
	query := `
		UPDATE high_water_marks
		SET previous_high_water_mark = current_high_water_mark,
		    current_high_water_mark = $1,
		    hwm_date = $2,
		    updated_at = $3
		WHERE hwm_id = $4
	`
	_, err := s.db.ExecContext(ctx, query, newValue, time.Now(), time.Now(), hwmID)
	return err
}
