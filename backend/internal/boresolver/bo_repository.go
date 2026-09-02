package boresolver

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/hondyman/uisce/backend/internal/bo"
	"github.com/jmoiron/sqlx"
)

var uuidLike = regexp.MustCompile(`^[0-9a-fA-F-]{36}$`)

func normalizeIdentifier(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Use last segment if fully qualified
	if idx := strings.LastIndex(s, "."); idx >= 0 {
		s = s[idx+1:]
	}
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", "_")
	s = strings.ReplaceAll(s, " ", "_")
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isUUIDLike(s string) bool {
	return uuidLike.MatchString(strings.TrimSpace(s))
}

func (r *PostgresBORepository) getTableColumns(table string) (map[string]struct{}, error) {
	cols := []string{}
	if err := r.DB.Select(&cols, `SELECT column_name FROM information_schema.columns WHERE table_schema='public' AND table_name=$1`, table); err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(cols))
	for _, c := range cols {
		set[c] = struct{}{}
	}
	return set, nil
}

func resolvePhysicalColumn(field bo.BOField, columns map[string]struct{}) string {
	// Candidate sources in priority order
	// Always prioritize TechnicalName as it's the most explicit column mapping
	candidates := []string{field.TechnicalName, field.Name, field.Key, field.DisplayName}
	for i, c := range candidates {
		candidate := normalizeIdentifier(c)
		if candidate == "" || isUUIDLike(candidate) {
			continue
		}

		// For the first candidate (TechnicalName), return it directly without column validation
		// because the BO may be tied to a different database than alpha
		if i == 0 && columns != nil {
			// Check if TechnicalName exists in the column set
			if _, ok := columns[candidate]; ok {
				return candidate
			}
			// If TechnicalName doesn't exist in alpha's column set,
			// still return it - it might be valid in the BO's actual datasource
			return candidate
		}

		if columns == nil {
			// If we don't have column info, return the candidate (usually TechnicalName)
			return candidate
		}
		if _, ok := columns[candidate]; ok {
			return candidate
		}
	}

	// If columns is nil, use normalized technical_name as fallback
	if columns == nil {
		if candidate := normalizeIdentifier(field.TechnicalName); candidate != "" && !isUUIDLike(candidate) {
			return candidate
		}
		if candidate := normalizeIdentifier(field.Name); candidate != "" && !isUUIDLike(candidate) {
			return candidate
		}
		return ""
	}

	// If we still haven't found anything, try keyword matching
	lowerName := strings.ToLower(field.Name)
	keywordMap := map[string]string{
		"name":    "name",
		"address": "address",
		"email":   "email",
		"phone":   "phone",
		"status":  "status",
	}
	for key, col := range keywordMap {
		if strings.Contains(lowerName, strings.TrimSpace(key)) {
			if _, ok := columns[col]; ok {
				return col
			}
		}
	}

	// Last resort: return a normalized non-uuid candidate if possible
	if candidate := normalizeIdentifier(field.Name); candidate != "" && !isUUIDLike(candidate) {
		return candidate
	}
	if candidate := normalizeIdentifier(field.TechnicalName); candidate != "" && !isUUIDLike(candidate) {
		return candidate
	}
	return ""
}

// PostgresBORepository implements BORepository using a Postgres database
type PostgresBORepository struct {
	DB *sqlx.DB
}

// NewPostgresBORepository creates a new repository
func NewPostgresBORepository(db *sqlx.DB) *PostgresBORepository {
	return &PostgresBORepository{DB: db}
}

// semanticField is one row of business_object_fields, the table that is
// actually FK'd to business_objects.id (verified: 100% of its rows join;
// bo_fields.business_object_id matches ZERO real business_objects rows in
// this environment - bo_fields is an orphaned table from an earlier BO
// metadata system and is kept below only as a legacy fallback).
type semanticField struct {
	ID            string `db:"id"`
	FieldName     string `db:"field_name"`
	TechnicalName string `db:"technical_name"`
	DisplayName   string `db:"display_name"`
	DataType      string `db:"data_type"`
	TermNodeID    string `db:"term_node_id"`
}

