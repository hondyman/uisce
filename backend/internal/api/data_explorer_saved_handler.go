package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/libs/jwt-middleware"
)

type DataExplorerSavedQuery struct {
	ID          string          `json:"id"`
	TenantID    string          `json:"tenant_id"`
	UserID      string          `json:"user_id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	SourceKind  string          `json:"source_kind"`
	SourceID    string          `json:"source_id"`
	BindingID   *string         `json:"binding_id,omitempty"`
	QueryState  json.RawMessage `json:"query_state"`
	Tags        []string        `json:"tags"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type DataExplorerSavedHandler struct {
	db *sql.DB
}

func NewDataExplorerSavedHandler(db *sql.DB) *DataExplorerSavedHandler {
	return &DataExplorerSavedHandler{db: db}
}

func (h *DataExplorerSavedHandler) ListSavedQueries(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "tenant_id missing from JWT", http.StatusBadRequest)
		return
	}
	userID := claims.UserID
	if userID == "" {
		http.Error(w, "user sub missing from JWT", http.StatusBadRequest)
		return
	}

	query := `
		SELECT id, tenant_id, user_id, name, description, source_kind,
		       source_id, binding_id, query_state, tags, created_at, updated_at
		FROM data_explorer.saved_query
		WHERE tenant_id = $1 AND user_id = $2
		ORDER BY updated_at DESC
		LIMIT 100`

	rows, err := h.db.QueryContext(r.Context(), query, tenantID, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to list saved queries: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := []DataExplorerSavedQuery{}
	for rows.Next() {
		var q DataExplorerSavedQuery
		var description sql.NullString
		var bindingID sql.NullString
		var tags []byte
		err := rows.Scan(
			&q.ID, &q.TenantID, &q.UserID, &q.Name, &description,
			&q.SourceKind, &q.SourceID, &bindingID, &q.QueryState, &tags,
			&q.CreatedAt, &q.UpdatedAt,
		)
		if err != nil {
			continue
		}
		if description.Valid {
			q.Description = description.String
		}
		if bindingID.Valid {
			q.BindingID = &bindingID.String
		}
		if len(tags) > 0 {
			_ = json.Unmarshal(tags, &q.Tags)
		}
		result = append(result, q)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

type CreateSavedQueryRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	SourceKind  string          `json:"source_kind"`
	SourceID    string          `json:"source_id"`
	BindingID   *string         `json:"binding_id,omitempty"`
	QueryState  json.RawMessage `json:"query_state"`
	Tags        []string        `json:"tags"`
}

func (h *DataExplorerSavedHandler) CreateSavedQuery(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "tenant_id missing from JWT", http.StatusBadRequest)
		return
	}
	userID := claims.UserID
	if userID == "" {
		http.Error(w, "user sub missing from JWT", http.StatusBadRequest)
		return
	}

	var req CreateSavedQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if req.SourceID == "" {
		http.Error(w, "source_id is required", http.StatusBadRequest)
		return
	}
	if req.QueryState == nil {
		req.QueryState = []byte("{}")
	}
	if req.Tags == nil {
		req.Tags = []string{}
	}
	tagsJSON, _ := json.Marshal(req.Tags)

	query := `
		INSERT INTO data_explorer.saved_query
			(tenant_id, user_id, name, description, source_kind, source_id, binding_id, query_state, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, tenant_id, user_id, name, description, source_kind,
		          source_id, binding_id, query_state, tags, created_at, updated_at`

	var q DataExplorerSavedQuery
	var description sql.NullString
	var bindingID sql.NullString
	var tags []byte
	err := h.db.QueryRowContext(r.Context(), query,
		tenantID, userID, req.Name, req.Description,
		req.SourceKind, req.SourceID, req.BindingID,
		req.QueryState, tagsJSON,
	).Scan(
		&q.ID, &q.TenantID, &q.UserID, &q.Name, &description,
		&q.SourceKind, &q.SourceID, &bindingID, &q.QueryState, &tags,
		&q.CreatedAt, &q.UpdatedAt,
	)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create saved query: %v", err), http.StatusInternalServerError)
		return
	}
	if description.Valid {
		q.Description = description.String
	}
	if bindingID.Valid {
		q.BindingID = &bindingID.String
	}
	if len(tags) > 0 {
		_ = json.Unmarshal(tags, &q.Tags)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(q)
}

func (h *DataExplorerSavedHandler) DeleteSavedQuery(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "tenant_id missing from JWT", http.StatusBadRequest)
		return
	}
	userID := claims.UserID
	if userID == "" {
		http.Error(w, "user sub missing from JWT", http.StatusBadRequest)
		return
	}

	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "id path parameter required", http.StatusBadRequest)
		return
	}

	query := `DELETE FROM data_explorer.saved_query WHERE id = $1 AND tenant_id = $2 AND user_id = $3`
	result, err := h.db.ExecContext(r.Context(), query, id, tenantID, userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to delete saved query: %v", err), http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "saved query not found", http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
