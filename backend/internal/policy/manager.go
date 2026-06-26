package policy

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// UiscePolicy represents a temporal policy definition.
// Named UiscePolicy to avoid conflict with existing Policy struct.
type UiscePolicy struct {
	ID        int
	Key       string
	CueSource string
	ValidFrom time.Time
}

type PolicyManager struct {
	db *sql.DB
}

func NewPolicyManager(db *sql.DB) *PolicyManager {
	return &PolicyManager{db: db}
}

// GetEffectivePolicy finds the single rule active for a specific moment in time
func (pm *PolicyManager) GetEffectivePolicy(ctx context.Context, key string, targetDate time.Time) (*UiscePolicy, error) {
	query := `
		SELECT id, policy_key, cue_definition, valid_from
		FROM uisce_policies
		WHERE policy_key = $1
		  AND status = 'ACTIVE'
		  AND valid_from <= $2
		  AND (valid_to IS NULL OR valid_to > $2)
		ORDER BY valid_from DESC
		LIMIT 1;
	`

	row := pm.db.QueryRowContext(ctx, query, key, targetDate)

	var p UiscePolicy
	if err := row.Scan(&p.ID, &p.Key, &p.CueSource, &p.ValidFrom); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no active policy found for '%s' on date %s", key, targetDate)
		}
		return nil, err
	}
	return &p, nil
}
