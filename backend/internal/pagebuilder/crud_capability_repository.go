package pagebuilder

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Capability is what a business object or field supports structurally
// (create/read/update/delete + stage conditions like immutable_after_create).
// It is independent of who may exercise that capability — see the
// bp_field_permissions entitlement facet in internal/security for that.
// Effective writability at render/save time is capability AND entitlement.
type Capability struct {
	BOName     string          `db:"bo_name"`
	FieldKeyID *string         `db:"field_key_id"` // NULL = object-level default
	CanCreate  bool            `db:"can_create"`
	CanRead    bool            `db:"can_read"`
	CanUpdate  bool            `db:"can_update"`
	CanDelete  bool            `db:"can_delete"`
	Conditions json.RawMessage `db:"conditions"`
}

type CRUDCapabilityRepository struct {
	db *sqlx.DB
}

func NewCRUDCapabilityRepository(db *sqlx.DB) *CRUDCapabilityRepository {
	return &CRUDCapabilityRepository{db: db}
}

// GetForBO returns the object-level default row plus every field-level
// override for boName. Callers resolve effective capability per field by
// falling back to the default row (field_key_id IS NULL) when no override
// exists for that field's key.
func (r *CRUDCapabilityRepository) GetForBO(ctx context.Context, boName string) ([]Capability, error) {
	var rows []Capability
	err := r.db.SelectContext(ctx, &rows, `
		SELECT bo_name, field_key_id, can_create, can_read, can_update, can_delete, conditions
		FROM bo_crud_capabilities
		WHERE bo_name = $1
	`, boName)
	if err != nil {
		return nil, fmt.Errorf("failed to get CRUD capabilities for %s: %w", boName, err)
	}
	return rows, nil
}

// Upsert writes the object-level default (fieldKeyID == nil) or a field-level
// override (fieldKeyID set, from FieldKeyRegistry.GetOrCreate).
func (r *CRUDCapabilityRepository) Upsert(ctx context.Context, cap Capability) error {
	conditions := cap.Conditions
	if conditions == nil {
		conditions = json.RawMessage(`{}`)
	}

	// The plain (bo_name, field_key_id) unique constraint only enforces
	// uniqueness among non-NULL field_key_id rows; object-level defaults
	// (field_key_id IS NULL) are covered by a separate partial unique index
	// and need their own ON CONFLICT target.
	conflictTarget := "(bo_name, field_key_id)"
	if cap.FieldKeyID == nil {
		conflictTarget = "(bo_name) WHERE field_key_id IS NULL"
	}

	query := fmt.Sprintf(`
		INSERT INTO bo_crud_capabilities
			(bo_name, field_key_id, can_create, can_read, can_update, can_delete, conditions, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, now())
		ON CONFLICT %s DO UPDATE SET
			can_create = EXCLUDED.can_create,
			can_read   = EXCLUDED.can_read,
			can_update = EXCLUDED.can_update,
			can_delete = EXCLUDED.can_delete,
			conditions = EXCLUDED.conditions,
			updated_at = now()
	`, conflictTarget)

	_, err := r.db.ExecContext(ctx, query,
		cap.BOName, cap.FieldKeyID, cap.CanCreate, cap.CanRead, cap.CanUpdate, cap.CanDelete, conditions)
	if err != nil {
		return fmt.Errorf("failed to upsert CRUD capability for %s: %w", cap.BOName, err)
	}
	return nil
}
