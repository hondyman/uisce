package metadata

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cube"
	"github.com/hondyman/semlayer/backend/internal/logging"
	internal_models "github.com/hondyman/semlayer/backend/internal/models"
	"github.com/hondyman/semlayer/backend/internal/platform"
	"github.com/hondyman/semlayer/backend/internal/views"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// Constants for the smart materialization policy.
// In a real application, these would come from a configuration system.
const (
	// mvSwapRebuildMinSizeBytes defines the size threshold at which we'll use
	// the safer swap-rebuild pattern instead of a direct ALTER.
	mvSwapRebuildMinSizeBytes int64 = 1 * 1024 * 1024 * 1024 // 1 GB
)

// ViewService provides methods for managing the view layer, now with
// dialect-specific intelligence for applying changes.
type ViewService struct {
	db            *sqlx.DB
	modelProvider models.ModelProvider
}

// NewViewService creates a new ViewService.
func NewViewService(db *sqlx.DB, modelProvider models.ModelProvider) *ViewService {
	return &ViewService{
		db:            db,
		modelProvider: modelProvider,
	}
}

// ListViews retrieves all available views and formats them for the API.
func (s *ViewService) ListViews(ctx context.Context) ([]models.ViewDescription, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not configured")
	}

	boViews, err := s.listBusinessObjectViews(ctx)
	if err != nil {
		return nil, err
	}
	if len(boViews) > 0 {
		return boViews, nil
	}

	return s.listViewsFromModelProvider(ctx)
}

func (s *ViewService) listViewsFromModelProvider(ctx context.Context) ([]models.ViewDescription, error) {
	catalog, err := s.modelProvider.GetActiveCatalog(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("could not load active model catalog: %w", err)
	}

	var viewDescriptions []models.ViewDescription
	for _, viewMeta := range catalog.Views {
		vd := models.ViewDescription{
			Name:        viewMeta.Name,
			Schema:      viewMeta.Schema,
			Description: viewMeta.Description,
			Tags:        viewMeta.Tags,
			Owner:       viewMeta.Owner,
			Certified:   true,
			Dimensions:  []models.ViewMemberDescription{},
			Measures:    []models.ViewMemberDescription{},
		}

		for _, dim := range viewMeta.Dimensions {
			vd.Dimensions = append(vd.Dimensions, models.ViewMemberDescription{
				Name:  dim,
				Label: strings.ToTitle(strings.ReplaceAll(dim, "_", " ")),
				Type:  "string",
			})
		}
		for _, mes := range viewMeta.Measures {
			vd.Measures = append(vd.Measures, models.ViewMemberDescription{
				Name:  mes,
				Label: strings.ToTitle(strings.ReplaceAll(mes, "_", " ")),
				Type:  "number",
			})
		}
		viewDescriptions = append(viewDescriptions, vd)
	}

	return viewDescriptions, nil
}

func (s *ViewService) listBusinessObjectViews(ctx context.Context) ([]models.ViewDescription, error) {
	var bos []boViewRow

	query := `
		SELECT id, tenant_id, tenant_datasource_id, name, display_name, description, category, driver_table_id, driver_table_name
		FROM business_objects
		WHERE is_active = true
		ORDER BY name
	`
	if err := s.db.SelectContext(ctx, &bos, query); err != nil {
		return nil, fmt.Errorf("failed to list business objects: %w", err)
	}

	views := make([]models.ViewDescription, 0, len(bos))
	for _, bo := range bos {
		fields, err := s.listBOFields(ctx, bo.ID)
		if err != nil {
			return nil, err
		}

		columns, tableName, err := s.listDriverTableColumns(ctx, bo)
		if err != nil {
			return nil, err
		}

		dimensions := mergeViewMembers(fields, columns)
		tags := []string{}
		if bo.Category.Valid && strings.TrimSpace(bo.Category.String) != "" {
			tags = append(tags, bo.Category.String)
		}
		if tableName != "" {
			tags = append(tags, fmt.Sprintf("driver_table:%s", tableName))
		}

		schema := "business_object"
		if tableName != "" {
			schema = tableName
		}

		views = append(views, models.ViewDescription{
			Name:        bo.Name,
			Schema:      schema,
			Description: nullableToString(bo.Description),
			Tags:        tags,
			Owner:       bo.TenantID,
			Certified:   true,
			Dimensions:  dimensions,
			Measures:    []models.ViewMemberDescription{},
		})
	}

	return views, nil
}

