package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CubeHandler provides HTTP endpoints for Cube.js semantic layer queries
type CubeHandler struct {
	cubeClient    *cube.Client
	cubeGenerator *services.CubeGenerator
}

// NewCubeHandler creates a new Cube.js API handler
func NewCubeHandler(cubeClient *cube.Client, cubeGenerator *services.CubeGenerator) *CubeHandler {
	return &CubeHandler{
		cubeClient:    cubeClient,
		cubeGenerator: cubeGenerator,
	}
}

// CubeQueryRequest represents a query request from the frontend
type CubeQueryRequest struct {
	Measures       []string             `json:"measures"`
	Dimensions     []string             `json:"dimensions"`
	Filters        []cube.Filter        `json:"filters"`
	TimeDimensions []cube.TimeDimension `json:"timeDimensions"`
	Order          map[string]string    `json:"order"`
	Limit          int                  `json:"limit"`
	Timezone       string               `json:"timezone"`
}

// CubeQueryResponse wraps the Cube.js query result
type CubeQueryResponse struct {
	Data       []map[string]interface{} `json:"data"`
	Annotation *cube.Annotation         `json:"annotation,omitempty"`
	Query      *cube.Query              `json:"query,omitempty"`
	Count      int                      `json:"count"`
}

// RegisterRoutes adds Cube.js routes to the router
func (h *CubeHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/cube", func(r chi.Router) {
		// Query routes
		r.Post("/query", h.ExecuteQuery)
		r.Get("/meta", h.GetMeta)
		r.Get("/pre-aggregations", h.GetPreAggregations)
		r.Post("/dry-run", h.DryRun)

		// Schema generation routes
		r.Post("/generate", h.GenerateCubeSchema)
		r.Post("/generate/{boID}", h.GenerateCubeFromBO)
		r.Get("/preview", h.PreviewCubeSchema)
	})
}

// ExecuteQuery handles POST /api/cube/query
func (h *CubeHandler) ExecuteQuery(w http.ResponseWriter, r *http.Request) {
	// Extract tenant context from headers (set by setupTenantFetch.ts)
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	datasourceIDStr := r.Header.Get("X-Tenant-Datasource-ID")
	userID := r.Header.Get("X-User-ID")

	if tenantIDStr == "" || datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "MISSING_TENANT_CONTEXT",
				Message: "X-Tenant-ID and X-Tenant-Datasource-ID headers are required",
			},
		})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_TENANT_ID",
				Message: "Invalid tenant ID format",
			},
		})
		return
	}

	datasourceID, err := uuid.Parse(datasourceIDStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_DATASOURCE_ID",
				Message: "Invalid datasource ID format",
			},
		})
		return
	}

	// Parse request
	var req CubeQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid query format",
				Details: err.Error(),
			},
		})
		return
	}

	// Build Cube.js query
	query := &cube.Query{
		Measures:       req.Measures,
		Dimensions:     req.Dimensions,
		Filters:        req.Filters,
		TimeDimensions: req.TimeDimensions,
		Order:          req.Order,
		Limit:          req.Limit,
		Timezone:       req.Timezone,
	}

	if query.Timezone == "" {
		query.Timezone = "UTC"
	}

	// Execute query with tenant context
	tenantCtx := cube.TenantContext{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		UserID:       userID,
	}

	result, err := h.cubeClient.ExecuteQuery(r.Context(), query, tenantCtx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "QUERY_FAILED",
				Message: "Failed to execute Cube.js query",
				Details: err.Error(),
			},
		})
		return
	}

	// Return result
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: CubeQueryResponse{
			Data:       result.Data,
			Annotation: result.Annotation,
			Query:      result.Query,
			Count:      len(result.Data),
		},
	})
}

// GetMeta handles GET /api/cube/meta
func (h *CubeHandler) GetMeta(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	datasourceIDStr := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantIDStr == "" || datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "MISSING_TENANT_CONTEXT",
				Message: "Tenant context required",
			},
		})
		return
	}

	tenantID, _ := uuid.Parse(tenantIDStr)
	datasourceID, _ := uuid.Parse(datasourceIDStr)

	tenantCtx := cube.TenantContext{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
	}

	meta, err := h.cubeClient.GetMeta(r.Context(), tenantCtx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "META_FAILED",
				Message: "Failed to retrieve metadata",
				Details: err.Error(),
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    meta,
	})
}

