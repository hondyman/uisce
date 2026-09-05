package api

import (
	"context"
	"database/sql"
	"log"

	"github.com/hondyman/uisce/backend/internal/models"
)

// ReresolveExistingRelationships re-computes cardinality for every active
// entity_relationship row that has enough column information to resolve
// (source_table/source_column/target_table/target_column), and updates the
// row when the resolved value differs from what's stored. Returns the
// number of rows updated.
func ReresolveExistingRelationships(ctx context.Context, db *sql.DB, resolver *CardinalityResolver) (int, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, source_table, source_column, target_table, target_column, COALESCE(cardinality, '')
		FROM public.entity_relationship
		WHERE is_active = true
		  AND source_table IS NOT NULL AND source_table != ''
		  AND target_table IS NOT NULL AND target_table != ''
		  AND source_column IS NOT NULL AND source_column != ''
		  AND target_column IS NOT NULL AND target_column != ''
	`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type row struct {
		id                                                   string
		sourceTable, sourceColumn, targetTable, targetColumn string
		oldCardinality                                       string
	}
	var candidates []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.sourceTable, &r.sourceColumn, &r.targetTable, &r.targetColumn, &r.oldCardinality); err != nil {
			return 0, err
		}
		candidates = append(candidates, r)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}

	updated := 0
	for _, r := range candidates {
		resolved, err := resolver.ResolveEdgeCardinality(
			ctx, "public", r.sourceTable, []string{r.sourceColumn},
			"public", r.targetTable, []string{r.targetColumn},
		)
		if err != nil {
			log.Printf("skipping relationship %s (%s.%s -> %s.%s): %v", r.id, r.sourceTable, r.sourceColumn, r.targetTable, r.targetColumn, err)
			continue
		}
		if resolved == models.CardinalityUnknown || string(resolved) == r.oldCardinality {
			continue
		}
		if _, err := db.ExecContext(ctx,
			`UPDATE public.entity_relationship SET cardinality = $1, updated_at = now() WHERE id = $2`,
			string(resolved), r.id,
		); err != nil {
			return updated, err
		}
		updated++
	}

	return updated, nil
}

// BackfillJunctionRelationships scans every table referenced by an existing
// entity_relationship row for the junction-table pattern, and inserts a
// synthesized MANY_TO_MANY relationship between the two parent entities
// when both are already registered (via entity_attribute_column_mapping).
//
// Idempotent: the insert's ON CONFLICT target
// (tenant_datasource_id, source_entity_id, target_entity_id,
// relationship_type) is exactly entity_relationship's own
// entity_relationship_unique constraint (006_relationship_discovery_schema.sql),
// so re-running this against the same data updates the existing synthesized
// row in place rather than erroring or duplicating it — see
// TestBackfillJunctionRelationships_IsIdempotent, which proves this by
// running it twice.
func BackfillJunctionRelationships(ctx context.Context, db *sql.DB, resolver *CardinalityResolver) (int, error) {
	// Candidate tables come from catalog_edge, not entity_relationship:
	// entity_relationship has no live discovery-time writer (the only
	// process that ever populated it, enhanced_relationship_discovery.go's
	// SaveDiscoveredRelationship, is dead code with zero callers), so on
	// any environment that hasn't already had this backfill run at least
	// once, entity_relationship starts empty and sourcing candidates from
	// it would find nothing — a no-op backfill. catalog_edge is what the
	// live scanner (internal/scanner/ansi_scanner.go) actually populates,
	// so that's the real source of "tables we know about."
	// Restricted to edge_type_name='foreign_key': the synthesized
	// many_to_many_junction edges (internal/scanner/ansi_scanner.go's
	// detectAndAppendJunctionEdges) ALSO carry source_table/target_table —
	// there they name the two PARENT tables, not the junction table itself —
	// so without this filter they'd surface as bogus extra candidates
	// (harmless — DetectJunctionTable just returns nil for a non-junction
	// table — but wasted queries, and a real risk if any other catalog_edge
	// writer ever reuses these property key names with different meaning).
	tableRows, err := db.QueryContext(ctx, `
		SELECT DISTINCT properties->>'source_table' AS table_name
		FROM public.catalog_edge
		WHERE properties->>'edge_type_name' = 'foreign_key'
		  AND properties ? 'source_table' AND properties->>'source_table' != ''
		UNION
		SELECT DISTINCT properties->>'target_table' AS table_name
		FROM public.catalog_edge
		WHERE properties->>'edge_type_name' = 'foreign_key'
		  AND properties ? 'target_table' AND properties->>'target_table' != ''
	`)
	if err != nil {
		return 0, err
	}
	var tables []string
	for tableRows.Next() {
		var t string
		if err := tableRows.Scan(&t); err != nil {
			tableRows.Close()
			return 0, err
		}
		tables = append(tables, t)
	}
	if err := tableRows.Err(); err != nil {
		tableRows.Close()
		return 0, err
	}
	tableRows.Close()

	inserted := 0
	for _, table := range tables {
		junction, err := resolver.DetectJunctionTable(ctx, "public", table)
		if err != nil {
			log.Printf("junction detection failed for %s: %v", table, err)
			continue
		}
		if junction == nil {
			continue
		}

		parentA, okA := lookupEntityForTable(ctx, db, junction.ParentA)
		parentB, okB := lookupEntityForTable(ctx, db, junction.ParentB)
		if !okA || !okB {
			log.Printf("junction %s detected but parent entities not both registered (%s, %s) — skipping", table, junction.ParentA, junction.ParentB)
			continue
		}
		if parentA.tenantDatasourceID != parentB.tenantDatasourceID {
			// Parents scanned from different datasources aren't a coherent
			// M:N relationship for this tenant/datasource-scoped table.
			continue
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO public.entity_relationship (
				tenant_id, tenant_datasource_id,
				source_entity_id, target_entity_id,
				relationship_type, cardinality, hierarchy_depth,
				fk_constraint, source_table, target_table,
				confidence, confidence_reason,
				source_discovery_method, is_active,
				description
			) VALUES ($1, $2, $3, $4, 'MANY_TO_MANY_JUNCTION', $5, 1, $6, $7, $8, 1.0, $9, 'JUNCTION_TABLE_ANALYSIS', true, $10)
			ON CONFLICT (tenant_datasource_id, source_entity_id, target_entity_id, relationship_type)
			DO UPDATE SET
				cardinality = EXCLUDED.cardinality,
				fk_constraint = EXCLUDED.fk_constraint,
				updated_at = now()
		`,
			parentA.tenantID, parentA.tenantDatasourceID,
			parentA.id, parentB.id,
			string(models.CardinalityManyToMany),
			"junction table: "+junction.TableName,
			junction.ParentA, junction.ParentB,
			"FK pair covered by junction table's own PRIMARY KEY/UNIQUE constraint",
			junction.ParentA+" <-> "+junction.ParentB+" via "+junction.TableName,
		)
		if err != nil {
			return inserted, err
		}
		inserted++
	}

	return inserted, nil
}

type entityRef struct {
	id                 string
	tenantID           string
	tenantDatasourceID string
}

// lookupEntityForTable finds the entity_attribute registered as backing
// tableName, via entity_attribute_column_mapping. When multiple entities
// map to the same table (shouldn't normally happen), the first match wins.
func lookupEntityForTable(ctx context.Context, db *sql.DB, tableName string) (entityRef, bool) {
	var ref entityRef
	err := db.QueryRowContext(ctx, `
		SELECT ea.id, ea.tenant_id, ea.tenant_datasource_id
		FROM public.entity_attribute_column_mapping eacm
		JOIN public.entity_attribute ea ON ea.id = eacm.entity_attribute_id
		WHERE eacm.table_name = $1
		LIMIT 1
	`, tableName).Scan(&ref.id, &ref.tenantID, &ref.tenantDatasourceID)
	if err != nil {
		return entityRef{}, false
	}
	return ref, true
}
