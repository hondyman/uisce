package semantic

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"github.com/jmoiron/sqlx"
)

// Service provides semantic layer operations
type Service struct {
	db *sqlx.DB
}

// NewService creates a new semantic layer service
func NewService(db *sqlx.DB) *Service {
	return &Service{db: db}
}

// CreateCube creates a new cube
func (s *Service) CreateCube(ctx context.Context, cube *Cube) error {
	query := `
		INSERT INTO semantic_cubes_v2 (
			tenant_id, name, display_name, description, sql, 
			refresh_key, pre_aggregations, joins, metadata, status, created_by,
			source_cube_id, is_system
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		) RETURNING id, created_at, updated_at
	`

	preAggsJSON, _ := json.Marshal(cube.PreAggregations)
	joinsJSON, _ := json.Marshal(cube.Joins)
	metadataJSON, _ := json.Marshal(cube.Metadata)

	err := s.db.QueryRowContext(ctx, query,
		cube.TenantID, cube.Name, cube.DisplayName, cube.Description, cube.SQL,
		cube.RefreshKey, preAggsJSON, joinsJSON, metadataJSON, cube.Status, cube.CreatedBy,
		cube.SourceCubeID, cube.IsSystem,
	).Scan(&cube.ID, &cube.CreatedAt, &cube.UpdatedAt)

	return err
}

