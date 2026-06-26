package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"errors"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/platform"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// AccessLevel represents the effective permission over a Business Object.
type AccessLevel string

const (
	AccessLevelNone  AccessLevel = "NONE"
	AccessLevelRead  AccessLevel = "READ"
	AccessLevelWrite AccessLevel = "WRITE"
)

// ErrForbidden is returned when a caller lacks the required permission.
var ErrForbidden = errors.New("forbidden")

// AccessDecision is the composed decision for a principal over a BO.
type AccessDecision struct {
	AccessLevel  AccessLevel
	RowPredicate string
	ColumnMasks  map[string]string
}

// RelationshipResult represents a related entity found via catalog edges
type RelationshipResult struct {
	RelatedObjectName string `json:"relatedObjectName" db:"related_object_name"`
	RelationshipType  string `json:"relationshipType" db:"relationship_type"`
	Description       string `json:"description" db:"description"`
}

// SemanticFieldResult represents a field mapped to a semantic term
type SemanticFieldResult struct {
	FieldName        string `json:"fieldName" db:"field_name"`
	SemanticTermName string `json:"semanticTermName" db:"semantic_term_name"`
	EdgeTypeName     string `json:"edge_type_name" db:"edge_type_name"`
}

// BORelationshipsResponse aggregates relationships and semantic mappings
type BORelationshipsResponse struct {
	RelatedObjects []RelationshipResult  `json:"relatedObjects"`
	SemanticFields []SemanticFieldResult `json:"semanticFields"`
	AvailableTerms []models.SemanticTerm `json:"availableTerms"`
}

// BusinessObjectService handles all BO operations
type BusinessObjectService struct {
	db             *sqlx.DB
	tenantManager  *platform.TenantDBManager
	auditPublisher *events.AuditEventPublisher
	lineageRepo    lineage.LineageRepository
}

var boFieldsColumnCache sync.Map

func (s *BusinessObjectService) boFieldsHasColumn(ctx context.Context, schema, column string) bool {
	cacheKey := fmt.Sprintf("%s.bo_fields.%s", schema, column)
	if v, ok := boFieldsColumnCache.Load(cacheKey); ok {
		return v.(bool)
	}

	var exists bool
	query := "SELECT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = $1 AND column_name = $2"
	args := []interface{}{"bo_fields", column}
	if schema != "" {
		query += " AND table_schema = $3"
		args = append(args, schema)
	}
	query += ")"

	if err := s.db.GetContext(ctx, &exists, query, args...); err != nil {
		exists = false
	}

	boFieldsColumnCache.Store(cacheKey, exists)
	return exists
}

func (s *BusinessObjectService) boFieldsDisplayNameExpr(ctx context.Context, schema string) string {
	if s.boFieldsHasColumn(ctx, schema, "display_name") {
		return "display_name"
	}
	if s.boFieldsHasColumn(ctx, schema, "display_label") {
		return "display_label"
	}
	return "name"
}

// NewBusinessObjectService creates a new BO service
func NewBusinessObjectService(db *sqlx.DB, tm *platform.TenantDBManager, ap *events.AuditEventPublisher, lr lineage.LineageRepository) *BusinessObjectService {
	return &BusinessObjectService{
		db:             db,
		tenantManager:  tm,
		auditPublisher: ap,
		lineageRepo:    lr,
	}
}

// atLeast returns true if current level meets the required level.
func (l AccessLevel) atLeast(required AccessLevel) bool {
	rank := map[AccessLevel]int{
		AccessLevelNone:  0,
		AccessLevelRead:  1,
		AccessLevelWrite: 2,
	}
	return rank[l] >= rank[required]
}

// resolveAccessDecision determines the access level for a user on a given business object.
func (s *BusinessObjectService) resolveAccessDecision(ctx context.Context, tenantID, boID string) (*AccessDecision, error) {
	// 1. Resolve Principal from context (Handler vs Service layer gap)
	// We use the security package's AuthInfo which is populated by middleware
	authInfo, ok := security.AuthInfoFromContext(ctx)
	if !ok {
		// If no auth info, assume no access? Or maybe strict mode?
		// For now, if no auth context, we might be in a system process or unauth allowed endpoint (unlikely for BOs)
		// But effectively, "no user" = "no groups" = "no access" unless fail-open
		return &AccessDecision{AccessLevel: AccessLevelNone, ColumnMasks: map[string]string{}}, nil
	}

	// 2. Global Admin / Global Ops Bypass (Root Access)
	for _, role := range authInfo.Roles {
		if role == "global_admin" || role == "global_ops" {
			// Full access, no row filters, no column masks
			return &AccessDecision{
				AccessLevel: AccessLevelWrite,
				ColumnMasks: map[string]string{},
			}, nil
		}
	}

	// 3. Fallback: Fail Open (Legacy Behavior) until AccessRuleRepository is wired
	// This ensures existing functionalities continue to work while we standardize the check structure.
	return &AccessDecision{AccessLevel: AccessLevelWrite, ColumnMasks: map[string]string{}}, nil
}

// requireAccess enforces the required level and returns the decision for downstream use.
func (s *BusinessObjectService) requireAccess(ctx context.Context, tenantID, boID string, required AccessLevel) (*AccessDecision, error) {
	decision, err := s.resolveAccessDecision(ctx, tenantID, boID)
	if err != nil {
		return nil, err
	}

	if !decision.AccessLevel.atLeast(required) {
		return nil, ErrForbidden
	}

	if decision.ColumnMasks == nil {
		decision.ColumnMasks = make(map[string]string)
	}

	return decision, nil
}

// small helpers
func toString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	if sn, ok := v.(json.Number); ok {
		return sn.String()
	}
	return fmt.Sprintf("%v", v)
}
func toInt(v interface{}) int {
	switch val := v.(type) {
	case nil:
		return 0
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case json.Number:
		i64, err := val.Int64()
		if err == nil {
			return int(i64)
		}
		if f, err := val.Float64(); err == nil {
			return int(f)
		}
	case string:
		var i int
		_, _ = fmt.Sscanf(val, "%d", &i)
		return i
	}
	return 0
}

// ============================================================================
// BUSINESS OBJECT OPERATIONS (Central DB)
// ============================================================================

// CreateBusinessObject creates a new BO
func (s *BusinessObjectService) CreateBusinessObject(
	ctx context.Context,
	secCtx *security.Context,
	req models.CreateBusinessObjectRequest,
	userID string,
) (*models.BusinessObjectDefinition, error) {
	tenantID := secCtx.TenantID
	// Generate key from name
	key := slugify(req.Name)
	technicalName := req.TechnicalName
	if technicalName == "" {
		technicalName = key
	}

	id := uuid.New().String()
	now := time.Now()

	logging.GetLogger().Sugar().Errorf("[META-SERVICE] CreateBusinessObject START: req.DatasourceID=%q req.ParentID=%q req.Name=%q", req.DatasourceID, req.ParentID, req.Name)

	bo := &models.BusinessObjectDefinition{
		ID:              id,
		TenantID:        tenantID,
		Key:             key,
		Name:            req.Name,
		DisplayName:     req.DisplayName,
		TechnicalName:   technicalName,
		Description:     req.Description,
		Icon:            req.Icon,
		IsCore:          false,
		Category:        req.Category,
		ParentID:        sql.NullString{String: req.ParentID, Valid: req.ParentID != ""},
		DatasourceID:    sql.NullString{String: req.DatasourceID, Valid: req.DatasourceID != ""},
		DriverTableID:   sql.NullString{String: req.DriverTableID, Valid: req.DriverTableID != ""},
		DriverTableName: req.DriverTableName,
		CoreFields:      []models.FieldDefinition{},
		CustomFields:    []models.FieldDefinition{},
		Subtypes:        make(map[string]models.SubtypeDefinition),
		CreatedAt:       now,
		CreatedBy:       userID,
		LastModifiedAt:  now,
		LastModifiedBy:  userID,
		IsActive:        true,
	}

	logging.GetLogger().Sugar().Errorf("[META-SERVICE] After init: bo.DatasourceID.Valid=%v bo.DatasourceID.String=%q bo.ParentID.Valid=%v", bo.DatasourceID.Valid, bo.DatasourceID.String, bo.ParentID.Valid)

	// Ensure subtypes always carry a datasource. Prefer request, otherwise inherit parent.
	if bo.ParentID.Valid {
		logging.GetLogger().Sugar().Errorf("[META-SERVICE] ParentID valid, checking datasource inheritance")
		if !bo.DatasourceID.Valid {
			logging.GetLogger().Sugar().Errorf("[META-SERVICE] DatasourceID not valid, will inherit from parent")
			if req.DatasourceID != "" {
				bo.DatasourceID = sql.NullString{String: req.DatasourceID, Valid: true}
			} else {
				parent, err := s.GetBusinessObject(ctx, secCtx, req.ParentID)
				if err != nil {
					return nil, fmt.Errorf("failed to resolve parent datasource: %w", err)
				}
				if parent.DatasourceID.Valid && parent.DatasourceID.String != "" {
					bo.DatasourceID = sql.NullString{String: parent.DatasourceID.String, Valid: true}
				} else {
					return nil, fmt.Errorf("missing datasource for subtype create (parent has none)")
				}
			}
		} else {
			logging.GetLogger().Sugar().Errorf("[META-SERVICE] DatasourceID already valid: %q", bo.DatasourceID.String)
		}
	}

	logging.GetLogger().Sugar().Errorf("[META-SERVICE] Final bo.DatasourceID: Valid=%v String=%q ABOUT TO INSERT", bo.DatasourceID.Valid, bo.DatasourceID.String)

	// If cloning, copy fields and subtypes from source
	if req.CloneFromKey != "" {
		if err := s.cloneBO(ctx, tenantID, bo, req.CloneFromKey, userID); err != nil {
			return nil, err
		}
	}

	// Insert BO
	query := `
		INSERT INTO business_objects (
			id, tenant_id, key, name, display_name, technical_name,
			description, icon, is_core, clones_from, clone_parent_key,
			clone_parent_display_name, category, parent_id, datasource_id,
			driver_table_id, driver_table_name,
			config,
			created_at, created_by, last_modified_at, last_modified_by, 
			is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, $10, $11,
			$12, $13, $14, $15,
			$16, $17,
			$18,
			$19, $20, $21, $22, 
			$23
		)
	`

	// Handle nullable parent_id UUID
	var parentID interface{} = nil
	if bo.ParentID.Valid && bo.ParentID.String != "" {
		parentID = bo.ParentID.String
	}

	var datasourceID interface{} = nil
	if bo.DatasourceID.Valid && bo.DatasourceID.String != "" {
		datasourceID = bo.DatasourceID.String
	}

	// Handle nullable driver_table_id UUID
	var driverTableID interface{} = nil
	if bo.DriverTableID.Valid {
		driverTableID = bo.DriverTableID.String
	}

	// Handle nullable created_by
	var createdBy interface{} = nil
	if bo.CreatedBy != "" {
		createdBy = bo.CreatedBy
	}

	// Handle nullable last_modified_by
	var lastModifiedBy interface{} = nil
	if bo.LastModifiedBy != "" {
		lastModifiedBy = bo.LastModifiedBy
	}

	// Prepare config JSON
	configJSON := map[string]interface{}{
		"is_core": bo.IsCore,
	}
	if req.Config != nil {
		for k, v := range req.Config {
			configJSON[k] = v
		}
	}
	configBytes, _ := json.Marshal(configJSON)

	logging.GetLogger().Sugar().Warnf("[META BO SERVICE] Create scope: tenant=%s datasource=%v parent_id=%v name=%s", bo.TenantID, datasourceID, parentID, bo.Name)
	fmt.Printf("[META DEBUG] tenant=%s datasource=%v parent_id=%v name=%s valid=%v\n", bo.TenantID, datasourceID, parentID, bo.Name, bo.DatasourceID.Valid)

	_, err := s.db.ExecContext(ctx, query,
		bo.ID, bo.TenantID, bo.Key, bo.Name, bo.DisplayName, bo.TechnicalName,
		bo.Description, bo.Icon, bo.IsCore, bo.ClonesFrom, bo.CloneParentKey,
		bo.CloneParentDisplayName, bo.Category, parentID, datasourceID,
		driverTableID, bo.DriverTableName,
		string(configBytes),
		bo.CreatedAt, createdBy, bo.LastModifiedAt, lastModifiedBy,
		bo.IsActive,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create business object: %w", err)
	}

	// Log audit
	s.logAudit(ctx, tenantID, "business_object", id, "create", nil, userID)

	return bo, nil
}

