package boresolver_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	br "github.com/hondyman/semlayer/backend/internal/boresolver"
)

// ============================================================================
// SEMANTIC TERM REPOSITORY TESTS
// ============================================================================

func TestSemanticTermRepository_GetSemanticTerm_Cached(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := br.NewSemanticTermRepository(sqlxDB)

	// Mock first query
	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "display_name", "description", "category", "is_system", "created_at", "updated_at",
	}).AddRow(
		"term-123", "tenant-1", "CUSTOMER_ADDRESS", "Customer Address", "Home address of customer", "party", false, "2025-01-01", "2025-01-01",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, tenant_id, name, display_name, description, category, is_system, created_at, updated_at
		 FROM semantic_terms
		 WHERE id = $1`,
	)).WithArgs("term-123").WillReturnRows(rows)

	// First call hits DB
	ctx := context.Background()
	term1, err := repo.GetSemanticTerm(ctx, "term-123")
	require.NoError(t, err)
	assert.Equal(t, "CUSTOMER_ADDRESS", term1.Name)

	// Second call should use cache (no new expectation)
	term2, err := repo.GetSemanticTerm(ctx, "term-123")
	require.NoError(t, err)
	assert.Equal(t, term1.ID, term2.ID)
	assert.Same(t, term1, term2) // Verify same pointer (cached)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestSemanticTermRepository_GetSemanticTerm_NotFound(t *testing.T) {
	db, mock, mockErr := sqlmock.New()
	require.NoError(t, mockErr)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := br.NewSemanticTermRepository(sqlxDB)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, tenant_id, name, display_name, description, category, is_system, created_at, updated_at
		 FROM semantic_terms
		 WHERE id = $1`,
	)).WithArgs("nonexistent").WillReturnError(fmt.Errorf("no rows"))

	ctx := context.Background()
	_, err := repo.GetSemanticTerm(ctx, "nonexistent")
	assert.Error(t, err)

	require.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================================
// CATALOG REPOSITORY TESTS
// ============================================================================

func TestCatalogRepository_GetNode_Cached(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := br.NewCatalogRepository(sqlxDB)

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "type", "name", "parent_id", "metadata", "created_at", "updated_at",
	}).AddRow(
		"node-col-1", "tenant-1", "column", "address", nil, "", "2025-01-01", "2025-01-01",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, tenant_id, type, name, parent_id, metadata, created_at, updated_at
		 FROM catalog_nodes
		 WHERE id = $1`,
	)).WithArgs("node-col-1").WillReturnRows(rows)

	ctx := context.Background()
	node, err := repo.GetNode(ctx, "node-col-1")
	require.NoError(t, err)
	assert.Equal(t, "address", node.Name)

	// Second call uses cache
	node2, err := repo.GetNode(ctx, "node-col-1")
	require.NoError(t, err)
	assert.Same(t, node, node2)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCatalogRepository_GetEdges_Cached(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := br.NewCatalogRepository(sqlxDB)

	rows := sqlmock.NewRows([]string{
		"id", "tenant_id", "from_id", "to_id", "type", "datasource_id", "metadata", "created_at",
	}).AddRow(
		"edge-1", "tenant-1", "term-123", "node-col-1", "TERM_MAPS_TO_COLUMN", "ds-postgres", "", "2025-01-01",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, tenant_id, from_id, to_id, type, datasource_id, metadata, created_at
		 FROM catalog_edges
		 WHERE from_id = $1 AND datasource_id = $2`,
	)).WithArgs("term-123", "ds-postgres").WillReturnRows(rows)

	ctx := context.Background()
	edges, err := repo.GetEdges(ctx, "term-123", "ds-postgres")
	require.NoError(t, err)
	require.Len(t, edges, 1)
	assert.Equal(t, "TERM_MAPS_TO_COLUMN", edges[0].Type)

	// Second call uses cache (no new expectation)
	edges2, err := repo.GetEdges(ctx, "term-123", "ds-postgres")
	require.NoError(t, err)
	assert.Equal(t, edges, edges2)

	require.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================================
// BUSINESS OBJECT REPOSITORY TESTS
// ============================================================================

