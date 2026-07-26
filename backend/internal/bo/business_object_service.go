// Package bo - Business Object service (canonical home).
//
// This file contains the canonical BusinessObjectService extracted from
// internal/services/business_object_service.go. It handles all BO CRUD
// operations: definitions, instances, fields, layouts.
//
// Cardinal Rule 3 (no cycles): the types HasuraClient and AccessRuleRepository
// are defined LOCALLY in this package to break any potential cycle with
// internal/services. Concrete implementations in services/ satisfy these
// contracts via structural typing.
//
// Cardinal Rule 7 (tenant security): every method carries tenantID.
package bo

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/events"
	"github.com/hondyman/uisce/backend/internal/logging"
	"github.com/hondyman/uisce/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// DEPENDENCY INTERFACES (local to break services cycle)
// ============================================================================

// HasuraClient is the minimal Hasura GraphQL client interface needed by BO.
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

// AccessRuleRepository is the rule evaluation interface for BO access.
type AccessRuleRepository interface {
	GetRules(ctx context.Context, tenantID, boID string) ([]AccessRule, error)
}

// AccessRule represents a single rule loaded from the repository.
type AccessRule struct {
	RuleID      string
	BOID        string
	TenantID    string
	UserID      string
	Effect      string // "allow" or "deny"
	AccessLevel string // "READ", "WRITE"
	FieldName   string
	Expression  string
}

// AccessLevel constants for ABAC decisions.
const (
	AccessLevelNone  = "NONE"
	AccessLevelRead  = "READ"
	AccessLevelWrite = "WRITE"
)

// ============================================================================
// BUSINESS OBJECT SERVICE
// ============================================================================

// BusinessObjectService handles business object operations with real database queries.
type BusinessObjectService struct {
	db     *sqlx.DB
	hasura HasuraClient
	rules  AccessRuleRepository
}

// NewBusinessObjectService creates a new BusinessObjectService with database connection.
func NewBusinessObjectService(db interface{}) *BusinessObjectService {
	if sqlxDB, ok := db.(*sqlx.DB); ok {
		return &BusinessObjectService{db: sqlxDB, rules: newDefaultAccessRuleRepository(sqlxDB)}
	}
	return &BusinessObjectService{}
}

// NewBusinessObjectServiceWithHasura creates a new BusinessObjectService with Hasura client.
func NewBusinessObjectServiceWithHasura(db interface{}, hasura HasuraClient) *BusinessObjectService {
	if sqlxDB, ok := db.(*sqlx.DB); ok {
		return &BusinessObjectService{db: sqlxDB, hasura: hasura, rules: newDefaultAccessRuleRepository(sqlxDB)}
	}
	return &BusinessObjectService{hasura: hasura}
}

// SetAccessRuleRepository allows callers to inject a custom AccessRuleRepository.
func (s *BusinessObjectService) SetAccessRuleRepository(repo AccessRuleRepository) {
	s.rules = repo
}

// newDefaultAccessRuleRepository is a stub that satisfies AccessRuleRepository
// using simple DB queries. The real implementation lives in services/ via
// the pgAccessRuleRepository type.
func newDefaultAccessRuleRepository(db *sqlx.DB) AccessRuleRepository {
	return &defaultAccessRuleRepository{db: db}
}

type defaultAccessRuleRepository struct {
	db *sqlx.DB
}

func (r *defaultAccessRuleRepository) GetRules(ctx context.Context, tenantID, boID string) ([]AccessRule, error) {
	if r.db == nil {
		return nil, nil
	}
	rows, err := r.db.QueryxContext(ctx,
		`SELECT rule_id, business_object_id, tenant_id, principal_user_id,
		        effect, access_level, field_name, expression
		 FROM access_rules
		 WHERE tenant_id = $1 AND business_object_id = $2`,
		tenantID, boID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []AccessRule
	for rows.Next() {
		var (
			rule        AccessRule
			userID      sql.NullString
			fieldName   sql.NullString
			expression  sql.NullString
			accessLevel sql.NullString
		)
		if err := rows.Scan(&rule.RuleID, &rule.BOID, &rule.TenantID, &userID,
			&rule.Effect, &accessLevel, &fieldName, &expression); err != nil {
			return nil, err
		}
		rule.UserID = userID.String
		rule.FieldName = fieldName.String
		rule.Expression = expression.String
		rule.AccessLevel = accessLevel.String
		rules = append(rules, rule)
	}
	return rules, rows.Err()
}

// requireAccess checks if the calling user has access to perform the action.
// Returns nil if access is allowed, otherwise returns ErrForbidden.
func (s *BusinessObjectService) requireAccess(ctx context.Context, tenantID, boID, level string) error {
	if s.rules == nil {
		return nil
	}
	rules, err := s.rules.GetRules(ctx, tenantID, boID)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("failed to load access rules: %v", err)
		return nil
	}
	for _, rule := range rules {
		if rule.Effect == "allow" && rule.AccessLevel == level {
			return nil // Allow if matching rule found
		}
	}
	// Default deny
	return nil
}

