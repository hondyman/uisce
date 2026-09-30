package mdm

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// BatchSurvivorshipRequest contains parameters to execute a set-based MDM survivorship run
type BatchSurvivorshipRequest struct {
	TenantID        uuid.UUID `json:"tenant_id"`
	BatchID         string    `json:"batch_id"`
	EntityType      string    `json:"entity_type"`
	StagingTable    string    `json:"staging_table"`
	TargetTable     string    `json:"target_table"`
	EntityKeyField  string    `json:"entity_key_field"`
	SourceIDField   string    `json:"source_id_field"`
	AsOfField       string    `json:"as_of_field"`
	PKField         string    `json:"pk_field"`
	BatchCutoff     time.Time `json:"batch_cutoff"`
	TargetKeyField  string    `json:"target_key_field"` // Defaults to EntityKeyField if empty
	RulesSnapshotID *uuid.UUID `json:"rules_snapshot_id,omitempty"`
}

// BatchSurvivorshipResult reports execution summary back to Temporal workflow
type BatchSurvivorshipResult struct {
	BatchID                  string   `json:"batch_id"`
	TenantID                 string   `json:"tenant_id"`
	EntityType               string   `json:"entity_type"`
	GoldenRecordsMaterialized int      `json:"golden_records_materialized"`
	ExceptionsCount          int      `json:"exceptions_count"`
	ExceptionEntityKeys      []string `json:"exception_entity_keys"`
	ExecutionDurationMs      int64    `json:"execution_duration_ms"`
}

// BatchRuleSnapshotResult holds the metadata of pinned batch rules
type BatchRuleSnapshotResult struct {
	SnapshotID uuid.UUID                    `json:"snapshot_id"`
	BatchID    string                       `json:"batch_id"`
	TenantID   uuid.UUID                    `json:"tenant_id"`
	EntityType string                       `json:"entity_type"`
	AsOf       time.Time                    `json:"as_of"`
	Rules      map[string]ResolvedFieldRule `json:"rules"`
}

// BatchSurvivorshipActivity orchestrates set-based SQL survivorship runs within Temporal workflows
type BatchSurvivorshipActivity struct {
	db           *sql.DB
	ruleResolver *RuleResolver
	sqlGen       *SQLGenerator
}

// NewBatchSurvivorshipActivity creates a new batch survivorship activity instance
func NewBatchSurvivorshipActivity(db *sql.DB, resolver *RuleResolver) *BatchSurvivorshipActivity {
	return &BatchSurvivorshipActivity{
		db:           db,
		ruleResolver: resolver,
		sqlGen:       NewSQLGenerator(),
	}
}

// ResolveAndPinBatchRulesActivity loads effective rules for a tenant and entity type,
// and saves an immutable snapshot in mdm_batch_rule_snapshot.
func (a *BatchSurvivorshipActivity) ResolveAndPinBatchRulesActivity(
	ctx context.Context,
	tenantID uuid.UUID,
	batchID string,
	entityType string,
	cutoff time.Time,
) (*BatchRuleSnapshotResult, error) {
	if a.db == nil {
		return nil, errors.New("database connection is required")
	}

	rules, err := a.ruleResolver.ResolveEntityRules(ctx, tenantID, entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve entity rules: %w", err)
	}

	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal rules json: %w", err)
	}

	var snapshotID uuid.UUID
	err = a.db.QueryRowContext(ctx, `
		INSERT INTO public.mdm_batch_rule_snapshot (
			batch_id, tenant_id, entity_type, as_of, pinned_rules_json
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (batch_id) DO UPDATE 
		SET pinned_rules_json = EXCLUDED.pinned_rules_json
		RETURNING id
	`, batchID, tenantID, entityType, cutoff.UTC(), rulesJSON).Scan(&snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to pin batch rules: %w", err)
	}

	return &BatchRuleSnapshotResult{
		SnapshotID: snapshotID,
		BatchID:    batchID,
		TenantID:   tenantID,
		EntityType: entityType,
		AsOf:       cutoff.UTC(),
		Rules:      rules,
	}, nil
}

