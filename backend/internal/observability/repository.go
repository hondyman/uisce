// observability/repository.go
package observability

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Repository defines the contract for reading/writing Observability telemetry
type Repository interface {
	ListETLRuns(ctx context.Context, tenantID, status, from, to string, limit int32) ([]ETLRunRecord, error)
	GetETLRun(ctx context.Context, runID string) (*ETLRunRecord, error)

	ListWASMVersions(ctx context.Context, moduleName string) ([]WASMVersionRecord, error)
	ActivateWASMVersion(ctx context.Context, versionID string) error

	GetRuleLineage(ctx context.Context, ruleID, from, to, portfolioID string) ([]RuleLineageRecord, error)
	GetScenarioLineage(ctx context.Context, scenarioID, from, to, portfolioID string) ([]ScenarioLineageRecord, error)
}

// SQLRepository implements Repository
type SQLRepository struct {
	db *sqlx.DB
}

// NewSQLRepository creates a query-enabled repository
func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

// Models match exactly what the REST handlers expect

type ETLRunRecord struct {
	ETLRunID            uuid.UUID  `db:"run_id" json:"etl_run_id"`
	TenantID            uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	ValuationDate       time.Time  `db:"valuation_date" json:"valuation_date"`
	EngineName          string     `db:"engine_name" json:"engine_name"`
	Status              string     `db:"status" json:"status"`
	StartTime           time.Time  `db:"start_time" json:"started_at"`
	EndTime             *time.Time `db:"end_time" json:"completed_at"`
	PortfoliosProcessed int        `db:"portfolios_processed" json:"portfolios_processed"`
	EvaluationsCount    int        `db:"evaluations_count" json:"evaluations_count"`
	BreachesCount       int        `db:"breaches_count" json:"breaches_count"`
}

type WASMVersionRecord struct {
	WASMVersionID  uuid.UUID `db:"wasm_version_id" json:"wasm_version_id"`
	ModuleName     string    `db:"module_name" json:"module_name"`
	Version        string    `db:"version" json:"version"`
	BuildHash      string    `db:"build_hash" json:"build_hash"`
	BuildTime      time.Time `db:"build_time" json:"build_time"`
	ArtifactURI    string    `db:"artifact_uri" json:"artifact_uri"`
	ChecksumSHA256 string    `db:"checksum_sha256" json:"checksum_sha256"`
	IsActive       bool      `db:"is_active" json:"is_active"`
}

type RuleLineageRecord struct {
	ValuationDate  time.Time `db:"valuation_date" json:"valuation_date"`
	PortfolioID    uuid.UUID `db:"portfolio_id" json:"portfolio_id"`
	Status         string    `db:"status" json:"status"`
	MetricValue    float64   `db:"metric_value" json:"metric_value"`
	ThresholdValue float64   `db:"threshold_value" json:"threshold_value"`
	ETLRunID       uuid.UUID `db:"etl_run_id" json:"etl_run_id"`
}

type ScenarioLineageRecord struct {
	ValuationDate time.Time `db:"valuation_date" json:"valuation_date"`
	PortfolioID   uuid.UUID `db:"portfolio_id" json:"portfolio_id"`
	PnL           float64   `db:"pnl" json:"pnl"`
	ETLRunID      uuid.UUID `db:"etl_run_id" json:"etl_run_id"`
}

func (r *SQLRepository) ListETLRuns(ctx context.Context, tenantID, status, from, to string, limit int32) ([]ETLRunRecord, error) {
	// Simple select with limit binding. (Production needs proper dynamic AND clauses)
	query := `
		SELECT run_id, tenant_id, valuation_date, engine_name, status,
		       start_time, end_time, portfolios_processed, evaluations_count, breaches_count
		FROM edm.etl_run 
		ORDER BY start_time DESC LIMIT $1
	`
	var runs []ETLRunRecord
	err := r.db.SelectContext(ctx, &runs, query, limit)
	return runs, err
}

func (r *SQLRepository) GetETLRun(ctx context.Context, runID string) (*ETLRunRecord, error) {
	query := `
		SELECT run_id, tenant_id, valuation_date, engine_name, status,
		       start_time, end_time, portfolios_processed, evaluations_count, breaches_count
		FROM edm.etl_run WHERE run_id = $1
	`
	var run ETLRunRecord
	err := r.db.GetContext(ctx, &run, query, runID)
	return &run, err
}

func (r *SQLRepository) ListWASMVersions(ctx context.Context, moduleName string) ([]WASMVersionRecord, error) {
	query := `
		SELECT wasm_version_id, module_name, version, build_hash, build_time, artifact_uri, checksum_sha256, is_active
		FROM edm.wasm_module_version
		WHERE module_name = $1 ORDER BY build_time DESC
	`
	var versions []WASMVersionRecord
	err := r.db.SelectContext(ctx, &versions, query, moduleName)
	return versions, err
}

func (r *SQLRepository) ActivateWASMVersion(ctx context.Context, versionID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Deactivate all
	_, err = tx.ExecContext(ctx, "UPDATE edm.wasm_module_version SET is_active = false")
	if err != nil {
		return err
	}

	// 2. Activate target
	_, err = tx.ExecContext(ctx, "UPDATE edm.wasm_module_version SET is_active = true WHERE wasm_version_id = $1", versionID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SQLRepository) GetRuleLineage(ctx context.Context, ruleID, from, to, portfolioID string) ([]RuleLineageRecord, error) {
	query := `
		SELECT valuation_date, portfolio_id, status, metric_value, threshold_value, etl_run_id
		FROM edm.rule_lineage
		WHERE rule_id = $1 ORDER BY valuation_date DESC LIMIT 100
	`
	var records []RuleLineageRecord
	err := r.db.SelectContext(ctx, &records, query, ruleID)
	return records, err
}

func (r *SQLRepository) GetScenarioLineage(ctx context.Context, scenarioID, from, to, portfolioID string) ([]ScenarioLineageRecord, error) {
	query := `
		SELECT valuation_date, portfolio_id, pnl, etl_run_id
		FROM edm.scenario_lineage
		WHERE scenario_id = $1 ORDER BY valuation_date DESC LIMIT 100
	`
	var records []ScenarioLineageRecord
	err := r.db.SelectContext(ctx, &records, query, scenarioID)
	return records, err
}