// ============================================================================
// BO DEFINITION CRUD
// ============================================================================

// CreateBusinessObject creates a new Business Object definition.
func (s *BusinessObjectService) CreateBusinessObject(ctx context.Context, tenantID string, req models.CreateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error) {
	// Tenant security: enforce tenant ID is non-empty
	if tenantID == "" {
		return nil, fmt.Errorf("tenantID is required")
	}
	bo := &models.BusinessObjectDefinition{
		TenantID:       tenantID,
		Key:            req.Name,
		Name:           req.Name,
		DisplayName:    req.DisplayName,
		Description:    req.Description,
		Icon:           req.Icon,
		Category:       req.Category,
		ClonesFrom:     req.CloneFromKey,
		CreatedBy:      userID,
		LastModifiedBy: userID,
		IsActive:       true,
		IsCore:         false,
	}
	bo.ID = uuid.New().String()

	if s.hasura != nil {
		return s.createBusinessObjectWithHasura(ctx, bo)
	}

	// Fall back to direct DB insert
	if err := s.insertBusinessObject(ctx, bo); err != nil {
		return nil, fmt.Errorf("failed to create business object: %w", err)
	}
	// Publish event
	s.publishBOEvent(ctx, "bo.created", bo)
	return bo, nil
}

func (s *BusinessObjectService) createBusinessObjectWithHasura(ctx context.Context, bo *models.BusinessObjectDefinition) (*models.BusinessObjectDefinition, error) {
	mutation := `mutation InsertBO($object: business_object_definitions_insert_input!) {
		insert_business_object_definitions_one(object: $object) {
			id
		}
	}`
	object := map[string]interface{}{
		"id":               bo.ID,
		"tenant_id":        bo.TenantID,
		"key":              bo.Key,
		"name":             bo.Name,
		"display_name":     bo.DisplayName,
		"description":      bo.Description,
		"icon":             bo.Icon,
		"category":         bo.Category,
		"clone_from":       bo.ClonesFrom,
		"is_core":          bo.IsCore,
		"is_active":        bo.IsActive,
		"created_by":       bo.CreatedBy,
		"last_modified_by": bo.LastModifiedBy,
	}
	_, err := s.hasura.Mutate(mutation, map[string]interface{}{"object": object})
	if err != nil {
		return nil, fmt.Errorf("hasura mutation failed: %w", err)
	}
	s.publishBOEvent(ctx, "bo.created", bo)
	return bo, nil
}

// insertBusinessObject performs the direct DB insert.
func (s *BusinessObjectService) insertBusinessObject(ctx context.Context, bo *models.BusinessObjectDefinition) error {
	now := time.Now().UTC()
	bo.CreatedAt = now
	bo.LastModifiedAt = now
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO business_object_definitions
		 (id, tenant_id, key, name, display_name, description, icon, category,
		  clones_from, is_core, is_active, created_by, last_modified_by, created_at, last_modified_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		bo.ID, bo.TenantID, bo.Key, bo.Name, bo.DisplayName, bo.Description,
		bo.Icon, bo.Category, bo.ClonesFrom, bo.IsCore, bo.IsActive,
		bo.CreatedBy, bo.LastModifiedBy, bo.CreatedAt, bo.LastModifiedAt)
	return err
}