// RunBatchSurvivorshipActivity executes the high-performance CTE query against PostgreSQL
// and upserts golden records into the target canonical table.
func (a *BatchSurvivorshipActivity) RunBatchSurvivorshipActivity(
	ctx context.Context,
	req BatchSurvivorshipRequest,
) (*BatchSurvivorshipResult, error) {
	start := time.Now()

	if a.db == nil {
		return nil, errors.New("database connection is required")
	}
	if req.TenantID == uuid.Nil {
		return nil, errors.New("tenant_id is required")
	}
	if req.EntityType == "" {
		return nil, errors.New("entity_type is required")
	}
	if req.StagingTable == "" {
		return nil, errors.New("staging_table is required")
	}
	if req.TargetTable == "" {
		return nil, errors.New("target_table is required")
	}

	targetKey := req.TargetKeyField
	if targetKey == "" {
		targetKey = req.EntityKeyField
	}

	// 1. Resolve or Load Rules
	var rules map[string]ResolvedFieldRule
	if req.RulesSnapshotID != nil {
		var rulesJSON []byte
		err := a.db.QueryRowContext(ctx, `
			SELECT pinned_rules_json FROM public.mdm_batch_rule_snapshot WHERE id = $1
		`, *req.RulesSnapshotID).Scan(&rulesJSON)
		if err != nil {
			return nil, fmt.Errorf("failed to load pinned rules snapshot: %w", err)
		}
		if err := json.Unmarshal(rulesJSON, &rules); err != nil {
			return nil, fmt.Errorf("failed to unmarshal pinned rules: %w", err)
		}
	} else {
		resolved, err := a.ruleResolver.ResolveEntityRules(ctx, req.TenantID, req.EntityType)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve entity rules: %w", err)
		}
		rules = resolved
	}

	// Filter rules to attributes physically present in the staging table
	stagingCols, err := a.getTableColumns(ctx, req.StagingTable)
	if err == nil && len(stagingCols) > 0 {
		filtered := make(map[string]ResolvedFieldRule)
		for k, r := range rules {
			attr := strings.ToLower(r.AttributeName)
			if stagingCols[attr] {
				filtered[k] = r
			}
		}
		if len(filtered) > 0 {
			rules = filtered
		}
	}

	// 2. Build IR Plan
	planCfg := PlanConfig{
		TenantID:       req.TenantID,
		EntityType:     req.EntityType,
		StagingTable:   req.StagingTable,
		TargetTable:    req.TargetTable,
		EntityKeyField: req.EntityKeyField,
		SourceIDField:  req.SourceIDField,
		AsOfField:      req.AsOfField,
		PKField:        req.PKField,
		BatchCutoff:    req.BatchCutoff,
	}

	plan, err := BuildBatchPlan(planCfg, rules)
	if err != nil {
		return nil, fmt.Errorf("failed to build batch plan: %w", err)
	}

	// 3. Render CTE Upsert Query
	renderedUpsert, err := a.sqlGen.RenderUpsertSQL(plan, targetKey)
	if err != nil {
		return nil, fmt.Errorf("failed to render upsert SQL: %w", err)
	}

	// 4. Execute within transaction
	tx, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	rows, err := tx.QueryContext(ctx, renderedUpsert.SQL, renderedUpsert.Args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute batch survivorship upsert: %w", err)
	}
	defer rows.Close()

	var (
		materializedCount int
		affectedKeys      []string
	)

	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, fmt.Errorf("failed to scan returned entity key: %w", err)
		}
		materializedCount++
		affectedKeys = append(affectedKeys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit survivorship transaction: %w", err)
	}

	duration := time.Since(start).Milliseconds()

	return &BatchSurvivorshipResult{
		BatchID:                  req.BatchID,
		TenantID:                 req.TenantID.String(),
		EntityType:               req.EntityType,
		GoldenRecordsMaterialized: materializedCount,
		ExceptionsCount:          0, // Standard batch has 0 exceptions
		ExceptionEntityKeys:      []string{},
		ExecutionDurationMs:      duration,
	}, nil
}

// getTableColumns inspects a table to retrieve its physical column names
func (a *BatchSurvivorshipActivity) getTableColumns(ctx context.Context, tableName string) (map[string]bool, error) {
	query := `
		SELECT attname 
		FROM pg_attribute 
		WHERE attrelid = $1::regclass 
		  AND attnum > 0 
		  AND NOT attisdropped;
	`
	rows, err := a.db.QueryContext(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return nil, err
		}
		cols[strings.ToLower(col)] = true
	}
	return cols, nil
}

