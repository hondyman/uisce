package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/events"
	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/models"
	"github.com/jmoiron/sqlx"
)

// BusinessObjectService handles business object operations with real database queries
type BusinessObjectService struct {
	db     *sqlx.DB
	hasura HasuraClient
	rules  AccessRuleRepository
}

// NewBusinessObjectService creates a new BusinessObjectService with database connection
func NewBusinessObjectService(db interface{}) *BusinessObjectService {
	if sqlxDB, ok := db.(*sqlx.DB); ok {
		return &BusinessObjectService{db: sqlxDB, rules: NewPgAccessRuleRepository(sqlxDB)}
	}
	// Fallback for other db types
	if sqlDB, ok := db.(*sql.DB); ok {
		sqlxDB := sqlx.NewDb(sqlDB, "postgres")
		return &BusinessObjectService{db: sqlxDB, rules: NewPgAccessRuleRepository(sqlxDB)}
	}
	return &BusinessObjectService{}
}

// NewBusinessObjectServiceWithHasura creates a new service with Hasura support
func NewBusinessObjectServiceWithHasura(db interface{}, hasura HasuraClient) *BusinessObjectService {
	svc := NewBusinessObjectService(db)
	svc.hasura = hasura
	return svc
}

// CreateBusinessObject creates a new business object definition
func (s *BusinessObjectService) CreateBusinessObject(ctx context.Context, tenantID string, req models.CreateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	bo := &models.BusinessObjectDefinition{
		ID:            uuid.New().String(),
		TenantID:      tenantID,
		Name:          req.Name,
		Description:   req.Description,
		EnableHistory: req.EnableHistory,
		HistoryMode:   models.HistoryMode(req.HistoryMode),
		CreatedBy:     userID,
	}

	if req.DatasourceID != "" {
		bo.DatasourceID = sql.NullString{String: req.DatasourceID, Valid: true}
	}

	if req.ParentID != "" {
		bo.ParentID = sql.NullString{String: req.ParentID, Valid: true}
		// Ensure subtypes inherit datasource from parent when caller omitted it
		if !bo.DatasourceID.Valid {
			parent, err := s.GetBusinessObject(ctx, tenantID, req.ParentID)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve parent datasource: %w", err)
			}
			if parent.DatasourceID.Valid && parent.DatasourceID.String != "" {
				bo.DatasourceID = sql.NullString{String: parent.DatasourceID.String, Valid: true}
			} else {
				return nil, fmt.Errorf("missing datasource for subtype create (parent has none)")
			}
		}
	}

	// Capture config from request
	if req.Config != nil {
		// Enforce: fields are stored in bo_fields, not in config JSON
		delete(req.Config, "fields")

		configJSON, err := json.Marshal(req.Config)
		if err == nil {
			bo.Config = configJSON
		}
	}

	// Validate and capture driver table (must belong to datasource in context when provided)
	if req.DriverTableID != "" {
		// Require datasource in context for driver table binding
		if req.DatasourceID == "" {
			return nil, fmt.Errorf("driver_table_id supplied but datasource_id is missing in request")
		}
		var exists bool
		err := s.db.GetContext(ctx, &exists, `
			SELECT EXISTS(
				SELECT 1 FROM catalog_node WHERE id = $1::uuid AND tenant_datasource_id = $2::uuid
			)
		`, req.DriverTableID, req.DatasourceID)
		if err != nil || !exists {
			return nil, fmt.Errorf("driver table not found in the given datasource")
		}
		bo.DriverTableID = sql.NullString{String: req.DriverTableID, Valid: true}
	}
	if req.DriverTableName != "" {
		// Validate by qualified_path when name provided
		if req.DatasourceID == "" {
			return nil, fmt.Errorf("driver_table_name supplied but datasource_id is missing in request")
		}
		var exists bool
		err := s.db.GetContext(ctx, &exists, `
			SELECT EXISTS(
				SELECT 1 FROM catalog_node WHERE qualified_path = $1 AND tenant_datasource_id = $2::uuid LIMIT 1
			)
		`, req.DriverTableName, req.DatasourceID)
		if err != nil || !exists {
			return nil, fmt.Errorf("driver table (by name) not found in the given datasource")
		}
		bo.DriverTableName = req.DriverTableName
	}

	if s.hasura != nil {
		result, err := s.createBusinessObjectWithHasura(ctx, bo)
		if err == nil {
			return result, nil
		}
		fmt.Printf("Hasura mutation failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback
	// Prepare datasource param and cast to uuid to avoid driver type mismatches
	var datasourceParam interface{}
	if bo.DatasourceID.Valid && bo.DatasourceID.String != "" {
		datasourceParam = bo.DatasourceID.String
	} else {
		datasourceParam = nil
	}

	// Log the resolved scope to catch missing datasource propagation early
	logging.GetLogger().Sugar().Warnf("[SERVICE] CreateBusinessObject scope: tenant=%s datasource=%v parent_id=%v name=%s", bo.TenantID, datasourceParam, bo.ParentID, bo.Name)

	query := `
		INSERT INTO business_objects (
			id, tenant_id, datasource_id, parent_id, name, description, config, enable_history, history_mode, created_by,
			driver_table_id, driver_table_name,
			created_at, last_modified_at
		)
		VALUES ($1, $2, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11::uuid, $12, NOW(), NOW())
		RETURNING created_at, last_modified_at
	`

	// Prepare driver table params (nullable)
	var driverTableIDParam interface{} = nil
	var driverTableNameParam interface{} = nil
	if bo.DriverTableID.Valid {
		driverTableIDParam = bo.DriverTableID.String
	}
	if bo.DriverTableName != "" {
		driverTableNameParam = bo.DriverTableName
	}

	err := s.db.QueryRowContext(ctx, query, bo.ID, bo.TenantID, datasourceParam, bo.ParentID, bo.Name, bo.Description, bo.Config, bo.EnableHistory, bo.HistoryMode, bo.CreatedBy, driverTableIDParam, driverTableNameParam).
		Scan(&bo.CreatedAt, &bo.LastModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create business object: %w", err)
	}

	return bo, nil
}

func (s *BusinessObjectService) createBusinessObjectWithHasura(ctx context.Context, bo *models.BusinessObjectDefinition) (*models.BusinessObjectDefinition, error) {
	mutation := `
		mutation CreateBusinessObject($object: business_objects_insert_input!) {
			insert_business_objects_one(object: $object) {
				id
				created_at
				last_modified_at
			}
		}
	`

	// Assuming we pass config as jsonb fields if Hasura expects 'fields'.
	// If Hasura expects 'config', we should use that.
	// For now, let's assume 'config' maps to 'config' in Hasura too if we updated DB.
	// If Hasura schema is old, this might break.
	// But aligning with Go models is priority.

	variables := map[string]interface{}{
		"object": map[string]interface{}{
			"id":             bo.ID,
			"tenant_id":      bo.TenantID,
			"name":           bo.Name,
			"description":    bo.Description,
			"config":         string(bo.Config),
			"enable_history": bo.EnableHistory,
			"history_mode":   bo.HistoryMode,
			"created_by":     bo.CreatedBy,
		},
	}

	resp, err := s.hasura.Mutate(mutation, variables)
	if err != nil {
		return nil, err
	}

	if result, ok := resp["insert_business_objects_one"].(map[string]interface{}); ok {
		if createdAt, ok := result["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				bo.CreatedAt = t
			}
		}
		if updatedAt, ok := result["last_modified_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				bo.LastModifiedAt = t
			}
		}
	}

	return bo, nil
}