// GetBusinessObject retrieves a BO by key or ID from either old or new schema
func (s *BusinessObjectService) GetBusinessObject(
	ctx context.Context,
	secCtx *security.Context,
	boKey string,
) (*models.BusinessObjectDefinition, error) {
	tenantID := secCtx.TenantID
	// Check if boKey is a UUID
	isUUID := false
	if _, err := uuid.Parse(boKey); err == nil {
		isUUID = true
	}

	logging.GetLogger().Sugar().Infof("DEBUG: GetBusinessObject - tenantID: %s, boKey: %s, isUUID: %v", tenantID, boKey, isUUID)

	// Enforce Read Access
	if _, err := s.requireAccess(ctx, tenantID, boKey, AccessLevelRead); err != nil {
		return nil, err
	}

	bo := &models.BusinessObjectDefinition{}

	// Try old schema first (business_objects table)
	oldQuery := `
		SELECT id, tenant_id, key, name, display_name, COALESCE(technical_name, '') AS technical_name,
		       COALESCE(description, '') AS description, COALESCE(icon, '') AS icon, is_core, 
		       COALESCE(clones_from, '') AS clones_from, COALESCE(clone_parent_key, '') AS clone_parent_key,
		       COALESCE(clone_parent_display_name, '') AS clone_parent_display_name, COALESCE(category, '') AS category, 
		       parent_id,
		       driver_table_id, COALESCE(driver_table_name, '') AS driver_table_name,
		       CAST(0 AS int) AS instance_count, created_at, COALESCE(CAST(created_by AS text), '') AS created_by, 
		       last_modified_at, COALESCE(CAST(last_modified_by AS text), '') AS last_modified_by, 
		       is_active,
		       config, datasource_id
		FROM business_objects
		WHERE tenant_id = $1::uuid AND (key = $2 OR ($3 = true AND id = CAST($2 AS uuid)))
	`

	err := s.db.GetContext(ctx, bo, oldQuery, tenantID, boKey, isUUID)
	if err == nil {
		// Found in old schema (User Tenant)
		logging.GetLogger().Sugar().Infof("DEBUG: GetBusinessObject found in old schema (User Tenant) - id=%s", bo.ID)
		s.populateDriverTableInfo(ctx, bo)

		// Load subtypes and fields
		if err := s.loadBOSubtypesAndFields(ctx, bo, tenantID); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to load subtypes and fields: %v", err)
		}
		return bo, nil
	}

	// Fallback: Check Gold Copy Tenant if not found in User Tenant
	var goldCopyTenantID string
	gcErr := s.db.QueryRowContext(ctx, `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyTenantID)

	if gcErr == nil && goldCopyTenantID != "" && goldCopyTenantID != tenantID {
		err = s.db.GetContext(ctx, bo, oldQuery, goldCopyTenantID, boKey, isUUID)
		if err == nil {
			// Found in Gold Copy
			logging.GetLogger().Sugar().Infof("DEBUG: GetBusinessObject found in Gold Copy - id=%s", bo.ID)
			s.populateDriverTableInfo(ctx, bo)

			// Load subtypes and fields - PASSING USER TENANT ID to find custom extensions
			if err := s.loadBOSubtypesAndFields(ctx, bo, tenantID); err != nil {
				logging.GetLogger().Sugar().Warnf("Warning: failed to load subtypes and fields for Gold Copy BO: %v", err)
			}
			return bo, nil
		}
	}

	// If not found in old schema, return error immediately
	logging.GetLogger().Sugar().Errorf("ERROR: GetBusinessObject not found in old schema (User or Gold Copy): %v", err)
	return nil, fmt.Errorf("business object not found")
}

func (s *BusinessObjectService) populateDriverTableInfo(ctx context.Context, bo *models.BusinessObjectDefinition) {
	// Populate driver table info from config if columns are empty (wizard writes it into config)
	if len(bo.Config) > 0 {
		var cfg map[string]interface{}
		if err := json.Unmarshal(bo.Config, &cfg); err == nil {
			if !bo.DriverTableID.Valid || bo.DriverTableID.String == "" {
				if v, ok := cfg["driver_table_id"].(string); ok && v != "" {
					bo.DriverTableID = sql.NullString{String: v, Valid: true}
				}
			}
			if bo.DriverTableName == "" {
				if v, ok := cfg["driver_table_name"].(string); ok && v != "" {
					bo.DriverTableName = v
				}
			}
		}
	}

	// Final fallback: If we have an ID but no name, look it up in catalog_node
	if bo.DriverTableName == "" && bo.DriverTableID.Valid && bo.DriverTableID.String != "" {
		var catName string
		lookupErr := s.db.GetContext(ctx, &catName, "SELECT node_name FROM catalog_node WHERE id = $1::uuid", bo.DriverTableID.String)
		if lookupErr == nil {
			bo.DriverTableName = catName
		}
	}
}

// ListBusinessObjects retrieves all BOs for a tenant from both old and new schema
func (s *BusinessObjectService) ListBusinessObjects(
	ctx context.Context,
	secCtx *security.Context,
) ([]*models.BusinessObjectDefinition, error) {
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID
	oldQuery := `
		SELECT id, tenant_id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, 
		       COALESCE(description, '') AS description, COALESCE(icon, '') AS icon, is_core, 
		       COALESCE(clones_from, '') AS clones_from, COALESCE(clone_parent_key, '') AS clone_parent_key,
		       COALESCE(clone_parent_display_name, '') AS clone_parent_display_name, COALESCE(category, '') AS category, 
		       parent_id,
		       driver_table_id, COALESCE(driver_table_name, '') AS driver_table_name,
		       CAST(0 AS int) AS instance_count, created_at, COALESCE(CAST(created_by AS text), '') AS created_by, 
		       last_modified_at, COALESCE(CAST(last_modified_by AS text), '') AS last_modified_by, 
		       is_active,
		       config, datasource_id
		FROM business_objects
		WHERE tenant_id = $1::uuid AND parent_id IS NULL
	`

	oldArgs := []interface{}{tenantID}
	if datasourceID != "" {
		oldQuery += " AND (datasource_id = $2::uuid OR datasource_id IS NULL)"
		oldArgs = append(oldArgs, datasourceID)
	}
	oldQuery += " ORDER BY name"

	var bos []*models.BusinessObjectDefinition
	oldSchemaErr := s.db.SelectContext(ctx, &bos, oldQuery, oldArgs...)
	if oldSchemaErr != nil {
		logging.GetLogger().Sugar().Warnf("Warning: failed to list from old schema: %v", oldSchemaErr)
		// Continue to new schema
	}

	// Only use old schema for now as business_object_def does not exist
	return bos, nil
}

// ListBusinessObjectsComposed returns Workday-style composed Core + Custom BOs for a tenant.
// Core BOs are loaded from the gold copy tenant, then merged with tenant-specific extensions.
// This method provides a unified view where tenants see their customizations merged onto the core.
func (s *BusinessObjectService) ListBusinessObjectsComposed(
	ctx context.Context,
	secCtx *security.Context,
) ([]*models.BusinessObjectDefinition, error) {
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID
	// 1. Get gold copy tenant ID
	var goldCopyTenantID string
	err := s.db.QueryRowContext(ctx, `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`).Scan(&goldCopyTenantID)
	if err != nil {
		// If no gold copy tenant, fall back to regular listing
		logging.GetLogger().Sugar().Warnf("No gold copy tenant found, falling back to regular listing: %v", err)
		return s.ListBusinessObjects(ctx, secCtx)
	}

	// 2. If requesting tenant IS the gold copy, return only core BOs
	if tenantID == goldCopyTenantID {
		return s.listCoreBusinessObjects(ctx, goldCopyTenantID)
	}

	// 3. Load core BOs from gold copy tenant
	coreBOs, err := s.listCoreBusinessObjects(ctx, goldCopyTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load core business objects: %w", err)
	}

	// 4. Load custom BOs for requesting tenant
	customBOs, err := s.listTenantCustomBusinessObjects(ctx, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tenant custom business objects: %w", err)
	}

	// 5. Compose: merge custom onto core
	return s.composeBusinessObjects(coreBOs, customBOs), nil
}

// listCoreBusinessObjects lists BOs from the gold copy tenant marked as core
func (s *BusinessObjectService) listCoreBusinessObjects(ctx context.Context, goldCopyTenantID string) ([]*models.BusinessObjectDefinition, error) {
	query := `
		SELECT id, tenant_id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, 
		       COALESCE(description, '') AS description, COALESCE(icon, '') AS icon, is_core, 
		       COALESCE(clones_from, '') AS clones_from, COALESCE(clone_parent_key, '') AS clone_parent_key,
		       COALESCE(clone_parent_display_name, '') AS clone_parent_display_name, COALESCE(category, '') AS category, 
		       parent_id,
		       driver_table_id, COALESCE(driver_table_name, '') AS driver_table_name,
		       CAST(0 AS int) AS instance_count, created_at, COALESCE(CAST(created_by AS text), '') AS created_by, 
		       last_modified_at, COALESCE(CAST(last_modified_by AS text), '') AS last_modified_by, 
		       is_active,
		       config, datasource_id, core_id
		FROM business_objects
		WHERE tenant_id = $1::uuid AND is_core = true AND parent_id IS NULL
		ORDER BY name
	`
	var bos []*models.BusinessObjectDefinition
	err := s.db.SelectContext(ctx, &bos, query, goldCopyTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list core business objects: %w", err)
	}
	return bos, nil
}

// listTenantCustomBusinessObjects lists only tenant-specific (non-core) BOs
func (s *BusinessObjectService) listTenantCustomBusinessObjects(ctx context.Context, tenantID, datasourceID string) ([]*models.BusinessObjectDefinition, error) {
	query := `
		SELECT id, tenant_id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, 
		       COALESCE(description, '') AS description, COALESCE(icon, '') AS icon, is_core, 
		       COALESCE(clones_from, '') AS clones_from, COALESCE(clone_parent_key, '') AS clone_parent_key,
		       COALESCE(clone_parent_display_name, '') AS clone_parent_display_name, COALESCE(category, '') AS category, 
		       parent_id,
		       driver_table_id, COALESCE(driver_table_name, '') AS driver_table_name,
		       CAST(0 AS int) AS instance_count, created_at, COALESCE(CAST(created_by AS text), '') AS created_by, 
		       last_modified_at, COALESCE(CAST(last_modified_by AS text), '') AS last_modified_by, 
		       is_active,
		       config, datasource_id, core_id
		FROM business_objects
		WHERE tenant_id = $1::uuid AND (is_core = false OR is_core IS NULL) AND parent_id IS NULL
	`
	args := []interface{}{tenantID}

	if datasourceID != "" {
		query += " AND (datasource_id = $2::uuid OR datasource_id IS NULL)"
		args = append(args, datasourceID)
	}

	query += " ORDER BY name"

	var bos []*models.BusinessObjectDefinition
	err := s.db.SelectContext(ctx, &bos, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant custom business objects: %w", err)
	}
	return bos, nil
}

// composeBusinessObjects merges custom tenant BOs with core BOs (Workday-style)
func (s *BusinessObjectService) composeBusinessObjects(coreBOs, customBOs []*models.BusinessObjectDefinition) []*models.BusinessObjectDefinition {
	result := make([]*models.BusinessObjectDefinition, 0, len(coreBOs)+len(customBOs))
	coreMap := make(map[string]*models.BusinessObjectDefinition)

	// Index core BOs by ID
	for _, bo := range coreBOs {
		bo.IsCore = true // Ensure marked as core
		coreMap[bo.ID] = bo
		result = append(result, bo)
	}

	// Process custom BOs
	for _, customBO := range customBOs {
		if customBO.CoreID.Valid && customBO.CoreID.String != "" {
			// This custom BO extends a core BO
			if coreBO, ok := coreMap[customBO.CoreID.String]; ok {
				// Merge custom fields onto core
				composed := s.mergeCustomOntoCore(coreBO, customBO)
				// Replace core entry with composed version
				for i, r := range result {
					if r.ID == coreBO.ID {
						result[i] = composed
						break
					}
				}
			} else {
				// Core BO not found, add custom as standalone
				customBO.IsCore = false
				result = append(result, customBO)
			}
		} else {
			// Pure tenant-only custom BO (no core_id)
			customBO.IsCore = false
			result = append(result, customBO)
		}
	}

	return result
}

// mergeCustomOntoCore creates a composed BO by merging custom fields onto core
func (s *BusinessObjectService) mergeCustomOntoCore(coreBO, customBO *models.BusinessObjectDefinition) *models.BusinessObjectDefinition {
	// Create a new composed BO based on core
	composed := &models.BusinessObjectDefinition{
		ID:          coreBO.ID, // Keep core ID for identity
		Key:         coreBO.Key,
		Name:        coreBO.Name,
		DisplayName: coreBO.DisplayName,
		Description: coreBO.Description,
		Icon:        coreBO.Icon,
		Category:    coreBO.Category,
		IsCore:      true, // Mark as core-based
		IsActive:    coreBO.IsActive,
		CreatedAt:   coreBO.CreatedAt,
		TenantID:    customBO.TenantID, // Use tenant's ID for context
		CoreID:      sql.NullString{String: coreBO.ID, Valid: true},
	}

	// Override with custom values where provided
	if customBO.DisplayName != "" {
		composed.DisplayName = customBO.DisplayName
	}
	if customBO.Description != "" {
		composed.Description = customBO.Description
	}
	if customBO.Icon != "" {
		composed.Icon = customBO.Icon
	}

	// Merge fields: core fields + custom fields
	composed.CoreFields = coreBO.CoreFields
	composed.CustomFields = append(coreBO.CustomFields, customBO.CustomFields...)

	// If custom BO has a config, use it (allows tenant overrides)
	if len(customBO.Config) > 0 {
		composed.Config = customBO.Config
	} else {
		composed.Config = coreBO.Config
	}

	return composed
}
func (s *BusinessObjectService) ListBusinessObjectsLegacy(
	ctx context.Context,
	secCtx *security.Context,
) ([]models.BusinessObjectListItem, error) {
	tenantID := secCtx.TenantID
	datasourceID := secCtx.DatasourceID

	// Use a transaction if we need to set local config (standard for legacy RLS compatibility)
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT set_config('hasura.tenant_id', $1, true)", tenantID); err != nil {
		return nil, fmt.Errorf("failed to set tenant context: %w", err)
	}

	query := `
		SELECT bo.id, bo.name, bo.display_name, COALESCE(bo.description, '') as description, COALESCE(bo.icon, '') as icon, 
		       COALESCE(bo.config, '{}'::jsonb) as config_json, bo.tenant_id, 
		       (SELECT gold_copy FROM public.tenants t WHERE t.id = bo.tenant_id) as owner_is_gold_copy
		FROM public.business_objects bo
		WHERE (bo.tenant_id = $1::uuid OR 
		       EXISTS(SELECT 1 FROM public.tenants t WHERE t.id = bo.tenant_id AND t.gold_copy = TRUE AND bo.tenant_id != $1::uuid))
		  AND bo.parent_id IS NULL
	`
	args := []interface{}{tenantID}

	if datasourceID != "" {
		query += ` AND (
			(bo.driver_table_id IS NOT NULL AND EXISTS(
				SELECT 1 FROM catalog_node cn WHERE cn.id = bo.driver_table_id::uuid AND cn.tenant_datasource_id = $2::uuid
			))
			OR
			(bo.driver_table_name IS NOT NULL AND EXISTS(
				SELECT 1 FROM catalog_node cn2 WHERE cn2.qualified_path = bo.driver_table_name AND cn2.tenant_datasource_id = $2::uuid
			))
		)`
		args = append(args, datasourceID)
	}

	query += " ORDER BY bo.name"

	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch business objects: %w", err)
	}
	defer rows.Close()

	var items []models.BusinessObjectListItem
	for rows.Next() {
		var id, name, displayName, description, icon string
		var configJSON []byte
		var ownerTenantID sql.NullString
		var ownerIsGoldCopy sql.NullBool

		if err := rows.Scan(&id, &name, &displayName, &description, &icon, &configJSON, &ownerTenantID, &ownerIsGoldCopy); err != nil {
			return nil, fmt.Errorf("failed to scan business object: %w", err)
		}

		config := make(map[string]interface{})
		_ = json.Unmarshal(configJSON, &config)

		// Metadata logic
		isOwnedByTenant := ownerTenantID.Valid && ownerTenantID.String == tenantID
		isInheritedFromCore := ownerIsGoldCopy.Valid && ownerIsGoldCopy.Bool && ownerTenantID.Valid && ownerTenantID.String != tenantID

		if isInheritedFromCore {
			config["is_read_only"] = true
			config["is_inherited_from_core"] = true
			config["inherited_from_tenant_id"] = ownerTenantID.String
		} else if isOwnedByTenant {
			config["is_read_only"] = false
			config["is_inherited_from_core"] = false
		}

		fieldsOut := make([]models.BusinessObjectListField, 0)
		if raw, ok := config["fields"]; ok {
			if arr, ok := raw.([]interface{}); ok {
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						fieldName, _ := m["key"].(string)
						if fieldName == "" {
							fieldName, _ = m["name"].(string)
						}
						fieldType, _ := m["type"].(string)
						label, _ := m["label"].(string)
						if label == "" {
							label, _ = m["displayName"].(string)
						}
						if label == "" {
							label, _ = m["name"].(string)
						}
						if label == "" {
							label = fieldName
						}
						if fieldName != "" {
							fieldsOut = append(fieldsOut, models.BusinessObjectListField{Name: fieldName, Type: fieldType, Label: label})
						}
					}
				}
			}
		}

		// Fallback to bo_fields table
		if len(fieldsOut) == 0 {
			fRows, err := tx.QueryContext(ctx, `
				SELECT field_name, field_type, COALESCE(display_label, field_name), COALESCE(column_name, field_name)
				FROM public.bo_fields
				WHERE tenant_id = $1 AND business_object_id = $2
				ORDER BY display_order
			`, tenantID, id)
			if err == nil {
				for fRows.Next() {
					var fn, ft, dl, cn string
					if err := fRows.Scan(&fn, &ft, &dl, &cn); err == nil {
						fieldsOut = append(fieldsOut, models.BusinessObjectListField{
							Name:       fn,
							Type:       ft,
							Label:      dl,
							ColumnName: cn,
						})
					}
				}
				fRows.Close()
			}
		}

		items = append(items, models.BusinessObjectListItem{
			ID:          id,
			Name:        name,
			DisplayName: displayName,
			Description: description,
			Fields:      fieldsOut,
			Icon:        icon,
			Config:      config,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return items, nil
}

func (s *BusinessObjectService) GetBusinessObjectLegacy(
	ctx context.Context,
	secCtx *security.Context,
	boID string,
) (*models.BusinessObjectListItem, error) {
	tenantID := secCtx.TenantID

	query := `
		SELECT bo.id, bo.name, bo.display_name, COALESCE(bo.description, '') as description, 
		       COALESCE(bo.icon, '') as icon, COALESCE(bo.config, '{}'::jsonb) as config_json,
		       bo.tenant_id, 
		       (SELECT gold_copy FROM public.tenants t WHERE t.id = bo.tenant_id) as owner_is_gold_copy
		FROM public.business_objects bo
		WHERE bo.id = $1::uuid
		  AND (bo.tenant_id = $2::uuid OR 
		       EXISTS(SELECT 1 FROM public.tenants t WHERE t.id = bo.tenant_id AND t.gold_copy = TRUE AND bo.tenant_id != $2::uuid))
	`

	var id, name, displayName, description, icon string
	var configJSON []byte
	var ownerTenantID sql.NullString
	var ownerIsGoldCopy sql.NullBool

	err := s.db.QueryRowContext(ctx, query, boID, tenantID).
		Scan(&id, &name, &displayName, &description, &icon, &configJSON,
			&ownerTenantID, &ownerIsGoldCopy)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("business object not found")
		}
		return nil, fmt.Errorf("failed to fetch business object: %w", err)
	}

	isOwnedByTenant := ownerTenantID.Valid && ownerTenantID.String == tenantID
	isInheritedFromCore := ownerIsGoldCopy.Valid && ownerIsGoldCopy.Bool && ownerTenantID.Valid && ownerTenantID.String != tenantID

	config := make(map[string]interface{})
	_ = json.Unmarshal(configJSON, &config)

	if isInheritedFromCore {
		config["is_read_only"] = true
		config["is_inherited_from_core"] = true
		config["inherited_from_tenant_id"] = ownerTenantID.String
	} else if isOwnedByTenant {
		config["is_read_only"] = false
		config["is_inherited_from_core"] = false
	}

	fieldsOut := make([]models.BusinessObjectListField, 0)
	if raw, ok := config["fields"]; ok {
		if arr, ok := raw.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					fieldName, _ := m["key"].(string)
					if fieldName == "" {
						fieldName, _ = m["name"].(string)
					}
					fieldType, _ := m["type"].(string)
					label, _ := m["label"].(string)
					if label == "" {
						label, _ = m["displayName"].(string)
					}
					if label == "" {
						label, _ = m["name"].(string)
					}
					if label == "" {
						label = fieldName
					}
					if fieldName != "" {
						fieldsOut = append(fieldsOut, models.BusinessObjectListField{Name: fieldName, Type: fieldType, Label: label})
					}
				}
			}
		}
	}

	if len(fieldsOut) == 0 {
		fRows, err := s.db.QueryContext(ctx, `
			SELECT field_name, field_type, COALESCE(display_label, field_name), COALESCE(column_name, field_name)
			FROM public.bo_fields
			WHERE tenant_id = $1 AND bo_id = $2
			ORDER BY display_order
		`, tenantID, id)
		if err == nil {
			defer fRows.Close()
			for fRows.Next() {
				var fn, ft, dl, cn string
				if err := fRows.Scan(&fn, &ft, &dl, &cn); err == nil {
					fieldsOut = append(fieldsOut, models.BusinessObjectListField{
						Name:       fn,
						Type:       ft,
						Label:      dl,
						ColumnName: cn,
					})
				}
			}
		}
	}

	return &models.BusinessObjectListItem{
		ID:          id,
		Name:        name,
		DisplayName: displayName,
		Description: description,
		Fields:      fieldsOut,
		Icon:        icon,
		Config:      config,
	}, nil
}

func (s *BusinessObjectService) UpdateBusinessObject(
	ctx context.Context,
	secCtx *security.Context,
	boKey string,
	req models.UpdateBusinessObjectRequest,
	userID string,
) (*models.BusinessObjectDefinition, error) {
	tenantID := secCtx.TenantID
	// Refactored to use dynamic update or just always update if we fetch first.
	// Let's fetch current to be safe and simple.
	current, err := s.GetBusinessObject(ctx, secCtx, boKey)
	if err != nil {
		return nil, err
	}

	// Enforce Write Access
	if _, err := s.requireAccess(ctx, tenantID, current.ID, AccessLevelWrite); err != nil {
		return nil, err
	}

	// Log entry for debugging whether update includes fields
	logging.GetLogger().Sugar().Infof("metadata.UpdateBusinessObject called: tenant=%s boKey=%s hasConfig=%v", tenantID, boKey, req.Config != nil)

	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	if req.DisplayName != "" {
		current.DisplayName = req.DisplayName
	}
	if req.Description != "" {
		current.Description = req.Description
	}
	if req.Icon != "" {
		current.Icon = req.Icon
	}
	if req.Category != "" {
		current.Category = req.Category
	}

	now := time.Now()
	var lastModifiedBy interface{} = nil
	if userID != "" {
		lastModifiedBy = userID
	}

	if req.Config != nil {
		// Update Config
		configBytes, err := json.Marshal(req.Config)
		if err == nil {
			query := `
				UPDATE business_objects
				SET config = $1, last_modified_at = $2, last_modified_by = $3
				WHERE tenant_id = $4::uuid AND key = $5
			`
			_, _ = s.db.ExecContext(ctx, query, configBytes, now, lastModifiedBy, tenantID, current.Key)
		}

		// Update Fields column if present in config
		if fields, ok := req.Config["fields"]; ok {
			fieldsBytes, err := json.Marshal(fields)
			if err == nil {
				// Also persist fields into normalized bo_fields table (replace existing custom fields)
				// We unmarshal the fields JSON and insert each as a bo_fields row. This keeps
				// the authoritative field list normalized for queries and UI.
				var newFields []map[string]interface{}
				if err := json.Unmarshal(fieldsBytes, &newFields); err == nil {
					logging.GetLogger().Sugar().Infof("metadata: UPDATE FIELDS - received %d fields for bo_id=%s, tenant=%s", len(newFields), current.ID, tenantID)

					// Update the fields column in business_objects table
					query := `
						UPDATE business_objects
						SET fields = $1, last_modified_at = $2, last_modified_by = $3
						WHERE tenant_id = $4::uuid AND key = $5
					`
					_, _ = s.db.ExecContext(ctx, query, fieldsBytes, now, lastModifiedBy, tenantID, current.Key)
					// Use a transaction to replace custom (non-core) fields
					tx, txErr := s.db.BeginTxx(ctx, nil)
					if txErr == nil {
						logging.GetLogger().Sugar().Errorf("Started bo_fields transaction for bo_id=%s", current.ID)
						defer func() {
							_ = tx.Rollback()
						}()
						// Delete existing custom fields for this BO from the catalog
						if _, err := tx.ExecContext(ctx, `DELETE FROM bo_fields WHERE business_object_id = $1::uuid`, current.ID); err != nil {
							logging.GetLogger().Sugar().Warnf("[FIELD_UPDATE] Failed to delete bo_fields for bo_id=%s: %v", current.ID, err)
						}

						insertQuery := `
							INSERT INTO bo_fields (
								id, tenant_id, business_object_id, key, name, field_name, display_label, technical_name, field_type, is_core, display_order, description
							) VALUES (
								$1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12
							)
							`
						for _, f := range newFields {
							id := uuid.New().String()
							name := toString(f["name"]) // using 'name' from JSON as the internal name
							displayName := toString(f["name"])
							typeName := toString(f["type"])
							seq := toInt(f["sequence"])
							desc := toString(f["description"])
							if _, err := tx.ExecContext(ctx, insertQuery,
								id, tenantID, current.ID, toString(f["key"]), name, name, displayName, toString(f["key"]), typeName, false, seq, desc,
							); err != nil {
								logging.GetLogger().Sugar().Errorf("[FIELD_UPDATE] FAILED to insert bo_field for bo_id=%s key=%s: %v", current.ID, toString(f["key"]), err)
							} else {
								logging.GetLogger().Sugar().Infof("[FIELD_UPDATE] Successfully inserted bo_field for bo_id=%s key=%s, name=%s", current.ID, toString(f["key"]), name)
							}
						}

						// Collect semantic term IDs for catalog sync event
						var selectedTermIDs []string
						for _, f := range newFields {
							if fType := toString(f["type"]); fType == "semantic_term" {
								// For semantic_term type, the key is the term ID
								if termID := toString(f["key"]); termID != "" {
									selectedTermIDs = append(selectedTermIDs, termID)
								}
							}
						}

						if err := tx.Commit(); err != nil {
							logging.GetLogger().Sugar().Errorf("[FIELD_UPDATE] FAILED to commit bo_fields transaction for bo_id=%s: %v", current.ID, err)
						} else {
							logging.GetLogger().Sugar().Infof("[FIELD_UPDATE] Successfully committed %d fields for bo_id=%s", len(newFields), current.ID)
							// Transaction committed successfully - emit catalog sync event
							logging.GetLogger().Sugar().Infof("[CATALOG_SYNC] Emitting event for BO %s with %d semantic terms", current.ID, len(selectedTermIDs))

							// Prepare event payload matching CatalogSyncEvent structure expected by catalog-worker
							var driverTableID string
							if current.DriverTableID.Valid {
								driverTableID = current.DriverTableID.String
							}
							var datasourceID string
							if current.DatasourceID.Valid {
								datasourceID = current.DatasourceID.String
							}

							catalogEvent := map[string]interface{}{
								"bo_id":           current.ID,
								"bo_key":          current.Key,
								"name":            current.Name,
								"display_name":    current.DisplayName,
								"driver_table_id": driverTableID,
								"selected_terms":  selectedTermIDs,
								"tenant_id":       tenantID,
								"datasource_id":   datasourceID,
							}

							// Start a new transaction for event publishing to ensure atomicity
							eventTx, eventTxErr := s.db.BeginTxx(ctx, nil)
							if eventTxErr != nil {
								logging.GetLogger().Sugar().Errorf("[CATALOG_SYNC] Failed to start event transaction for bo_id=%s: %v", current.ID, eventTxErr)
							} else {
								defer func() { _ = eventTx.Rollback() }()

								if publishErr := events.PublishEvent(ctx, eventTx, "BusinessObject.CatalogSync", catalogEvent); publishErr != nil {
									logging.GetLogger().Sugar().Errorf("[CATALOG_SYNC] Failed to publish event for bo_id=%s: %v", current.ID, publishErr)
								} else if commitErr := eventTx.Commit(); commitErr != nil {
									logging.GetLogger().Sugar().Errorf("[CATALOG_SYNC] Failed to commit event transaction for bo_id=%s: %v", current.ID, commitErr)
								} else {
									logging.GetLogger().Sugar().Infof("[CATALOG_SYNC] Successfully published event for BO %s (key=%s) with %d terms to catalog-worker", current.ID, current.Key, len(selectedTermIDs))
								}
							}
						}

						// After normalizing fields, create semantic mapping edges in catalog_edge for any field
						// that carries a semanticTermId. We attempt to locate the corresponding column node
						// by deriving qualified_path from the selected driver table and the field's technical name.
						// If driver table context is missing, we skip edge creation.
						// Resolve tenant_datasource_id using driver_table_id or driver_table_name.
						var tenantDatasourceID string
						var driverQualifiedPath string
						// Prefer request-provided driver table context first, then fall back to current
						if req.DriverTableID != "" {
							var tdID, qpath string
							_ = s.db.QueryRowContext(ctx, `SELECT tenant_datasource_id, qualified_path FROM catalog_node WHERE id = $1`, req.DriverTableID).Scan(&tdID, &qpath)
							tenantDatasourceID = tdID
							driverQualifiedPath = qpath
						} else if req.DriverTableName != "" {
							var tdID string
							_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(tenant_datasource_id, '') FROM catalog_node WHERE qualified_path = $1 LIMIT 1`, req.DriverTableName).Scan(&tdID)
							tenantDatasourceID = tdID
							driverQualifiedPath = req.DriverTableName
						} else if current.DriverTableID.Valid && current.DriverTableID.String != "" {
							var tdID, qpath string
							_ = s.db.QueryRowContext(ctx, `SELECT tenant_datasource_id, qualified_path FROM catalog_node WHERE id = $1`, current.DriverTableID.String).Scan(&tdID, &qpath)
							tenantDatasourceID = tdID
							driverQualifiedPath = qpath
						} else if current.DriverTableName != "" {
							var tdID string
							_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(tenant_datasource_id, '') FROM catalog_node WHERE qualified_path = $1 LIMIT 1`, current.DriverTableName).Scan(&tdID)
							tenantDatasourceID = tdID
							driverQualifiedPath = current.DriverTableName
						}

						if tenantDatasourceID == "" || driverQualifiedPath == "" {
							logging.GetLogger().Sugar().Warnf("semantic-edge: missing driver table context for bo_id=%s; skipping edge creation", current.ID)
						} else {
							for _, f := range newFields {
								// semanticTermId may appear as 'semanticTermId' or 'semantic_term_id'
								semanticTermID := toString(f["semanticTermId"])
								if semanticTermID == "" {
									semanticTermID = toString(f["semantic_term_id"])
								}
								if semanticTermID == "" {
									continue
								}

								// Technical name used to locate the column node
								technicalName := toString(f["technicalName"])
								if technicalName == "" {
									technicalName = toString(f["key"])
								}
								if technicalName == "" {
									logging.GetLogger().Sugar().Warnf("semantic-edge: field missing technical name; bo_id=%s", current.ID)
									continue
								}

								// Attempt to find the column node by exact qualified_path match schema.table.column
								candidatePath := fmt.Sprintf("%s.%s", driverQualifiedPath, technicalName)
								var columnNodeID string
								err := s.db.QueryRowContext(ctx, `
									SELECT id FROM catalog_node
									WHERE tenant_datasource_id = $1 AND qualified_path = $2
									LIMIT 1
								`, tenantDatasourceID, candidatePath).Scan(&columnNodeID)

								if err != nil || columnNodeID == "" {
									// Fallback: match by node_name within the table namespace
									_ = s.db.QueryRowContext(ctx, `
										SELECT id FROM catalog_node
										WHERE tenant_datasource_id = $1
										  AND qualified_path LIKE ($2 || '.%')
										  AND LOWER(node_name) = LOWER($3)
										LIMIT 1
									`, tenantDatasourceID, driverQualifiedPath, technicalName).Scan(&columnNodeID)
								}

								if columnNodeID == "" {
									logging.GetLogger().Sugar().Warnf("semantic-edge: could not resolve column for %s on table %s; skipping", technicalName, driverQualifiedPath)
									continue
								}

								// Create mapping edge: columnNodeID has_context semanticTermID (idempotent)
								edgeID := uuid.New().String()
								_, edgeErr := s.db.ExecContext(ctx, `
									INSERT INTO catalog_edge (
										id, tenant_datasource_id, source_node_id, target_node_id,
										relationship_type, edge_type_id, tenant_id, created_at, updated_at
									) VALUES ($1,$2,$3,$4,'has_context','0434ca1a-6543-42d3-9fce-f0b58b5fba34',$5,$6,$7)
									ON CONFLICT (tenant_datasource_id, source_node_id, edge_type_id, target_node_id)
									DO NOTHING
								`, edgeID, tenantDatasourceID, columnNodeID, semanticTermID, tenantID, time.Now(), time.Now())
								if edgeErr != nil {
									logging.GetLogger().Sugar().Warnf("semantic-edge: failed to insert has_context for column=%s term=%s: %v", columnNodeID, semanticTermID, edgeErr)
								} else {
									logging.GetLogger().Sugar().Infof("semantic-edge: created has_context %s -> %s (tenant %s, ds %s)", columnNodeID, semanticTermID, tenantID, tenantDatasourceID)

									// Sync to AGE
									if s.lineageRepo != nil {
										edge := lineage.LineageEdge{
											FromID:   columnNodeID,
											ToID:     semanticTermID,
											Type:     "has_context",
											TenantID: &tenantID,
											Env:      "dev",
										}
										if err := s.lineageRepo.UpsertEdge(ctx, edge); err != nil {
											logging.GetLogger().Sugar().Warnf("Warning: Failed to sync has_context edge to graph: %v", err)
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Always update driver table fields if they're present in the request
	// IMPORTANT: Once driver_table_id is set, it CANNOT be changed (immutable after first save)
	logging.GetLogger().Sugar().Infof("DEBUG immutability check: current.DriverTableID.Valid=%v, current.DriverTableID.String=%q, req.DriverTableID=%q",
		current.DriverTableID.Valid, current.DriverTableID.String, req.DriverTableID)
	if current.DriverTableID.Valid && current.DriverTableID.String != "" {
		// Driver table already set - prevent changes
		if req.DriverTableID != "" && req.DriverTableID != current.DriverTableID.String {
			logging.GetLogger().Sugar().Errorf("IMMUTABILITY VIOLATION: attempted to change driver_table_id from %q to %q", current.DriverTableID.String, req.DriverTableID)
			return nil, fmt.Errorf("driver_table_id cannot be changed once set (current: %s, attempted: %s)", current.DriverTableID.String, req.DriverTableID)
		}
		// Keep existing value
		// current.DriverTableID remains unchanged
	} else {
		// Not yet set - allow setting it now
		if req.DriverTableID == "" {
			current.DriverTableID = sql.NullString{Valid: false}
		} else {
			current.DriverTableID = sql.NullString{String: req.DriverTableID, Valid: true}
		}
	}

	if req.DriverTableName != "" {
		current.DriverTableName = req.DriverTableName
	} else if !current.DriverTableID.Valid {
		// Only clear driver_table_name if driver_table_id is also not set
		current.DriverTableName = ""
	}

	// Check if boKey is a UUID to determine which field to match on
	isUUID := false
	if _, err := uuid.Parse(boKey); err == nil {
		isUUID = true
	}

	var query string
	if isUUID {
		query = `
			UPDATE business_objects
			SET display_name = $1, description = $2, icon = $3, category = $4,
				is_active = $5, last_modified_at = $6, last_modified_by = $7,
				driver_table_id = $8, driver_table_name = $9
			WHERE tenant_id = $10::uuid AND id = CAST($11 AS uuid)
		`
	} else {
		query = `
			UPDATE business_objects
			SET display_name = $1, description = $2, icon = $3, category = $4,
				is_active = $5, last_modified_at = $6, last_modified_by = $7,
				driver_table_id = $8, driver_table_name = $9
			WHERE tenant_id = $10::uuid AND key = $11
		`
	}

	_, err = s.db.ExecContext(ctx, query,
		current.DisplayName, current.Description, current.Icon, current.Category,
		current.IsActive, now, lastModifiedBy,
		current.DriverTableID, current.DriverTableName,
		tenantID, boKey,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update business object: %w", err)
	}

	// Log audit
	changes := map[string]interface{}{
		"displayName":     req.DisplayName,
		"description":     req.Description,
		"icon":            req.Icon,
		"category":        req.Category,
		"driverTableId":   req.DriverTableID,
		"driverTableName": req.DriverTableName,
		"isActive":        req.IsActive,
	}
	s.logAuditByKey(ctx, tenantID, "business_object", current.Key, "update", changes, userID)

	// Debug: check bo_fields count for this BO
	var bfCount int
	_ = s.db.GetContext(ctx, &bfCount, "SELECT COUNT(*) FROM bo_fields WHERE business_object_id = $1::uuid", current.ID)
	logging.GetLogger().Sugar().Infof("[FIELD_UPDATE] bo_fields count for bo_id=%s -> %d (before GetBusinessObject)", current.ID, bfCount)

	return s.GetBusinessObject(ctx, secCtx, current.Key)
}

// DeleteBusinessObject deletes a BO and all associated data
func (s *BusinessObjectService) DeleteBusinessObject(
	ctx context.Context,
	secCtx *security.Context,
	boKey string,
	userID string,
) error {
	tenantID := secCtx.TenantID
	// Get the BO first to get its ID for logging
	bo, err := s.GetBusinessObject(ctx, secCtx, boKey)
	if err != nil {
		return fmt.Errorf("business object not found: %w", err)
	}

	// Enforce Write Access
	if _, err := s.requireAccess(ctx, tenantID, bo.ID, AccessLevelWrite); err != nil {
		return err
	}

	query := `
		DELETE FROM business_objects
		WHERE tenant_id = $1::uuid AND key = $2
	`

	_, err = s.db.ExecContext(ctx, query, tenantID, bo.Key)
	if err != nil {
		return fmt.Errorf("failed to delete business object: %w", err)
	}

	// Log audit
	s.logAudit(ctx, tenantID, "business_object", bo.ID, "delete", nil, userID)

	return nil
}

// RenameSubtype renames a subtype within a business object
func (s *BusinessObjectService) RenameSubtype(
	ctx context.Context,
	secCtx *security.Context,
	boKey, subtypeKey, newName, userID string,
) (*models.BusinessObjectDefinition, error) {
	tenantID := secCtx.TenantID
	// Get the parent business object
	bo, err := s.GetBusinessObject(ctx, secCtx, boKey)
	if err != nil {
		return nil, fmt.Errorf("business object not found: %w", err)
	}

	// Check if subtype exists by key or ID
	var subtype models.SubtypeDefinition
	var foundKey string
	if s, ok := bo.Subtypes[subtypeKey]; ok {
		subtype = s
		foundKey = subtypeKey
	} else {
		// Try to find by ID
		for k, s := range bo.Subtypes {
			if s.ID == subtypeKey {
				subtype = s
				foundKey = k
				break
			}
		}
	}

	if foundKey == "" {
		return nil, fmt.Errorf("subtype not found: %s", subtypeKey)
	}

	// Update the subtype name
	subtype.Name = newName
	subtype.DisplayName = newName
	bo.Subtypes[foundKey] = subtype

	// Update the business object with new subtypes in the config
	subtypesJSON, err := json.Marshal(bo.Subtypes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subtypes: %w", err)
	}

	query := `
		UPDATE business_objects
		SET config = jsonb_set(config, '{subtypes}', $1::jsonb),
		    last_modified_at = $2,
		    last_modified_by = $3
		WHERE id = $4::uuid AND tenant_id = $5::uuid
	`

	now := time.Now()
	var lastModifiedBy interface{} = nil
	if userID != "" {
		lastModifiedBy = userID
	}
	_, err = s.db.ExecContext(ctx, query, string(subtypesJSON), now, lastModifiedBy, bo.ID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to rename subtype in config: %w", err)
	}

	// Also update in bo_subtypes if it exists there
	_, _ = s.db.ExecContext(ctx, `
		UPDATE bo_subtypes 
		SET name = $1, display_name = $1, last_modified_at = $2, last_modified_by = $3
		WHERE (id = $4::uuid OR key = $4) AND business_object_id = $5::uuid
	`, newName, now, lastModifiedBy, subtypeKey, bo.ID)

	// Also update in business_objects if it's a child BO
	_, _ = s.db.ExecContext(ctx, `
		UPDATE business_objects
		SET name = $1, display_name = $1, last_modified_at = $2, last_modified_by = $3
		WHERE (id = $4::uuid OR key = $4) AND parent_id = $5::uuid
	`, newName, now, lastModifiedBy, subtypeKey, bo.ID)

	// Log audit
	s.logAudit(ctx, tenantID, "subtype", bo.ID, "rename", map[string]interface{}{
		"subtype_key": subtypeKey,
		"old_name":    subtype.DisplayName,
		"new_name":    newName,
	}, userID)

	// Return updated business object
	return s.GetBusinessObject(ctx, secCtx, boKey)
}

// DeleteSubtype deletes a subtype from a business object
func (s *BusinessObjectService) DeleteSubtype(
	ctx context.Context,
	secCtx *security.Context,
	boKey, subtypeKey, userID string,
) (*models.BusinessObjectDefinition, error) {
	tenantID := secCtx.TenantID

	// Get the parent business object
	bo, err := s.GetBusinessObject(ctx, secCtx, boKey)
	if err != nil {
		return nil, fmt.Errorf("business object not found: %w", err)
	}

	// Check if subtype exists by key or ID
	var subtype models.SubtypeDefinition
	var foundKey string
	if s, ok := bo.Subtypes[subtypeKey]; ok {
		subtype = s
		foundKey = subtypeKey
	} else {
		// Try to find by ID
		for k, s := range bo.Subtypes {
			if s.ID == subtypeKey {
				subtype = s
				foundKey = k
				break
			}
		}
	}

	if foundKey == "" {
		return nil, fmt.Errorf("subtype not found: %s", subtypeKey)
	}

	// Remove from the map
	delete(bo.Subtypes, foundKey)

	// Update the business object with new subtypes in the config
	subtypesJSON, err := json.Marshal(bo.Subtypes)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal subtypes: %w", err)
	}

	query := `
		UPDATE business_objects
		SET config = jsonb_set(config, '{subtypes}', $1::jsonb),
		    last_modified_at = $2,
		    last_modified_by = $3
		WHERE id = $4::uuid AND tenant_id = $5::uuid
	`

	now := time.Now()
	var lastModifiedBy interface{} = nil
	if userID != "" {
		lastModifiedBy = userID
	}
	_, err = s.db.ExecContext(ctx, query, string(subtypesJSON), now, lastModifiedBy, bo.ID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete subtype from config: %w", err)
	}

	// Also delete from bo_subtypes table if it exists there
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM bo_subtypes 
		WHERE key = $1 AND business_object_id = $2::uuid
	`, subtypeKey, bo.ID)

	// Also delete from business_objects table if it's a child BO
	_, _ = s.db.ExecContext(ctx, `
		DELETE FROM business_objects
		WHERE (key = $1 OR id = CAST($2 AS uuid)) AND parent_id = $3::uuid
	`, subtypeKey, subtypeKey, bo.ID)

	// Log audit
	s.logAudit(ctx, tenantID, "subtype", bo.ID, "delete", map[string]interface{}{
		"subtype_key":  foundKey,
		"subtype_name": subtype.DisplayName,
		"subtype_id":   subtype.ID,
	}, userID)

	// Return updated business object
	return s.GetBusinessObject(ctx, secCtx, boKey)
}

