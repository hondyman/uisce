package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/auth"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Lookup represents a simple key for a tenant-wide lookup table
type Lookup struct {
	ID          string     `json:"id"`
	TenantID    string     `json:"tenant_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	SourceTable *string    `json:"source_table,omitempty"` // Optional: if set, values come from this table
	IsCore      bool       `json:"is_core"`                // Computed: true if tenant is gold_copy
	CreatedAt   *time.Time `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at"`
}

// LookupValue represents an individual lookup value in a lookup table
type LookupValue struct {
	ID        string                 `json:"id"`
	LookupID  string                 `json:"lookup_id"`
	TenantID  string                 `json:"tenant_id"`
	Value     string                 `json:"value"`
	Label     string                 `json:"label"`
	ParentID  *string                `json:"parent_id,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	IsCore    bool                   `json:"is_core"` // Computed: true if tenant is gold_copy
	CreatedAt *time.Time             `json:"created_at"`
}

// RegisterLookupsRoutes registers lookup endpoints for frontend
func RegisterLookupsRoutes(r chi.Router, db *sql.DB) {
	r.Get("/lookups", handleListLookups(db))
	r.Get("/lookups/{id}/values", handleGetLookupValues(db))
	r.Get("/lookups/{id}/export", handleExportLookupValues(db))
	r.Post("/lookups", handleCreateLookup(db))
	r.Put("/lookups/{id}", handleUpdateLookup(db))
	r.Delete("/lookups/{id}", handleDeleteLookup(db))

	r.Post("/lookups/{id}/values", handleCreateLookupValue(db))
	r.Put("/lookups/{id}/values/{valueId}", handleUpdateLookupValue(db))
	r.Delete("/lookups/{id}/values/{valueId}", handleDeleteLookupValue(db))
	r.Post("/lookups/{id}/propagate", handlePropagateLookup(db))
}

// handleCreateLookup creates a new lookup for a tenant
func handleCreateLookup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Priority 1: X-Tenant-ID header (sent by frontend for explicit tenant context)
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		// Priority 2: authenticated user context
		if tenantID == "" {
			if user, ok := auth.GetUserFromContext(r.Context()); ok && user.TenantID != "" {
				tenantID = user.TenantID
			}
		}
		// Priority 3: query param for backwards compatibility
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}
		log.Printf("[handleCreateLookup] Using tenantID=%s (header=%s, query=%s)", tenantID, jwtmiddleware.GetClaimsFromContext(r).TenantID, r.URL.Query().Get("tenant_id"))

		var payload struct {
			Name        string  `json:"name"`
			Description string  `json:"description"`
			SourceTable *string `json:"source_table"` // Optional: table name for table-backed lookups
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body", "decode_error", err.Error())
			return
		}
		id := uuid.NewString()
		_, err := db.Exec(`INSERT INTO lookups (id, tenant_id, name, description, source_table, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`, id, tenantID, payload.Name, payload.Description, payload.SourceTable)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to insert lookup", "insert_error", err.Error())
			return
		}
		// Return the created lookup
		var l Lookup
		row := db.QueryRow(`SELECT id, tenant_id, name, COALESCE(description, '') as description, source_table, created_at, updated_at FROM lookups WHERE id=$1 AND tenant_id=$2`, id, tenantID)
		var createdAt sql.NullTime
		var updatedAt sql.NullTime
		var sourceTable sql.NullString
		if err := row.Scan(&l.ID, &l.TenantID, &l.Name, &l.Description, &sourceTable, &createdAt, &updatedAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch created lookup", "fetch_error", err.Error())
			return
		}
		if createdAt.Valid {
			l.CreatedAt = &createdAt.Time
		}
		if updatedAt.Valid {
			l.UpdatedAt = &updatedAt.Time
		}
		if sourceTable.Valid {
			l.SourceTable = &sourceTable.String
		}

		// Check if this tenant is gold copy to set IsCore
		var goldCopyTenantID string
		err = db.QueryRow(`SELECT id FROM tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyTenantID)
		if err == nil && goldCopyTenantID == tenantID {
			l.IsCore = true
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(l)
	}
}

