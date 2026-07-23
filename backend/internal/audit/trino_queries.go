package audit

import (
	"fmt"
	"strings"
	"time"
)

// TrinoQueryBuilder helps construct optimized Trino queries for audit data
type TrinoQueryBuilder struct {
	catalog string
	schema  string
}

// NewTrinoQueryBuilder creates a new query builder
func NewTrinoQueryBuilder(catalog, schema string) *TrinoQueryBuilder {
	return &TrinoQueryBuilder{
		catalog: catalog,
		schema:  schema,
	}
}

// BuildTimelineQuery builds the unified timeline query combining all audit sources
func (tqb *TrinoQueryBuilder) BuildTimelineQuery(tenantScope []string, from, to time.Time, filters map[string]interface{}, limit, offset int) string {
	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")

	tenantFilter := tqb.buildTenantFilter(tenantScope)

	// UNION all audit tables
	return fmt.Sprintf(`
WITH combined_events AS (
  -- Job Runs
  SELECT
    'job_run' AS type,
    run_id AS id,
    tenant_id,
    start_ts AS timestamp,
    status,
    'job' AS artifact_type,
    job_id AS artifact_id,
    CONCAT('Job Run: ', job_id, ' - ', status) AS title,
    submitted_by AS actor,
    CASE 
      WHEN status = 'FAILED' THEN 'HIGH'
      WHEN status = 'SUCCESS' THEN 'LOW'
      ELSE 'MEDIUM'
    END AS risk_level,
    json_extract_scalar(semantic_context, '$') AS semantic_context,
    json_extract_scalar(compliance_context, '$') AS compliance_context,
    json_extract_scalar(ai_narrative, '$') AS ai_narrative
  FROM %s.%s.scheduler_job_runs
  WHERE tenant_id IN (%s)
    AND start_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)

  UNION ALL

  -- DAG Runs
  SELECT
    'dag_run' AS type,
    dag_run_id AS id,
    tenant_id,
    start_ts AS timestamp,
    status,
    'dag' AS artifact_type,
    dag_id AS artifact_id,
    CONCAT('DAG Run: ', dag_id, ' - ', status) AS title,
    triggered_by AS actor,
    CASE 
      WHEN status = 'FAILED' THEN 'HIGH'
      WHEN status = 'SUCCESS' THEN 'LOW'
      ELSE 'MEDIUM'
    END AS risk_level,
    NULL AS semantic_context,
    NULL AS compliance_context,
    json_extract_scalar(ai_root_cause, '$') AS ai_narrative
  FROM %s.%s.scheduler_dag_runs
  WHERE tenant_id IN (%s)
    AND start_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)

  UNION ALL

  -- ChangeSets
  SELECT
    'changeset' AS type,
    changeset_id AS id,
    json_extract_scalar(semantic_impact, '$.tenantId') AS tenant_id,
    created_at AS timestamp,
    status,
    'changeset' AS artifact_type,
    changeset_id AS artifact_id,
    CONCAT('ChangeSet: ', title, ' - ', status) AS title,
    created_by AS actor,
    CASE 
      WHEN risk_score > 0.7 THEN 'HIGH'
      WHEN risk_score > 0.4 THEN 'MEDIUM'
      ELSE 'LOW'
    END AS risk_level,
    json_extract_scalar(semantic_impact, '$') AS semantic_context,
    json_extract_scalar(compliance_impact, '$') AS compliance_context,
    json_extract_scalar(ai_summary, '$') AS ai_narrative
  FROM %s.%s.governance_changesets
  WHERE created_at BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
    AND (
      json_extract_scalar(semantic_impact, '$.tenantId') IN (%s)
      OR json_extract_scalar(semantic_impact, '$.tenantId') IS NULL
    )

  UNION ALL

  -- Semantic Snapshots
  SELECT
    'semantic_snapshot' AS type,
    snapshot_id AS id,
    tenant_id,
    snapshot_ts AS timestamp,
    'COMPLETED' AS status,
    'semantic_term' AS artifact_type,
    object_id AS artifact_id,
    CONCAT('Semantic Snapshot: ', object_id) AS title,
    NULL AS actor,
    'MEDIUM' AS risk_level,
    NULL AS semantic_context,
    NULL AS compliance_context,
    json_extract_scalar(full_payload, '$') AS ai_narrative
  FROM %s.%s.semantic_snapshots
  WHERE tenant_id IN (%s)
    AND snapshot_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)

  UNION ALL

  -- Compliance Violations
  SELECT
    'compliance_violation' AS type,
    violation_id AS id,
    tenant_id,
    detected_ts AS timestamp,
    'VIOLATION' AS status,
    resource_type AS artifact_type,
    resource_id AS artifact_id,
    CONCAT(violation_type, ': ', resource_type, ' ', resource_id) AS title,
    detected_by AS actor,
    CASE 
      WHEN severity = 'CRITICAL' THEN 'CRITICAL'
      WHEN severity = 'HIGH' THEN 'HIGH'
      ELSE 'MEDIUM'
    END AS risk_level,
    NULL AS semantic_context,
    json_extract_scalar(compliance_context, '$') AS compliance_context,
    json_extract_scalar(ai_narrative, '$') AS ai_narrative
  FROM %s.%s.compliance_violations
  WHERE tenant_id IN (%s)
    AND detected_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
)
SELECT *
FROM combined_events
ORDER BY timestamp DESC
LIMIT %d OFFSET %d
`, tqb.catalog, tqb.schema, tenantFilter, fromStr, toStr, tqb.catalog, tqb.schema, tenantFilter, fromStr, toStr, tqb.catalog, tqb.schema, fromStr, toStr, tenantFilter, tqb.catalog, tqb.schema, tenantFilter, fromStr, toStr, tqb.catalog, tqb.schema, tenantFilter, fromStr, toStr, limit, offset)
}

