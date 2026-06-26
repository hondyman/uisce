package audit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Repository defines all audit data access operations
type Repository interface {
	// List audit events with filtering and pagination
	ListEvents(ctx context.Context, scope TenantScope, filters QueryFilters) ([]AuditEvent, int, error)

	// Get all audit events for a specific entity
	GetEntityAudit(ctx context.Context, scope TenantScope, entityType, entityID string, from, to time.Time, limit, offset int) (*EntityAudit, error)

	// List incident clusters
	ListIncidents(ctx context.Context, scope TenantScope, from, to time.Time, limit, offset int) ([]IncidentCluster, error)

	// Get single incident details
	GetIncident(ctx context.Context, scope TenantScope, incidentID string) (*IncidentCluster, error)

	// List compliance events
	ListComplianceEvents(ctx context.Context, scope TenantScope, from, to time.Time, violationTypes []string, limit, offset int) ([]ComplianceEvent, error)

	// Global admin dashboards
	GetGlobalAdminDashboard(ctx context.Context, from, to time.Time) (*GlobalAdminDashboard, error)

	// Global ops dashboards
	GetGlobalOpsDashboard(ctx context.Context, scope TenantScope, from, to time.Time) (*GlobalOpsDashboard, error)

	// Tenant admin dashboards
	GetTenantAdminDashboard(ctx context.Context, tenantID string, from, to time.Time) (*TenantAdminDashboard, error)

	// Tenant ops dashboards
	GetTenantOpsDashboard(ctx context.Context, tenantID string, from, to time.Time) (*TenantOpsDashboard, error)
}

// TrinoRepository implements Repository using Trino queries
type TrinoRepository struct {
	db *sql.DB
}

// NewTrinoRepository creates a new Trino-backed repository
func NewTrinoRepository(db *sql.DB) *TrinoRepository {
	return &TrinoRepository{db: db}
}

// ListEvents queries audit events from all tables
func (tr *TrinoRepository) ListEvents(ctx context.Context, scope TenantScope, filters QueryFilters) ([]AuditEvent, int, error) {
	query, args := buildListEventsQuery(scope, filters)

	rows, err := tr.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		if err := rows.Scan(
			&e.ID, &e.Type, &e.TenantID, &e.Timestamp, &e.Status, &e.ArtifactType, &e.ArtifactID,
			&e.Title, &e.Actor, &e.RiskLevel, &e.SemanticContext, &e.ComplianceContext,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, e)
	}

	// Get total count
	countQuery := strings.Replace(query, "SELECT", "SELECT COUNT(*) FROM (SELECT 1", 1)
	countQuery += ") t"
	var total int
	if err := tr.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count events: %w", err)
	}

	return events, total, nil
}

// GetEntityAudit retrieves all audit events related to an entity
func (tr *TrinoRepository) GetEntityAudit(ctx context.Context, scope TenantScope, entityType, entityID string, from, to time.Time, limit, offset int) (*EntityAudit, error) {
	query := buildEntityAuditQuery(entityType, entityID, scope, from, to, limit, offset)

	rows, err := tr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query entity audit: %w", err)
	}
	defer rows.Close()

	result := &EntityAudit{
		EntityType: entityType,
		EntityID:   entityID,
		Timeline:   []AuditEvent{},
		Changes:    []AuditEvent{},
		Compliance: []AuditEvent{},
	}

	for rows.Next() {
		var e AuditEvent
		var eventCategory string
		if err := rows.Scan(
			&eventCategory, &e.ID, &e.Type, &e.TenantID, &e.Timestamp,
			&e.Status, &e.Title, &e.AINarrative,
		); err != nil {
			return nil, fmt.Errorf("failed to scan entity audit: %w", err)
		}

		switch eventCategory {
		case "change":
			result.Changes = append(result.Changes, e)
		case "compliance":
			result.Compliance = append(result.Compliance, e)
		default:
			result.Timeline = append(result.Timeline, e)
		}
	}

	result.LastUpdated = time.Now()
	return result, nil
}