// GetPreAggregations handles GET /api/cube/pre-aggregations
func (h *CubeHandler) GetPreAggregations(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	datasourceIDStr := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantIDStr == "" || datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "MISSING_TENANT_CONTEXT",
				Message: "Tenant context required",
			},
		})
		return
	}

	tenantID, _ := uuid.Parse(tenantIDStr)
	datasourceID, _ := uuid.Parse(datasourceIDStr)

	tenantCtx := cube.TenantContext{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
	}

	preAggs, err := h.cubeClient.PreAggregationStatus(r.Context(), tenantCtx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "PREAGG_FAILED",
				Message: "Failed to retrieve pre-aggregation status",
				Details: err.Error(),
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    preAggs,
	})
}

// DryRun handles POST /api/cube/dry-run
func (h *CubeHandler) DryRun(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	datasourceIDStr := r.Header.Get("X-Tenant-Datasource-ID")

	if tenantIDStr == "" || datasourceIDStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "MISSING_TENANT_CONTEXT",
				Message: "Tenant context required",
			},
		})
		return
	}

	tenantID, _ := uuid.Parse(tenantIDStr)
	datasourceID, _ := uuid.Parse(datasourceIDStr)

	var req CubeQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid query format",
			},
		})
		return
	}

	query := &cube.Query{
		Measures:       req.Measures,
		Dimensions:     req.Dimensions,
		Filters:        req.Filters,
		TimeDimensions: req.TimeDimensions,
		Order:          req.Order,
		Limit:          req.Limit,
		Timezone:       req.Timezone,
	}

	tenantCtx := cube.TenantContext{
		TenantID:     tenantID,
		DatasourceID: datasourceID,
	}

	dryRunResult, err := h.cubeClient.DryRun(r.Context(), query, tenantCtx)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "DRY_RUN_FAILED",
				Message: "Dry run failed",
				Details: err.Error(),
			},
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    dryRunResult,
	})
}

// GenerateCubeSchemaRequest is the request body for schema generation
type GenerateCubeSchemaRequest struct {
	CubeName       string   `json:"cube_name"`
	TermIDs        []string `json:"term_ids"`
	Transformation string   `json:"transformation,omitempty"`
}

// GenerateCubeSchema handles POST /api/cube/generate
func (h *CubeHandler) GenerateCubeSchema(w http.ResponseWriter, r *http.Request) {
	var req GenerateCubeSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "INVALID_REQUEST",
				Message: "Invalid request body",
			},
		})
		return
	}

	if req.CubeName == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error: &APIError{
				Code:    "MISSING_CUBE_NAME",
				Message: "cube_name is required",
			},
		})
		return
	}

	// For now, return a mock response indicating the feature is available
	// Full implementation requires CubeGenerator integration
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"cube_name": req.CubeName,
			"term_ids":  req.TermIDs,
			"message":   "Schema generation endpoint configured. Full generation requires CubeGenerator integration.",
		},
	})
}

// GenerateCubeFromBO handles POST /api/cube/generate/{boID}
func (h *CubeHandler) GenerateCubeFromBO(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boID")
	if boID == "" {
		handleError(w, http.StatusBadRequest, "MISSING_BO_ID", "Business Object ID is required")
		return
	}

	model, err := h.cubeGenerator.GenerateFromBusinessObject(r.Context(), boID)
	if err != nil {
		handleError(w, http.StatusInternalServerError, "GENERATION_FAILED", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    model,
	})
}

// PreviewCubeSchema handles GET /api/cube/preview
func (h *CubeHandler) PreviewCubeSchema(w http.ResponseWriter, r *http.Request) {
	cubeName := r.URL.Query().Get("cube_name")
	if cubeName == "" {
		cubeName = "preview"
	}

	termIDs := r.URL.Query()["term_id"]

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"cube_name": cubeName,
			"term_ids":  termIDs,
			"preview":   true,
			"message":   "Preview endpoint configured.",
		},
	})
}

func handleError(w http.ResponseWriter, status int, code string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}
