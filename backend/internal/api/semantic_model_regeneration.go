package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/lib/pq"
)

// ============================================================================
// Semantic Model Regeneration Service
// ============================================================================

// ModelRegenerationService manages semantic model regeneration
type ModelRegenerationService struct {
	db *sql.DB
}

// NewModelRegenerationService creates a new regeneration service
func NewModelRegenerationService(db *sql.DB) *ModelRegenerationService {
	return &ModelRegenerationService{
		db: db,
	}
}

// ============================================================================
// Data Structures
// ============================================================================

// ModelChangeDetection represents detected changes
type ModelChangeDetection struct {
	EntityAttributeID string          `json:"entity_attribute_id"`
	ChangeType        string          `json:"change_type"` // ATTRIBUTE_CHANGED, RELATIONSHIP_ADDED, etc.
	ChangeDetail      json.RawMessage `json:"change_detail"`
	ChangedFields     []string        `json:"changed_fields"`
	AffectedCount     int             `json:"affected_count"`
	RequiresSync      bool            `json:"requires_sync"`
}

// SemanticModelSignature represents a model version signature
type SemanticModelSignature struct {
	EntityAttributeID string          `json:"entity_attribute_id"`
	VersionNumber     int             `json:"version_number"`
	Signature         string          `json:"signature"`
	GeneratedAt       time.Time       `json:"generated_at"`
	ModelContent      json.RawMessage `json:"model_content"`
}

// ModelRegenerationRequest represents a regeneration request
type ModelRegenerationRequest struct {
	EntityAttributeID   string          `json:"entity_attribute_id"`
	TriggerType         string          `json:"trigger_type"`
	TriggerSource       string          `json:"trigger_source"`
	ChangeDetail        json.RawMessage `json:"change_detail"`
	Priority            int             `json:"priority"`
	RequestedBy         string          `json:"requested_by"`
	Reason              string          `json:"reason"`
	DependsOnRequestIDs []string        `json:"depends_on_request_ids"`
}

// SemanticModel represents a semantic model
type SemanticModel struct {
	ID               string              `json:"id"`
	EntityID         string              `json:"entity_id"`
	EntityName       string              `json:"entity_name"`
	SemanticTermID   string              `json:"semantic_term_id"`
	SemanticTermName string              `json:"semantic_term_name"`
	VersionNumber    int                 `json:"version_number"`
	ModelSignature   string              `json:"model_signature"`
	Attributes       []ModelAttribute    `json:"attributes"`
	Relationships    []ModelRelationship `json:"relationships"`
	Metrics          []ModelMetric       `json:"metrics"`
	GeneratedAt      time.Time           `json:"generated_at"`
	LastModified     time.Time           `json:"last_modified"`
	IsPublished      bool                `json:"is_published"`
}

// ModelAttribute represents an entity attribute in the model
type ModelAttribute struct {
	Name           string  `json:"name"`
	DisplayName    string  `json:"display_name"`
	DataType       string  `json:"data_type"`
	IsPrimaryKey   bool    `json:"is_primary_key"`
	IsForeignKey   bool    `json:"is_foreign_key"`
	SemanticTermID string  `json:"semantic_term_id"`
	SemanticName   string  `json:"semantic_name"`
	Confidence     float64 `json:"confidence"`
}

// ModelRelationship represents a relationship in the model
type ModelRelationship struct {
	TargetEntityID   string  `json:"target_entity_id"`
	TargetEntityName string  `json:"target_entity_name"`
	RelationType     string  `json:"relation_type"`
	Cardinality      string  `json:"cardinality"`
	FKConstraint     string  `json:"fk_constraint"`
	IsDiscovered     bool    `json:"is_discovered"`
	Confidence       float64 `json:"confidence"`
}

// ModelMetric represents a metric in the model
type ModelMetric struct {
	Name        string `json:"name"`
	Expression  string `json:"expression"`
	DataType    string `json:"data_type"`
	Aggregation string `json:"aggregation"`
}