// ListIncidents retrieves incident clusters
func (tr *TrinoRepository) ListIncidents(ctx context.Context, scope TenantScope, from, to time.Time, limit, offset int) ([]IncidentCluster, error) {
	query := buildIncidentsQuery(scope, from, to, limit, offset)

	rows, err := tr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query incidents: %w", err)
	}
	defer rows.Close()

	var incidents []IncidentCluster
	for rows.Next() {
		var ic IncidentCluster
		if err := rows.Scan(
			&ic.ID, &ic.Status, &ic.EventCount, &ic.AIRootCause, &ic.AINarrative,
			&ic.TimeWindow.Start, &ic.TimeWindow.End,
		); err != nil {
			return nil, fmt.Errorf("failed to scan incident: %w", err)
		}
		incidents = append(incidents, ic)
	}

	return incidents, nil
}

// GetIncident retrieves a single incident with full details
func (tr *TrinoRepository) GetIncident(ctx context.Context, scope TenantScope, incidentID string) (*IncidentCluster, error) {
	query := buildIncidentDetailQuery(incidentID, scope)

	var ic IncidentCluster
	err := tr.db.QueryRowContext(ctx, query).Scan(
		&ic.ID, &ic.Status, &ic.EventCount, &ic.AIRootCause, &ic.AINarrative,
		&ic.TimeWindow.Start, &ic.TimeWindow.End,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query incident detail: %w", err)
	}

	return &ic, nil
}

// ListComplianceEvents retrieves compliance-related audit events
func (tr *TrinoRepository) ListComplianceEvents(ctx context.Context, scope TenantScope, from, to time.Time, violationTypes []string, limit, offset int) ([]ComplianceEvent, error) {
	query := buildComplianceEventsQuery(scope, from, to, violationTypes, limit, offset)

	rows, err := tr.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query compliance events: %w", err)
	}
	defer rows.Close()

	var events []ComplianceEvent
	for rows.Next() {
		var ce ComplianceEvent
		if err := rows.Scan(
			&ce.ID, &ce.TenantID, &ce.Timestamp, &ce.ViolationType,
			&ce.Status, &ce.ArtifactType, &ce.ArtifactID, &ce.Severity,
			&ce.AIExplanation,
		); err != nil {
			return nil, fmt.Errorf("failed to scan compliance event: %w", err)
		}
		events = append(events, ce)
	}

	return events, nil
}

// GetGlobalAdminDashboard returns platform-wide metrics
func (tr *TrinoRepository) GetGlobalAdminDashboard(ctx context.Context, from, to time.Time) (*GlobalAdminDashboard, error) {
	dashboard := &GlobalAdminDashboard{
		FailedRunsLastDay:    make(map[string]int),
		ComplianceViolations: make(map[string]int),
		SLOBreachRisk:        make(map[string]float64),
		PlatformHealth:       make(map[string]interface{}),
	}

	// Count distinct tenants
	tenantCountSQL := `SELECT COUNT(DISTINCT tenant_id) as cnt FROM iceberg.audit.events WHERE timestamp >= ? AND timestamp < ?`
	var tenantCount int
	err := tr.db.QueryRowContext(ctx, tenantCountSQL, from, to).Scan(&tenantCount)
	if err != nil {
		return nil, err
	}
	dashboard.TenantCount = tenantCount

	// Failed runs by tenant (last day)
	failedRunsSQL := `
		SELECT tenant_id, COUNT(*) as failed_count 
		FROM iceberg.audit.events 
		WHERE type IN ('job_run', 'dag_run') 
		  AND status = 'FAILED' 
		  AND timestamp >= ? AND timestamp < ?
		GROUP BY tenant_id
		ORDER BY failed_count DESC
		LIMIT 20`
	rows, err := tr.db.QueryContext(ctx, failedRunsSQL, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tenantID string
		var count int
		if err := rows.Scan(&tenantID, &count); err != nil {
			continue
		}
		dashboard.FailedRunsLastDay[tenantID] = count
	}

	// Compliance violations by tenant
	complianceSQL := `
		SELECT tenant_id, COUNT(*) as violation_count 
		FROM iceberg.audit.events 
		WHERE type = 'compliance_violation' 
		  AND timestamp >= ? AND timestamp < ?
		GROUP BY tenant_id
		ORDER BY violation_count DESC
		LIMIT 20`
	rows2, err := tr.db.QueryContext(ctx, complianceSQL, from, to)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var tenantID string
		var count int
		if err := rows2.Scan(&tenantID, &count); err != nil {
			continue
		}
		dashboard.ComplianceViolations[tenantID] = count
	}

	// High-risk changesets
	changesetSQL := `
		SELECT id, type, tenant_id, timestamp, status, artifact_type, artifact_id, title, actor, risk_level
		FROM iceberg.audit.events 
		WHERE type = 'changeset' 
		  AND risk_level IN ('HIGH', 'CRITICAL')
		  AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp DESC
		LIMIT 10`
	rows3, err := tr.db.QueryContext(ctx, changesetSQL, from, to)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()
	for rows3.Next() {
		var evt AuditEvent
		if err := rows3.Scan(&evt.ID, &evt.Type, &evt.TenantID, &evt.Timestamp, &evt.Status, &evt.ArtifactType, &evt.ArtifactID, &evt.Title, &evt.Actor, &evt.RiskLevel); err != nil {
			continue
		}
		dashboard.HighRiskChangeSets = append(dashboard.HighRiskChangeSets, evt)
	}

	return dashboard, nil
}

