package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ValidationRuleResponse represents a single validation rule for API response
type ValidationRuleResponse struct {
	ID            string          `json:"id"`
	RuleName      string          `json:"rule_name"`
	RuleType      string          `json:"rule_type"`
	TargetEntity  string          `json:"target_entity"`
	SubEntityType *string         `json:"sub_entity_type,omitempty"`
	Severity      string          `json:"severity"`
	Description   string          `json:"description"`
	Condition     json.RawMessage `json:"condition"`
	IsActive      bool            `json:"is_active"`
	CreatedAt     string          `json:"created_at"`
}

// FacetOption represents a single facet option with count
type FacetOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// ListValidationRulesResponse represents the paginated response with facets
type ListValidationRulesResponse struct {
	Rules           []ValidationRuleResponse `json:"rules"`
	Total           int                      `json:"total"`
	Page            int                      `json:"page"`
	Limit           int                      `json:"limit"`
	HasMore         bool                     `json:"has_more"`
	EntityFacets    []FacetOption            `json:"entity_facets"`
	SubEntityFacets []FacetOption            `json:"sub_entity_facets"`
	RuleTypeFacets  []FacetOption            `json:"rule_type_facets"`
	SeverityFacets  []FacetOption            `json:"severity_facets"`
}

// ListValidationRulesHandler handles GET /api/validation-rules with pagination and facets
func ListValidationRulesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get tenant scope
		tenantID := r.URL.Query().Get("tenant_id")
		datasourceID := r.URL.Query().Get("datasource_id")
		if tenantID == "" || datasourceID == "" {
			http.Error(w, "Missing tenant_id or datasource_id", http.StatusBadRequest)
			return
		}

		// Parse pagination
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		if limit < 1 || limit > 100 {
			limit = 20
		}

		// Parse filters
		entities := parseCommaSeparated(r.URL.Query().Get("entities"))
		subEntities := parseCommaSeparated(r.URL.Query().Get("sub_entities"))
		ruleTypes := parseCommaSeparated(r.URL.Query().Get("rule_types"))
		severities := parseCommaSeparated(r.URL.Query().Get("severities"))
		searchQuery := r.URL.Query().Get("search")

		// DEBUG: Log the parsed entities
		fmt.Printf("DEBUG: entities parameter: %q, parsed: %v\n", r.URL.Query().Get("entities"), entities)

		// Build WHERE clause
		whereClause := "WHERE tenant_id = $1 AND datasource_id = $2"
		args := []interface{}{tenantID, datasourceID}
		argIndex := 3

		if len(entities) > 0 {
			placeholders := buildPlaceholders(argIndex, len(entities))
			whereClause += fmt.Sprintf(" AND (target_entity IN (%s)", strings.Join(placeholders, ","))
			for _, e := range entities {
				args = append(args, e)
			}
			argIndex += len(entities)

			// Also check if entity is in target_entities array using @> (contains) operator
			entityConditions := make([]string, len(entities))
			for i, e := range entities {
				entityConditions[i] = fmt.Sprintf("target_entities::jsonb @> $%d::jsonb", argIndex+i)
				args = append(args, fmt.Sprintf("\"%s\"", e)) // JSON string format for array element
			}
			whereClause += fmt.Sprintf(" OR (%s))", strings.Join(entityConditions, " OR "))
			argIndex += len(entities)
		}

		if len(subEntities) > 0 {
			placeholders := buildPlaceholders(argIndex, len(subEntities))
			whereClause += fmt.Sprintf(" AND sub_entity_type IN (%s)", strings.Join(placeholders, ","))
			for _, s := range subEntities {
				args = append(args, s)
			}
			argIndex += len(subEntities)
		}

		if len(ruleTypes) > 0 {
			placeholders := buildPlaceholders(argIndex, len(ruleTypes))
			whereClause += fmt.Sprintf(" AND rule_type IN (%s)", strings.Join(placeholders, ","))
			for _, t := range ruleTypes {
				args = append(args, t)
			}
			argIndex += len(ruleTypes)
		}

		if len(severities) > 0 {
			placeholders := buildPlaceholders(argIndex, len(severities))
			whereClause += fmt.Sprintf(" AND severity IN (%s)", strings.Join(placeholders, ","))
			for _, s := range severities {
				args = append(args, s)
			}
			argIndex += len(severities)
		}

		// Full-text search
		if searchQuery != "" {
			whereClause += fmt.Sprintf(" AND to_tsvector('english', rule_name || ' ' || description) @@ plainto_tsquery('english', $%d)", argIndex)
			args = append(args, searchQuery)
			argIndex++
		}

		// Get total count
		countQuery := fmt.Sprintf("SELECT COUNT(*) FROM validation_rules %s", whereClause)
		var total int
		if err := db.QueryRow(countQuery, args...).Scan(&total); err != nil {
			http.Error(w, "Error counting rules", http.StatusInternalServerError)
			return
		}

		// Get rules with pagination
		offset := (page - 1) * limit
		rulesQuery := fmt.Sprintf(`
			SELECT id, rule_name, rule_type, target_entity, sub_entity_type,
				   severity, description, condition, is_active, created_at
			FROM validation_rules
			%s
			ORDER BY created_at DESC
			LIMIT $%d OFFSET $%d
		`, whereClause, argIndex, argIndex+1)

		args = append(args, limit, offset)

		rows, err := db.Query(rulesQuery, args...)
		if err != nil {
			http.Error(w, "Error querying rules", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var rules []ValidationRuleResponse
		for rows.Next() {
			var rule ValidationRuleResponse
			var createdAt sql.NullTime
			var subEntityType sql.NullString
			var condition sql.NullString

			if err := rows.Scan(
				&rule.ID,
				&rule.RuleName,
				&rule.RuleType,
				&rule.TargetEntity,
				&subEntityType,
				&rule.Severity,
				&rule.Description,
				&condition,
				&rule.IsActive,
				&createdAt,
			); err != nil {
				http.Error(w, "Error scanning rule", http.StatusInternalServerError)
				return
			}

			if subEntityType.Valid {
				rule.SubEntityType = &subEntityType.String
			}
			if condition.Valid {
				rule.Condition = json.RawMessage(condition.String)
			}
			if createdAt.Valid {
				rule.CreatedAt = createdAt.Time.Format("2006-01-02T15:04:05Z")
			}

			rules = append(rules, rule)
		}

		// Get facets (use base where clause, not including search)
		entityFacets := getFacets(db, tenantID, datasourceID, entities, subEntities, ruleTypes, severities, "target_entity", searchQuery)
		subEntityFacets := getFacets(db, tenantID, datasourceID, entities, subEntities, ruleTypes, severities, "sub_entity_type", searchQuery)
		ruleTypeFacets := getFacets(db, tenantID, datasourceID, entities, subEntities, ruleTypes, severities, "rule_type", searchQuery)
		severityFacets := getFacets(db, tenantID, datasourceID, entities, subEntities, ruleTypes, severities, "severity", searchQuery)

		// Build response
		response := ListValidationRulesResponse{
			Rules:           rules,
			Total:           total,
			Page:            page,
			Limit:           limit,
			HasMore:         offset+limit < total,
			EntityFacets:    entityFacets,
			SubEntityFacets: subEntityFacets,
			RuleTypeFacets:  ruleTypeFacets,
			SeverityFacets:  severityFacets,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// getFacets retrieves facet counts for a specific dimension
func getFacets(
	db *sql.DB,
	tenantID, datasourceID string,
	entities, subEntities, ruleTypes, severities []string,
	facetDimension, searchQuery string,
) []FacetOption {
	var facets []FacetOption

	whereClause := "WHERE tenant_id = $1 AND datasource_id = $2"
	args := []interface{}{tenantID, datasourceID}
	argIndex := 3

	// Add filters (excluding the current facet dimension)
	if len(entities) > 0 && facetDimension != "target_entity" {
		placeholders := buildPlaceholders(argIndex, len(entities))
		whereClause += fmt.Sprintf(" AND target_entity IN (%s)", strings.Join(placeholders, ","))
		for _, e := range entities {
			args = append(args, e)
		}
		argIndex += len(entities)
	}

	if len(subEntities) > 0 && facetDimension != "sub_entity_type" {
		placeholders := buildPlaceholders(argIndex, len(subEntities))
		whereClause += fmt.Sprintf(" AND sub_entity_type IN (%s)", strings.Join(placeholders, ","))
		for _, s := range subEntities {
			args = append(args, s)
		}
		argIndex += len(subEntities)
	}

	if len(ruleTypes) > 0 && facetDimension != "rule_type" {
		placeholders := buildPlaceholders(argIndex, len(ruleTypes))
		whereClause += fmt.Sprintf(" AND rule_type IN (%s)", strings.Join(placeholders, ","))
		for _, t := range ruleTypes {
			args = append(args, t)
		}
		argIndex += len(ruleTypes)
	}

	if len(severities) > 0 && facetDimension != "severity" {
		placeholders := buildPlaceholders(argIndex, len(severities))
		whereClause += fmt.Sprintf(" AND severity IN (%s)", strings.Join(placeholders, ","))
		for _, s := range severities {
			args = append(args, s)
		}
		argIndex += len(severities)
	}

	// Add search filter
	if searchQuery != "" {
		whereClause += fmt.Sprintf(" AND to_tsvector('english', rule_name || ' ' || description) @@ plainto_tsquery('english', $%d)", argIndex)
		args = append(args, searchQuery)
		argIndex++
	}

	// Add condition to exclude NULL for sub_entity_type
	if facetDimension == "sub_entity_type" {
		whereClause += " AND sub_entity_type IS NOT NULL"
	}

	query := fmt.Sprintf(`
		SELECT %s as value, COUNT(*) as count
		FROM validation_rules
		%s
		GROUP BY %s
		ORDER BY count DESC, %s ASC
		LIMIT 50
	`, facetDimension, whereClause, facetDimension, facetDimension)

	rows, err := db.Query(query, args...)
	if err != nil {
		return facets
	}
	defer rows.Close()

	for rows.Next() {
		var value string
		var count int
		if err := rows.Scan(&value, &count); err != nil {
			continue
		}

		facet := FacetOption{
			Value: value,
			Label: formatFacetLabel(value, facetDimension),
			Count: count,
		}
		facets = append(facets, facet)
	}

	return facets
}

// Helper functions

func parseCommaSeparated(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func buildPlaceholders(startIndex, count int) []string {
	var placeholders []string
	for i := 0; i < count; i++ {
		placeholders = append(placeholders, fmt.Sprintf("$%d", startIndex+i))
	}
	return placeholders
}

func formatFacetLabel(value string, dimension string) string {
	switch dimension {
	case "rule_type":
		typeMap := map[string]string{
			"business_logic": "Business Logic",
			"field_format":   "Field Format",
			"cardinality":    "Cardinality",
			"referential":    "Referential Integrity",
			"uniqueness":     "Uniqueness",
		}
		if label, ok := typeMap[value]; ok {
			return label
		}
	}
	return value
}
