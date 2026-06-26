package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
)

type UpgradeHandler struct {
	svc              *services.UpgradeRuntimeService
	errorCount       int64
	lastErrorTime    int64
	circuitOpen      int32
	circuitOpenTime  int64
	failureThreshold int64
	recoveryTimeout  time.Duration
}

func NewUpgradeHandler(s *services.UpgradeRuntimeService) *UpgradeHandler {
	return &UpgradeHandler{
		svc:              s,
		failureThreshold: 5,                // Open circuit after 5 failures
		recoveryTimeout:  30 * time.Second, // Try to close circuit after 30 seconds
	}
}

// RegisterRoutes mounts upgrade routes
func (h *UpgradeHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/upgrades", func(r chi.Router) {
		r.Get("/versions", h.ListVersions)
		r.Post("/preview", h.SetPreview)
		r.Post("/canary", h.StartCanary)
		r.Post("/activate", h.Activate)
		r.Post("/rollback", h.Rollback)
		r.Get("/diff", h.GetDiff)
		r.Get("/notifications", h.ListNotifications)
		r.Post("/prepare", h.PrepareUpgrade)
		r.Post("/diff-report", h.GenerateDiffReport)
		r.Get("/alias-map", h.GetAliasMap)
		r.Post("/alias-map", h.GenerateAliasMap)
		r.Post("/fixes/analyze", h.AnalyzeExtensionFixes)
		r.Post("/fixes/apply", h.ApplyExtensionFixesV2)
		r.Post("/fixes/preview", h.PreviewExtensionFixes)
		r.Get("/golden-queries", h.ListGoldenQueries)
		r.Post("/golden-queries", h.AddGoldenQuery)
		r.Post("/golden-queries/run", h.RunGoldenQueries)
		r.Get("/schema/version", h.GetSchemaVersion)
		r.Get("/overview", h.GetUpgradeOverview)
		r.Get("/overview/multi", h.GetMultiUpgradeOverview)

		r.Route("/{version}", func(r chi.Router) {
			r.Get("/broken-references", h.ListBrokenRefs)
			r.Post("/fixes", h.ApplyExtensionFixes)
			r.Post("/preview/run", h.RunPreview)
			r.Post("/generate", h.GenerateCore)
			r.Post("/merge", h.MergeCustom)
			r.Post("/validate", h.Validate)
			r.Post("/shadow", h.RunShadow)
			r.Get("/validation-report", h.GetValidationReport)
			r.Post("/archive", h.Archive)
			r.Get("/schema-changes", h.GetSchemaChanges)
			r.Get("/deprecation-map", h.GetDeprecationMap)
			r.Get("/pre-agg-rebuild", h.GetPreAggRebuild)
			r.Put("/golden-queries", h.UpdateGoldenQuery)
			r.Delete("/golden-queries", h.DeleteGoldenQuery)
		})
	})
}

// isCircuitBreakerOpen checks if the circuit breaker is open
func (h *UpgradeHandler) isCircuitBreakerOpen() bool {
	if atomic.LoadInt32(&h.circuitOpen) == 1 {
		// Check if recovery timeout has passed
		if time.Now().Unix()-atomic.LoadInt64(&h.circuitOpenTime) > int64(h.recoveryTimeout.Seconds()) {
			// Try to close the circuit
			atomic.StoreInt32(&h.circuitOpen, 0)
			atomic.StoreInt64(&h.errorCount, 0)
			return false
		}
		return true
	}
	return false
}

// recordError records an error and potentially opens the circuit breaker
func (h *UpgradeHandler) recordError(_ string) {
	atomic.AddInt64(&h.errorCount, 1)
	atomic.StoreInt64(&h.lastErrorTime, time.Now().Unix())

	if atomic.LoadInt64(&h.errorCount) >= h.failureThreshold {
		atomic.StoreInt32(&h.circuitOpen, 1)
		atomic.StoreInt64(&h.circuitOpenTime, time.Now().Unix())
	}
}