// GetGlobalOpsDashboard returns multi-tenant ops metrics
func (tr *TrinoRepository) GetGlobalOpsDashboard(ctx context.Context, scope TenantScope, from, to time.Time) (*GlobalOpsDashboard, error) {
	dashboard := &GlobalOpsDashboard{
		AssignedTenants:          scope,
		IncidentClustersByTenant: make(map[string]int),
		SLOPressure:              make(map[string]float64),
	}

	// Build tenant filter
	tenantFilter := ""
	if !scope.IsGlobal() {
		tenantFilter = fmt.Sprintf("AND tenant_id IN ('%s')", strings.Join(scope, "','"))
	}

	// Count incidents by tenant
	incidentSQL := fmt.Sprintf(`
		SELECT tenant_id, COUNT(*) as incident_count 
		FROM iceberg.audit.events 
		WHERE type = 'incident' 
		  AND status = 'open'
		  AND timestamp >= ? AND timestamp < ?
		  %s
		GROUP BY tenant_id`, tenantFilter)
	rows, err := tr.db.QueryContext(ctx, incidentSQL, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var tenantID string
		var count int
		if err := rows.Scan(&tenantID, &count); err != nil {
			continue
		}
		dashboard.IncidentClustersByTenant[tenantID] = count
	}

	// Count jobs at risk (recent failures)
	jobRiskSQL := fmt.Sprintf(`
		SELECT COUNT(DISTINCT artifact_id) 
		FROM iceberg.audit.events 
		WHERE type = 'job_run' 
		  AND status = 'FAILED'
		  AND timestamp >= ? AND timestamp < ?
		  %s`, tenantFilter)
	var jobsAtRisk int
	err = tr.db.QueryRowContext(ctx, jobRiskSQL, from, to).Scan(&jobsAtRisk)
	if err == nil {
		dashboard.JobsAtRisk = jobsAtRisk
	}

	// Count DAGs under stress
	dagStressSQL := fmt.Sprintf(`
		SELECT COUNT(DISTINCT artifact_id) 
		FROM iceberg.audit.events 
		WHERE type = 'dag_run' 
		  AND status = 'FAILED'
		  AND timestamp >= ? AND timestamp < ?
		  %s`, tenantFilter)
	var dagsUnderStress int
	err = tr.db.QueryRowContext(ctx, dagStressSQL, from, to).Scan(&dagsUnderStress)
	if err == nil {
		dashboard.DAGsUnderStress = dagsUnderStress
	}

	return dashboard, nil
}

