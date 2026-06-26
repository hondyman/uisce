package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/pkg/semantic"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// SemanticLayerHandler handles semantic layer API requests
type SemanticLayerHandler struct {
	service      *semantic.Service
	engine       *semantic.QueryEngine
	analyticsSvc *analytics.SemanticService
	server       *Server
}

// NewSemanticLayerHandler creates a new semantic layer handler
func NewSemanticLayerHandler(service *semantic.Service, analyticsSvc *analytics.SemanticService, server *Server) *SemanticLayerHandler {
	return &SemanticLayerHandler{
		service:      service,
		engine:       semantic.NewQueryEngine(service),
		analyticsSvc: analyticsSvc,
		server:       server,
	}
}

// ListCubes returns all cubes for a tenant
// GET /api/semantic/cubes
func (h *SemanticLayerHandler) ListCubes(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	cubes, err := h.service.ListCubes(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cubes)
}

// GetCube returns a specific cube
// GET /api/semantic/cubes/{name}
func (h *SemanticLayerHandler) GetCube(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	cubeName := chi.URLParam(r, "name")
	if cubeName == "" {
		http.Error(w, "cube name required", http.StatusBadRequest)
		return
	}

	cube, err := h.service.GetCube(r.Context(), tenantID, cubeName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cube)
}

// CreateCube creates a new cube
// POST /api/semantic/cubes
func (h *SemanticLayerHandler) CreateCube(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var cube semantic.Cube
	if err := json.NewDecoder(r.Body).Decode(&cube); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cube.TenantID = tenantID

	if err := h.service.CreateCube(r.Context(), &cube); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(cube)
}

// UpdateCube updates an existing cube
// PUT /api/semantic/cubes/{name}
func (h *SemanticLayerHandler) UpdateCube(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	cubeName := chi.URLParam(r, "name")
	if cubeName == "" {
		http.Error(w, "cube name required", http.StatusBadRequest)
		return
	}

	// Get existing cube to ensure it exists and get ID
	existingCube, err := h.service.GetCube(r.Context(), tenantID, cubeName)
	if err != nil {
		http.Error(w, "Cube not found", http.StatusNotFound)
		return
	}

	var cube semantic.Cube
	if err := json.NewDecoder(r.Body).Decode(&cube); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	cube.ID = existingCube.ID
	cube.TenantID = tenantID
	// Name in URL overrides name in body if present, or ensures consistency
	cube.Name = cubeName

	if err := h.service.UpdateCube(r.Context(), &cube); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(cube)
}

