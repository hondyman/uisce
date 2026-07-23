package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// HasuraClient defines the interface for Hasura GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// ReportEngine handles report generation from semantic views
type ReportEngine struct {
	db     *sql.DB
	hasura HasuraClient
}

// NewReportEngine creates a new report engine
func NewReportEngine(db *sql.DB) *ReportEngine {
	return &ReportEngine{db: db}
}

// NewReportEngineWithHasura creates a new report engine with Hasura support
func NewReportEngineWithHasura(db *sql.DB, hasura HasuraClient) *ReportEngine {
	return &ReportEngine{db: db, hasura: hasura}
}

// GetSemanticViews retrieves all semantic views for a tenant
func (re *ReportEngine) GetSemanticViews(ctx context.Context, tenantID string) ([]SemanticView, error) {
	if re.hasura != nil {
		views, err := re.getSemanticViewsWithHasura(ctx, tenantID)
		if err == nil {
			return views, nil
		}
		fmt.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// TODO: Hasura-first pattern already implemented via getSemanticViewsWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See getSemanticViewsWithHasura() for the Hasura query: query GetSemanticViews
	// SQL fallback
	rows, err := re.db.QueryContext(ctx, `
		SELECT id, name, description, tenant_id, entity_type, semantic_content, created_at, updated_at
		FROM semantic_views
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	var views []SemanticView
	for rows.Next() {
		var view SemanticView
		if err := rows.Scan(&view.ID, &view.Name, &view.Description, &view.TenantID,
			&view.EntityType, &view.SemanticContent, &view.CreatedAt, &view.UpdatedAt); err != nil {
			return nil, err
		}
		views = append(views, view)
	}

	return views, rows.Err()
}

func (re *ReportEngine) getSemanticViewsWithHasura(ctx context.Context, tenantID string) ([]SemanticView, error) {
	query := `
		query GetSemanticViews($tenantId: String!) {
			semantic_views(where: {tenant_id: {_eq: $tenantId}}, order_by: {created_at: desc}) {
				id
				name
				description
				tenant_id
				entity_type
				semantic_content
				created_at
				updated_at
			}
		}
	`

	variables := map[string]interface{}{
		"tenantId": tenantID,
	}

	resp, err := re.hasura.Query(query, variables)
	if err != nil {
		return nil, err
	}

	viewsData, ok := resp["semantic_views"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	var views []SemanticView
	for _, item := range viewsData {
		viewMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		v := SemanticView{}
		if id, ok := viewMap["id"].(string); ok {
			v.ID, _ = uuid.Parse(id)
		}
		if name, ok := viewMap["name"].(string); ok {
			v.Name = name
		}
		if description, ok := viewMap["description"].(string); ok {
			v.Description = description
		}
		if tenantID, ok := viewMap["tenant_id"].(string); ok {
			v.TenantID, _ = uuid.Parse(tenantID)
		}
		if entityType, ok := viewMap["entity_type"].(string); ok {
			v.EntityType = entityType
		}
		if semanticContent, ok := viewMap["semantic_content"]; ok {
			if contentJSON, err := json.Marshal(semanticContent); err == nil {
				v.SemanticContent = contentJSON
			}
		}
		if createdAt, ok := viewMap["created_at"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, createdAt); err == nil {
				v.CreatedAt = parsedTime
			}
		}
		if updatedAt, ok := viewMap["updated_at"].(string); ok {
			if parsedTime, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				v.UpdatedAt = parsedTime
			}
		}

		views = append(views, v)
	}

	return views, nil
}

// GetEntitiesFromView extracts draggable entities from a semantic view
func (re *ReportEngine) GetEntitiesFromView(ctx context.Context, viewID string) ([]SemanticEntity, error) {
	// Query the semantic view
	var content json.RawMessage
	err := re.db.QueryRowContext(ctx, `
		SELECT semantic_content FROM semantic_views WHERE id = $1
	`, viewID).Scan(&content)
	if err != nil {
		return nil, fmt.Errorf("failed to get view: %w", err)
	}

	// Parse JSON to extract entities
	var viewData map[string]interface{}
	if err := json.Unmarshal(content, &viewData); err != nil {
		return nil, fmt.Errorf("failed to parse view content: %w", err)
	}

	// Extract entities from view (schema-based extraction)
	entities := re.extractEntitiesFromSchema(viewData)
	return entities, nil
}

// extractEntitiesFromSchema extracts draggable entities from semantic view schema
func (re *ReportEngine) extractEntitiesFromSchema(schema map[string]interface{}) []SemanticEntity {
	var entities []SemanticEntity

	// Navigate schema to find all fields
	for key, value := range schema {
		if key == "_metadata" {
			continue
		}

		entity := SemanticEntity{
			ID:        uuid.New(),
			Name:      key,
			Path:      key,
			Droppable: true,
		}

		// Determine type and data type
		switch value.(type) {
		case map[string]interface{}:
			entity.Type = "relationship"
			entity.DataType = "object"
		case []interface{}:
			entity.Type = "collection"
			entity.DataType = "array"
		case float64:
			entity.Type = "measure"
			entity.DataType = "number"
		case string:
			entity.Type = "attribute"
			entity.DataType = "string"
		case bool:
			entity.Type = "attribute"
			entity.DataType = "boolean"
		default:
			entity.Type = "attribute"
			entity.DataType = "mixed"
		}

		entities = append(entities, entity)
	}

	return entities
}

// CreateReportTemplate creates a new report template from semantic views
func (re *ReportEngine) CreateReportTemplate(ctx context.Context, template *ReportTemplate) error {
	template.ID = uuid.New()
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()

	sectionsJSON, _ := json.Marshal(template.Sections)
	filtersJSON, _ := json.Marshal(template.Filters)
	rulesJSON, _ := json.Marshal(template.Rules)

	if re.hasura != nil {
		err := re.createReportTemplateWithHasura(ctx, template, sectionsJSON, filtersJSON, rulesJSON)
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// TODO: Hasura-first pattern already implemented via createReportTemplateWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See createReportTemplateWithHasura() for the Hasura mutation: mutation CreateReportTemplate
	// SQL fallback
	_, err := re.db.ExecContext(ctx, `
		INSERT INTO report_templates 
		(id, name, description, tenant_id, created_by, sections, filters, rules, refresh_interval, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, template.ID, template.Name, template.Description, template.TenantID, template.CreatedBy,
		string(sectionsJSON), string(filtersJSON), string(rulesJSON), template.RefreshInterval, template.IsActive,
		template.CreatedAt, template.UpdatedAt)

	return err
}

func (re *ReportEngine) createReportTemplateWithHasura(ctx context.Context, template *ReportTemplate, sectionsJSON, filtersJSON, rulesJSON []byte) error {
	mutation := `
		mutation CreateReportTemplate($template: report_templates_insert_input!) {
			insert_report_templates_one(object: $template) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"template": map[string]interface{}{
			"id":               template.ID.String(),
			"name":             template.Name,
			"description":      template.Description,
			"tenant_id":        template.TenantID.String(),
			"created_by":       template.CreatedBy.String(),
			"sections":         string(sectionsJSON),
			"filters":          string(filtersJSON),
			"rules":            string(rulesJSON),
			"refresh_interval": template.RefreshInterval,
			"is_active":        template.IsActive,
			"created_at":       template.CreatedAt.Format(time.RFC3339),
			"updated_at":       template.UpdatedAt.Format(time.RFC3339),
		},
	}

	_, err := re.hasura.Mutate(mutation, variables)
	return err
}

// GetReportTemplate retrieves a report template by ID
func (re *ReportEngine) GetReportTemplate(ctx context.Context, templateID string) (*ReportTemplate, error) {
	if re.hasura != nil {
		template, err := re.getReportTemplateWithHasura(ctx, templateID)
		if err == nil {
			return template, nil
		}
		fmt.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	var template ReportTemplate
	var sectionsJSON, filtersJSON, rulesJSON string

	err := re.db.QueryRowContext(ctx, `
		SELECT id, name, description, tenant_id, created_by, sections, filters, rules, refresh_interval, is_active, created_at, updated_at
		FROM report_templates WHERE id = $1
	`, templateID).Scan(&template.ID, &template.Name, &template.Description, &template.TenantID,
		&template.CreatedBy, &sectionsJSON, &filtersJSON, &rulesJSON,
		&template.RefreshInterval, &template.IsActive, &template.CreatedAt, &template.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal([]byte(sectionsJSON), &template.Sections)
	json.Unmarshal([]byte(filtersJSON), &template.Filters)
	json.Unmarshal([]byte(rulesJSON), &template.Rules)

	return &template, nil
}

func (re *ReportEngine) getReportTemplateWithHasura(ctx context.Context, templateID string) (*ReportTemplate, error) {
	query := `
		query GetReportTemplate($templateId: uuid!) {
			report_templates_by_pk(id: $templateId) {
				id
				name
				description
				tenant_id
				created_by
				sections
				filters
				rules
				refresh_interval
				is_active
				created_at
				updated_at
			}
		}
	`

	variables := map[string]interface{}{
		"templateId": templateID,
	}

	resp, err := re.hasura.Query(query, variables)
	if err != nil {
		return nil, err
	}

	templateData, ok := resp["report_templates_by_pk"].(map[string]interface{})
	if !ok || templateData == nil {
		return nil, fmt.Errorf("template not found")
	}

	template := &ReportTemplate{}
	if id, ok := templateData["id"].(string); ok {
		template.ID, _ = uuid.Parse(id)
	}
	if name, ok := templateData["name"].(string); ok {
		template.Name = name
	}
	if description, ok := templateData["description"].(string); ok {
		template.Description = description
	}
	if tenantID, ok := templateData["tenant_id"].(string); ok {
		template.TenantID, _ = uuid.Parse(tenantID)
	}
	if createdBy, ok := templateData["created_by"].(string); ok {
		template.CreatedBy, _ = uuid.Parse(createdBy)
	}
	if sectionsJSON, ok := templateData["sections"].(string); ok {
		json.Unmarshal([]byte(sectionsJSON), &template.Sections)
	}
	if filtersJSON, ok := templateData["filters"].(string); ok {
		json.Unmarshal([]byte(filtersJSON), &template.Filters)
	}
	if rulesJSON, ok := templateData["rules"].(string); ok {
		json.Unmarshal([]byte(rulesJSON), &template.Rules)
	}
	if refreshInterval, ok := templateData["refresh_interval"].(float64); ok {
		template.RefreshInterval = int(refreshInterval)
	}
	if isActive, ok := templateData["is_active"].(bool); ok {
		template.IsActive = isActive
	}
	if createdAt, ok := templateData["created_at"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, createdAt); err == nil {
			template.CreatedAt = parsedTime
		}
	}
	if updatedAt, ok := templateData["updated_at"].(string); ok {
		if parsedTime, err := time.Parse(time.RFC3339, updatedAt); err == nil {
			template.UpdatedAt = parsedTime
		}
	}

	return template, nil
}

// AddSectionToTemplate adds a new section to a report template
func (re *ReportEngine) AddSectionToTemplate(ctx context.Context, templateID string, section ReportSection) error {
	template, err := re.GetReportTemplate(ctx, templateID)
	if err != nil {
		return err
	}

	section.ID = uuid.New()
	section.Order = len(template.Sections) + 1
	template.Sections = append(template.Sections, section)

	sectionsJSON, _ := json.Marshal(template.Sections)

	if re.hasura != nil {
		err = re.updateReportFieldWithHasura(ctx, templateID, "sections", string(sectionsJSON))
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// TODO: Hasura-first pattern already implemented via updateReportFieldWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See updateReportFieldWithHasura() for the Hasura mutation: mutation UpdateReportTemplate
	// SQL fallback
	_, err = re.db.ExecContext(ctx, `
		UPDATE report_templates SET sections = $1, updated_at = $2 WHERE id = $3
	`, string(sectionsJSON), time.Now(), templateID)

	return err
}

// ApplyFilterToTemplate applies a filter to a report template
func (re *ReportEngine) ApplyFilterToTemplate(ctx context.Context, templateID string, filter ReportFilter) error {
	template, err := re.GetReportTemplate(ctx, templateID)
	if err != nil {
		return err
	}

	filter.ID = uuid.New()
	template.Filters = append(template.Filters, filter)

	filtersJSON, _ := json.Marshal(template.Filters)

	if re.hasura != nil {
		err = re.updateReportFieldWithHasura(ctx, templateID, "filters", string(filtersJSON))
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// TODO: Hasura-first pattern already implemented via updateReportFieldWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See updateReportFieldWithHasura() for the Hasura mutation: mutation UpdateReportTemplate
	// SQL fallback
	_, err = re.db.ExecContext(ctx, `
		UPDATE report_templates SET filters = $1, updated_at = $2 WHERE id = $3
	`, string(filtersJSON), time.Now(), templateID)

	return err
}

// ApplyRuleToTemplate applies a business rule to a report template
func (re *ReportEngine) ApplyRuleToTemplate(ctx context.Context, templateID string, rule ReportRule) error {
	template, err := re.GetReportTemplate(ctx, templateID)
	if err != nil {
		return err
	}

	rule.ID = uuid.New()
	template.Rules = append(template.Rules, rule)

	rulesJSON, _ := json.Marshal(template.Rules)

	if re.hasura != nil {
		err = re.updateReportFieldWithHasura(ctx, templateID, "rules", string(rulesJSON))
		if err == nil {
			return nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// TODO: Hasura-first pattern already implemented via updateReportFieldWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See updateReportFieldWithHasura() for the Hasura mutation: mutation UpdateReportTemplate
	// SQL fallback
	_, err = re.db.ExecContext(ctx, `
		UPDATE report_templates SET rules = $1, updated_at = $2 WHERE id = $3
	`, string(rulesJSON), time.Now(), templateID)

	return err
}

// updateReportFieldWithHasura updates a specific field in a report template using Hasura
func (re *ReportEngine) updateReportFieldWithHasura(ctx context.Context, templateID, fieldName, value string) error {
	mutation := `
		mutation UpdateReportTemplate($templateId: uuid!, $updates: report_templates_set_input!) {
			update_report_templates_by_pk(pk_columns: {id: $templateId}, _set: $updates) {
				id
			}
		}
	`

	updates := map[string]interface{}{
		fieldName:    value,
		"updated_at": time.Now().Format(time.RFC3339),
	}

	variables := map[string]interface{}{
		"templateId": templateID,
		"updates":    updates,
	}

	_, err := re.hasura.Mutate(mutation, variables)
	return err
}

// GenerateReportFromTemplate generates a report instance from a template
func (re *ReportEngine) GenerateReportFromTemplate(ctx context.Context, templateID string, additionalFilters []ReportFilter) (*ReportGeneration, error) {
	template, err := re.GetReportTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	generation := &ReportGeneration{
		ID:             uuid.New(),
		TemplateID:     uuid.MustParse(templateID),
		GeneratedAt:    time.Now(),
		FiltersApplied: append(template.Filters, additionalFilters...),
		Status:         "pending",
		CreatedAt:      time.Now(),
	}

	startTime := time.Now()

	// Execute template sections and collect data
	dataSnapshot := make(map[string]interface{})
	for _, section := range template.Sections {
		sectionData := re.executeSectionQuery(ctx, section, generation.FiltersApplied)
		dataSnapshot[section.Title] = sectionData
	}

	snapshotJSON, _ := json.Marshal(dataSnapshot)
	generation.DataSnapshot = snapshotJSON
	generation.ExecutionTime = int(time.Since(startTime).Milliseconds())
	generation.Status = "success"

	// Store generation in database
	filtersJSON, _ := json.Marshal(generation.FiltersApplied)

	if re.hasura != nil {
		err = re.saveReportGenerationWithHasura(ctx, generation, filtersJSON, snapshotJSON)
		if err != nil {
			fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
			// Fall through to SQL
		} else {
			return generation, nil
		}
	}

	// TODO: Hasura-first pattern already implemented via saveReportGenerationWithHasura()
	// Primary implementation uses Hasura GraphQL, this SQL is fallback only
	// See saveReportGenerationWithHasura() for the Hasura mutation: mutation SaveReportGeneration
	// SQL fallback
	_, err = re.db.ExecContext(ctx, `
		INSERT INTO report_generations
		(id, template_id, generated_at, filters_applied, data_snapshot, execution_time, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, generation.ID, generation.TemplateID, generation.GeneratedAt, string(filtersJSON),
		string(snapshotJSON), generation.ExecutionTime, generation.Status, generation.CreatedAt)

	if err != nil {
		generation.Status = "failed"
		s := err.Error()
		generation.ErrorMessage = &s
	}

	return generation, nil
}

func (re *ReportEngine) saveReportGenerationWithHasura(ctx context.Context, generation *ReportGeneration, filtersJSON, snapshotJSON []byte) error {
	mutation := `
		mutation SaveReportGeneration($generation: report_generations_insert_input!) {
			insert_report_generations_one(object: $generation) {
				id
			}
		}
	`

	variables := map[string]interface{}{
		"generation": map[string]interface{}{
			"id":              generation.ID.String(),
			"template_id":     generation.TemplateID.String(),
			"generated_at":    generation.GeneratedAt.Format(time.RFC3339),
			"filters_applied": string(filtersJSON),
			"data_snapshot":   string(snapshotJSON),
			"execution_time":  generation.ExecutionTime,
			"status":          generation.Status,
			"created_at":      generation.CreatedAt.Format(time.RFC3339),
		},
	}

	_, err := re.hasura.Mutate(mutation, variables)
	return err
}

// executeSectionQuery executes query for a report section
func (re *ReportEngine) executeSectionQuery(ctx context.Context, section ReportSection, filters []ReportFilter) map[string]interface{} {
	result := map[string]interface{}{
		"title":        section.Title,
		"type":         section.SectionType,
		"data":         []map[string]interface{}{},
		"row_count":    0,
		"aggregations": map[string]interface{}{},
	}

	// Build query based on dropped entities
	if len(section.DroppedEntities) == 0 {
		return result
	}

	// TODO: Build dynamic query from entities and filters
	// For now, return placeholder
	result["data"] = []map[string]interface{}{
		{"status": "query execution placeholder"},
	}

	return result
}

// GetEntityRelationships retrieves relationships between entities in semantic views
func (re *ReportEngine) GetEntityRelationships(ctx context.Context, viewID string) ([]EntityRelationship, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { entity_relationships(
	//   where: {_or: [{source_entity_id: {_cast: {String: {_like: $pattern}}}},
	//                  {target_entity_id: {_cast: {String: {_like: $pattern}}}}]}
	// ) { id source_entity_id target_entity_id relation_type relationship_key cardinality created_at }}
	// Note: Complex CAST + LIKE pattern may need custom SQL function or view
	rows, err := re.db.QueryContext(ctx, `
		SELECT id, source_entity_id, target_entity_id, relation_type, relationship_key, cardinality, created_at
		FROM entity_relationships
		WHERE (CAST(source_entity_id AS TEXT) LIKE $1 OR CAST(target_entity_id AS TEXT) LIKE $1)
	`, "%"+viewID+"%")
	if err != nil {
		return nil, fmt.Errorf("failed to query relationships: %w", err)
	}
	defer rows.Close()

	var relationships []EntityRelationship
	for rows.Next() {
		var rel EntityRelationship
		var sourceEntityIDStr, targetEntityIDStr string
		if err := rows.Scan(&rel.ID, &sourceEntityIDStr, &targetEntityIDStr,
			&rel.RelationType, &rel.RelationshipKey, &rel.Cardinality, &rel.CreatedAt); err != nil {
			return nil, err
		}
		rel.SourceEntityID, _ = uuid.Parse(sourceEntityIDStr)
		rel.TargetEntityID, _ = uuid.Parse(targetEntityIDStr)
		relationships = append(relationships, rel)
	}

	return relationships, rows.Err()
}

// ValidateDragDrop validates a drag-drop operation
func (re *ReportEngine) ValidateDragDrop(ctx context.Context, state DragDropState) (bool, error) {
	// Check if entity can be dropped in target section
	// Check data types, relationships, etc

	allowedActions := []string{}

	switch state.SourceEntity.EntityType {
	case "measure":
		allowedActions = []string{"add_to_table", "create_aggregation"}
	case "attribute":
		allowedActions = []string{"add_to_table", "create_filter", "group_by"}
	case "relationship":
		allowedActions = []string{"create_join", "add_to_table"}
	}

	state.AllowedActions = allowedActions
	return len(allowedActions) > 0, nil
}
