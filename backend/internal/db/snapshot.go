package db

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Node represents a single node in a catalog snapshot.
type Node struct {
	ID            uuid.UUID       `db:"id"`
	CoreID        *uuid.UUID      `db:"core_id"`
	NodeName      string          `db:"node_name"`
	TypeName      string          `db:"type_name"`
	QualifiedPath string          `db:"qualified_path"`
	Properties    json.RawMessage `db:"properties"`
	Edges         json.RawMessage `db:"edges"`
	CanonicalHash string
}

// BuildSnapshot creates a snapshot of the database.
func BuildSnapshot(ctx context.Context, db *sqlx.DB, datasourceID uuid.UUID) ([]Node, error) {
	var nodes []Node
	query := `SELECT id, core_id, node_name, type_name, qualified_path, properties, edges FROM catalog_node WHERE datasource_id = $1`
	if err := db.SelectContext(ctx, &nodes, query, datasourceID); err != nil {
		return nil, fmt.Errorf("failed to select nodes for snapshot: %w", err)
	}

	// Calculate canonical hash for each node. This is crucial for the diffing logic.
	for i := range nodes {
		// The hashNode function is in the same package (in compare.go)
		// and can be used here.
		nodes[i].CanonicalHash = HashNode(nodes[i])
	}

	return nodes, nil
}
