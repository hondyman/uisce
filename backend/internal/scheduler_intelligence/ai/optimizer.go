package ai

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ScheduleOptimizer optimizes job and DAG schedules for better performance
type ScheduleOptimizer struct {
	logger *slog.Logger
}

// NewScheduleOptimizer creates a new schedule optimizer
func NewScheduleOptimizer(logger *slog.Logger) *ScheduleOptimizer {
	return &ScheduleOptimizer{
		logger: logger,
	}
}

// OptimizationRequest represents a request to optimize schedules
type OptimizationRequest struct {
	TenantID       uuid.UUID    `json:"tenant_id"`
	Jobs           []JobContext `json:"jobs"`
	Constraints    Constraints  `json:"constraints"`
	Objectives     []string     `json:"objectives"` // minimize_latency, maximize_throughput, balance_load
	LookaheadHours int          `json:"lookahead_hours"`
}

// JobContext represents a job with its historical performance data
type JobContext struct {
	JobID              uuid.UUID     `json:"job_id"`
	Name               string        `json:"name"`
	Category           string        `json:"category"`
	CurrentSchedule    string        `json:"current_schedule"`
	AvgDurationSeconds int           `json:"avg_duration_seconds"`
	FailureRate        float64       `json:"failure_rate"`
	SLOCritical        bool          `json:"slo_critical"`
	Priority           int           `json:"priority"`
	Dependencies       []uuid.UUID   `json:"dependencies"`
	ResourceUsage      ResourceUsage `json:"resource_usage"`
}

// ResourceUsage tracks resource consumption patterns
type ResourceUsage struct {
	AvgCPU    float64 `json:"avg_cpu"`
	AvgMemory float64 `json:"avg_memory"`
	AvgIO     float64 `json:"avg_io"`
}

// Constraints defines scheduling constraints
type Constraints struct {
	BlackoutWindows    []TimeWindow `json:"blackout_windows"`
	MaintenanceWindows []TimeWindow `json:"maintenance_windows"`
	MaxConcurrency     int          `json:"max_concurrency"`
	PreferredHours     []int        `json:"preferred_hours"` // 0-23
	AvoidHours         []int        `json:"avoid_hours"`
}

// TimeWindow represents a time range
type TimeWindow struct {
	Start  time.Time `json:"start"`
	End    time.Time `json:"end"`
	Reason string    `json:"reason,omitempty"`
}

// OptimizationResult contains the optimized schedule recommendations
type OptimizationResult struct {
	Recommendations []ScheduleRecommendation `json:"recommendations"`
	ImpactSummary   ImpactSummary            `json:"impact_summary"`
	Warnings        []string                 `json:"warnings,omitempty"`
	Confidence      float64                  `json:"confidence"`
}

// ScheduleRecommendation is a single scheduling recommendation
type ScheduleRecommendation struct {
	JobID            uuid.UUID `json:"job_id"`
	JobName          string    `json:"job_name"`
	CurrentSchedule  string    `json:"current_schedule"`
	ProposedSchedule string    `json:"proposed_schedule"`
	Reason           string    `json:"reason"`
	ExpectedBenefit  string    `json:"expected_benefit"`
	RiskLevel        string    `json:"risk_level"` // low, medium, high
}

// ImpactSummary summarizes the overall optimization impact
type ImpactSummary struct {
	EstimatedLatencyReduction    float64 `json:"estimated_latency_reduction_pct"`
	EstimatedThroughputGain      float64 `json:"estimated_throughput_gain_pct"`
	EstimatedContentionReduction float64 `json:"estimated_contention_reduction_pct"`
	SLORiskChange                string  `json:"slo_risk_change"` // improved, neutral, degraded
}

