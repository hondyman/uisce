package datapipeline

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ModeLoader marks data_pipeline_definitions rows whose dag_json is a Spec.
const ModeLoader = "loader"

// ErrNotFound is returned for a pipeline or run the tenant cannot see.
var ErrNotFound = errors.New("not found")

// Definition is a saved pipeline.
type Definition struct {
	ID             string    `db:"id" json:"id"`
	TenantID       string    `db:"tenant_id" json:"tenant_id"`
	Name           string    `db:"name" json:"name"`
	Description    string    `db:"description" json:"description"`
	Spec           Spec      `db:"-" json:"spec"`
	SpecJSON       []byte    `db:"dag_json" json:"-"`
	IsActive       bool      `db:"is_active" json:"is_active"`
	CreatedBy      string    `db:"created_by" json:"created_by"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	LastModifiedAt time.Time `db:"last_modified_at" json:"last_modified_at"`
}

// RunRecord is one execution.
type RunRecord struct {
	ID         string          `db:"id" json:"id"`
	PipelineID string          `db:"pipeline_id" json:"pipeline_id"`
	Status     string          `db:"status" json:"status"` // queued|running|completed|completed_with_errors|failed
	StartTime  time.Time       `db:"start_time" json:"start_time"`
	EndTime    *time.Time      `db:"end_time" json:"end_time,omitempty"`
	RecordsIn  int64           `db:"total_records_in" json:"records_in"`
	RecordsOut int64           `db:"total_records_out" json:"records_out"`
	Errors     int64           `db:"total_errors" json:"errors"`
	ErrorJSON  json.RawMessage `db:"error_details" json:"errors_sample"`
	Steps      []NodeStats     `db:"-" json:"steps,omitempty"`
}

// Store persists definitions and runs, always scoped to one tenant.
type Store struct{ DB *sqlx.DB }

func (s *Store) List(ctx context.Context, tenantID string) ([]Definition, error) {
	var out []Definition
	err := s.DB.SelectContext(ctx, &out, `
		SELECT id, tenant_id, name, COALESCE(description,'') AS description, dag_json, is_active,
		       COALESCE(created_by,'') AS created_by, created_at, last_modified_at
		FROM data_pipeline_definitions
		WHERE tenant_id = $1 AND mode = $2 AND is_active
		ORDER BY name`, tenantID, ModeLoader)
	if err != nil {
		return nil, err
	}
	for i := range out {
		if err := json.Unmarshal(out[i].SpecJSON, &out[i].Spec); err != nil {
			return nil, fmt.Errorf("pipeline %s: stored spec is invalid: %w", out[i].ID, err)
		}
	}
	return out, nil
}

func (s *Store) Get(ctx context.Context, tenantID, id string) (*Definition, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrNotFound
	}
	var d Definition
	err := s.DB.GetContext(ctx, &d, `
		SELECT id, tenant_id, name, COALESCE(description,'') AS description, dag_json, is_active,
		       COALESCE(created_by,'') AS created_by, created_at, last_modified_at
		FROM data_pipeline_definitions
		WHERE id = $1 AND tenant_id = $2 AND mode = $3 AND is_active`, id, tenantID, ModeLoader)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &d, json.Unmarshal(d.SpecJSON, &d.Spec)
}

// Save creates (empty ID) or updates a definition. The spec is stored as
// given; validity is checked at run time and by Validate in the editor, so
// an analyst can save work in progress.
func (s *Store) Save(ctx context.Context, tenantID, userID string, d *Definition) error {
	b, err := json.Marshal(d.Spec)
	if err != nil {
		return err
	}
	if d.ID == "" {
		d.ID = uuid.NewString()
		return s.DB.QueryRowxContext(ctx, `
			INSERT INTO data_pipeline_definitions (id, tenant_id, name, description, mode, dag_json, batch_size, error_policy, created_by)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING tenant_id, is_active, created_by, created_at, last_modified_at`,
			d.ID, tenantID, d.Name, d.Description, ModeLoader, b, batchOrDefault(d.Spec), policyOrDefault(d.Spec), userID,
		).Scan(&d.TenantID, &d.IsActive, &d.CreatedBy, &d.CreatedAt, &d.LastModifiedAt)
	}
	err = s.DB.QueryRowxContext(ctx, `
		UPDATE data_pipeline_definitions
		SET name = $3, description = $4, dag_json = $5, batch_size = $6, error_policy = $7, last_modified_at = now()
		WHERE id = $1 AND tenant_id = $2 AND mode = $8 AND is_active
		RETURNING tenant_id, is_active, COALESCE(created_by,''), created_at, last_modified_at`,
		d.ID, tenantID, d.Name, d.Description, b, batchOrDefault(d.Spec), policyOrDefault(d.Spec), ModeLoader,
	).Scan(&d.TenantID, &d.IsActive, &d.CreatedBy, &d.CreatedAt, &d.LastModifiedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// Delete deactivates a definition (run history stays).
func (s *Store) Delete(ctx context.Context, tenantID, id string) error {
	res, err := s.DB.ExecContext(ctx, `UPDATE data_pipeline_definitions SET is_active = false, last_modified_at = now()
		WHERE id = $1 AND tenant_id = $2 AND mode = $3 AND is_active`, id, tenantID, ModeLoader)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

func batchOrDefault(s Spec) int {
	if s.BatchSize > 0 {
		return s.BatchSize
	}
	return defaultBatchSize
}

func policyOrDefault(s Spec) string {
	if s.ErrorPolicy != "" {
		return s.ErrorPolicy
	}
	return ErrorPolicySkipAndLog
}

// CreateRun records a queued run with a snapshot of the spec it will execute,
// so later edits to the pipeline never change what a run did.
func (s *Store) CreateRun(ctx context.Context, tenantID string, d *Definition) (string, error) {
	id := uuid.NewString()
	b, err := json.Marshal(d.Spec)
	if err != nil {
		return "", err
	}
	_, err = s.DB.ExecContext(ctx, `
		INSERT INTO data_pipeline_runs (id, tenant_id, pipeline_id, status, start_time, dag_json)
		VALUES ($1, $2, $3, 'queued', now(), $4)`, id, tenantID, d.ID, b)
	return id, err
}

// RunSpec loads a run's snapshot spec.
func (s *Store) RunSpec(ctx context.Context, tenantID, runID string) (*Spec, error) {
	var raw []byte
	err := s.DB.GetContext(ctx, &raw, `SELECT dag_json FROM data_pipeline_runs WHERE id = $1 AND tenant_id = $2`, runID, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var spec Spec
	return &spec, json.Unmarshal(raw, &spec)
}

func (s *Store) MarkRunning(ctx context.Context, tenantID, runID string) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE data_pipeline_runs SET status = 'running', start_time = now(), updated_at = now()
		WHERE id = $1 AND tenant_id = $2`, runID, tenantID)
	return err
}

