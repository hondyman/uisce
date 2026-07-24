package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ========== Phase 3.14: Analytics & Batch Operations Handlers ==========

// ListSLAComplianceTrendsResponse returns SLA compliance trends
type ListSLAComplianceTrendsResponse struct {
	Trends []SLAComplianceTrend `json:"trends"`
	Total  int                  `json:"total"`
}

// ListSLAComplianceTrends handles GET /admin/ops/analytics/sla-trends
// Returns SLA compliance trends over time
func (h *Handler) ListSLAComplianceTrends(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	limit := 30
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}

	trends, err := h.store.ListSLAComplianceTrends(r.Context(), tenantID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, ListSLAComplianceTrendsResponse{
		Trends: trends,
		Total:  len(trends),
	})
}

// GetConflictTrendResponse returns conflict resolution statistics
type GetConflictTrendResponse struct {
	TotalConflicts  int       `json:"total_conflicts"`
	ResolvedCount   int       `json:"resolved_count"`
	FailedCount     int       `json:"failed_count"`
	ResolutionRate  float64   `json:"resolution_rate"`
	AvgResolutionMs int64     `json:"avg_resolution_ms"`
	MostCommonRule  string    `json:"most_common_rule"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
}

// GetConflictResolutionTrend handles GET /admin/ops/analytics/conflict-trends
// Returns conflict resolution statistics for current period
func (h *Handler) GetConflictResolutionTrend(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	// Get trend for current day
	now := time.Now().UTC()
	periodStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	trend, err := h.store.GetConflictResolutionTrend(r.Context(), tenantID, periodStart)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if trend == nil {
		// Return empty trend if not found
		respondJSON(w, http.StatusOK, GetConflictTrendResponse{
			TotalConflicts:  0,
			ResolvedCount:   0,
			FailedCount:     0,
			ResolutionRate:  0,
			AvgResolutionMs: 0,
			PeriodStart:     periodStart,
			PeriodEnd:       now,
		})
		return
	}

	respondJSON(w, http.StatusOK, GetConflictTrendResponse{
		TotalConflicts:  trend.TotalConflicts,
		ResolvedCount:   trend.ResolvedCount,
		FailedCount:     trend.FailedCount,
		ResolutionRate:  trend.ResolutionRate,
		AvgResolutionMs: trend.AvgResolutionMs,
		MostCommonRule:  trend.MostCommonRule,
		PeriodStart:     trend.PeriodStart,
		PeriodEnd:       trend.PeriodEnd,
	})
}

// GetChainHealthResponse returns chain health report
type GetChainHealthResponse struct {
	ChainID             uuid.UUID `json:"chain_id"`
	OverallHealth       int       `json:"overall_health"` // 0-100
	LastExecutionStatus string    `json:"last_execution_status"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
	IsHealthy           bool      `json:"is_healthy"`
	RecommendedAction   string    `json:"recommended_action"`
	ReportedAt          time.Time `json:"reported_at"`
}

// GetChainHealth handles GET /admin/ops/chains/{chainID}/health
// Returns current health status and recommended actions
func (h *Handler) GetChainHealth(w http.ResponseWriter, r *http.Request) {
	chainIDStr := chi.URLParam(r, "chainID")

	chainID, err := uuid.Parse(chainIDStr)
	if err != nil {
		http.Error(w, "invalid chain_id", http.StatusBadRequest)
		return
	}

	report, err := h.store.GetChainHealthReport(r.Context(), chainID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if report == nil {
		http.Error(w, "health report not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, GetChainHealthResponse{
		ChainID:             report.ChainID,
		OverallHealth:       report.OverallHealth,
		LastExecutionStatus: report.LastExecutionStatus,
		ConsecutiveFailures: report.ConsecutiveFailures,
		IsHealthy:           report.IsHealthy,
		RecommendedAction:   report.RecommendedAction,
		ReportedAt:          report.ReportedAt,
	})
}

// SearchChainsRequest is the query parameters for chain search
type SearchChainsResponse struct {
	Chains []FailoverChain `json:"chains"`
	Total  int             `json:"total"`
	Query  string          `json:"query"`
}

// SearchChains handles GET /admin/ops/chains/search?q=...
// Full-text search for chains by name or region
func (h *Handler) SearchChains(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "search query (q) is required", http.StatusBadRequest)
		return
	}

	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if v, err := strconv.Atoi(limitStr); err == nil && v > 0 && v <= 500 {
			limit = v
		}
	}

	chains, err := h.store.SearchChains(r.Context(), tenantID, query, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, SearchChainsResponse{
		Chains: chains,
		Total:  len(chains),
		Query:  query,
	})
}

// FilterChainsRequest is the request body for advanced filtering
type FilterChainsRequest struct {
	SourceRegion     string   `json:"source_region,omitempty"`
	MinSLACompliance *float64 `json:"min_sla_compliance,omitempty"`
	IsEnabled        *bool    `json:"is_enabled,omitempty"`
	SortBy           string   `json:"sort_by,omitempty"`    // "sla_compliance", "success_rate", "created_at"
	SortOrder        string   `json:"sort_order,omitempty"` // "asc", "desc"
	Limit            int      `json:"limit,omitempty"`
}

// FilterChainsResponse returns filtered chains
type FilterChainsResponse struct {
	Chains []FailoverChain `json:"chains"`
	Total  int             `json:"total"`
}

