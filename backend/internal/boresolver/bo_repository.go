package boresolver

import (
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
	var schema, tbl string
	if idx := strings.Index(table, "."); idx >= 0 {
		schema = table[:idx]
		tbl = table[idx+1:]
	} else {
		schema = "public"
		tbl = table
	}
	cols := []string{}
	if err := r.DB.Select(&cols, `SELECT column_name FROM information_schema.columns WHERE table_schema=$1 AND table_name=$2`, schema, tbl); err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(cols))
	for _, c := range cols {
		set[c] = struct{}{}
	}
	return set, nil
}

// fetchFieldsFromBusinessObjectFields reads fields from the master business_object_fields table,
// joining with catalog_node to get semantic term information.
// This is the fallback when bo_fields returns no results for STI parent BOs.
func (r *PostgresBORepository) fetchFieldsFromBusinessObjectFields(boID string) ([]bo.BOField, error) {
	query := `
		SELECT
			bof.id,
			bof.tenant_id,
			bof.bo_id AS business_object_id,
			bof.field_name,
			COALESCE(bof.display_name, cn.node_name, bof.field_name) AS display_name,
			COALESCE(bof.technical_name, bof.field_name) AS technical_name,
			COALESCE(bof.data_type, 'string') AS type,
			bof.inherits_defaults AS is_core,
			bof.is_required,
			false AS is_readonly,
			false AS is_searchable,
			COALESCE(bof.description, '') AS description,
			COALESCE(bof.display_order, 0) AS sequence,
			COALESCE(bof.section_name, '') AS section,
			COALESCE(bof.default_value, '') AS default_value,
			COALESCE(bof.validation_rules, '{}') AS validation_rules,
			COALESCE(bof.reference_entity, '') AS reference_bo,
			COALESCE(bof.picklist_values, '{}') AS picklist_values,
			bof.created_at,
			bof.updated_at
		FROM public.business_object_fields bof
		LEFT JOIN catalog_node cn ON cn.id = bof.term_node_id
		WHERE bof.bo_id = $1
		ORDER BY bof.display_order
	`

	var fields []bo.BOField
	err := r.DB.Select(&fields, query, boID)
	if err != nil {
		return nil, err
	}
	return fields, nil
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

// GetBODefinition fetches the BO definition from the database
func (r *PostgresBORepository) GetBODefinition(boID string) (*BODefinition, error) {
	// 1. Fetch Fields
	// Try bo_fields first (legacy table), then fall back to business_object_fields
	// (the master table for STI BOs where subtype fields are attached to child BOs,
	// and semantic-term-based fields are populated via catalog sync).
	var boFields []bo.BOField

	// Legacy bo_fields queries
	bofQueries := []string{
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

	var bofErr error
	for _, q := range bofQueries {
		bofErr = r.DB.Select(&boFields, q, boID)
		if bofErr == nil {
			break
		}
	}

	// If bo_fields returned no results, fall back to business_object_fields (master table)
	if len(boFields) == 0 {
		bofErr = nil // clear the error if we're going to try the fallback
		boFields, bofErr = r.fetchFieldsFromBusinessObjectFields(boID)
	}

	if bofErr != nil {
		return nil, fmt.Errorf("failed to fetch bo fields: %w", bofErr)
	}

	// 2. Query BO Metadata with support for both legacy and STI schema (bo_key, bo_name vs name, technical_name)
	type BOQueryResult struct {
		ID              string  `db:"id"`
		Name            string  `db:"name"`
		TechnicalName   string  `db:"technical_name"`
		DriverTableName *string `db:"driver_table_name"`
	}
	var res BOQueryResult
	boMetaQueries := []string{
		`SELECT id,
		        COALESCE(bo_name, name, bo_key, '') AS name,
		        COALESCE(bo_key, technical_name, name, '') AS technical_name,
		        COALESCE(driver_table_name, bo_key, technical_name, name, '') AS driver_table_name
		 FROM public.business_objects
		 WHERE id = $1`,
		`SELECT id,
		        COALESCE(name, technical_name, '') AS name,
		        COALESCE(technical_name, name, '') AS technical_name,
		        COALESCE(driver_table_name, technical_name, name, '') AS driver_table_name
		 FROM public.business_objects
		 WHERE id = $1`,
		`SELECT id,
		        COALESCE(bo_name, bo_key, '') AS name,
		        COALESCE(bo_key, bo_name, '') AS technical_name,
		        COALESCE(bo_key, bo_name, '') AS driver_table_name
		 FROM public.business_objects
		 WHERE id = $1`,
	}
	var boErr error
	for _, bq := range boMetaQueries {
		boErr = r.DB.Get(&res, bq, boID)
		if boErr == nil {
			break
		}
	}
	if boErr != nil {
		return nil, fmt.Errorf("failed to fetch BO metadata: %w", boErr)
	}

	drivingTable := res.TechnicalName
	if res.DriverTableName != nil && *res.DriverTableName != "" {
		drivingTable = *res.DriverTableName
	}

	def := &BODefinition{
		ID:            res.ID,
		DrivingTable:  drivingTable,
		Fields:        make([]BOField, 0, len(boFields)),
		Relationships: make([]BORelationship, 0),
	}

	columns, err := r.getTableColumns(drivingTable)
	if err != nil {
		columns = nil
	}

	for _, field := range boFields {
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
				JoinType:   "LEFT",
				Conditions: []string{
					fmt.Sprintf("${SOURCE}.%s = ${TARGET}.id", joinColumn),
				},
			})
		}

		def.Fields = append(def.Fields, BOField{
			ID:             field.ID,
			Name:           field.Name,
			Path:           field.Name,
			PhysicalColumn: physicalColumn,
		})
	}

	return def, nil
}

