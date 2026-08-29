package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// relatedRelationship is the subset of business_object_relationships needed to resolve and
// scope a parent -> child record query/mutation.
type relatedRelationship struct {
	ID       string `db:"id"`
	FromBoID string `db:"from_bo_id"`
	ToBoID   string `db:"to_bo_id"`
	RelKey   string `db:"rel_key"`
}

// resolveRelationship finds the live business_object_relationships row for relKey, oriented so
// that the requesting BO (rootBoID) may be on either side of the edge.
func (h *BOCRUDHandler) resolveRelationship(ctx context.Context, tenantID, rootBoID, relKey string) (*relatedRelationship, error) {
	var rel relatedRelationship
	query := `
		SELECT id, from_bo_id, to_bo_id, rel_key
		FROM business_object_relationships
		WHERE tenant_id = $1 AND rel_key = $2 AND is_active = TRUE
		  AND (from_bo_id = $3 OR to_bo_id = $3)
		LIMIT 1;
	`
	if err := h.db.GetContext(ctx, &rel, query, tenantID, relKey, rootBoID); err != nil {
		return nil, fmt.Errorf("relationship '%s' not found: %w", relKey, err)
	}
	return &rel, nil
}

// resolveChildFKColumn finds the column on the child (target) driving table that holds the
// parent's id. Preference order: (1) an authored relationship_bindings.join_condition_sql,
// parsed for the child-side column; (2) the naming convention already relied on elsewhere in
// this file (parentBoName + "_id"), verified to actually exist on the child table.
func (h *BOCRUDHandler) resolveChildFKColumn(ctx context.Context, tenantID, relID, fromBoID, childTable, parentBoName string) (string, error) {
	var joinSQL string
	bindingQuery := `
		SELECT rb.join_condition_sql
		FROM relationship_bindings rb
		JOIN business_object_bindings bob ON bob.id = rb.binding_id
		WHERE rb.tenant_id = $1 AND rb.rel_id = $2 AND bob.bo_id = $3
		LIMIT 1;
	`
	if err := h.db.GetContext(ctx, &joinSQL, bindingQuery, tenantID, relID, fromBoID); err == nil && joinSQL != "" {
		if col := parseChildFKColumnFromJoinCondition(joinSQL, childTable); col != "" {
			return col, nil
		}
	}

	schema, table := splitSchemaTable(childTable)
	candidates := []string{parentBoName + "_id", "parent_id", "account_id", "sponsor_id"}
	for _, col := range candidates {
		var exists bool
		checkQuery := `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.columns
				WHERE table_schema = $1 AND table_name = $2 AND column_name = $3
			);
		`
		if err := h.db.GetContext(ctx, &exists, checkQuery, schema, table, col); err == nil && exists {
			return col, nil
		}
	}
	return "", fmt.Errorf("could not resolve a foreign key column on %s referencing %s", childTable, parentBoName)
}

// parseChildFKColumnFromJoinCondition extracts the bare column name on childTable's side of a
// "schema.table.column = schema.table.column"-style join predicate.
func parseChildFKColumnFromJoinCondition(joinSQL, childTable string) string {
	parts := strings.SplitN(joinSQL, "=", 2)
	if len(parts) != 2 {
		return ""
	}
	_, childBareTable := splitSchemaTable(childTable)
	for _, side := range parts {
		side = strings.TrimSpace(side)
		segments := strings.Split(side, ".")
		col := segments[len(segments)-1]
		if len(segments) >= 2 && strings.EqualFold(segments[len(segments)-2], childBareTable) {
			return col
		}
	}
	return ""
}

func splitSchemaTable(qualified string) (schema, table string) {
	if idx := strings.Index(qualified, "."); idx >= 0 {
		return qualified[:idx], qualified[idx+1:]
	}
	return "public", qualified
}