// ============================================================================
// CLONE OPERATIONS
// ============================================================================

// CloneBusinessObject creates a clone of an existing BO
func (s *BusinessObjectService) CloneBusinessObject(
	ctx context.Context,
	secCtx *security.Context,
	req models.CloneBORequest,
	userID string,
) (*models.BusinessObjectDefinition, error) {
	sourceBO, err := s.GetBusinessObject(ctx, secCtx, req.SourceBOKey)
	if err != nil {
		return nil, fmt.Errorf("source business object not found: %w", err)
	}

	createReq := models.CreateBusinessObjectRequest{
		Name:         req.NewName,
		DisplayName:  req.NewName,
		Description:  req.Description,
		Icon:         req.Icon,
		CloneFromKey: sourceBO.Key,
	}

	return s.CreateBusinessObject(ctx, secCtx, createReq, userID)
}

func (s *BusinessObjectService) cloneBO(
	ctx context.Context,
	tenantID string,
	newBO *models.BusinessObjectDefinition,
	sourceKey string,
	userID string,
) error {
	sourceBO, err := s.GetBusinessObject(ctx, &security.Context{TenantID: tenantID}, sourceKey)
	if err != nil {
		return fmt.Errorf("failed to get source BO for cloning: %w", err)
	}

	newBO.ClonesFrom = sourceBO.Key
	newBO.CloneParentKey = sourceBO.Key
	newBO.CloneParentDisplayName = sourceBO.Name

	// Copy core fields
	for _, field := range sourceBO.CoreFields {
		newField := field
		newField.ID = uuid.New().String()
		newField.IsCore = false // Cloned fields are no longer "core"
		newBO.CoreFields = append(newBO.CoreFields, newField)
	}

	// Copy subtypes with their fields
	for subtypeKey, subtype := range sourceBO.Subtypes {
		newSubtype := models.SubtypeDefinition{
			ID:             uuid.New().String(),
			Key:            subtype.Key,
			Name:           subtype.Name,
			DisplayName:    subtype.DisplayName,
			TechnicalName:  subtype.TechnicalName,
			Description:    subtype.Description,
			IsCore:         false,
			BasedOnEntity:  sourceBO.Key,
			CloneParentKey: sourceBO.Key,
		}

		for _, field := range subtype.SubtypeFields {
			newField := field
			newField.ID = uuid.New().String()
			newField.IsCore = false
			newSubtype.SubtypeFields = append(newSubtype.SubtypeFields, newField)
		}

		newBO.Subtypes[subtypeKey] = newSubtype
	}

	return nil
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func (s *BusinessObjectService) loadBOSubtypesAndFields(
	ctx context.Context,
	bo *models.BusinessObjectDefinition,
	viewTenantID string,
) error {
	logging.GetLogger().Sugar().Infof("DEBUG: loadBOSubtypesAndFields start - bo.id=%s tenant=%s", bo.ID, bo.TenantID)
	if bo.Subtypes == nil {
		bo.Subtypes = make(map[string]models.SubtypeDefinition)
	}

	// STRATEGY 0: Load subtypes from config JSONB column (if present)
	if len(bo.Config) > 0 {
		var configMap map[string]interface{}
		if err := json.Unmarshal(bo.Config, &configMap); err == nil {
			if subtypesRaw, ok := configMap["subtypes"].(map[string]interface{}); ok {
				for k, v := range subtypesRaw {
					// Re-marshal and unmarshal to get proper struct
					vJSON, _ := json.Marshal(v)
					var sd models.SubtypeDefinition
					if err := json.Unmarshal(vJSON, &sd); err == nil {
						bo.Subtypes[k] = sd
					}
				}
			}
		}
	}

	// STRATEGY 1: Load child business objects via parent_id (inheritance pattern)
	// We look for subtypes in EITHER the BO's tenant (e.g. gold copy) OR the viewer's tenant (custom extensions)
	childBOQuery := `
		SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, 
		       COALESCE(description, '') AS description, is_core, tenant_id
		FROM business_objects
		WHERE parent_id = $1::uuid AND (tenant_id = $2::uuid OR tenant_id = $3::uuid)
		ORDER BY name
	`

	type ChildBO struct {
		ID            string `db:"id"`
		Key           string `db:"key"`
		Name          string `db:"name"`
		DisplayName   string `db:"display_name"`
		TechnicalName string `db:"technical_name"`
		Description   string `db:"description"`
		IsCore        bool   `db:"is_core"`
		TenantID      string `db:"tenant_id"`
	}

	var childBOs []ChildBO
	if err := s.db.SelectContext(ctx, &childBOs, childBOQuery, bo.ID, bo.TenantID, viewTenantID); err != nil {
		logging.GetLogger().Sugar().Warnf("Warning: failed to load child business objects (with tenant filter): %v", err)
	}

	// Fallback query removed for security - we must strictly respect tenant boundaries

	logging.GetLogger().Sugar().Infof("DEBUG: loading %d child BO(s) for parent %s (tenant %s)", len(childBOs), bo.ID, bo.TenantID)

	// For each child BO, load its fields and add as subtype
	for _, child := range childBOs {
		// Load fields for this child BO
		fieldQuery := `
			SELECT id, key, name, display_label AS display_name, COALESCE(technical_name, '') AS technical_name, field_type AS type,
			       COALESCE(is_core, false) AS is_core, COALESCE(is_required, false) AS is_required,
			       COALESCE(is_system_field, false) AS is_system, COALESCE(description, '') AS description,
			       COALESCE(reference_bo_id::text, '') AS reference_entity, COALESCE(display_order, 0) AS sequence,
			       created_at, '' AS created_by,
			       created_at AS last_modified_at, '' AS last_modified_by
			FROM bo_fields
			WHERE business_object_id::text = $1 AND tenant_id::text = $2 AND subtype_id IS NULL
			ORDER BY display_order
		`

		var fields []models.FieldDefinition
		if err := s.db.SelectContext(ctx, &fields, fieldQuery, child.ID, child.TenantID); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to load fields for child BO %s: %v", child.Key, err)

			// Try old schema fallback for bo_fields (legacy deployments)
			oldFieldQuery := `
				SELECT id, business_object_id, field_name, display_label, field_type, is_required, is_readonly, is_searchable, is_sortable, display_order
				FROM bo_fields
				WHERE business_object_id = $1
				ORDER BY display_order
			`
			type OldField struct {
				ID           string `db:"id"`
				BoID         string `db:"business_object_id"`
				FieldName    string `db:"field_name"`
				DisplayLabel string `db:"display_label"`
				FieldType    string `db:"field_type"`
				IsRequired   bool   `db:"is_required"`
				IsReadOnly   bool   `db:"is_readonly"`
				IsSearchable bool   `db:"is_searchable"`
				IsSortable   bool   `db:"is_sortable"`
				Sequence     int    `db:"display_order"`
			}

			var oldFields []OldField
			if err2 := s.db.SelectContext(ctx, &oldFields, oldFieldQuery, child.ID); err2 != nil {
				logging.GetLogger().Sugar().Warnf("Warning: failed to load fields for child BO %s (old schema): %v", child.Key, err2)
				continue
			}

			// Map old fields to models.FieldDefinition
			fields = make([]models.FieldDefinition, 0, len(oldFields))
			for _, of := range oldFields {
				f := models.FieldDefinition{
					ID:          of.ID,
					Key:         of.FieldName,
					Name:        of.FieldName,
					DisplayName: of.DisplayLabel,
					Type:        of.FieldType,
					IsCore:      false,
					IsRequired:  of.IsRequired,
					IsSystem:    of.IsReadOnly,
					Sequence:    of.Sequence,
				}
				fields = append(fields, f)
			}
		}

		// Create subtype definition from child BO
		// Only use child BO data if config doesn't already have an entry for this key
		// (Config takes precedence because it has the latest renamed values)
		if _, exists := bo.Subtypes[child.Key]; !exists {
			subtype := models.SubtypeDefinition{
				ID:            child.ID,
				Key:           child.Key,
				Name:          child.Name,
				DisplayName:   child.DisplayName,
				TechnicalName: child.TechnicalName,
				Description:   child.Description,
				IsCore:        child.IsCore,
				BasedOnEntity: bo.Key, // Parent BO key
				SubtypeFields: fields,
			}
			bo.Subtypes[child.Key] = subtype
		}
	}

	// STRATEGY 2: Also load from bo_subtypes table for backward compatibility
	subtypeQuery := `
		SELECT id, key, name, display_name, COALESCE(technical_name, '') AS technical_name, 
		       COALESCE(description, '') AS description, is_core, based_on_entity, 
		       COALESCE(clone_parent_key, '') AS clone_parent_key, sequence, created_at, 
		       COALESCE(created_by, '') AS created_by, last_modified_at, COALESCE(last_modified_by, '') AS last_modified_by
		FROM bo_subtypes
		WHERE business_object_id = $1::uuid
		ORDER BY sequence
	`

	var legacySubtypes []models.SubtypeDefinition
	if err := s.db.SelectContext(ctx, &legacySubtypes, subtypeQuery, bo.ID); err != nil {
		logging.GetLogger().Sugar().Warnf("Warning: failed to load bo_subtypes: %v", err)
	}

	// Load fields for legacy subtypes
	for i := range legacySubtypes {
		displayNameExpr := s.boFieldsDisplayNameExpr(ctx, "")
		fieldQuery := fmt.Sprintf(`
			SELECT id, key, name, %s AS display_name, COALESCE(technical_name, '') AS technical_name, field_type,
			       is_core, is_required, is_readonly AS is_system, COALESCE(description, '') AS description,
			       COALESCE(reference_entity, '') AS reference_entity, sequence,
			       created_at, COALESCE(created_by, '') AS created_by, last_modified_at, 
			       COALESCE(last_modified_by, '') AS last_modified_by
			FROM bo_fields
			WHERE subtype_id = $1
			ORDER BY sequence
		`, displayNameExpr)

		var fields []models.FieldDefinition
		if err := s.db.SelectContext(ctx, &fields, fieldQuery, legacySubtypes[i].ID); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to load subtype fields: %v", err)

			// Try old schema fallback (subtype-based fields stored in bo_fields with subtype_id)
			oldFieldQuery := `
				SELECT id, business_object_id, field_name, display_label, field_type, is_required, is_readonly, is_searchable, is_sortable, display_order
				FROM bo_fields
				WHERE subtype_id = $1
				ORDER BY display_order
			`
			type OldField struct {
				ID           string `db:"id"`
				BoID         string `db:"business_object_id"`
				FieldName    string `db:"field_name"`
				DisplayLabel string `db:"display_label"`
				FieldType    string `db:"field_type"`
				IsRequired   bool   `db:"is_required"`
				IsReadOnly   bool   `db:"is_readonly"`
				IsSearchable bool   `db:"is_searchable"`
				IsSortable   bool   `db:"is_sortable"`
				Sequence     int    `db:"display_order"`
			}

			var oldFields []OldField
			if err2 := s.db.SelectContext(ctx, &oldFields, oldFieldQuery, legacySubtypes[i].ID); err2 != nil {
				logging.GetLogger().Sugar().Warnf("Warning: failed to load subtype fields (old schema): %v", err2)
				continue
			}

			fields = make([]models.FieldDefinition, 0, len(oldFields))
			for _, of := range oldFields {
				f := models.FieldDefinition{
					ID:          of.ID,
					Key:         of.FieldName,
					Name:        of.FieldName,
					DisplayName: of.DisplayLabel,
					Type:        of.FieldType,
					IsCore:      false,
					IsRequired:  of.IsRequired,
					IsSystem:    of.IsReadOnly,
					Sequence:    of.Sequence,
				}
				fields = append(fields, f)
			}
		}

		legacySubtypes[i].SubtypeFields = fields

		// Only add if not already loaded from child BOs
		if _, exists := bo.Subtypes[legacySubtypes[i].Key]; !exists {
			bo.Subtypes[legacySubtypes[i].Key] = legacySubtypes[i]
		}
	}

	// Load entity-level fields (non-subtype fields)
	var entityFields []models.FieldDefinition

	// MIGRATION STRATEGY: Prefer loading fields from Config JSONB if available
	// This ensures we get the latest state even if the normalized tables (bo_fields) are out of sync or missing
	if len(bo.Config) > 0 {
		var configMap map[string]interface{}
		if err := json.Unmarshal(bo.Config, &configMap); err == nil {
			if fieldsRaw, ok := configMap["fields"]; ok {
				// Marshal/Unmarshal to convert map structures to FieldDefinition slice
				if fieldsJSON, err := json.Marshal(fieldsRaw); err == nil {
					_ = json.Unmarshal(fieldsJSON, &entityFields)
				}
			}
		}
	}

	fieldQuery := `
		SELECT id, key, name, display_label AS display_name, COALESCE(technical_name, '') AS technical_name, field_type AS type,
		       COALESCE(is_core, false) AS is_core, COALESCE(is_required, false) AS is_required,
		       COALESCE(is_system_field, false) AS is_system, COALESCE(description, '') AS description,
		       COALESCE(reference_bo_id::text, '') AS reference_entity, COALESCE(display_order, 0) AS sequence,
		       created_at, '' AS created_by, created_at AS last_modified_at, 
		       '' AS last_modified_by,
		       COALESCE(CAST(semantic_term_id AS text), '') AS semantic_term_id
		FROM bo_fields
		WHERE business_object_id::text = $1 AND tenant_id::text = $2 AND subtype_id IS NULL
		ORDER BY display_order
	`

	// If config didn't yield fields, try the DB tables
	if len(entityFields) == 0 {
		if err := s.db.SelectContext(ctx, &entityFields, fieldQuery, bo.ID, bo.TenantID); err != nil {
			logging.GetLogger().Sugar().Warnf("Warning: failed to load entity fields (new schema): %v", err)
		}
	}

	// Try old schema fallback where bo_fields stores field info differently
	if len(entityFields) == 0 {
		oldFieldQuery := `
			SELECT id, business_object_id, field_name, display_label, field_type, is_required, is_readonly, is_searchable, is_sortable, display_order
			FROM bo_fields
			WHERE business_object_id = $1
			ORDER BY display_order
		`
		type OldField struct {
			ID           string `db:"id"`
			BoID         string `db:"business_object_id"`
			FieldName    string `db:"field_name"`
			DisplayLabel string `db:"display_label"`
			FieldType    string `db:"field_type"`
			IsRequired   bool   `db:"is_required"`
			IsReadOnly   bool   `db:"is_readonly"`
			IsSearchable bool   `db:"is_searchable"`
			IsSortable   bool   `db:"is_sortable"`
			Sequence     int    `db:"display_order"`
		}

		var oldFields []OldField
		if err2 := s.db.SelectContext(ctx, &oldFields, oldFieldQuery, bo.ID); err2 != nil {
			return fmt.Errorf("failed to load entity fields (old schema): %w", err2)
		}

		entityFields = make([]models.FieldDefinition, 0, len(oldFields))
		for _, of := range oldFields {
			f := models.FieldDefinition{
				ID:          of.ID,
				Key:         of.FieldName,
				Name:        of.FieldName,
				DisplayName: of.DisplayLabel,
				Type:        of.FieldType,
				IsCore:      false,
				IsRequired:  of.IsRequired,
				IsSystem:    of.IsReadOnly,
				Sequence:    of.Sequence,
			}
			entityFields = append(entityFields, f)
		}
	}

	logging.GetLogger().Sugar().Infof("DEBUG: loadBOSubtypesAndFields - entity fields loaded: %d for bo.id=%s", len(entityFields), bo.ID)
	bo.CoreFields = []models.FieldDefinition{}
	bo.CustomFields = []models.FieldDefinition{}

	for _, field := range entityFields {
		if field.IsCore {
			bo.CoreFields = append(bo.CoreFields, field)
		} else {
			bo.CustomFields = append(bo.CustomFields, field)
		}
	}

	return nil
}