// FinishRun records the outcome.
func (s *Store) FinishRun(ctx context.Context, tenantID, runID string, sum *Summary, runErr error) error {
	status := "completed"
	if sum != nil && sum.Errors > 0 {
		status = "completed_with_errors"
	}
	if runErr != nil {
		status = "failed"
	}
	var in, out, errs int64
	if sum != nil {
		in, out, errs = sum.RecordsIn, sum.RecordsOut, sum.Errors
	}
	_, err := s.DB.ExecContext(ctx, `
		UPDATE data_pipeline_runs
		SET status = $3, end_time = now(), updated_at = now(),
		    total_records_in = $4, total_records_out = $5, total_errors = $6,
		    error_details = CASE WHEN $7::text = '' THEN error_details
		                         ELSE error_details || jsonb_build_array(jsonb_build_object('run_error', $7::text)) END
		WHERE id = $1 AND tenant_id = $2`,
		runID, tenantID, status, in, out, errs, errString(runErr))
	return err
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Store) ListRuns(ctx context.Context, tenantID, pipelineID string, limit int) ([]RunRecord, error) {
	var out []RunRecord
	err := s.DB.SelectContext(ctx, &out, `
		SELECT id, pipeline_id, status, start_time, end_time, total_records_in, total_records_out, total_errors,
		       COALESCE(error_details, '[]'::jsonb) AS error_details
		FROM data_pipeline_runs WHERE tenant_id = $1 AND pipeline_id = $2
		ORDER BY start_time DESC LIMIT $3`, tenantID, pipelineID, limit)
	return out, err
}