// HandleListRelatedRecords lists the "many" side of a relationship for one parent record.
func (h *BOCRUDHandler) HandleListRelatedRecords(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")
	recordID := chi.URLParam(r, "recordId")
	relKey := chi.URLParam(r, "relKey")

	rootBoID, err := h.resolveBusinessObjectID(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving business object id: %v", err), http.StatusNotFound)
		return
	}
	rel, err := h.resolveRelationship(r.Context(), tenantID.String(), rootBoID, relKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	childBoID := rel.ToBoID
	if childBoID == rootBoID {
		childBoID = rel.FromBoID
	}
	childKey, err := h.resolveBOKeyByID(r.Context(), childBoID, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving related business object: %v", err), http.StatusNotFound)
		return
	}
	childMeta, err := h.resolveBOMetadata(r.Context(), childKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving related BO contract: %v", err), http.StatusNotFound)
		return
	}
	fkColumn, err := h.resolveChildFKColumn(r.Context(), tenantID.String(), rel.ID, rel.FromBoID, childMeta.DrivingTable, boKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	query := fmt.Sprintf(`
		SELECT * FROM %s
		WHERE tenant_id = $1 AND %s = $2
		ORDER BY %s DESC
		LIMIT $3 OFFSET $4;
	`, childMeta.DrivingTable, fkColumn, childMeta.KeyColumn)

	rows, err := h.db.QueryxContext(r.Context(), query, tenantID, recordID, limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed listing related records: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	records := make([]map[string]interface{}, 0)
	for rows.Next() {
		item := make(map[string]interface{})
		if err := rows.MapScan(item); err == nil {
			cleanScanResult(item)
			records = append(records, item)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"records": records,
		"count":   len(records),
		"limit":   limit,
		"offset":  offset,
	})
}

// HandleCreateRelatedRecord creates a child record, forcing the FK to the parent server-side.
func (h *BOCRUDHandler) HandleCreateRelatedRecord(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")
	recordID := chi.URLParam(r, "recordId")
	relKey := chi.URLParam(r, "relKey")

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	rootBoID, err := h.resolveBusinessObjectID(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving business object id: %v", err), http.StatusNotFound)
		return
	}
	rel, err := h.resolveRelationship(r.Context(), tenantID.String(), rootBoID, relKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	childBoID := rel.ToBoID
	if childBoID == rootBoID {
		childBoID = rel.FromBoID
	}
	childKey, err := h.resolveBOKeyByID(r.Context(), childBoID, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving related business object: %v", err), http.StatusNotFound)
		return
	}
	childMeta, err := h.resolveBOMetadata(r.Context(), childKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving related BO contract: %v", err), http.StatusNotFound)
		return
	}
	fkColumn, err := h.resolveChildFKColumn(r.Context(), tenantID.String(), rel.ID, rel.FromBoID, childMeta.DrivingTable, boKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	// Force the FK value server-side, overriding whatever the client sent — closes off a
	// tampering vector where a client could target an unrelated parent record.
	payload[fkColumn] = recordID

	columns := []string{"tenant_id"}
	placeholders := []string{"$1"}
	args := []interface{}{tenantID}
	argIdx := 2
	for fieldKey, val := range payload {
		lower := strings.ToLower(fieldKey)
		if lower == "tenant_id" || lower == "created_at" || lower == "updated_at" {
			continue
		}
		columns = append(columns, fieldKey)
		placeholders = append(placeholders, fmt.Sprintf("$%d", argIdx))
		args = append(args, val)
		argIdx++
	}

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s (%s)
		VALUES (%s)
		RETURNING *;
	`, childMeta.DrivingTable, strings.Join(columns, ", "), strings.Join(placeholders, ", "))

	rows, err := h.db.QueryxContext(r.Context(), insertSQL, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed creating related record: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make(map[string]interface{})
	if rows.Next() {
		if err := rows.MapScan(result); err != nil {
			http.Error(w, "failed mapping created record", http.StatusInternalServerError)
			return
		}
	}
	cleanScanResult(result)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(result)
}

// HandleUpdateRelatedRecord updates a child record, scoped to both its own key and the parent FK.
func (h *BOCRUDHandler) HandleUpdateRelatedRecord(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")
	recordID := chi.URLParam(r, "recordId")
	relKey := chi.URLParam(r, "relKey")
	childID := chi.URLParam(r, "childId")

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	rootBoID, err := h.resolveBusinessObjectID(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving business object id: %v", err), http.StatusNotFound)
		return
	}
	rel, err := h.resolveRelationship(r.Context(), tenantID.String(), rootBoID, relKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	childBoID := rel.ToBoID
	if childBoID == rootBoID {
		childBoID = rel.FromBoID
	}
	childKey, err := h.resolveBOKeyByID(r.Context(), childBoID, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving related business object: %v", err), http.StatusNotFound)
		return
	}
	childMeta, err := h.resolveBOMetadata(r.Context(), childKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving related BO contract: %v", err), http.StatusNotFound)
		return
	}
	fkColumn, err := h.resolveChildFKColumn(r.Context(), tenantID.String(), rel.ID, rel.FromBoID, childMeta.DrivingTable, boKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	setClauses := make([]string, 0)
	args := []interface{}{tenantID, childID, recordID}
	argIdx := 4
	for fieldKey, val := range payload {
		lower := strings.ToLower(fieldKey)
		if lower == "id" || lower == "tenant_id" || lower == "created_at" || lower == "created_by" || fieldKey == fkColumn {
			continue
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", fieldKey, argIdx))
		args = append(args, val)
		argIdx++
	}
	if len(setClauses) == 0 {
		http.Error(w, "no writable attributes provided", http.StatusBadRequest)
		return
	}

	updateSQL := fmt.Sprintf(`
		UPDATE %s
		SET %s, updated_at = NOW()
		WHERE tenant_id = $1 AND %s = $2 AND %s = $3
		RETURNING *;
	`, childMeta.DrivingTable, strings.Join(setClauses, ", "), childMeta.KeyColumn, fkColumn)

	rows, err := h.db.QueryxContext(r.Context(), updateSQL, args...)
	if err != nil {
		http.Error(w, fmt.Sprintf("database mutation error: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	result := make(map[string]interface{})
	if rows.Next() {
		if err := rows.MapScan(result); err != nil {
			http.Error(w, "failed mapping updated record: "+err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, "record not found or does not belong to this parent", http.StatusNotFound)
		return
	}
	cleanScanResult(result)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// HandleDeleteRelatedRecord deletes a child record, scoped to both its own key and the parent FK.
func (h *BOCRUDHandler) HandleDeleteRelatedRecord(w http.ResponseWriter, r *http.Request) {
	tenantID := extractTenantUUIDFromRequest(r)
	boKey := chi.URLParam(r, "boKey")
	recordID := chi.URLParam(r, "recordId")
	relKey := chi.URLParam(r, "relKey")
	childID := chi.URLParam(r, "childId")

	rootBoID, err := h.resolveBusinessObjectID(r.Context(), boKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving business object id: %v", err), http.StatusNotFound)
		return
	}
	rel, err := h.resolveRelationship(r.Context(), tenantID.String(), rootBoID, relKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	childBoID := rel.ToBoID
	if childBoID == rootBoID {
		childBoID = rel.FromBoID
	}
	childKey, err := h.resolveBOKeyByID(r.Context(), childBoID, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving related business object: %v", err), http.StatusNotFound)
		return
	}
	childMeta, err := h.resolveBOMetadata(r.Context(), childKey, tenantID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed resolving related BO contract: %v", err), http.StatusNotFound)
		return
	}
	fkColumn, err := h.resolveChildFKColumn(r.Context(), tenantID.String(), rel.ID, rel.FromBoID, childMeta.DrivingTable, boKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	deleteSQL := fmt.Sprintf(`DELETE FROM %s WHERE tenant_id = $1 AND %s = $2 AND %s = $3`, childMeta.DrivingTable, childMeta.KeyColumn, fkColumn)
	res, err := h.db.ExecContext(r.Context(), deleteSQL, tenantID, childID, recordID)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed deleting related record: %v", err), http.StatusInternalServerError)
		return
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, "record not found or does not belong to this parent", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// resolveBusinessObjectID looks up the live business_objects.id for a boKey, tolerating either
// the bare key ("account") or a schema-qualified key ("oms.account") since callers use both.
func (h *BOCRUDHandler) resolveBusinessObjectID(ctx context.Context, boKey string, tenantID uuid.UUID) (string, error) {
	var id string
	query := `
		SELECT id::text FROM business_objects
		WHERE tenant_id = $1::uuid AND (bo_key = $2 OR bo_key LIKE '%%.' || $2)
		ORDER BY (bo_key = $2) DESC
		LIMIT 1;
	`
	if err := h.db.GetContext(ctx, &id, query, tenantID.String(), boKey); err != nil {
		return "", fmt.Errorf("business object '%s' not found: %w", boKey, err)
	}
	return id, nil
}

// resolveBOKeyByID is the inverse of resolveBusinessObjectID, needed to turn a relationship's
// from_bo_id/to_bo_id back into a boKey usable with resolveBOMetadata.
func (h *BOCRUDHandler) resolveBOKeyByID(ctx context.Context, boID string, tenantID uuid.UUID) (string, error) {
	var boKey string
	query := `SELECT bo_key FROM business_objects WHERE id = $1::uuid AND tenant_id = $2::uuid LIMIT 1;`
	if err := h.db.GetContext(ctx, &boKey, query, boID, tenantID.String()); err != nil {
		return "", fmt.Errorf("business object id '%s' not found: %w", boID, err)
	}
	return boKey, nil
}
