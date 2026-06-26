package ops

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// ========== Phase 3.13: Advanced Chain Management API Handlers ==========

// CreateChainStateRequest is the request body for initializing chain state
type CreateChainStateRequest struct {
	ChainID uuid.UUID `json:"chain_id"`
	// TenantID comes from context
}

// CreateChainStateResponse returns the initialized chain state
type CreateChainStateResponse struct {
	ChainID        uuid.UUID  `json:"chain_id"`
	TenantID       uuid.UUID  `json:"tenant_id"`
	IsExecuting    bool       `json:"is_executing"`
	LastError      *string    `json:"last_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	NextEligibleAt *time.Time `json:"next_eligible_at,omitempty"`
}

// InitializeChainState handles POST /admin/ops/chains/{chainID}/state
// Initializes execution state tracking for a failover chain
func (h *Handler) InitializeChainState(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)
	chainIDStr := chi.URLParam(r, "chainID")

	chainID, err := uuid.Parse(chainIDStr)
	if err != nil {
		http.Error(w, "invalid chain_id", http.StatusBadRequest)
		return
	}

	// Create initial state
	state := &FailoverChainState{
		ID:               uuid.New(),
		ChainID:          chainID,
		TenantID:         tenantID,
		CurrentStepIndex: 0,
		IsExecuting:      false,
		UpdatedAt:        time.Now().UTC(),
	}

	if err := h.store.InsertFailoverChainState(r.Context(), state); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, CreateChainStateResponse{
		ChainID:        chainID,
		TenantID:       tenantID,
		IsExecuting:    false,
		LastError:      nil,
		CreatedAt:      state.UpdatedAt,
		NextEligibleAt: state.NextEligibleAt,
	})
}

// GetChainStateResponse returns current chain execution state
type GetChainStateResponse struct {
	ChainID             uuid.UUID  `json:"chain_id"`
	TenantID            uuid.UUID  `json:"tenant_id"`
	LastExecutedAt      *time.Time `json:"last_executed_at,omitempty"`
	NextEligibleAt      *time.Time `json:"next_eligible_at,omitempty"`
	CurrentStepIndex    int        `json:"current_step_index"`
	IsExecuting         bool       `json:"is_executing"`
	LastError           *string    `json:"last_error,omitempty"`
	ConsecutiveFailures int        `json:"consecutive_failures"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// GetChainState handles GET /admin/ops/chains/{chainID}/state
// Retrieves current execution state and cooldown status
func (h *Handler) GetChainState(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)
	chainIDStr := chi.URLParam(r, "chainID")

	chainID, err := uuid.Parse(chainIDStr)
	if err != nil {
		http.Error(w, "invalid chain_id", http.StatusBadRequest)
		return
	}

	state, err := h.store.GetFailoverChainState(r.Context(), chainID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if state == nil {
		http.Error(w, "chain state not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, GetChainStateResponse{
		ChainID:             state.ChainID,
		TenantID:            state.TenantID,
		LastExecutedAt:      state.LastExecutedAt,
		NextEligibleAt:      state.NextEligibleAt,
		CurrentStepIndex:    state.CurrentStepIndex,
		IsExecuting:         state.IsExecuting,
		LastError:           state.LastError,
		ConsecutiveFailures: state.ConsecutiveFailures,
		UpdatedAt:           state.UpdatedAt,
	})
}

// ListChainStatesResponse returns all states for a tenant
type ListChainStatesResponse struct {
	States []GetChainStateResponse `json:"states"`
	Total  int                     `json:"total"`
}

// ListChainStates handles GET /admin/ops/chains/states
// Lists all chain execution states for the current tenant
func (h *Handler) ListChainStates(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	states, err := h.store.ListFailoverChainStates(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responses []GetChainStateResponse
	for _, s := range states {
		responses = append(responses, GetChainStateResponse{
			ChainID:             s.ChainID,
			TenantID:            s.TenantID,
			LastExecutedAt:      s.LastExecutedAt,
			NextEligibleAt:      s.NextEligibleAt,
			CurrentStepIndex:    s.CurrentStepIndex,
			IsExecuting:         s.IsExecuting,
			LastError:           s.LastError,
			ConsecutiveFailures: s.ConsecutiveFailures,
			UpdatedAt:           s.UpdatedAt,
		})
	}

	respondJSON(w, http.StatusOK, ListChainStatesResponse{
		States: responses,
		Total:  len(responses),
	})
}

// ========== Conflict Detection & Resolution Endpoints ==========

// GetConflict handles GET /admin/ops/chains/{chainID}/conflicts
func (h *Handler) GetConflict(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)
	chainIDStr := chi.URLParam(r, "chainID")

	chainID, err := uuid.Parse(chainIDStr)
	if err != nil {
		http.Error(w, "invalid chain_id", http.StatusBadRequest)
		return
	}

	conflicts, err := h.store.ListFailoverChainConflicts(r.Context(), chainID, tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"conflicts": conflicts,
		"total":     len(conflicts),
	})
}