func (s *BusinessObjectService) logAudit(
	ctx context.Context,
	tenantID, entityType, entityID, action string,
	changes map[string]interface{},
	userID string,
) {
	// Prefer publishing to RabbitMQ audit exchange; fallback is no-op if publisher nil
	if s.auditPublisher != nil {
		evt := events.AuditEvent{
			ID:         uuid.New().String(),
			InstanceID: entityID,
			TenantID:   tenantID,
			BPKey:      entityType,
			EventType:  action,
			StepKey:    "",
			ActorID:    userID,
			ActorRole:  "",
			OldValue:   map[string]interface{}{},
			NewValue:   changes,
			Reason:     "",
			IPAddress:  "",
			UserAgent:  "",
			CreatedAt:  time.Now().Format(time.RFC3339),
		}
		_ = s.auditPublisher.PublishAuditEvent(ctx, evt)
		return
	}
	// If publisher not configured, do nothing (avoid DB dependency per new audit pipeline)
}

func (s *BusinessObjectService) logAuditByKey(
	ctx context.Context,
	tenantID, entityType, entityKey, action string,
	changes map[string]interface{},
	userID string,
) {
	if s.auditPublisher != nil {
		evt := events.AuditEvent{
			ID:         uuid.New().String(),
			InstanceID: entityKey,
			TenantID:   tenantID,
			BPKey:      entityType,
			EventType:  action,
			StepKey:    "",
			ActorID:    userID,
			ActorRole:  "",
			OldValue:   map[string]interface{}{},
			NewValue:   changes,
			Reason:     "",
			IPAddress:  "",
			UserAgent:  "",
			CreatedAt:  time.Now().Format(time.RFC3339),
		}
		_ = s.auditPublisher.PublishAuditEvent(ctx, evt)
		return
	}
}