// recordSuccess records a successful operation and resets error count
func (h *UpgradeHandler) recordSuccess() {
	atomic.StoreInt64(&h.errorCount, 0)
}

func (h *UpgradeHandler) ListVersions(w http.ResponseWriter, r *http.Request) {
	versions, canary, slo := h.svc.ListVersions()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"versions": versions, "canary": canary, "slo": slo})
}

func (h *UpgradeHandler) SetPreview(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "version is required"})
		return
	}
	if err := h.svc.SetPreview(req.Version); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (h *UpgradeHandler) StartCanary(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CoreVersion string   `json:"coreVersion"`
		Tenants     []string `json:"tenants"`
	}

	// Parse and validate request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if req.CoreVersion == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "coreVersion is required"})
		return
	}

	// Check circuit breaker
	if h.isCircuitBreakerOpen() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "Service temporarily unavailable",
			"retryAfter": 30,
		})
		return
	}

	// 1. Validate version exists with detailed error
	if !h.svc.VersionExists(req.CoreVersion) {
		h.recordError("version_not_found")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Version not found",
			"version": req.CoreVersion,
		})
		return
	}

	// 2. Assign tenants to canary version with error handling
	if err := h.svc.AssignTenantsToVersion(req.Tenants, req.CoreVersion); err != nil {
		h.recordError("tenant_assignment_failed")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Failed to assign tenants to canary",
			"details": err.Error(),
		})
		return
	}

	// 3. Warm caches for canary tenants with timeout
	cacheCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		h.svc.WarmCachesForTenants(req.Tenants, req.CoreVersion)
		done <- nil
	}()

	select {
	case <-cacheCtx.Done():
		h.recordError("cache_warming_timeout")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusRequestTimeout)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   "Cache warming timed out",
			"tenants": req.Tenants,
		})
		return
	case err := <-done:
		if err != nil {
			h.recordError("cache_warming_failed")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   "Cache warming failed",
				"details": err.Error(),
			})
			return
		}
	}

	// 4. Audit log
	user := r.Header.Get("X-User")
	if user == "" {
		user = "system"
	}
	h.svc.LogUpgradeAction("canary", req.CoreVersion, time.Now(), user)

	// 5. Broadcast to UI
	h.svc.BroadcastStatusChange(req.CoreVersion, "canary")

	h.recordSuccess()
	w.WriteHeader(http.StatusNoContent)
}

func (h *UpgradeHandler) Activate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CoreVersion string `json:"coreVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CoreVersion == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "coreVersion is required"})
		return
	}

	// 1. Validate version exists
	if !h.svc.VersionExists(req.CoreVersion) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "version not found"})
		return
	}

	// 2. Update active version in registry
	if err := h.svc.SetActiveVersion(req.CoreVersion); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "failed to activate: " + err.Error()})
		return
	}

	// 3. Trigger post-activation jobs
	h.svc.WarmCachesForAllTenants(req.CoreVersion)
	h.svc.RebuildPreAggsIfNeeded(req.CoreVersion)

	// 4. Audit log
	user := r.Header.Get("X-User")
	if user == "" {
		user = "system"
	}
	h.svc.LogUpgradeAction("activate", req.CoreVersion, time.Now(), user)

	// 5. Broadcast to UI
	h.svc.BroadcastStatusChange(req.CoreVersion, "active")

	w.WriteHeader(http.StatusNoContent)
}

func (h *UpgradeHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CoreVersion string `json:"coreVersion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CoreVersion == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "coreVersion is required"})
		return
	}

	// 1. Get previous active version
	prev := h.svc.GetPreviousActiveVersion()
	if prev == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "no previous version to rollback to"})
		return
	}

	// 2. Get tenants currently on the version being rolled back
	tenants := h.svc.GetTenantsOnVersion(req.CoreVersion)

	// 3. Assign tenants back to previous version
	if err := h.svc.AssignTenantsToVersion(tenants, prev); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "failed to rollback: " + err.Error()})
		return
	}

	// 4. Audit log
	user := r.Header.Get("X-User")
	if user == "" {
		user = "system"
	}
	h.svc.LogUpgradeAction("rollback", req.CoreVersion, time.Now(), user)

	// 5. Broadcast to UI
	h.svc.BroadcastStatusChange(req.CoreVersion, "rolled_back")

	w.WriteHeader(http.StatusNoContent)
}