func (s *Store) GetRun(ctx context.Context, tenantID, runID string) (*RunRecord, error) {
	if _, err := uuid.Parse(runID); err != nil {
		return nil, ErrNotFound
	}
	var r RunRecord
	err := s.DB.GetContext(ctx, &r, `
		SELECT id, pipeline_id, status, start_time, end_time, total_records_in, total_records_out, total_errors,
		       COALESCE(error_details, '[]'::jsonb) AS error_details
		FROM data_pipeline_runs WHERE id = $1 AND tenant_id = $2`, runID, tenantID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.DB.QueryxContext(ctx, `
		SELECT node_id, node_label, node_type, records_in, records_out, records_error, duration_ms, status,
		       COALESCE(error_message,''), step_order_index
		FROM data_pipeline_step_telemetry WHERE run_id = $1 ORDER BY step_order_index`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var st NodeStats
		var ms int64
		if err := rows.Scan(&st.NodeID, &st.Label, &st.Type, &st.In, &st.Out, &st.Errors, &ms, &st.Status, &st.Err, &st.OrderIndex); err != nil {
			return nil, err
		}
		st.Duration = time.Duration(ms) * time.Millisecond
		r.Steps = append(r.Steps, st)
	}
	return &r, rows.Err()
}

// maxErrorSamples caps the row errors kept per run; counts stay exact.
const maxErrorSamples = 500

// dbRecorder persists telemetry and a sample of rejected rows for one run.
type dbRecorder struct {
	store  *Store
	tenant string
	runID  string
	mu     sync.Mutex
	sample []map[string]any
}

func (s *Store) Recorder(tenantID, runID string) Recorder {
	return &dbRecorder{store: s, tenant: tenantID, runID: runID}
}

func (r *dbRecorder) add(kind string, rj Reject) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.sample) >= maxErrorSamples {
		return
	}
	r.sample = append(r.sample, map[string]any{
		"kind": kind, "node_id": rj.NodeID, "row": rj.Row.Num, "field": rj.Field, "reason": rj.Reason,
	})
}

func (r *dbRecorder) Rejected(_ context.Context, rj Reject) { r.add("error", rj) }
func (r *dbRecorder) Warned(_ context.Context, rj Reject)   { r.add("warning", rj) }

func (r *dbRecorder) NodeDone(ctx context.Context, st NodeStats) {
	rps := 0.0
	if st.Duration > 0 {
		rps = float64(st.In) / st.Duration.Seconds()
	}
	_, _ = r.store.DB.ExecContext(ctx, `
		INSERT INTO data_pipeline_step_telemetry
		    (run_id, node_id, node_label, node_type, records_in, records_out, records_error, duration_ms, rows_per_sec, status, error_message, step_order_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULLIF($11,''), $12)
		ON CONFLICT (run_id, node_id) DO UPDATE SET
		    records_in = EXCLUDED.records_in, records_out = EXCLUDED.records_out, records_error = EXCLUDED.records_error,
		    duration_ms = EXCLUDED.duration_ms, rows_per_sec = EXCLUDED.rows_per_sec, status = EXCLUDED.status,
		    error_message = EXCLUDED.error_message`,
		r.runID, st.NodeID, st.Label, st.Type, st.In, st.Out, st.Errors, st.Duration.Milliseconds(), rps, st.Status, st.Err, st.OrderIndex)
}

// Flush writes the error sample to the run.
func (r *dbRecorder) Flush(ctx context.Context) error {
	r.mu.Lock()
	b, _ := json.Marshal(r.sample)
	r.mu.Unlock()
	_, err := r.store.DB.ExecContext(ctx, `UPDATE data_pipeline_runs SET error_details = $3::jsonb, updated_at = now()
		WHERE id = $1 AND tenant_id = $2`, r.runID, r.tenant, string(b))
	return err
}