func slugify(s string) string {
	// Simple slugify: lowercase, replace spaces with underscore
	result := ""
	for _, c := range s {
		if c == ' ' {
			result += "_"
		} else if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			result += string(c)
		} else if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		}
	}
	return result
}

// ============================================================================
// BUSINESS OBJECT INSTANCE OPERATIONS (Tenant DB)
// ============================================================================

// CreateInstance creates a new business object instance
func (s *BusinessObjectService) CreateInstance(ctx context.Context, tenantID, userID string, instance *models.BusinessObjectInstance) (*models.BusinessObjectInstance, error) {
	// 1. Get Tenant DB Connection
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return nil, err
	}

	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}

	instance.TenantID = tenantID
	instance.CreatedAt = time.Now()
	instance.CreatedBy = userID
	instance.LastModifiedAt = time.Now()
	instance.LastModifiedBy = userID
	instance.IsDeleted = false

	query := `
		INSERT INTO bo_instances (
			id, tenant_id, business_object_id, business_object_key, datasource_id,
			subtype_id, subtype_key, core_field_values, custom_field_values,
			created_at, created_by, last_modified_at, last_modified_by, is_deleted
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
	`

	coreJSON, _ := json.Marshal(instance.CoreFieldValues)
	customJSON, _ := json.Marshal(instance.CustomFieldValues)

	_, err = db.ExecContext(ctx, query,
		instance.ID,
		tenantID,
		instance.BusinessObjectID,
		instance.BusinessObjectKey,
		instance.DatasourceID,
		instance.SubtypeID,
		instance.SubtypeKey,
		coreJSON,
		customJSON,
		instance.CreatedAt,
		userID,
		instance.LastModifiedAt,
		userID,
		false,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	s.logInstanceAction(ctx, tenantID, instance.BusinessObjectKey, instance.ID, "CREATE", "Instance created")
	return instance, nil
}