// ExecuteQuery executes a semantic query
// POST /api/semantic/query
func (h *SemanticLayerHandler) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var query semantic.Query
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.engine.ExecuteQuery(r.Context(), tenantID, &query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GenerateSQL generates SQL from a semantic query without executing
// POST /api/semantic/query/sql
func (h *SemanticLayerHandler) GenerateSQL(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var query semantic.Query
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	sql, annotation, err := h.engine.GenerateSQL(r.Context(), tenantID, &query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"sql":        sql,
		"annotation": annotation,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// PlanHandler proxies planner-only semantics: POST /api/semantic/plan
// It returns the raw SemanticQuery produced by the planner. Enforces that
// the planner includes a top-level `region` equal to the request region and
// validates a region-scoped snapshot exists for the tenant.
func (h *SemanticLayerHandler) PlanHandler(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	// Ensure region header present
	region := r.Header.Get("X-Tenant-Region")
	if region == "" {
		http.Error(w, "X-Tenant-Region header required", http.StatusBadRequest)
		return
	}

	var req struct {
		Datasource string `json:"datasource"`
		Version    string `json:"version,omitempty"`
		Prompt     string `json:"prompt"`
		Mode       string `json:"mode,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if req.Datasource == "" || req.Prompt == "" {
		http.Error(w, "datasource and prompt are required", http.StatusBadRequest)
		return
	}

	// Load bundle (region-scoped) via the LLM gateway so snapshots are validated
	if h.server == nil {
		http.Error(w, "server not initialized", http.StatusInternalServerError)
		return
	}
	gateway := NewLLMGateway(h.server)

	bundle, err := gateway.loadSemanticBundle(r.Context(), tenantID, req.Datasource, region, req.Version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Call planner LLM
	semQuery, err := gateway.callPlannerLLM(r.Context(), bundle, req.Prompt, req.Mode, region)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Validate
	if semQuery.Region == "" {
		http.Error(w, "region is required for all semantic operations.", http.StatusBadRequest)
		return
	}
	if semQuery.Region != region {
		http.Error(w, "planner returned a different region than request region", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(semQuery)
}

// GetQueryHistory returns query execution history
// GET /api/semantic/analytics/history
func (h *SemanticLayerHandler) GetQueryHistory(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	limit := 100 // Default limit

	history, err := h.service.GetQueryHistory(r.Context(), tenantID, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"history": history,
		"count":   len(history),
	})
}

// GetPerformanceMetrics returns query performance metrics
// GET /api/semantic/analytics/performance
func (h *SemanticLayerHandler) GetPerformanceMetrics(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	// Get recent query history for metrics
	history, err := h.service.GetQueryHistory(r.Context(), tenantID, 1000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate metrics
	var totalQueries int
	var cachedQueries int
	var totalExecutionTime int64
	var avgExecutionTime float64

	for _, h := range history {
		totalQueries++
		if h.CacheHit {
			cachedQueries++
		}
		totalExecutionTime += int64(h.ExecutionTimeMs)
	}

	if totalQueries > 0 {
		avgExecutionTime = float64(totalExecutionTime) / float64(totalQueries)
	}

	cacheHitRate := 0.0
	if totalQueries > 0 {
		cacheHitRate = float64(cachedQueries) / float64(totalQueries)
	}

	metrics := map[string]interface{}{
		"total_queries":        totalQueries,
		"cached_queries":       cachedQueries,
		"cache_hit_rate":       cacheHitRate,
		"avg_execution_time":   avgExecutionTime,
		"total_execution_time": totalExecutionTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// CreateDimension creates a new dimension for a cube
// POST /api/semantic/cubes/{name}/dimensions
func (h *SemanticLayerHandler) CreateDimension(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	cubeName := chi.URLParam(r, "name")
	cube, err := h.service.GetCube(r.Context(), tenantID, cubeName)
	if err != nil {
		http.Error(w, "Cube not found", http.StatusNotFound)
		return
	}

	var dimension semantic.Dimension
	if err := json.NewDecoder(r.Body).Decode(&dimension); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	dimension.CubeID = cube.ID

	if err := h.service.CreateDimension(r.Context(), &dimension); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate cube cache
	h.service.InvalidateCubeCache(r.Context(), tenantID, cubeName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(dimension)
}

// CreateMeasure creates a new measure for a cube
// POST /api/semantic/cubes/{name}/measures
func (h *SemanticLayerHandler) CreateMeasure(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	cubeName := chi.URLParam(r, "name")
	cube, err := h.service.GetCube(r.Context(), tenantID, cubeName)
	if err != nil {
		http.Error(w, "Cube not found", http.StatusNotFound)
		return
	}

	var measure semantic.Measure
	if err := json.NewDecoder(r.Body).Decode(&measure); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	measure.CubeID = cube.ID

	if err := h.service.CreateMeasure(r.Context(), &measure); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Invalidate cube cache
	h.service.InvalidateCubeCache(r.Context(), tenantID, cubeName)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(measure)
}

// GetBundle returns a semantic bundle for a domain
// GET /api/semantic/bundles/{domain}
func (h *SemanticLayerHandler) GetBundle(w http.ResponseWriter, r *http.Request) {
	if h.server != nil {
		h.server.getBundleByDomain(w, r)
		return
	}
	http.Error(w, "server instance not available in handler", http.StatusInternalServerError)
}
