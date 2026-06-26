package audit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// AuditAPIHandler provides HTTP endpoints for querying audit data
type AuditAPIHandler struct {
	querier *TrinoAuditQuerier
}

// NewAuditAPIHandler creates a new audit API handler
func NewAuditAPIHandler(querier *TrinoAuditQuerier) *AuditAPIHandler {
	return &AuditAPIHandler{
		querier: querier,
	}
}

// RegisterRoutes registers audit API routes with Gin
func (h *AuditAPIHandler) RegisterRoutes(r *gin.RouterGroup) {
	audit := r.Group("/audit")
	{
		// Job run endpoints
		audit.GET("/job-runs", h.GetJobRuns)
		audit.GET("/job-runs/:run_id", h.GetJobRun)

		// DAG run endpoints
		audit.GET("/dag-runs", h.GetDAGRuns)

		// Changeset endpoints
		audit.GET("/changesets", h.GetChangeSets)
		audit.GET("/changesets/:changeset_id", h.GetChangeSet)

		// Compliance endpoints
		audit.GET("/violations", h.GetComplianceViolations)
		audit.GET("/violations/:violation_id", h.GetComplianceViolation)

		// Semantic lineage endpoints
		audit.GET("/semantic/:semantic_term_id/lineage", h.GetSemanticLineage)
		audit.GET("/semantic/:semantic_term_id/versions", h.GetSemanticVersions)

		// AI narrative endpoints
		audit.GET("/ai-narratives", h.GetAINarratives)

		// Dashboard endpoints (materialized views)
		audit.GET("/dashboard/slo", h.GetSLODashboard)
		audit.GET("/dashboard/compliance", h.GetComplianceDashboard)
		audit.GET("/dashboard/governance", h.GetGovernanceDashboard)
	}
}

// GetJobRuns queries scheduler job runs with multi-tenant filtering
func (h *AuditAPIHandler) GetJobRuns(c *gin.Context) {
	// Extract tenant_id from headers (enforced by middleware)
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	params := JobRunQueryParams{
		TenantID: tenantID,
		JobID:    c.Query("job_id"),
		Status:   c.Query("status"),
	}

	if semanticTermID := c.Query("semantic_term_id"); semanticTermID != "" {
		params.SemanticTermID = semanticTermID
	}

	if startDate := c.Query("start_date"); startDate != "" {
		if t, err := time.Parse(time.RFC3339, startDate); err == nil {
			params.StartDate = t
		}
	}

	if endDate := c.Query("end_date"); endDate != "" {
		if t, err := time.Parse(time.RFC3339, endDate); err == nil {
			params.EndDate = t
		}
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			params.Limit = l
		}
	}

	results, err := h.querier.QueryJobRuns(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}

// GetJobRun retrieves a single job run by ID
func (h *AuditAPIHandler) GetJobRun(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	runID := c.Param("run_id")

	// Use a custom query to filter by run_id
	query := `
		SELECT 
			run_id, job_id, dag_id, tenant_id, start_ts, end_ts, status,
			error_message, semantic_context, compliance_context, 
			slo_context, ai_narrative, metadata,
			_ingest_ts, _source_service, _schema_version
		FROM scheduler_job_runs
		WHERE tenant_id = $1 AND run_id = $2
		LIMIT 1
	`

	var r SchedulerJobRun
	err := h.querier.db.QueryRowContext(c.Request.Context(), query, tenantID, runID).Scan(
		&r.RunID, &r.JobID, &r.DagID, &r.TenantID, &r.StartTS, &r.EndTS, &r.Status,
		&r.ErrorMessage, &r.SemanticContext, &r.ComplianceContext,
		&r.SLOContext, &r.AINarrative, &r.Metadata,
		&r.IngestTS, &r.SourceService, &r.SchemaVersion,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job run not found"})
		return
	}

	c.JSON(http.StatusOK, r)
}