// ListBusinessObjects lists all business objects for a tenant
func (s *BusinessObjectService) ListBusinessObjects(ctx context.Context, tenantID, datasourceID string) ([]*models.BusinessObjectDefinition, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	if s.hasura != nil {
		objects, err := s.listBusinessObjectsWithHasura(ctx, tenantID, datasourceID)
		if err == nil {
			return objects, nil
		}
		fmt.Printf("Hasura query failed, falling back to SQL: %v\n", err)
	}

	// SQL fallback - include parent_id to allow frontend to filter subtypes
	query := `
		SELECT id, tenant_id, name, display_name, parent_id, description, config, icon,
			   created_at, last_modified_at, is_core, is_active, enable_history, history_mode
		FROM business_objects
		WHERE (tenant_id = $1 OR is_core = true)
	`
	args := []interface{}{tenantID}

	if datasourceID != "" {
		query += " AND (datasource_id = $2::uuid OR datasource_id IS NULL)"
		args = append(args, datasourceID)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list business objects: %w", err)
	}
	defer rows.Close()

	var objects []*models.BusinessObjectDefinition
	for rows.Next() {
		bo := &models.BusinessObjectDefinition{}
		var displayName, description, icon sql.NullString
		var parentID sql.NullString
		var config []byte
		var isCore, isActive sql.NullBool
		var historyMode sql.NullString

		err := rows.Scan(&bo.ID, &bo.TenantID, &bo.Name, &displayName, &parentID, &description,
			&config, &icon, &bo.CreatedAt, &bo.LastModifiedAt, &isCore, &isActive, &bo.EnableHistory, &historyMode)
		if err != nil {
			return nil, fmt.Errorf("failed to scan business object: %w", err)
		}

		if displayName.Valid {
			bo.DisplayName = displayName.String // Use DisplayName from model
		}
		// Set parent ID so clients can exclude subtypes
		if parentID.Valid {
			bo.ParentID = parentID
		}
		if description.Valid {
			bo.Description = description.String
		}
		if icon.Valid {
			bo.Icon = icon.String
		}

		if len(config) > 0 {
			bo.Config = config
		}

		// Use name as key if not set
		bo.Key = bo.Name
		if isCore.Valid {
			bo.IsCore = isCore.Bool
		}
		if isActive.Valid {
			bo.IsActive = isActive.Bool
		}
		if historyMode.Valid {
			bo.HistoryMode = models.HistoryMode(historyMode.String)
		}

		objects = append(objects, bo)
	}

	return objects, nil
}