// resolveCatalogPhysicalColumn looks up a physical column for fieldName
// under drivingTable (e.g. "oms.customer" or "customers") via field_bindings
// (when a RESOLVED binding exists for fieldID) or, failing that, by matching
// against catalog_node rows scanned from the real schema
// (properties->>'is_physical_column' = 'true', qualified_path
// "/<schema>/<table>/<column>" - see 20260112-era catalog scan output).
// Returns ("", "") when nothing resolves; callers must treat that as
// "unbound", not guess.
func (r *PostgresBORepository) resolveCatalogPhysicalColumn(fieldID, drivingTable, fieldName, technicalName string) (physicalColumn string, sourceNodeID string) {
	// 1. A real, explicitly authored binding always wins.
	var boundCol, boundNode sql.NullString
	err := r.DB.QueryRow(`
		SELECT cn.node_name, cn.id::text
		FROM field_bindings fb
		JOIN catalog_node cn ON cn.id = fb.source_node_id
		WHERE fb.field_id = $1::uuid AND fb.binding_status = 'RESOLVED' AND fb.source_type = 'COLUMN'
		LIMIT 1
	`, fieldID).Scan(&boundCol, &boundNode)
	if err == nil && boundCol.Valid && boundCol.String != "" {
		return fmt.Sprintf("%s.%s", drivingTable, boundCol.String), boundNode.String
	}

	// 2. No explicit binding yet: try to match a scanned catalog_node physical
	// column under the driving table by name.
	schema, table := parseDrivingTable(drivingTable)

	for _, candidate := range []string{technicalName, fieldName} {
		norm := normalizeIdentifier(candidate)
		if norm == "" || isUUIDLike(norm) {
			continue
		}
		var nodeName, nodeID sql.NullString
		err := r.DB.QueryRow(`
			SELECT node_name, id::text
			FROM catalog_node
			WHERE qualified_path = '/' || $1 || '/' || $2 || '/' || $3
			  AND COALESCE(properties->>'is_physical_column', 'false') = 'true'
			LIMIT 1
		`, schema, table, norm).Scan(&nodeName, &nodeID)
		if err == nil && nodeName.Valid && nodeName.String != "" {
			return fmt.Sprintf("%s.%s", drivingTable, nodeName.String), nodeID.String
		}
	}

	return "", ""
}

// parseDrivingTable splits a driving table into (schema, table). drivingTable
// is a catalog qualified_path ("/schema/table", e.g. "/public/customers" —
// the same convention BuildFROMClause's sanitizeTableName already assumes),
// with a dot-separated "schema.table" form supported for back-compat with
// pre-scan-based driving tables.
func parseDrivingTable(drivingTable string) (schema, table string) {
	schema = "public"
	table = strings.Trim(drivingTable, "/")
	if idx := strings.LastIndex(table, "/"); idx >= 0 {
		schema = table[:idx]
		table = table[idx+1:]
	} else if idx := strings.LastIndex(drivingTable, "."); idx >= 0 {
		schema = drivingTable[:idx]
		table = drivingTable[idx+1:]
	}
	return schema, table
}

// HasPhysicalColumn reports whether columnName should be treated as present
// on drivingTable's physical table. Trusts scanned catalog_node metadata
// only when the table was actually scanned (has at least one physical
// column registered there) — an unscanned driving table (e.g. this
// platform's own internal tables, which are never run through the catalog
// scanner) falls back to "column exists" so tenant scoping isn't silently
// disabled for tables we simply have no metadata about.
func (r *PostgresBORepository) HasPhysicalColumn(drivingTable, columnName string) bool {
	schema, table := parseDrivingTable(drivingTable)

	var tableWasScanned bool
	if err := r.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM catalog_node
			WHERE qualified_path LIKE '/' || $1 || '/' || $2 || '/%'
			  AND COALESCE(properties->>'is_physical_column', 'false') = 'true'
		)
	`, schema, table).Scan(&tableWasScanned); err != nil || !tableWasScanned {
		return true
	}

	var exists bool
	err := r.DB.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM catalog_node
			WHERE qualified_path = '/' || $1 || '/' || $2 || '/' || $3
			  AND COALESCE(properties->>'is_physical_column', 'false') = 'true'
		)
	`, schema, table, columnName).Scan(&exists)
	return err == nil && exists
}