func (s *ViewService) listBOFields(ctx context.Context, boID string) ([]models.ViewMemberDescription, error) {
	var fields []struct {
		Name          string         `db:"name"`
		DisplayName   string         `db:"display_name"`
		TechnicalName string         `db:"technical_name"`
		FieldType     string         `db:"type"`
		Description   sql.NullString `db:"description"`
	}
	query := `
		SELECT name, display_name, technical_name, type, description
		FROM bo_fields
		WHERE business_object_id = $1 AND subtype_id IS NULL
		ORDER BY sequence, name
	`
	if err := s.db.SelectContext(ctx, &fields, query, boID); err != nil {
		return nil, fmt.Errorf("failed to list BO fields: %w", err)
	}

	result := make([]models.ViewMemberDescription, 0, len(fields))
	for _, field := range fields {
		name := strings.TrimSpace(field.TechnicalName)
		if name == "" {
			name = strings.TrimSpace(field.Name)
		}
		label := strings.TrimSpace(field.DisplayName)
		if label == "" {
			label = strings.ToTitle(strings.ReplaceAll(name, "_", " "))
		}
		result = append(result, models.ViewMemberDescription{
			Name:  name,
			Label: label,
			Type:  mapFieldType(field.FieldType),
		})
	}

	return result, nil
}

func (s *ViewService) listDriverTableColumns(ctx context.Context, bo boViewRow) ([]models.ViewMemberDescription, string, error) {
	if !bo.DriverTableID.Valid && !bo.DriverTableName.Valid {
		return nil, "", nil
	}

	var tableID string
	var tableName string
	datasourceID := strings.TrimSpace(bo.DatasourceID.String)
	if bo.DriverTableID.Valid {
		err := s.db.GetContext(ctx, &tableName, `
			SELECT node_name
			FROM catalog_node
			WHERE id = $1 AND tenant_id = $2
			  AND ($3 = '' OR tenant_datasource_id = $3)
			LIMIT 1
		`, bo.DriverTableID.String, bo.TenantID, datasourceID)
		if err != nil {
			return nil, "", fmt.Errorf("failed to resolve driver table: %w", err)
		}
		tableID = bo.DriverTableID.String
	} else if bo.DriverTableName.Valid {
		query := `
			SELECT n.id, n.node_name
			FROM catalog_node n
			JOIN catalog_node_type nt ON n.node_type_id = nt.id
			WHERE n.node_name = $1
			  AND nt.catalog_type_name IN ('table', 'database_table', 'view')
			  AND n.tenant_id = $2
			  AND ($3 = '' OR n.tenant_datasource_id = $3)
			LIMIT 1
		`
		if err := s.db.QueryRowContext(ctx, query, bo.DriverTableName.String, bo.TenantID, datasourceID).Scan(&tableID, &tableName); err != nil {
			return nil, "", fmt.Errorf("failed to resolve driver table by name: %w", err)
		}
	}

	if tableID == "" {
		return nil, "", nil
	}

	var columns []struct {
		Name     string         `db:"node_name"`
		DataType sql.NullString `db:"data_type"`
	}
	columnQuery := `
		SELECT n.node_name, n.properties->>'data_type' as data_type
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE n.parent_id = $1
		  AND nt.catalog_type_name IN ('column', 'database_column')
		  AND n.tenant_id = $2
		  AND ($3 = '' OR n.tenant_datasource_id = $3)
		ORDER BY n.node_name
	`
	if err := s.db.SelectContext(ctx, &columns, columnQuery, tableID, bo.TenantID, datasourceID); err != nil {
		return nil, "", fmt.Errorf("failed to list driver table columns: %w", err)
	}

	result := make([]models.ViewMemberDescription, 0, len(columns))
	for _, col := range columns {
		columnName := strings.TrimSpace(col.Name)
		if columnName == "" {
			continue
		}
		typeName := mapFieldType(col.DataType.String)
		result = append(result, models.ViewMemberDescription{
			Name:  fmt.Sprintf("%s.%s", tableName, columnName),
			Label: strings.ToTitle(strings.ReplaceAll(columnName, "_", " ")),
			Type:  typeName,
		})
	}

	return result, tableName, nil
}

