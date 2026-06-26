package api

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// createMetadataVersion creates a new version record for a semantic metadata change
func (s *Server) createMetadataVersion(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var req struct {
		BusinessObjectID string                 `json:"business_object_id"`
		ChangeType       string                 `json:"change_type"` // field_added, field_renamed, field_removed
		PreviousValue    map[string]interface{} `json:"previous_value"`
		NewValue         map[string]interface{} `json:"new_value"`
		CreatedBy        string                 `json:"created_by"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.BusinessObjectID == "" || req.ChangeType == "" {
		http.Error(w, "business_object_id and change_type are required", http.StatusBadRequest)
		return
	}

	if s.DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	// Get current version number
	var currentVersion int
	err := s.DB.QueryRowContext(r.Context(),
		`SELECT COALESCE(MAX(version), 0) FROM public.metadata_versions 
		 WHERE tenant_id = $1 AND business_object_id = $2`,
		tenantID, req.BusinessObjectID).Scan(&currentVersion)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get current version: %v", err), http.StatusInternalServerError)
		return
	}

	nextVersion := currentVersion + 1
	versionID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert new metadata version record
	_, err = s.DB.ExecContext(r.Context(),
		`INSERT INTO public.metadata_versions 
		 (id, tenant_id, business_object_id, version, created_at, created_by, change_type, previous_value, new_value)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		versionID, tenantID, req.BusinessObjectID, nextVersion, now, req.CreatedBy, req.ChangeType,
		json.RawMessage(mustMarshal(req.PreviousValue)),
		json.RawMessage(mustMarshal(req.NewValue)))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create version: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"version_id": versionID,
		"version":    nextVersion,
		"created_at": now,
	})
}

// getMetadataVersionHistory retrieves version history for a business object
func (s *Server) getMetadataVersionHistory(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	boID := chi.URLParam(r, "bo_id")
	if boID == "" {
		http.Error(w, "bo_id parameter required", http.StatusBadRequest)
		return
	}

	if s.DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT id, version, created_at, created_by, change_type, previous_value, new_value
		 FROM public.metadata_versions
		 WHERE tenant_id = $1 AND business_object_id = $2
		 ORDER BY version DESC`,
		tenantID, boID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch version history: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	versions := make([]MetadataVersion, 0)
	for rows.Next() {
		var v MetadataVersion
		var prevValue, newValue sql.NullString
		if err := rows.Scan(&v.VersionID, &v.Version, &v.CreatedAt, &v.CreatedBy, &v.ChangeType, &prevValue, &newValue); err != nil {
			continue
		}

		if prevValue.Valid {
			json.Unmarshal([]byte(prevValue.String), &v.PreviousValue)
		}
		if newValue.Valid {
			json.Unmarshal([]byte(newValue.String), &v.NewValue)
		}

		versions = append(versions, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

// createFieldAlias creates an alias for a field (for backward compatibility with renamed fields)
func (s *Server) createFieldAlias(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	var req struct {
		FieldID     string `json:"field_id"`
		OldName     string `json:"old_name"`
		RenamedBy   string `json:"renamed_by"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.FieldID == "" || req.OldName == "" {
		http.Error(w, "field_id and old_name are required", http.StatusBadRequest)
		return
	}

	if s.DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	aliasID := uuid.New().String()
	now := time.Now().UTC().Format(time.RFC3339)

	// Insert field alias
	_, err := s.DB.ExecContext(r.Context(),
		`INSERT INTO public.field_aliases 
		 (id, tenant_id, field_id, old_name, renamed_at, renamed_by, is_active, description)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		aliasID, tenantID, req.FieldID, req.OldName, now, req.RenamedBy, true, req.Description)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create alias: %v", err), http.StatusInternalServerError)
		return
	}

	// Refresh the semantic name resolver to pick up the new alias
	if s.SemanticNameResolver != nil {
		_ = s.SemanticNameResolver.Refresh(r.Context())
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alias_id":   aliasID,
		"field_id":   req.FieldID,
		"old_name":   req.OldName,
		"created_at": now,
	})
}

// getFieldAliases retrieves all aliases for a field
func (s *Server) getFieldAliases(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		http.Error(w, "X-Tenant-ID header required", http.StatusBadRequest)
		return
	}

	fieldID := chi.URLParam(r, "field_id")
	if fieldID == "" {
		http.Error(w, "field_id parameter required", http.StatusBadRequest)
		return
	}

	if s.DB == nil {
		http.Error(w, "Database not initialized", http.StatusInternalServerError)
		return
	}

	rows, err := s.DB.QueryContext(r.Context(),
		`SELECT id, old_name, renamed_at, renamed_by, is_active, description
		 FROM public.field_aliases
		 WHERE tenant_id = $1 AND field_id = $2
		 ORDER BY renamed_at DESC`,
		tenantID, fieldID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch aliases: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	aliases := make([]FieldAlias, 0)
	for rows.Next() {
		var a FieldAlias
		a.FieldID = fieldID
		if err := rows.Scan(&a.AliasID, &a.OldName, &a.RenamedAt, &a.RenamedBy, &a.IsActive, &a.Description); err != nil {
			continue
		}
		aliases = append(aliases, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(aliases)
}

// getSemanticNameResolverStats returns cache statistics
func (s *Server) getSemanticNameResolverStats(w http.ResponseWriter, r *http.Request) {
	if s.SemanticNameResolver == nil {
		http.Error(w, "SemanticNameResolver not initialized", http.StatusInternalServerError)
		return
	}

	stats := s.SemanticNameResolver.GetCacheStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Helper function
func mustMarshal(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