// ============================================================================
// Core Methods
// ============================================================================

// DetectModelChanges detects what changed in an entity
func (s *ModelRegenerationService) DetectModelChanges(
	ctx context.Context,
	tenantID, datasourceID, entityAttributeID string,
) (*ModelChangeDetection, error) {

	if entityAttributeID == "" {
		return nil, fmt.Errorf("entity attribute ID is required")
	}

	query := `
WITH attribute_change AS (
	-- Get latest changes to this entity attribute
	SELECT 
		eaa.entity_attribute_id,
		action,
		changed_fields,
		COUNT(*) as change_count
	FROM public.entity_attribute_audit eaa
	WHERE eaa.entity_attribute_id = $1::uuid
		AND eaa.tenant_datasource_id = $2::uuid
		AND eaa.changed_at >= NOW() - INTERVAL '1 hour'
	GROUP BY eaa.entity_attribute_id, eaa.action, eaa.changed_fields
	ORDER BY eaa.changed_at DESC
	LIMIT 1
),

relationship_change AS (
	-- Get latest relationship changes involving this entity
	SELECT 
		CASE 
			WHEN source_entity_id = $1::uuid THEN 'OUTBOUND'
			ELSE 'INBOUND'
		END as direction,
		COUNT(*) as change_count
	FROM public.model_regeneration_trigger
	WHERE (source_entity_id = $1::uuid OR target_entity_id = $1::uuid)
		AND tenant_datasource_id = $2::uuid
		AND trigger_type LIKE 'RELATIONSHIP_%'
		AND triggered_at >= NOW() - INTERVAL '1 hour'
	GROUP BY direction
)

SELECT 
	$1::uuid as entity_attribute_id,
	COALESCE(
		(SELECT action FROM attribute_change),
		(SELECT 'RELATIONSHIP_CHANGE' WHERE EXISTS(SELECT 1 FROM relationship_change)),
		'NO_CHANGE'
	) as change_type,
	COALESCE(
		(SELECT changed_fields FROM attribute_change),
		'[]'::text[]
	)::text[] as changed_fields,
	COALESCE((SELECT SUM(change_count) FROM attribute_change), 0) +
	COALESCE((SELECT SUM(change_count) FROM relationship_change), 0) as affected_count;
`

	var changeType string
	var changedFields pq.StringArray
	var affectedCount int

	err := s.db.QueryRowContext(ctx, query, entityAttributeID, datasourceID).Scan(
		&changeType, &changedFields, &affectedCount,
	)
	if err != nil && err != sql.ErrNoRows {
		logging.GetLogger().Sugar().Errorf("failed to detect model changes: %v", err)
		return nil, fmt.Errorf("failed to detect model changes: %w", err)
	}

	// Determine if sync is needed
	requiresSync := affectedCount > 0 && changeType != "NO_CHANGE"

	return &ModelChangeDetection{
		EntityAttributeID: entityAttributeID,
		ChangeType:        changeType,
		ChangedFields:     changedFields,
		AffectedCount:     affectedCount,
		RequiresSync:      requiresSync,
	}, nil
}

// CalculateModelSignature calculates SHA256 hash of model content
func (s *ModelRegenerationService) CalculateModelSignature(modelContent interface{}) (string, error) {
	// Convert to JSON
	jsonBytes, err := json.Marshal(modelContent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal model content: %w", err)
	}

	// Calculate SHA256
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}

