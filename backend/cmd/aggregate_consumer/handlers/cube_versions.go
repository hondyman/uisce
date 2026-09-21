package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hondyman/uisce/backend/cmd/aggregate_consumer/dedupe"
)

// CubeCustomModels transposes the trigger `cube_custom_model_version`.
// The original trigger inserted a new row into cube_custom_model_versions
// when OLD.custom_config IS DISTINCT FROM NEW.custom_config. The consumer
// preserves the IS DISTINCT FROM gate — that's the trigger's bug-fix that
// also fixes the semantic_query_templates version bug.
type CubeCustomModels struct {
	pool   *pgxpool.Pool
	dedupe *dedupe.Store
}

const cubeCustomHandler = "cube_custom_models"

func NewCubeCustomModels(pool *pgxpool.Pool, d *dedupe.Store) *CubeCustomModels {
	return &CubeCustomModels{pool: pool, dedupe: d}
}

func (h *CubeCustomModels) Topic() string { return "alpha_trg.public.cube_custom_models" }

func (h *CubeCustomModels) Handle(ctx context.Context, evt Event) error {
	// Only updates; inserts don't trigger versioning in the original.
	if evt.Op != "u" {
		return nil
	}
	if err := h.dedupe.MarkIfNew(ctx, cubeCustomHandler, evt.LSN, evt.Table, evt.Op); err != nil {
		if errors.Is(err, dedupe.ErrAlreadyProcessed) {
			return nil
		}
		return fmt.Errorf("dedupe: %w", err)
	}

	beforeJSON, _ := json.Marshal(evt.Before)
	afterJSON, _ := json.Marshal(evt.After)
	var beforeCfg, afterCfg any
	_ = json.Unmarshal(beforeJSON, &beforeCfg)
	_ = json.Unmarshal(afterJSON, &afterCfg)

	// Use Postgres IS DISTINCT FROM via the comparison on the raw JSONB.
	// Trigger logic was: IF OLD.custom_config IS DISTINCT FROM NEW.custom_config
	// We pull the values from evt.After/Before directly to avoid a DB round-trip.
	afterCustomCfg := evt.After["custom_config"]
	beforeCustomCfg := evt.Before["custom_config"]

	// json.Equal handles nil/null cases correctly: nil IS DISTINCT FROM nil is false.
	if jsonEqual(beforeCustomCfg, afterCustomCfg) {
		return nil
	}

	const insertVersion = `
INSERT INTO cube_custom_model_versions (custom_model_id, version, custom_config, changed_by)
VALUES ($1, $2, $3::jsonb, $4::uuid)
`
	customModelID := stringOf(evt.After["id"])
	version := intOf(evt.After["version"])
	customConfig := mustJSON(afterCustomCfg)
	createdBy := stringOf(evt.After["created_by"])

	if _, err := h.pool.Exec(ctx, insertVersion, customModelID, version, customConfig, createdBy); err != nil {
		return fmt.Errorf("insert cube_custom_model_versions: %w", err)
	}
	return nil
}

// CubeSecurityPolicies transposes the trigger `cube_security_policy_version`.
// Same pattern as CubeCustomModels — IS DISTINCT FROM gate preserved.
type CubeSecurityPolicies struct {
	pool   *pgxpool.Pool
	dedupe *dedupe.Store
}

const cubeSecurityHandler = "cube_security_policies"

func NewCubeSecurityPolicies(pool *pgxpool.Pool, d *dedupe.Store) *CubeSecurityPolicies {
	return &CubeSecurityPolicies{pool: pool, dedupe: d}
}

func (h *CubeSecurityPolicies) Topic() string { return "alpha_trg.public.cube_security_policies" }

func (h *CubeSecurityPolicies) Handle(ctx context.Context, evt Event) error {
	if evt.Op != "u" {
		return nil
	}
	if err := h.dedupe.MarkIfNew(ctx, cubeSecurityHandler, evt.LSN, evt.Table, evt.Op); err != nil {
		if errors.Is(err, dedupe.ErrAlreadyProcessed) {
			return nil
		}
		return fmt.Errorf("dedupe: %w", err)
	}

	beforeConditions := evt.Before["conditions"]
	afterConditions := evt.After["conditions"]
	beforeEffects := evt.Before["effects"]
	afterEffects := evt.After["effects"]

	if jsonEqual(beforeConditions, afterConditions) && jsonEqual(beforeEffects, afterEffects) {
		return nil
	}

	const insertVersion = `
INSERT INTO cube_security_policy_versions (policy_id, version, conditions, effects, changed_by)
VALUES ($1, $2, $3::jsonb, $4::jsonb, $5::uuid)
`
	policyID := stringOf(evt.After["id"])
	version := intOf(evt.After["version"])
	conditions := mustJSON(afterConditions)
	effects := mustJSON(afterEffects)
	createdBy := stringOf(evt.After["created_by"])

	if _, err := h.pool.Exec(ctx, insertVersion, policyID, version, conditions, effects, createdBy); err != nil {
		return fmt.Errorf("insert cube_security_policy_versions: %w", err)
	}
	return nil
}