// GetBODefinition fetches the BO definition from the database
func (r *PostgresBORepository) GetBODefinition(boID string) (*BODefinition, error) {
	if def, err := r.getBODefinitionFromSemanticFields(boID); err == nil && def != nil {
		return def, nil
	}
	return r.getBODefinitionLegacy(boID)
}

// getBODefinitionFromSemanticFields loads fields from business_object_fields
// (the table correctly FK'd to business_objects) and resolves each one's
// physical column via resolveCatalogPhysicalColumn. Returns (nil, nil) - not
// an error - when the BO has no business_object_fields rows, so callers fall
// through to the legacy path.
func (r *PostgresBORepository) getBODefinitionFromSemanticFields(boID string) (*BODefinition, error) {
	var fields []semanticField
	err := r.DB.Select(&fields, `
		SELECT id, field_name, COALESCE(technical_name, '') AS technical_name,
		       COALESCE(display_name, field_name) AS display_name,
		       COALESCE(data_type, '') AS data_type,
		       term_node_id::text
		FROM public.business_object_fields
		WHERE bo_id = $1::uuid
		ORDER BY display_order, field_name
	`, boID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business_object_fields: %w", err)
	}
	if len(fields) == 0 {
		return nil, nil
	}

	var res struct {
		ID              string  `db:"id"`
		BOKey           string  `db:"bo_key"`
		DriverTableName *string `db:"driver_table_name"`
	}
	if err := r.DB.Get(&res, `
		SELECT id, bo_key, driver_table_name
		FROM public.business_objects WHERE id = $1
	`, boID); err != nil {
		return nil, fmt.Errorf("failed to fetch BO metadata: %w", err)
	}

	drivingTable := res.BOKey
	if res.DriverTableName != nil && *res.DriverTableName != "" {
		drivingTable = *res.DriverTableName
	}

	def := &BODefinition{
		ID:            res.ID,
		DrivingTable:  drivingTable,
		Fields:        make([]BOField, 0, len(fields)),
		Relationships: make([]BORelationship, 0),
	}

	for _, f := range fields {
		physicalColumn, _ := r.resolveCatalogPhysicalColumn(f.ID, drivingTable, f.FieldName, f.TechnicalName)
		def.Fields = append(def.Fields, BOField{
			ID:             f.ID,
			Name:           f.FieldName,
			DisplayName:    f.DisplayName,
			Path:           f.FieldName,
			SemanticTermID: f.TermNodeID,
			PhysicalColumn: physicalColumn,
			Type:           f.DataType,
		})
	}

	// Relationship/join inference isn't available from business_object_fields
	// today (no reference-BO FK column on this table, unlike legacy bo_fields'
	// reference_bo_id) - a BO loaded via this path has no auto-inferred joins
	// yet. Selecting a field whose PhysicalColumn is "" will surface a clear
	// resolution error rather than silently guessing.
	return def, nil
}

