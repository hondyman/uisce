package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	appslos "github.com/hondyman/semlayer/backend/internal/app_platform/app_slos"
	"github.com/hondyman/semlayer/backend/internal/app_platform/governance"
	"github.com/hondyman/semlayer/backend/internal/app_platform/promotion"
	"github.com/hondyman/semlayer/backend/internal/app_platform/templates"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

type AppPlatformHandler struct {
	templateManager *templates.TemplateManager
	promotionEngine *promotion.PromotionEngine
	sloEvaluator    *appslos.AppSLOEvaluator
	appReviewer     *governance.AppReviewer
}

func NewAppPlatformHandler(
	tmpl *templates.TemplateManager,
	promo *promotion.PromotionEngine,
	slo *appslos.AppSLOEvaluator,
	rev *governance.AppReviewer,
) *AppPlatformHandler {
	return &AppPlatformHandler{
		templateManager: tmpl,
		promotionEngine: promo,
		sloEvaluator:    slo,
		appReviewer:     rev,
	}
}

func (h *AppPlatformHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// Templates
	r.Post("/templates/create", h.CreateTemplate)
	r.Get("/templates", h.ListTemplates)
	r.Post("/templates/{id}/install", h.InstallTemplate)
	r.Get("/templates/{id}/export", h.ExportTemplate)

	// Promotion
	r.Post("/promote/{appId}", h.PromoteApp)
	r.Post("/changeset/create", h.CreateChangeSet)

	// App SLOs
	r.Get("/slos/{appId}", h.GetAppSLOs)
	r.Post("/slos/create", h.CreateAppSLO)

	// Governance
	r.Post("/review/{appId}", h.ReviewApp)

	return r
}

func (h *AppPlatformHandler) CreateTemplate(w http.ResponseWriter, r *http.Request) {
	var template templates.AppTemplate
	if err := json.NewDecoder(r.Body).Decode(&template); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.templateManager.CreateTemplate(r.Context(), &template)
	json.NewEncoder(w).Encode(template)
}

func (h *AppPlatformHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	tmpls, _ := h.templateManager.ListTemplates(r.Context(), tenantID)
	json.NewEncoder(w).Encode(tmpls)
}

func (h *AppPlatformHandler) InstallTemplate(w http.ResponseWriter, r *http.Request) {
	var req templates.InstallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, _ := h.templateManager.InstallTemplate(r.Context(), &req)
	json.NewEncoder(w).Encode(result)
}

func (h *AppPlatformHandler) ExportTemplate(w http.ResponseWriter, r *http.Request) {
	appID, _ := uuid.Parse(chi.URLParam(r, "id"))
	template, _ := h.templateManager.ExportTemplate(r.Context(), appID)
	json.NewEncoder(w).Encode(template)
}

func (h *AppPlatformHandler) PromoteApp(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ChangeSetID uuid.UUID `json:"changeset_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, _ := h.promotionEngine.PromoteApp(r.Context(), body.ChangeSetID)
	json.NewEncoder(w).Encode(result)
}

func (h *AppPlatformHandler) CreateChangeSet(w http.ResponseWriter, r *http.Request) {
	var body struct {
		AppID     uuid.UUID `json:"app_id"`
		SourceEnv string    `json:"source_env"`
		TargetEnv string    `json:"target_env"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cs, _ := h.promotionEngine.CreateChangeSet(r.Context(), body.AppID, body.SourceEnv, body.TargetEnv)
	json.NewEncoder(w).Encode(cs)
}

func (h *AppPlatformHandler) GetAppSLOs(w http.ResponseWriter, r *http.Request) {
	appID, _ := uuid.Parse(chi.URLParam(r, "appId"))
	status, _ := h.sloEvaluator.EvaluateSLOs(r.Context(), appID)
	json.NewEncoder(w).Encode(status)
}

func (h *AppPlatformHandler) CreateAppSLO(w http.ResponseWriter, r *http.Request) {
	var slo appslos.AppSLO
	if err := json.NewDecoder(r.Body).Decode(&slo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.sloEvaluator.CreateSLO(r.Context(), &slo)
	json.NewEncoder(w).Encode(slo)
}

func (h *AppPlatformHandler) ReviewApp(w http.ResponseWriter, r *http.Request) {
	appID, _ := uuid.Parse(chi.URLParam(r, "appId"))
	report, _ := h.appReviewer.ReviewApp(r.Context(), appID)
	json.NewEncoder(w).Encode(report)
}