// GetTenantAdminDashboard returns tenant-specific metrics
func (tr *TrinoRepository) GetTenantAdminDashboard(ctx context.Context, tenantID string, from, to time.Time) (*TenantAdminDashboard, error) {
	dashboard := &TenantAdminDashboard{
		TenantID:     tenantID,
		TenantHealth: make(map[string]interface{}),
	}

	// Failed runs count
	failedRunsSQL := `
		SELECT COUNT(*) 
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type IN ('job_run', 'dag_run') 
		  AND status = 'FAILED'
		  AND timestamp >= ? AND timestamp < ?`
	var failedRuns int
	err := tr.db.QueryRowContext(ctx, failedRunsSQL, tenantID, from, to).Scan(&failedRuns)
	if err == nil {
		dashboard.FailedRunsLastDay = failedRuns
	}

	// Compliance violations count
	complianceSQL := `
		SELECT COUNT(*) 
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type = 'compliance_violation'
		  AND timestamp >= ? AND timestamp < ?`
	var violations int
	err = tr.db.QueryRowContext(ctx, complianceSQL, tenantID, from, to).Scan(&violations)
	if err == nil {
		dashboard.ComplianceViolations = violations
	}

	// Pending approvals
	approvalSQL := `
		SELECT id, type, tenant_id, timestamp, status, artifact_type, artifact_id, title, actor, risk_level
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type = 'changeset' 
		  AND status = 'PENDING'
		  AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp DESC
		LIMIT 20`
	rows, err := tr.db.QueryContext(ctx, approvalSQL, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var evt AuditEvent
		if err := rows.Scan(&evt.ID, &evt.Type, &evt.TenantID, &evt.Timestamp, &evt.Status, &evt.ArtifactType, &evt.ArtifactID, &evt.Title, &evt.Actor, &evt.RiskLevel); err != nil {
			continue
		}
		dashboard.PendingApprovals = append(dashboard.PendingApprovals, evt)
	}

	// High-risk changesets
	changesetSQL := `
		SELECT id, type, tenant_id, timestamp, status, artifact_type, artifact_id, title, actor, risk_level
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type = 'changeset' 
		  AND risk_level IN ('HIGH', 'CRITICAL')
		  AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp DESC
		LIMIT 10`
	rows2, err := tr.db.QueryContext(ctx, changesetSQL, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var evt AuditEvent
		if err := rows2.Scan(&evt.ID, &evt.Type, &evt.TenantID, &evt.Timestamp, &evt.Status, &evt.ArtifactType, &evt.ArtifactID, &evt.Title, &evt.Actor, &evt.RiskLevel); err != nil {
			continue
		}
		dashboard.HighRiskChangeSets = append(dashboard.HighRiskChangeSets, evt)
	}

	return dashboard, nil
}