// GetInstance retrieves a single business object instance
func (s *BusinessObjectService) GetInstance(ctx context.Context, tenantID, instanceID string) (*models.BusinessObjectInstance, error) {
	// 1. Get Tenant DB Connection
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return nil, err
	}

	instance := &models.BusinessObjectInstance{}

	query := `
		SELECT 
			id, tenant_id, business_object_id, business_object_key, datasource_id,
			subtype_id, subtype_key, core_field_values, custom_field_values,
			created_at, created_by, last_modified_at, last_modified_by, is_deleted, deleted_at
		FROM bo_instances
		WHERE id = $1 AND tenant_id = $2 AND is_deleted = false
	`

	var coreJSON, customJSON []byte

	// Use QueryRow because sql.DB doesn't have SelectContext/GetContext from sqlx directly unless we wrap it
	// But TenantDBManager returns *sql.DB. We can wrap it or just use standard sql.
	row := db.QueryRowContext(ctx, query, instanceID, tenantID)
	err = row.Scan(
		&instance.ID,
		&instance.TenantID,
		&instance.BusinessObjectID,
		&instance.BusinessObjectKey,
		&instance.DatasourceID,
		&instance.SubtypeID,
		&instance.SubtypeKey,
		&coreJSON,
		&customJSON,
		&instance.CreatedAt,
		&instance.CreatedBy,
		&instance.LastModifiedAt,
		&instance.LastModifiedBy,
		&instance.IsDeleted,
		&instance.DeletedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if len(coreJSON) > 0 {
		json.Unmarshal(coreJSON, &instance.CoreFieldValues)
	}
	if len(customJSON) > 0 {
		json.Unmarshal(customJSON, &instance.CustomFieldValues)
	}

	return instance, nil
}

