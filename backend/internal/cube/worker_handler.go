package cube

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// WorkerHandler provides HTTP handlers for worker management
type WorkerHandler struct {
	service *WorkerService
}

// NewWorkerHandler creates a new worker handler
func NewWorkerHandler(service *WorkerService) *WorkerHandler {
	return &WorkerHandler{service: service}
}

// RegisterRoutes registers worker management routes
func (h *WorkerHandler) RegisterRoutes(r chi.Router) {
	// Worker Pools
	r.Route("/worker-pools", func(r chi.Router) {
		r.Get("/", h.ListWorkerPools)
		r.Get("/{poolId}", h.GetWorkerPool)
		r.Post("/{poolId}/scale", h.ScaleWorkerPool)
		r.Get("/{poolId}/workers", h.ListWorkerInstances)
	})

	// Worker Registration (for workers to self-register)
	r.Route("/workers", func(r chi.Router) {
		r.Post("/register", h.RegisterWorker)
		r.Post("/{workerId}/heartbeat", h.WorkerHeartbeat)
		r.Delete("/{workerId}", h.DeregisterWorker)
		r.Post("/{workerId}/claim-job", h.ClaimJob)
		r.Post("/{workerId}/complete-job/{jobId}", h.CompleteJob)
		r.Post("/{workerId}/fail-job/{jobId}", h.FailJob)
	})

	// Pre-Aggregation Definitions
	r.Route("/preagg-definitions", func(r chi.Router) {
		r.Get("/", h.ListPreAggDefinitions)
		r.Post("/", h.CreatePreAggDefinition)
		r.Get("/{defId}", h.GetPreAggDefinition)
		r.Put("/{defId}", h.UpdatePreAggDefinition)
		r.Delete("/{defId}", h.DeletePreAggDefinition)
		r.Get("/{defId}/partitions", h.ListPartitions)
		r.Post("/{defId}/build", h.TriggerBuild)
	})

	// Job Queue
	r.Route("/jobs", func(r chi.Router) {
		r.Get("/", h.ListJobs)
		r.Get("/stats", h.GetJobQueueStats)
		r.Get("/{jobId}", h.GetJob)
		r.Post("/{jobId}/cancel", h.CancelJob)
		r.Post("/{jobId}/retry", h.RetryJob)
	})
}

// ListWorkerPools handles GET /worker-pools
func (h *WorkerHandler) ListWorkerPools(w http.ResponseWriter, r *http.Request) {
	pools, err := h.service.ListWorkerPools(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, pools)
}

// GetWorkerPool handles GET /worker-pools/{poolId}
func (h *WorkerHandler) GetWorkerPool(w http.ResponseWriter, r *http.Request) {
	poolID, err := uuid.Parse(chi.URLParam(r, "poolId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool ID"})
		return
	}

	pool, err := h.service.GetWorkerPool(r.Context(), poolID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if pool == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "pool not found"})
		return
	}
	writeJSON(w, http.StatusOK, pool)
}

// ScaleWorkerPool handles POST /worker-pools/{poolId}/scale
func (h *WorkerHandler) ScaleWorkerPool(w http.ResponseWriter, r *http.Request) {
	poolID, err := uuid.Parse(chi.URLParam(r, "poolId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool ID"})
		return
	}

	var req struct {
		TargetWorkers int `json:"target_workers"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.ScaleWorkerPool(r.Context(), poolID, req.TargetWorkers); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "scaling"})
}

// ListWorkerInstances handles GET /worker-pools/{poolId}/workers
func (h *WorkerHandler) ListWorkerInstances(w http.ResponseWriter, r *http.Request) {
	poolID, err := uuid.Parse(chi.URLParam(r, "poolId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool ID"})
		return
	}

	workers, err := h.service.ListWorkerInstances(r.Context(), poolID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, workers)
}

// RegisterWorker handles POST /workers/register
func (h *WorkerHandler) RegisterWorker(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PoolID     string `json:"pool_id"`
		InstanceID string `json:"instance_id"`
		Hostname   string `json:"hostname"`
		IPAddress  string `json:"ip_address"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	poolID, err := uuid.Parse(req.PoolID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pool ID"})
		return
	}

	worker, err := h.service.RegisterWorker(r.Context(), poolID, req.InstanceID, req.Hostname, req.IPAddress)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, worker)
}

// WorkerHeartbeat handles POST /workers/{workerId}/heartbeat
func (h *WorkerHandler) WorkerHeartbeat(w http.ResponseWriter, r *http.Request) {
	workerID, err := uuid.Parse(chi.URLParam(r, "workerId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid worker ID"})
		return
	}

	var req struct {
		Status     string  `json:"status"`
		MemoryMB   int     `json:"memory_mb"`
		CPUPercent float64 `json:"cpu_percent"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.UpdateWorkerHeartbeat(r.Context(), workerID, req.Status, req.MemoryMB, req.CPUPercent); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// DeregisterWorker handles DELETE /workers/{workerId}
func (h *WorkerHandler) DeregisterWorker(w http.ResponseWriter, r *http.Request) {
	workerID, err := uuid.Parse(chi.URLParam(r, "workerId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid worker ID"})
		return
	}

	if err := h.service.DeregisterWorker(r.Context(), workerID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ClaimJob handles POST /workers/{workerId}/claim-job
func (h *WorkerHandler) ClaimJob(w http.ResponseWriter, r *http.Request) {
	workerID, err := uuid.Parse(chi.URLParam(r, "workerId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid worker ID"})
		return
	}

	var req struct {
		PoolID string `json:"pool_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	poolID, _ := uuid.Parse(req.PoolID)

	job, err := h.service.ClaimJob(r.Context(), workerID, poolID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if job == nil {
		writeJSON(w, http.StatusNoContent, nil)
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// CompleteJob handles POST /workers/{workerId}/complete-job/{jobId}
func (h *WorkerHandler) CompleteJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(chi.URLParam(r, "jobId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job ID"})
		return
	}

	var req struct {
		RowsProcessed int64           `json:"rows_processed"`
		BytesWritten  int64           `json:"bytes_written"`
		Metadata      json.RawMessage `json:"metadata"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.CompleteJob(r.Context(), jobID, req.RowsProcessed, req.BytesWritten, req.Metadata); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "completed"})
}

// FailJob handles POST /workers/{workerId}/fail-job/{jobId}
func (h *WorkerHandler) FailJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(chi.URLParam(r, "jobId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job ID"})
		return
	}

	var req struct {
		ErrorMessage string `json:"error_message"`
		ErrorStack   string `json:"error_stack"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.FailJob(r.Context(), jobID, req.ErrorMessage, req.ErrorStack); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "failed"})
}