// ResolveConflictRequest is the request body for conflict resolution
type ResolveConflictRequest struct {
	ResolutionRule string `json:"resolution_rule"` // "priority", "first_win", "serial_execute"
}

// ResolveConflict handles PUT /admin/ops/conflicts/{conflictID}
func (h *Handler) ResolveConflict(w http.ResponseWriter, r *http.Request) {
	conflictIDStr := chi.URLParam(r, "conflictID")

	conflictID, err := uuid.Parse(conflictIDStr)
	if err != nil {
		http.Error(w, "invalid conflict_id", http.StatusBadRequest)
		return
	}

	var req ResolveConflictRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.ResolutionRule == "" {
		http.Error(w, "resolution_rule is required", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateConflictResolution(r.Context(), conflictID, true, req.ResolutionRule); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "conflict resolved",
	})
}

// ========== Metrics & SLA Compliance Endpoints ==========

// GetChainMetricsResponse returns SLA compliance metrics
type GetChainMetricsResponse struct {
	ChainID           uuid.UUID `json:"chain_id"`
	TotalExecutions   int       `json:"total_executions"`
	P50DurationMs     int64     `json:"p50_duration_ms"`
	P75DurationMs     int64     `json:"p75_duration_ms"`
	P95DurationMs     int64     `json:"p95_duration_ms"`
	P99DurationMs     int64     `json:"p99_duration_ms"`
	MinDurationMs     int64     `json:"min_duration_ms"`
	MaxDurationMs     int64     `json:"max_duration_ms"`
	StdDevDurationMs  float64   `json:"std_dev_duration_ms"`
	SuccessRate99th   float64   `json:"success_rate_99th"`
	AvgStepsNeeded    float64   `json:"avg_steps_needed"`
	P95StepsNeeded    int       `json:"p95_steps_needed"`
	MostCommonFailure *string   `json:"most_common_failure,omitempty"`
	SLACompliance     float64   `json:"sla_compliance"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// GetChainMetrics handles GET /admin/ops/chains/{chainID}/metrics
// Returns percentile latency, success rates, and SLA compliance
func (h *Handler) GetChainMetrics(w http.ResponseWriter, r *http.Request) {
	chainIDStr := chi.URLParam(r, "chainID")

	chainID, err := uuid.Parse(chainIDStr)
	if err != nil {
		http.Error(w, "invalid chain_id", http.StatusBadRequest)
		return
	}

	metrics, err := h.store.GetChainExecutionMetricsAdvanced(r.Context(), chainID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if metrics == nil {
		http.Error(w, "metrics not found", http.StatusNotFound)
		return
	}

	respondJSON(w, http.StatusOK, GetChainMetricsResponse{
		ChainID:           metrics.ChainID,
		TotalExecutions:   metrics.TotalExecutions,
		P50DurationMs:     metrics.P50DurationMs,
		P75DurationMs:     metrics.P75DurationMs,
		P95DurationMs:     metrics.P95DurationMs,
		P99DurationMs:     metrics.P99DurationMs,
		MinDurationMs:     metrics.MinDurationMs,
		MaxDurationMs:     metrics.MaxDurationMs,
		StdDevDurationMs:  metrics.StdDevDurationMs,
		SuccessRate99th:   metrics.SuccessRate99th,
		AvgStepsNeeded:    metrics.AvgStepsNeeded,
		P95StepsNeeded:    metrics.P95StepsNeeded,
		MostCommonFailure: metrics.MostCommonFailure,
		SLACompliance:     metrics.SLACompliance,
		UpdatedAt:         metrics.UpdatedAt,
	})
}

// ListChainsResponse returns chains sorted by SLA compliance
type ListChainsResponse struct {
	Chains []GetChainMetricsResponse `json:"chains"`
	Total  int                       `json:"total"`
}

// ListChainsBySLACompliance handles GET /admin/ops/chains?sort=sla_compliance
// Returns all chains sorted by SLA compliance score (descending)
func (h *Handler) ListChainsBySLACompliance(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	// TODO: Implement limit parameter when multi-tenancy is added to metrics retrieval
	chains, err := h.store.ListChainsSortedBySLACompliance(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responses []GetChainMetricsResponse
	for _, c := range chains {
		responses = append(responses, GetChainMetricsResponse{
			ChainID:           c.ChainID,
			TotalExecutions:   c.TotalExecutions,
			P50DurationMs:     c.P50DurationMs,
			P75DurationMs:     c.P75DurationMs,
			P95DurationMs:     c.P95DurationMs,
			P99DurationMs:     c.P99DurationMs,
			MinDurationMs:     c.MinDurationMs,
			MaxDurationMs:     c.MaxDurationMs,
			StdDevDurationMs:  c.StdDevDurationMs,
			SuccessRate99th:   c.SuccessRate99th,
			AvgStepsNeeded:    c.AvgStepsNeeded,
			P95StepsNeeded:    c.P95StepsNeeded,
			MostCommonFailure: c.MostCommonFailure,
			SLACompliance:     c.SLACompliance,
			UpdatedAt:         c.UpdatedAt,
		})
	}

	respondJSON(w, http.StatusOK, ListChainsResponse{
		Chains: responses,
		Total:  len(responses),
	})
}

// ========== Priority Queue Management Endpoints ==========

// CreateQueueRequest is the request body for creating a priority execution queue
type CreateQueueRequest struct {
	IncidentID uuid.UUID   `json:"incident_id"`
	ChainIDs   []uuid.UUID `json:"chain_ids"` // Chains to include in queue
}

// CreateQueueResponse returns the created queue
type CreateQueueResponse struct {
	ExecutionID     uuid.UUID `json:"execution_id"`
	Status          string    `json:"status"`
	CurrentChainIdx int       `json:"current_chain_idx"`
	TotalChains     int       `json:"total_chains"`
	StartedAt       time.Time `json:"started_at"`
}

// CreateChainExecutionQueue handles POST /admin/ops/chain-queues
// Creates a priority-based execution queue for multiple chains
func (h *Handler) CreateChainExecutionQueue(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	var req CreateQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.IncidentID == uuid.Nil {
		http.Error(w, "incident_id is required", http.StatusBadRequest)
		return
	}

	if len(req.ChainIDs) == 0 {
		http.Error(w, "at least one chain_id is required", http.StatusBadRequest)
		return
	}

	// Create execution queue
	chainsJSON, _ := json.Marshal(req.ChainIDs)
	execution := &ChainPriorityExecution{
		ID:              uuid.New(),
		TenantID:        tenantID,
		IncidentID:      req.IncidentID,
		ChainsToExecute: string(chainsJSON),
		ExecutionOrder:  string(chainsJSON), // Initial order = priority order
		CurrentChainIdx: 0,
		Status:          "pending",
		CompletedChains: "[]",
		FailedChains:    "[]",
		StartedAt:       time.Now().UTC(),
		CreatedAt:       time.Now().UTC(),
		UpdatedAt:       time.Now().UTC(),
	}

	if err := h.store.InsertChainPriorityExecution(r.Context(), execution); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, CreateQueueResponse{
		ExecutionID:     execution.ID,
		Status:          execution.Status,
		CurrentChainIdx: execution.CurrentChainIdx,
		TotalChains:     len(req.ChainIDs),
		StartedAt:       execution.StartedAt,
	})
}

// GetQueueResponse returns queue execution status
type GetQueueResponse struct {
	ExecutionID     uuid.UUID  `json:"execution_id"`
	TenantID        uuid.UUID  `json:"tenant_id"`
	IncidentID      uuid.UUID  `json:"incident_id"`
	Status          string     `json:"status"`
	CurrentChainIdx int        `json:"current_chain_idx"`
	TotalChains     int        `json:"total_chains"`
	CompletedCount  int        `json:"completed_count"`
	FailedCount     int        `json:"failed_count"`
	StartedAt       time.Time  `json:"started_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	ProgressPercent float64    `json:"progress_percent"`
}