// BuildEntityAuditQuery builds query for all events related to a specific entity
func (tqb *TrinoQueryBuilder) BuildEntityAuditQuery(entityType, entityID string, tenantScope []string, from, to time.Time, limit, offset int) string {
	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")
	tenantFilter := tqb.buildTenantFilter(tenantScope)

	// Build entity-specific queries based on type
	var query string

	switch entityType {
	case "semantic_term":
		query = fmt.Sprintf(`
SELECT 
  'timeline' AS category,
  id, type, tenant_id, timestamp, status, title, ai_narrative
FROM (
  -- ChangeSets affecting this semantic term
  SELECT
    changeset_id AS id,
    'changeset' AS type,
    json_extract_scalar(semantic_impact, '$.tenantId') AS tenant_id,
    created_at AS timestamp,
    status,
    CONCAT('ChangeSet: ', title) AS title,
    json_extract_scalar(ai_summary, '$') AS ai_narrative
  FROM %[1]s.%[2]s.governance_changesets
  WHERE json_extract_scalar(semantic_impact, '$.semanticTermId') = '%s'
    AND created_at BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)

  UNION ALL

  -- Jobs that use this semantic term
  SELECT
    run_id AS id,
    'job_run' AS type,
    tenant_id,
    start_ts AS timestamp,
    status,
    CONCAT('Job Run: ', job_id) AS title,
    json_extract_scalar(ai_narrative, '$') AS ai_narrative
  FROM %s.%s.scheduler_job_runs
  WHERE json_extract_scalar(semantic_context, '$.semanticTermId') = '%s'
    AND start_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)

  UNION ALL

  -- Semantic snapshots
  SELECT
    snapshot_id AS id,
    'semantic_snapshot' AS type,
    tenant_id,
    snapshot_ts AS timestamp,
    'COMPLETED' AS status,
    CONCAT('Snapshot: ', object_id) AS title,
    json_extract_scalar(full_payload, '$') AS ai_narrative
  FROM %s.%s.semantic_snapshots
  WHERE object_id = '%s'
    AND snapshot_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
) t
WHERE tenant_id IN (%s)
ORDER BY timestamp DESC
LIMIT %d OFFSET %d
`, tqb.catalog, tqb.schema, entityID, fromStr, toStr, tqb.catalog, tqb.schema, entityID, fromStr, toStr, tqb.catalog, tqb.schema, entityID, fromStr, toStr, tenantFilter, limit, offset)

	case "job":
		query = fmt.Sprintf(`
SELECT 
  'timeline' AS category,
  id, type, tenant_id, timestamp, status, title, ai_narrative
FROM (
  -- Job runs
  SELECT
    run_id AS id,
    'job_run' AS type,
    tenant_id,
    start_ts AS timestamp,
    status,
    CONCAT('Job Run: ', job_id, ' - ', status) AS title,
    json_extract_scalar(ai_narrative, '$') AS ai_narrative
  FROM %[1]s.%[2]s.scheduler_job_runs
  WHERE job_id = '%s'
    AND start_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)

  UNION ALL

  -- Compliance violations
  SELECT
    violation_id AS id,
    'compliance_violation' AS type,
    tenant_id,
    detected_ts AS timestamp,
    'VIOLATION' AS status,
    CONCAT(violation_type, ': Job blocked') AS title,
    json_extract_scalar(ai_narrative, '$') AS ai_narrative
  FROM %s.%s.compliance_violations
  WHERE resource_type = 'job' AND resource_id = '%s'
    AND detected_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
) t
WHERE tenant_id IN (%s)
ORDER BY timestamp DESC
LIMIT %d OFFSET %d
`, tqb.catalog, tqb.schema, entityID, fromStr, toStr, tqb.catalog, tqb.schema, entityID, fromStr, toStr, tenantFilter, limit, offset)

	default:
		// Generic entity audit query
		query = fmt.Sprintf(`
SELECT 
  'timeline' AS category,
  id, type, tenant_id, timestamp, status, title, ai_narrative
FROM %[1]s.%[2]s.events
WHERE entity_type = '%s' 
  AND entity_id = '%s'
  AND tenant_id IN (%s)
  AND timestamp BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
ORDER BY timestamp DESC
LIMIT %d OFFSET %d
`, tqb.catalog, tqb.schema, entityType, entityID, tenantFilter, fromStr, toStr, limit, offset)
	}

	return query
}

