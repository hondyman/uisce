package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type PreAggregationHandler struct {
	svc           *analytics.PreAggregationService
	suggestionSvc *analytics.PreAggSuggestionService
}

func NewPreAggregationHandler(svc *analytics.PreAggregationService, suggestionSvc *analytics.PreAggSuggestionService) *PreAggregationHandler {
	return &PreAggregationHandler{svc: svc, suggestionSvc: suggestionSvc}
}

func (h *PreAggregationHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/pre-aggregations", func(r chi.Router) {
		r.Post("/", h.UpsertPreAggregation)
		r.Get("/", h.ListByBO)
		r.Get("/{id}/ddl", h.GetDDL)
		r.Post("/{id}/materialize", h.ApplyMaterialization)
		r.Post("/{id}/refresh", h.Refresh)
	})

	// New /api/preaggs routes for enhanced management
	r.Route("/api/preaggs", func(r chi.Router) {
		r.Get("/", h.ListPreAggs)
		r.Post("/", h.CreatePreAgg)
		r.Get("/suggestions", h.ListSuggestions)
		r.Get("/{id}", h.GetPreAgg)
		r.Get("/{id}/sql", h.GetPreAggSQL)
		r.Put("/{id}", h.UpdatePreAgg)
		r.Delete("/{id}", h.DeletePreAgg)
		r.Post("/{id}/refresh", h.RefreshPreAgg)
		r.Post("/{id}/disable", h.DisablePreAgg)
	})
}

// UpsertPreAggregation creates or updates a pre-aggregation definition.
func (h *PreAggregationHandler) UpsertPreAggregation(w http.ResponseWriter, r *http.Request) {
	var req models.UpsertPreAggRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Use tenant from header if not in body
	if req.TenantID == "" {
		req.TenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}

	desc, err := h.svc.UpsertPreAggregation(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

// ListByBO returns pre-aggregations for a given BO and tenant.
func (h *PreAggregationHandler) ListByBO(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}
	boName := r.URL.Query().Get("bo_name")

	list, err := h.svc.ListByBO(r.Context(), tenantID, boName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// GetDDL returns the generated DDL for a pre-aggregation.
func (h *PreAggregationHandler) GetDDL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	ddl, err := h.svc.GenerateDDL(r.Context(), id, "starrocks")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(ddl))
}

// ApplyMaterialization executes the DDL in StarRocks.
func (h *PreAggregationHandler) ApplyMaterialization(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.ApplyMaterialization(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "materialized"}`))
}

// Refresh triggers a refresh of the materialized view.
func (h *PreAggregationHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.Refresh(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "refreshed"}`))
}

// --- New /api/preaggs handlers ---

// ListPreAggs returns all pre-aggregations for the tenant.
func (h *PreAggregationHandler) ListPreAggs(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)
	boName := r.URL.Query().Get("bo_name")
	datasource := r.URL.Query().Get("datasource")

	// If datasource specified, filter by datasource; otherwise use bo_name for backwards compat
	var list []models.PreAggDescriptor
	var err error

	if datasource != "" {
		list, err = h.svc.ListByDatasource(r.Context(), tenantID, datasource)
	} else {
		list, err = h.svc.ListByBO(r.Context(), tenantID, boName)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

// GetPreAgg returns a single pre-aggregation by ID.
func (h *PreAggregationHandler) GetPreAgg(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	preagg, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(preagg)
}

// CreatePreAgg creates a new pre-aggregation.
func (h *PreAggregationHandler) CreatePreAgg(w http.ResponseWriter, r *http.Request) {
	var req models.UpsertPreAggRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.getTenantID(r)
	}

	desc, err := h.svc.UpsertPreAggregation(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(desc)
}

// UpdatePreAgg updates an existing pre-aggregation.
func (h *PreAggregationHandler) UpdatePreAgg(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var req models.UpsertPreAggRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.TenantID == "" {
		req.TenantID = h.getTenantID(r)
	}

	desc, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(desc)
}

// DeletePreAgg deletes a pre-aggregation.
func (h *PreAggregationHandler) DeletePreAgg(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RefreshPreAgg triggers a refresh via the new API path.
func (h *PreAggregationHandler) RefreshPreAgg(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.Refresh(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte(`{"status": "refresh_triggered"}`))
}

// DisablePreAgg disables a pre-aggregation without deleting it.
func (h *PreAggregationHandler) DisablePreAgg(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := h.svc.Disable(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "disabled"}`))
}

// ListSuggestions returns pre-aggregation suggestions for the tenant.
func (h *PreAggregationHandler) ListSuggestions(w http.ResponseWriter, r *http.Request) {
	tenantID := h.getTenantID(r)

	if h.suggestionSvc == nil {
		// Return empty list if suggestion service not configured
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	suggestions, err := h.suggestionSvc.ListSuggestions(r.Context(), tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}

// PreAggSQLResponse contains the generated SQL for both Iceberg and StarRocks.
type PreAggSQLResponse struct {
	IcebergSQL     string `json:"iceberg_sql"`
	StarRocksMVSQL string `json:"starrocks_mv_sql"`
}

// GetPreAggSQL returns the generated DDL for a pre-aggregation.
func (h *PreAggregationHandler) GetPreAggSQL(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	preagg, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Use template renderer to generate SQL
	renderer, err := analytics.NewPreAggTemplateRenderer()
	if err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert measures to MeasureDef
	var measures []analytics.MeasureDef
	for _, m := range preagg.Measures {
		measures = append(measures, analytics.MeasureDef{
			Expression: m,
			Alias:      m,
		})
	}

	data := analytics.PreAggTemplateData{
		Tenant:     preagg.TenantID,
		Datasource: preagg.TargetDatabase,
		PreAggID:   preagg.ID.String(),
		GroupBy:    preagg.GroupBy,
		Measures:   measures,
	}

	icebergSQL, err := renderer.RenderTrinoIcebergRollup(data)
	if err != nil {
		icebergSQL = "-- Error rendering Iceberg SQL: " + err.Error()
	}

	starrocksSQL, err := renderer.RenderStarRocksMV(data)
	if err != nil {
		starrocksSQL = "-- Error rendering StarRocks SQL: " + err.Error()
	}

	resp := PreAggSQLResponse{
		IcebergSQL:     icebergSQL,
		StarRocksMVSQL: starrocksSQL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// getTenantID extracts tenant ID from request header or query param.
func (h *PreAggregationHandler) getTenantID(r *http.Request) string {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}
	if tenantID == "me" {
		// Resolve "me" from context if available
		if ctxTenant, ok := r.Context().Value("tenant_id").(string); ok {
			tenantID = ctxTenant
		}
	}
	return tenantID
}