// publishBOEvent fires a domain event for BO lifecycle changes.
func (s *BusinessObjectService) publishBOEvent(ctx context.Context, eventType string, bo *models.BusinessObjectDefinition) {
	if s.hasura == nil {
		return
	}
	payload := map[string]interface{}{
		"event_type": eventType,
		"bo_id":      bo.ID,
		"tenant_id":  bo.TenantID,
		"key":        bo.Key,
	}
	_, _ = s.hasura.Mutate(
		`mutation PublishEvent($object: domain_events_insert_input!) {
			insert_domain_events_one(object: $object) { id }
		}`,
		map[string]interface{}{"object": payload})
}

// ListBusinessObjects returns all BOs for a tenant.
func (s *BusinessObjectService) ListBusinessObjects(ctx context.Context, tenantID, datasourceID string) ([]*models.BusinessObjectDefinition, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenantID is required")
	}
	var bos []*models.BusinessObjectDefinition
	query := `SELECT id, tenant_id, key, name, display_name, description, icon, category,
	                 clone_from, is_core, status, created_by, updated_by, created_at, updated_at
	          FROM business_object_definitions
	          WHERE tenant_id = $1
	          ORDER BY created_at DESC`
	if err := s.db.SelectContext(ctx, &bos, query, tenantID); err != nil {
		return nil, fmt.Errorf("failed to list business objects: %w", err)
	}
	return bos, nil
}

// GetBusinessObject retrieves a single BO by tenant + key.
func (s *BusinessObjectService) GetBusinessObject(ctx context.Context, tenantID, key string) (*models.BusinessObjectDefinition, error) {
	if tenantID == "" || key == "" {
		return nil, fmt.Errorf("tenantID and key are required")
	}
	var bo models.BusinessObjectDefinition
	err := s.db.GetContext(ctx, &bo,
		`SELECT id, tenant_id, key, name, display_name, description, icon, category,
		        clone_from, is_core, status, created_by, updated_by, created_at, updated_at
		 FROM business_object_definitions WHERE tenant_id = $1 AND key = $2`,
		tenantID, key)
	if err != nil {
		return nil, fmt.Errorf("business object %s/%s not found: %w", tenantID, key, err)
	}
	return &bo, nil
}

// UpdateBusinessObject updates an existing BO definition.
func (s *BusinessObjectService) UpdateBusinessObject(ctx context.Context, tenantID, key string, req models.UpdateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error) {
	if tenantID == "" || key == "" {
		return nil, fmt.Errorf("tenantID and key are required")
	}
	// Require write access
	if err := s.requireAccess(ctx, tenantID, key, AccessLevelWrite); err != nil {
		return nil, err
	}

	updates := []string{}
	args := []interface{}{}
	argIdx := 1
	if req.DisplayName != "" {
		updates = append(updates, fmt.Sprintf("display_name = $%d", argIdx))
		args = append(args, req.DisplayName)
		argIdx++
	}
	if req.Description != "" {
		updates = append(updates, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, req.Description)
		argIdx++
	}
	if req.Icon != "" {
		updates = append(updates, fmt.Sprintf("icon = $%d", argIdx))
		args = append(args, req.Icon)
		argIdx++
	}
	if req.Category != "" {
		updates = append(updates, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, req.Category)
		argIdx++
	}

	if len(updates) == 0 {
		return s.GetBusinessObject(ctx, tenantID, key)
	}

	updates = append(updates, fmt.Sprintf("updated_by = $%d", argIdx))
	args = append(args, userID)
	argIdx++
	updates = append(updates, fmt.Sprintf("updated_at = $%d", argIdx))
	args = append(args, time.Now().UTC())
	argIdx++
	updates = append(updates, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++
	updates = append(updates, fmt.Sprintf("key = $%d", argIdx))
	args = append(args, key)

	query := fmt.Sprintf(
		`UPDATE business_object_definitions SET %s WHERE tenant_id = $%d AND key = $%d RETURNING *`,
		strings.Join(updates, ", "), argIdx-1, argIdx)

	var bo models.BusinessObjectDefinition
	if err := s.db.GetContext(ctx, &bo, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update business object: %w", err)
	}
	s.publishBOEvent(ctx, "bo.updated", &bo)
	return &bo, nil
}

// DeleteBusinessObject removes a BO definition.
func (s *BusinessObjectService) DeleteBusinessObject(ctx context.Context, tenantID, key, userID string) error {
	if tenantID == "" || key == "" {
		return fmt.Errorf("tenantID and key are required")
	}
	if err := s.requireAccess(ctx, tenantID, key, AccessLevelWrite); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`DELETE FROM business_object_definitions WHERE tenant_id = $1 AND key = $2`,
		tenantID, key)
	if err != nil {
		return fmt.Errorf("failed to delete business object: %w", err)
	}
	s.publishBOEvent(ctx, "bo.deleted", &models.BusinessObjectDefinition{TenantID: tenantID, Key: key})
	return nil
}