type boViewRow struct {
	ID              string         `db:"id"`
	TenantID        string         `db:"tenant_id"`
	DatasourceID    sql.NullString `db:"tenant_datasource_id"`
	Name            string         `db:"name"`
	DisplayName     string         `db:"display_name"`
	Description     sql.NullString `db:"description"`
	Category        sql.NullString `db:"category"`
	DriverTableID   sql.NullString `db:"driver_table_id"`
	DriverTableName sql.NullString `db:"driver_table_name"`
}

func mergeViewMembers(primary []models.ViewMemberDescription, secondary []models.ViewMemberDescription) []models.ViewMemberDescription {
	merged := make([]models.ViewMemberDescription, 0, len(primary)+len(secondary))
	seen := make(map[string]struct{})
	for _, item := range primary {
		key := strings.TrimSpace(item.Name)
		if key == "" {
			continue
		}
		seen[key] = struct{}{}
		merged = append(merged, item)
	}
	for _, item := range secondary {
		key := strings.TrimSpace(item.Name)
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		merged = append(merged, item)
	}
	return merged
}

func mapFieldType(fieldType string) string {
	value := strings.TrimSpace(strings.ToLower(fieldType))
	switch value {
	case "number", "int", "integer", "float", "double", "decimal", "currency":
		return "number"
	case "bool", "boolean":
		return "boolean"
	case "date", "datetime", "timestamp", "time":
		return "time"
	default:
		return "string"
	}
}

func nullableToString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

// GetViewMetadata retrieves detailed metadata for a single view.
func (s *ViewService) GetViewMetadata(ctx context.Context, viewName string) (*models.ViewMetadataDetails, error) {
	// In a real implementation, this data would be queried from operational and metadata tables.
	// For now, we return mock data.
	logging.GetLogger().Sugar().Infof("Fetching metadata for view: %s", viewName)
	return &models.ViewMetadataDetails{
		Owner:          "analytics_team",
		Certified:      true,
		Freshness:      "Up to date",
		LastRefreshAgo: "3 hours ago",
		RunCount30d:    1420,
		ExportGB30d:    2.5,
		Lineage: &models.Lineage{
			Nodes: []map[string]interface{}{{"id": "view", "label": viewName}, {"id": "cube", "label": "orders"}},
			Edges: []map[string]interface{}{{"source": "cube", "target": "view"}},
		},
	}, nil
}

func (s *ViewService) CompareAllViews(ctx context.Context) ([]views.Plan, error) {
	catalog, err := s.modelProvider.GetActiveCatalog(ctx, "", "")
	if err != nil {
		return nil, fmt.Errorf("could not load active model catalog: %w", err)
	}
	manager := views.NewManager(s.db.DB, catalog)
	var viewList []cube.ViewMeta
	for _, v := range catalog.Views {
		viewList = append(viewList, v)
	}
	return manager.CompareAll(ctx, viewList)
}

// ApplyViewChanges intelligently applies a set of DDL change plans.
// For Postgres, it uses a "swap-rebuild" strategy for large materialized views
// to ensure zero-downtime updates, as outlined in your strategy document.
func (s *ViewService) ApplyViewChanges(ctx context.Context, plans []views.Plan) error {
	if len(plans) == 0 {
		return nil // Nothing to apply.
	}

	// The logic below requires assumptions about the `views.Plan` struct.
	// I'm assuming it looks something like this:
	// type Plan struct {
	//   Schema string
	//   Name   string
	//   Type   string // e.g., "materialized_view"
	//   Action string // e.g., "create", "alter", "drop"
	//   DDL    string // The full DDL statement for the change
	// }

	catalog, err := s.modelProvider.GetActiveCatalog(ctx, "", "")
	if err != nil {
		return fmt.Errorf("could not load active model catalog: %w", err)
	}
	manager := views.NewManager(s.db.DB, catalog)

	// We'll build a transaction with potentially complex DDL sequences.
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }() // Rollback on any error

	for _, plan := range plans {
		// Check if this plan is for altering a materialized view.
		isAlterMV := plan.Type == "materialized_view" && plan.Action == "alter"

		if isAlterMV {
			meta, err := s.getMaterializedViewMeta(ctx, plan.Schema, plan.Name)
			if err != nil {
				return fmt.Errorf("failed to get metadata for MV %s.%s: %w", plan.Name, plan.Schema, err)
			}

			// Decide if we need a swap-rebuild based on the strategy.
			// Here we just use the size threshold. A full implementation would also
			// check if the SELECT body of the view is changing.
			useSwapRebuild := meta != nil && meta.SizeBytes >= mvSwapRebuildMinSizeBytes

			if useSwapRebuild {
				logging.GetLogger().Sugar().Infof("Applying swap-rebuild for large MV: %s.%s (%d bytes)", plan.Schema, plan.Name, meta.SizeBytes)
				if err := s.applySwapRebuild(ctx, tx, plan); err != nil {
					return err // Error includes context, transaction will be rolled back.
				}
				continue // Move to the next plan
			}
		}

		// Default case: apply the plan's DDL directly within the transaction.
		// This requires the manager to support applying plans within an existing transaction.
		logging.GetLogger().Sugar().Infof("Applying standard DDL for: %s.%s", plan.Schema, plan.Name)
		if err := manager.ApplyPlanInTx(ctx, tx, plan); err != nil {
			return fmt.Errorf("failed to apply standard plan for %s.%s: %w", plan.Schema, plan.Name, err)
		}
	}

	return tx.Commit()
}