// Optimize generates schedule optimization recommendations
func (o *ScheduleOptimizer) Optimize(ctx context.Context, req OptimizationRequest) (*OptimizationResult, error) {
	o.logger.Info("Starting schedule optimization",
		"tenant_id", req.TenantID,
		"job_count", len(req.Jobs),
		"objectives", req.Objectives,
	)

	// Analyze current schedule
	analysis := o.analyzeCurrentSchedule(req.Jobs)

	// Identify optimization opportunities
	opportunities := o.identifyOpportunities(req.Jobs, req.Constraints, analysis)

	// Generate recommendations
	recommendations := o.generateRecommendations(opportunities, req.Objectives)

	// Calculate impact
	impact := o.calculateImpact(req.Jobs, recommendations)

	result := &OptimizationResult{
		Recommendations: recommendations,
		ImpactSummary:   impact,
		Confidence:      o.calculateConfidence(len(req.Jobs), len(recommendations)),
	}

	o.logger.Info("Optimization complete",
		"recommendations", len(recommendations),
		"confidence", result.Confidence,
	)

	return result, nil
}

// ScheduleAnalysis contains analysis of current schedules
type ScheduleAnalysis struct {
	ContentionHours    map[int]int  // hour -> job count
	HighLoadPeriods    []TimeWindow // periods with high load
	UnderutilizedSlots []int        // hours with low utilization
	SLORisks           []SLORisk    // jobs at risk of SLO breach
}

// SLORisk identifies an SLO risk
type SLORisk struct {
	JobID   uuid.UUID
	JobName string
	Reason  string
}

// analyzeCurrentSchedule examines the current scheduling patterns
func (o *ScheduleOptimizer) analyzeCurrentSchedule(jobs []JobContext) *ScheduleAnalysis {
	analysis := &ScheduleAnalysis{
		ContentionHours: make(map[int]int),
	}

	// Count jobs per hour (simplified - would parse cron in real impl)
	for i := 0; i < 24; i++ {
		analysis.ContentionHours[i] = 0
	}

	// Identify high-contention periods (more than 5 jobs in same hour)
	for hour, count := range analysis.ContentionHours {
		if count > 5 {
			analysis.HighLoadPeriods = append(analysis.HighLoadPeriods, TimeWindow{
				Reason: fmt.Sprintf("%d jobs scheduled at hour %d", count, hour),
			})
		}
	}

	// Check SLO-critical jobs
	for _, job := range jobs {
		if job.SLOCritical && job.FailureRate > 0.05 {
			analysis.SLORisks = append(analysis.SLORisks, SLORisk{
				JobID:   job.JobID,
				JobName: job.Name,
				Reason:  fmt.Sprintf("High failure rate (%.1f%%) for SLO-critical job", job.FailureRate*100),
			})
		}
	}

	return analysis
}

// OptimizationOpportunity represents a potential improvement
type OptimizationOpportunity struct {
	Type        string // stagger, move, parallelize, consolidate
	JobIDs      []uuid.UUID
	Description string
	Impact      float64 // 0-1, higher is better
	Risk        float64 // 0-1, higher is riskier
}