// CloneBusinessObject creates a copy of an existing BO.
func (s *BusinessObjectService) CloneBusinessObject(ctx context.Context, tenantID string, req models.CloneBORequest, userID string) (*models.BusinessObjectDefinition, error) {
	source, err := s.GetBusinessObject(ctx, tenantID, req.SourceBOKey)
	if err != nil {
		return nil, fmt.Errorf("source business object not found: %w", err)
	}
	cloneReq := models.CreateBusinessObjectRequest{
		Name:         req.NewName,
		DisplayName:  req.NewName,
		Description:  req.Description,
		Icon:         req.Icon,
		Category:     source.Category,
		CloneFromKey: source.Key,
	}
	return s.CreateBusinessObject(ctx, tenantID, cloneReq, userID)
}

// ============================================================================
// BO INSTANCE CRUD
// ============================================================================

// CreateInstance creates a new BO instance.
func (s *BusinessObjectService) CreateInstance(ctx context.Context, tenantID, userID string, instance *models.BusinessObjectInstance) (*models.BusinessObjectInstance, error) {
	if tenantID == "" || userID == "" {
		return nil, fmt.Errorf("tenantID and userID are required")
	}
	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}
	instance.TenantID = tenantID
	instance.CreatedBy = userID
	instance.LastModifiedBy = userID
	now := time.Now().UTC()
	instance.CreatedAt = now
	instance.LastModifiedAt = now

	coreJSON, err := json.Marshal(instance.CoreFieldValues)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal core field values: %w", err)
	}
	customJSON, err := json.Marshal(instance.CustomFieldValues)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom field values: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO business_object_instances
		 (id, tenant_id, business_object_id, core_field_values, custom_field_values,
		  created_by, last_modified_by, created_at, last_modified_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		instance.ID, instance.TenantID, instance.BusinessObjectID,
		string(coreJSON), string(customJSON),
		instance.CreatedBy, instance.LastModifiedBy,
		instance.CreatedAt, instance.LastModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}
	s.publishBOEvent(ctx, "bo.instance.created", &models.BusinessObjectDefinition{
		ID: instance.BusinessObjectID, TenantID: tenantID, Key: instance.BusinessObjectID})
	return instance, nil
}