// BuildIncidentsQuery builds query for incident clusters
func (tqb *TrinoQueryBuilder) BuildIncidentsQuery(tenantScope []string, from, to time.Time, limit, offset int) string {
	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")
	tenantFilter := tqb.buildTenantFilter(tenantScope)

	return fmt.Sprintf(`
SELECT 
  incident_id,
  status,
  event_count,
  ai_root_cause,
  ai_narrative,
  min(timestamp) AS start_ts,
  max(timestamp) AS end_ts,
  count(DISTINCT tenant_id) AS affected_tenant_count
FROM %[1]s.%[2]s.incidents
WHERE tenant_id IN (%s)
  AND timestamp BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
GROUP BY incident_id, status, event_count, ai_root_cause, ai_narrative
ORDER BY max(timestamp) DESC
LIMIT %d OFFSET %d
`, tqb.catalog, tqb.schema, tenantFilter, fromStr, toStr, limit, offset)
}

// BuildComplianceEventsQuery builds query for compliance violations
func (tqb *TrinoQueryBuilder) BuildComplianceEventsQuery(tenantScope []string, from, to time.Time, violationTypes []string, limit, offset int) string {
	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")
	tenantFilter := tqb.buildTenantFilter(tenantScope)

	violationFilter := ""
	if len(violationTypes) > 0 {
		violations := tqb.buildStringList(violationTypes)
		violationFilter = fmt.Sprintf(" AND violation_type IN (%s)", violations)
	}

	return fmt.Sprintf(`
SELECT 
  violation_id AS id,
  tenant_id,
  detected_ts AS timestamp,
  violation_type,
  CASE WHEN resolved_at IS NOT NULL THEN 'RESOLVED' ELSE 'UNRESOLVED' END AS status,
  resource_type AS artifact_type,
  resource_id AS artifact_id,
  severity,
  json_extract_scalar(ai_narrative, '$') AS ai_explanation
FROM %[1]s.%[2]s.compliance_violations
WHERE tenant_id IN (%s)
  AND detected_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
  %s
ORDER BY detected_ts DESC, severity DESC
LIMIT %d OFFSET %d
`, tqb.catalog, tqb.schema, tenantFilter, fromStr, toStr, violationFilter, limit, offset)
}

