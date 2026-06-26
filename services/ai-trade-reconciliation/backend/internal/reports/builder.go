package reports

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ReportBuilder handles building reports through semantic views and drag-drop
type ReportBuilder struct {
	db          *sql.DB
	cache       *TemplateCache
	metrics     *MetricsCollector
	auditLogger *AuditLogger
}

// NewReportBuilder creates a new report builder
func NewReportBuilder(db *sql.DB) *ReportBuilder {
	return &ReportBuilder{db: db}
}

// NewReportBuilderWithCache creates a report builder with caching enabled
func NewReportBuilderWithCache(db *sql.DB, cacheTTL time.Duration) *ReportBuilder {
	return &ReportBuilder{
		db:      db,
		cache:   NewTemplateCache(cacheTTL),
		metrics: NewMetricsCollector(),
	}
}

// NewReportBuilderWithAudit creates a report builder with audit logging
func NewReportBuilderWithAudit(db *sql.DB, auditQueueSize int) *ReportBuilder {
	return &ReportBuilder{
		db:          db,
		cache:       NewTemplateCache(5 * time.Minute),
		metrics:     NewMetricsCollector(),
		auditLogger: NewAuditLogger(db, auditQueueSize),
	}
}

// GetSemanticViewsForReporting retrieves all semantic views available for report building with validation
func (rb *ReportBuilder) GetSemanticViewsForReporting(ctx context.Context, tenantID string) ([]SemanticViewWithEntities, error) {
	// Validate inputs
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}
	if err := ValidateUUID(tenantID); err != nil {
		return nil, fmt.Errorf("invalid tenant ID: %w", err)
	}

	// TODO: Refactor to Hasura GraphQL
	// query { semantic_views(
	//   where: {tenant_id: {_eq: $tenantId}, is_published: {_eq: true}}
	//   order_by: {name: asc}
	// ) { id name description tenant_id entity_type semantic_content created_at updated_at }}
	// JSONB field: semantic_content
	rows, err := rb.db.QueryContext(ctx, `
		SELECT id, name, description, tenant_id, entity_type, semantic_content, created_at, updated_at
		FROM semantic_views
		WHERE tenant_id = $1 AND is_published = true
		ORDER BY name ASC
	`, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic views: %w", err)
	}
	defer rows.Close()

	var views []SemanticViewWithEntities
	for rows.Next() {
		var view SemanticViewWithEntities
		var content json.RawMessage

		if err := rows.Scan(&view.ID, &view.Name, &view.Description, &view.TenantID,
			&view.EntityType, &content, &view.CreatedAt, &view.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan semantic view: %w", err)
		}

		// Validate view has required fields
		if view.ID == uuid.Nil {
			return nil, fmt.Errorf("semantic view has nil ID")
		}
		if view.Name == "" {
			return nil, fmt.Errorf("semantic view has empty name")
		}

		// Extract draggable entities from semantic content
		entities, relationships, err := rb.extractEntitiesAndRelationships(content)
		if err != nil {
			return nil, fmt.Errorf("failed to extract entities from view %s: %w", view.ID, err)
		}

		view.DraggableEntities = entities
		view.EntityRelationships = relationships
		views = append(views, view)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating semantic views: %w", err)
	}

	return views, nil
}

// extractEntitiesAndRelationships extracts both entities and relationships from semantic content with validation
func (rb *ReportBuilder) extractEntitiesAndRelationships(content json.RawMessage) ([]DraggableEntity, []EntityRelationship, error) {
	if len(content) == 0 {
		return nil, nil, fmt.Errorf("semantic content is empty")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal semantic content: %w", err)
	}

	var entities []DraggableEntity
	var relationships []EntityRelationship
	entityMap := make(map[string]string) // key -> entity ID

	// Extract entities
	for key, value := range data {
		if key == "_metadata" || key == "_relationships" {
			continue
		}

		// Validate entity name
		if key == "" {
			continue // skip empty keys
		}

		entityID := uuid.New().String()
		entity := DraggableEntity{
			ID:                 entityID,
			Name:               key,
			Path:               key,
			Droppable:          true,
			AllowedDropActions: []string{"add_to_table", "create_filter", "create_aggregation", "create_rule"},
		}

		// Determine type and data type from value
		entity.Type, entity.DataType = rb.inferTypeFromValue(value)
		entities = append(entities, entity)
		entityMap[key] = entityID
	}

	// Extract relationships if defined in metadata
	if relData, ok := data["_relationships"]; ok {
		if relArray, ok := relData.([]interface{}); ok {
			for idx, rel := range relArray {
				if relMap, ok := rel.(map[string]interface{}); ok {
					source, _ := relMap["source"].(string)
					target, _ := relMap["target"].(string)
					relType, _ := relMap["type"].(string)

					// Validate relationship has required fields
					if source == "" || target == "" {
						continue // skip invalid relationships
					}

					if srcIDStr, ok := entityMap[source]; ok {
						if tgtIDStr, ok := entityMap[target]; ok {
							srcID, err := uuid.Parse(srcIDStr)
							if err != nil {
								continue // or log error
							}
							tgtID, err := uuid.Parse(tgtIDStr)
							if err != nil {
								continue // or log error
							}

							relationship := EntityRelationship{
								ID:              uuid.New(),
								SourceEntityID:  srcID,
								TargetEntityID:  tgtID,
								RelationType:    relType,
								RelationshipKey: fmt.Sprintf("%s_%s", source, target),
								Cardinality:     "one-to-many", // default
							}
							relationships = append(relationships, relationship)
						}
					}
				} else if idx == 0 {
					// Log if first relationship is invalid but continue
					continue
				}
			}
		}
	}

	return entities, relationships, nil
}