func (h *UpgradeHandler) GetDiff(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "from and to are required"})
		return
	}
	rep, err := h.svc.GetDiff(from, to)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rep)
}

func (h *UpgradeHandler) ListBrokenRefs(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	refs, err := h.svc.GetBrokenReferences(version)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(refs)
}

func (h *UpgradeHandler) ApplyFixes(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	var patches map[string]string
	if err := json.NewDecoder(r.Body).Decode(&patches); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid patches"})
		return
	}
	if err := h.svc.ApplyExtensionFixes(version, patches); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (h *UpgradeHandler) RunPreview(w http.ResponseWriter, r *http.Request) {
	var req services.PreviewRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid request"})
		return
	}
	res, err := h.svc.RunPreview(req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *UpgradeHandler) ListNotifications(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.svc.ListNotifications())
}

// New lifecycle handlers
func (h *UpgradeHandler) PrepareUpgrade(w http.ResponseWriter, r *http.Request) {
	var req struct {
		NewVersion string `json:"new_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewVersion == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "new_version is required"})
		return
	}
	hash, changes, depMap, err := h.svc.PrepareUpgrade(req.NewVersion)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"schema_hash": hash, "changes": changes, "deprecation_map": depMap})
}

func (h *UpgradeHandler) GenerateCore(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	if err := h.svc.GenerateCore(version); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (h *UpgradeHandler) MergeCustom(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	if err := h.svc.MergeCustom(version); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (h *UpgradeHandler) Validate(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	report, err := h.svc.Validate(version)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (h *UpgradeHandler) RunShadow(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	var req struct {
		Queries []string `json:"queries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid request"})
		return
	}
	results, err := h.svc.RunShadow(version, req.Queries)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *UpgradeHandler) GetValidationReport(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	report, err := h.svc.GetValidationReport(version)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (h *UpgradeHandler) Archive(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	if err := h.svc.Archive(version); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (h *UpgradeHandler) GetSchemaChanges(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	changes, err := h.svc.GetSchemaChanges(version)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(changes)
}

func (h *UpgradeHandler) GetDeprecationMap(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	dep, err := h.svc.GetDeprecationMap(version)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dep)
}

func (h *UpgradeHandler) GetPreAggRebuild(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	rebuild, err := h.svc.GetPreAggRebuild(version)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rebuild)
}

// Diff Report and Alias Map handlers
func (h *UpgradeHandler) GenerateDiffReport(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromVersion string `json:"from_version"`
		ToVersion   string `json:"to_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FromVersion == "" || req.ToVersion == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "from_version and to_version are required"})
		return
	}
	report, err := h.svc.GenerateDiffReport(req.FromVersion, req.ToVersion)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (h *UpgradeHandler) GetAliasMap(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	if from == "" || to == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "from and to are required"})
		return
	}
	aliasMap, err := h.svc.GetAliasMap(from, to)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(aliasMap)
}

func (h *UpgradeHandler) GenerateAliasMap(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromVersion string `json:"from_version"`
		ToVersion   string `json:"to_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FromVersion == "" || req.ToVersion == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "from_version and to_version are required"})
		return
	}
	aliasMap, err := h.svc.GenerateAliasMap(req.FromVersion, req.ToVersion)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(aliasMap)
}

