package api

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/lib/pq"
)

// Registry manages database interactions for templates.
type Registry struct{ DB *sql.DB }

func (r *Registry) hashTemplate(t *Template) (string, []byte, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", nil, err
	}
	h := sha256.Sum256(b)
	return fmt.Sprintf("%x", h[:]), b, nil
}

// UpsertTemplate transactionally saves a template to the registry and version history.
func (r *Registry) UpsertTemplate(ctx context.Context, t *Template) error {
	hash, payload, err := r.hashTemplate(t)
	if err != nil {
		return err
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	status := "draft"
	if t.Governance != nil && t.Governance.Status != "" {
		status = t.Governance.Status
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.template_registry
		  (node_id, version, node_type, domain, category, subcategory, owner, tags, lineage, status, schema_hash, template, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,NOW(),NOW())
		ON CONFLICT (node_id) DO UPDATE SET
		  version=$2, node_type=$3, domain=$4, category=$5, subcategory=$6,
		  owner=$7, tags=$8, lineage=$9, status=$10,
		  schema_hash=$11, template=$12, updated_at=NOW()
	`, t.NodeID, t.Version, t.NodeType, t.Domain, t.Category, t.Subcategory, t.Owner, pq.Array(t.Tags), pq.Array(t.Lineage), status, hash, payload)
	if err != nil {
		return fmt.Errorf("failed to upsert into template_registry: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.template_versions (node_id, version, schema_hash, template, created_at)
		VALUES ($1,$2,$3,$4,NOW())
		ON CONFLICT (node_id, version) DO NOTHING
	`, t.NodeID, t.Version, hash, payload)
	if err != nil {
		return fmt.Errorf("failed to insert into template_versions: %w", err)
	}

	return tx.Commit()
}

// GetTemplate retrieves the current version of a template by its node_id.
func (r *Registry) GetTemplate(ctx context.Context, nodeID string) (*Template, error) {
	var payload []byte
	err := r.DB.QueryRowContext(ctx, `SELECT template FROM public.template_registry WHERE node_id=$1`, nodeID).Scan(&payload)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template '%s' not found", nodeID)
		}
		return nil, err
	}
	var t Template
	if err := json.Unmarshal(payload, &t); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}
	return &t, nil
}

// ListTemplates retrieves a filtered list of template metadata.
func (r *Registry) ListTemplates(ctx context.Context, filter map[string]string, tag string) ([]map[string]any, error) {
	query := `SELECT node_id, version, domain, category, subcategory, owner, tags, status, updated_at FROM public.template_registry WHERE 1=1`
	args := []any{}
	argIdx := 1

	for k, v := range filter {
		if v != "" {
			query += fmt.Sprintf(" AND %s = $%d", pq.QuoteIdentifier(k), argIdx)
			args = append(args, v)
			argIdx++
		}
	}
	if tag != "" {
		query += fmt.Sprintf(" AND $%d = ANY(tags)", argIdx)
		args = append(args, tag)
	}

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []map[string]any
	for rows.Next() {
		var nodeID, version, domain, category, subcategory, owner, status string
		var tags []string
		var updatedAt sql.NullTime
		if err := rows.Scan(&nodeID, &version, &domain, &category, &subcategory, &owner, pq.Array(&tags), &status, &updatedAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{
			"node_id":     nodeID,
			"version":     version,
			"domain":      domain,
			"category":    category,
			"subcategory": subcategory,
			"owner":       owner,
			"tags":        tags,
			"status":      status,
			"updated_at":  updatedAt.Time,
		})
	}
	return out, nil
}

// PromoteTemplate updates the status of a template.
func (r *Registry) PromoteTemplate(ctx context.Context, nodeID, status string) (sql.Result, error) {
	return r.DB.ExecContext(ctx, `UPDATE public.template_registry SET status=$1, updated_at=NOW() WHERE node_id=$2`, status, nodeID)
}