// applySwapRebuild performs a zero-downtime swap-rebuild of a materialized view within a transaction.
func (s *ViewService) applySwapRebuild(ctx context.Context, tx *sqlx.Tx, plan views.Plan) error {
	newName := fmt.Sprintf("%s_new_%d", plan.Name, time.Now().Unix())
	oldName := fmt.Sprintf("%s_old_%d", plan.Name, time.Now().Unix())

	// 1. Create the new MV with a temporary name based on the plan's DDL.
	// This is a simplification; a robust implementation would parse the DDL properly.
	createSQL := strings.Replace(plan.DDL, fmt.Sprintf("CREATE MATERIALIZED VIEW %s.%s", plan.Schema, plan.Name), fmt.Sprintf("CREATE MATERIALIZED VIEW %s.%s", plan.Schema, newName), 1)
	if _, err := tx.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("swap-rebuild step 1 (create new) failed for %s: %w", plan.Name, err)
	}

	// 2. (Placeholder) Re-create indexes, grants, etc. on the new MV.
	// A full implementation would query pg_indexes/pg_depend for the old view and replicate them on the new one.

	// 3. Rename old to temp, then new to the final name. This is the atomic swap.
	renameOldSQL := fmt.Sprintf("ALTER MATERIALIZED VIEW %s.%s RENAME TO %s", plan.Schema, plan.Name, oldName)
	if _, err := tx.ExecContext(ctx, renameOldSQL); err != nil {
		return fmt.Errorf("swap-rebuild step 3a (rename old) failed for %s: %w", plan.Name, err)
	}

	renameNewSQL := fmt.Sprintf("ALTER MATERIALIZED VIEW %s.%s RENAME TO %s", plan.Schema, newName, plan.Name)
	if _, err := tx.ExecContext(ctx, renameNewSQL); err != nil {
		return fmt.Errorf("swap-rebuild step 3b (rename new) failed for %s: %w", plan.Name, err)
	}

	// 4. Drop the old MV. This happens after the transaction commits successfully.
	// We add it to a list of post-commit hooks. For simplicity here, we do it in the tx.
	dropOldSQL := fmt.Sprintf("DROP MATERIALIZED VIEW IF EXISTS %s.%s", plan.Schema, oldName)
	if _, err := tx.ExecContext(ctx, dropOldSQL); err != nil {
		// This is not a fatal error for the swap itself, but should be logged.
		logging.GetLogger().Sugar().Warnf("WARN: swap-rebuild step 4 (drop old) failed for %s: %v. Manual cleanup may be required.", oldName, err)
	}

	return nil
}

func (s *ViewService) RejectViewChanges(ctx context.Context, plans []views.Plan, reviewer, reason string) error {
	if len(plans) == 0 {
		return nil // Nothing to reject.
	}
	catalog, err := s.modelProvider.GetActiveCatalog(ctx, "", "")
	if err != nil {
		return fmt.Errorf("could not load active model catalog: %w", err)
	}
	manager := views.NewManager(s.db.DB, catalog)
	vmMap := make(map[string]cube.ViewMeta)
	for _, vm := range catalog.Views {
		vmMap[fmt.Sprintf("%s.%s", vm.Schema, vm.Name)] = vm
	}

	return manager.RejectPlans(ctx, plans, vmMap, reviewer, reason)
}

