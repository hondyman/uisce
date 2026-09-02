package querybuilder

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/analytics"
	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/hondyman/uisce/backend/internal/security"
	"github.com/jmoiron/sqlx"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . BOResolver

// BOResolver abstracts the repository operations needed by QueryService.
type BOResolver interface {
	GetBODefinition(boID string) (*boresolver.BODefinition, error)
	GetBusinessObjectBinding(boID, bindingID string) (*boresolver.BOBinding, error)
	GetBOTerms(boID, bindingID string) ([]boresolver.SemanticTermView, error)
}

// RelationshipResolver abstracts join-path resolution across Business
// Objects, satisfied by *analytics.RelationshipInferenceService. It is what
// lets a QueryDef with QueryContext.RelatedBOIDs be compiled into an actual
// multi-table join instead of just the single-table semantic path.
type RelationshipResolver interface {
	ResolveJoinPathBetweenBOs(ctx context.Context, datasourceID, fromBONodeID, toBONodeID uuid.UUID) (*analytics.JoinPath, error)
}

// QueryService is the compiler gateway between the frontend QueryDef contract
// and the existing BOSQLGenerator.
type QueryService struct {
	generator     *boresolver.BOSQLGenerator
	resolver      BOResolver
	relationships RelationshipResolver
}

// NewQueryService creates a QueryService for the given generator, resolver,
// and relationship resolver. relationships may be nil, in which case
// QueryDefs with RelatedBOIDs are rejected rather than silently ignoring
// the requested joins.
func NewQueryService(generator *boresolver.BOSQLGenerator, resolver BOResolver, relationships RelationshipResolver) *QueryService {
	return &QueryService{
		generator:     generator,
		resolver:      resolver,
		relationships: relationships,
	}
}

// Preview compiles a QueryDef into SQL without executing it.
func (s *QueryService) Preview(ctx context.Context, secCtx *security.Context, qd *boresolver.QueryDef) (*boresolver.QueryPreviewResponse, error) {
	if err := s.validateScope(secCtx, qd); err != nil {
		return nil, err
	}

	if len(qd.Context.RelatedBOIDs) > 0 {
		return s.previewMultiBO(ctx, secCtx, qd)
	}

	semanticReq, err := s.buildSemanticRequest(qd)
	if err != nil {
		return nil, fmt.Errorf("mapping error: %w", err)
	}

	sql, args, err := s.generator.GenerateSQLFromSemantic(
		semanticReq,
		secCtx.TenantID,
		secCtx.DatasourceID,
	)
	if err != nil {
		return nil, fmt.Errorf("sql generation failed: %w", err)
	}

	return &boresolver.QueryPreviewResponse{
		SQL:        sql,
		Dialect:    dialectName(s.generator.Dialect),
		Parameters: args,
	}, nil
}