// ListInstances lists business object instances with pagination
func (s *BusinessObjectService) ListInstances(ctx context.Context, tenantID, boKey string, offset, limit int) ([]*models.BusinessObjectInstance, int, error) {
	// 1. Get Tenant DB Connection
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return nil, 0, err
	}

	countQuery := `
		SELECT COUNT(*) FROM bo_instances
		WHERE tenant_id = $1 AND business_object_key = $2 AND is_deleted = false
	`

	var total int
	err = db.QueryRowContext(ctx, countQuery, tenantID, boKey).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count instances: %w", err)
	}

	query := `
		SELECT 
			id, tenant_id, business_object_id, business_object_key, datasource_id,
			subtype_id, subtype_key, core_field_values, custom_field_values,
			created_at, created_by, last_modified_at, last_modified_by, is_deleted, deleted_at
		FROM bo_instances
		WHERE tenant_id = $1 AND business_object_key = $2 AND is_deleted = false
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := db.QueryContext(ctx, query, tenantID, boKey, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list instances: %w", err)
	}
	defer rows.Close()

	var instances []*models.BusinessObjectInstance

	for rows.Next() {
		instance := &models.BusinessObjectInstance{}
		var coreJSON, customJSON []byte

		err := rows.Scan(
			&instance.ID,
			&instance.TenantID,
			&instance.BusinessObjectID,
			&instance.BusinessObjectKey,
			&instance.DatasourceID,
			&instance.SubtypeID,
			&instance.SubtypeKey,
			&coreJSON,
			&customJSON,
			&instance.CreatedAt,
			&instance.CreatedBy,
			&instance.LastModifiedAt,
			&instance.LastModifiedBy,
			&instance.IsDeleted,
			&instance.DeletedAt,
		)

		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan instance: %w", err)
		}

		if len(coreJSON) > 0 {
			json.Unmarshal(coreJSON, &instance.CoreFieldValues)
		}
		if len(customJSON) > 0 {
			json.Unmarshal(customJSON, &instance.CustomFieldValues)
		}

		instances = append(instances, instance)
	}

	return instances, total, nil
}

// UpdateInstance updates a business object instance
func (s *BusinessObjectService) UpdateInstance(ctx context.Context, tenantID, instanceID, userID string, coreUpdates, customUpdates map[string]interface{}) (*models.BusinessObjectInstance, error) {
	// 1. Get Tenant DB Connection
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return nil, err
	}

	instance, err := s.GetInstance(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	if coreUpdates != nil {
		if instance.CoreFieldValues == nil {
			instance.CoreFieldValues = make(map[string]interface{})
		}
		for key, value := range coreUpdates {
			instance.CoreFieldValues[key] = value
		}
	}

	if customUpdates != nil {
		if instance.CustomFieldValues == nil {
			instance.CustomFieldValues = make(map[string]interface{})
		}
		for key, value := range customUpdates {
			instance.CustomFieldValues[key] = value
		}
	}

	instance.LastModifiedAt = time.Now()
	instance.LastModifiedBy = userID

	query := `
		UPDATE bo_instances
		SET core_field_values = $1, custom_field_values = $2, 
		    last_modified_at = $3, last_modified_by = $4
		WHERE id = $5 AND tenant_id = $6
	`

	coreJSON, _ := json.Marshal(instance.CoreFieldValues)
	customJSON, _ := json.Marshal(instance.CustomFieldValues)

	_, err = db.ExecContext(ctx, query,
		coreJSON,
		customJSON,
		instance.LastModifiedAt,
		userID,
		instanceID,
		tenantID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update instance: %w", err)
	}

	s.logInstanceAction(ctx, tenantID, instance.BusinessObjectKey, instanceID, "UPDATE", "Instance updated")
	return instance, nil
}

// DeleteInstance soft-deletes a business object instance
func (s *BusinessObjectService) DeleteInstance(ctx context.Context, tenantID, instanceID, userID string) error {
	// 1. Get Tenant DB Connection
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	now := time.Now()

	query := `
		UPDATE bo_instances
		SET is_deleted = true, deleted_at = $1, last_modified_at = $2, last_modified_by = $3
		WHERE id = $4 AND tenant_id = $5
	`

	result, err := db.ExecContext(ctx, query, now, now, userID, instanceID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("instance not found")
	}

	s.logInstanceAction(ctx, tenantID, "", instanceID, "DELETE", "Instance deleted")
	return nil
}

// HardDeleteInstance permanently deletes a business object instance
func (s *BusinessObjectService) HardDeleteInstance(ctx context.Context, tenantID, instanceID string) error {
	// 1. Get Tenant DB Connection
	db, err := s.tenantManager.GetConnection(tenantID)
	if err != nil {
		return err
	}

	query := `DELETE FROM bo_instances WHERE id = $1 AND tenant_id = $2`
	_, err = db.ExecContext(ctx, query, instanceID, tenantID)
	return err
}

// logInstanceAction logs instance operations to audit table
func (s *BusinessObjectService) logInstanceAction(ctx context.Context, tenantID, boKey, instanceID, action, details string) {
	if s.auditPublisher != nil {
		evt := events.AuditEvent{
			ID:         uuid.New().String(),
			InstanceID: instanceID,
			TenantID:   tenantID,
			BPKey:      boKey,
			EventType:  action,
			StepKey:    "",
			ActorID:    "",
			ActorRole:  "",
			OldValue:   map[string]interface{}{},
			NewValue:   map[string]interface{}{"details": details},
			Reason:     "",
			IPAddress:  "",
			UserAgent:  "",
			CreatedAt:  time.Now().Format(time.RFC3339),
		}
		_ = s.auditPublisher.PublishAuditEvent(ctx, evt)
	}
}

// GetBusinessObjectRelationships retrieves related objects and semantic mappings for a BO
func (s *BusinessObjectService) GetBusinessObjectRelationships(ctx context.Context, secCtx *security.Context, boID string) (*BORelationshipsResponse, error) {
	tenantID := secCtx.TenantID
	// 1. Get the BO to find its driver table
	boQuery := `SELECT driver_table_id FROM business_objects WHERE id = $1::uuid AND tenant_id = $2::uuid`
	var driverTableID sql.NullString
	err := s.db.GetContext(ctx, &driverTableID, boQuery, boID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver table for BO %s: %w", boID, err)
	}

	response := &BORelationshipsResponse{
		RelatedObjects: []RelationshipResult{},
		SemanticFields: []SemanticFieldResult{},
	}

	if !driverTableID.Valid || driverTableID.String == "" {
		return response, nil
	}

	// 2. Find related objects via catalog edges
	// Query edges where the driver table is source or target
	relatedQuery := `
		SELECT 
			CASE 
				WHEN e.source_node_id = $1::uuid THEN t.name 
				ELSE src.name 
			END as related_object_name,
			e.edge_type_name as relationship_type,
			COALESCE(e.fk_column, '') || ' ' || COALESCE(e.cardinality, '') as description
		FROM catalog_edge e
		JOIN catalog_node src ON e.source_node_id = src.id
		JOIN catalog_node t ON e.target_node_id = t.id
		WHERE (e.source_node_id = $1::uuid OR e.target_node_id = $1::uuid)
	`
	// Note: We might want to filter out edges to columns, but for now assuming table-to-table edges are direct

	err = s.db.SelectContext(ctx, &response.RelatedObjects, relatedQuery, driverTableID.String)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to fetch related objects for BO %s: %v", boID, err)
		// Don't fail the whole request
	}

	// 3. Find semantic field mappings
	// Find columns of the driver table (parent_id = driver_table_id) that have edges to other nodes (semantic terms)
	// We assume semantic terms have a specific kind or we just list all non-structural edges
	// Relaxing the 'kind' check if we are unsure, or matching specific edge types
	// Using a broader query for now:
	semanticQueryv2 := `
		SELECT 
			col.name as field_name,
			term.name as semantic_term_name,
			e.edge_type_name as edge_type_name
		FROM catalog_node col
		JOIN catalog_edge e ON (e.source_node_id = col.id OR e.target_node_id = col.id)
		JOIN catalog_node term ON (e.source_node_id = term.id OR e.target_node_id = term.id)
		WHERE col.parent_id = $1::uuid
		  AND term.id != col.id
		  AND term.kind NOT IN ('table', 'view', 'column') -- Exclude structural nodes
	`

	err = s.db.SelectContext(ctx, &response.SemanticFields, semanticQueryv2, driverTableID.String)
	if err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to fetch semantic fields for BO %s: %v", boID, err)
	}

	// 4. Find available semantic terms for the driver table
	// These are terms that have an edge to ANY column of this table
	availableQuery := `
		SELECT DISTINCT
			term.id, term.node_name as node_name, 
			COALESCE(term.properties->>'display_name', term.node_name) as display_name,
			COALESCE(term.properties->>'description', '') as description,
			COALESCE(term.properties->>'data_type', 'string') as data_type,
			COALESCE(term.properties->>'role', 'DIMENSION') as role
		FROM catalog_node col
		JOIN catalog_edge e ON (e.source_node_id = col.id OR e.target_node_id = col.id)
		JOIN catalog_node term ON (e.source_node_id = term.id OR e.target_node_id = term.id)
		WHERE col.parent_id = $1::uuid
		  AND term.id != col.id
		  AND term.node_type_id = '1439f761-606a-44cb-b4f8-7aa6b27a9bf5' -- SEMANTIC_COLUMN node type
	`
	// Note: The node_type_id should ideally be fetched from a constant or lookup

	rows, err := s.db.QueryxContext(ctx, availableQuery, driverTableID.String)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var t models.SemanticTerm
			if err := rows.Scan(&t.ID, &t.NodeName, &t.DisplayName, &t.Description, &t.DataType, &t.Role); err == nil {
				response.AvailableTerms = append(response.AvailableTerms, t)
			}
		}
	}

	return response, nil
}

// ============================================================================
// BUSINESS TERMS & COMPLIANCE
// ============================================================================

// GetBusinessTerm retrieves a business term by ID
func (s *BusinessObjectService) GetBusinessTerm(ctx context.Context, termID string) (*BusinessTerm, error) {
	query := `SELECT * FROM business_terms WHERE id = $1`
	var term BusinessTerm
	if err := s.db.GetContext(ctx, &term, query, termID); err != nil {
		return nil, fmt.Errorf("failed to get business term: %w", err)
	}
	return &term, nil
}

// UpdateBusinessTerm updates a business term and propagates compliance flags to semantic terms
func (s *BusinessObjectService) UpdateBusinessTerm(ctx context.Context, termID string, req UpdateBusinessTermRequest) error {
	// 1. Build dynamic update query for business_terms
	setParts := []string{}
	args := []interface{}{}
	argID := 1

	if req.Name != nil {
		setParts = append(setParts, fmt.Sprintf("name = $%d", argID))
		args = append(args, *req.Name)
		argID++
	}
	if req.Description != nil {
		setParts = append(setParts, fmt.Sprintf("description = $%d", argID))
		args = append(args, *req.Description)
		argID++
	}
	if req.PIIFlag != nil {
		setParts = append(setParts, fmt.Sprintf("pii_flag = $%d", argID))
		args = append(args, *req.PIIFlag)
		argID++
	}
	if req.Residency != nil {
		setParts = append(setParts, fmt.Sprintf("residency = $%d", argID))
		args = append(args, *req.Residency)
		argID++
	}
	if req.SensitivityLevel != nil {
		setParts = append(setParts, fmt.Sprintf("sensitivity_level = $%d", argID))
		args = append(args, *req.SensitivityLevel)
		argID++
	}
	if req.SemanticTermIDs != nil {
		setParts = append(setParts, fmt.Sprintf("semantic_term_ids = $%d", argID))
		args = append(args, pq.Array(*req.SemanticTermIDs))
		argID++
	}

	if len(setParts) == 0 {
		return nil // Nothing to update
	}

	setParts = append(setParts, "updated_at = NOW()")

	// Add ID as last arg
	args = append(args, termID)
	query := fmt.Sprintf("UPDATE business_terms SET %s WHERE id = $%d RETURNING *", strings.Join(setParts, ", "), argID)

	var term BusinessTerm
	if err := s.db.GetContext(ctx, &term, query, args...); err != nil {
		return fmt.Errorf("failed to update business term: %w", err)
	}

	// 2. Propagate compliance to linked semantic terms
	if err := s.propagateComplianceToSemanticTerms(ctx, &term); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to propagate compliance: %v", err)
	}

	// 3. Emit BusinessTermComplianceUpdated event
	btEvent := &events.BusinessTermComplianceUpdatedEvent{
		EventID:         uuid.New().String(),
		EventType:       events.BusinessTermComplianceUpdated,
		BusinessTermID:  term.ID,
		PIIFlag:         term.PIIFlag,
		Residency:       term.Residency,
		Sensitivity:     term.SensitivityLevel,
		SemanticTermIDs: term.SemanticTermIDs,
		Timestamp:       time.Now(),
	}

	if err := s.auditPublisher.PublishBusinessTermComplianceUpdatedEvent(ctx, btEvent); err != nil {
		logging.GetLogger().Sugar().Warnf("Failed to publish business term compliance event: %v", err)
	}

	return nil
}

// AddBusinessTermMappings links semantic terms to a business term
func (s *BusinessObjectService) AddBusinessTermMappings(ctx context.Context, termID string, semanticTermIDs []string) error {
	// 1. Get current term
	term, err := s.GetBusinessTerm(ctx, termID)
	if err != nil {
		return err
	}

	// 2. Merge IDs
	existing := make(map[string]bool)
	for _, id := range term.SemanticTermIDs {
		existing[id] = true
	}

	changed := false
	for _, id := range semanticTermIDs {
		if !existing[id] {
			term.SemanticTermIDs = append(term.SemanticTermIDs, id)
			existing[id] = true
			changed = true
		}
	}

	if !changed {
		return nil
	}

	// 3. Update DB
	query := `UPDATE business_terms SET semantic_term_ids = $1, updated_at = NOW() WHERE id = $2`
	if _, err := s.db.ExecContext(ctx, query, pq.Array(term.SemanticTermIDs), termID); err != nil {
		return fmt.Errorf("failed to update mappings: %w", err)
	}

	// 4. Propagate & Emit
	// We can reuse the logic by creating a shared method or just calling UpdateBusinessTerm?
	// But UpdateBusinessTerm takes a request struct.
	// I'll just call propagateComplianceToSemanticTerms and emit the event manually.

	if err := s.propagateComplianceToSemanticTerms(ctx, term); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to propagate compliance: %v", err)
	}

	// Emit BusinessTermEvent
	// We fetch the updated term to get the full state for the event
	updatedTerm, err := s.GetBusinessTerm(ctx, termID)
	if err == nil {
		btEvent := &events.BusinessTermComplianceUpdatedEvent{
			EventID:         uuid.New().String(),
			EventType:       events.BusinessTermComplianceUpdated,
			BusinessTermID:  updatedTerm.ID,
			PIIFlag:         updatedTerm.PIIFlag,
			Residency:       updatedTerm.Residency,
			Sensitivity:     updatedTerm.SensitivityLevel,
			SemanticTermIDs: updatedTerm.SemanticTermIDs,
			Timestamp:       time.Now(),
		}
		if err := s.auditPublisher.PublishBusinessTermComplianceUpdatedEvent(ctx, btEvent); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to publish business term compliance event: %v", err)
		}
	}

	return nil
}

// RemoveBusinessTermMapping unlinks a semantic term
func (s *BusinessObjectService) RemoveBusinessTermMapping(ctx context.Context, termID string, semanticTermID string) error {
	// 1. Get current term
	term, err := s.GetBusinessTerm(ctx, termID)
	if err != nil {
		return err
	}

	// 2. Remove ID
	newIDs := make([]string, 0, len(term.SemanticTermIDs))
	found := false
	for _, id := range term.SemanticTermIDs {
		if id == semanticTermID {
			found = true
			continue
		}
		newIDs = append(newIDs, id)
	}

	if !found {
		return nil
	}
	term.SemanticTermIDs = newIDs

	// 3. Update DB
	query := `UPDATE business_terms SET semantic_term_ids = $1, updated_at = NOW() WHERE id = $2`
	if _, err := s.db.ExecContext(ctx, query, pq.Array(term.SemanticTermIDs), termID); err != nil {
		return fmt.Errorf("failed to update mappings: %w", err)
	}

	// 4. Clear compliance on the removed semantic term
	// We should reset its inherited properties.
	clearQuery := `
        UPDATE catalog_node 
        SET properties = properties - 'inherited_pii_flag' - 'inherited_residency' - 'inherited_sensitivity',
            updated_at = NOW()
        WHERE id = $1
    `
	if _, err := s.db.ExecContext(ctx, clearQuery, semanticTermID); err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to clear compliance from term %s: %v", semanticTermID, err)
	}

	// 5. Emit BusinessTermEvent
	// We fetch the updated term to get the full state for the event
	updatedTerm, err := s.GetBusinessTerm(ctx, termID)
	if err == nil {
		btEvent := &events.BusinessTermComplianceUpdatedEvent{
			EventID:         uuid.New().String(),
			EventType:       events.BusinessTermComplianceUpdated,
			BusinessTermID:  updatedTerm.ID,
			PIIFlag:         updatedTerm.PIIFlag,
			Residency:       updatedTerm.Residency,
			Sensitivity:     updatedTerm.SensitivityLevel,
			SemanticTermIDs: updatedTerm.SemanticTermIDs,
			Timestamp:       time.Now(),
		}
		if err := s.auditPublisher.PublishBusinessTermComplianceUpdatedEvent(ctx, btEvent); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to publish business term compliance event: %v", err)
		}
	}

	return nil
}

func (s *BusinessObjectService) propagateComplianceToSemanticTerms(ctx context.Context, term *BusinessTerm) error {
	if len(term.SemanticTermIDs) == 0 {
		return nil
	}

	// Iterate and update each semantic term
	// In a real/optimized scenario, we might do a batch update, but we need to read-modify-write JSONB carefully
	// or use jsonb_set. Using loop for clarity and event emission.

	for _, semanticID := range term.SemanticTermIDs {
		// Update catalog_node properties
		// We want to set inherited_pii, inherited_residency, inherited_sensitivity in properties JSON
		updateQuery := `
			UPDATE catalog_node 
			SET properties = jsonb_set(
				jsonb_set(
					jsonb_set(
						properties, 
						'{inherited_pii_flag}', 
						to_jsonb($1::boolean)
					),
					'{inherited_residency}', 
					to_jsonb($2::text)
				),
				'{inherited_sensitivity}', 
				to_jsonb($3::text)
			),
			updated_at = NOW()
			WHERE id = $4
		`

		_, err := s.db.ExecContext(ctx, updateQuery, term.PIIFlag, term.Residency, term.SensitivityLevel, semanticID)
		if err != nil {
			logging.GetLogger().Sugar().Errorf("Failed to propagate compliance to semantic term %s: %v", semanticID, err)
			continue // Continue with others
		}

		// Emit event for Scheduler Intelligence
		event := events.SemanticTermComplianceUpdatedEvent{
			EventID:              uuid.New().String(),
			EventType:            events.SemanticTermComplianceUpdated,
			TenantID:             term.TenantID,
			SemanticTermID:       semanticID,
			BusinessTermID:       term.ID,
			InheritedPIIFlag:     term.PIIFlag,
			InheritedResidency:   term.Residency,
			InheritedSensitivity: term.SensitivityLevel,
			Timestamp:            time.Now(),
		}

		if err := s.auditPublisher.PublishSemanticTermComplianceUpdatedEvent(ctx, event); err != nil {
			logging.GetLogger().Sugar().Warnf("Failed to publish compliance event for term %s: %v", semanticID, err)
		}
	}

	return nil
}

// ListCatalogNodes retrieves catalog nodes with flexible filtering
func (s *BusinessObjectService) ListCatalogNodes(
	ctx context.Context,
	tenantID string,
	datasourceID string,
	nodeType string,
	searchQuery string,
) ([]models.CatalogNode, error) {
	query := `
		SELECT n.id, n.node_name, n.tenant_datasource_id, n.node_type_id as catalog_type_name, 
		       n.description, n.is_active, n.config, n.created_at, n.updated_at, 
		       n.tenant_id, n.properties, n.qualified_path
		FROM catalog_node n
		LEFT JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE n.tenant_id = $1::uuid
	`
	args := []interface{}{tenantID}
	argIdx := 2

	// Filter by datasource if provided
	if datasourceID != "" {
		query += fmt.Sprintf(" AND n.tenant_datasource_id = $%d::uuid", argIdx)
		args = append(args, datasourceID)
		argIdx++
	}

	// Filter by node type (joining with catalog_node_type or checking cached type map would be better,
	// but here we might just filter by catalog_type_name on the JOINed table)
	if nodeType != "" {
		// We assume nodeType param matches catalog_type_name in catalog_node_type
		// OR we can check if it matches the ID directly.
		// Let's try matching the name for friendliness.
		query += fmt.Sprintf(" AND (nt.catalog_type_name = $%d OR n.node_type_id = $%d)", argIdx, argIdx)
		args = append(args, nodeType)
		argIdx++
	}

	// Search query
	if searchQuery != "" {
		searchPattern := "%" + searchQuery + "%"
		query += fmt.Sprintf(" AND (n.node_name ILIKE $%d OR n.qualified_path ILIKE $%d)", argIdx, argIdx)
		args = append(args, searchPattern)
		argIdx++
	}

	query += " ORDER BY n.node_name LIMIT 100"

	var nodes []models.CatalogNode
	// We need to map the result manually or ensure struct tags match.
	// CatalogNode struct uses `db` tags.
	// Note: `catalog_type_name` in struct is expected to be string name, but DB has `node_type_id`.
	// The query aliases `node_type_id` as `catalog_type_name` which might be a UUID string, not the human readable name.
	// For the wizard, we mainly need ID and Name.

	err := s.db.SelectContext(ctx, &nodes, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list catalog nodes: %w", err)
	}

	return nodes, nil
}

// GetSemanticTermsByTable returns semantic terms linked to columns from a specific driver table via catalog edges
func (s *BusinessObjectService) GetSemanticTermsByTable(
	ctx context.Context,
	tableID string,
	datasourceID string,
) ([]models.CatalogNode, error) {
	query := `
		WITH table_columns AS (
			SELECT id
			FROM catalog_node
			WHERE parent_id = $1::uuid
			  AND tenant_datasource_id = $2::uuid
		)
		SELECT DISTINCT
			st.id,
			st.node_name,
			st.qualified_path,
			st.node_type_id,
			st.tenant_datasource_id,
			COALESCE(st.node_type, '') AS catalog_type,
			st.description,
			st.properties,
			COALESCE(st.created_at, NOW()) AS created_at,
			COALESCE(st.updated_at, NOW()) AS updated_at,
			st.tenant_id
		FROM catalog_node st
		INNER JOIN catalog_edge e ON e.target_node_id = st.id
		INNER JOIN table_columns tc ON e.source_node_id = tc.id
		WHERE st.node_type_id = '820b942a-9c9e-4abc-acdc-84616db33098' -- semantic term type
		  AND e.edge_type_name = 'semantic_mapping'
		ORDER BY st.node_name
	`

	var terms []models.CatalogNode
	logging.GetLogger().Sugar().Infof("DEBUG: GetSemanticTermsByTable - tableID: %s, datasourceID: %s", tableID, datasourceID)
	err := s.db.SelectContext(ctx, &terms, query, tableID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get semantic terms by table: %w", err)
	}

	return terms, nil
}