// HasModelChanged checks if model content actually changed
func (s *ModelRegenerationService) HasModelChanged(
	ctx context.Context,
	entityAttributeID string,
	newModelContent interface{},
) (bool, error) {

	// Calculate new signature
	newSignature, err := s.CalculateModelSignature(newModelContent)
	if err != nil {
		return false, err
	}

	// Get latest version signature
	query := `
SELECT model_signature
FROM model_version_history
WHERE entity_attribute_id = $1
	AND is_active = true
ORDER BY version_number DESC
LIMIT 1;
`

	var lastSignature string
	err = s.db.QueryRowContext(ctx, query, entityAttributeID).Scan(&lastSignature)
	if err == sql.ErrNoRows {
		// No previous version, so it's a new model (counts as changed)
		return true, nil
	}
	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to get last signature: %v", err)
		return false, fmt.Errorf("failed to get last signature: %w", err)
	}

	// Compare signatures
	changed := newSignature != lastSignature
	logging.GetLogger().Sugar().Infof(
		"model signature comparison for %s: old=%s, new=%s, changed=%v",
		entityAttributeID, lastSignature, newSignature, changed)

	return changed, nil
}

// TriggerModelRegeneration triggers model regeneration for an entity
func (s *ModelRegenerationService) TriggerModelRegeneration(
	ctx context.Context,
	tenantID, datasourceID string,
	request *ModelRegenerationRequest,
) (string, error) {

	if request.EntityAttributeID == "" || request.TriggerType == "" {
		return "", fmt.Errorf("entity attribute ID and trigger type are required")
	}

	triggerID := uuid.NewString()
	// Insert regeneration trigger
	query := `
INSERT INTO model_regeneration_trigger (
	id, tenant_id, tenant_datasource_id, entity_attribute_id,
	trigger_type, trigger_source, change_detail,
	triggered_by, regeneration_status
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'PENDING');
`

	_, err := s.db.ExecContext(
		ctx, query,
		triggerID,
		tenantID, datasourceID, request.EntityAttributeID,
		request.TriggerType, request.TriggerSource, request.ChangeDetail,
		request.RequestedBy,
	)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to create regeneration trigger: %v", err)
		return "", fmt.Errorf("failed to create regeneration trigger: %w", err)
	}

	queueID := uuid.NewString()
	// Add to regeneration queue
	queueQuery := `
INSERT INTO model_regeneration_queue (
	id, tenant_id, tenant_datasource_id, entity_attribute_id,
	queue_status, priority, reason,
	triggered_by_trigger_id
) VALUES ($1, $2, $3, $4, 'QUEUED', $5, $6, $7);
`

	_, err = s.db.ExecContext(
		ctx, queueQuery,
		queueID,
		tenantID, datasourceID, request.EntityAttributeID,
		request.Priority, request.Reason, triggerID,
	)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to add to regeneration queue: %v", err)
		return "", fmt.Errorf("failed to add to regeneration queue: %w", err)
	}

	logging.GetLogger().Sugar().Infof(
		"triggered model regeneration for entity %s (trigger: %s, queue: %s)",
		request.EntityAttributeID, triggerID, queueID)

	return queueID, nil
}

// GenerateSemanticModel generates or regenerates a semantic model for an entity
func (s *ModelRegenerationService) GenerateSemanticModel(
	ctx context.Context,
	tenantID, datasourceID, entityAttributeID string,
) (*SemanticModel, error) {

	// Fetch entity details
	query := `
SELECT 
	ea.id,
	ea.name,
	ea.business_name,
	cn.id as semantic_term_id,
	cn.node_name as semantic_term_name
FROM public.entity_attribute ea
LEFT JOIN public.catalog_node cn ON ea.catalog_node_id = cn.id
WHERE ea.id = $1::uuid
	AND ea.tenant_id = $2::uuid
	AND ea.tenant_datasource_id = $3::uuid;
`

	var entityID, entityName, businessName string
	var semanticTermID, semanticTermName sql.NullString

	err := s.db.QueryRowContext(ctx, query, entityAttributeID, tenantID, datasourceID).Scan(
		&entityID, &entityName, &businessName, &semanticTermID, &semanticTermName,
	)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to fetch entity details: %v", err)
		return nil, fmt.Errorf("failed to fetch entity details: %w", err)
	}

	// Fetch attributes
	attributes, err := s.fetchEntityAttributes(ctx, entityAttributeID, datasourceID)
	if err != nil {
		return nil, err
	}

	// Fetch relationships
	relationships, err := s.fetchEntityRelationships(ctx, entityAttributeID, datasourceID)
	if err != nil {
		return nil, err
	}

	// Build semantic model
	model := &SemanticModel{
		ID:            entityID,
		EntityID:      entityID,
		EntityName:    entityName,
		Attributes:    attributes,
		Relationships: relationships,
		GeneratedAt:   time.Now(),
		LastModified:  time.Now(),
	}

	if semanticTermID.Valid {
		model.SemanticTermID = semanticTermID.String
		model.SemanticTermName = semanticTermName.String
	}

	// Calculate signature
	signature, err := s.CalculateModelSignature(model)
	if err != nil {
		return nil, err
	}
	model.ModelSignature = signature

	return model, nil
}