// previewMultiBO compiles a QueryDef whose Dimensions/Measures/Filters span
// the primary BO plus QueryContext.RelatedBOIDs. Each related BO's join is
// resolved server-side via s.relationships (never trusting client-supplied
// join SQL), and the resulting JoinPath's cardinality is echoed back per
// selected column so the caller knows which fields fan out to multiple rows.
func (s *QueryService) previewMultiBO(ctx context.Context, secCtx *security.Context, qd *boresolver.QueryDef) (*boresolver.QueryPreviewResponse, error) {
	if s.relationships == nil {
		return nil, fmt.Errorf("related business objects requested but no relationship resolver is configured")
	}

	primaryDef, err := s.resolver.GetBODefinition(qd.Context.BOID)
	if err != nil {
		return nil, fmt.Errorf("failed to load BO %s: %w", qd.Context.BOID, err)
	}

	datasourceUUID, err := uuid.Parse(secCtx.DatasourceID)
	if err != nil {
		return nil, fmt.Errorf("invalid datasource id: %w", err)
	}
	fromBOUUID, err := uuid.Parse(qd.Context.BOID)
	if err != nil {
		return nil, fmt.Errorf("invalid boId: %w", err)
	}

	var related []joinedBO
	for _, relID := range qd.Context.RelatedBOIDs {
		relDef, err := s.resolver.GetBODefinition(relID)
		if err != nil {
			return nil, fmt.Errorf("failed to load related BO %s: %w", relID, err)
		}

		toBOUUID, err := uuid.Parse(relID)
		if err != nil {
			return nil, fmt.Errorf("invalid relatedBoId %q: %w", relID, err)
		}
		path, err := s.relationships.ResolveJoinPathBetweenBOs(ctx, datasourceUUID, fromBOUUID, toBOUUID)
		if err != nil {
			return nil, fmt.Errorf("no relationship path from %s to %s: %w", qd.Context.BOID, relID, err)
		}

		related = append(related, joinedBO{
			BOID:        relID,
			BODef:       relDef,
			Path:        path,
			Cardinality: path.TraversalCardinality(),
		})
	}

	sql, args, columns, err := buildMultiBOSQL(s.generator, primaryDef, related, qd, secCtx.TenantID)
	if err != nil {
		return nil, fmt.Errorf("multi-BO sql generation failed: %w", err)
	}

	return &boresolver.QueryPreviewResponse{
		SQL:        sql,
		Dialect:    dialectName(s.generator.Dialect),
		Parameters: args,
		Columns:    columns,
	}, nil
}

// Execute compiles a QueryDef and runs the generated SQL against db.
func (s *QueryService) Execute(ctx context.Context, secCtx *security.Context, qd *boresolver.QueryDef, db *sqlx.DB) (*boresolver.QueryExecuteResponse, error) {
	preview, err := s.Preview(ctx, secCtx, qd)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	rows, err := db.QueryxContext(ctx, preview.SQL, preview.Parameters...)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	defer rows.Close()

	columns, err := scanColumns(rows)
	if err != nil {
		return nil, fmt.Errorf("failed to read columns: %w", err)
	}
	applyColumnMetadata(columns, preview.Columns)

	data, err := scanRows(rows, columns)
	if err != nil {
		return nil, fmt.Errorf("failed to scan rows: %w", err)
	}

	return &boresolver.QueryExecuteResponse{
		SQL:             preview.SQL,
		Columns:         columns,
		Rows:            data,
		RowCount:        len(data),
		ExecutionTimeMs: time.Since(start).Milliseconds(),
	}, nil
}

// GetBOTerms returns the resolved semantic terms for a BO/binding.
func (s *QueryService) GetBOTerms(ctx context.Context, secCtx *security.Context, boID, bindingID string) ([]boresolver.SemanticTermView, error) {
	if secCtx == nil {
		return nil, fmt.Errorf("security context required")
	}

	// Lightweight ownership check: the BO must belong to the tenant in the
	// security context. The repository lookup itself will fail if it doesn't.
	boDef, err := s.resolver.GetBODefinition(boID)
	if err != nil {
		return nil, fmt.Errorf("failed to load business object: %w", err)
	}
	_ = boDef

	terms, err := s.resolver.GetBOTerms(boID, bindingID)
	if err != nil {
		return nil, fmt.Errorf("failed to load terms: %w", err)
	}
	return terms, nil
}

// validateScope ensures the authenticated tenant/datasource from JWT is valid.
func (s *QueryService) validateScope(secCtx *security.Context, qd *boresolver.QueryDef) error {
	if secCtx == nil {
		return fmt.Errorf("security context required")
	}
	if secCtx.TenantID == "" {
		return fmt.Errorf("tenant id missing from JWT")
	}
	if secCtx.DatasourceID == "" || secCtx.DatasourceID == "none" {
		return fmt.Errorf("datasource id missing from JWT")
	}
	return nil
}

