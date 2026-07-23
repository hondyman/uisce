package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	// Trino driver temporarily disabled due to Go version compatibility issues
	// _ "github.com/trinodb/trino-go-client/trino"
)

// TrinoAuditQuerier queries Iceberg audit tables via Trino
type TrinoAuditQuerier struct {
	db *sql.DB
}

// NewTrinoAuditQuerier creates a new Trino-backed audit querier
func NewTrinoAuditQuerier(trinoHost string, trinoPort int, catalog, schema string) (*TrinoAuditQuerier, error) {
	dsn := fmt.Sprintf("http://admin@%s:%d?catalog=%s&schema=%s",
		trinoHost, trinoPort, catalog, schema)

	db, err := sql.Open("trino", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to trino: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping trino: %w", err)
	}

	return &TrinoAuditQuerier{
		db: db,
	}, nil
}

// JobRunQueryParams defines query parameters for job run searches
type JobRunQueryParams struct {
	TenantID       string
	JobID          string
	Status         string
	SemanticTermID string
	StartDate      time.Time
	EndDate        time.Time
	Limit          int
}

// QueryJobRuns queries scheduler job runs with multi-tenant scoping
func (q *TrinoAuditQuerier) QueryJobRuns(ctx context.Context, params JobRunQueryParams) ([]SchedulerJobRun, error) {
	query := `
		SELECT 
			run_id, job_id, dag_id, tenant_id, start_ts, end_ts, status,
			error_message, semantic_context, compliance_context, 
			slo_context, ai_narrative, metadata,
			_ingest_ts, _source_service, _schema_version
		FROM scheduler_job_runs
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	// CRITICAL: Always enforce tenant scoping
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for audit queries")
	}
	query += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
	args = append(args, params.TenantID)
	argIdx++

	if params.JobID != "" {
		query += fmt.Sprintf(" AND job_id = $%d", argIdx)
		args = append(args, params.JobID)
		argIdx++
	}

	if params.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, params.Status)
		argIdx++
	}

	if params.SemanticTermID != "" {
		query += fmt.Sprintf(" AND JSON_EXTRACT_SCALAR(semantic_context, '$.semantic_term_id') = $%d", argIdx)
		args = append(args, params.SemanticTermID)
		argIdx++
	}

	if !params.StartDate.IsZero() {
		query += fmt.Sprintf(" AND start_ts >= $%d", argIdx)
		args = append(args, params.StartDate)
		argIdx++
	}

	if !params.EndDate.IsZero() {
		query += fmt.Sprintf(" AND start_ts <= $%d", argIdx)
		args = append(args, params.EndDate)
		argIdx++
	}

	query += " ORDER BY start_ts DESC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query job runs: %w", err)
	}
	defer rows.Close()

	var results []SchedulerJobRun
	for rows.Next() {
		var r SchedulerJobRun
		var semanticCtx, complianceCtx, sloCtx, aiNarrative, metadata sql.NullString

		err := rows.Scan(
			&r.RunID, &r.JobID, &r.DagID, &r.TenantID, &r.StartTS, &r.EndTS, &r.Status,
			&r.ErrorMessage, &semanticCtx, &complianceCtx, &sloCtx, &aiNarrative, &metadata,
			&r.IngestTS, &r.SourceService, &r.SchemaVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan job run: %w", err)
		}

		if semanticCtx.Valid {
			r.SemanticContext = json.RawMessage(semanticCtx.String)
		}
		if complianceCtx.Valid {
			r.ComplianceContext = json.RawMessage(complianceCtx.String)
		}
		if sloCtx.Valid {
			r.SLOContext = json.RawMessage(sloCtx.String)
		}
		if aiNarrative.Valid {
			r.AINarrative = json.RawMessage(aiNarrative.String)
		}
		if metadata.Valid {
			r.Metadata = json.RawMessage(metadata.String)
		}

		results = append(results, r)
	}

	return results, nil
}

// ChangeSetQueryParams defines query parameters for changeset searches
type ChangeSetQueryParams struct {
	TenantID       string
	Type           string
	Actor          string
	Status         string
	SemanticTermID string
	StartDate      time.Time
	EndDate        time.Time
	Limit          int
}

// QueryChangeSets queries governance changesets with multi-tenant scoping
func (q *TrinoAuditQuerier) QueryChangeSets(ctx context.Context, params ChangeSetQueryParams) ([]GovernanceChangeSet, error) {
	query := `
		SELECT 
			changeset_id, type, actor, tenant_id, created_at,
			payload_old, payload_new, semantic_impact, compliance_impact, tenant_impact,
			ai_summary, ai_risk, approvers, status,
			_ingest_ts, _source_service, _schema_version
		FROM governance_changesets
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	// Support both tenant-specific and cross-tenant queries for internal users
	if params.TenantID != "" {
		query += fmt.Sprintf(" AND (tenant_id = $%d OR JSON_EXTRACT_SCALAR(tenant_impact, '$.tenants[*]') LIKE '%%' || $%d || '%%')", argIdx, argIdx)
		args = append(args, params.TenantID)
		argIdx++
	}

	if params.Type != "" {
		query += fmt.Sprintf(" AND type = $%d", argIdx)
		args = append(args, params.Type)
		argIdx++
	}

	if params.Actor != "" {
		query += fmt.Sprintf(" AND actor = $%d", argIdx)
		args = append(args, params.Actor)
		argIdx++
	}

	if params.Status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, params.Status)
		argIdx++
	}

	if params.SemanticTermID != "" {
		query += fmt.Sprintf(" AND JSON_EXTRACT_SCALAR(semantic_impact, '$.semantic_term_id') = $%d", argIdx)
		args = append(args, params.SemanticTermID)
		argIdx++
	}

	if !params.StartDate.IsZero() {
		query += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, params.StartDate)
		argIdx++
	}

	if !params.EndDate.IsZero() {
		query += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, params.EndDate)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query changesets: %w", err)
	}
	defer rows.Close()

	var results []GovernanceChangeSet
	for rows.Next() {
		var r GovernanceChangeSet
		var payloadOld, payloadNew, semanticImpact, complianceImpact, tenantImpact sql.NullString
		var aiSummary, aiRisk sql.NullString
		var approvers string

		err := rows.Scan(
			&r.ChangesetID, &r.Type, &r.Actor, &r.TenantID, &r.CreatedAt,
			&payloadOld, &payloadNew, &semanticImpact, &complianceImpact, &tenantImpact,
			&aiSummary, &aiRisk, &approvers, &r.Status,
			&r.IngestTS, &r.SourceService, &r.SchemaVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan changeset: %w", err)
		}

		if payloadOld.Valid {
			r.PayloadOld = json.RawMessage(payloadOld.String)
		}
		if payloadNew.Valid {
			r.PayloadNew = json.RawMessage(payloadNew.String)
		}
		if semanticImpact.Valid {
			r.SemanticImpact = json.RawMessage(semanticImpact.String)
		}
		if complianceImpact.Valid {
			r.ComplianceImpact = json.RawMessage(complianceImpact.String)
		}
		if tenantImpact.Valid {
			r.TenantImpact = json.RawMessage(tenantImpact.String)
		}
		if aiSummary.Valid {
			r.AISummary = json.RawMessage(aiSummary.String)
		}
		if aiRisk.Valid {
			r.AIRisk = json.RawMessage(aiRisk.String)
		}

		// Parse approvers array
		if err := json.Unmarshal([]byte(approvers), &r.Approvers); err != nil {
			r.Approvers = []string{}
		}

		results = append(results, r)
	}

	return results, nil
}