// identifyOpportunities finds optimization opportunities
func (o *ScheduleOptimizer) identifyOpportunities(jobs []JobContext, constraints Constraints, analysis *ScheduleAnalysis) []OptimizationOpportunity {
	var opportunities []OptimizationOpportunity

	// Find staggering opportunities (jobs running at same time)
	hourCounts := make(map[int][]JobContext)
	for _, job := range jobs {
		// Simplified: would parse cron expression
		hour := 2 // Assume 2 AM for demo
		hourCounts[hour] = append(hourCounts[hour], job)
	}

	for hour, hourJobs := range hourCounts {
		if len(hourJobs) > 3 {
			jobIDs := make([]uuid.UUID, len(hourJobs))
			for i, j := range hourJobs {
				jobIDs[i] = j.JobID
			}
			opportunities = append(opportunities, OptimizationOpportunity{
				Type:        "stagger",
				JobIDs:      jobIDs,
				Description: fmt.Sprintf("Stagger %d jobs currently scheduled at %d:00", len(hourJobs), hour),
				Impact:      0.3,
				Risk:        0.2,
			})
		}
	}

	// Find SLO-critical jobs that could benefit from earlier scheduling
	for _, job := range jobs {
		if job.SLOCritical && job.AvgDurationSeconds > 600 {
			opportunities = append(opportunities, OptimizationOpportunity{
				Type:        "move",
				JobIDs:      []uuid.UUID{job.JobID},
				Description: fmt.Sprintf("Move SLO-critical job '%s' earlier to reduce deadline pressure", job.Name),
				Impact:      0.5,
				Risk:        0.3,
			})
		}
	}

	// Find parallelization opportunities
	independentJobs := o.findIndependentJobs(jobs)
	if len(independentJobs) > 1 {
		opportunities = append(opportunities, OptimizationOpportunity{
			Type:        "parallelize",
			JobIDs:      independentJobs,
			Description: fmt.Sprintf("Parallelize %d independent jobs", len(independentJobs)),
			Impact:      0.4,
			Risk:        0.1,
		})
	}

	// Sort by impact (higher first)
	sort.Slice(opportunities, func(i, j int) bool {
		return opportunities[i].Impact > opportunities[j].Impact
	})

	return opportunities
}

// findIndependentJobs identifies jobs with no dependencies between them
func (o *ScheduleOptimizer) findIndependentJobs(jobs []JobContext) []uuid.UUID {
	var independent []uuid.UUID
	for _, job := range jobs {
		if len(job.Dependencies) == 0 {
			independent = append(independent, job.JobID)
		}
	}
	return independent
}

// generateRecommendations creates actionable recommendations from opportunities
func (o *ScheduleOptimizer) generateRecommendations(opportunities []OptimizationOpportunity, objectives []string) []ScheduleRecommendation {
	var recommendations []ScheduleRecommendation

	for _, opp := range opportunities {
		// Only take top opportunities
		if len(recommendations) >= 10 {
			break
		}

		rec := ScheduleRecommendation{
			JobID:           opp.JobIDs[0], // Primary job
			Reason:          opp.Description,
			ExpectedBenefit: fmt.Sprintf("%.0f%% improvement", opp.Impact*100),
			RiskLevel:       o.riskLevel(opp.Risk),
		}

		switch opp.Type {
		case "stagger":
			rec.ProposedSchedule = "Stagger by 15 minute intervals"
		case "move":
			rec.ProposedSchedule = "Move 30 minutes earlier"
		case "parallelize":
			rec.ProposedSchedule = "Run in parallel batch"
		}

		recommendations = append(recommendations, rec)
	}

	return recommendations
}

// riskLevel converts risk score to label
func (o *ScheduleOptimizer) riskLevel(risk float64) string {
	if risk < 0.3 {
		return "low"
	}
	if risk < 0.6 {
		return "medium"
	}
	return "high"
}

// calculateImpact estimates the overall impact of recommendations
func (o *ScheduleOptimizer) calculateImpact(jobs []JobContext, recs []ScheduleRecommendation) ImpactSummary {
	// Simplified impact calculation
	recCount := float64(len(recs))
	jobCount := float64(len(jobs))

	return ImpactSummary{
		EstimatedLatencyReduction:    recCount / jobCount * 15, // up to 15% reduction
		EstimatedThroughputGain:      recCount / jobCount * 10,
		EstimatedContentionReduction: recCount / jobCount * 20,
		SLORiskChange:                "improved",
	}
}

// calculateConfidence determines confidence in recommendations
func (o *ScheduleOptimizer) calculateConfidence(totalJobs, recommendations int) float64 {
	if totalJobs == 0 {
		return 0
	}
	// Higher confidence with more data and fewer recommendations
	base := 0.7
	if totalJobs > 10 {
		base = 0.85
	}
	if recommendations > 5 {
		base -= 0.1 // More recommendations = less certainty
	}
	return base
}