// ListPreAggDefinitions handles GET /preagg-definitions
func (h *WorkerHandler) ListPreAggDefinitions(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(r.URL.Query().Get("tenant_id"))
	datasourceID, _ := uuid.Parse(r.URL.Query().Get("datasource_id"))

	defs, err := h.service.ListPreAggDefinitions(r.Context(), tenantID, datasourceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, defs)
}

// CreatePreAggDefinition handles POST /preagg-definitions
func (h *WorkerHandler) CreatePreAggDefinition(w http.ResponseWriter, r *http.Request) {
	var def PreAggDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.service.CreatePreAggDefinition(r.Context(), &def); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, def)
}

// GetPreAggDefinition handles GET /preagg-definitions/{defId}
func (h *WorkerHandler) GetPreAggDefinition(w http.ResponseWriter, r *http.Request) {
	defID, err := uuid.Parse(chi.URLParam(r, "defId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid definition ID"})
		return
	}

	def, err := h.service.GetPreAggDefinition(r.Context(), defID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, def)
}

// UpdatePreAggDefinition handles PUT /preagg-definitions/{defId}
func (h *WorkerHandler) UpdatePreAggDefinition(w http.ResponseWriter, r *http.Request) {
	defID, err := uuid.Parse(chi.URLParam(r, "defId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid definition ID"})
		return
	}

	var def PreAggDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}
	def.ID = defID

	if err := h.service.UpdatePreAggDefinition(r.Context(), &def); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, def)
}

// DeletePreAggDefinition handles DELETE /preagg-definitions/{defId}
func (h *WorkerHandler) DeletePreAggDefinition(w http.ResponseWriter, r *http.Request) {
	defID, err := uuid.Parse(chi.URLParam(r, "defId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid definition ID"})
		return
	}

	if err := h.service.DeletePreAggDefinition(r.Context(), defID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ListPartitions handles GET /preagg-definitions/{defId}/partitions
func (h *WorkerHandler) ListPartitions(w http.ResponseWriter, r *http.Request) {
	preAggID, err := uuid.Parse(chi.URLParam(r, "defId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pre-aggregation ID"})
		return
	}

	parts, err := h.service.ListPartitions(r.Context(), preAggID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, parts)
}

// TriggerBuild handles POST /preagg-definitions/{defId}/build
func (h *WorkerHandler) TriggerBuild(w http.ResponseWriter, r *http.Request) {
	preAggID, err := uuid.Parse(chi.URLParam(r, "defId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid pre-aggregation ID"})
		return
	}

	var req struct {
		JobType      string          `json:"job_type"`
		PartitionKey string          `json:"partition_key"`
		Priority     int             `json:"priority"`
		Options      json.RawMessage `json:"options"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	tenantID, _ := uuid.Parse(r.URL.Query().Get("tenant_id"))
	datasourceID, _ := uuid.Parse(r.URL.Query().Get("datasource_id"))

	job := &PreAggJob{
		PreAggID:     preAggID,
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		JobType:      req.JobType,
		PartitionKey: req.PartitionKey,
		Priority:     req.Priority,
		BuildOptions: req.Options,
		MaxRetries:   3,
	}

	if err := h.service.EnqueueJob(r.Context(), job); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, job)
}

// ListJobs handles GET /jobs
func (h *WorkerHandler) ListJobs(w http.ResponseWriter, r *http.Request) {
	tenantID, _ := uuid.Parse(r.URL.Query().Get("tenant_id"))
	status := r.URL.Query().Get("status")
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	jobs, err := h.service.ListJobs(r.Context(), tenantID, status, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, jobs)
}

// GetJobQueueStats handles GET /jobs/stats
func (h *WorkerHandler) GetJobQueueStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.service.GetJobQueueStats(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

// GetJob handles GET /jobs/{jobId}
func (h *WorkerHandler) GetJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(chi.URLParam(r, "jobId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job ID"})
		return
	}

	job, err := h.service.GetJob(r.Context(), jobID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, job)
}

// CancelJob handles POST /jobs/{jobId}/cancel
func (h *WorkerHandler) CancelJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(chi.URLParam(r, "jobId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job ID"})
		return
	}

	if err := h.service.CancelJob(r.Context(), jobID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// RetryJob handles POST /jobs/{jobId}/retry
func (h *WorkerHandler) RetryJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := uuid.Parse(chi.URLParam(r, "jobId"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid job ID"})
		return
	}

	if err := h.service.RetryJob(r.Context(), jobID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "retrying"})
}
