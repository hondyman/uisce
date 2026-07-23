package aso

import (
	"context"
	"fmt"
	"hash/crc32"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Experiment Types
// ============================================================================

type ExperimentStatus string

const (
	ExpStatusCreated   ExperimentStatus = "created"
	ExpStatusRunning   ExperimentStatus = "running"
	ExpStatusStopped   ExperimentStatus = "stopped"
	ExpStatusCompleted ExperimentStatus = "completed"
)

// Experiment defines an A/B test configuration
type Experiment struct {
	ID                    uuid.UUID        `json:"id" db:"id"`
	Env                   string           `json:"env" db:"env"`
	TenantID              *uuid.UUID       `json:"tenant_id,omitempty" db:"tenant_id"`
	Name                  string           `json:"name" db:"name"`
	OptimizationID        uuid.UUID        `json:"optimization_id" db:"optimization_id"`
	ControlChangeSetID    uuid.UUID        `json:"control_changeset_id" db:"control_changeset_id"`
	TreatmentChangeSetID  uuid.UUID        `json:"treatment_changeset_id" db:"treatment_changeset_id"`
	TrafficSplitControl   float64          `json:"traffic_split_control" db:"traffic_split_control"`
	TrafficSplitTreatment float64          `json:"traffic_split_treatment" db:"traffic_split_treatment"`
	Status                ExperimentStatus `json:"status" db:"status"`
	CreatedAt             time.Time        `json:"created_at" db:"created_at"`
	StartedAt             *time.Time       `json:"started_at,omitempty" db:"started_at"`
	StoppedAt             *time.Time       `json:"stopped_at,omitempty" db:"stopped_at"`
	CreatedBy             *string          `json:"created_by,omitempty" db:"created_by"`
}

// ExperimentMetrics captures performance for a variant
type ExperimentMetrics struct {
	ExperimentID          uuid.UUID `json:"experiment_id" db:"experiment_id"`
	Variant               string    `json:"variant" db:"variant"` // control | treatment
	WindowStart           time.Time `json:"window_start" db:"window_start"`
	WindowEnd             time.Time `json:"window_end" db:"window_end"`
	Queries               int       `json:"queries" db:"queries"`
	AvgLatencyMs          float64   `json:"avg_latency_ms" db:"avg_latency_ms"`
	P95LatencyMs          float64   `json:"p95_latency_ms" db:"p95_latency_ms"`
	ErrorCount            int       `json:"error_count" db:"error_count"`
	CorrectnessMismatches int       `json:"correctness_mismatches" db:"correctness_mismatches"`
}

// ============================================================================
// Experiment Repository
// ============================================================================

type ExperimentRepository interface {
	Create(ctx context.Context, exp *Experiment) error
	Get(ctx context.Context, id uuid.UUID) (*Experiment, error)
	ListActive(ctx context.Context, env string) ([]Experiment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status ExperimentStatus) error
}

type experimentRepo struct {
	db *sqlx.DB
}

func NewExperimentRepository(db *sqlx.DB) ExperimentRepository {
	return &experimentRepo{db: db}
}

func (r *experimentRepo) Create(ctx context.Context, exp *Experiment) error {
	query := `
		INSERT INTO aso.experiments (
			id, env, tenant_id, name, optimization_id, 
			control_changeset_id, treatment_changeset_id, 
			traffic_split_control, traffic_split_treatment, 
			status, created_at, created_by
		) VALUES (
			:id, :env, :tenant_id, :name, :optimization_id,
			:control_changeset_id, :treatment_changeset_id,
			:traffic_split_control, :traffic_split_treatment,
			:status, :created_at, :created_by
		)
	`
	_, err := r.db.NamedExecContext(ctx, query, exp)
	return err
}

func (r *experimentRepo) Get(ctx context.Context, id uuid.UUID) (*Experiment, error) {
	var exp Experiment
	err := r.db.GetContext(ctx, &exp, "SELECT * FROM aso.experiments WHERE id = $1", id)
	return &exp, err
}

func (r *experimentRepo) ListActive(ctx context.Context, env string) ([]Experiment, error) {
	var exps []Experiment
	err := r.db.SelectContext(ctx, &exps, "SELECT * FROM aso.experiments WHERE env = $1 AND status = 'running'", env)
	return exps, err
}

func (r *experimentRepo) UpdateStatus(ctx context.Context, id uuid.UUID, status ExperimentStatus) error {
	now := time.Now()
	var updateQuery string

	switch status {
	case ExpStatusRunning:
		updateQuery = "UPDATE aso.experiments SET status = $1, started_at = $2 WHERE id = $3"
	case ExpStatusStopped, ExpStatusCompleted:
		updateQuery = "UPDATE aso.experiments SET status = $1, stopped_at = $2 WHERE id = $3"
	default:
		return fmt.Errorf("invalid status transition")
	}

	_, err := r.db.ExecContext(ctx, updateQuery, status, now, id)
	return err
}

// ============================================================================
// Experiment Router
// ============================================================================

// ExperimentRouter determines which model variant to serve
type ExperimentRouter interface {
	Route(ctx context.Context, req BOSQLRequest) (variant string, model ModelVersion)
}

type experimentRouter struct {
	repo ExperimentRepository
}

func NewExperimentRouter(repo ExperimentRepository) ExperimentRouter {
	return &experimentRouter{repo: repo}
}

func (r *experimentRouter) Route(ctx context.Context, req BOSQLRequest) (string, ModelVersion) {
	// Look for active experiment for this tenant/env
	// Note: In a real system, we'd cache this heavily
	exps, err := r.repo.ListActive(ctx, req.Env)
	if err != nil || len(exps) == 0 {
		return "none", CurrentModelVersion
	}

	// Simplification: just pick the first matching experiment
	// Real logic: match on BO, specific criteria, etc.
	// We'll assume if there's an experiment running in this env/tenant, we should check it
	var activeExp *Experiment
	for i := range exps {
		if exps[i].TenantID == nil || (req.TenantID != nil && *exps[i].TenantID == *req.TenantID) {
			activeExp = &exps[i]
			break
		}
	}

	if activeExp == nil {
		return "none", CurrentModelVersion
	}

	// Deterministic hashing for sticky sessions
	hashInput := req.CurrentUserID
	if hashInput == "" {
		hashInput = "anonymous" // Fallback
	}
	hash := crc32.ChecksumIEEE([]byte(hashInput))
	split := float64(hash%100) / 100.0

	if split < activeExp.TrafficSplitControl {
		return "control", ModelVersion{
			ChangeSetID:  activeExp.ControlChangeSetID,
			Optimization: nil, // Control usually implies baseline
		}
	}

	// Helper to fetch optimization details would go here if needed
	// For now we return the treatment changeset
	return "treatment", ModelVersion{
		ChangeSetID: activeExp.TreatmentChangeSetID,
		// Optimization would be linked here
	}
}
