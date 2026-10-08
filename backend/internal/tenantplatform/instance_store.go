package tenantplatform

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// storedEndpoints is the JSON shape kept under tenant_instance.config.endpoints.
type storedEndpoints struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Issuer   string `json:"issuer"`
}

func toStored(ep Endpoints) storedEndpoints {
	return storedEndpoints{Host: ep.PostgresHost, Port: ep.PostgresPort, Database: ep.DatabaseName, Issuer: ep.Issuer}
}

// PgInstanceStore implements InstanceStore against tenant_instance.
type PgInstanceStore struct {
	Pool *pgxpool.Pool
}

// MarkRunning moves an instance from provisioning to running and records its
// endpoints in one statement.
//
// The statement is guarded by status = 'provisioning'. So a stale or replayed
// run cannot reactivate an instance that was paused or retired.
//
// A retry after success finds zero rows updated. It is treated as success only
// when the instance is already running with the same endpoints. Any other status,
// or running with different endpoints, is an error.
func (s *PgInstanceStore) MarkRunning(ctx context.Context, instanceID string, ep Endpoints) error {
	if s.Pool == nil {
		return errors.New("instance store has no database pool")
	}
	endpointsJSON, err := json.Marshal(toStored(ep))
	if err != nil {
		return errors.New("encode endpoints")
	}
	tag, err := s.Pool.Exec(ctx, `
		UPDATE tenant_instance
		SET status = 'running',
		    config = jsonb_set(config, '{endpoints}', $2::jsonb, true),
		    updated_at = now()
		WHERE id = $1::uuid AND status = 'provisioning'`, instanceID, endpointsJSON)
	if err != nil {
		return errors.New("mark instance running: the database rejected the write")
	}
	if tag.RowsAffected() == 1 {
		return nil
	}

	// Zero rows: decide whether this is a successful retry or a refused transition.
	var status, stored string
	err = s.Pool.QueryRow(ctx, `
		SELECT status, COALESCE(config->'endpoints', 'null'::jsonb)::text
		FROM tenant_instance WHERE id = $1::uuid`, instanceID).Scan(&status, &stored)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("instance not found")
	}
	if err != nil {
		return errors.New("read instance state")
	}
	if status != "running" {
		return errors.New("instance is not provisioning; the transition was refused")
	}
	var got storedEndpoints
	if err := json.Unmarshal([]byte(stored), &got); err != nil || got != toStored(ep) {
		return errors.New("instance is already running with different endpoints")
	}
	return nil
}