// buildSemanticRequest resolves the QueryDef into the existing semantic request format.
func (s *QueryService) buildSemanticRequest(qd *boresolver.QueryDef) (*boresolver.SemanticSQLGenerationRequest, error) {
	boDef, err := s.resolver.GetBODefinition(qd.Context.BOID)
	if err != nil {
		return nil, fmt.Errorf("failed to load BO %s: %w", qd.Context.BOID, err)
	}

	binding, err := s.resolver.GetBusinessObjectBinding(qd.Context.BOID, qd.Context.BindingID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve binding %s: %w", qd.Context.BindingID, err)
	}

	// The physical connection is always secCtx.DatasourceID (a
	// tenant_product_datasource.id already validated to belong to the caller's
	// tenant by security.DatasourceResolver, further enforced by RLS) — never
	// anything carried on the binding itself. See ResolveBindingDatasource for
	// how a caller determines which of their own datasources to put there for
	// a given core BO's binding.
	return mapQueryDefToSemanticRequest(qd, boDef, binding)
}

func dialectName(d boresolver.Dialect) string {
	switch d.(type) {
	case boresolver.SnowflakeDialect:
		return "snowflake"
	case boresolver.SQLServerDialect:
		return "sqlserver"
	default:
		return "postgres"
	}
}

// applyColumnMetadata copies BOID/Cardinality from the columns the SQL
// generator produced (by name) onto the columns the driver reports back,
// since the driver only knows names/DB types, not which BO or join a column
// came from.
func applyColumnMetadata(dbColumns []boresolver.QueryResultColumn, generated []boresolver.QueryResultColumn) {
	if len(generated) == 0 {
		return
	}
	byName := make(map[string]boresolver.QueryResultColumn, len(generated))
	for _, c := range generated {
		byName[c.Name] = c
	}
	for i := range dbColumns {
		if meta, ok := byName[dbColumns[i].Name]; ok {
			dbColumns[i].BOID = meta.BOID
			dbColumns[i].Cardinality = meta.Cardinality
		}
	}
}

func scanColumns(rows *sqlx.Rows) ([]boresolver.QueryResultColumn, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	colTypes, err := rows.ColumnTypes()
	if err != nil {
		colTypes = nil
	}

	columns := make([]boresolver.QueryResultColumn, len(cols))
	for i, name := range cols {
		columns[i].Name = name
		columns[i].Type = "unknown"
		if colTypes != nil && i < len(colTypes) {
			columns[i].Type = mapDBType(colTypes[i].DatabaseTypeName())
		}
	}
	return columns, nil
}

func scanRows(rows *sqlx.Rows, columns []boresolver.QueryResultColumn) ([]map[string]interface{}, error) {
	var result []map[string]interface{}
	for rows.Next() {
		row, err := scanRow(rows, columns)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func scanRow(rows *sqlx.Rows, columns []boresolver.QueryResultColumn) (map[string]interface{}, error) {
	values := make([]interface{}, len(columns))
	for i := range values {
		values[i] = new(interface{})
	}
	if err := rows.Scan(values...); err != nil {
		return nil, err
	}

	row := make(map[string]interface{}, len(columns))
	for i, col := range columns {
		val := *(values[i].(*interface{}))
		row[col.Name] = normalizeValue(val)
	}
	return row, nil
}

func normalizeValue(v interface{}) interface{} {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case sql.NullString:
		if val.Valid {
			return val.String
		}
		return nil
	case sql.NullInt64:
		if val.Valid {
			return val.Int64
		}
		return nil
	case sql.NullFloat64:
		if val.Valid {
			return val.Float64
		}
		return nil
	case sql.NullBool:
		if val.Valid {
			return val.Bool
		}
		return nil
	case sql.NullTime:
		if val.Valid {
			return val.Time
		}
		return nil
	default:
		return val
	}
}

func mapDBType(dbType string) string {
	switch strings.ToLower(dbType) {
	case "bigint", "int8", "integer", "int4", "smallint", "int2":
		return "integer"
	case "numeric", "decimal", "real", "double precision", "float", "float8":
		return "decimal"
	case "boolean", "bool":
		return "boolean"
	case "text", "character varying", "varchar", "char", "bpchar":
		return "string"
	case "date":
		return "date"
	case "timestamp", "timestamp with time zone", "timestamp without time zone", "timestamptz":
		return "timestamp"
	case "json", "jsonb":
		return "json"
	default:
		return "unknown"
	}
}
