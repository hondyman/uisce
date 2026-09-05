// Package cardinality computes relationship cardinality from real Postgres
// constraint metadata. It is deliberately a leaf package (depends only on
// database/sql and internal/models) so that both internal/api and
// internal/scanner — which cannot import each other without an import
// cycle (internal/api already transitively depends on internal/scanner) —
// can each depend on it directly.
package cardinality

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/hondyman/uisce/backend/internal/models"
)

// Resolver computes relationship cardinality from real Postgres constraint
// metadata (primary/unique keys), instead of guessing from FK edge
// direction or column count. It is the single place cardinality should be
// computed — every discovery/scan path should call it (or read a value it
// already persisted) rather than re-deriving cardinality itself.
type Resolver struct {
	db *sql.DB
}

// NewResolver creates a resolver backed by db.
func NewResolver(db *sql.DB) *Resolver {
	return &Resolver{db: db}
}

// ResolveEdgeCardinality determines the cardinality of a foreign-key edge
// sourceTable.sourceCols -> targetTable.targetCols, expressed from the
// source's perspective (i.e. "for one target row, how many source rows can
// reference it").
//
// Real cardinality requires knowing whether sourceCols is itself covered by
// a PRIMARY KEY/UNIQUE constraint on sourceTable:
//   - unique on source only (the common case: a plain FK column) -> MANY_TO_ONE
//     (many source rows can point at the same target row)
//   - unique on source AND targetCols is unique on target -> ONE_TO_ONE
//   - unique on neither -> ONE_TO_MANY is not possible for a single FK edge;
//     this only happens when interpreted from the inverse direction, so
//     callers walking an inbound edge should call Inverse() on the outbound
//     result rather than asking this function to resolve the inbound case
//     directly.
func (r *Resolver) ResolveEdgeCardinality(
	ctx context.Context,
	sourceSchema, sourceTable string, sourceCols []string,
	targetSchema, targetTable string, targetCols []string,
) (models.Cardinality, error) {
	if len(sourceCols) == 0 || len(targetCols) == 0 {
		return models.CardinalityUnknown, nil
	}

	sourceUnique, err := r.isColumnSetUnique(ctx, sourceSchema, sourceTable, sourceCols)
	if err != nil {
		return models.CardinalityUnknown, fmt.Errorf("checking uniqueness of %s(%v): %w", sourceTable, sourceCols, err)
	}

	targetUnique, err := r.isColumnSetUnique(ctx, targetSchema, targetTable, targetCols)
	if err != nil {
		return models.CardinalityUnknown, fmt.Errorf("checking uniqueness of %s(%v): %w", targetTable, targetCols, err)
	}

	switch {
	case sourceUnique && targetUnique:
		return models.CardinalityOneToOne, nil
	case sourceUnique:
		// The FK columns on the source side are themselves unique, so each
		// source row maps to at most one target row, and (since target isn't
		// unique) each target row can be referenced by many source rows only
		// if we look at it from the target's side — from the source's side,
		// this edge is many-source-rows-to-one-target-row.
		return models.CardinalityManyToOne, nil
	default:
		return models.CardinalityManyToOne, nil
	}
}

