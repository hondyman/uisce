package semantic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"
)

// SemanticVersionStore handles storage and versioning of semantic objects
type SemanticVersionStore struct {
	db *sqlx.DB
}

// NewSemanticVersionStore creates a new version store
func NewSemanticVersionStore(db *sqlx.DB) *SemanticVersionStore {
	return &SemanticVersionStore{db: db}
}

// SaveObject saves a new version of a semantic object
func (s *SemanticVersionStore) SaveObject(ctx context.Context, obj SemanticObject, actor string) error {
	var current sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `
		SELECT current_version FROM semantic.heads WHERE id=$1
	`, obj.ID).Scan(&current)

	next := 1
	if current.Valid {
		next = int(current.Int64) + 1
	}

	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO semantic.objects (id, version, env, tenant_id, type, payload, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, obj.ID, next, obj.Env, obj.TenantID, obj.Type, obj.Payload, actor)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `
		INSERT INTO semantic.heads (id, env, tenant_id, type, current_version)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (id) DO UPDATE SET current_version=$5
	`, obj.ID, obj.Env, obj.TenantID, obj.Type, next)
	if err != nil {
		return err
	}

	return tx.Commit()
}

// GetVersion retrieves a specific version of an object
func (s *SemanticVersionStore) GetVersion(ctx context.Context, id string, version int) (*SemanticObject, error) {
	var obj SemanticObject
	var query string
	var args []interface{}

	if version == -1 {
		// Get Head
		query = `
			SELECT o.* 
			FROM semantic.objects o
			JOIN semantic.heads h ON o.id = h.id AND o.version = h.current_version
			WHERE o.id = $1
		`
		args = []interface{}{id}
	} else {
		query = `SELECT * FROM semantic.objects WHERE id=$1 AND version=$2`
		args = []interface{}{id, version}
	}

	err := s.db.GetContext(ctx, &obj, query, args...)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

// GetHistory retrieves the version history for an object
func (s *SemanticVersionStore) GetHistory(ctx context.Context, id string) ([]SemanticObject, error) {
	var history []SemanticObject
	query := `
		SELECT * FROM semantic.objects 
		WHERE id=$1 
		ORDER BY version DESC
	`
	err := s.db.SelectContext(ctx, &history, query, id)
	if err != nil {
		return nil, err
	}
	return history, nil
}

// Diff compares two versions of an object
func (s *SemanticVersionStore) Diff(ctx context.Context, id string, from, to int) (*SemanticDiffDTO, error) {
	var p1, p2 []byte
	// Should optimize to fetch both in one query or parallel, but sequential is fine
	err := s.db.QueryRowContext(ctx, `SELECT payload FROM semantic.objects WHERE id=$1 AND version=$2`, id, from).Scan(&p1)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version %d: %w", from, err)
	}
	err = s.db.QueryRowContext(ctx, `SELECT payload FROM semantic.objects WHERE id=$1 AND version=$2`, id, to).Scan(&p2)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch version %d: %w", to, err)
	}

	changes, err := jsonDiff(p1, p2)
	if err != nil {
		return nil, err
	}

	diffDTO := make(SemanticDiffDTO)
	diffDTO[id] = struct {
		Changes []SemanticDiffChange `json:"changes"`
	}{
		Changes: changes,
	}

	return &diffDTO, nil
}

// Rollback creates a new version from an old version
func (s *SemanticVersionStore) Rollback(ctx context.Context, id string, targetVersion int, actor string) error {
	target, err := s.GetVersion(ctx, id, targetVersion)
	if err != nil {
		return err
	}

	// Save as new version
	return s.SaveObject(ctx, SemanticObject{
		ID:       id,
		Env:      target.Env,
		TenantID: target.TenantID,
		Type:     target.Type,
		Payload:  target.Payload,
	}, actor)
}

func jsonDiff(b1, b2 []byte) ([]SemanticDiffChange, error) {
	var m1, m2 map[string]interface{}
	if err := json.Unmarshal(b1, &m1); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b2, &m2); err != nil {
		return nil, err
	}

	var changes []SemanticDiffChange
	compareMap("", m1, m2, &changes)
	return changes, nil
}

func compareMap(path string, m1, m2 map[string]interface{}, changes *[]SemanticDiffChange) {
	// Check for removals and modifications
	for k, v1 := range m1 {
		currPath := joinPath(path, k)
		v2, ok := m2[k]
		if !ok {
			*changes = append(*changes, SemanticDiffChange{
				Path: currPath,
				Old:  v1,
				Type: "removed",
			})
			continue
		}

		if !reflect.DeepEqual(v1, v2) {
			// If both are maps, recurse
			if isMap(v1) && isMap(v2) {
				compareMap(currPath, v1.(map[string]interface{}), v2.(map[string]interface{}), changes)
			} else {
				*changes = append(*changes, SemanticDiffChange{
					Path: currPath,
					Old:  v1,
					New:  v2,
					Type: "modified",
				})
			}
		}
	}

	// Check for additions
	for k, v2 := range m2 {
		if _, ok := m1[k]; !ok {
			*changes = append(*changes, SemanticDiffChange{
				Path: joinPath(path, k),
				New:  v2,
				Type: "added",
			})
		}
	}
}

func joinPath(base, key string) string {
	if base == "" {
		return key
	}
	return base + "." + key
}

func isMap(v interface{}) bool {
	_, ok := v.(map[string]interface{})
	return ok
}