// ComplianceViolationQueryParams defines query parameters for violation searches
type ComplianceViolationQueryParams struct {
	TenantID      string
	Severity      string
	ViolationType string
	PIIExposed    *bool
	Remediated    *bool
	StartDate     time.Time
	EndDate       time.Time
	Limit         int
}

// QueryComplianceViolations queries compliance violations with multi-tenant scoping
func (q *TrinoAuditQuerier) QueryComplianceViolations(ctx context.Context, params ComplianceViolationQueryParams) ([]ComplianceViolation, error) {
	query := `
		SELECT 
			violation_id, tenant_id, job_run_id, violated_at, remediated_at,
			violation_type, severity, pii_exposed, affected_records,
			compliance_refs, narrative, metadata,
			_ingest_ts, _source_service, _schema_version
		FROM compliance_violations
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	// CRITICAL: Always enforce tenant scoping
	if params.TenantID == "" {
		return nil, fmt.Errorf("tenant_id is required for compliance queries")
	}
	query += fmt.Sprintf(" AND tenant_id = $%d", argIdx)
	args = append(args, params.TenantID)
	argIdx++

	if params.Severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argIdx)
		args = append(args, params.Severity)
		argIdx++
	}

	if params.ViolationType != "" {
		query += fmt.Sprintf(" AND violation_type = $%d", argIdx)
		args = append(args, params.ViolationType)
		argIdx++
	}

	if params.PIIExposed != nil {
		query += fmt.Sprintf(" AND pii_exposed = $%d", argIdx)
		args = append(args, *params.PIIExposed)
		argIdx++
	}

	if params.Remediated != nil {
		if *params.Remediated {
			query += " AND remediated_at IS NOT NULL"
		} else {
			query += " AND remediated_at IS NULL"
		}
	}

	if !params.StartDate.IsZero() {
		query += fmt.Sprintf(" AND violated_at >= $%d", argIdx)
		args = append(args, params.StartDate)
		argIdx++
	}

	if !params.EndDate.IsZero() {
		query += fmt.Sprintf(" AND violated_at <= $%d", argIdx)
		args = append(args, params.EndDate)
		argIdx++
	}

	query += " ORDER BY violated_at DESC"

	if params.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", params.Limit)
	} else {
		query += " LIMIT 100"
	}

	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query compliance violations: %w", err)
	}
	defer rows.Close()

	var results []ComplianceViolation
	for rows.Next() {
		var r ComplianceViolation
		var jobRunID sql.NullString
		var remediatedAt sql.NullTime
		var complianceRefs, metadata sql.NullString

		err := rows.Scan(
			&r.ViolationID, &r.TenantID, &jobRunID, &r.ViolatedAt, &remediatedAt,
			&r.ViolationType, &r.Severity, &r.PIIExposed, &r.AffectedRecords,
			&complianceRefs, &r.Narrative, &metadata,
			&r.IngestTS, &r.SourceService, &r.SchemaVersion,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan compliance violation: %w", err)
		}

		if jobRunID.Valid {
			r.JobRunID = jobRunID.String
		}
		if remediatedAt.Valid {
			r.RemediatedAt = remediatedAt.Time
		}
		if complianceRefs.Valid {
			if err := json.Unmarshal([]byte(complianceRefs.String), &r.ComplianceRefs); err != nil {
				r.ComplianceRefs = []string{}
			}
		}
		if metadata.Valid {
			r.Metadata = json.RawMessage(metadata.String)
		}

		results = append(results, r)
	}

	return results, nil
}

// QuerySemanticLineage performs time-travel query on semantic snapshots
func (q *TrinoAuditQuerier) QuerySemanticLineage(ctx context.Context, tenantID, semanticTermID string, version int) (*SemanticSnapshot, error) {
	query := `
		SELECT 
			snapshot_id, semantic_term_id, version, timestamp, definition,
			business_term_id, tenant_id, compliance, lineage, metadata,
			_ingest_ts, _source_service, _schema_version
		FROM semantic_snapshots
		WHERE semantic_term_id = $1
		  AND version = $2
	`

	// If tenant-scoped, add tenant filter
	if tenantID != "" {
		query += " AND (tenant_id = $3 OR tenant_id IS NULL)"
	}

	query += " LIMIT 1"

	var r SemanticSnapshot
	var compliance, lineage, metadata sql.NullString
	var tenID sql.NullString

	args := []interface{}{semanticTermID, version}
	if tenantID != "" {
		args = append(args, tenantID)
	}

	err := q.db.QueryRowContext(ctx, query, args...).Scan(
		&r.SnapshotID, &r.SemanticTermID, &r.Version, &r.Timestamp, &r.Definition,
		&r.BusinessTermID, &tenID, &compliance, &lineage, &metadata,
		&r.IngestTS, &r.SourceService, &r.SchemaVersion,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to query semantic lineage: %w", err)
	}

	if tenID.Valid {
		r.TenantID = tenID.String
	}
	if compliance.Valid {
		r.Compliance = json.RawMessage(compliance.String)
	}
	if lineage.Valid {
		r.Lineage = json.RawMessage(lineage.String)
	}
	if metadata.Valid {
		r.Metadata = json.RawMessage(metadata.String)
	}

	return &r, nil
}

// Close closes the Trino connection
func (q *TrinoAuditQuerier) Close() error {
	return q.db.Close()
}