func TestBusinessObjectCachedRepository_GetFieldsForBO_WithCache(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	repo := br.NewBusinessObjectCachedRepository(sqlxDB)

	rows := sqlmock.NewRows([]string{
		"id", "name", "technical_name", "business_object_id", "semantic_term_id", "physical_table", "physical_column", "type", "is_required",
	}).
		AddRow("f1", "address", "address", "bo-1", "term-addr", nil, nil, "string", false).
		AddRow("f2", "name", "name", "bo-1", "term-name", nil, nil, "string", true)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, technical_name, business_object_id, semantic_term_id, NULL::text AS physical_table, NULL::text AS physical_column, type, is_required
		 FROM bo_fields
		 WHERE business_object_id = $1
		 ORDER BY id`,
	)).WithArgs("bo-1").WillReturnRows(rows)

	ctx := context.Background()
	fields, err := repo.GetFieldsForBO(ctx, "bo-1")
	require.NoError(t, err)
	require.Len(t, fields, 2)

	// Verify both fields and fieldsByBO cache were populated
	field, err := repo.GetFieldByID(ctx, "f1")
	require.NoError(t, err)
	assert.Equal(t, "address", field.Name)

	// Second call to GetFieldsForBO should use cache (no new expectation)
	fields2, err := repo.GetFieldsForBO(ctx, "bo-1")
	require.NoError(t, err)
	assert.Equal(t, fields, fields2)

	require.NoError(t, mock.ExpectationsWereMet())
}

// ============================================================================
// FIELD RESOLVER TESTS
// ============================================================================

func TestFieldResolver_ResolveFieldToPhysical_WithOverride(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	boRepo := br.NewBusinessObjectCachedRepository(sqlxDB)
	semanticRepo := br.NewSemanticTermRepository(sqlxDB)
	catalogRepo := br.NewCatalogRepository(sqlxDB)
	resolver := br.NewFieldResolver(boRepo, semanticRepo, catalogRepo)

	// Mock GetFieldByID - return field with physical override
	tableOverride := "custom_table"
	columnOverride := "custom_column"

	fieldRows := sqlmock.NewRows([]string{
		"id", "name", "technical_name", "business_object_id", "semantic_term_id", "physical_table", "physical_column", "type", "is_required",
	}).AddRow(
		"f1", "address", "address", "bo-1", "term-addr", tableOverride, columnOverride, "string", false,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, technical_name, business_object_id, semantic_term_id, NULL::text AS physical_table, NULL::text AS physical_column, type, is_required
		 FROM bo_fields
		 WHERE id = $1`,
	)).WithArgs("f1").WillReturnRows(fieldRows)

	ctx := context.Background()
	resolved, err := resolver.ResolveFieldToPhysical(ctx, "f1", "ds-postgres")
	require.NoError(t, err)

	assert.Equal(t, "f1", resolved.FieldID)
	assert.Equal(t, "address", resolved.FieldName)
	assert.Equal(t, "custom_table", resolved.Table)
	assert.Equal(t, "custom_column", resolved.Column)
	assert.Equal(t, "OVERRIDE", resolved.SourceType)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFieldResolver_ResolveFieldToPhysical_ViaSemanticTerm(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	boRepo := br.NewBusinessObjectCachedRepository(sqlxDB)
	semanticRepo := br.NewSemanticTermRepository(sqlxDB)
	catalogRepo := br.NewCatalogRepository(sqlxDB)
	resolver := br.NewFieldResolver(boRepo, semanticRepo, catalogRepo)

	// Mock GetFieldByID - no override, has semantic term
	fieldRows := sqlmock.NewRows([]string{
		"id", "name", "technical_name", "business_object_id", "semantic_term_id", "physical_table", "physical_column", "type", "is_required",
	}).AddRow(
		"f1", "address", "address", "bo-1", "term-addr", nil, nil, "string", false,
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, technical_name, business_object_id, semantic_term_id, NULL::text AS physical_table, NULL::text AS physical_column, type, is_required
		 FROM bo_fields
		 WHERE id = $1`,
	)).WithArgs("f1").WillReturnRows(fieldRows)

	// Mock GetSemanticTerm
	termRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "name", "display_name", "description", "category", "is_system", "created_at", "updated_at",
	}).AddRow(
		"term-addr", "tenant-1", "CUSTOMER_ADDRESS", "Customer Address", "Home address", "party", false, "2025-01-01", "2025-01-01",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, tenant_id, name, display_name, description, category, is_system, created_at, updated_at
		 FROM semantic_terms
		 WHERE id = $1`,
	)).WithArgs("term-addr").WillReturnRows(termRows)

	// Mock GetEdges
	edgeRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "from_id", "to_id", "type", "datasource_id", "metadata", "created_at",
	}).AddRow(
		"edge-1", "tenant-1", "term-addr", "node-col", "TERM_MAPS_TO_COLUMN", "ds-postgres", "", "2025-01-01",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, tenant_id, from_id, to_id, type, datasource_id, metadata, created_at
		 FROM catalog_edges
		 WHERE from_id = $1 AND datasource_id = $2`,
	)).WithArgs("term-addr", "ds-postgres").WillReturnRows(edgeRows)

	// Mock GetNode (column)
	colNodeRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "type", "name", "parent_id", "metadata", "created_at", "updated_at",
	}).AddRow(
		"node-col", "tenant-1", "column", "customer_address", "node-tbl", "", "2025-01-01", "2025-01-01",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, tenant_id, type, name, parent_id, metadata, created_at, updated_at
		 FROM catalog_nodes
		 WHERE id = $1`,
	)).WithArgs("node-col").WillReturnRows(colNodeRows)

	// Mock GetNode (table)
	tblNodeRows := sqlmock.NewRows([]string{
		"id", "tenant_id", "type", "name", "parent_id", "metadata", "created_at", "updated_at",
	}).AddRow(
		"node-tbl", "tenant-1", "table", "customers", nil, "", "2025-01-01", "2025-01-01",
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, tenant_id, type, name, parent_id, metadata, created_at, updated_at
		 FROM catalog_nodes
		 WHERE id = $1`,
	)).WithArgs("node-tbl").WillReturnRows(tblNodeRows)

	ctx := context.Background()
	resolved, err := resolver.ResolveFieldToPhysical(ctx, "f1", "ds-postgres")
	require.NoError(t, err)

	assert.Equal(t, "f1", resolved.FieldID)
	assert.Equal(t, "address", resolved.FieldName)
	assert.Equal(t, "customers", resolved.Table)
	assert.Equal(t, "customer_address", resolved.Column)
	assert.Equal(t, "SEMANTIC", resolved.SourceType)

	require.NoError(t, mock.ExpectationsWereMet())
}

func TestFieldResolver_ResolveFieldToPhysical_NoSemanticOrOverride(t *testing.T) {
	db, mock, mockErr := sqlmock.New()
	require.NoError(t, mockErr)
	defer db.Close()

	sqlxDB := sqlx.NewDb(db, "postgres")
	boRepo := br.NewBusinessObjectCachedRepository(sqlxDB)
	semanticRepo := br.NewSemanticTermRepository(sqlxDB)
	catalogRepo := br.NewCatalogRepository(sqlxDB)
	resolver := br.NewFieldResolver(boRepo, semanticRepo, catalogRepo)

	// Mock GetFieldByID - no override, no semantic term
	fieldRows := sqlmock.NewRows([]string{
		"id", "name", "technical_name", "business_object_id", "semantic_term_id", "physical_table", "physical_column", "type", "is_required",
	}).AddRow(
		"f1", "address", "address", "bo-1", "", nil, nil, "string", false, // Empty semantic_term_id
	)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT id, name, technical_name, business_object_id, semantic_term_id, NULL::text AS physical_table, NULL::text AS physical_column, type, is_required
		 FROM bo_fields
		 WHERE id = $1`,
	)).WithArgs("f1").WillReturnRows(fieldRows)

	ctx := context.Background()
	_, err := resolver.ResolveFieldToPhysical(ctx, "f1", "ds-postgres")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no semantic term and no physical override")

	require.NoError(t, mock.ExpectationsWereMet())
}
