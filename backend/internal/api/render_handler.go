package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/reporting"
)

type RenderHandler struct {
	db            *sqlx.DB
	renderService *reporting.ReportRenderService
	securityDeps  handlers.SecurityContextDeps
}

func NewRenderHandler(db *sqlx.DB, rs *reporting.ReportRenderService, securityDeps handlers.SecurityContextDeps) *RenderHandler {
	return &RenderHandler{
		db:            db,
		renderService: rs,
		securityDeps:  securityDeps,
	}
}

func (h *RenderHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/v1/reports/by-key/{key}/render", h.RenderByKey)
	r.Post("/api/v1/reports/{id}/render", h.RenderByID)
}

func (h *RenderHandler) RenderByKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if key == "" {
		http.Error(w, `{"error":"report key required"}`, http.StatusBadRequest)
		return
	}
	h.handleRender(w, r, key, uuid.Nil)
}

func (h *RenderHandler) RenderByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		http.Error(w, `{"error":"report id required"}`, http.StatusBadRequest)
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid report id"}`, http.StatusBadRequest)
		return
	}

	tenantID, datasourceID, ctx := h.resolveSecurityContext(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"tenant required"}`, http.StatusBadRequest)
		return
	}

	repo := reporting.NewRepository(h.db)
	def, err := repo.GetDefinition(ctx, id)
	if err != nil || def == nil {
		http.Error(w, `{"error":"report not found"}`, http.StatusNotFound)
		return
	}

	h.handleRender(w, r, def.ReportKey, datasourceID)
}

func (h *RenderHandler) handleRender(w http.ResponseWriter, r *http.Request, reportKey string, datasourceID uuid.UUID) {
	tenantID, dsID, ctx := h.resolveSecurityContext(r)
	if tenantID == uuid.Nil {
		http.Error(w, `{"error":"tenant required"}`, http.StatusBadRequest)
		return
	}
	if datasourceID == uuid.Nil {
		datasourceID = dsID
	}

	var body struct {
		Parameters json.RawMessage `json:"parameters"`
	}
	if r.Body != nil {
		json.NewDecoder(r.Body).Decode(&body)
	}

	result, renderErr := h.renderService.RenderByKey(ctx, tenantID, datasourceID, reportKey, body.Parameters)
	if renderErr != nil {
		http.Error(w, `{"error":"`+renderErr.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *RenderHandler) resolveSecurityContext(r *http.Request) (uuid.UUID, uuid.UUID, context.Context) {
	secCtx, ctx, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil || secCtx == nil {
		tenantIDStr := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
		if tenantIDStr == "" {
			return uuid.Nil, uuid.Nil, r.Context()
		}
		tenantID, _ := uuid.Parse(tenantIDStr)
		return tenantID, uuid.Nil, r.Context()
	}

	tenantID, _ := uuid.Parse(secCtx.TenantID)
	datasourceID := uuid.Nil
	if secCtx.DatasourceID != "" {
		datasourceID, _ = uuid.Parse(secCtx.DatasourceID)
	}
	return tenantID, datasourceID, ctx
}
