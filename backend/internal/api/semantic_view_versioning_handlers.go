package api

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/api/common"
	"github.com/hondyman/semlayer/backend/internal/semanticviews"
)

// VersioningHandlers contains HTTP handlers for semantic view versioning
type VersioningHandlers struct {
	service *semanticviews.VersioningService
}

// NewVersioningHandlers creates new versioning HTTP handlers
func NewVersioningHandlers(db *sql.DB) *VersioningHandlers {
	return &VersioningHandlers{
		service: semanticviews.NewVersioningService(db),
	}
}

// RegisterRoutes registers all versioning routes
func (h *VersioningHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/semantic-views/{viewID}/versions", func(r chi.Router) {
		r.Post("/", h.CreateVersion)
		r.Get("/", h.ListVersions)
		r.Get("/latest", h.GetLatestVersion)
		r.Get("/{version}", h.GetVersion)
		r.Post("/{version}/deprecate", h.DeprecateVersion)
		r.Post("/migrate", h.MigrateVersion)
	})

	r.Get("/api/semantic-views/{viewID}/migrations", h.GetMigrationHistory)
}

// CreateVersion creates a new version
func (h *VersioningHandlers) CreateVersion(w http.ResponseWriter, r *http.Request) {
	viewIDStr := chi.URLParam(r, "viewID")
	viewID, err := uuid.Parse(viewIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	var req semanticviews.ViewVersion
	if err := common.ParseJSONBody(r, &req); err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	req.ViewID = viewID

	if err := h.service.CreateVersion(r.Context(), &req); err != nil {
		common.HandleInternalError(w, err)
		return
	}

	common.WriteCreated(w, req)
}

// GetVersion retrieves a specific version
func (h *VersioningHandlers) GetVersion(w http.ResponseWriter, r *http.Request) {
	viewIDStr := chi.URLParam(r, "viewID")
	viewID, err := uuid.Parse(viewIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	versionStr := chi.URLParam(r, "version")
	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	vv, err := h.service.GetVersion(r.Context(), viewID, version)
	if err != nil {
		common.HandleNotFound(w, err)
		return
	}

	common.WriteSuccess(w, vv, nil)
}

// GetLatestVersion retrieves the latest active version
func (h *VersioningHandlers) GetLatestVersion(w http.ResponseWriter, r *http.Request) {
	viewIDStr := chi.URLParam(r, "viewID")
	viewID, err := uuid.Parse(viewIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	vv, err := h.service.GetLatestVersion(r.Context(), viewID)
	if err != nil {
		common.HandleNotFound(w, err)
		return
	}

	common.WriteSuccess(w, vv, nil)
}

// ListVersions lists all versions
func (h *VersioningHandlers) ListVersions(w http.ResponseWriter, r *http.Request) {
	viewIDStr := chi.URLParam(r, "viewID")
	viewID, err := uuid.Parse(viewIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	versions, err := h.service.ListVersions(r.Context(), viewID)
	if err != nil {
		common.HandleInternalError(w, err)
		return
	}

	common.WriteSuccess(w, versions, nil)
}

// DeprecateVersion marks a version as deprecated
func (h *VersioningHandlers) DeprecateVersion(w http.ResponseWriter, r *http.Request) {
	viewIDStr := chi.URLParam(r, "viewID")
	viewID, err := uuid.Parse(viewIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	versionStr := chi.URLParam(r, "version")
	var version int
	if _, err := fmt.Sscanf(versionStr, "%d", &version); err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	if err := h.service.DeprecateVersion(r.Context(), viewID, version); err != nil {
		common.HandleInternalError(w, err)
		return
	}

	common.WriteNoContent(w)
}

// MigrateVersion migrates from one version to another
func (h *VersioningHandlers) MigrateVersion(w http.ResponseWriter, r *http.Request) {
	viewIDStr := chi.URLParam(r, "viewID")
	viewID, err := uuid.Parse(viewIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	var req struct {
		FromVersion int    `json:"from_version"`
		ToVersion   int    `json:"to_version"`
		ExecutedBy  string `json:"executed_by"`
	}

	if err := common.ParseJSONBody(r, &req); err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	migration, err := h.service.MigrateView(r.Context(), viewID, req.FromVersion, req.ToVersion, req.ExecutedBy)
	if err != nil {
		common.HandleInternalError(w, err)
		return
	}

	common.WriteSuccess(w, migration, nil)
}

// GetMigrationHistory retrieves migration history
func (h *VersioningHandlers) GetMigrationHistory(w http.ResponseWriter, r *http.Request) {
	viewIDStr := chi.URLParam(r, "viewID")
	viewID, err := uuid.Parse(viewIDStr)
	if err != nil {
		common.HandleBadRequest(w, err)
		return
	}

	migrations, err := h.service.GetMigrationHistory(r.Context(), viewID)
	if err != nil {
		common.HandleInternalError(w, err)
		return
	}

	common.WriteSuccess(w, migrations, nil)
}