// GetBusinessObject retrieves a single business object by key
func (s *BusinessObjectService) GetBusinessObject(ctx context.Context, tenantID, key string) (*models.BusinessObjectDefinition, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	query := `
		SELECT id, tenant_id, datasource_id, key, name, display_name, parent_id, technical_name, 
               description, config, icon, is_core, is_active, category, enable_history, history_mode,
		       created_at, last_modified_at, entity_key, catalog_node_id
		FROM business_objects
		WHERE tenant_id = $1 AND (id::text = $2 OR key = $2 OR technical_name = $2 OR name = $2 OR entity_key = $2)
	`

	bo := &models.BusinessObjectDefinition{}
	var parentID, technicalName, datasourceID, businessName, entityKey, catalogNodeID, boKey sql.NullString
	var description, icon, category sql.NullString
	var isCore, isActive sql.NullBool
	var config []byte

	err := s.db.QueryRowContext(ctx, query, tenantID, key).
		Scan(&bo.ID, &bo.TenantID, &datasourceID, &boKey, &bo.Name, &businessName,
			&parentID, &technicalName, &description, &config, &icon, &isCore, &isActive, &category, &bo.EnableHistory, &bo.HistoryMode,
			&bo.CreatedAt, &bo.LastModifiedAt, &entityKey, &catalogNodeID)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("business object not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get business object: %w", err)
	}

	if businessName.Valid {
		bo.DisplayName = businessName.String
	}
	if boKey.Valid {
		bo.Key = boKey.String
	}
	if description.Valid {
		bo.Description = description.String
	}
	if icon.Valid {
		bo.Icon = icon.String
	}
	if category.Valid {
		bo.Category = category.String
	}
	if entityKey.Valid {
		bo.Key = entityKey.String
	}
	if isCore.Valid {
		bo.IsCore = isCore.Bool
	}
	if isActive.Valid {
		bo.IsActive = isActive.Bool
	}
	if parentID.Valid {
		bo.ParentID = sql.NullString{String: parentID.String, Valid: true}
	}
	if len(config) > 0 {
		bo.Config = config
		logging.GetLogger().Sugar().Infof("[BO-SERVICE] config len=%d", len(config))

		// Extract driver_table_id from config if present
		var cfg map[string]interface{}
		if err := json.Unmarshal(config, &cfg); err == nil {
			logging.GetLogger().Sugar().Infof("[BO-SERVICE] config unmarshaled: %v", cfg)
			if v, ok := cfg["driver_table_id"].(string); ok && v != "" {
				bo.DriverTableID = sql.NullString{String: v, Valid: true}
				logging.GetLogger().Sugar().Infof("[BO-SERVICE] Extracted driver_table_id: %s", v)
			} else {
				logging.GetLogger().Sugar().Infof("[BO-SERVICE] driver_table_id not found: ok=%v v=%v", ok, cfg["driver_table_id"])
			}
		} else {
			logging.GetLogger().Sugar().Infof("[BO-SERVICE] Failed to unmarshal config: %v", err)
		}
	} else {
		logging.GetLogger().Sugar().Infof("[BO-SERVICE] config is empty")
	}

	// Fallback: if driver_table_id did not get extracted (e.g., type mismatch), try again defensively
	if !bo.DriverTableID.Valid && len(config) > 0 {
		var cfg map[string]interface{}
		if err := json.Unmarshal(config, &cfg); err == nil {
			if v, ok := cfg["driver_table_id"]; ok {
				switch t := v.(type) {
				case string:
					if t != "" {
						bo.DriverTableID = sql.NullString{String: t, Valid: true}
						logging.GetLogger().Sugar().Infof("[BO-SERVICE] Fallback extracted driver_table_id (string): %s", t)
					}
				case []byte:
					if len(t) > 0 {
						bo.DriverTableID = sql.NullString{String: string(t), Valid: true}
						logging.GetLogger().Sugar().Infof("[BO-SERVICE] Fallback extracted driver_table_id (bytes): %s", string(t))
					}
				default:
					logging.GetLogger().Sugar().Infof("[BO-SERVICE] Fallback driver_table_id not string/bytes: %v", v)
				}
			}
		}
	}

	if datasourceID.Valid {
		bo.DatasourceID = datasourceID
	}

	// 3. Fetch bo_fields (semantic terms linked to this BO) from bo_fields
	fieldQuery := `
		SELECT id, key, name, display_label AS display_name, field_type AS type, is_core, display_order AS sequence, created_at, created_by, semantic_term_id, role
		FROM bo_fields

		WHERE business_object_id = $1
		ORDER BY display_order, created_at
	`

	logging.GetLogger().Sugar().Debugf("Fetching bo_fields for BO: boID=%s", bo.ID)
	rows, err := s.db.QueryContext(ctx, fieldQuery, bo.ID)
	if err != nil {
		// Log but don't fail - fields are optional
		logging.GetLogger().Sugar().Warnf("Failed to fetch bo_fields for BO %s (tenant %s): %v", bo.ID, tenantID, err)
	} else {
		defer rows.Close()
		var fields []models.FieldDefinition
		for rows.Next() {
			field := models.FieldDefinition{}
			var displayName, createdBy, semanticTermID, role sql.NullString
			err := rows.Scan(&field.ID, &field.Key, &field.Name, &displayName, &field.Type, &field.IsCore, &field.Sequence, &field.CreatedAt, &createdBy, &semanticTermID, &role)
			if err != nil {
				logging.GetLogger().Sugar().Warnf("Failed to scan bo_field row: %v", err)
				continue
			}
			if displayName.Valid {
				field.DisplayName = displayName.String
			} else {
				field.DisplayName = field.Name
			}
			if createdBy.Valid {
				field.CreatedBy = createdBy.String
			}
			if semanticTermID.Valid {
				field.SemanticTermID = semanticTermID.String
			}
			if role.Valid {
				field.Role = models.FieldRole(role.String)
			}
			fields = append(fields, field)
		}
		logging.GetLogger().Sugar().Debugf("Fetched %d bo_fields for BO %s", len(fields), bo.ID)
		if len(fields) > 0 {
			// All wizard-created fields are custom (not core)
			bo.CustomFields = fields
		}
	}

	// Fallback: if no fields returned, try without tenant filter to confirm data exists (helps diagnose tenant mismatch)
	if len(bo.CustomFields) == 0 {
		altQuery := `
			SELECT id, key, name, display_label AS display_name, field_type AS type, is_core, display_order AS sequence, created_at, created_by
			FROM bo_fields
			WHERE business_object_id = $1
			ORDER BY display_order, created_at
		`
		rows, err := s.db.QueryContext(ctx, altQuery, bo.ID)
		if err != nil {
			logging.GetLogger().Sugar().Warnf("Fallback fetch bo_fields failed for BO %s: %v", bo.ID, err)
		} else {
			defer rows.Close()
			var fields []models.FieldDefinition
			for rows.Next() {
				field := models.FieldDefinition{}
				var displayName, createdBy sql.NullString
				err := rows.Scan(&field.ID, &field.Key, &field.Name, &displayName, &field.Type, &field.IsCore, &field.Sequence, &field.CreatedAt, &createdBy)
				if err != nil {
					logging.GetLogger().Sugar().Warnf("Fallback failed to scan bo_field row: %v", err)
					continue
				}
				if displayName.Valid {
					field.DisplayName = displayName.String
				} else {
					field.DisplayName = field.Name
				}
				if createdBy.Valid {
					field.CreatedBy = createdBy.String
				}
				fields = append(fields, field)
			}
			if len(fields) > 0 {
				logging.GetLogger().Sugar().Infof("Fallback fetched %d bo_fields without tenant filter for BO %s", len(fields), bo.ID)
				bo.CustomFields = fields
			}
		}
	}

	return bo, nil
}

