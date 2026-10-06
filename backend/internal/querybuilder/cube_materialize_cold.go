package querybuilder

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// defaultIcebergCatalog is the StarRocks external catalog used for cube cold
// tables. Override with CUBE_ICEBERG_CATALOG (must already exist / be configured
// against Lakekeeper for the deploy environment).
func defaultIcebergCatalog() string {
	if v := strings.TrimSpace(os.Getenv("CUBE_ICEBERG_CATALOG")); v != "" {
		return sanitizeIdentifier(v)
	}
	return "iceberg_catalog"
}

// starRocksColdWriter commits cold Iceberg via StarRocks FE:
// CREATE DATABASE → DROP TABLE (replace) → CREATE TABLE … AS SELECT * FROM hot.
type starRocksColdWriter struct {
	m *CubeMaterializer
}

func (w *starRocksColdWriter) ApplyCold(
	ctx context.Context,
	plan *CubeMaterializePlan,
	hot *CubeMaterializeHotResult,
) (*CubeMaterializeColdResult, error) {
	if w == nil || w.m == nil {
		return nil, fmt.Errorf("cube cold writer: not configured")
	}
	if plan == nil {
		return nil, fmt.Errorf("cube cold writer: plan is required")
	}
	db := w.m.starrocksDB
	if db == nil {
		return nil, fmt.Errorf("starrocks connection is not available (check STARROCKS_HOST/PORT/USER/PASSWORD)")
	}

	cat := sanitizeIdentifier(plan.IcebergCatalog)
	schema := sanitizeIdentifier(plan.IcebergDatabase)
	table := sanitizeIdentifier(plan.MaterializationName)
	if cat == "" || schema == "" || table == "" {
		return nil, fmt.Errorf("cube cold writer: iceberg catalog/database/table are required")
	}

	// Track B: refresh Lakekeeper bearer on the StarRocks catalog before any
	// Iceberg DDL. Nil refresher (gate off / tests) skips.
	if w.m.catalogAuth != nil {
		if err := w.m.catalogAuth.Ensure(ctx, cat); err != nil {
			return nil, err
		}
	}

	hotQualified := fmt.Sprintf("%s.%s",
		quoteStarRocksIdent(plan.TargetDatabase),
		quoteStarRocksIdent(plan.MaterializationName),
	)
	dbQualified := fmt.Sprintf("%s.%s", quoteStarRocksIdent(cat), quoteStarRocksIdent(schema))
	tableQualified := fmt.Sprintf("%s.%s", dbQualified, quoteStarRocksIdent(table))

	stmts := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbQualified),
		// Full replace for this grain: one CREATE TABLE AS SELECT = one Iceberg commit.
		fmt.Sprintf("DROP TABLE IF EXISTS %s", tableQualified),
		fmt.Sprintf(
			`CREATE TABLE %s PROPERTIES ("file_format" = "parquet") AS SELECT * FROM %s`,
			tableQualified, hotQualified,
		),
	}
	for i, q := range stmts {
		if _, err := db.ExecContext(ctx, q); err != nil {
			// Best-effort cleanup of a partial cold table so reconcile sees no half-commit.
			_, _ = db.ExecContext(ctx, fmt.Sprintf("DROP TABLE IF EXISTS %s", tableQualified))
			return nil, fmt.Errorf("iceberg cold commit step %d/%d: %w", i+1, len(stmts), err)
		}
	}

	rowCount := int64(0)
	if hot != nil {
		rowCount = hot.RowCount
	}
	_ = db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", tableQualified)).Scan(&rowCount)

	return &CubeMaterializeColdResult{
		IcebergTable: fmt.Sprintf("%s.%s.%s", cat, schema, table),
		Applied:      true,
		RowCount:     rowCount,
		CommittedAt:  time.Now().UTC(),
	}, nil
}