// GetDAGRuns queries DAG runs
func (h *AuditAPIHandler) GetDAGRuns(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	query := `
		SELECT 
			dag_run_id, dag_id, tenant_id, start_ts, end_ts, status,
			critical_path, ai_root_cause, metadata,
			_ingest_ts, _source_service, _schema_version
		FROM scheduler_dag_runs
		WHERE tenant_id = $1
		ORDER BY start_ts DESC
		LIMIT 100
	`

	rows, err := h.querier.db.QueryContext(c.Request.Context(), query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []SchedulerDAGRun
	for rows.Next() {
		var r SchedulerDAGRun
		err := rows.Scan(
			&r.DagRunID, &r.DagID, &r.TenantID, &r.StartTS, &r.EndTS, &r.Status,
			&r.CriticalPath, &r.AIRootCause, &r.Metadata,
			&r.IngestTS, &r.SourceService, &r.SchemaVersion,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		results = append(results, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}

// GetChangeSets queries governance changesets
func (h *AuditAPIHandler) GetChangeSets(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	// Note: changeset queries support cross-tenant for internal users
	// but still require header for context

	params := ChangeSetQueryParams{
		TenantID: tenantID,
		Type:     c.Query("type"),
		Actor:    c.Query("actor"),
		Status:   c.Query("status"),
	}

	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil {
			params.Limit = l
		}
	}

	results, err := h.querier.QueryChangeSets(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}

// GetChangeSet retrieves a single changeset by ID
func (h *AuditAPIHandler) GetChangeSet(c *gin.Context) {
	changesetID := c.Param("changeset_id")
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID

	params := ChangeSetQueryParams{
		TenantID: tenantID,
		Limit:    1,
	}

	results, err := h.querier.QueryChangeSets(c.Request.Context(), params)
	if err != nil || len(results) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "changeset not found"})
		return
	}

	// Filter by changeset_id client-side (could optimize with direct query)
	for _, cs := range results {
		if cs.ChangesetID == changesetID {
			c.JSON(http.StatusOK, cs)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "changeset not found"})
}

// GetComplianceViolations queries compliance violations
func (h *AuditAPIHandler) GetComplianceViolations(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	params := ComplianceViolationQueryParams{
		TenantID:      tenantID,
		Severity:      c.Query("severity"),
		ViolationType: c.Query("violation_type"),
	}

	if piiExposed := c.Query("pii_exposed"); piiExposed != "" {
		val := piiExposed == "true"
		params.PIIExposed = &val
	}

	if remediated := c.Query("remediated"); remediated != "" {
		val := remediated == "true"
		params.Remediated = &val
	}

	results, err := h.querier.QueryComplianceViolations(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}

// GetComplianceViolation retrieves a single violation by ID
func (h *AuditAPIHandler) GetComplianceViolation(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	violationID := c.Param("violation_id")
	params := ComplianceViolationQueryParams{
		TenantID: tenantID,
		Limit:    100,
	}

	results, err := h.querier.QueryComplianceViolations(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	for _, v := range results {
		if v.ViolationID == violationID {
			c.JSON(http.StatusOK, v)
			return
		}
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "violation not found"})
}

// GetSemanticLineage performs time-travel query on semantic term
func (h *AuditAPIHandler) GetSemanticLineage(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	semanticTermID := c.Param("semantic_term_id")
	versionStr := c.Query("version")

	if versionStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "version parameter required"})
		return
	}

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	result, err := h.querier.QuerySemanticLineage(c.Request.Context(), tenantID, semanticTermID, version)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "semantic term version not found"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetSemanticVersions gets all versions of a semantic term
func (h *AuditAPIHandler) GetSemanticVersions(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	semanticTermID := c.Param("semantic_term_id")

	query := `
		SELECT version, timestamp, definition
		FROM semantic_snapshots
		WHERE semantic_term_id = $1
		  AND (tenant_id = $2 OR tenant_id IS NULL)
		ORDER BY version DESC
	`

	rows, err := h.querier.db.QueryContext(c.Request.Context(), query, semanticTermID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type Version struct {
		Version    int       `json:"version"`
		Timestamp  time.Time `json:"timestamp"`
		Definition string    `json:"definition"`
	}

	var versions []Version
	for rows.Next() {
		var v Version
		if err := rows.Scan(&v.Version, &v.Timestamp, &v.Definition); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		versions = append(versions, v)
	}

	c.JSON(http.StatusOK, gin.H{
		"semantic_term_id": semanticTermID,
		"versions":         versions,
		"count":            len(versions),
	})
}

// GetAINarratives retrieves AI-generated audit narratives
func (h *AuditAPIHandler) GetAINarratives(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	query := `
		SELECT 
			suggestion_id, audit_record_id, record_type, tenant_id, timestamp,
			narrative, root_cause, blast_radius, recommended_fix, 
			suggested_actions, confidence
		FROM ai_suggestions
		WHERE tenant_id = $1
		ORDER BY timestamp DESC
		LIMIT 100
	`

	rows, err := h.querier.db.QueryContext(c.Request.Context(), query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var results []AIAuditSuggestion
	for rows.Next() {
		var r AIAuditSuggestion
		err := rows.Scan(
			&r.SuggestionID, &r.AuditRecordID, &r.RecordType, &r.TenantID, &r.Timestamp,
			&r.Narrative, &r.RootCause, &r.BlastRadius, &r.RecommendedFix,
			&r.SuggestedActions, &r.Confidence,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		results = append(results, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  results,
		"count": len(results),
	})
}

// GetSLODashboard returns SLO dashboard data from materialized view
func (h *AuditAPIHandler) GetSLODashboard(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	query := `
		SELECT 
			run_date, total_runs, successful_runs, failed_runs, blocked_runs,
			avg_duration_seconds, max_duration_seconds, p95_duration_seconds
		FROM mv_tenant_scheduler_slo
		WHERE tenant_id = $1
		ORDER BY run_date DESC
		LIMIT 30
	`

	rows, err := h.querier.db.QueryContext(c.Request.Context(), query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type SLODay struct {
		RunDate            time.Time `json:"run_date"`
		TotalRuns          int       `json:"total_runs"`
		SuccessfulRuns     int       `json:"successful_runs"`
		FailedRuns         int       `json:"failed_runs"`
		BlockedRuns        int       `json:"blocked_runs"`
		AvgDurationSeconds float64   `json:"avg_duration_seconds"`
		MaxDurationSeconds float64   `json:"max_duration_seconds"`
		P95DurationSeconds float64   `json:"p95_duration_seconds"`
	}

	var results []SLODay
	for rows.Next() {
		var r SLODay
		if err := rows.Scan(
			&r.RunDate, &r.TotalRuns, &r.SuccessfulRuns, &r.FailedRuns, &r.BlockedRuns,
			&r.AvgDurationSeconds, &r.MaxDurationSeconds, &r.P95DurationSeconds,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		results = append(results, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"data":      results,
	})
}

// GetComplianceDashboard returns compliance dashboard data
func (h *AuditAPIHandler) GetComplianceDashboard(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	query := `
		SELECT 
			violation_date, violation_count, pii_exposure_count,
			critical_violations, high_violations, avg_remediation_hours, open_violations
		FROM mv_tenant_compliance_violations
		WHERE tenant_id = $1
		ORDER BY violation_date DESC
		LIMIT 30
	`

	rows, err := h.querier.db.QueryContext(c.Request.Context(), query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type ComplianceDay struct {
		ViolationDate       time.Time `json:"violation_date"`
		ViolationCount      int       `json:"violation_count"`
		PIIExposureCount    int       `json:"pii_exposure_count"`
		CriticalViolations  int       `json:"critical_violations"`
		HighViolations      int       `json:"high_violations"`
		AvgRemediationHours float64   `json:"avg_remediation_hours"`
		OpenViolations      int       `json:"open_violations"`
	}

	var results []ComplianceDay
	for rows.Next() {
		var r ComplianceDay
		if err := rows.Scan(
			&r.ViolationDate, &r.ViolationCount, &r.PIIExposureCount,
			&r.CriticalViolations, &r.HighViolations, &r.AvgRemediationHours, &r.OpenViolations,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		results = append(results, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"data":      results,
	})
}

// GetGovernanceDashboard returns governance activity dashboard data
func (h *AuditAPIHandler) GetGovernanceDashboard(c *gin.Context) {
	claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Tenant-ID header required"})
		return
	}

	query := `
		SELECT 
			activity_date, changeset_count, unique_actors,
			pending_count, approved_count, rejected_count, applied_count, avg_risk_score
		FROM mv_tenant_governance_activity
		WHERE tenant_id = $1
		ORDER BY activity_date DESC
		LIMIT 30
	`

	rows, err := h.querier.db.QueryContext(c.Request.Context(), query, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	type GovernanceDay struct {
		ActivityDate   time.Time `json:"activity_date"`
		ChangesetCount int       `json:"changeset_count"`
		UniqueActors   int       `json:"unique_actors"`
		PendingCount   int       `json:"pending_count"`
		ApprovedCount  int       `json:"approved_count"`
		RejectedCount  int       `json:"rejected_count"`
		AppliedCount   int       `json:"applied_count"`
		AvgRiskScore   float64   `json:"avg_risk_score"`
	}

	var results []GovernanceDay
	for rows.Next() {
		var r GovernanceDay
		if err := rows.Scan(
			&r.ActivityDate, &r.ChangesetCount, &r.UniqueActors,
			&r.PendingCount, &r.ApprovedCount, &r.RejectedCount, &r.AppliedCount, &r.AvgRiskScore,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		results = append(results, r)
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"data":      results,
	})
}

// TenantScopeMiddleware enforces tenant isolation on all audit queries
func TenantScopeMiddlewareGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract tenant from header or query
		claims := jwtmiddleware.GetGinClaimsFromContext(c)
	if claims == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error":"unauthorized"})
		return
	}
	tenantID := claims.TenantID
		if tenantID == "" {
			tenantID = c.Query("tenant_id")
		}

		// For audit endpoints, tenant is mandatory
		if tenantID == "" && c.Request.URL.Path[:11] == "/api/audit/" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "X-Tenant-ID header required for audit queries",
			})
			return
		}

		// Set in context for downstream handlers
		c.Set("tenant_id", tenantID)
		c.Next()
	}
}
