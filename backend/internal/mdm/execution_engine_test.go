package mdm

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

func TestExecutionEngine_RecursiveResolution(t *testing.T) {
	ctx := context.Background()

	// Setup in-memory DB for SemanticGraphService
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create necessary tables
	db.MustExec(`
		CREATE TABLE catalog_node_type (
			id UUID PRIMARY KEY,
			catalog_type_name TEXT,
			node_type TEXT,
			tenant_id UUID
		)
	`)
	db.MustExec(`
		CREATE TABLE catalog_node (
			id UUID PRIMARY KEY,
			node_name TEXT,
			description TEXT,
			properties TEXT,
			config TEXT,
			node_type_id UUID,
			tenant_id UUID,
			qualified_path TEXT
		)
	`)
	db.MustExec(`
		CREATE TABLE catalog_edge (
			id UUID PRIMARY KEY,
			source_node_id UUID,
			target_node_id UUID,
			edge_type_name TEXT,
			tenant_id UUID
		)
	`)

	graphService := analytics.NewSemanticGraphService(db)
	engine, err := NewExecutionEngine(ctx, graphService, nil)
	assert.NoError(t, err)
	defer engine.Close(ctx)

	// Seed node types
	calcTypeID := uuid.New()
	termTypeID := uuid.New()
	tenantID := uuid.New()
	db.MustExec("INSERT INTO catalog_node_type (id, catalog_type_name, node_type, tenant_id) VALUES (?, ?, ?, ?)", calcTypeID, "calculation_term", "calculation_term", tenantID)
	db.MustExec("INSERT INTO catalog_node_type (id, catalog_type_name, node_type, tenant_id) VALUES (?, ?, ?, ?)", termTypeID, "semantic_term", "semantic_term", tenantID)

	// Seed nodes
	navID := uuid.New()
	posValID := uuid.New()
	priceID := uuid.New()

	db.MustExec("INSERT INTO catalog_node (id, node_name, properties, config, node_type_id, tenant_id, qualified_path) VALUES (?, ?, ?, ?, ?, ?, ?)",
		navID, "NetAssetValue", `{"engine":"mock", "expression":"sum"}`, `{}`, calcTypeID, tenantID, "calc/nav")
	db.MustExec("INSERT INTO catalog_node (id, node_name, properties, config, node_type_id, tenant_id, qualified_path) VALUES (?, ?, ?, ?, ?, ?, ?)",
		posValID, "PositionValue", `{"engine":"mock", "expression":"sum"}`, `{}`, calcTypeID, tenantID, "calc/pos")
	db.MustExec("INSERT INTO catalog_node (id, node_name, properties, config, node_type_id, tenant_id, qualified_path) VALUES (?, ?, ?, ?, ?, ?, ?)",
		priceID, "MarketPrice", `{}`, `{}`, termTypeID, tenantID, "term/price")

	// Seed edges
	db.MustExec("INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_name, tenant_id) VALUES (?, ?, ?, ?, ?)",
		uuid.New(), navID, posValID, "calc_depends_on_calc", tenantID)
	db.MustExec("INSERT INTO catalog_edge (id, source_node_id, target_node_id, edge_type_name, tenant_id) VALUES (?, ?, ?, ?, ?)",
		uuid.New(), posValID, priceID, "calc_depends_on_term", tenantID)

	// Execute
	context := map[string]interface{}{
		"MarketPrice": 150.0,
	}

	result, trace, err := engine.ExecuteCalculation(ctx, navID, context)

	assert.NoError(t, err)
	assert.Equal(t, 150.0, result)
	assert.NotNil(t, trace)
	assert.Equal(t, "NetAssetValue", trace.TermName)
	assert.Contains(t, trace.Dependencies, "PositionValue")
	assert.Contains(t, trace.Dependencies["PositionValue"].Dependencies, "MarketPrice")
}