// FilterChains handles POST /admin/ops/chains/filter
// Advanced filtering with multiple criteria
func (h *Handler) FilterChains(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	var req FilterChainsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	// Build filter criteria
	criteria := &ChainFilterCriteria{
		TenantID:  &tenantID,
		IsEnabled: req.IsEnabled,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		Limit:     req.Limit,
	}

	if req.SourceRegion != "" {
		criteria.SourceRegion = &req.SourceRegion
	}
	if req.MinSLACompliance != nil {
		criteria.MinSLACompliance = req.MinSLACompliance
	}

	if criteria.Limit <= 0 {
		criteria.Limit = 50
	}

	chains, err := h.store.ListChainsByFilter(r.Context(), criteria)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, FilterChainsResponse{
		Chains: chains,
		Total:  len(chains),
	})
}

// ========== Batch Operations ==========

// BatchResolveConflictsRequest resolves multiple conflicts at once
type BatchResolveConflictsRequest struct {
	ConflictIDs    []uuid.UUID `json:"conflict_ids"`
	ResolutionRule string      `json:"resolution_rule"` // "priority", "first_win", "serial_execute"
}

// BatchResolveConflictsResponse returns batch operation details
type BatchResolveConflictsResponse struct {
	BatchID        uuid.UUID `json:"batch_id"`
	TotalConflicts int       `json:"total_conflicts"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// BatchResolveConflicts handles POST /admin/ops/batch/conflicts/resolve
// Resolves multiple conflicts with the same rule
func (h *Handler) BatchResolveConflicts(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	var req BatchResolveConflictsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.ConflictIDs) == 0 {
		http.Error(w, "conflict_ids is required", http.StatusBadRequest)
		return
	}

	if req.ResolutionRule == "" {
		http.Error(w, "resolution_rule is required", http.StatusBadRequest)
		return
	}

	// Create batch operation
	conflictIDsJSON, _ := json.Marshal(req.ConflictIDs)
	batch := &BatchConflictResolution{
		ID:             uuid.New(),
		TenantID:       tenantID,
		ConflictIDs:    string(conflictIDsJSON),
		ResolutionRule: req.ResolutionRule,
		Status:         "pending",
		TotalConflicts: len(req.ConflictIDs),
		ResolvedCount:  0,
		FailedCount:    0,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	if err := h.store.InsertBatchConflictResolution(r.Context(), batch); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, BatchResolveConflictsResponse{
		BatchID:        batch.ID,
		TotalConflicts: batch.TotalConflicts,
		Status:         batch.Status,
		CreatedAt:      batch.CreatedAt,
	})
}

// GetBatchOperationResponse returns batch operation status
type GetBatchOperationResponse struct {
	BatchID        uuid.UUID  `json:"batch_id"`
	Status         string     `json:"status"`
	TotalConflicts int        `json:"total_conflicts"`
	ResolvedCount  int        `json:"resolved_count"`
	FailedCount    int        `json:"failed_count"`
	ProgressPct    float64    `json:"progress_percent"`
	ExecutedAt     *time.Time `json:"executed_at,omitempty"`
}

// GetBatchOperation handles GET /admin/ops/batch/conflicts/{batchID}
// Returns batch operation status and progress
func (h *Handler) GetBatchOperation(w http.ResponseWriter, r *http.Request) {
	batchIDStr := chi.URLParam(r, "batchID")

	batchID, err := uuid.Parse(batchIDStr)
	if err != nil {
		http.Error(w, "invalid batch_id", http.StatusBadRequest)
		return
	}

	batch, err := h.store.GetBatchConflictResolution(r.Context(), batchID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if batch == nil {
		http.Error(w, "batch operation not found", http.StatusNotFound)
		return
	}

	progressPct := 0.0
	if batch.TotalConflicts > 0 {
		progressPct = float64(batch.ResolvedCount+batch.FailedCount) / float64(batch.TotalConflicts) * 100
	}

	respondJSON(w, http.StatusOK, GetBatchOperationResponse{
		BatchID:        batch.ID,
		Status:         batch.Status,
		TotalConflicts: batch.TotalConflicts,
		ResolvedCount:  batch.ResolvedCount,
		FailedCount:    batch.FailedCount,
		ProgressPct:    progressPct,
		ExecutedAt:     batch.ExecutedAt,
	})
}

// ========= Analytics Engine ==========

// AnalyticsEngine computes analytics and reports
type AnalyticsEngine struct {
	store Store
}

// NewAnalyticsEngine creates a new analytics engine
func NewAnalyticsEngine(store Store) *AnalyticsEngine {
	return &AnalyticsEngine{store: store}
}

// ComputeChainHealth computes overall health score for a chain
func (ae *AnalyticsEngine) ComputeChainHealth(ctx context.Context, chainID uuid.UUID, stats *ChainExecutionStats) int {
	if stats == nil {
		return 0
	}

	// Health = weighted average of: success_rate (60%) + execution_count impact (40%)
	// More executions = more confidence in the metric
	successComponent := stats.SuccessRatePct * 0.6

	// Execution count component: more executions = higher confidence
	// Cap at 100 executions for scoring purposes
	execCount := float64(stats.TotalExecutions)
	if execCount > 100 {
		execCount = 100
	}
	executionComponent := (execCount / 100) * 40

	health := int(successComponent + executionComponent)
	if health > 100 {
		health = 100
	}
	return health
}

// GetRecommendedAction suggests an action based on chain health
func (ae *AnalyticsEngine) GetRecommendedAction(stats *ChainExecutionStats) string {
	if stats == nil {
		return "investigate"
	}

	successRate := stats.SuccessRatePct
	if successRate < 50 {
		return "disable"
	} else if successRate < 80 {
		return "investigate"
	} else if successRate < 95 {
		return "retry"
	}
	return "none"
}