// inferTypeFromValue determines entity type and data type from JSON value using centralized mapping
func (rb *ReportBuilder) inferTypeFromValue(value interface{}) (string, string) {
	entityType := InferEntityType(value)
	dataType := InferDataType(value)
	return entityType, dataType
}

// DropEntityToSection handles dropping an entity onto a report section with validation and handlers
func (rb *ReportBuilder) DropEntityToSection(ctx context.Context, templateID string, dropState DragDropState) error {
	// Validate inputs
	if err := ValidateDragDropState(&dropState); err != nil {
		return fmt.Errorf("invalid drop state: %w", err)
	}
	if err := ValidateUUID(templateID); err != nil {
		return fmt.Errorf("invalid template ID: %w", err)
	}

	// Get template
	template, err := rb.GetReportTemplate(ctx, templateID)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Find section by ID
	sectionIndex, err := FindSectionByID(template.Sections, dropState.TargetSectionID)
	if err != nil {
		return fmt.Errorf("section lookup failed: %w", err)
	}

	// Route to appropriate handler based on action
	switch dropState.Action {
	case "add_to_table":
		handler := &AddToTableHandler{}
		if err := handler.Handle(&template.Sections[sectionIndex], dropState.SourceEntity, dropState.TargetSectionID); err != nil {
			return fmt.Errorf("failed to add entity to table: %w", err)
		}

	case "create_filter":
		filter := ReportFilter{
			ID:              uuid.New(),
			FilterType:      GetDefaultFilterType(dropState.SourceEntity.DataType),
			EntityID:        dropState.SourceEntity.EntityID,
			EntityName:      dropState.SourceEntity.EntityName,
			ApplyToSections: []string{dropState.TargetSectionID},
			DroppedFrom:     "drag_drop",
			Operator:        "and",
		}
		template.Filters = append(template.Filters, filter)

	case "create_aggregation":
		handler := &CreateAggregationHandler{}
		if err := handler.Handle(&template.Sections[sectionIndex], dropState.SourceEntity, dropState.TargetSectionID); err != nil {
			return fmt.Errorf("failed to create aggregation: %w", err)
		}

	case "create_rule":
		rule := ReportRule{
			ID:               uuid.New(),
			Name:             fmt.Sprintf("Rule for %s", dropState.SourceEntity.EntityName),
			Description:      fmt.Sprintf("Auto-generated rule from %s", dropState.SourceEntity.EntityName),
			EntitiesInvolved: []string{dropState.SourceEntity.EntityID},
			CreatedFrom: []DragDropEntity{
				{
					EntityID:   dropState.SourceEntity.EntityID,
					EntityName: dropState.SourceEntity.EntityName,
					EntityType: dropState.SourceEntity.EntityType,
				},
			},
			IsActive: true,
		}
		template.Rules = append(template.Rules, rule)

	default:
		return fmt.Errorf("unknown drop action: %s", dropState.Action)
	}

	// Save updated template
	if err := rb.SaveReportTemplate(ctx, template); err != nil {
		return fmt.Errorf("failed to save template after drop action: %w", err)
	}

	return nil
}

// getDefaultFilterType returns the default filter type for a data type
func (rb *ReportBuilder) getDefaultFilterType(dataType string) string {
	return GetDefaultFilterType(dataType)
}

// getDefaultAggregation returns the default aggregation for an entity type
func (rb *ReportBuilder) getDefaultAggregation(entityType string) string {
	return GetDefaultAggregation(entityType)
}

