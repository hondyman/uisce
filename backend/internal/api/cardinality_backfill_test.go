package api

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
)

// TestBackfillJunctionRelationships_SourcesCandidatesFromCatalogEdge proves
// the fix for a real bootstrap gap found in review: candidate-table
// discovery must read from catalog_edge (what the live scanner,
// internal/scanner/ansi_scanner.go, actually populates), not from
// entity_relationship — which has no live discovery-time writer at all
// (its only would-be writer, enhanced_relationship_discovery.go's
// SaveDiscoveredRelationship, is dead code with zero callers). Sourcing
// candidates from entity_relationship would make this backfill a no-op on
// every environment that hasn't already had it run at least once.
func TestBackfillJunctionRelationships_SourcesCandidatesFromCatalogEdge(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// The candidate-discovery query must hit catalog_edge, not
	// entity_relationship. If a future change reverts to sourcing from
	// entity_relationship, this mock's query pattern won't match and the
	// test fails with an unexpected-query error.
	mock.ExpectQuery("FROM public\\.catalog_edge").
		WillReturnRows(sqlmock.NewRows([]string{"table_name"}))

	resolver := NewCardinalityResolver(db)
	inserted, err := BackfillJunctionRelationships(context.Background(), db, resolver)
	require.NoError(t, err)
	require.Equal(t, 0, inserted)
	require.NoError(t, mock.ExpectationsWereMet())
}

// TestBackfillJunctionRelationships_FiltersToForeignKeyEdges proves the
// candidate query restricts to edge_type_name='foreign_key'. Without this
// filter, the synthesized many_to_many_junction edges (which also carry
// source_table/target_table — there naming the two PARENT tables, not the
// junction table) would surface as bogus extra candidates. sqlmock's
// regex match on the query text is what actually enforces this here: if
// the filter clause is removed, this mock's stricter pattern stops
// matching and the test fails with an unexpected-query error.
func TestBackfillJunctionRelationships_FiltersToForeignKeyEdges(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("FROM public\\.catalog_edge\\s+WHERE properties->>'edge_type_name' = 'foreign_key'").
		WillReturnRows(sqlmock.NewRows([]string{"table_name"}))

	resolver := NewCardinalityResolver(db)
	_, err = BackfillJunctionRelationships(context.Background(), db, resolver)
	require.NoError(t, err)
	require.NoError(t, mock.ExpectationsWereMet())
}