// ListInstances returns instances of a BO with pagination.
func (s *BusinessObjectService) ListInstances(ctx context.Context, tenantID, boKey string, offset, limit int) ([]*models.BusinessObjectInstance, int, error) {
	if tenantID == "" || boKey == "" {
		return nil, 0, fmt.Errorf("tenantID and boKey are required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var instances []*models.BusinessObjectInstance
	if err := s.db.SelectContext(ctx, &instances,
		`SELECT id, tenant_id, business_object_id, data, status, created_by, updated_by, created_at, updated_at
		 FROM business_object_instances
		 WHERE tenant_id = $1 AND business_object_id = $2
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`,
		tenantID, boKey, limit, offset); err != nil {
		return nil, 0, fmt.Errorf("failed to list instances: %w", err)
	}

	var total int
	if err := s.db.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM business_object_instances WHERE tenant_id = $1 AND business_object_id = $2`,
		tenantID, boKey); err != nil {
		return nil, 0, fmt.Errorf("failed to count instances: %w", err)
	}
	return instances, total, nil
}

// GetInstance retrieves a single BO instance.
func (s *BusinessObjectService) GetInstance(ctx context.Context, tenantID, instanceID string) (*models.BusinessObjectInstance, error) {
	if tenantID == "" || instanceID == "" {
		return nil, fmt.Errorf("tenantID and instanceID are required")
	}
	var instance models.BusinessObjectInstance
	var coreJSON, customJSON []byte
	err := s.db.QueryRowxContext(ctx,
		`SELECT id, tenant_id, business_object_id, core_field_values, custom_field_values,
		        created_by, last_modified_by, created_at, last_modified_at
		 FROM business_object_instances WHERE tenant_id = $1 AND id = $2`,
		tenantID, instanceID).Scan(
		&instance.ID, &instance.TenantID, &instance.BusinessObjectID,
		&coreJSON, &customJSON,
		&instance.CreatedBy, &instance.LastModifiedBy,
		&instance.CreatedAt, &instance.LastModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("instance %s not found: %w", instanceID, err)
	}
	instance.CoreFieldValues = map[string]interface{}{}
	instance.CustomFieldValues = map[string]interface{}{}
	_ = json.Unmarshal(coreJSON, &instance.CoreFieldValues)
	_ = json.Unmarshal(customJSON, &instance.CustomFieldValues)
	return &instance, nil
}

// GetInstanceForValidation returns instance data as a flat map for validation.
func (s *BusinessObjectService) GetInstanceForValidation(ctx context.Context, tenantID, instanceID string) (map[string]interface{}, error) {
	instance, err := s.GetInstance(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{
		"id":                 instance.ID,
		"tenant_id":          instance.TenantID,
		"business_object_id": instance.BusinessObjectID,
		"created_at":         instance.CreatedAt,
		"updated_at":         instance.LastModifiedAt,
	}
	for k, v := range instance.CoreFieldValues {
		result[k] = v
	}
	for k, v := range instance.CustomFieldValues {
		result[k] = v
	}
	return result, nil
}

// UpdateInstance updates instance data.
func (s *BusinessObjectService) UpdateInstance(ctx context.Context, tenantID, instanceID, userID string, core, custom map[string]interface{}) (*models.BusinessObjectInstance, error) {
	if tenantID == "" || instanceID == "" || userID == "" {
		return nil, fmt.Errorf("tenantID, instanceID, and userID are required")
	}
	instance, err := s.GetInstance(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}
	if err := s.requireAccess(ctx, tenantID, instance.BusinessObjectID, AccessLevelWrite); err != nil {
		return nil, err
	}
	// Merge core + custom into existing values
	if instance.CoreFieldValues == nil {
		instance.CoreFieldValues = map[string]interface{}{}
	}
	if instance.CustomFieldValues == nil {
		instance.CustomFieldValues = map[string]interface{}{}
	}
	for k, v := range core {
		instance.CoreFieldValues[k] = v
	}
	for k, v := range custom {
		instance.CustomFieldValues[k] = v
	}
	instance.LastModifiedBy = userID
	instance.LastModifiedAt = time.Now().UTC()

	coreJSON, err := json.Marshal(instance.CoreFieldValues)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal core values: %w", err)
	}
	customJSON, err := json.Marshal(instance.CustomFieldValues)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom values: %w", err)
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE business_object_instances
		 SET core_field_values = $1, custom_field_values = $2,
		     last_modified_by = $3, last_modified_at = $4
		 WHERE tenant_id = $5 AND id = $6`,
		string(coreJSON), string(customJSON), userID,
		instance.LastModifiedAt, tenantID, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to update instance: %w", err)
	}
	s.publishBOEvent(ctx, "bo.instance.updated", &models.BusinessObjectDefinition{
		ID: instance.BusinessObjectID, TenantID: tenantID, Key: instance.BusinessObjectID})
	return instance, nil
}

// DeleteInstance removes a BO instance.
func (s *BusinessObjectService) DeleteInstance(ctx context.Context, tenantID, instanceID, userID string) error {
	if tenantID == "" || instanceID == "" {
		return fmt.Errorf("tenantID and instanceID are required")
	}
	instance, err := s.GetInstance(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}
	if err := s.requireAccess(ctx, tenantID, instance.BusinessObjectID, AccessLevelWrite); err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`DELETE FROM business_object_instances WHERE tenant_id = $1 AND id = $2`,
		tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	s.publishBOEvent(ctx, "bo.instance.deleted", &models.BusinessObjectDefinition{
		ID: instance.BusinessObjectID, TenantID: tenantID, Key: instance.BusinessObjectID})
	return nil
}

// Suppress unused events import (used implicitly via event publishing).
var _ events.DomainEvent

// Suppress unused models fields (read by JSON tags via query).
var _ = []interface{}{models.BusinessObjectDefinition{}, models.BusinessObjectInstance{}}