// GetBusinessObjectBinding resolves the binding context for a BO.
func (r *PostgresBORepository) GetBusinessObjectBinding(boID, bindingID string) (*BOBinding, error) {
	query := `
		SELECT id, datasource_id, COALESCE(driver_table_name, technical_name, name) AS driving_table
		FROM public.business_objects
		WHERE id = $1::uuid
		LIMIT 1
	`
	var binding BOBinding
	if err := r.DB.Get(&binding, query, boID); err != nil {
		return nil, fmt.Errorf("failed to resolve binding for BO %s: %w", boID, err)
	}

	binding.BOID = boID
	if bindingID != "" {
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
			COALESCE(semantic_term_id::text, id) AS term_node_id,
			COALESCE(key, name, technical_name) AS term_key,
			COALESCE(name, key, technical_name) AS term_name,
			COALESCE(display_label, name, key, technical_name) AS display_name,
			COALESCE(description, '') AS description,
			COALESCE(field_type, 'string') AS data_type,
			COALESCE(field_role, 'DIMENSION') AS role,
			COALESCE(binding_status, 'RESOLVED') AS binding_status
		FROM public.bo_fields
		WHERE business_object_id = $1::uuid
		  AND COALESCE(binding_status, 'RESOLVED') = 'RESOLVED'
		ORDER BY display_order, name
	`
	rows, err := r.DB.Queryx(query, boID)
	if err != nil {
		return nil, fmt.Errorf("failed to load terms for BO %s: %w", boID, err)
	}
	defer rows.Close()

	var terms []SemanticTermView
	for rows.Next() {
		var t SemanticTermView
		if err := rows.Scan(
			&t.TermNodeID,
			&t.TermKey,
			&t.TermName,
			&t.DisplayName,
			&t.Description,
			&t.DataType,
			&t.Role,
			&t.BindingStatus,
		); err != nil {
			return nil, fmt.Errorf("failed to scan term: %w", err)
		}
		if t.Role == "MEASURE" {
			t.DefaultAggregation = "SUM"
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
	// First find the BO ID by technical name
	var boID string
	query := `
		SELECT id FROM business_objects
		WHERE tenant_id = $1::uuid
		AND datasource_id = $2::uuid
		AND name = $3
		LIMIT 1
	`
	err := r.DB.Get(&boID, query, tenantID, datasourceID, technicalName)
	if err != nil {
		return nil, fmt.Errorf("failed to find business object with name '%s': %w", technicalName, err)
	}

	// Then get the full definition
	return r.GetBODefinition(boID)
}