// UpdateBusinessObject updates an existing business object
func (s *BusinessObjectService) UpdateBusinessObject(ctx context.Context, tenantID, key string, req models.UpdateBusinessObjectRequest, userID string) (*models.BusinessObjectDefinition, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	// Marshal config if present
	var configJSON []byte
	var err error
	if req.Config != nil {
		configJSON, err = json.Marshal(req.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal config: %w", err)
		}
	}

	query := `
		UPDATE business_objects
		SET display_name = $1, description = $2, icon = $3, category = $4, is_active = COALESCE($5, is_active), 
		    enable_history = COALESCE($6, enable_history), history_mode = COALESCE($7, history_mode), 
		    config = CASE WHEN $10::jsonb IS NOT NULL THEN $10::jsonb ELSE config END,
		    last_modified_at = NOW()
		WHERE tenant_id = $8 AND (id::text = $9 OR technical_name = $9)
		RETURNING id, tenant_id, name, display_name, description, config, icon, category, is_active, enable_history, history_mode, created_by, created_at, last_modified_at
	`

	bo := &models.BusinessObjectDefinition{}
	var config []byte

	// Use sql.Null types or just pass nil for configJSON if empty?
	// Postgres driver handles []byte as bytea usually, but we cast to jsonb.
	// We need to be careful with nil vs empty slice.
	// If configJSON is nil, it passes nil.

	err = s.db.QueryRowContext(ctx, query,
		req.DisplayName,
		req.Description,
		req.Icon,
		req.Category,
		req.IsActive,
		req.EnableHistory,
		req.HistoryMode,
		tenantID,
		key,
		configJSON).
		Scan(&bo.ID, &bo.TenantID, &bo.Name, &bo.DisplayName, &bo.Description, &config,
			&bo.Icon, &bo.Category, &bo.IsActive, &bo.EnableHistory, &bo.HistoryMode,
			&bo.CreatedBy, &bo.CreatedAt, &bo.LastModifiedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("business object not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update business object: %w", err)
	}

	if len(config) > 0 {
		bo.Config = config
	}

	// Logic for child fields update (bo_fields)
	// This logic extracted from Config['fields']
	if req.Config != nil {
		if fieldsRaw, ok := req.Config["fields"]; ok {
			// Marshal and unmarshal into structured slice to iterate
			fieldsBytes, _ := json.Marshal(fieldsRaw)
			var newFields []map[string]interface{}
			if err := json.Unmarshal(fieldsBytes, &newFields); err == nil {
				// Use transaction to replace bo_fields for this BO
				tx, err := s.db.Beginx()
				if err == nil {
					defer func() { _ = tx.Rollback() }()
					// delete existing fields for BO
					if _, err := tx.ExecContext(ctx, `DELETE FROM bo_fields WHERE business_object_id = $1::uuid`, bo.ID); err != nil {
						fmt.Printf("warning: failed to delete bo_fields for bo_id=%s: %v\n", bo.ID, err)
					}
					// insert new fields
					insertQ := `INSERT INTO bo_fields (id, business_object_id, field_name, field_type, display_label, display_order, help_text, is_required, semantic_term_id, role) VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8, $9::uuid, $10)`

					var selectedTermIDs []string

					for _, f := range newFields {
						id := uuid.New()
						fieldName := ""
						if s, ok := f["key"].(string); ok {
							fieldName = s
						} else if s, ok := f["technicalName"].(string); ok {
							fieldName = s
						}
						displayLabel := ""
						if s, ok := f["name"].(string); ok {
							displayLabel = s
						}
						fieldType := "string"
						if s, ok := f["type"].(string); ok {
							fieldType = s
						}

						// Extract semanticTermID
						var semanticTermID *string
						if s, ok := f["semanticTermId"].(string); ok && s != "" {
							semanticTermID = &s
						} else if s, ok := f["semantic_term_id"].(string); ok && s != "" {
							semanticTermID = &s
						}

						// Extract Role
						role := ""
						if s, ok := f["role"].(string); ok {
							role = s
						}

						// If implementation details: fieldName (key) is the semantic term ID for wizard-created fields
						// BUT if we have explicit semanticTermID, use that for tracking.
						if fieldType == "semantic_term" {
							if semanticTermID != nil {
								selectedTermIDs = append(selectedTermIDs, *semanticTermID)
							} else {
								// Fallback to key if semanticTermID not explicitly in payload (legacy)
								selectedTermIDs = append(selectedTermIDs, fieldName)
							}
						}

						seq := 0
						if n, ok := f["sequence"].(float64); ok {
							seq = int(n)
						}
						help := ""
						if s, ok := f["description"].(string); ok {
							help = s
						}
						if _, err := tx.ExecContext(ctx, insertQ, id, bo.ID, fieldName, fieldType, displayLabel, seq, help, false, semanticTermID, role); err != nil {
							fmt.Printf("warning: failed to insert bo_field for bo_id=%s key=%s: %v\n", bo.ID, fieldName, err)
						}
					}

					// Publish CatalogSync event to update edges
					// We need to fetch current BO details to fully populate the event as consumers expect
					// BO struct already has most info, but let's be safe with Config

					// Re-extract driver table id
					var driverTableID string
					if bo.DriverTableID.Valid {
						driverTableID = bo.DriverTableID.String
					}

					catalogEvent := map[string]interface{}{
						"bo_id":           bo.ID,
						"bo_key":          bo.Key,
						"name":            bo.Name,
						"display_name":    bo.DisplayName,
						"description":     bo.Description,
						"driver_table_id": driverTableID,
						"selected_terms":  selectedTermIDs,
						"tenant_id":       bo.TenantID,
						"datasource_id":   "",
					}
					if bo.DatasourceID.Valid {
						catalogEvent["datasource_id"] = bo.DatasourceID.String
					}

					if err := events.PublishEvent(ctx, tx, "BusinessObject.CatalogSync", catalogEvent); err != nil {
						fmt.Printf("warning: failed to publish CatalogSync event for bo_id=%s: %v\n", bo.ID, err)
					}

					if err := tx.Commit(); err != nil {
						fmt.Printf("warning: failed to commit bo_fields transaction for bo_id=%s: %v\n", bo.ID, err)
					}
				}
			}
			// delete(req.Config, "fields") - MOVED OUTSIDE to be unconditional
		}
	}

	// Force remove fields from config to avoid duplication in JSON column
	// This must happen before re-marshaling configJSON below
	if req.Config != nil {
		delete(req.Config, "fields")
	}

	// Marshaling logic is already below...

	return bo, nil
}