// handleUpdateLookup updates a lookup's name/description
func handleUpdateLookup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}
		lookupID := chi.URLParam(r, "id")
		if lookupID == "" {
			writeJSONError(w, http.StatusBadRequest, "lookup id is required", "missing_lookup", nil)
			return
		}
		var payload struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body", "decode_error", err.Error())
			return
		}

		_, err := db.Exec(`UPDATE lookups SET name=$1, description=$2, updated_at=CURRENT_TIMESTAMP WHERE id=$3 AND tenant_id=$4`, payload.Name, payload.Description, lookupID, tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to update lookup", "update_error", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleDeleteLookup deletes a lookup and its values
func handleDeleteLookup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}
		lookupID := chi.URLParam(r, "id")
		if lookupID == "" {
			writeJSONError(w, http.StatusBadRequest, "lookup id is required", "missing_lookup", nil)
			return
		}

		tx, err := db.Begin()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to start transaction", "tx_error", err.Error())
			return
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`DELETE FROM lookup_values WHERE lookup_id=$1 AND tenant_id=$2`, lookupID, tenantID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to delete lookup values", "delete_error", err.Error())
			return
		}
		if _, err := tx.Exec(`DELETE FROM lookups WHERE id=$1 AND tenant_id=$2`, lookupID, tenantID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to delete lookup", "delete_error", err.Error())
			return
		}
		if err := tx.Commit(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "transaction failed", "tx_error", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleCreateLookupValue creates a new value in a lookup
func handleCreateLookupValue(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Priority 1: X-Tenant-ID header (sent by frontend for explicit tenant context)
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		// Priority 2: authenticated user context
		if tenantID == "" {
			if user, ok := auth.GetUserFromContext(r.Context()); ok && user.TenantID != "" {
				tenantID = user.TenantID
			}
		}
		// Priority 3: query param for backwards compatibility
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}
		log.Printf("[handleCreateLookupValue] Using tenantID=%s (header=%s, query=%s)", tenantID, jwtmiddleware.GetClaimsFromContext(r).TenantID, r.URL.Query().Get("tenant_id"))

		lookupID := chi.URLParam(r, "id")
		if lookupID == "" {
			writeJSONError(w, http.StatusBadRequest, "lookup id is required", "missing_lookup", nil)
			return
		}

		var payload struct {
			Value    string                 `json:"value"`
			Label    string                 `json:"label"`
			ParentID *string                `json:"parent_id"`
			Metadata map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body", "decode_error", err.Error())
			return
		}

		id := uuid.NewString()
		metaJSON := []byte("null")
		if payload.Metadata != nil {
			if b, err := json.Marshal(payload.Metadata); err == nil {
				metaJSON = b
			}
		}
		_, err := db.Exec(`INSERT INTO lookup_values (id, lookup_id, tenant_id, value, label, parent_id, metadata, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,CURRENT_TIMESTAMP)`, id, lookupID, tenantID, payload.Value, payload.Label, payload.ParentID, metaJSON)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to insert lookup value", "insert_error", err.Error())
			return
		}

		// return created value
		var v LookupValue
		var meta sql.NullString
		var createdAt sql.NullTime
		row := db.QueryRow(`SELECT id, lookup_id, tenant_id, value, label, parent_id, metadata, created_at FROM lookup_values WHERE id=$1 AND tenant_id=$2`, id, tenantID)
		if err := row.Scan(&v.ID, &v.LookupID, &v.TenantID, &v.Value, &v.Label, &v.ParentID, &meta, &createdAt); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch created value", "fetch_error", err.Error())
			return
		}
		if meta.Valid && len(meta.String) > 0 {
			_ = json.Unmarshal([]byte(meta.String), &v.Metadata)
		}
		if createdAt.Valid {
			v.CreatedAt = &createdAt.Time
		}

		// Check if this tenant is gold copy to set IsCore
		var goldCopyTenantID string
		err = db.QueryRow(`SELECT id FROM tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyTenantID)
		if err == nil && goldCopyTenantID == tenantID {
			v.IsCore = true
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(v)
	}
}

// handleUpdateLookupValue updates an existing lookup value
func handleUpdateLookupValue(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}
		lookupID := chi.URLParam(r, "id")
		valueID := chi.URLParam(r, "valueId")
		if lookupID == "" || valueID == "" {
			writeJSONError(w, http.StatusBadRequest, "missing id", "missing_id", nil)
			return
		}
		var payload struct {
			Value    string                 `json:"value"`
			Label    string                 `json:"label"`
			ParentID *string                `json:"parent_id"`
			Metadata map[string]interface{} `json:"metadata"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid request body", "decode_error", err.Error())
			return
		}
		metaJSON := []byte("null")
		if payload.Metadata != nil {
			if b, err := json.Marshal(payload.Metadata); err == nil {
				metaJSON = b
			}
		}
		_, err := db.Exec(`UPDATE lookup_values SET value=$1, label=$2, parent_id=$3, metadata=$4 WHERE id=$5 AND lookup_id=$6 AND tenant_id=$7`, payload.Value, payload.Label, payload.ParentID, metaJSON, valueID, lookupID, tenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to update value", "update_error", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleDeleteLookupValue deletes a lookup value
func handleDeleteLookupValue(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}
		lookupID := chi.URLParam(r, "id")
		valueID := chi.URLParam(r, "valueId")
		if lookupID == "" || valueID == "" {
			writeJSONError(w, http.StatusBadRequest, "missing id", "missing_id", nil)
			return
		}
		if _, err := db.Exec(`DELETE FROM lookup_values WHERE id=$1 AND lookup_id=$2 AND tenant_id=$3`, valueID, lookupID, tenantID); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to delete value", "delete_error", err.Error())
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// handleListLookups returns configured lookup tables for a tenant
func handleListLookups(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		q := r.URL.Query().Get("q")
		limitStr := r.URL.Query().Get("limit")
		limit := 50
		if limitStr != "" {
			if li, err := strconv.Atoi(limitStr); err == nil && li > 0 && li <= 1000 {
				limit = li
			}
		}
		// cursor is an offset integer encoded as a string; default 0
		cursor := 0
		if c := r.URL.Query().Get("cursor"); c != "" {
			if ci, err := strconv.Atoi(c); err == nil && ci >= 0 {
				cursor = ci
			}
		}
		if tenantID == "" {
			http.Error(w, "tenant_id is required", http.StatusBadRequest)
			return
		}

		// First, get the gold copy tenant ID (tenant with gold_copy = true)
		var goldCopyTenantID sql.NullString
		err := db.QueryRow(`SELECT id FROM tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyTenantID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Failed to find gold copy tenant: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var rows *sql.Rows
		// Query lookups from BOTH the gold copy tenant (core lookups) AND the current tenant (custom lookups)
		if q == "" {
			if goldCopyTenantID.Valid {
				rows, err = db.Query(`
					SELECT id, tenant_id, name, COALESCE(description, '') as description, source_table, created_at, updated_at 
					FROM lookups 
					WHERE tenant_id = $1 OR tenant_id = $2
					ORDER BY name 
					LIMIT $3 OFFSET $4`, tenantID, goldCopyTenantID.String, limit, cursor)
			} else {
				rows, err = db.Query(`SELECT id, tenant_id, name, COALESCE(description, '') as description, source_table, created_at, updated_at FROM lookups WHERE tenant_id = $1 ORDER BY name LIMIT $2 OFFSET $3`, tenantID, limit, cursor)
			}
		} else {
			search := "%" + q + "%"
			// Use LOWER(...) LIKE LOWER(...) for cross-database case-insensitive search (Postgres ILIKE not supported in SQLite)
			if goldCopyTenantID.Valid {
				rows, err = db.Query(`
					SELECT id, tenant_id, name, COALESCE(description, '') as description, source_table, created_at, updated_at 
					FROM lookups 
					WHERE (tenant_id = $1 OR tenant_id = $2) 
					  AND (LOWER(name) LIKE LOWER($3) OR LOWER(COALESCE(description, '')) LIKE LOWER($3)) 
					ORDER BY name 
					LIMIT $4 OFFSET $5`, tenantID, goldCopyTenantID.String, search, limit, cursor)
			} else {
				rows, err = db.Query(`SELECT id, tenant_id, name, COALESCE(description, '') as description, source_table, created_at, updated_at FROM lookups WHERE tenant_id = $1 AND (LOWER(name) LIKE LOWER($2) OR LOWER(COALESCE(description, '')) LIKE LOWER($2)) ORDER BY name LIMIT $3 OFFSET $4`, tenantID, search, limit, cursor)
			}
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var lookups []Lookup
		for rows.Next() {
			var l Lookup
			var createdAt sql.NullTime
			var updatedAt sql.NullTime
			var sourceTable sql.NullString
			if err := rows.Scan(&l.ID, &l.TenantID, &l.Name, &l.Description, &sourceTable, &createdAt, &updatedAt); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if sourceTable.Valid {
				l.SourceTable = &sourceTable.String
			}
			if createdAt.Valid {
				l.CreatedAt = &createdAt.Time
			}
			if updatedAt.Valid {
				l.UpdatedAt = &updatedAt.Time
			}
			// Set IsCore flag
			if goldCopyTenantID.Valid && l.TenantID == goldCopyTenantID.String {
				l.IsCore = true
			}
			lookups = append(lookups, l)
		}

		if lookups == nil {
			lookups = []Lookup{}
		}

		// Return next cursor (offset) for client-side pagination. We use offset as a simple cursor token.
		w.Header().Set("Content-Type", "application/json")
		// Client can use `limit` and `cursor` to fetch the next set. Provide a computed next cursor if more rows available.
		// NOTE: This is a simple offset-based cursor; if your dataset grows very large consider keyset cursor.
		nextCursor := 0
		if len(lookups) == limit {
			// next cursor is offset + limit
			nextCursor = cursor + limit
		}
		resp := map[string]interface{}{
			"items":       lookups,
			"next_cursor": nextCursor,
		}
		json.NewEncoder(w).Encode(resp)
	}
}

// handleGetLookupValues returns all values for a given lookup id (tenant scoped)
// Supports both static lookup_values and table-backed lookups (if source_table is set)
func handleGetLookupValues(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID == "" {
			http.Error(w, "tenant_id is required", http.StatusBadRequest)
			return
		}
		lookupID := chi.URLParam(r, "id")
		if lookupID == "" {
			http.Error(w, "lookup id is required", http.StatusBadRequest)
			return
		}

		// support limiting/pagination via limit and cursor (offset)
		limit := 100
		if l := r.URL.Query().Get("limit"); l != "" {
			if li, err := strconv.Atoi(l); err == nil && li > 0 && li <= 1000 {
				limit = li
			}
		}
		cursor := 0
		if c := r.URL.Query().Get("cursor"); c != "" {
			if ci, err := strconv.Atoi(c); err == nil && ci >= 0 {
				cursor = ci
			}
		}

		// Optional filter by parent_id for cascading lookups
		parentIDFilter := r.URL.Query().Get("parent_id")
		parentValueFilter := r.URL.Query().Get("parent_value")

		// Get gold copy tenant
		var goldCopyTenantID sql.NullString
		err := db.QueryRow(`SELECT id FROM tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyTenantID)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "Failed to find gold copy tenant: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// First, check if this lookup is table-backed
		var sourceTable sql.NullString
		// Check lookup existence in either tenant or gold copy
		lookupQuery := `SELECT source_table FROM lookups WHERE id = $1 AND (tenant_id = $2 OR tenant_id = $3)`
		if !goldCopyTenantID.Valid {
			lookupQuery = `SELECT source_table FROM lookups WHERE id = $1 AND tenant_id = $2 AND $3=$3` // dummy check for $3
		}
		err = db.QueryRow(lookupQuery, lookupID, tenantID, goldCopyTenantID.String).Scan(&sourceTable)
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var values []LookupValue

		// If source_table is set, query that table instead of lookup_values
		if sourceTable.Valid && sourceTable.String != "" {
			// Query the specified table for id, name, and parent_id (if available) columns
			// Most tables will have these columns to support cascading lookups
			var query string
			var args []interface{}

			// If a parent value (label) is provided instead of an id, resolve it to id first
			if parentValueFilter != "" && parentIDFilter == "" {
				var resolvedParent sql.NullString
				// Find id of parent record by name (case-insensitive) — tenant-scoped
				err := db.QueryRow("SELECT id FROM "+sourceTable.String+" WHERE tenant_id = $1 AND LOWER(name) = LOWER($2) LIMIT 1", tenantID, parentValueFilter).Scan(&resolvedParent)
				if err != nil && err != sql.ErrNoRows {
					http.Error(w, "Failed to resolve parent value: "+err.Error(), http.StatusInternalServerError)
					return
				}
				if resolvedParent.Valid {
					// Set parentIDFilter from resolved parent so it's used below
					parentIDFilter = resolvedParent.String
				}
			}

			if parentIDFilter != "" {
				// Filter by parent_id for cascading
				query = `SELECT id, name, COALESCE(parent_id, NULL) as parent_id, tenant_id FROM ` + sourceTable.String + ` WHERE (tenant_id = $1 OR tenant_id = $2) AND parent_id = $3 ORDER BY name LIMIT $4 OFFSET $5`
				args = []interface{}{tenantID, goldCopyTenantID.String, parentIDFilter, limit, cursor}
			} else {
				// Get top-level items (where parent_id IS NULL)
				query = `SELECT id, name, COALESCE(parent_id, NULL) as parent_id, tenant_id FROM ` + sourceTable.String + ` WHERE (tenant_id = $1 OR tenant_id = $2) AND parent_id IS NULL ORDER BY name LIMIT $3 OFFSET $4`
				args = []interface{}{tenantID, goldCopyTenantID.String, limit, cursor}
			}

			// Debug: log the SQL and args we will execute — helps verify filtering
			log.Printf("[GetLookupValues] SQL=%s args=%v", query, args)
			rows, err := db.Query(query, args...)
			if err != nil {
				http.Error(w, "Failed to query source table: "+err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			for rows.Next() {
				var id, name string
				var parentID sql.NullString
				var rowTenantID string
				if err := rows.Scan(&id, &name, &parentID, &rowTenantID); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				lv := LookupValue{
					ID:       id,
					LookupID: lookupID,
					TenantID: rowTenantID,
					Value:    id,
					Label:    name,
					IsCore:   goldCopyTenantID.Valid && rowTenantID == goldCopyTenantID.String,
				}
				if parentID.Valid {
					lv.ParentID = &parentID.String
				}
				values = append(values, lv)
			}
			// Debug log: report how many values were fetched for table-backed lookup
			log.Printf("[GetLookupValues] lookup=%s tenant=%s parent=%s items=%d", lookupID, tenantID, parentIDFilter, len(values))
		} else {
			// Query static lookup_values with parent_id filtering

			// If a parent value (label) is provided instead of an id, resolve it to id first
			if parentValueFilter != "" && parentIDFilter == "" {
				var resolvedParent sql.NullString
				err := db.QueryRow("SELECT id FROM lookup_values WHERE tenant_id = $1 AND lookup_id = $2 AND LOWER(label) = LOWER($3) LIMIT 1", tenantID, lookupID, parentValueFilter).Scan(&resolvedParent)
				if err != nil && err != sql.ErrNoRows {
					http.Error(w, "Failed to resolve parent value: "+err.Error(), http.StatusInternalServerError)
					return
				}
				if resolvedParent.Valid {
					parentIDFilter = resolvedParent.String
				}
			}

			var rows *sql.Rows
			if parentIDFilter != "" {
				// Filter by parent_id for cascading
				rows, err = db.Query(`SELECT id, lookup_id, tenant_id, COALESCE(value, '') as value, COALESCE(label, '') as label, parent_id, metadata, created_at 
					FROM lookup_values 
					WHERE (tenant_id = $1 OR tenant_id = $2) AND lookup_id = $3 AND parent_id = $4 
					ORDER BY label LIMIT $5 OFFSET $6`,
					tenantID, goldCopyTenantID.String, lookupID, parentIDFilter, limit, cursor)
			} else {
				// Get top-level items (where parent_id IS NULL)
				rows, err = db.Query(`SELECT id, lookup_id, tenant_id, COALESCE(value, '') as value, COALESCE(label, '') as label, parent_id, metadata, created_at 
					FROM lookup_values 
					WHERE (tenant_id = $1 OR tenant_id = $2) AND lookup_id = $3 AND parent_id IS NULL 
					ORDER BY label LIMIT $4 OFFSET $5`,
					tenantID, goldCopyTenantID.String, lookupID, limit, cursor)
			}
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			for rows.Next() {
				var v LookupValue
				var metadataJSON []byte
				var createdAt sql.NullTime
				if err := rows.Scan(&v.ID, &v.LookupID, &v.TenantID, &v.Value, &v.Label, &v.ParentID, &metadataJSON, &createdAt); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if len(metadataJSON) > 0 {
					json.Unmarshal(metadataJSON, &v.Metadata)
				}
				if createdAt.Valid {
					v.CreatedAt = &createdAt.Time
				}
				if goldCopyTenantID.Valid && v.TenantID == goldCopyTenantID.String {
					v.IsCore = true
				}
				values = append(values, v)
			}
			// Debug log for static lookup_values
			log.Printf("[GetLookupValues] lookup=%s tenant=%s parent=%s items=%d (static)", lookupID, tenantID, parentIDFilter, len(values))
		}

		if values == nil {
			values = []LookupValue{}
		}

		// Determine next cursor (offset) for more
		nextCursor := 0
		if len(values) == limit {
			nextCursor = cursor + limit
		}
		resp := map[string]interface{}{"items": values, "next_cursor": nextCursor}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// handlePropagateLookup copies a lookup and its values to all other tenants
func handlePropagateLookup(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		srcTenantID := r.URL.Query().Get("tenant_id")
		if srcTenantID == "" {
			writeJSONError(w, http.StatusBadRequest, "tenant_id is required", "missing_tenant", nil)
			return
		}
		lookupID := chi.URLParam(r, "id")
		if lookupID == "" {
			writeJSONError(w, http.StatusBadRequest, "lookup id is required", "missing_lookup", nil)
			return
		}

		// 1. Fetch Source Lookup
		var l Lookup
		var sourceTable sql.NullString
		err := db.QueryRow(`SELECT name, description, source_table FROM lookups WHERE id=$1 AND tenant_id=$2`, lookupID, srcTenantID).Scan(&l.Name, &l.Description, &sourceTable)
		if err != nil {
			if err == sql.ErrNoRows {
				writeJSONError(w, http.StatusNotFound, "lookup not found", "not_found", nil)
			} else {
				writeJSONError(w, http.StatusInternalServerError, "failed to fetch lookup", "fetch_error", err.Error())
			}
			return
		}

		// 2. Fetch Source Values
		rows, err := db.Query(`SELECT value, label, parent_id, metadata FROM lookup_values WHERE lookup_id=$1 AND tenant_id=$2`, lookupID, srcTenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch values", "fetch_error", err.Error())
			return
		}
		defer rows.Close()

		var srcValues []LookupValue
		for rows.Next() {
			var lv LookupValue
			var metaJSON []byte
			if err := rows.Scan(&lv.Value, &lv.Label, &lv.ParentID, &metaJSON); err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to scan values", "scan_error", err.Error())
				return
			}
			if len(metaJSON) > 0 {
				json.Unmarshal(metaJSON, &lv.Metadata)
			}
			srcValues = append(srcValues, lv)
		}

		// 3. Get All Tenants (excluding source)
		tRows, err := db.Query(`SELECT id FROM tenants WHERE id != $1 AND is_active = true`, srcTenantID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to fetch tenants", "fetch_error", err.Error())
			return
		}
		defer tRows.Close()

		var targetTenants []string
		for tRows.Next() {
			var tid string
			if err := tRows.Scan(&tid); err == nil {
				targetTenants = append(targetTenants, tid)
			}
		}

		// 4. Propagate
		tx, err := db.Begin()
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to begin tx", "tx_error", err.Error())
			return
		}
		defer tx.Rollback()

		stats := map[string]int{"tenants": 0, "lookups_created": 0, "values_created": 0}

		for _, targetTID := range targetTenants {
			stats["tenants"]++

			// Check if lookup exists by name
			var targetLookupID string
			err := tx.QueryRow(`SELECT id FROM lookups WHERE tenant_id=$1 AND name=$2`, targetTID, l.Name).Scan(&targetLookupID)

			if err == sql.ErrNoRows {
				// Create Lookup
				targetLookupID = uuid.NewString()
				_, err = tx.Exec(`INSERT INTO lookups (id, tenant_id, name, description, source_table, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
					targetLookupID, targetTID, l.Name, l.Description, sourceTable)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, "failed to insert lookup for tenant "+targetTID, "insert_error", err.Error())
					return
				}
				stats["lookups_created"]++
			} else if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to check lookup existence", "check_error", err.Error())
				return
			}

			// Clear existing values for simplicity/idempotency or perform merge?
			// Simplest "propagate" is overwrite/sync. Let's delete existing values for this lookup to ensure full sync.
			_, err = tx.Exec(`DELETE FROM lookup_values WHERE lookup_id=$1 AND tenant_id=$2`, targetLookupID, targetTID)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "failed to clear values for tenant "+targetTID, "delete_error", err.Error())
				return
			}

			// Insert values
			for _, lv := range srcValues {
				newID := uuid.NewString()
				metaJSON := []byte("null")
				if lv.Metadata != nil {
					if b, err := json.Marshal(lv.Metadata); err == nil {
						metaJSON = b
					}
				}
				_, err = tx.Exec(`INSERT INTO lookup_values (id, lookup_id, tenant_id, value, label, parent_id, metadata, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,CURRENT_TIMESTAMP)`,
					newID, targetLookupID, targetTID, lv.Value, lv.Label, lv.ParentID, metaJSON)
				if err != nil {
					writeJSONError(w, http.StatusInternalServerError, "failed to insert value for tenant "+targetTID, "insert_error", err.Error())
					return
				}
				stats["values_created"]++
			}
		}

		if err := tx.Commit(); err != nil {
			writeJSONError(w, http.StatusInternalServerError, "failed to commit tx", "tx_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"stats":   stats,
		})
	}
}

// handleExportLookupValues exports all lookup values for a specific dataset in CSV format
func handleExportLookupValues(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")
		if tenantID == "" {
			http.Error(w, "tenant_id is required", http.StatusBadRequest)
			return
		}
		lookupID := chi.URLParam(r, "id")
		if lookupID == "" {
			http.Error(w, "lookup id is required", http.StatusBadRequest)
			return
		}

		// Get lookup name for filename
		var lookupName string
		err := db.QueryRow(`SELECT name FROM lookups WHERE id = $1`, lookupID).Scan(&lookupName)
		if err != nil {
			http.Error(w, "Lookup not found: "+err.Error(), http.StatusNotFound)
			return
		}

		// Query all values for this lookup (no limit)
		var query string
		var args []interface{}
		if datasourceID != "" {
			// Filter by datasource if provided
			query = `SELECT id, value, label, COALESCE(parent_id, '') as parent_id, COALESCE(metadata::text, '{}') as metadata 
			         FROM lookup_values 
			         WHERE lookup_id = $1 AND tenant_id = $2 
			         ORDER BY label`
			args = []interface{}{lookupID, tenantID}
		} else {
			query = `SELECT id, value, label, COALESCE(parent_id, '') as parent_id, COALESCE(metadata::text, '{}') as metadata 
			         FROM lookup_values 
			         WHERE lookup_id = $1 AND tenant_id = $2 
			         ORDER BY label`
			args = []interface{}{lookupID, tenantID}
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			http.Error(w, "Failed to query values: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		// Set CSV headers
		w.Header().Set("Content-Type", "text/csv")
		w.Header().Set("Content-Disposition", "attachment; filename=\""+lookupName+".csv\"")

		// Write CSV header
		w.Write([]byte("ID,Value,Label,Parent ID,Metadata\n"))

		// Write CSV rows
		for rows.Next() {
			var id, value, label, parentID, metadata string
			if err := rows.Scan(&id, &value, &label, &parentID, &metadata); err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			// Escape CSV values (simple escaping - wrap in quotes if contains comma or quote)
			escapeCSV := func(s string) string {
				if len(s) == 0 {
					return ""
				}
				// If contains comma, newline, or quote, wrap in quotes and escape internal quotes
				if len(s) > 0 && (s[0] == '"' || ContainsAny(s, ",\n\"")) {
					return `"` + ReplaceAll(s, `"`, `""`) + `"`
				}
				return s
			}
			line := escapeCSV(id) + "," + escapeCSV(value) + "," + escapeCSV(label) + "," + escapeCSV(parentID) + "," + escapeCSV(metadata) + "\n"
			w.Write([]byte(line))
		}
	}
}

// Helper functions for CSV escaping
func ContainsAny(s string, chars string) bool {
	for _, c := range chars {
		for _, sc := range s {
			if c == sc {
				return true
			}
		}
	}
	return false
}

func ReplaceAll(s string, old string, new string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += new
			i += len(old) - 1
		} else {
			result += string(s[i])
		}
	}
	return result
}
