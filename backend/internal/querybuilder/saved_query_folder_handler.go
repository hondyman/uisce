package querybuilder

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/jmoiron/sqlx"
)

// SavedQueryFolder is a user-scoped folder for organizing saved queries,
// mirroring internal/reports.ReportFolder (see the migration comment in
// 20261026_001_saved_query_folders_favorites_sharing.up.sql for why this is
// its own table rather than a shared one).
type SavedQueryFolder struct {
	ID        string    `json:"id"`
	TenantID  string    `json:"tenantId"`
	UserID    string    `json:"userId"`
	ParentID  string    `json:"parentId,omitempty"`
	Name      string    `json:"name"`
	ItemCount int       `json:"itemCount"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type savedQueryFolderRow struct {
	ID        string         `db:"id"`
	TenantID  string         `db:"tenant_id"`
	UserID    string         `db:"user_id"`
	ParentID  sql.NullString `db:"parent_id"`
	Name      string         `db:"name"`
	ItemCount int            `db:"item_count"`
	CreatedAt time.Time      `db:"created_at"`
	UpdatedAt time.Time      `db:"updated_at"`
}

func (r savedQueryFolderRow) toFolder() SavedQueryFolder {
	return SavedQueryFolder{
		ID: r.ID, TenantID: r.TenantID, UserID: r.UserID, ParentID: r.ParentID.String,
		Name: r.Name, ItemCount: r.ItemCount, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

const savedQueryFolderCols = "id, tenant_id, user_id, parent_id, name, created_at, updated_at"

// SavedQueryFolderHandler implements /api/explorer/saved-query-folders.
// Only the owner ever sees or mutates their own folders - unlike saved
// queries themselves, folders aren't shared, matching report_folders.
type SavedQueryFolderHandler struct {
	db   *sqlx.DB
	deps handlers.SecurityContextDeps
}

func NewSavedQueryFolderHandler(db *sqlx.DB, deps handlers.SecurityContextDeps) *SavedQueryFolderHandler {
	return &SavedQueryFolderHandler{db: db, deps: deps}
}

func (h *SavedQueryFolderHandler) writeJSON(w http.ResponseWriter, status int, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *SavedQueryFolderHandler) writeError(w http.ResponseWriter, err error, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": http.StatusText(status), "details": err.Error()})
}

func (h *SavedQueryFolderHandler) fetchWithCount(where string, args ...interface{}) ([]SavedQueryFolder, error) {
	query := `
		SELECT f.id, f.tenant_id, f.user_id, f.parent_id, f.name, f.created_at, f.updated_at,
		       COUNT(q.id) AS item_count
		FROM data_explorer.saved_query_folder f
		LEFT JOIN data_explorer.saved_query q ON q.folder_id = f.id
		WHERE ` + where + `
		GROUP BY f.id
		ORDER BY f.name ASC`
	rows, err := h.db.Queryx(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []SavedQueryFolder{}
	for rows.Next() {
		var r savedQueryFolderRow
		if err := rows.StructScan(&r); err != nil {
			return nil, err
		}
		out = append(out, r.toFolder())
	}
	return out, rows.Err()
}

// HandleListFolders handles GET /api/explorer/saved-query-folders.
func (h *SavedQueryFolderHandler) HandleListFolders(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	folders, err := h.fetchWithCount("f.tenant_id = $1 AND f.user_id = $2", secCtx.TenantID, secCtx.UserID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to list folders: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, map[string]interface{}{"folders": folders})
}

type savedQueryFolderRequest struct {
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}

// HandleCreateFolder handles POST /api/explorer/saved-query-folders.
func (h *SavedQueryFolderHandler) HandleCreateFolder(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	var req savedQueryFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		h.writeError(w, errors.New("name is required"), http.StatusBadRequest)
		return
	}
	var parentID interface{}
	if req.ParentID != "" {
		parentID = req.ParentID
	}
	id := uuid.NewString()
	var row savedQueryFolderRow
	err = h.db.QueryRowx(`
		INSERT INTO data_explorer.saved_query_folder (id, tenant_id, user_id, parent_id, name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+savedQueryFolderCols+`, 0 AS item_count
	`, id, secCtx.TenantID, secCtx.UserID, parentID, req.Name).StructScan(&row)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to create folder: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusCreated, row.toFolder())
}

// HandleUpdateFolder handles PUT /api/explorer/saved-query-folders/{id} -
// rename and/or move (reparent).
func (h *SavedQueryFolderHandler) HandleUpdateFolder(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	var req savedQueryFolderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, fmt.Errorf("invalid request body: %w", err), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		h.writeError(w, errors.New("name is required"), http.StatusBadRequest)
		return
	}
	if req.ParentID == id {
		h.writeError(w, errors.New("a folder cannot be its own parent"), http.StatusBadRequest)
		return
	}
	var parentID interface{}
	if req.ParentID != "" {
		parentID = req.ParentID
	}
	res, err := h.db.Exec(`
		UPDATE data_explorer.saved_query_folder SET name = $1, parent_id = $2, updated_at = NOW()
		WHERE id = $3 AND tenant_id = $4 AND user_id = $5
	`, req.Name, parentID, id, secCtx.TenantID, secCtx.UserID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to update folder: %w", err), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		h.writeError(w, errors.New("folder not found"), http.StatusNotFound)
		return
	}
	folders, err := h.fetchWithCount("f.id = $1 AND f.tenant_id = $2", id, secCtx.TenantID)
	if err != nil || len(folders) == 0 {
		h.writeError(w, fmt.Errorf("failed to reload folder: %w", err), http.StatusInternalServerError)
		return
	}
	h.writeJSON(w, http.StatusOK, folders[0])
}

// HandleDeleteFolder handles DELETE /api/explorer/saved-query-folders/{id}.
// Queries inside are not deleted - the FK is ON DELETE SET NULL, so they
// simply fall back to unfiled, same as deleting a report folder does not
// delete its reports.
func (h *SavedQueryFolderHandler) HandleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.deps)
	if err != nil {
		h.writeError(w, err, http.StatusBadRequest)
		return
	}
	id := chi.URLParam(r, "id")
	res, err := h.db.Exec(`DELETE FROM data_explorer.saved_query_folder WHERE id = $1 AND tenant_id = $2 AND user_id = $3`,
		id, secCtx.TenantID, secCtx.UserID)
	if err != nil {
		h.writeError(w, fmt.Errorf("failed to delete folder: %w", err), http.StatusInternalServerError)
		return
	}
	if n, _ := res.RowsAffected(); n == 0 {
		h.writeError(w, errors.New("folder not found"), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