// Extension Fix handlers
func (h *UpgradeHandler) AnalyzeExtensionFixes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version string   `json:"version"`
		Files   []string `json:"files"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "version is required"})
		return
	}
	fixes, err := h.svc.AnalyzeExtensionFixes(req.Version, req.Files)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(fixes)
}

// Rename ApplyFixes to ApplyExtensionFixes (Batch 1 ref)
func (h *UpgradeHandler) ApplyExtensionFixes(w http.ResponseWriter, r *http.Request) {
	version := chi.URLParam(r, "version")
	var patches map[string]string
	if err := json.NewDecoder(r.Body).Decode(&patches); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid patches"})
		return
	}
	if err := h.svc.ApplyExtensionFixes(version, patches); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (h *UpgradeHandler) ApplyExtensionFixesV2(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version string                  `json:"version"`
		Fixes   []services.ExtensionFix `json:"fixes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid request"})
		return
	}
	result, err := h.svc.ApplyExtensionFixesV2(req.Version, req.Fixes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *UpgradeHandler) PreviewExtensionFixes(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Version string                  `json:"version"`
		Fixes   []services.ExtensionFix `json:"fixes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Version == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid request"})
		return
	}
	result, err := h.svc.PreviewExtensionFixes(req.Version, req.Fixes)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// Golden Query handlers
func (h *UpgradeHandler) ListGoldenQueries(w http.ResponseWriter, r *http.Request) {
	queries, err := h.svc.ListGoldenQueries()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(queries)
}

func (h *UpgradeHandler) AddGoldenQuery(w http.ResponseWriter, r *http.Request) {
	var query struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Query       string   `json:"query"`
		Tags        []string `json:"tags,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil || query.Name == "" || query.Query == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "name and query are required"})
		return
	}
	if query.Tags == nil {
		query.Tags = []string{"default"}
	}
	addedQuery, err := h.svc.AddGoldenQuery(query.Name, query.Description, query.Query, query.Tags)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(addedQuery)
}

func (h *UpgradeHandler) RunGoldenQueries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		FromVersion string   `json:"from_version"`
		ToVersion   string   `json:"to_version"`
		Queries     []string `json:"queries,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FromVersion == "" || req.ToVersion == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "from_version and to_version are required"})
		return
	}
	results, err := h.svc.RunGoldenQueries(req.FromVersion, req.ToVersion, req.Queries)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (h *UpgradeHandler) UpdateGoldenQuery(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	var updates struct {
		Description *string  `json:"description,omitempty"`
		Query       *string  `json:"query,omitempty"`
		Tags        []string `json:"tags,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "invalid request"})
		return
	}
	updatedQuery, err := h.svc.UpdateGoldenQuery(name, updates.Description, updates.Query, updates.Tags)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updatedQuery)
}

func (h *UpgradeHandler) DeleteGoldenQuery(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := h.svc.DeleteGoldenQuery(name); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok"})
}

func (h *UpgradeHandler) GetSchemaVersion(w http.ResponseWriter, r *http.Request) {
	version, err := h.svc.GetSchemaVersion()
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"schema_version": version})
}

// New overview handlers
func (h *UpgradeHandler) GetUpgradeOverview(w http.ResponseWriter, r *http.Request) {
	coreVersion := r.URL.Query().Get("coreVersion")
	if coreVersion == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "coreVersion parameter is required"})
		return
	}

	overview, err := h.svc.GetUpgradeOverview(coreVersion)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}

func (h *UpgradeHandler) GetMultiUpgradeOverview(w http.ResponseWriter, r *http.Request) {
	versionsParam := r.URL.Query().Get("coreVersions")
	statusParam := r.URL.Query().Get("status")
	sortParam := r.URL.Query().Get("sort")

	var coreVersions []string
	if versionsParam != "" {
		coreVersions = strings.Split(versionsParam, ",")
		// Trim whitespace from each version
		for i, v := range coreVersions {
			coreVersions[i] = strings.TrimSpace(v)
		}
	}

	var statusFilter []string
	if statusParam != "" {
		statusFilter = strings.Split(statusParam, ",")
		// Trim whitespace and convert to lowercase
		for i, s := range statusFilter {
			statusFilter[i] = strings.ToLower(strings.TrimSpace(s))
		}
	}

	overview, err := h.svc.GetMultiUpgradeOverview(coreVersions, statusFilter, sortParam)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overview)
}