// fetchEntityAttributes fetches attributes for an entity
func (s *ModelRegenerationService) fetchEntityAttributes(
	ctx context.Context,
	entityAttributeID string,
	datasourceID string,
) ([]ModelAttribute, error) {

	query := `
SELECT 
	eacm.column_name,
	mc.display_name,
	mc.data_type,
	eacm.is_primary_key,
	eacm.is_foreign_key,
	cn.id as semantic_term_id,
	cn.node_name as semantic_name,
	eacm.confidence
FROM public.entity_attribute_column_mapping eacm
LEFT JOIN public.metadata_columns mc ON eacm.metadata_column_id = mc.id
LEFT JOIN public.catalog_node cn ON eacm.semantic_term_id = cn.id
WHERE eacm.entity_attribute_id = $1::uuid
	AND eacm.tenant_datasource_id = $2::uuid
ORDER BY eacm.is_primary_key DESC, eacm.column_name;
`

	rows, err := s.db.QueryContext(ctx, query, entityAttributeID, datasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to fetch attributes: %v", err)
		return nil, fmt.Errorf("failed to fetch attributes: %w", err)
	}
	defer rows.Close()

	var attributes []ModelAttribute
	for rows.Next() {
		var attr ModelAttribute
		var semanticID, semanticName sql.NullString
		var confidence sql.NullFloat64

		err := rows.Scan(
			&attr.Name, &attr.DisplayName, &attr.DataType,
			&attr.IsPrimaryKey, &attr.IsForeignKey,
			&semanticID, &semanticName, &confidence,
		)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to scan attribute: %v", err)
			continue
		}

		attr.SemanticTermID = semanticID.String
		attr.SemanticName = semanticName.String
		if confidence.Valid {
			attr.Confidence = confidence.Float64
		}

		attributes = append(attributes, attr)
	}

	return attributes, rows.Err()
}

// fetchEntityRelationships fetches relationships for an entity
func (s *ModelRegenerationService) fetchEntityRelationships(
	ctx context.Context,
	entityAttributeID string,
	datasourceID string,
) ([]ModelRelationship, error) {

	query := `
SELECT 
	target_entity_id,
	(SELECT name FROM public.entity_attribute WHERE id = er.target_entity_id) as target_entity_name,
	relationship_type,
	cardinality,
	fk_constraint,
	is_user_applied,
	confidence
FROM public.entity_relationship er
WHERE source_entity_id = $1::uuid
	AND tenant_datasource_id = $2::uuid
	AND is_active = true
ORDER BY confidence DESC, relationship_type;
`

	rows, err := s.db.QueryContext(ctx, query, entityAttributeID, datasourceID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to fetch relationships: %v", err)
		return nil, fmt.Errorf("failed to fetch relationships: %w", err)
	}
	defer rows.Close()

	var relationships []ModelRelationship
	for rows.Next() {
		var rel ModelRelationship
		var targetEntityName sql.NullString
		var isUserApplied sql.NullBool
		var confidence sql.NullFloat64

		err := rows.Scan(
			&rel.TargetEntityID, &targetEntityName, &rel.RelationType,
			&rel.Cardinality, &rel.FKConstraint, &isUserApplied, &confidence,
		)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to scan relationship: %v", err)
			continue
		}

		rel.TargetEntityName = targetEntityName.String
		if isUserApplied.Valid {
			rel.IsDiscovered = !isUserApplied.Bool
		}
		if confidence.Valid {
			rel.Confidence = confidence.Float64
		}

		relationships = append(relationships, rel)
	}

	return relationships, rows.Err()
}