// BuildSLOSummaryQuery builds query for SLO metrics per tenant
func (tqb *TrinoQueryBuilder) BuildSLOSummaryQuery(tenantScope []string, from, to time.Time) string {
	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")
	tenantFilter := tqb.buildTenantFilter(tenantScope)

	return fmt.Sprintf(`
SELECT
  tenant_id,
  CAST(date(start_ts) AS varchar) AS run_date,
  count(*) AS total_runs,
  count(*) FILTER (WHERE status = 'SUCCESS') AS successful_runs,
  count(*) FILTER (WHERE status = 'FAILED') AS failed_runs,
  count(*) FILTER (WHERE status = 'COMPLIANCE_BLOCK') AS compliance_blocked_runs,
  cast(count(*) FILTER (WHERE status = 'SUCCESS') as double) / count(*) AS success_rate,
  avg(cast(end_ts - start_ts as double)) AS avg_duration_ms,
  max_by(duration_ms, end_ts) AS latest_duration_ms
FROM %[1]s.%[2]s.scheduler_job_runs
WHERE tenant_id IN (%s)
  AND start_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
GROUP BY tenant_id, date(start_ts)
ORDER BY run_date DESC, tenant_id
`, tqb.catalog, tqb.schema, tenantFilter, fromStr, toStr)
}

// BuildComplianceSummaryQuery builds query for compliance trends
func (tqb *TrinoQueryBuilder) BuildComplianceSummaryQuery(tenantScope []string, from, to time.Time) string {
	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")
	tenantFilter := tqb.buildTenantFilter(tenantScope)

	return fmt.Sprintf(`
SELECT
  tenant_id,
  CAST(date(detected_ts) AS varchar) AS violation_date,
  violation_type,
  severity,
  count(*) AS violation_count,
  count(*) FILTER (WHERE resolved_at IS NOT NULL) AS resolved_count
FROM %[1]s.%[2]s.compliance_violations
WHERE tenant_id IN (%s)
  AND detected_ts BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
GROUP BY tenant_id, date(detected_ts), violation_type, severity
ORDER BY violation_date DESC, severity DESC
`, tqb.catalog, tqb.schema, tenantFilter, fromStr, toStr)
}

// BuildGovernanceActivityQuery builds query for governance changes
func (tqb *TrinoQueryBuilder) BuildGovernanceActivityQuery(tenantScope []string, from, to time.Time) string {
	fromStr := from.Format("2006-01-02 15:04:05")
	toStr := to.Format("2006-01-02 15:04:05")
	tenantFilter := tqb.buildTenantFilter(tenantScope)

	return fmt.Sprintf(`
SELECT
  json_extract_scalar(semantic_impact, '$.tenantId') AS tenant_id,
  changeset_type,
  status,
  created_by AS actor,
  created_at,
  CASE 
    WHEN risk_score > 0.7 THEN 'HIGH'
    WHEN risk_score > 0.4 THEN 'MEDIUM'
    ELSE 'LOW'
  END AS risk_level,
  count(*) AS changeset_count
FROM %[1]s.%[2]s.governance_changesets
WHERE created_at BETWEEN CAST('%s' AS timestamp) AND CAST('%s' AS timestamp)
  AND json_extract_scalar(semantic_impact, '$.tenantId') IN (%s)
GROUP BY json_extract_scalar(semantic_impact, '$.tenantId'), changeset_type, status, created_by, created_at, risk_score
ORDER BY created_at DESC
`, tqb.catalog, tqb.schema, fromStr, toStr, tenantFilter)
}

// Helper functions
func (tqb *TrinoQueryBuilder) buildTenantFilter(tenantScope []string) string {
	return tqb.buildStringList(tenantScope)
}

func (tqb *TrinoQueryBuilder) buildStringList(items []string) string {
	if len(items) == 0 {
		return "''"
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = fmt.Sprintf("'%s'", item)
	}
	return strings.Join(parts, ", ")
}
