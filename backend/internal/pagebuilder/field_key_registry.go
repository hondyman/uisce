// Package pagebuilder holds the CRUD-capability and widget-policy facets that
// back the page builder, plus the field-key registry that lets those facets
// key on a stable surrogate ID rather than the field-name strings the older
// entitlement and validation tables still use.
package pagebuilder

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
)

type FieldKeyRegistry struct {
	db *sqlx.DB
}

func NewFieldKeyRegistry(db *sqlx.DB) *FieldKeyRegistry {
	return &FieldKeyRegistry{db: db}
}

// GetOrCreate returns the stable field_key_id for (boName, fieldName),
// creating the registry row on first reference.
func (r *FieldKeyRegistry) GetOrCreate(ctx context.Context, boName, fieldName string) (string, error) {
	var id string
	err := r.db.GetContext(ctx, &id, `
		INSERT INTO bo_field_key_registry (bo_name, field_name)
		VALUES ($1, $2)
		ON CONFLICT (bo_name, field_name) DO UPDATE SET bo_name = EXCLUDED.bo_name
		RETURNING id
	`, boName, fieldName)
	if err != nil {
		return "", fmt.Errorf("failed to get or create field key for %s.%s: %w", boName, fieldName, err)
	}
	return id, nil
}

// Resolve looks up the field_key_id for (boName, fieldName) without creating
// it. Returns "" if no registry entry exists yet.
func (r *FieldKeyRegistry) Resolve(ctx context.Context, boName, fieldName string) (string, error) {
	var id string
	err := r.db.GetContext(ctx, &id, `
		SELECT id FROM bo_field_key_registry WHERE bo_name = $1 AND field_name = $2
	`, boName, fieldName)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to resolve field key for %s.%s: %w", boName, fieldName, err)
	}
	return id, nil
}