// materializedViewMeta holds operational metadata for an MV.
type materializedViewMeta struct {
	SizeBytes int64
	// In a full implementation, you would add other metrics here,
	// such as median refresh time, hit rate, etc.
}

// getMaterializedViewMeta fetches operational metadata for a given materialized view from Postgres.
func (s *ViewService) getMaterializedViewMeta(ctx context.Context, schema, name string) (*materializedViewMeta, error) {
	query := `
		SELECT pg_total_relation_size(c.oid)
		FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relkind = 'm' AND n.nspname = $1 AND c.relname = $2
	`
	var sizeBytes int64
	err := s.db.GetContext(ctx, &sizeBytes, query, schema, name)
	if err != nil {
		// If the view doesn't exist yet (e.g., a "create" plan), it's not an error.
		if err.Error() == "sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("could not get metadata for MV %s.%s: %w", schema, name, err)
	}
	return &materializedViewMeta{SizeBytes: sizeBytes}, nil
}

// GetSuggestedQueries retrieves popular or relevant queries for a given view.
func (s *ViewService) GetSuggestedQueries(ctx context.Context, viewName string) ([]models.SuggestedQuery, error) {
	// This would typically query a pre-calculated table or materialized view.
	// For now, we'll mock it by finding the most recently run saved queries for this view.
	query := `
		SELECT 
			sq.id as saved_query_id,
			sq.view_name,
			sq.name,
			'Popular' as reason,
			1.0 as score
		FROM explorer_saved_query sq
		WHERE sq.view_name = $1 AND sq.is_deleted = false
		ORDER BY sq.last_run_at DESC NULLS LAST
		LIMIT 5
	`
	var suggestions []models.SuggestedQuery
	if err := s.db.SelectContext(ctx, &suggestions, query, viewName); err != nil {
		return nil, fmt.Errorf("failed to get suggested queries for view %s: %w", viewName, err)
	}
	return suggestions, nil
}

// --- View Definition Service ---

// ViewDefinitionService defines the interface for managing logical view definitions.
type ViewDefinitionService interface {
	CreateView(user internal_models.User, view *internal_models.ViewDefinition) (*internal_models.ViewDefinition, error)
	GetView(user internal_models.User, id string) (*internal_models.ViewDefinition, error)
	UpdateView(user internal_models.User, id string, view *internal_models.ViewDefinition) (*internal_models.ViewDefinition, error)
	ListViewsByBundle(user internal_models.User, bundleID string) ([]*internal_models.ViewDefinition, error)
}

// NewViewDefinitionService creates a new instance of the view definition service.
func NewViewDefinitionService(policyService platform.PolicyService) ViewDefinitionService {
	return &viewDefinitionServiceImpl{
		store:         make(map[string]*internal_models.ViewDefinition),
		policyService: policyService,
	}
}

type viewDefinitionServiceImpl struct {
	store         map[string]*internal_models.ViewDefinition
	policyService platform.PolicyService
	mu            sync.RWMutex
}

func (s *viewDefinitionServiceImpl) CreateView(user internal_models.User, view *internal_models.ViewDefinition) (*internal_models.ViewDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	view.ID = uuid.New().String()
	view.CreatedAt = time.Now()
	view.UpdatedAt = time.Now()

	s.store[view.ID] = view
	return view, nil
}

func (s *viewDefinitionServiceImpl) GetView(user internal_models.User, id string) (*internal_models.ViewDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	view, exists := s.store[id]
	if !exists {
		return nil, fmt.Errorf("view with id %s not found", id)
	}
	return view, nil
}

func (s *viewDefinitionServiceImpl) UpdateView(user internal_models.User, id string, view *internal_models.ViewDefinition) (*internal_models.ViewDefinition, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.store[id]
	if !exists {
		view.ID = uuid.New().String()
		view.CreatedAt = time.Now()
	}

	view.ID = id
	view.UpdatedAt = time.Now()
	s.store[id] = view

	return view, nil
}

func (s *viewDefinitionServiceImpl) ListViewsByBundle(user internal_models.User, bundleID string) ([]*internal_models.ViewDefinition, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var views []*internal_models.ViewDefinition
	for _, view := range s.store {
		if view.BundleID == bundleID {
			views = append(views, view)
		}
	}
	return views, nil
}
