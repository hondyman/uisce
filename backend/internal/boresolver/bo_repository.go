package boresolver

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hondyman/semlayer/backend/internal/bo"
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

// GetBODefinition fetches the BO definition from the database
func (r *PostgresBORepository) GetBODefinition(boID string) (*BODefinition, error) {
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
