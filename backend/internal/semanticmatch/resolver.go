package semanticmatch

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// NodeResolver finds or creates catalog_node rows for physical columns
// (ATTRIBUTE) and semantic terms (SEMANTIC_TERM), returning node IDs for
// WriteEdges.
//
// Schema assumptions — adjust the SQL to your DDL:
//   - node_id TEXT PK (if yours is bigserial, drop node_id from the INSERT
//     and keep RETURNING node_id)
//   - UNIQUE (tenant_id, node_key)  (if absent, swap to SELECT-then-INSERT)
//   - properties JSONB
type NodeResolver struct {
	DB     *sql.DB
	Tenant string
}

func NewNodeResolver(db *sql.DB, tenantID string) *NodeResolver {
	return &NodeResolver{DB: db, Tenant: tenantID}
}

// ColumnNodeKey: "prod.public.ts_order_broker.amount"
func ColumnNodeKey(c ColumnMeta) string {
	var parts []string
	for _, p := range []string{c.Database, c.Schema, c.Table, c.Column} {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, strings.ToLower(p))
		}
	}
	return strings.Join(parts, ".")
}

// TermNodeKey: "bb:px_last" / "glossary:57"
func TermNodeKey(t *SemanticTerm) string { return strings.ToLower(t.ID) }

// Resolve satisfies the signature WriteEdges expects.
func (r *NodeResolver) Resolve(ctx context.Context, c ColumnMeta, t *SemanticTerm) (string, string, bool) {
	src, err := r.ensureColumnNode(ctx, c)
	if err != nil {
		return "", "", false
	}
	dst, err := r.ensureTermNode(ctx, t)
	if err != nil {
		return "", "", false
	}
	return src, dst, true
}

func (r *NodeResolver) ensureColumnNode(ctx context.Context, c ColumnMeta) (string, error) {
	props, _ := json.Marshal(map[string]any{
		"database": c.Database, "schema": c.Schema, "table": c.Table,
		"column": c.Column, "data_type": c.DataType, "nullable": c.Nullable,
	})
	return r.upsertNode(ctx, "ATTRIBUTE", ColumnNodeKey(c), c.Column, string(props))
}

func (r *NodeResolver) ensureTermNode(ctx context.Context, t *SemanticTerm) (string, error) {
	props, _ := json.Marshal(map[string]any{
		"source": t.Source, "mnemonic": t.Mnemonic, "domain": t.Domain,
		"description": t.Description, "raw_type": t.RawType,
	})
	return r.upsertNode(ctx, "SEMANTIC_TERM", TermNodeKey(t), t.Name, string(props))
}

const upsertNodeSQL = `
INSERT INTO catalog_node (tenant_id, node_id, node_type, node_key, node_name, properties)
VALUES ($1, $2, $3, $4, $5, $6::jsonb)
ON CONFLICT (tenant_id, node_key)
DO UPDATE SET node_name = EXCLUDED.node_name,
              properties = catalog_node.properties || EXCLUDED.properties
RETURNING node_id`

func (r *NodeResolver) upsertNode(ctx context.Context, nodeType, key, name, propsJSON string) (string, error) {
	var nodeID string
	err := r.DB.QueryRowContext(ctx, upsertNodeSQL,
		r.Tenant, newID(), nodeType, key, name, propsJSON).Scan(&nodeID)
	if err != nil {
		return "", fmt.Errorf("upsert %s node %q: %w", nodeType, key, err)
	}
	return nodeID, nil
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