// DeleteBusinessObject deletes a business object
func (s *BusinessObjectService) DeleteBusinessObject(ctx context.Context, tenantID, key, userID string) error {
	if s.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	query := `
		DELETE FROM business_objects
		WHERE id::text = $1 OR technical_name = $1
	`

	result, err := s.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete business object: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("business object not found")
	}

	return nil
}

// CloneBusinessObject creates a copy of an existing business object
func (s *BusinessObjectService) CloneBusinessObject(ctx context.Context, tenantID string, req models.CloneBORequest, userID string) (*models.BusinessObjectDefinition, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	// First, get the source business object
	source, err := s.GetBusinessObject(ctx, tenantID, req.SourceBOKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get source business object: %w", err)
	}

	// Create new business object with cloned data
	newBO := &models.BusinessObjectDefinition{
		ID:          uuid.New().String(),
		TenantID:    source.TenantID,
		Name:        req.NewName,
		Description: source.Description,
		Config:      source.Config,
		CreatedBy:   userID,
	}

	query := `
		INSERT INTO business_objects (id, tenant_id, name, description, config, created_by, created_at, last_modified_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, last_modified_at
	`

	err = s.db.QueryRowContext(ctx, query, newBO.ID, newBO.TenantID, newBO.Name, newBO.Description, newBO.Config, newBO.CreatedBy).
		Scan(&newBO.CreatedAt, &newBO.LastModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to clone business object: %w", err)
	}

	return newBO, nil
}

// CreateInstance creates a new instance of a business object
func (s *BusinessObjectService) CreateInstance(ctx context.Context, tenantID, userID string, instance *models.BusinessObjectInstance) (*models.BusinessObjectInstance, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	// Resolve BO ID if caller provided only a key
	boID := instance.BusinessObjectID
	if boID == "" && instance.BusinessObjectKey != "" {
		bo, err := s.GetBusinessObject(ctx, tenantID, instance.BusinessObjectKey)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve business object: %w", err)
		}
		boID = bo.ID
		instance.BusinessObjectID = boID
	}

	if boID == "" {
		return nil, fmt.Errorf("business object id is required for instance creation")
	}

	if _, err := s.requireAccess(ctx, tenantID, boID, AccessLevelWrite); err != nil {
		return nil, fmt.Errorf("access denied for business object: %w", err)
	}

	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}

	coreJSON, err := json.Marshal(instance.CoreFieldValues)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal core attributes: %w", err)
	}

	customJSON, err := json.Marshal(instance.CustomFieldValues)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom attributes: %w", err)
	}

	query := `
		INSERT INTO business_object_instances 
		(id, tenant_id, business_object_id, core_attributes, custom_attributes, created_by, created_at, last_modified_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING created_at, last_modified_at
	`

	err = s.db.QueryRowContext(ctx, query, instance.ID, tenantID, instance.BusinessObjectID, coreJSON, customJSON, userID).
		Scan(&instance.CreatedAt, &instance.LastModifiedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}

	return instance, nil
}

