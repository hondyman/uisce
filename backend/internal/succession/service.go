package succession

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
	Mutate(ctx context.Context, mutation string, variables map[string]interface{}, result interface{}) error
}

type Service struct {
	DB           *sqlx.DB
	hasuraClient HasuraClient
}

func NewService(db *sqlx.DB) *Service {
	return &Service{DB: db}
}

// NewServiceWithHasura creates a new succession service with Hasura support
func NewServiceWithHasura(db *sqlx.DB, hasuraClient HasuraClient) *Service {
	return &Service{DB: db, hasuraClient: hasuraClient}
}

// CalculatePracticeMetrics calculates practice metrics and valuation for an advisor
func (s *Service) CalculatePracticeMetrics(ctx context.Context, advisorID uuid.UUID) (*AdvisorPracticeMetrics, error) {
	// Mock implementation - in production, this would aggregate from client/revenue data
	metrics := &AdvisorPracticeMetrics{
		AdvisorID:           advisorID,
		EvaluationDate:      time.Now(),
		TotalAUM:            25000000, // $25M
		ClientCount:         50,
		AvgClientAge:        62.5,
		AvgAccountSize:      500000,
		Trailing12MoRevenue: 250000,
		RevenueGrowthRate:   0.08,
		ClientRetentionRate: 0.95,
		Top10ClientsAUMPct:  0.42,
		ValuationMultiple:   2.5,
		EstimatedValuation:  625000, // 2.5x revenue
		SuccessionReadiness: 65,
		KeyPersonDependency: 75,
		HasServiceManual:    true,
		HasInvestmentDoc:    true,
		CRMHygieneScore:     0.85,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// Save to database
	err := s.savePracticeMetricsRecord(ctx, metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to save metrics: %w", err)
	}

	return metrics, nil
}

// RecommendSuccessor finds the best successor match for a departing advisor
func (s *Service) RecommendSuccessor(ctx context.Context, departingAdvisorID uuid.UUID) ([]SuccessorCompatibility, error) {
	// Get departing advisor's book profile (mock for now)
	// In production, this would aggregate client demographics, service style, etc.

	// Mock candidate advisors
	candidates := []uuid.UUID{
		uuid.New(), // Mock candidate 1
		uuid.New(), // Mock candidate 2
	}

	var recommendations []SuccessorCompatibility

	for _, candidateID := range candidates {
		// Calculate compatibility scores
		score := s.calculateCompatibilityScore(departingAdvisorID, candidateID)
		recommendations = append(recommendations, score)

		// Save to database
		err := s.saveCompatibilityScoreRecord(ctx, score)
		if err != nil {
			return nil, fmt.Errorf("failed to save compatibility score: %w", err)
		}
	}

	return recommendations, nil
}

// calculateCompatibilityScore computes compatibility between advisors (simplified)
func (s *Service) calculateCompatibilityScore(departingID, candidateID uuid.UUID) SuccessorCompatibility {
	// Mock scoring logic - in production, use ML model
	clientMatch := 0.85
	styleMatch := 0.90
	specializationOverlap := 0.75
	capacityMatch := 0.80
	geoMatch := 0.95

	// Weighted average
	overallScore := (clientMatch*0.3 + styleMatch*0.25 + specializationOverlap*0.2 +
		capacityMatch*0.15 + geoMatch*0.10)

	reasoning := fmt.Sprintf("Strong client demographic match (%.0f%%), excellent service style compatibility (%.0f%%), and sufficient capacity",
		clientMatch*100, styleMatch*100)

	return SuccessorCompatibility{
		ScoreID:                uuid.New(),
		DepartingAdvisorID:     departingID,
		CandidateAdvisorID:     candidateID,
		ClientDemographicMatch: clientMatch,
		ServiceStyleMatch:      styleMatch,
		SpecializationOverlap:  specializationOverlap,
		CapacityMatch:          capacityMatch,
		GeographicMatch:        geoMatch,
		OverallScore:           overallScore,
		Reasoning:              &reasoning,
		CalculatedAt:           time.Now(),
	}
}

// CreateSuccessionPlan creates a new succession plan
func (s *Service) CreateSuccessionPlan(ctx context.Context, plan *SuccessionPlan) error {
	if plan.PlanID == uuid.Nil {
		plan.PlanID = uuid.New()
	}
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = time.Now()

	err := s.createSuccessionPlanRecord(ctx, plan)
	if err != nil {
		return fmt.Errorf("failed to create succession plan: %w", err)
	}

	return nil
}

// GetAdvisorPlans retrieves succession plans for an advisor
func (s *Service) GetAdvisorPlans(ctx context.Context, advisorID uuid.UUID) ([]SuccessionPlan, error) {
	plans, err := s.getAdvisorPlansRecords(ctx, advisorID)
	if err != nil {
		return nil, fmt.Errorf("failed to get succession plans: %w", err)
	}

	return plans, nil
}

// Helper methods for SQL operations with Hasura fallback

// savePracticeMetricsRecord inserts or updates advisor practice metrics
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: NamedExec INSERT with ON CONFLICT upsert for 19 fields
func (s *Service) savePracticeMetricsRecord(ctx context.Context, metrics *AdvisorPracticeMetrics) error {
	query := `
		INSERT INTO advisor_practice_metrics (
advisor_id, evaluation_date, total_aum, client_count, average_client_age,
average_account_size, trailing_12mo_revenue, revenue_growth_rate, client_retention_rate,
top_10_clients_aum_pct, estimated_valuation, valuation_multiple, succession_readiness_score,
key_person_dependency_score, has_client_service_manual, has_investment_philosophy_doc,
crm_hygiene_score, created_at, updated_at
) VALUES (
:advisor_id, :evaluation_date, :total_aum, :client_count, :average_client_age,
:average_account_size, :trailing_12mo_revenue, :revenue_growth_rate, :client_retention_rate,
:top_10_clients_aum_pct, :estimated_valuation, :valuation_multiple, :succession_readiness_score,
:key_person_dependency_score, :has_client_service_manual, :has_investment_philosophy_doc,
:crm_hygiene_score, :created_at, :updated_at
)
		ON CONFLICT (advisor_id) DO UPDATE SET
			evaluation_date = EXCLUDED.evaluation_date,
			total_aum = EXCLUDED.total_aum,
			updated_at = NOW()`

	_, err := s.DB.NamedExecContext(ctx, query, metrics)
	return err
}

// saveCompatibilityScoreRecord inserts a successor compatibility score
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: NamedExec INSERT for 11 compatibility score fields
func (s *Service) saveCompatibilityScoreRecord(ctx context.Context, score SuccessorCompatibility) error {
	query := `
		INSERT INTO successor_compatibility_scores (
score_id, departing_advisor_id, candidate_advisor_id,
client_demographic_match, service_style_match, specialization_overlap,
capacity_match, geographic_match, overall_compatibility_score,
reasoning, calculated_at
) VALUES (
:score_id, :departing_advisor_id, :candidate_advisor_id,
:client_demographic_match, :service_style_match, :specialization_overlap,
:capacity_match, :geographic_match, :overall_compatibility_score,
:reasoning, :calculated_at
)`

	_, err := s.DB.NamedExecContext(ctx, query, score)
	return err
}

// createSuccessionPlanRecord inserts a new succession plan
// TODO: Implement Hasura GraphQL mutation
// SQL fallback: NamedExec INSERT for 15 succession plan fields
func (s *Service) createSuccessionPlanRecord(ctx context.Context, plan *SuccessionPlan) error {
	query := `
		INSERT INTO succession_plans (
plan_id, departing_advisor_id, successor_advisor_id, plan_type,
target_transition_date, transition_period_months, revenue_split_structure,
clients_to_transition, transition_complete, purchase_price, payment_terms,
earnout_structure, status, created_at, updated_at
) VALUES (
:plan_id, :departing_advisor_id, :successor_advisor_id, :plan_type,
:target_transition_date, :transition_period_months, :revenue_split_structure,
:clients_to_transition, :transition_complete, :purchase_price, :payment_terms,
:earnout_structure, :status, :created_at, :updated_at
)`

	_, err := s.DB.NamedExecContext(ctx, query, plan)
	return err
}

// getAdvisorPlansRecords retrieves succession plans for an advisor
// TODO: Replace SQL with Hasura GraphQL query:
//
//	query GetAdvisorPlans($advisorId: uuid!) {
//	  succession_plans(where: {
//	    _or: [{departing_advisor_id: {_eq: $advisorId}}, {successor_advisor_id: {_eq: $advisorId}}]
//	  }, order_by: {created_at: desc}) {
//	    plan_id departing_advisor_id successor_advisor_id plan_type
//	    target_transition_date status purchase_price
//	  }
//	}
//
// Use: http://localhost:8080/v1/graphql with header X-Hasura-Admin-Secret: newadminsecretkey
func (s *Service) getAdvisorPlansRecords(ctx context.Context, advisorID uuid.UUID) ([]SuccessionPlan, error) {
	var plans []SuccessionPlan
	query := `
		SELECT * FROM succession_plans 
		WHERE departing_advisor_id = $1 OR successor_advisor_id = $1
		ORDER BY created_at DESC
	`
	err := s.DB.SelectContext(ctx, &plans, query, advisorID)
	return plans, err
}
