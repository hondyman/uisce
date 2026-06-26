package api

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Store manages database interactions for templates.
type Store struct {
	DB *sqlx.DB
}

// NewStore creates a new Store.
func NewStore(db *sqlx.DB) *Store {
	return &Store{DB: db}
}

// SaveTemplate inserts or updates a template in the registry.
func (s *Store) SaveTemplate(tmpl *Template) error {
	templateJSON, err := json.Marshal(tmpl)
	if err != nil {
		return fmt.Errorf("failed to marshal template: %w", err)
	}

	// Calculate a hash of the template content for versioning/integrity.
	hash := sha256.Sum256(templateJSON)
	schemaHash := fmt.Sprintf("%x", hash)

	query := `
        INSERT INTO public.template_registry (
            node_id, version, calc_type, owner, tags, lineage, schema_hash, template, created_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
        ON CONFLICT (node_id) DO UPDATE SET
            version = EXCLUDED.version,
            calc_type = EXCLUDED.calc_type,
            owner = EXCLUDED.owner,
            tags = EXCLUDED.tags,
            lineage = EXCLUDED.lineage,
            schema_hash = EXCLUDED.schema_hash,
            template = EXCLUDED.template,
            updated_at = NOW()
    `
	_, err = s.DB.Exec(query, tmpl.NodeID, tmpl.Version, tmpl.Financial.Type, tmpl.Owner, pq.Array(tmpl.Tags), pq.Array([]string{}), schemaHash, templateJSON)
	return err
}

// GetTemplate retrieves a single template by its node_id.
func (s *Store) GetTemplate(nodeID string) (*Template, error) {
	var templateJSON []byte
	err := s.DB.Get(&templateJSON, "SELECT template FROM public.template_registry WHERE node_id = $1", nodeID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("template with node_id '%s' not found", nodeID)
		}
		return nil, err
	}

	var tmpl Template
	if err := json.Unmarshal(templateJSON, &tmpl); err != nil {
		return nil, fmt.Errorf("failed to unmarshal template: %w", err)
	}
	return &tmpl, nil
}

// ListTemplates retrieves all templates from the registry.
func (s *Store) ListTemplates() ([]Template, error) {
	var templates []Template
	var rows *sql.Rows
	rows, err := s.DB.Query("SELECT template FROM public.template_registry ORDER BY node_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var templateJSON []byte
		if err := rows.Scan(&templateJSON); err != nil {
			return nil, err
		}
		var tmpl Template
		if err := json.Unmarshal(templateJSON, &tmpl); err == nil {
			templates = append(templates, tmpl)
		}
	}
	return templates, nil
}
