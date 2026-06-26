package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/models"
)

type BOExportImportHandler struct {
	ExportSvc    *analytics.BOExportService
	ImportSvc    *analytics.BOImportService
	SecurityDeps SecurityContextDeps
}

func NewBOExportImportHandler(exportSvc *analytics.BOExportService, importSvc *analytics.BOImportService, deps SecurityContextDeps) *BOExportImportHandler {
	return &BOExportImportHandler{
		ExportSvc:    exportSvc,
		ImportSvc:    importSvc,
		SecurityDeps: deps,
	}
}

func (h *BOExportImportHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/bo", func(r chi.Router) {
		r.Get("/{boId}/export", h.ExportBO)
		r.Post("/export/multiple", h.ExportMultipleBOs)
		r.Post("/import", h.ImportBO)
	})
}

// ExportBO handles singular BO export
func (h *BOExportImportHandler) ExportBO(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.SecurityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tenantID := secCtx.TenantID

	bundle, err := h.ExportSvc.ExportBO(ctx, tenantID, boID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=bo_export_%s_%d.json", boID, time.Now().Unix()))
	json.NewEncoder(w).Encode(bundle)
}

// ExportMultipleBOs handles bulk export
func (h *BOExportImportHandler) ExportMultipleBOs(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BOIDs []string `json:"bo_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.SecurityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tenantID := secCtx.TenantID

	bundle, err := h.ExportSvc.ExportMultipleBOs(ctx, tenantID, req.BOIDs)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=bo_bundle_%d.json", time.Now().Unix()))
	json.NewEncoder(w).Encode(bundle)
}

// ImportBO handles the import process (analyze or apply)
func (h *BOExportImportHandler) ImportBO(w http.ResponseWriter, r *http.Request) {
	var req models.ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	userID := r.Header.Get("X-User-ID")
	secCtx, ctx, err := SecurityContextFromRequest(r, req.DatasourceID, req.Region, h.SecurityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.ImportSvc.ImportBO(ctx, secCtx, req, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