// getBODefinitionLegacy is the original bo_fields-based path, kept as a
// fallback for any BO that still only has legacy bo_fields rows (today, that
// matches zero real business_objects.id values in this environment, but the
// path is kept for forward/backward compatibility rather than deleted).
func (r *PostgresBORepository) getBODefinitionLegacy(boID string) (*BODefinition, error) {
	// 1. Fetch Fields (with explicit column mapping to avoid struct scan mismatches like "field_name")
	// Note: Some deployments have `sequence`, others use `display_order`, and older ones may have neither.
	// Try queries in order of preference and fall back to plain select if the ordered queries fail.
	var boFields []bo.BOField

	queries := []string{
		`SELECT id, tenant_id, business_object_id,
		        COALESCE(key, name, technical_name, field_name) AS key,
		        COALESCE(name, key, technical_name, field_name) AS name,
		        display_label AS display_name, 
		        technical_name,
		        field_type AS type,
		        is_core, is_required, is_readonly, is_searchable, COALESCE(description, '') AS description, display_order AS sequence, COALESCE(section_name, '') AS section,
		        COALESCE(default_value, '') AS default_value, '{}'::jsonb AS validation_rules, COALESCE(reference_bo_id::text, '') AS reference_bo, picklist_values, created_at, updated_at
		 FROM public.bo_fields
		 WHERE business_object_id = $1
		 ORDER BY display_order`,
		`SELECT id, tenant_id, business_object_id,
		        COALESCE(key, name, technical_name, field_name) AS key,
		        COALESCE(name, key, technical_name, field_name) AS name,
		        display_label AS display_name, 
		        technical_name,
		        field_type AS type,
		        is_core, is_required, is_readonly, is_searchable, COALESCE(description, '') AS description, display_order AS sequence, COALESCE(section_name, '') AS section,
		        COALESCE(default_value, '') AS default_value, '{}'::jsonb AS validation_rules, COALESCE(reference_bo_id::text, '') AS reference_bo, picklist_values, created_at, updated_at
		 FROM public.bo_fields
		 WHERE business_object_id = $1`,
	}

	var lastErr error
	for _, q := range queries {
		lastErr = r.DB.Select(&boFields, q, boID)
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("failed to fetch bo fields: %w", lastErr)
	}

	// 3. Construct BODefinition
	// Note: We are assuming 'TechnicalName' or 'Name' maps to the physical table for now.
	// If there is a separate mapping to the driving table, it should be in the BO definition.
	// The user request says "Driving table" is part of BO.
	// Looking at `listBusinessObjects` in `api.go`, it references `driver_table_name` column in query (lines 228-235),
	// but the `BusinessObject` struct in `bo/types.go` DOES NOT have `DrivingTable`.
	// Wait, the query in `api.go` (line 218) selects `display_name`, `description`, etc.
	// But `listBusinessObjects` logic (lines 228-236) CONDITIONALLY checks `driver_table_id` or `driver_table_name`.
	// It seems `public.business_objects` table HAS `driver_table_name` but `bo.BusinessObject` struct missed it?
	// Let's assume `TechnicalName` is the table name for now, or check if I should update the struct.
	// Or I can query it directly into a struct that has it.

	// Let's check if `driver_table_name` is in the DB schema by trying to select it.
	// I will use a struct distinct from `bo.BusinessObject` to be safe/complete.

	type BOMetadata struct {
		bo.BusinessObject
		DriverTableName *string `db:"driver_table_name"`
	}

	// var boMeta BOMetadata
	// Use a custom query for flexibility
	query := `
        SELECT id, name, technical_name, 
               driver_table_name 
        FROM public.business_objects 
        WHERE id = $1
    `
	// We only need a few fields for SQL generation
	type BOQueryResult struct {
		ID              string  `db:"id"`
		Name            string  `db:"name"`
		TechnicalName   string  `db:"technical_name"`
		DriverTableName *string `db:"driver_table_name"`
	}
	var res BOQueryResult
	if err := r.DB.Get(&res, query, boID); err != nil {
		// Fallback: maybe driver_table_name doesn't exist? Try without it?
		// But user said "Load BO Definition: driving_table".
		// Let's assume TechnicalName is the table if DriverTableName is missing.
		return nil, fmt.Errorf("failed to fetch BO metadata: %w", err)
	}

	drivingTable := res.TechnicalName
	if res.DriverTableName != nil && *res.DriverTableName != "" {
		drivingTable = *res.DriverTableName
	}

	def := &BODefinition{
		ID:            res.ID,
		DrivingTable:  drivingTable, // Use determined table
		Fields:        make([]BOField, 0, len(boFields)),
		Relationships: make([]BORelationship, 0),
	}

	columns, err := r.getTableColumns(drivingTable)
	if err != nil {
		columns = nil
	}

	for _, field := range boFields {
		// Map BOField to our internal BOField
		// Physical Column: user request says `physical_column` is in BO field.
		// `bo.BOField` doesn't have `PhysicalColumn` explicitly, but has `Name` and `TechnicalName`.
		// Usually `TechnicalName` is the column name in the driving table if it's a direct field.

		physicalColumnName := resolvePhysicalColumn(field, columns)
		physicalColumn := ""
		if physicalColumnName != "" {
			physicalColumn = fmt.Sprintf("%s.%s", drivingTable, physicalColumnName)
		}

		// If it is a reference, it defines a relationship
		if field.Type == bo.FieldTypeReference && field.ReferenceBO != "" {
			joinColumn := field.TechnicalName
			if physicalColumnName != "" {
				joinColumn = physicalColumnName
			}
			def.Relationships = append(def.Relationships, BORelationship{
				TargetBOID: field.ReferenceBO,
				JoinType:   "LEFT", // Default to Left Join
				// Condition: t0.field_id = t1.id (assuming t1 is the target BO's driving table)
				// We can't fully resolve the condition string here without knowing the target BO's table.
				// So we store the field info to resolve later.
				Conditions: []string{
					fmt.Sprintf("${SOURCE}.%s = ${TARGET}.id", joinColumn),
				},
			})
		}

		def.Fields = append(def.Fields, BOField{
			ID:             field.ID,
			Name:           field.Name,
			Path:           field.Name, // Using Name as path for now
			PhysicalColumn: physicalColumn,
			// SemanticTermID: we need to link to semantic term. `bo.BOField` doesn't have it explicitly?
			// `api.go` mentions `catalog_node_id` in `BusinessEntity`.
			// User request says "Fields (each mapped to a semantic term)".
			// I'll leave SemanticTermID empty for now, relying on direct physical mapping.
		})
	}

	return def, nil
}

