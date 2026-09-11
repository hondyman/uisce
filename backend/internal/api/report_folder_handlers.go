package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/reports"
)

// handleFolderError translates domain errors from reports folder operations into standard HTTP responses.
// Enforces zero existence leaks: cross-user or cross-tenant access returns 404 Not Found.
func handleFolderError(w http.ResponseWriter, err error) {
	if errors.Is(err, reports.ErrNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, reports.ErrConflict) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if errors.Is(err, reports.ErrCycleDetected) || errors.Is(err, reports.ErrDepthLimitExceeded) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// ListFolders lists all folders for the authenticated user in the authenticated tenant.
// GET /api/v1/reports/folders
func (h *ReportHandler) ListFolders(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	folders, err := h.service.ListFolders(r.Context(), tenantID, userID)
	if err != nil {
		handleFolderError(w, err)
		return
	}
	if folders == nil {
		folders = []reports.ReportFolder{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(folders)
}

type createFolderRequest struct {
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

// CreateFolder creates a new folder for the authenticated user under an optional parent folder.
// POST /api/v1/reports/folders
func (h *ReportHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var req createFolderRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "folder name is required", http.StatusBadRequest)
		return
	}

	folder := &reports.ReportFolder{
		ID:       uuid.New(),
		TenantID: tenantID,
		UserID:   userID,
		ParentID: req.ParentID,
		Name:     name,
	}

	if err := h.service.CreateFolder(r.Context(), folder); err != nil {
		handleFolderError(w, err)
		return
	}

	logging.GetLogger().Sugar().Infof("audit: user %s created folder %s (%s) in tenant %s", userID, folder.ID, folder.Name, tenantID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(folder)
}

type renameFolderRequest struct {
	Name string `json:"name"`
}

// RenameFolder renames an existing folder for the authenticated user.
// Strictly rename only: parent_id is completely ignored by this endpoint, preventing
// accidental subtree re-rooting.
// PUT /api/v1/reports/folders/{id}
func (h *ReportHandler) RenameFolder(w http.ResponseWriter, r *http.Request) {
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid folder id", http.StatusBadRequest)
		return
	}

	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	var req renameFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		http.Error(w, "folder name is required", http.StatusBadRequest)
		return
	}

	if err := h.service.RenameFolder(r.Context(), tenantID, userID, folderID, name); err != nil {
		handleFolderError(w, err)
		return
	}

	logging.GetLogger().Sugar().Infof("audit: user %s renamed folder %s to '%s' in tenant %s", userID, folderID, name, tenantID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":   folderID,
		"name": name,
	})
}

// MoveFolder moves a folder under a new parent (or to root when parent_id is null).
// POST /api/v1/reports/folders/{id}/move
func (h *ReportHandler) MoveFolder(w http.ResponseWriter, r *http.Request) {
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid folder id", http.StatusBadRequest)
		return
	}

	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var newParentID *uuid.UUID
	if val, ok := raw["parent_id"]; ok && val != nil {
		if pidStr, ok := val.(string); ok && strings.TrimSpace(pidStr) != "" {
			pid, parseErr := uuid.Parse(strings.TrimSpace(pidStr))
			if parseErr != nil {
				http.Error(w, "invalid parent_id UUID", http.StatusBadRequest)
				return
			}
			newParentID = &pid
		}
	}

	if err := h.service.MoveFolder(r.Context(), tenantID, userID, folderID, newParentID); err != nil {
		handleFolderError(w, err)
		return
	}

	logging.GetLogger().Sugar().Infof("audit: user %s moved folder %s to parent %v in tenant %s", userID, folderID, newParentID, tenantID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id":        folderID,
		"parent_id": newParentID,
	})
}

// DeleteFolder deletes a folder and cascades its subfolders and item links.
// Preserves underlying report templates.
// DELETE /api/v1/reports/folders/{id}
func (h *ReportHandler) DeleteFolder(w http.ResponseWriter, r *http.Request) {
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid folder id", http.StatusBadRequest)
		return
	}

	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := h.service.DeleteFolder(r.Context(), tenantID, userID, folderID); err != nil {
		handleFolderError(w, err)
		return
	}

	logging.GetLogger().Sugar().Infof("audit: user %s deleted folder %s in tenant %s", userID, folderID, tenantID)

	w.WriteHeader(http.StatusNoContent)
}

// ListFolderItems lists report template IDs filed under a folder.
// Re-enforces lockstep visibility on read.
// GET /api/v1/reports/folders/{id}/items
func (h *ReportHandler) ListFolderItems(w http.ResponseWriter, r *http.Request) {
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid folder id", http.StatusBadRequest)
		return
	}

	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	ids, err := h.service.ListFolderReportIDs(r.Context(), tenantID, userID, folderID)
	if err != nil {
		handleFolderError(w, err)
		return
	}
	if ids == nil {
		ids = []uuid.UUID{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ids)
}

type addFolderItemRequest struct {
	TemplateID *uuid.UUID `json:"template_id"`
	ReportID   *uuid.UUID `json:"report_id"`
}

// AddFolderItem files a report template into a folder.
// Idempotent: re-filing an already-filed report succeeds with 200 OK.
// Visibility-checked: filing another tenant's or an unshared personal report returns 404.
// POST /api/v1/reports/folders/{id}/items
func (h *ReportHandler) AddFolderItem(w http.ResponseWriter, r *http.Request) {
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid folder id", http.StatusBadRequest)
		return
	}

	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var req addFolderItemRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	var templateID uuid.UUID
	if req.TemplateID != nil && *req.TemplateID != uuid.Nil {
		templateID = *req.TemplateID
	} else if req.ReportID != nil && *req.ReportID != uuid.Nil {
		templateID = *req.ReportID
	} else {
		http.Error(w, "template_id is required", http.StatusBadRequest)
		return
	}

	if err := h.service.AddReportToFolder(r.Context(), tenantID, userID, folderID, templateID); err != nil {
		handleFolderError(w, err)
		return
	}

	logging.GetLogger().Sugar().Infof("audit: user %s filed template %s into folder %s in tenant %s", userID, templateID, folderID, tenantID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"folder_id":   folderID,
		"template_id": templateID,
	})
}

// RemoveFolderItem removes a report template from a folder.
// DELETE /api/v1/reports/folders/{id}/items/{templateId}
func (h *ReportHandler) RemoveFolderItem(w http.ResponseWriter, r *http.Request) {
	folderID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "invalid folder id", http.StatusBadRequest)
		return
	}

	templateID, err := uuid.Parse(chi.URLParam(r, "templateId"))
	if err != nil {
		http.Error(w, "invalid template id", http.StatusBadRequest)
		return
	}

	tenantID, userID, _, err := h.resolveAuthContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := h.service.RemoveReportFromFolder(r.Context(), tenantID, userID, folderID, templateID); err != nil {
		handleFolderError(w, err)
		return
	}

	logging.GetLogger().Sugar().Infof("audit: user %s removed template %s from folder %s in tenant %s", userID, templateID, folderID, tenantID)

	w.WriteHeader(http.StatusNoContent)
}
