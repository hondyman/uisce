package succession

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

type AdvisorPracticeMetrics struct {
	AdvisorID      uuid.UUID `db:"advisor_id" json:"advisor_id"`
	EvaluationDate time.Time `db:"evaluation_date" json:"evaluation_date"`

	TotalAUM       float64 `db:"total_aum" json:"total_aum"`
	ClientCount    int     `db:"client_count" json:"client_count"`
	AvgClientAge   float64 `db:"average_client_age" json:"average_client_age"`
	AvgAccountSize float64 `db:"average_account_size" json:"average_account_size"`

	Trailing12MoRevenue float64 `db:"trailing_12mo_revenue" json:"trailing_12mo_revenue"`
	RevenueGrowthRate   float64 `db:"revenue_growth_rate" json:"revenue_growth_rate"`
	ClientRetentionRate float64 `db:"client_retention_rate" json:"client_retention_rate"`

	Top10ClientsAUMPct float64 `db:"top_10_clients_aum_pct" json:"top_10_clients_aum_pct"`

	EstimatedValuation float64 `db:"estimated_valuation" json:"estimated_valuation"`
	ValuationMultiple  float64 `db:"valuation_multiple" json:"valuation_multiple"`

	SuccessionReadiness int `db:"succession_readiness_score" json:"succession_readiness_score"`
	KeyPersonDependency int `db:"key_person_dependency_score" json:"key_person_dependency_score"`

	HasServiceManual bool    `db:"has_client_service_manual" json:"has_client_service_manual"`
	HasInvestmentDoc bool    `db:"has_investment_philosophy_doc" json:"has_investment_philosophy_doc"`
	CRMHygieneScore  float64 `db:"crm_hygiene_score" json:"crm_hygiene_score"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type SuccessionPlan struct {
	PlanID             uuid.UUID  `db:"plan_id" json:"plan_id"`
	DepartingAdvisorID uuid.UUID  `db:"departing_advisor_id" json:"departing_advisor_id"`
	SuccessorAdvisorID *uuid.UUID `db:"successor_advisor_id" json:"successor_advisor_id,omitempty"`

	PlanType             string     `db:"plan_type" json:"plan_type"` // RETIREMENT, INTERNAL_PROMOTION, EXTERNAL_BUYER, EMERGENCY
	TargetTransitionDate *time.Time `db:"target_transition_date" json:"target_transition_date,omitempty"`

	TransitionPeriodMonths int            `db:"transition_period_months" json:"transition_period_months"`
	RevenueSplitStructure  types.JSONText `db:"revenue_split_structure" json:"revenue_split_structure,omitempty"`

	ClientsToTransition []uuid.UUID `db:"clients_to_transition" json:"clients_to_transition,omitempty"`
	TransitionComplete  bool        `db:"transition_complete" json:"transition_complete"`

	PurchasePrice    *float64       `db:"purchase_price" json:"purchase_price,omitempty"`
	PaymentTerms     *string        `db:"payment_terms" json:"payment_terms,omitempty"`
	EarnoutStructure types.JSONText `db:"earnout_structure" json:"earnout_structure,omitempty"`

	Status string `db:"status" json:"status"` // PLANNING, ANNOUNCED, IN_PROGRESS, COMPLETE, CANCELLED

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ClientTransition struct {
	TransitionID     uuid.UUID  `db:"transition_id" json:"transition_id"`
	ClientID         uuid.UUID  `db:"client_id" json:"client_id"`
	FromAdvisorID    uuid.UUID  `db:"from_advisor_id" json:"from_advisor_id"`
	ToAdvisorID      uuid.UUID  `db:"to_advisor_id" json:"to_advisor_id"`
	SuccessionPlanID *uuid.UUID `db:"succession_plan_id" json:"succession_plan_id,omitempty"`

	TransitionStatus string `db:"transition_status" json:"transition_status"`

	AnnouncementDate    *time.Time `db:"announcement_date" json:"announcement_date,omitempty"`
	FirstJointMeeting   *time.Time `db:"first_joint_meeting_date" json:"first_joint_meeting_date,omitempty"`
	HandoffCompleteDate *time.Time `db:"handoff_complete_date" json:"handoff_complete_date,omitempty"`

	SatisfactionBefore *float64 `db:"client_satisfaction_before" json:"client_satisfaction_before,omitempty"`
	SatisfactionAfter  *float64 `db:"client_satisfaction_after" json:"client_satisfaction_after,omitempty"`
	ClientRetained     *bool    `db:"client_retained" json:"client_retained,omitempty"`

	Notes *string `db:"notes" json:"notes,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type SuccessorCompatibility struct {
	ScoreID            uuid.UUID `db:"score_id" json:"score_id"`
	DepartingAdvisorID uuid.UUID `db:"departing_advisor_id" json:"departing_advisor_id"`
	CandidateAdvisorID uuid.UUID `db:"candidate_advisor_id" json:"candidate_advisor_id"`

	ClientDemographicMatch float64 `db:"client_demographic_match" json:"client_demographic_match"`
	ServiceStyleMatch      float64 `db:"service_style_match" json:"service_style_match"`
	SpecializationOverlap  float64 `db:"specialization_overlap" json:"specialization_overlap"`
	CapacityMatch          float64 `db:"capacity_match" json:"capacity_match"`
	GeographicMatch        float64 `db:"geographic_match" json:"geographic_match"`

	OverallScore float64 `db:"overall_compatibility_score" json:"overall_compatibility_score"`

	Reasoning    *string   `db:"reasoning" json:"reasoning,omitempty"`
	CalculatedAt time.Time `db:"calculated_at" json:"calculated_at"`
}
