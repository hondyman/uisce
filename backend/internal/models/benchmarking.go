package models

import (
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Industry Benchmarks
// ============================================================================

type IndustryBenchmark struct {
	ID                         uuid.UUID `json:"id" db:"id"`
	Industry                   string    `json:"industry" db:"industry"`
	ProcessType                string    `json:"process_type" db:"process_type"`
	MedianDurationMinutes      *float64  `json:"median_duration_minutes,omitempty" db:"median_duration_minutes"`
	TopQuartileDurationMinutes *float64  `json:"top_quartile_duration_minutes,omitempty" db:"top_quartile_duration_minutes"`
	MedianSuccessRate          *float64  `json:"median_success_rate,omitempty" db:"median_success_rate"`
	TopQuartileSuccessRate     *float64  `json:"top_quartile_success_rate,omitempty" db:"top_quartile_success_rate"`
	MedianCostPerProcess       *float64  `json:"median_cost_per_process,omitempty" db:"median_cost_per_process"`
	TopQuartileCostPerProcess  *float64  `json:"top_quartile_cost_per_process,omitempty" db:"top_quartile_cost_per_process"`
	MedianAutomationRate       *float64  `json:"median_automation_rate,omitempty" db:"median_automation_rate"`
	TopQuartileAutomationRate  *float64  `json:"top_quartile_automation_rate,omitempty" db:"top_quartile_automation_rate"`
	SampleSize                 int       `json:"sample_size" db:"sample_size"`
	LastUpdated                time.Time `json:"last_updated" db:"last_updated"`
	DataSource                 *string   `json:"data_source,omitempty" db:"data_source"`
	CreatedAt                  time.Time `json:"created_at" db:"created_at"`
}

// ============================================================================
// Performance Scores
// ============================================================================

type PerformanceScore struct {
	ID              uuid.UUID `json:"id" db:"id"`
	TenantID        uuid.UUID `json:"tenant_id" db:"tenant_id"`
	WorkflowType    string    `json:"workflow_type" db:"workflow_type"`
	OverallScore    int       `json:"overall_score" db:"overall_score"`
	Grade           string    `json:"grade" db:"grade"`
	Percentile      *int      `json:"percentile,omitempty" db:"percentile"`
	EfficiencyScore int       `json:"efficiency_score" db:"efficiency_score"`
	QualityScore    int       `json:"quality_score" db:"quality_score"`
	SpeedScore      int       `json:"speed_score" db:"speed_score"`
	AutomationScore int       `json:"automation_score" db:"automation_score"`
	ComplianceScore int       `json:"compliance_score" db:"compliance_score"`
	Industry        *string   `json:"industry,omitempty" db:"industry"`
	SampleSize      int       `json:"sample_size" db:"sample_size"`
	ConfidenceLevel float64   `json:"confidence_level" db:"confidence_level"`
	CalculatedAt    time.Time `json:"calculated_at" db:"calculated_at"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

type DimensionScores struct {
	Efficiency int `json:"efficiency"`
	Quality    int `json:"quality"`
	Speed      int `json:"speed"`
	Automation int `json:"automation"`
	Compliance int `json:"compliance"`
}

// ============================================================================
// Best Practices
// ============================================================================

type BestPractice struct {
	ID                         uuid.UUID              `json:"id" db:"id"`
	Title                      string                 `json:"title" db:"title"`
	Description                string                 `json:"description" db:"description"`
	Industry                   *string                `json:"industry,omitempty" db:"industry"`
	ProcessType                *string                `json:"process_type,omitempty" db:"process_type"`
	Category                   *string                `json:"category,omitempty" db:"category"`
	ExpectedImprovementPercent *int                   `json:"expected_improvement_percent,omitempty" db:"expected_improvement_percent"`
	ImplementationEffort       *string                `json:"implementation_effort,omitempty" db:"implementation_effort"`
	ImplementationTimeWeeks    *int                   `json:"implementation_time_weeks,omitempty" db:"implementation_time_weeks"`
	IndustryAdoptionPercent    *int                   `json:"industry_adoption_percent,omitempty" db:"industry_adoption_percent"`
	SuccessRate                *float64               `json:"success_rate,omitempty" db:"success_rate"`
	Prerequisites              *string                `json:"prerequisites,omitempty" db:"prerequisites"`
	ImplementationSteps        map[string]interface{} `json:"implementation_steps,omitempty" db:"implementation_steps"`
	RequiredTools              []string               `json:"required_tools,omitempty" db:"required_tools"`
	EstimatedCostRange         *string                `json:"estimated_cost_range,omitempty" db:"estimated_cost_range"`
	CaseStudyCompany           *string                `json:"case_study_company,omitempty" db:"case_study_company"`
	CaseStudyResults           *string                `json:"case_study_results,omitempty" db:"case_study_results"`
	CaseStudyTimeline          *string                `json:"case_study_timeline,omitempty" db:"case_study_timeline"`
	Priority                   *string                `json:"priority,omitempty" db:"priority"`
	Tags                       []string               `json:"tags,omitempty" db:"tags"`
	ExternalResources          map[string]interface{} `json:"external_resources,omitempty" db:"external_resources"`
	CreatedAt                  time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt                  time.Time              `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// Peer Groups
// ============================================================================

type PeerGroup struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	Name             string     `json:"name" db:"name"`
	Description      *string    `json:"description,omitempty" db:"description"`
	Industry         string     `json:"industry" db:"industry"`
	CompanySizeMin   *int       `json:"company_size_min,omitempty" db:"company_size_min"`
	CompanySizeMax   *int       `json:"company_size_max,omitempty" db:"company_size_max"`
	Geography        *string    `json:"geography,omitempty" db:"geography"`
	AnnualRevenueMin *float64   `json:"annual_revenue_min,omitempty" db:"annual_revenue_min"`
	AnnualRevenueMax *float64   `json:"annual_revenue_max,omitempty" db:"annual_revenue_max"`
	IsActive         bool       `json:"is_active" db:"is_active"`
	CreatedBy        *uuid.UUID `json:"created_by,omitempty" db:"created_by"`
	CreatedAt        time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at" db:"updated_at"`
}

type PeerGroupMember struct {
	ID            uuid.UUID `json:"id" db:"id"`
	PeerGroupID   uuid.UUID `json:"peer_group_id" db:"peer_group_id"`
	TenantID      uuid.UUID `json:"tenant_id" db:"tenant_id"`
	JoinedAt      time.Time `json:"joined_at" db:"joined_at"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	CompanySize   *int      `json:"company_size,omitempty" db:"company_size"`
	AnnualRevenue *float64  `json:"annual_revenue,omitempty" db:"annual_revenue"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// ============================================================================
// Gap Analysis
// ============================================================================

type GapAnalysis struct {
	ID                     uuid.UUID   `json:"id" db:"id"`
	TenantID               uuid.UUID   `json:"tenant_id" db:"tenant_id"`
	WorkflowType           string      `json:"workflow_type" db:"workflow_type"`
	Dimension              string      `json:"dimension" db:"dimension"`
	CurrentScore           int         `json:"current_score" db:"current_score"`
	TargetScore            int         `json:"target_score" db:"target_score"`
	GapPoints              int         `json:"gap_points" db:"gap_points"`
	Priority               *string     `json:"priority,omitempty" db:"priority"`
	Title                  string      `json:"title" db:"title"`
	Description            string      `json:"description" db:"description"`
	RecommendedAction      *string     `json:"recommended_action,omitempty" db:"recommended_action"`
	ExpectedImprovement    *int        `json:"expected_improvement,omitempty" db:"expected_improvement"`
	ImplementationTimeline *string     `json:"implementation_timeline,omitempty" db:"implementation_timeline"`
	RelatedBestPracticeIDs []uuid.UUID `json:"related_best_practice_ids,omitempty" db:"related_best_practice_ids"`
	Status                 string      `json:"status" db:"status"`
	ResolutionNotes        *string     `json:"resolution_notes,omitempty" db:"resolution_notes"`
	IdentifiedAt           time.Time   `json:"identified_at" db:"identified_at"`
	ResolvedAt             *time.Time  `json:"resolved_at,omitempty" db:"resolved_at"`
	CreatedAt              time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time   `json:"updated_at" db:"updated_at"`
}

// ============================================================================
// Response Types for API
// ============================================================================

type BenchmarkScoreResponse struct {
	OverallScore    int             `json:"overall_score"`
	Grade           string          `json:"grade"`
	Percentile      int             `json:"percentile"`
	DimensionScores DimensionScores `json:"dimension_scores"`
	Industry        string          `json:"industry"`
	WorkflowType    string          `json:"workflow_type"`
	CalculatedAt    time.Time       `json:"calculated_at"`
}

type IndustryBenchmarkResponse struct {
	Industry    string             `json:"industry"`
	ProcessType string             `json:"process_type"`
	Median      map[string]float64 `json:"median"`
	TopQuartile map[string]float64 `json:"top_quartile"`
	SampleSize  int                `json:"sample_size"`
	LastUpdated time.Time          `json:"last_updated"`
}

type PeerComparisonResponse struct {
	YourRank      int                    `json:"your_rank"`
	TotalPeers    int                    `json:"total_peers"`
	Percentile    int                    `json:"percentile"`
	PeerGroupName string                 `json:"peer_group_name"`
	Metrics       []PeerMetricComparison `json:"metrics"`
}

type PeerMetricComparison struct {
	MetricName  string  `json:"metric_name"`
	YourValue   float64 `json:"your_value"`
	PeerAverage float64 `json:"peer_average"`
	PeerBest    float64 `json:"peer_best"`
	Unit        string  `json:"unit"`
}