// GetBusinessObjectBinding resolves the binding context for a BO. It prefers
// a real business_object_bindings row (bindingID if given, else the
// tenant's is_default row) and falls back to driver_table_name/bo_key on the
// business_objects row itself when no binding has been authored yet (true
// for every BO today - business_object_bindings has zero rows in this
// environment; see the backfill note in the SQL generator plan).
func (r *PostgresBORepository) GetBusinessObjectBinding(boID, bindingID string) (*BOBinding, error) {
	var binding BOBinding
	bindingQuery := `
		SELECT id::text AS binding_id, backend_type AS dialect_name,
		       COALESCE(alpha_product_id::text, '') AS alpha_product_id,
		       COALESCE(alpha_datasource_id::text, '') AS alpha_datasource_id
		FROM public.business_object_bindings
		WHERE bo_id = $1::uuid AND ($2 = '' OR id::text = $2)
		ORDER BY is_default DESC
		LIMIT 1
	`
	if err := r.DB.Get(&binding, bindingQuery, boID, bindingID); err == nil {
		binding.BOID = boID
	}

	var res struct {
		ID              string  `db:"id"`
		BOKey           string  `db:"bo_key"`
		DriverTableName *string `db:"driver_table_name"`
	}
	if err := r.DB.Get(&res, `
		SELECT id, bo_key, driver_table_name FROM public.business_objects WHERE id = $1::uuid LIMIT 1
	`, boID); err != nil {
		return nil, fmt.Errorf("failed to resolve binding for BO %s: %w", boID, err)
	}

	binding.BOID = boID
	binding.DrivingTable = res.BOKey
	if res.DriverTableName != nil && *res.DriverTableName != "" {
		binding.DrivingTable = *res.DriverTableName
	}
	if binding.BindingID == "" && bindingID != "" {
		binding.BindingID = bindingID
		binding.DatasourceID = bindingID
	}
	return &binding, nil
}