// ListInstances lists instances of a business object with pagination
func (s *BusinessObjectService) ListInstances(ctx context.Context, tenantID, boKey string, offset, limit int) ([]*models.BusinessObjectInstance, int, error) {
	if s.db == nil {
		return nil, 0, fmt.Errorf("database connection not initialized")
	}

	// First get the business object ID
	bo, err := s.GetBusinessObject(ctx, tenantID, boKey)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get business object: %w", err)
	}

	decision, err := s.requireAccess(ctx, tenantID, bo.ID, AccessLevelRead)
	if err != nil {
		return nil, 0, fmt.Errorf("access denied for business object: %w", err)
	}

	// Count total instances
	var total int
	whereParts := []string{"tenant_id = $1", "business_object_id = $2"}
	if decision.RowPredicate != "" {
		whereParts = append(whereParts, "("+decision.RowPredicate+")")
	}
	countQuery := "SELECT COUNT(*) FROM business_object_instances WHERE " + strings.Join(whereParts, " AND ")
	err = s.db.QueryRowContext(ctx, countQuery, tenantID, bo.ID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count instances: %w", err)
	}

	// Get paginated instances
	query := "SELECT id, business_object_id, core_attributes, custom_attributes, created_by, created_at, last_modified_at " +
		"FROM business_object_instances WHERE " + strings.Join(whereParts, " AND ") + " ORDER BY created_at DESC LIMIT $3 OFFSET $4"

	rows, err := s.db.QueryContext(ctx, query, tenantID, bo.ID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list instances: %w", err)
	}
	defer rows.Close()

	var instances []*models.BusinessObjectInstance
	for rows.Next() {
		inst := &models.BusinessObjectInstance{}
		var coreJSON, customJSON []byte

		err := rows.Scan(&inst.ID, &inst.BusinessObjectID, &coreJSON, &customJSON,
			&inst.CreatedBy, &inst.CreatedAt, &inst.LastModifiedAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan instance: %w", err)
		}

		if len(coreJSON) > 0 {
			json.Unmarshal(coreJSON, &inst.CoreFieldValues)
		}
		if len(customJSON) > 0 {
			json.Unmarshal(customJSON, &inst.CustomFieldValues)
		}

		applyColumnMasksToInstance(inst, decision.ColumnMasks)

		instances = append(instances, inst)
	}

	return instances, total, nil
}