// GetChainExecutionQueue handles GET /admin/ops/chain-queues/{executionID}
func (h *Handler) GetChainExecutionQueue(w http.ResponseWriter, r *http.Request) {
	executionIDStr := chi.URLParam(r, "executionID")

	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		http.Error(w, "invalid execution_id", http.StatusBadRequest)
		return
	}

	execution, err := h.store.GetChainPriorityExecution(r.Context(), executionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if execution == nil {
		http.Error(w, "execution queue not found", http.StatusNotFound)
		return
	}

	// Parse completed and failed chains to count them
	var completed, failed []string
	_ = json.Unmarshal([]byte(execution.CompletedChains), &completed)
	_ = json.Unmarshal([]byte(execution.FailedChains), &failed)

	// Parse total chains
	var chainsToExecute []string
	_ = json.Unmarshal([]byte(execution.ChainsToExecute), &chainsToExecute)

	progressPercent := 0.0
	if len(chainsToExecute) > 0 {
		progressPercent = float64(len(completed)+len(failed)) / float64(len(chainsToExecute)) * 100
	}

	respondJSON(w, http.StatusOK, GetQueueResponse{
		ExecutionID:     execution.ID,
		TenantID:        execution.TenantID,
		IncidentID:      execution.IncidentID,
		Status:          execution.Status,
		CurrentChainIdx: execution.CurrentChainIdx,
		TotalChains:     len(chainsToExecute),
		CompletedCount:  len(completed),
		FailedCount:     len(failed),
		StartedAt:       execution.StartedAt,
		CompletedAt:     execution.CompletedAt,
		ProgressPercent: progressPercent,
	})
}