// GetTenantOpsDashboard returns tenant ops metrics
func (tr *TrinoRepository) GetTenantOpsDashboard(ctx context.Context, tenantID string, from, to time.Time) (*TenantOpsDashboard, error) {
	dashboard := &TenantOpsDashboard{
		TenantID:          tenantID,
		OperationalHealth: make(map[string]interface{}),
	}

	// Failed runs count
	failedRunsSQL := `
		SELECT COUNT(*) 
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type = 'job_run'
		  AND status = 'FAILED'
		  AND timestamp >= ? AND timestamp < ?`
	var failedRuns int
	err := tr.db.QueryRowContext(ctx, failedRunsSQL, tenantID, from, to).Scan(&failedRuns)
	if err == nil {
		dashboard.FailedRunsLastDay = failedRuns
	}

	// Failed DAGs count
	failedDAGsSQL := `
		SELECT COUNT(*) 
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type = 'dag_run'
		  AND status = 'FAILED'
		  AND timestamp >= ? AND timestamp < ?`
	var failedDAGs int
	err = tr.db.QueryRowContext(ctx, failedDAGsSQL, tenantID, from, to).Scan(&failedDAGs)
	if err == nil {
		dashboard.FailedDAGsLastDay = failedDAGs
	}

	// Open incidents count
	incidentSQL := `
		SELECT COUNT(*) 
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type = 'incident'
		  AND status = 'open'
		  AND timestamp >= ? AND timestamp < ?`
	var openIncidents int
	err = tr.db.QueryRowContext(ctx, incidentSQL, tenantID, from, to).Scan(&openIncidents)
	if err == nil {
		dashboard.OpenIncidents = openIncidents
	}

	// Recent failures
	recentFailuresSQL := `
		SELECT id, type, tenant_id, timestamp, status, artifact_type, artifact_id, title, actor, risk_level
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type IN ('job_run', 'dag_run')
		  AND status = 'FAILED'
		  AND timestamp >= ? AND timestamp < ?
		ORDER BY timestamp DESC
		LIMIT 20`
	rows, err := tr.db.QueryContext(ctx, recentFailuresSQL, tenantID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var evt AuditEvent
		if err := rows.Scan(&evt.ID, &evt.Type, &evt.TenantID, &evt.Timestamp, &evt.Status, &evt.ArtifactType, &evt.ArtifactID, &evt.Title, &evt.Actor, &evt.RiskLevel); err != nil {
			continue
		}
		dashboard.RecentFailures = append(dashboard.RecentFailures, evt)
	}

	// Compliance blocks
	complianceBlockSQL := `
		SELECT COUNT(*) 
		FROM iceberg.audit.events 
		WHERE tenant_id = ? 
		  AND type = 'compliance_violation'
		  AND status = 'BLOCKED'
		  AND timestamp >= ? AND timestamp < ?`
	var complianceBlocks int
	err = tr.db.QueryRowContext(ctx, complianceBlockSQL, tenantID, from, to).Scan(&complianceBlocks)
	if err == nil {
		dashboard.ComplianceBlockCount = complianceBlocks
	}

	return dashboard, nil
}

