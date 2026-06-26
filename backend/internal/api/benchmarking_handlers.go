package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/services/benchmarking"
)

// ============================================================================
// Benchmarking Handler
// ============================================================================

type BenchmarkingHandler struct {
	db             *sql.DB
	scoringService *benchmarking.ScoringService
}

func NewBenchmarkingHandler(db *sql.DB) *BenchmarkingHandler {
	return &BenchmarkingHandler{
		db:             db,
		scoringService: benchmarking.NewScoringService(db),
	}
}

// ============================================================================
// GET /api/process-benchmarking/score
// Calculate and return performance score for a workflow
// ============================================================================

func (h *BenchmarkingHandler) GetBenchmarkScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get tenant ID from context (set by middleware)
	tenantID, err := uuid.Parse(r.URL.Query().Get("tenant_id"))
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Get query parameters
	workflowType := r.URL.Query().Get("workflow_type")
	if workflowType == "" {
		http.Error(w, "workflow_type is required", http.StatusBadRequest)
		return
	}

	industry := r.URL.Query().Get("industry")
	if industry == "" {
		industry = "financial_services" // Default
	}

	// Calculate or retrieve cached score
	score, err := h.scoringService.CalculatePerformanceScore(ctx, tenantID, workflowType, industry)
	if err != nil {
		http.Error(w, "Failed to calculate score: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Format response
	response := models.BenchmarkScoreResponse{
		OverallScore: score.OverallScore,
		Grade:        score.Grade,
		Percentile:   *score.Percentile,
		DimensionScores: models.DimensionScores{
			Efficiency: score.EfficiencyScore,
			Quality:    score.QualityScore,
			Speed:      score.SpeedScore,
			Automation: score.AutomationScore,
			Compliance: score.ComplianceScore,
		},
		Industry:     *score.Industry,
		WorkflowType: score.WorkflowType,
		CalculatedAt: score.CalculatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// GET /api/process-benchmarking/industry
// Get industry benchmark data for comparison
// ============================================================================

func (h *BenchmarkingHandler) GetIndustryBenchmark(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	industry := r.URL.Query().Get("industry")
	processType := r.URL.Query().Get("process_type")

	if industry == "" || processType == "" {
		http.Error(w, "industry and process_type are required", http.StatusBadRequest)
		return
	}

	// Query industry benchmark
	query := `
		SELECT 
			industry, process_type, 
			median_duration_minutes, median_success_rate, median_cost_per_process, median_automation_rate,
			top_quartile_duration_minutes, top_quartile_success_rate, top_quartile_cost_per_process, top_quartile_automation_rate,
			sample_size, last_updated
		FROM bp_industry_benchmarks
		WHERE industry = $1 AND process_type = $2
	`

	var benchmark models.IndustryBenchmark
	err := h.db.QueryRowContext(ctx, query, industry, processType).Scan(
		&benchmark.Industry,
		&benchmark.ProcessType,
		&benchmark.MedianDurationMinutes,
		&benchmark.MedianSuccessRate,
		&benchmark.MedianCostPerProcess,
		&benchmark.MedianAutomationRate,
		&benchmark.TopQuartileDurationMinutes,
		&benchmark.TopQuartileSuccessRate,
		&benchmark.TopQuartileCostPerProcess,
		&benchmark.TopQuartileAutomationRate,
		&benchmark.SampleSize,
		&benchmark.LastUpdated,
	)
	if err == sql.ErrNoRows {
		http.Error(w, "No benchmark data found for this industry/process", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to retrieve benchmark: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Format response
	response := models.IndustryBenchmarkResponse{
		Industry:    benchmark.Industry,
		ProcessType: benchmark.ProcessType,
		Median: map[string]float64{
			"duration_minutes": ptrToFloat(benchmark.MedianDurationMinutes),
			"success_rate":     ptrToFloat(benchmark.MedianSuccessRate),
			"cost_per_process": ptrToFloat(benchmark.MedianCostPerProcess),
			"automation_rate":  ptrToFloat(benchmark.MedianAutomationRate),
		},
		TopQuartile: map[string]float64{
			"duration_minutes": ptrToFloat(benchmark.TopQuartileDurationMinutes),
			"success_rate":     ptrToFloat(benchmark.TopQuartileSuccessRate),
			"cost_per_process": ptrToFloat(benchmark.TopQuartileCostPerProcess),
			"automation_rate":  ptrToFloat(benchmark.TopQuartileAutomationRate),
		},
		SampleSize:  benchmark.SampleSize,
		LastUpdated: benchmark.LastUpdated,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// GET /api/process-benchmarking/peers
// Get peer comparison data
// ============================================================================

func (h *BenchmarkingHandler) GetPeerComparison(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := uuid.Parse(r.URL.Query().Get("tenant_id"))
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	workflowType := r.URL.Query().Get("workflow_type")
	if workflowType == "" {
		http.Error(w, "workflow_type is required", http.StatusBadRequest)
		return
	}

	// Find peer group for this tenant
	peerGroupQuery := `
		SELECT pg.id, pg.name
		FROM bp_peer_groups pg
		JOIN bp_peer_group_members pgm ON pg.id = pgm.peer_group_id
		WHERE pgm.tenant_id = $1 AND pgm.is_active = true
		LIMIT 1
	`

	var peerGroupID uuid.UUID
	var peerGroupName string
	err = h.db.QueryRowContext(ctx, peerGroupQuery, tenantID).Scan(&peerGroupID, &peerGroupName)
	if err == sql.ErrNoRows {
		http.Error(w, "No peer group found for tenant", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "Failed to find peer group: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get tenant's score
	tenantScoreQuery := `
		SELECT overall_score, efficiency_score, quality_score, speed_score, automation_score, compliance_score
		FROM bp_performance_scores
		WHERE tenant_id = $1 AND workflow_type = $2
	`

	var tenantScore struct {
		Overall    int
		Efficiency int
		Quality    int
		Speed      int
		Automation int
		Compliance int
	}
	err = h.db.QueryRowContext(ctx, tenantScoreQuery, tenantID, workflowType).Scan(
		&tenantScore.Overall,
		&tenantScore.Efficiency,
		&tenantScore.Quality,
		&tenantScore.Speed,
		&tenantScore.Automation,
		&tenantScore.Compliance,
	)
	if err != nil {
		http.Error(w, "Score not found for tenant", http.StatusNotFound)
		return
	}

	// Get peer statistics
	peerStatsQuery := `
		SELECT 
			COUNT(*) as total_peers,
			AVG(ps.overall_score) as avg_overall,
			MAX(ps.overall_score) as best_overall,
			AVG(ps.efficiency_score) as avg_efficiency,
			AVG(ps.quality_score) as avg_quality,
			AVG(ps.speed_score) as avg_speed,
			AVG(ps.automation_score) as avg_automation,
			AVG(ps.compliance_score) as avg_compliance,
			COUNT(CASE WHEN ps.overall_score < $1 THEN 1 END) as lower_count
		FROM bp_performance_scores ps
		JOIN bp_peer_group_members pgm ON ps.tenant_id = pgm.tenant_id
		WHERE pgm.peer_group_id = $2 
		  AND ps.workflow_type = $3
		  AND ps.tenant_id != $4
	`

	var stats struct {
		TotalPeers    int
		AvgOverall    float64
		BestOverall   float64
		AvgEfficiency float64
		AvgQuality    float64
		AvgSpeed      float64
		AvgAutomation float64
		AvgCompliance float64
		LowerCount    int
	}

	err = h.db.QueryRowContext(ctx, peerStatsQuery, tenantScore.Overall, peerGroupID, workflowType, tenantID).Scan(
		&stats.TotalPeers,
		&stats.AvgOverall,
		&stats.BestOverall,
		&stats.AvgEfficiency,
		&stats.AvgQuality,
		&stats.AvgSpeed,
		&stats.AvgAutomation,
		&stats.AvgCompliance,
		&stats.LowerCount,
	)
	if err != nil {
		http.Error(w, "Failed to calculate peer stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate rank and percentile
	rank := stats.TotalPeers - stats.LowerCount + 1
	percentile := 0
	if stats.TotalPeers > 0 {
		percentile = int((float64(stats.LowerCount) / float64(stats.TotalPeers)) * 100)
	}

	// Build metrics comparison
	metrics := []models.PeerMetricComparison{
		{
			MetricName:  "Overall Score",
			YourValue:   float64(tenantScore.Overall),
			PeerAverage: stats.AvgOverall,
			PeerBest:    stats.BestOverall,
			Unit:        "points",
		},
		{
			MetricName:  "Efficiency",
			YourValue:   float64(tenantScore.Efficiency),
			PeerAverage: stats.AvgEfficiency,
			PeerBest:    100.0,
			Unit:        "points",
		},
		{
			MetricName:  "Quality",
			YourValue:   float64(tenantScore.Quality),
			PeerAverage: stats.AvgQuality,
			PeerBest:    100.0,
			Unit:        "points",
		},
		{
			MetricName:  "Speed",
			YourValue:   float64(tenantScore.Speed),
			PeerAverage: stats.AvgSpeed,
			PeerBest:    100.0,
			Unit:        "points",
		},
		{
			MetricName:  "Automation",
			YourValue:   float64(tenantScore.Automation),
			PeerAverage: stats.AvgAutomation,
			PeerBest:    100.0,
			Unit:        "points",
		},
		{
			MetricName:  "Compliance",
			YourValue:   float64(tenantScore.Compliance),
			PeerAverage: stats.AvgCompliance,
			PeerBest:    100.0,
			Unit:        "points",
		},
	}

	response := models.PeerComparisonResponse{
		YourRank:      rank,
		TotalPeers:    stats.TotalPeers + 1, // Include self
		Percentile:    percentile,
		PeerGroupName: peerGroupName,
		Metrics:       metrics,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ============================================================================
// GET /api/process-benchmarking/best-practices
// Get recommended best practices
// ============================================================================

func (h *BenchmarkingHandler) GetBestPractices(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	industry := r.URL.Query().Get("industry")
	processType := r.URL.Query().Get("process_type")
	category := r.URL.Query().Get("category")

	query := `
		SELECT 
			id, title, description, industry, process_type, category,
			expected_improvement_percent, implementation_effort, implementation_time_weeks,
			industry_adoption_percent, success_rate, prerequisites, implementation_steps,
			required_tools, estimated_cost_range, case_study_company, case_study_results,
			case_study_timeline, priority, tags, external_resources
		FROM bp_best_practices
		WHERE (industry = $1 OR industry IS NULL)
		  AND (process_type = $2 OR process_type IS NULL)
		  AND ($3 = '' OR category = $3)
		ORDER BY priority DESC, expected_improvement_percent DESC
		LIMIT 20
	`

	rows, err := h.db.QueryContext(ctx, query, industry, processType, category)
	if err != nil {
		http.Error(w, "Failed to query best practices: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var practices []models.BestPractice
	for rows.Next() {
		var bp models.BestPractice
		err := rows.Scan(
			&bp.ID,
			&bp.Title,
			&bp.Description,
			&bp.Industry,
			&bp.ProcessType,
			&bp.Category,
			&bp.ExpectedImprovementPercent,
			&bp.ImplementationEffort,
			&bp.ImplementationTimeWeeks,
			&bp.IndustryAdoptionPercent,
			&bp.SuccessRate,
			&bp.Prerequisites,
			&bp.ImplementationSteps,
			&bp.RequiredTools,
			&bp.EstimatedCostRange,
			&bp.CaseStudyCompany,
			&bp.CaseStudyResults,
			&bp.CaseStudyTimeline,
			&bp.Priority,
			&bp.Tags,
			&bp.ExternalResources,
		)
		if err != nil {
			continue
		}
		practices = append(practices, bp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(practices)
}

// ============================================================================
// GET /api/process-benchmarking/gap-analysis
// Get performance gaps and recommendations
// ============================================================================

func (h *BenchmarkingHandler) GetGapAnalysis(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := uuid.Parse(r.URL.Query().Get("tenant_id"))
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	workflowType := r.URL.Query().Get("workflow_type")
	if workflowType == "" {
		http.Error(w, "workflow_type is required", http.StatusBadRequest)
		return
	}

	query := `
		SELECT 
			id, workflow_type, dimension, current_score, target_score, gap_points,
			priority, title, description, recommended_action, expected_improvement,
			implementation_timeline, related_best_practice_ids, status
		FROM bp_gap_analysis
		WHERE tenant_id = $1 AND workflow_type = $2 AND status = 'identified'
		ORDER BY priority DESC, gap_points DESC
	`

	rows, err := h.db.QueryContext(ctx, query, tenantID, workflowType)
	if err != nil {
		http.Error(w, "Failed to query gap analysis: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var gaps []models.GapAnalysis
	for rows.Next() {
		var gap models.GapAnalysis
		err := rows.Scan(
			&gap.ID,
			&gap.WorkflowType,
			&gap.Dimension,
			&gap.CurrentScore,
			&gap.TargetScore,
			&gap.GapPoints,
			&gap.Priority,
			&gap.Title,
			&gap.Description,
			&gap.RecommendedAction,
			&gap.ExpectedImprovement,
			&gap.ImplementationTimeline,
			&gap.RelatedBestPracticeIDs,
			&gap.Status,
		)
		if err != nil {
			continue
		}
		gaps = append(gaps, gap)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gaps)
}

// ============================================================================
// POST /api/process-benchmarking/calculate-score
// Manually trigger score recalculation
// ============================================================================

func (h *BenchmarkingHandler) CalculateScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		TenantID     string `json:"tenant_id"`
		WorkflowType string `json:"workflow_type"`
		Industry     string `json:"industry"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "Invalid tenant_id", http.StatusBadRequest)
		return
	}

	// Calculate score
	score, err := h.scoringService.CalculatePerformanceScore(ctx, tenantID, req.WorkflowType, req.Industry)
	if err != nil {
		http.Error(w, "Failed to calculate score: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Score calculated successfully",
		"score":   score,
	})
}

// ============================================================================
// Route Registration
// ============================================================================

func (h *BenchmarkingHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/process-benchmarking/score", h.GetBenchmarkScore)
	r.Get("/api/process-benchmarking/industry", h.GetIndustryBenchmark)
	r.Get("/api/process-benchmarking/peers", h.GetPeerComparison)
	r.Get("/api/process-benchmarking/best-practices", h.GetBestPractices)
	r.Get("/api/process-benchmarking/gap-analysis", h.GetGapAnalysis)
	r.Post("/api/process-benchmarking/calculate-score", h.CalculateScore)
}

// ============================================================================
// Helper Functions
// ============================================================================

func ptrToFloat(ptr *float64) float64 {
	if ptr == nil {
		return 0.0
	}
	return *ptr
}