// SaveModelVersion saves a generated model as a new version
func (s *ModelRegenerationService) SaveModelVersion(
	ctx context.Context,
	tenantID, datasourceID, entityAttributeID string,
	model *SemanticModel,
	changeSummary string,
	generatedBy string,
) (int, error) {

	modelJSON, err := json.Marshal(model)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal model: %w", err)
	}

	query := `
SELECT public.create_model_version(
	$1::uuid, $2::uuid, $3::uuid,
	$4::jsonb, $5, $6
)::text;
`

	var versionIDStr string
	err = s.db.QueryRowContext(
		ctx, query,
		entityAttributeID, tenantID, datasourceID,
		modelJSON, changeSummary, generatedBy,
	).Scan(&versionIDStr)

	if err != nil {
		logging.GetLogger().Sugar().Errorf("failed to save model version: %v", err)
		return 0, fmt.Errorf("failed to save model version: %w", err)
	}

	// Get version number
	versionQuery := `
SELECT version_number
FROM public.model_version_history
WHERE entity_attribute_id = $1::uuid
ORDER BY version_number DESC
LIMIT 1;
`

	var versionNumber int
	err = s.db.QueryRowContext(ctx, versionQuery, entityAttributeID).Scan(&versionNumber)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("failed to get version number: %v", err)
		versionNumber = 1
	}

	logging.GetLogger().Sugar().Infof(
		"saved model version %d for entity %s (id: %s)",
		versionNumber, model.EntityName, versionIDStr)

	return versionNumber, nil
}

// UpdateRegenerationStatus updates regeneration trigger status
func (s *ModelRegenerationService) UpdateRegenerationStatus(
	ctx context.Context,
	regenerationID, status string,
	errorMsg string,
) error {

	var query string
	if status == "COMPLETED" {
		query = `
UPDATE public.model_regeneration_trigger
SET regeneration_status = 'COMPLETED',
	regeneration_completed_at = NOW(),
	updated_at = NOW()
WHERE id = $1::uuid;
`
		_, err := s.db.ExecContext(ctx, query, regenerationID)
		return err
	} else if status == "FAILED" && errorMsg != "" {
		query = `
UPDATE public.model_regeneration_trigger
SET regeneration_status = 'FAILED',
	regeneration_error = $2,
	updated_at = NOW()
WHERE id = $1::uuid;
`
		_, err := s.db.ExecContext(ctx, query, regenerationID, errorMsg)
		return err
	}

	return fmt.Errorf("invalid status: %s", status)
}

// ============================================================================
// Helper Functions
// ============================================================================

// GetPendingRegenerations retrieves pending regeneration requests
func (s *ModelRegenerationService) GetPendingRegenerations(
	ctx context.Context,
	tenantID, datasourceID string,
	limit int,
) ([]ModelRegenerationRequest, error) {

	if limit == 0 {
		limit = 10
	}

	query := `
SELECT 
	entity_attribute_id,
	trigger_type,
	trigger_source,
	change_detail,
	triggered_by
FROM public.model_regeneration_trigger
WHERE tenant_datasource_id = $1::uuid
	AND regeneration_status = 'PENDING'
	AND is_active = true
ORDER BY triggered_at ASC
LIMIT $2;
`

	rows, err := s.db.QueryContext(ctx, query, datasourceID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending regenerations: %w", err)
	}
	defer rows.Close()

	var requests []ModelRegenerationRequest
	for rows.Next() {
		var req ModelRegenerationRequest
		var triggeredBy string

		err := rows.Scan(
			&req.EntityAttributeID, &req.TriggerType, &req.TriggerSource,
			&req.ChangeDetail, &triggeredBy,
		)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("failed to scan regeneration: %v", err)
			continue
		}

		req.RequestedBy = triggeredBy
		requests = append(requests, req)
	}

	return requests, rows.Err()
}