// UpdateCube updates an existing cube
func (s *Service) UpdateCube(ctx context.Context, cube *Cube) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation { update_semantic_cubes_v2(
	//   where: {id: {_eq: $id}, tenant_id: {_eq: $tenant_id}}
	//   _set: {display_name, description, sql, refresh_key, pre_aggregations, joins, metadata, status, updated_at, source_cube_id, is_system}
	// ) { affected_rows }}
	query := `
		UPDATE semantic_cubes_v2 SET
			display_name = $1, description = $2, sql = $3, 
			refresh_key = $4, pre_aggregations = $5, joins = $6, 
			metadata = $7, status = $8, updated_at = now(),
			source_cube_id = $9, is_system = $10
		WHERE id = $11 AND tenant_id = $12
	`

	preAggsJSON, _ := json.Marshal(cube.PreAggregations)
	joinsJSON, _ := json.Marshal(cube.Joins)
	metadataJSON, _ := json.Marshal(cube.Metadata)

	result, err := s.db.ExecContext(ctx, query,
		cube.DisplayName, cube.Description, cube.SQL,
		cube.RefreshKey, preAggsJSON, joinsJSON, metadataJSON, cube.Status,
		cube.SourceCubeID, cube.IsSystem,
		cube.ID, cube.TenantID,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	// Invalidate cache
	return s.InvalidateCubeCache(ctx, cube.TenantID, cube.Name)
}

// GetCube retrieves a cube by name, handling inheritance
func (s *Service) GetCube(ctx context.Context, tenantID, name string) (*Cube, error) {
	// Try cache first
	cachedCube, err := s.getCubeFromCache(ctx, tenantID, name)
	if err == nil && cachedCube != nil {
		return &cachedCube.Metadata, nil
	}

	// Load from database
	cube := &Cube{}
	query := `
		SELECT id, tenant_id, name, display_name, description, sql,
		       refresh_key, pre_aggregations, joins, metadata, status, version,
		       created_by, created_at, updated_at, source_cube_id, is_system
		FROM semantic_cubes_v2
		WHERE tenant_id = $1 AND name = $2 AND status != 'deleted'
	`

	var preAggsJSON, joinsJSON, metadataJSON []byte
	var sourceCubeID *string
	err = s.db.QueryRowContext(ctx, query, tenantID, name).Scan(
		&cube.ID, &cube.TenantID, &cube.Name, &cube.DisplayName, &cube.Description, &cube.SQL,
		&cube.RefreshKey, &preAggsJSON, &joinsJSON, &metadataJSON, &cube.Status, &cube.Version,
		&cube.CreatedBy, &cube.CreatedAt, &cube.UpdatedAt, &sourceCubeID, &cube.IsSystem,
	)
	if err != nil {
		return nil, err
	}

	cube.SourceCubeID = sourceCubeID
	json.Unmarshal(preAggsJSON, &cube.PreAggregations)
	json.Unmarshal(joinsJSON, &cube.Joins)
	json.Unmarshal(metadataJSON, &cube.Metadata)

	// Load dimensions and measures
	cube.Dimensions, _ = s.GetDimensions(ctx, cube.ID)
	cube.Measures, _ = s.GetMeasures(ctx, cube.ID)

	// Handle inheritance
	if cube.SourceCubeID != nil {
		sourceCube, err := s.GetCubeByID(ctx, *cube.SourceCubeID)
		if err == nil {
			cube = s.mergeCubes(sourceCube, cube)
		}
	}

	// Cache the cube
	s.cacheCube(ctx, tenantID, name, cube)

	return cube, nil
}

// GetCubeByID retrieves a cube by ID (internal helper for inheritance)
func (s *Service) GetCubeByID(ctx context.Context, id string) (*Cube, error) {
	cube := &Cube{}
	query := `
		SELECT id, tenant_id, name, display_name, description, sql,
		       refresh_key, pre_aggregations, joins, metadata, status, version,
		       created_by, created_at, updated_at, source_cube_id, is_system
		FROM semantic_cubes_v2
		WHERE id = $1
	`

	var preAggsJSON, joinsJSON, metadataJSON []byte
	var sourceCubeID *string
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&cube.ID, &cube.TenantID, &cube.Name, &cube.DisplayName, &cube.Description, &cube.SQL,
		&cube.RefreshKey, &preAggsJSON, &joinsJSON, &metadataJSON, &cube.Status, &cube.Version,
		&cube.CreatedBy, &cube.CreatedAt, &cube.UpdatedAt, &sourceCubeID, &cube.IsSystem,
	)
	if err != nil {
		return nil, err
	}

	cube.SourceCubeID = sourceCubeID
	json.Unmarshal(preAggsJSON, &cube.PreAggregations)
	json.Unmarshal(joinsJSON, &cube.Joins)
	json.Unmarshal(metadataJSON, &cube.Metadata)

	cube.Dimensions, _ = s.GetDimensions(ctx, cube.ID)
	cube.Measures, _ = s.GetMeasures(ctx, cube.ID)

	if cube.SourceCubeID != nil {
		sourceCube, err := s.GetCubeByID(ctx, *cube.SourceCubeID)
		if err == nil {
			cube = s.mergeCubes(sourceCube, cube)
		}
	}

	return cube, nil
}

// mergeCubes merges a custom cube (override) into a core cube (base)
func (s *Service) mergeCubes(base, override *Cube) *Cube {
	// Start with base as the foundation
	merged := *base

	// Override top-level properties if specified
	merged.ID = override.ID // Keep custom ID
	merged.TenantID = override.TenantID
	merged.Name = override.Name
	merged.DisplayName = override.DisplayName
	if override.Description != "" {
		merged.Description = override.Description
	}
	if override.SQL != "" {
		merged.SQL = override.SQL
	}
	merged.Status = override.Status
	merged.Version = override.Version
	merged.IsSystem = override.IsSystem
	merged.SourceCubeID = override.SourceCubeID

	// Merge Dimensions
	dimMap := make(map[string]Dimension)
	for _, d := range base.Dimensions {
		dimMap[d.Name] = d
	}
	for _, d := range override.Dimensions {
		dimMap[d.Name] = d // Override or Add
	}
	merged.Dimensions = make([]Dimension, 0, len(dimMap))
	for _, d := range dimMap {
		if d.Shown { // Only include visible dimensions
			merged.Dimensions = append(merged.Dimensions, d)
		}
	}

	// Merge Measures
	measureMap := make(map[string]Measure)
	for _, m := range base.Measures {
		measureMap[m.Name] = m
	}
	for _, m := range override.Measures {
		measureMap[m.Name] = m // Override or Add
	}
	merged.Measures = make([]Measure, 0, len(measureMap))
	for _, m := range measureMap {
		merged.Measures = append(merged.Measures, m)
	}

	// Merge Joins
	joinMap := make(map[string]Join)
	for _, j := range base.Joins {
		joinMap[j.Name] = j
	}
	for _, j := range override.Joins {
		joinMap[j.Name] = j
	}
	merged.Joins = make([]Join, 0, len(joinMap))
	for _, j := range joinMap {
		merged.Joins = append(merged.Joins, j)
	}

	return &merged
}

// ListCubes lists all cubes for a tenant
func (s *Service) ListCubes(ctx context.Context, tenantID string) ([]*Cube, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { semantic_cubes_v2(
	//   where: {tenant_id: {_eq: $tenant_id}, status: {_neq: "deleted"}}
	//   order_by: {name: asc}
	// ) { id tenant_id name display_name description sql refresh_key status version created_at updated_at }}
	query := `
		SELECT id, tenant_id, name, display_name, description, sql,
		       refresh_key, status, version, created_at, updated_at
		FROM semantic_cubes_v2
		WHERE tenant_id = $1 AND status != 'deleted'
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cubes []*Cube
	for rows.Next() {
		cube := &Cube{}
		err := rows.Scan(
			&cube.ID, &cube.TenantID, &cube.Name, &cube.DisplayName, &cube.Description, &cube.SQL,
			&cube.RefreshKey, &cube.Status, &cube.Version, &cube.CreatedAt, &cube.UpdatedAt,
		)
		if err != nil {
			continue
		}
		cubes = append(cubes, cube)
	}

	return cubes, nil
}

// CreateDimension creates a new dimension
func (s *Service) CreateDimension(ctx context.Context, dim *Dimension) error {
	query := `
		INSERT INTO semantic_dimensions_v2 (
			cube_id, name, display_name, type, sql, format,
			case_sensitive, primary_key, shown, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	metadataJSON, _ := json.Marshal(dim.Metadata)

	return s.db.QueryRowContext(ctx, query,
		dim.CubeID, dim.Name, dim.DisplayName, dim.Type, dim.SQL, dim.Format,
		dim.CaseSensitive, dim.PrimaryKey, dim.Shown, metadataJSON,
	).Scan(&dim.ID, &dim.CreatedAt)
}

// GetDimensions retrieves all dimensions for a cube
func (s *Service) GetDimensions(ctx context.Context, cubeID string) ([]Dimension, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { semantic_dimensions_v2(
	//   where: {cube_id: {_eq: $cube_id}}
	//   order_by: {name: asc}
	// ) { id cube_id name display_name type sql format case_sensitive primary_key shown metadata created_at }}
	query := `
		SELECT id, cube_id, name, display_name, type, sql, format,
		       case_sensitive, primary_key, shown, metadata, created_at
		FROM semantic_dimensions_v2
		WHERE cube_id = $1
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query, cubeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dimensions []Dimension
	for rows.Next() {
		dim := Dimension{}
		var metadataJSON []byte
		err := rows.Scan(
			&dim.ID, &dim.CubeID, &dim.Name, &dim.DisplayName, &dim.Type, &dim.SQL, &dim.Format,
			&dim.CaseSensitive, &dim.PrimaryKey, &dim.Shown, &metadataJSON, &dim.CreatedAt,
		)
		if err != nil {
			continue
		}
		json.Unmarshal(metadataJSON, &dim.Metadata)
		dimensions = append(dimensions, dim)
	}

	return dimensions, nil
}

// CreateMeasure creates a new measure
func (s *Service) CreateMeasure(ctx context.Context, measure *Measure) error {
	query := `
		INSERT INTO semantic_measures_v2 (
			cube_id, name, display_name, type, sql, format,
			rolling_window, drill_members, filters, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	drillMembersJSON, _ := json.Marshal(measure.DrillMembers)
	filtersJSON, _ := json.Marshal(measure.Filters)
	metadataJSON, _ := json.Marshal(measure.Metadata)

	return s.db.QueryRowContext(ctx, query,
		measure.CubeID, measure.Name, measure.DisplayName, measure.Type, measure.SQL, measure.Format,
		measure.RollingWindow, drillMembersJSON, filtersJSON, metadataJSON,
	).Scan(&measure.ID, &measure.CreatedAt)
}

// GetMeasures retrieves all measures for a cube
func (s *Service) GetMeasures(ctx context.Context, cubeID string) ([]Measure, error) {
	// TODO: Refactor to Hasura GraphQL
	// query { semantic_measures_v2(
	//   where: {cube_id: {_eq: $cube_id}}
	//   order_by: {name: asc}
	// ) { id cube_id name display_name type sql format rolling_window drill_members filters metadata created_at }}
	query := `
		SELECT id, cube_id, name, display_name, type, sql, format,
		       rolling_window, drill_members, filters, metadata, created_at
		FROM semantic_measures_v2
		WHERE cube_id = $1
		ORDER BY name
	`

	rows, err := s.db.QueryContext(ctx, query, cubeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var measures []Measure
	for rows.Next() {
		measure := Measure{}
		var drillMembersJSON, filtersJSON, metadataJSON []byte
		err := rows.Scan(
			&measure.ID, &measure.CubeID, &measure.Name, &measure.DisplayName, &measure.Type, &measure.SQL, &measure.Format,
			&measure.RollingWindow, &drillMembersJSON, &filtersJSON, &metadataJSON, &measure.CreatedAt,
		)
		if err != nil {
			continue
		}
		json.Unmarshal(drillMembersJSON, &measure.DrillMembers)
		json.Unmarshal(filtersJSON, &measure.Filters)
		json.Unmarshal(metadataJSON, &measure.Metadata)
		measures = append(measures, measure)
	}

	return measures, nil
}

// Cache operations

func (s *Service) getCubeFromCache(ctx context.Context, tenantID, cubeName string) (*CubeMetadata, error) {
	query := `
		SELECT tenant_id, cube_name, metadata, dimensions, measures, pre_aggregations, cached_at
		FROM semantic_cube_cache
		WHERE tenant_id = $1 AND cube_name = $2
	`

	cached := &CubeMetadata{}
	var metadataJSON, dimensionsJSON, measuresJSON, preAggsJSON []byte

	err := s.db.QueryRowContext(ctx, query, tenantID, cubeName).Scan(
		&cached.TenantID, &cached.CubeName, &metadataJSON, &dimensionsJSON, &measuresJSON, &preAggsJSON, &cached.CachedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	json.Unmarshal(metadataJSON, &cached.Metadata)
	json.Unmarshal(dimensionsJSON, &cached.Dimensions)
	json.Unmarshal(measuresJSON, &cached.Measures)
	json.Unmarshal(preAggsJSON, &cached.PreAggregations)

	return cached, nil
}

func (s *Service) cacheCube(ctx context.Context, tenantID, cubeName string, cube *Cube) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation { insert_semantic_cube_cache_one(
	//   object: {tenant_id, cube_name, metadata, dimensions, measures, pre_aggregations}
	//   on_conflict: {constraint: semantic_cube_cache_pkey, update_columns: [metadata, dimensions, measures, pre_aggregations, cached_at]}
	// ) { tenant_id cube_name }}
	query := `
		INSERT INTO semantic_cube_cache (tenant_id, cube_name, metadata, dimensions, measures, pre_aggregations)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (tenant_id, cube_name) DO UPDATE
		SET metadata = EXCLUDED.metadata,
		    dimensions = EXCLUDED.dimensions,
		    measures = EXCLUDED.measures,
		    pre_aggregations = EXCLUDED.pre_aggregations,
		    cached_at = now()
	`

	metadataJSON, _ := json.Marshal(cube)
	dimensionsJSON, _ := json.Marshal(cube.Dimensions)
	measuresJSON, _ := json.Marshal(cube.Measures)
	preAggsJSON, _ := json.Marshal(cube.PreAggregations)

	_, err := s.db.ExecContext(ctx, query, tenantID, cubeName, metadataJSON, dimensionsJSON, measuresJSON, preAggsJSON)
	if err != nil {
		log.Printf("Failed to cache cube: %v", err)
	}

	return err
}

// InvalidateCubeCache invalidates the cache for a cube
func (s *Service) InvalidateCubeCache(ctx context.Context, tenantID, cubeName string) error {
	// TODO: Refactor to Hasura GraphQL
	// mutation { delete_semantic_cube_cache(
	//   where: {tenant_id: {_eq: $tenant_id}, cube_name: {_eq: $cube_name}}
	// ) { affected_rows }}
	query := `DELETE FROM semantic_cube_cache WHERE tenant_id = $1 AND cube_name = $2`
	_, err := s.db.ExecContext(ctx, query, tenantID, cubeName)
	return err
}

// RecordQueryHistory records a query execution
func (s *Service) RecordQueryHistory(ctx context.Context, history *QueryHistory) error {
	// TODO(hasura-migration): Replace SQL INSERT with Hasura GraphQL mutation
	// Example GraphQL mutation:
	// mutation RecordQueryHistory($object: semantic_query_history_v2_insert_input!) {
	//   insert_semantic_query_history_v2_one(object: $object) { id }
	// }
	//
	// SQL fallback (kept for backward compatibility):
	query := `
		INSERT INTO semantic_query_history_v2 (
			tenant_id, user_id, cube_name, query, generated_sql,
			execution_time_ms, result_rows, cache_hit, pre_agg_used, error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	queryJSON, _ := json.Marshal(history.Query)

	_, err := s.db.ExecContext(ctx, query,
		history.TenantID, history.UserID, history.CubeName, queryJSON, history.GeneratedSQL,
		history.ExecutionTimeMs, history.ResultRows, history.CacheHit, history.PreAggUsed, history.Error,
	)

	return err
}

// GetQueryHistory retrieves query history for a tenant
func (s *Service) GetQueryHistory(ctx context.Context, tenantID string, limit int) ([]QueryHistory, error) {
	// TODO(hasura-migration): Replace SQL SELECT with Hasura GraphQL query
	// Example GraphQL query:
	// query GetQueryHistory($tenant_id: String!, $limit: Int!) {
	//   semantic_query_history_v2(
	//     where: {tenant_id: {_eq: $tenant_id}},
	//     order_by: {created_at: desc},
	//     limit: $limit
	//   ) {
	//     id tenant_id user_id cube_name query generated_sql execution_time_ms result_rows cache_hit pre_agg_used error created_at
	//   }
	// }
	//
	// SQL fallback (kept for backward compatibility):
	query := `
		SELECT id, tenant_id, user_id, cube_name, query, generated_sql,
		       execution_time_ms, result_rows, cache_hit, pre_agg_used, error, created_at
		FROM semantic_query_history_v2
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []QueryHistory
	for rows.Next() {
		h := QueryHistory{}
		var queryJSON []byte
		var userID, cubeName, generatedSQL, preAggUsed, errorMsg sql.NullString

		err := rows.Scan(
			&h.ID, &h.TenantID, &userID, &cubeName, &queryJSON, &generatedSQL,
			&h.ExecutionTimeMs, &h.ResultRows, &h.CacheHit, &preAggUsed, &errorMsg, &h.CreatedAt,
		)
		if err != nil {
			continue
		}

		json.Unmarshal(queryJSON, &h.Query)
		if userID.Valid {
			h.UserID = userID.String
		}
		if cubeName.Valid {
			h.CubeName = cubeName.String
		}
		if generatedSQL.Valid {
			h.GeneratedSQL = generatedSQL.String
		}
		if preAggUsed.Valid {
			h.PreAggUsed = preAggUsed.String
		}
		if errorMsg.Valid {
			h.Error = errorMsg.String
		}

		history = append(history, h)
	}

	return history, nil
}