// GetInstance retrieves a single business object instance
func (s *BusinessObjectService) GetInstance(ctx context.Context, tenantID, instanceID string) (*models.BusinessObjectInstance, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	boID, err := s.getInstanceBusinessObjectID(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	decision, err := s.requireAccess(ctx, tenantID, boID, AccessLevelRead)
	if err != nil {
		return nil, fmt.Errorf("access denied for business object: %w", err)
	}

	whereParts := []string{"tenant_id = $1", "id = $2"}
	if decision.RowPredicate != "" {
		whereParts = append(whereParts, "("+decision.RowPredicate+")")
	}

	query := "SELECT id, business_object_id, core_attributes, custom_attributes, created_by, created_at, last_modified_at " +
		"FROM business_object_instances WHERE " + strings.Join(whereParts, " AND ")

	inst := &models.BusinessObjectInstance{}
	var coreJSON, customJSON []byte

	err = s.db.QueryRowContext(ctx, query, tenantID, instanceID).
		Scan(&inst.ID, &inst.BusinessObjectID, &coreJSON, &customJSON,
			&inst.CreatedBy, &inst.CreatedAt, &inst.LastModifiedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instance not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if len(coreJSON) > 0 {
		json.Unmarshal(coreJSON, &inst.CoreFieldValues)
	}
	if len(customJSON) > 0 {
		json.Unmarshal(customJSON, &inst.CustomFieldValues)
	}

	applyColumnMasksToInstance(inst, decision.ColumnMasks)

	return inst, nil
}

// GetInstanceForValidation retrieves a business object instance and formats it for validation
// It merges core and custom field values into a single flat JSON object
func (s *BusinessObjectService) GetInstanceForValidation(ctx context.Context, tenantID, instanceID string) (map[string]interface{}, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	boID, err := s.getInstanceBusinessObjectID(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	decision, err := s.requireAccess(ctx, tenantID, boID, AccessLevelRead)
	if err != nil {
		return nil, fmt.Errorf("access denied for business object: %w", err)
	}

	whereParts := []string{"tenant_id = $1", "id = $2"}
	if decision.RowPredicate != "" {
		whereParts = append(whereParts, "("+decision.RowPredicate+")")
	}

	query := "SELECT id, business_object_id, core_attributes, custom_attributes, created_by, created_at, last_modified_at " +
		"FROM business_object_instances WHERE " + strings.Join(whereParts, " AND ")

	inst := &models.BusinessObjectInstance{}
	var coreJSON, customJSON []byte

	err = s.db.QueryRowContext(ctx, query, tenantID, instanceID).
		Scan(&inst.ID, &inst.BusinessObjectID, &coreJSON, &customJSON,
			&inst.CreatedBy, &inst.CreatedAt, &inst.LastModifiedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instance not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Create merged flat JSON object for validation
	result := make(map[string]interface{})

	// Add core fields
	if len(coreJSON) > 0 {
		var coreFields map[string]interface{}
		if err := json.Unmarshal(coreJSON, &coreFields); err == nil {
			for k, v := range coreFields {
				result[k] = v
			}
		}
	}

	// Add custom fields (will override core if same key exists)
	if len(customJSON) > 0 {
		var customFields map[string]interface{}
		if err := json.Unmarshal(customJSON, &customFields); err == nil {
			for k, v := range customFields {
				result[k] = v
			}
		}
	}

	// Add metadata fields that validation rules might need
	result["_instanceId"] = inst.ID
	result["_businessObjectId"] = inst.BusinessObjectID
	result["_createdAt"] = inst.CreatedAt
	result["_lastModifiedAt"] = inst.LastModifiedAt

	for term, mask := range decision.ColumnMasks {
		switch mask {
		case "HIDE":
			delete(result, term)
		case "MASK":
			if _, ok := result[term]; ok {
				result[term] = "[MASKED]"
			}
		}
	}

	return result, nil
}

// UpdateInstance updates an existing business object instance
func (s *BusinessObjectService) UpdateInstance(ctx context.Context, tenantID, instanceID, userID string, core, custom map[string]interface{}) (*models.BusinessObjectInstance, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	boID, err := s.getInstanceBusinessObjectID(ctx, tenantID, instanceID)
	if err != nil {
		return nil, err
	}

	decision, err := s.requireAccess(ctx, tenantID, boID, AccessLevelWrite)
	if err != nil {
		return nil, fmt.Errorf("access denied for business object: %w", err)
	}

	coreJSON, err := json.Marshal(core)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal core attributes: %w", err)
	}

	customJSON, err := json.Marshal(custom)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal custom attributes: %w", err)
	}

	whereParts := []string{"tenant_id = $3", "id = $4"}
	if decision.RowPredicate != "" {
		whereParts = append(whereParts, "("+decision.RowPredicate+")")
	}

	query := "UPDATE business_object_instances SET core_attributes = $1, custom_attributes = $2, last_modified_at = NOW() " +
		"WHERE " + strings.Join(whereParts, " AND ") + " RETURNING id, business_object_id, core_attributes, custom_attributes, created_by, created_at, last_modified_at"

	inst := &models.BusinessObjectInstance{}
	var returnedCoreJSON, returnedCustomJSON []byte

	err = s.db.QueryRowContext(ctx, query, coreJSON, customJSON, tenantID, instanceID).
		Scan(&inst.ID, &inst.BusinessObjectID, &returnedCoreJSON, &returnedCustomJSON,
			&inst.CreatedBy, &inst.CreatedAt, &inst.LastModifiedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("instance not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to update instance: %w", err)
	}

	if len(returnedCoreJSON) > 0 {
		json.Unmarshal(returnedCoreJSON, &inst.CoreFieldValues)
	}
	if len(returnedCustomJSON) > 0 {
		json.Unmarshal(returnedCustomJSON, &inst.CustomFieldValues)
	}

	applyColumnMasksToInstance(inst, decision.ColumnMasks)

	return inst, nil
}

// DeleteInstance deletes a business object instance
func (s *BusinessObjectService) DeleteInstance(ctx context.Context, tenantID, instanceID, userID string) error {
	if s.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	boID, err := s.getInstanceBusinessObjectID(ctx, tenantID, instanceID)
	if err != nil {
		return err
	}

	decision, err := s.requireAccess(ctx, tenantID, boID, AccessLevelWrite)
	if err != nil {
		return fmt.Errorf("access denied for business object: %w", err)
	}

	whereParts := []string{"tenant_id = $1", "id = $2"}
	if decision.RowPredicate != "" {
		whereParts = append(whereParts, "("+decision.RowPredicate+")")
	}

	query := "DELETE FROM business_object_instances WHERE " + strings.Join(whereParts, " AND ")

	result, err := s.db.ExecContext(ctx, query, tenantID, instanceID)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("instance not found")
	}

	return nil
}

// getInstanceBusinessObjectID fetches the BO ID for an instance with tenant scoping.
func (s *BusinessObjectService) getInstanceBusinessObjectID(ctx context.Context, tenantID, instanceID string) (string, error) {
	var boID string
	err := s.db.QueryRowContext(ctx, `SELECT business_object_id FROM business_object_instances WHERE tenant_id = $1 AND id = $2`, tenantID, instanceID).Scan(&boID)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("instance not found")
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve business object id: %w", err)
	}
	return boID, nil
}

// Hasura helper functions

// listBusinessObjectsWithHasura lists all business objects via Hasura
func (s *BusinessObjectService) listBusinessObjectsWithHasura(ctx context.Context, tenantID, datasourceID string) ([]*models.BusinessObjectDefinition, error) {
	query := `
		query ListBusinessObjects($tenantId: uuid!) {
			business_objects(where: {tenant_id: {_eq: $tenantId}}, order_by: {created_at: desc}) {
				id
				tenant_id
				name
				description
				fields
				parent_id
				technical_name
				created_by
				created_at
				updated_at
			}
		}
	`

	variables := map[string]interface{}{
		"tenantId": tenantID,
	}

	resp, err := s.hasura.Query(query, variables)
	if err != nil {
		return nil, err
	}

	objectsData, ok := resp["business_objects"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected response format")
	}

	var objects []*models.BusinessObjectDefinition
	for _, item := range objectsData {
		objMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		bo := &models.BusinessObjectDefinition{}
		if id, ok := objMap["id"].(string); ok {
			bo.ID = id
		}
		if tenantID, ok := objMap["tenant_id"].(string); ok {
			bo.TenantID = tenantID
		}
		if name, ok := objMap["name"].(string); ok {
			bo.Name = name
		}
		if description, ok := objMap["description"].(string); ok {
			bo.Description = description
		}
		// Assuming Hasura returns 'config' or 'fields'
		// If it returns 'fields' as string, we put it into Config for now as legacy
		if fieldsStr, ok := objMap["fields"].(string); ok {
			bo.Config = json.RawMessage(fieldsStr)
		} else if configStr, ok := objMap["config"].(string); ok {
			bo.Config = json.RawMessage(configStr)
		}

		if createdBy, ok := objMap["created_by"].(string); ok {
			bo.CreatedBy = createdBy
		}
		if createdAt, ok := objMap["created_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
				bo.CreatedAt = t
			}
		}
		if updatedAt, ok := objMap["last_modified_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
				bo.LastModifiedAt = t
			}
		}

		objects = append(objects, bo)
	}

	return objects, nil
}