// GetModelVersion retrieves a specific model version
func (s *ModelRegenerationService) GetModelVersion(
	ctx context.Context,
	entityAttributeID string,
	versionNumber int,
) (*SemanticModel, error) {

	query := `
SELECT model_content
FROM model_version_history
WHERE entity_attribute_id = $1
	AND version_number = $2
	AND is_active = true
LIMIT 1;
`

	var modelValue any
	err := s.db.QueryRowContext(ctx, query, entityAttributeID, versionNumber).Scan(&modelValue)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("model version not found")
		}
		return nil, fmt.Errorf("failed to fetch model version: %w", err)
	}

	var modelJSON []byte
	switch v := modelValue.(type) {
	case []byte:
		modelJSON = v
	case string:
		modelJSON = []byte(v)
	default:
		return nil, fmt.Errorf("unexpected model_content type: %T", modelValue)
	}

	var model SemanticModel
	err = json.Unmarshal(modelJSON, &model)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal model: %w", err)
	}

	return &model, nil
}

// CompareModelVersions compares two model versions and returns differences
func (s *ModelRegenerationService) CompareModelVersions(
	ctx context.Context,
	entityAttributeID string,
	version1, version2 int,
) (map[string]interface{}, error) {

	model1, err := s.GetModelVersion(ctx, entityAttributeID, version1)
	if err != nil {
		return nil, err
	}

	model2, err := s.GetModelVersion(ctx, entityAttributeID, version2)
	if err != nil {
		return nil, err
	}

	differences := make(map[string]interface{})

	// Compare attributes
	differences["attributes_added"] = findNewAttributes(model1.Attributes, model2.Attributes)
	differences["attributes_removed"] = findRemovedAttributes(model1.Attributes, model2.Attributes)

	// Compare relationships
	differences["relationships_added"] = findNewRelationships(model1.Relationships, model2.Relationships)
	differences["relationships_removed"] = findRemovedRelationships(model1.Relationships, model2.Relationships)

	return differences, nil
}

// findNewAttributes finds attributes in model2 not in model1
func findNewAttributes(model1, model2 []ModelAttribute) []ModelAttribute {
	var new []ModelAttribute
	for _, attr2 := range model2 {
		found := false
		for _, attr1 := range model1 {
			if attr1.Name == attr2.Name {
				found = true
				break
			}
		}
		if !found {
			new = append(new, attr2)
		}
	}
	return new
}

// findRemovedAttributes finds attributes in model1 not in model2
func findRemovedAttributes(model1, model2 []ModelAttribute) []ModelAttribute {
	var removed []ModelAttribute
	for _, attr1 := range model1 {
		found := false
		for _, attr2 := range model2 {
			if attr1.Name == attr2.Name {
				found = true
				break
			}
		}
		if !found {
			removed = append(removed, attr1)
		}
	}
	return removed
}

// findNewRelationships finds relationships in model2 not in model1
func findNewRelationships(model1, model2 []ModelRelationship) []ModelRelationship {
	var new []ModelRelationship
	for _, rel2 := range model2 {
		found := false
		for _, rel1 := range model1 {
			if rel1.TargetEntityID == rel2.TargetEntityID {
				found = true
				break
			}
		}
		if !found {
			new = append(new, rel2)
		}
	}
	return new
}

// findRemovedRelationships finds relationships in model1 not in model2
func findRemovedRelationships(model1, model2 []ModelRelationship) []ModelRelationship {
	var removed []ModelRelationship
	for _, rel1 := range model1 {
		found := false
		for _, rel2 := range model2 {
			if rel1.TargetEntityID == rel2.TargetEntityID {
				found = true
				break
			}
		}
		if !found {
			removed = append(removed, rel1)
		}
	}
	return removed
}