// GetReportTemplate retrieves a report template
func (rb *ReportBuilder) GetReportTemplate(ctx context.Context, templateID string) (*ReportTemplate, error) {
	// Try cache first if available
	if rb.cache != nil {
		if cached := rb.cache.Get(templateID); cached != nil {
			if rb.metrics != nil {
				rb.metrics.RecordCacheHit()
			}
			return cached.(*ReportTemplate), nil
		}
		if rb.metrics != nil {
			rb.metrics.RecordCacheMiss()
		}
	}

	timer := NewTimer()

	var template ReportTemplate
	var sectionsJSON, filtersJSON, rulesJSON string

	err := rb.db.QueryRowContext(ctx, `
		SELECT id, name, description, tenant_id, created_by, sections, filters, rules, refresh_interval, is_active, created_at, updated_at
		FROM report_templates WHERE id = $1
	`, templateID).Scan(&template.ID, &template.Name, &template.Description, &template.TenantID,
		&template.CreatedBy, &sectionsJSON, &filtersJSON, &rulesJSON,
		&template.RefreshInterval, &template.IsActive, &template.CreatedAt, &template.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Unmarshal JSON fields with proper error handling
	if sectionsJSON != "" {
		if err := json.Unmarshal([]byte(sectionsJSON), &template.Sections); err != nil {
			return nil, fmt.Errorf("failed to unmarshal sections: %w", err)
		}
	}
	if filtersJSON != "" {
		if err := json.Unmarshal([]byte(filtersJSON), &template.Filters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal filters: %w", err)
		}
	}
	if rulesJSON != "" {
		if err := json.Unmarshal([]byte(rulesJSON), &template.Rules); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
		}
	}

	// Record metrics
	if rb.metrics != nil {
		rb.metrics.RecordTemplateLoad(timer.Elapsed())
	}

	// Cache the result
	if rb.cache != nil {
		rb.cache.Set(templateID, &template)
	}

	return &template, nil
}

// SaveReportTemplate saves a report template to database
func (rb *ReportBuilder) SaveReportTemplate(ctx context.Context, template *ReportTemplate) error {
	if template == nil {
		return fmt.Errorf("template cannot be nil")
	}
	if template.ID == uuid.Nil {
		return fmt.Errorf("template ID is required")
	}
	if ctx == nil {
		return fmt.Errorf("context cannot be nil")
	}

	timer := NewTimer()
	template.UpdatedAt = time.Now()

	sectionsJSON, err := json.Marshal(template.Sections)
	if err != nil {
		return fmt.Errorf("failed to marshal sections: %w", err)
	}
	filtersJSON, err := json.Marshal(template.Filters)
	if err != nil {
		return fmt.Errorf("failed to marshal filters: %w", err)
	}
	rulesJSON, err := json.Marshal(template.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	// TODO: Refactor to Hasura GraphQL
	// mutation { update_report_templates_by_pk(
	//   pk_columns: {id: $id}
	//   _set: {sections: $sections, filters: $filters, rules: $rules, updated_at: $updated_at}
	// ) { id }}
	// JSONB fields: sections, filters, rules
	_, err = rb.db.ExecContext(ctx, `
		UPDATE report_templates 
		SET sections = $1, filters = $2, rules = $3, updated_at = $4
		WHERE id = $5
	`, string(sectionsJSON), string(filtersJSON), string(rulesJSON), template.UpdatedAt, template.ID)

	if err != nil {
		return fmt.Errorf("failed to save template: %w", err)
	}

	// Invalidate cache
	if rb.cache != nil {
		rb.cache.Delete(template.ID.String())
	}

	// Record metrics
	if rb.metrics != nil {
		rb.metrics.RecordTemplateSave(timer.Elapsed())
	}

	return nil
}

// GenerateReportFromTemplate generates a report instance from a template
func (rb *ReportBuilder) GenerateReportFromTemplate(ctx context.Context, templateID string, appliedFilters []ReportFilter) (*ReportData, error) {
	template, err := rb.GetReportTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}

	reportData := &ReportData{
		ID:         uuid.New(),
		TemplateID: template.ID,
		Sections:   make([]SectionData, len(template.Sections)),
		Metadata: map[string]interface{}{
			"generated_at":    time.Now(),
			"filters_applied": len(appliedFilters),
		},
	}

	// Generate data for each section
	for i, section := range template.Sections {
		sectionData := SectionData{
			ID:    section.ID,
			Title: section.Title,
			Type:  section.SectionType,
			Data:  []interface{}{}, // Placeholder - would be populated from actual data source
		}
		reportData.Sections[i] = sectionData
	}

	return reportData, nil
}