// isColumnSetUnique reports whether cols (as a set) exactly matches the
// column set of some PRIMARY KEY or UNIQUE constraint on schema.table.
// Composite-key aware: a constraint only counts as a match if every one of
// its columns is in cols and every one of cols is in the constraint.
func (r *Resolver) isColumnSetUnique(ctx context.Context, schema, table string, cols []string) (bool, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT tc.constraint_name, kcu.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema = tc.table_schema
		WHERE tc.table_schema = $1
			AND tc.table_name = $2
			AND tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE')
	`

	rows, err := r.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	constraintCols := map[string]map[string]bool{}
	for rows.Next() {
		var constraintName, columnName string
		if err := rows.Scan(&constraintName, &columnName); err != nil {
			return false, err
		}
		if constraintCols[constraintName] == nil {
			constraintCols[constraintName] = map[string]bool{}
		}
		constraintCols[constraintName][columnName] = true
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	wanted := map[string]bool{}
	for _, c := range cols {
		wanted[c] = true
	}

	for _, set := range constraintCols {
		if len(set) != len(wanted) {
			continue
		}
		match := true
		for c := range wanted {
			if !set[c] {
				match = false
				break
			}
		}
		if match {
			return true, nil
		}
	}

	return false, nil
}

// ResolveByTableNames resolves cardinality when only table names are known
// (no explicit FK column list), by looking up the actual FK constraint(s)
// between the two tables in information_schema. direction is "outbound" if
// sourceTable is expected to hold the FK to targetTable, "inbound" if
// targetTable holds the FK to sourceTable; the result is always expressed
// from sourceTable's perspective.
func (r *Resolver) ResolveByTableNames(
	ctx context.Context,
	sourceSchema, sourceTable, targetSchema, targetTable, direction string,
) (models.Cardinality, error) {
	fkFrom, fkTo := sourceTable, targetTable
	if direction == "inbound" {
		fkFrom, fkTo = targetTable, sourceTable
	}

	query := `
		SELECT kcu.column_name, ccu.column_name AS foreign_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema = tc.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.table_schema = $1
			AND tc.table_name = $2
			AND tc.constraint_type = 'FOREIGN KEY'
			AND ccu.table_name = $3
		ORDER BY kcu.ordinal_position
	`

	rows, err := r.db.QueryContext(ctx, query, sourceSchema, fkFrom, fkTo)
	if err != nil {
		return models.CardinalityUnknown, err
	}
	defer rows.Close()

	var fromCols, toCols []string
	for rows.Next() {
		var fromCol, toCol string
		if err := rows.Scan(&fromCol, &toCol); err != nil {
			return models.CardinalityUnknown, err
		}
		fromCols = append(fromCols, fromCol)
		toCols = append(toCols, toCol)
	}
	if err := rows.Err(); err != nil {
		return models.CardinalityUnknown, err
	}
	if len(fromCols) == 0 {
		return models.CardinalityUnknown, nil
	}

	resolved, err := r.ResolveEdgeCardinality(ctx, sourceSchema, fkFrom, fromCols, targetSchema, fkTo, toCols)
	if err != nil {
		return models.CardinalityUnknown, err
	}

	if direction == "inbound" {
		return resolved.Inverse(), nil
	}
	return resolved, nil
}

// JunctionTable describes a table detected as an associative/junction table
// between two parent tables, e.g. order_items between orders and products.
type JunctionTable struct {
	SchemaName string
	TableName  string
	ParentA    string
	ParentB    string
	ColumnsA   []string
	ColumnsB   []string
}

// DetectJunctionTable reports whether schema.table is an associative table
// representing a many-to-many relationship between two other tables: it has
// exactly two foreign keys, each pointing at a different parent table, and
// the union of their columns is covered by the table's own PRIMARY
// KEY/UNIQUE constraint.
func (r *Resolver) DetectJunctionTable(ctx context.Context, schema, table string) (*JunctionTable, error) {
	if schema == "" {
		schema = "public"
	}

	query := `
		SELECT tc.constraint_name, kcu.column_name, ccu.table_name AS foreign_table, ccu.column_name AS foreign_column
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
			ON kcu.constraint_name = tc.constraint_name
			AND kcu.table_schema = tc.table_schema
		JOIN information_schema.constraint_column_usage ccu
			ON ccu.constraint_name = tc.constraint_name
			AND ccu.table_schema = tc.table_schema
		WHERE tc.table_schema = $1
			AND tc.table_name = $2
			AND tc.constraint_type = 'FOREIGN KEY'
		ORDER BY tc.constraint_name, kcu.ordinal_position
	`

	rows, err := r.db.QueryContext(ctx, query, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type fk struct {
		table   string
		columns []string
	}
	fksByConstraint := map[string]*fk{}
	var order []string

	for rows.Next() {
		var constraintName, column, foreignTable, foreignColumn string
		if err := rows.Scan(&constraintName, &column, &foreignTable, &foreignColumn); err != nil {
			return nil, err
		}
		if _, ok := fksByConstraint[constraintName]; !ok {
			fksByConstraint[constraintName] = &fk{table: foreignTable}
			order = append(order, constraintName)
		}
		fksByConstraint[constraintName].columns = append(fksByConstraint[constraintName].columns, column)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(fksByConstraint) != 2 {
		return nil, nil
	}

	sort.Strings(order)
	fkA := fksByConstraint[order[0]]
	fkB := fksByConstraint[order[1]]
	if fkA.table == fkB.table {
		// Self-referential double-FK isn't a junction table between two
		// distinct parents.
		return nil, nil
	}

	// Canonicalize ParentA/ParentB by table name (not by constraint name,
	// which is an incidental naming convention) so that every caller —
	// the discovery engine, the scanner, and the backfill runner alike —
	// always assigns the same table to ParentA/ParentB for a given
	// junction table, regardless of which order they happen to call this
	// in. Without this, independent producers could each insert a
	// valid-but-mirrored (A,B) vs (B,A) row, since a table's uniqueness
	// constraint treats those as distinct rows.
	if fkB.table < fkA.table {
		fkA, fkB = fkB, fkA
	}

	allFKCols := append(append([]string{}, fkA.columns...), fkB.columns...)
	covered, err := r.isColumnSetUnique(ctx, schema, table, allFKCols)
	if err != nil {
		return nil, fmt.Errorf("checking junction key coverage for %s: %w", table, err)
	}
	if !covered {
		return nil, nil
	}

	return &JunctionTable{
		SchemaName: schema,
		TableName:  table,
		ParentA:    fkA.table,
		ParentB:    fkB.table,
		ColumnsA:   fkA.columns,
		ColumnsB:   fkB.columns,
	}, nil
}