// ============================================================================
// WORKDAY-STYLE CORE/CUSTOM COMPOSITION HELPERS
// ============================================================================

// getGoldCopyTenantID retrieves the ID of the gold copy (core) tenant
func (s *BusinessObjectService) getGoldCopyTenantID(ctx context.Context) (string, error) {
	var goldCopyTenantID string
	query := `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`
	err := s.db.QueryRowContext(ctx, query).Scan(&goldCopyTenantID)
	if err != nil {
		return "", fmt.Errorf("failed to get gold copy tenant: %w", err)
	}
	return goldCopyTenantID, nil
}

// listCoreBusinessObjects lists BOs from the gold copy tenant
func (s *BusinessObjectService) listCoreBusinessObjects(ctx context.Context, goldCopyTenantID string) ([]*models.BusinessObjectDefinition, error) {
	query := `
		SELECT id, tenant_id, name, display_name, parent_id, description, config, icon,
		       created_at, last_modified_at, is_core, is_active, enable_history, core_id
		FROM business_objects
		WHERE tenant_id = $1 AND is_core = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, goldCopyTenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list core business objects: %w", err)
	}
	defer rows.Close()

	var objects []*models.BusinessObjectDefinition
	for rows.Next() {
		bo := &models.BusinessObjectDefinition{}
		var displayName, description, icon sql.NullString
		var parentID, coreID sql.NullString
		var config []byte
		var isCore, isActive sql.NullBool

		err := rows.Scan(&bo.ID, &bo.TenantID, &bo.Name, &displayName, &parentID, &description,
			&config, &icon, &bo.CreatedAt, &bo.LastModifiedAt, &isCore, &isActive, &bo.EnableHistory, &coreID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan core business object: %w", err)
		}

		if displayName.Valid {
			bo.DisplayName = displayName.String
		}
		if parentID.Valid {
			bo.ParentID = parentID
		}
		if description.Valid {
			bo.Description = description.String
		}
		if icon.Valid {
			bo.Icon = icon.String
		}
		if coreID.Valid {
			bo.CoreID = coreID
		}
		if len(config) > 0 {
			bo.Config = config
		}
		bo.Key = bo.Name
		if isCore.Valid {
			bo.IsCore = isCore.Bool
		}
		if isActive.Valid {
			bo.IsActive = isActive.Bool
		}

		objects = append(objects, bo)
	}

	return objects, nil
}

// listTenantCustomBusinessObjects lists only tenant-specific (non-core) BOs
func (s *BusinessObjectService) listTenantCustomBusinessObjects(ctx context.Context, tenantID, datasourceID string) ([]*models.BusinessObjectDefinition, error) {
	query := `
		SELECT id, tenant_id, name, display_name, parent_id, description, config, icon,
		       created_at, last_modified_at, is_core, is_active, enable_history, core_id
		FROM business_objects
		WHERE tenant_id = $1 AND (is_core = false OR is_core IS NULL)
	`
	args := []interface{}{tenantID}

	if datasourceID != "" {
		query += " AND (datasource_id = $2::uuid OR datasource_id IS NULL)"
		args = append(args, datasourceID)
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tenant custom business objects: %w", err)
	}
	defer rows.Close()

	var objects []*models.BusinessObjectDefinition
	for rows.Next() {
		bo := &models.BusinessObjectDefinition{}
		var displayName, description, icon sql.NullString
		var parentID, coreID sql.NullString
		var config []byte
		var isCore, isActive sql.NullBool

		err := rows.Scan(&bo.ID, &bo.TenantID, &bo.Name, &displayName, &parentID, &description,
			&config, &icon, &bo.CreatedAt, &bo.LastModifiedAt, &isCore, &isActive, &bo.EnableHistory, &coreID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant custom business object: %w", err)
		}

		if displayName.Valid {
			bo.DisplayName = displayName.String
		}
		if parentID.Valid {
			bo.ParentID = parentID
		}
		if description.Valid {
			bo.Description = description.String
		}
		if icon.Valid {
			bo.Icon = icon.String
		}
		if coreID.Valid {
			bo.CoreID = coreID
		}
		if len(config) > 0 {
			bo.Config = config
		}
		bo.Key = bo.Name
		if isCore.Valid {
			bo.IsCore = isCore.Bool
		}
		if isActive.Valid {
			bo.IsActive = isActive.Bool
		}

		objects = append(objects, bo)
	}

	return objects, nil
}

// ListBusinessObjectsComposed returns Workday-style composed Core + Custom BOs for a tenant
// Core BOs are loaded from the gold copy tenant, then merged with tenant-specific extensions
func (s *BusinessObjectService) ListBusinessObjectsComposed(ctx context.Context, tenantID, datasourceID string) ([]*models.BusinessObjectDefinition, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}

	// 1. Get gold copy tenant ID
	goldCopyTenantID, err := s.getGoldCopyTenantID(ctx)
	if err != nil {
		// If no gold copy tenant, fall back to regular listing
		logging.GetLogger().Sugar().Warnf("No gold copy tenant found, falling back to regular listing: %v", err)
		return s.ListBusinessObjects(ctx, tenantID, datasourceID)
	}

	// 2. If requesting tenant IS the gold copy, return core-only
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