// UpdateQueueRequest is the request body for advancing queue execution
type UpdateQueueRequest struct {
	Status          string   `json:"status,omitempty"` // "in_progress", "completed", "failed"
	CurrentChainIdx int      `json:"current_chain_idx,omitempty"`
	CompletedChains []string `json:"completed_chains,omitempty"`
	FailedChains    []string `json:"failed_chains,omitempty"`
}

// UpdateChainExecutionQueue handles PUT /admin/ops/chain-queues/{executionID}
// Advances execution progress through the chain queue
func (h *Handler) UpdateChainExecutionQueue(w http.ResponseWriter, r *http.Request) {
	executionIDStr := chi.URLParam(r, "executionID")

	executionID, err := uuid.Parse(executionIDStr)
	if err != nil {
		http.Error(w, "invalid execution_id", http.StatusBadRequest)
		return
	}

	var req UpdateQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}

	if err := h.store.UpdateChainPriorityExecution(r.Context(), executionID, req.CurrentChainIdx, req.Status, req.CompletedChains, req.FailedChains); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "queue updated",
		"status":  req.Status,
	})
}

// ListQueuesResponse returns pending execution queues
type ListQueuesResponse struct {
	Queues []GetQueueResponse `json:"queues"`
	Total  int                `json:"total"`
}

// ListPendingChainQueues handles GET /admin/ops/chain-queues?status=pending
// Lists pending chain execution queues for the current tenant
func (h *Handler) ListPendingChainQueues(w http.ResponseWriter, r *http.Request) {
	tenantID := r.Context().Value("tenant_id").(uuid.UUID)

	queues, err := h.store.ListPendingChainQueues(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var responses []GetQueueResponse
	for _, q := range queues {
		var completed, failed []string
		_ = json.Unmarshal([]byte(q.CompletedChains), &completed)
		_ = json.Unmarshal([]byte(q.FailedChains), &failed)

		var chainsToExecute []string
		_ = json.Unmarshal([]byte(q.ChainsToExecute), &chainsToExecute)

		progressPercent := 0.0
		if len(chainsToExecute) > 0 {
			progressPercent = float64(len(completed)+len(failed)) / float64(len(chainsToExecute)) * 100
		}

		responses = append(responses, GetQueueResponse{
			ExecutionID:     q.ID,
			TenantID:        q.TenantID,
			IncidentID:      q.IncidentID,
			Status:          q.Status,
			CurrentChainIdx: q.CurrentChainIdx,
			TotalChains:     len(chainsToExecute),
			CompletedCount:  len(completed),
			FailedCount:     len(failed),
			StartedAt:       q.StartedAt,
			CompletedAt:     q.CompletedAt,
			ProgressPercent: progressPercent,
		})
	}

	respondJSON(w, http.StatusOK, ListQueuesResponse{
		Queues: responses,
		Total:  len(responses),
	})
}