// Query builders
func buildListEventsQuery(scope TenantScope, filters QueryFilters) (string, []interface{}) {
	var args []interface{}
	var conditions []string

	// Add tenant scope condition
	if !scope.IsGlobal() {
		tenantPlaceholders := make([]string, len(scope))
		for i, t := range scope {
			tenantPlaceholders[i] = "?"
			args = append(args, t)
		}
		conditions = append(conditions, fmt.Sprintf("tenant_id IN (%s)", strings.Join(tenantPlaceholders, ",")))
	}

	// Add time range
	conditions = append(conditions, "timestamp BETWEEN ? AND ?")
	args = append(args, filters.TimeRange.From, filters.TimeRange.To)

	// Add filters
	if len(filters.ArtifactTypes) > 0 {
		placeholders := make([]string, len(filters.ArtifactTypes))
		for i, at := range filters.ArtifactTypes {
			placeholders[i] = "?"
			args = append(args, at)
		}
		conditions = append(conditions, fmt.Sprintf("artifact_type IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filters.Statuses) > 0 {
		placeholders := make([]string, len(filters.Statuses))
		for i, s := range filters.Statuses {
			placeholders[i] = "?"
			args = append(args, s)
		}
		conditions = append(conditions, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
	}

	if len(filters.RiskLevels) > 0 {
		placeholders := make([]string, len(filters.RiskLevels))
		for i, rl := range filters.RiskLevels {
			placeholders[i] = "?"
			args = append(args, rl)
		}
		conditions = append(conditions, fmt.Sprintf("risk_level IN (%s)", strings.Join(placeholders, ",")))
	}

	whereClause := strings.Join(conditions, " AND ")

	query := fmt.Sprintf(`
		SELECT id, type, tenant_id, timestamp, status, artifact_type, artifact_id, 
		       title, actor, risk_level, semantic_context, compliance_context
		FROM iceberg.audit.events
		WHERE %s
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, filters.Limit, filters.Offset)
	return query, args
}

func buildEntityAuditQuery(entityType, entityID string, scope TenantScope, from, to time.Time, limit, offset int) string {
	scopeFilter := ""
	if !scope.IsGlobal() {
		tenants := make([]string, len(scope))
		for i, t := range scope {
			tenants[i] = fmt.Sprintf("'%s'", t)
		}
		scopeFilter = fmt.Sprintf(" AND tenant_id IN (%s)", strings.Join(tenants, ","))
	}

	return fmt.Sprintf(`
		SELECT 
			CASE 
				WHEN type IN ('changeset', 'semantic_snapshot') THEN 'change'
				WHEN type LIKE 'compliance_%%' THEN 'compliance'
				ELSE 'timeline'
			END AS category,
			id, type, tenant_id, timestamp, status, title, ai_narrative
		FROM iceberg.audit.events
		WHERE entity_type = '%s' 
		  AND entity_id = '%s'
		  AND timestamp BETWEEN '%s' AND '%s'
		  %s
		ORDER BY timestamp DESC
		LIMIT %d OFFSET %d
	`, entityType, entityID, from.Format(time.RFC3339), to.Format(time.RFC3339), scopeFilter, limit, offset)
}

func buildIncidentsQuery(scope TenantScope, from, to time.Time, limit, offset int) string {
	scopeFilter := ""
	if !scope.IsGlobal() {
		tenants := make([]string, len(scope))
		for i, t := range scope {
			tenants[i] = fmt.Sprintf("'%s'", t)
		}
		scopeFilter = fmt.Sprintf(" AND tenant_id IN (%s)", strings.Join(tenants, ","))
	}

	return fmt.Sprintf(`
		SELECT 
			incident_id, status, event_count, ai_root_cause, ai_narrative,
			min(timestamp) AS start_ts,
			max(timestamp) AS end_ts
		FROM iceberg.audit.incidents
		WHERE timestamp BETWEEN '%s' AND '%s'
		  %s
		GROUP BY incident_id, status, event_count, ai_root_cause, ai_narrative
		ORDER BY max(timestamp) DESC
		LIMIT %d OFFSET %d
	`, from.Format(time.RFC3339), to.Format(time.RFC3339), scopeFilter, limit, offset)
}

func buildIncidentDetailQuery(incidentID string, scope TenantScope) string {
	scopeFilter := ""
	if !scope.IsGlobal() {
		tenants := make([]string, len(scope))
		for i, t := range scope {
			tenants[i] = fmt.Sprintf("'%s'", t)
		}
		scopeFilter = fmt.Sprintf(" AND tenant_id IN (%s)", strings.Join(tenants, ","))
	}

	return fmt.Sprintf(`
		SELECT 
			incident_id, status, event_count, ai_root_cause, ai_narrative,
			min(timestamp) AS start_ts,
			max(timestamp) AS end_ts
		FROM iceberg.audit.incidents
		WHERE incident_id = '%s'
		  %s
		GROUP BY incident_id, status, event_count, ai_root_cause, ai_narrative
	`, incidentID, scopeFilter)
}

func buildComplianceEventsQuery(scope TenantScope, from, to time.Time, violationTypes []string, limit, offset int) string {
	scopeFilter := ""
	if !scope.IsGlobal() {
		tenants := make([]string, len(scope))
		for i, t := range scope {
			tenants[i] = fmt.Sprintf("'%s'", t)
		}
		scopeFilter = fmt.Sprintf(" AND tenant_id IN (%s)", strings.Join(tenants, ","))
	}

	violationFilter := ""
	if len(violationTypes) > 0 {
		violations := make([]string, len(violationTypes))
		for i, vt := range violationTypes {
			violations[i] = fmt.Sprintf("'%s'", vt)
		}
		violationFilter = fmt.Sprintf(" AND violation_type IN (%s)", strings.Join(violations, ","))
	}

	return fmt.Sprintf(`
		SELECT 
			id, tenant_id, timestamp, violation_type, status, artifact_type, 
			artifact_id, severity, ai_explanation
		FROM iceberg.audit.compliance_violations
		WHERE timestamp BETWEEN '%s' AND '%s'
		  %s
		  %s
		ORDER BY timestamp DESC
		LIMIT %d OFFSET %d
	`, from.Format(time.RFC3339), to.Format(time.RFC3339), scopeFilter, violationFilter, limit, offset)
}