// GetBOTerms returns the semantic terms that are RESOLVED for the given BO/binding.
func (r *PostgresBORepository) GetBOTerms(boID, bindingID string) ([]SemanticTermView, error) {
	// TODO: when business_object_binding_fields exists, join to it and filter on
	// binding_status = 'RESOLVED' for the specific binding. Until then, we use the
	// bo_fields.binding_status column as the source of truth.
	query := `
		SELECT
			COALESCE(bf.semantic_term_id::text, bf.id) AS term_node_id,
			COALESCE(bf.key, bf.name, bf.technical_name) AS term_key,
			COALESCE(bf.name, bf.key, bf.technical_name) AS term_name,
			COALESCE(bf.display_label, bf.name, bf.key, bf.technical_name) AS display_name,
			COALESCE(bf.description, '') AS description,
			COALESCE(bf.field_type, 'string') AS data_type,
			COALESCE(bf.field_role, 'DIMENSION') AS role,
			COALESCE(bf.binding_status, 'RESOLVED') AS binding_status,
			COALESCE(ct.properties->'drill_path', '[]'::jsonb) AS drill_path_json
		FROM public.bo_fields bf
		LEFT JOIN public.catalog_node ct ON ct.id = bf.semantic_term_id
		WHERE bf.business_object_id = $1::uuid
		  AND COALESCE(bf.binding_status, 'RESOLVED') = 'RESOLVED'
		ORDER BY bf.display_order, bf.name
	`
	rows, err := r.DB.Queryx(query, boID)
	if err != nil {
		return nil, fmt.Errorf("failed to load terms for BO %s: %w", boID, err)
	}
	defer rows.Close()

	var terms []SemanticTermView
	for rows.Next() {
		var t SemanticTermView
		var drillPathJSON []byte
		if err := rows.Scan(
			&t.TermNodeID,
			&t.TermKey,
			&t.TermName,
			&t.DisplayName,
			&t.Description,
			&t.DataType,
			&t.Role,
			&t.BindingStatus,
			&drillPathJSON,
		); err != nil {
			return nil, fmt.Errorf("failed to scan term: %w", err)
		}
		if t.Role == "MEASURE" {
			t.DefaultAggregation = "SUM"
		}
		if len(drillPathJSON) > 0 {
			_ = json.Unmarshal(drillPathJSON, &t.DrillPath)
		}
		terms = append(terms, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// bindingID is accepted for forward compatibility but currently unused because
	// binding_status is stored on bo_fields.
	_ = bindingID
	return terms, nil
}

// GetBOByTechnicalName fetches a BO definition by its technical name
func (r *PostgresBORepository) GetBOByTechnicalName(technicalName, tenantID, datasourceID string) (*BODefinition, error) {
	// business_objects has no name/datasource_id columns on this schema;
	// bo_key (e.g. "oms.customer") plus tenant_id (and the gold-copy tenant,
	// same read pattern used elsewhere for core/custom BOs) is what actually
	// identifies a BO. datasourceID is accepted for interface compatibility
	// but unused here - business_objects.driver_table_name/driver_table_id,
	// not a datasource_id column, carries binding info on this schema.
	var boID string
	query := `
		SELECT id FROM business_objects
		WHERE bo_key = $1
		  AND (tenant_id = $2::uuid OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1))
		ORDER BY (tenant_id = $2::uuid) DESC
		LIMIT 1
	`
	err := r.DB.Get(&boID, query, technicalName, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to find business object with key '%s': %w", technicalName, err)
	}

	// Then get the full definition
	return r.GetBODefinition(boID)
}
