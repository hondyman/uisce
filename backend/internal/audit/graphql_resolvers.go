package audit

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// AuditGraphResolver handles GraphQL queries for the audit semantic graph
type AuditGraphResolver struct {
	trinoQuerier TrinoQuerier
	catalogRead  CatalogReader
	logger       *zap.Logger
}

// NewAuditGraphResolver creates a new resolver for audit graph queries
func NewAuditGraphResolver(
	trinoQuerier TrinoQuerier,
	catalogRead CatalogReader,
	logger *zap.Logger,
) *AuditGraphResolver {
	return &AuditGraphResolver{
		trinoQuerier: trinoQuerier,
		catalogRead:  catalogRead,
		logger:       logger,
	}
}

// AuditEventsArgs represents arguments for auditEvents query
type AuditEventsArgs struct {
	TenantIds  []string
	Types      []string
	Statuses   []string
	Severities []string
	From       time.Time
	To         time.Time
	Search     string
	Limit      int
	Offset     int
}

// QueryAuditEvents queries audit events from Trino with filtering
func (r *AuditGraphResolver) QueryAuditEvents(ctx context.Context, tenantIds []string, types []string, statuses []string, severities []string, from, to time.Time, limit, offset int) ([]*AuditEventResponse, error) {
	// Validate tenant scope from auth context
	allowedTenants := r.getTenantScope(ctx)
	scope := intersectTenants(allowedTenants, tenantIds)

	if len(scope) == 0 {
		r.logger.Warn("query denied: no allowed tenants", zap.Strings("requested", tenantIds))
		return nil, fmt.Errorf("access denied: no valid tenant scope")
	}

	// Build Trino query
	query := r.buildAuditEventsQuery(scope, types, statuses, severities, from, to, limit, offset)

	// Execute via Trino
	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to query audit events", zap.Error(err))
		return nil, err
	}

	// Parse results
	events := parseAuditEventResults(results)
	r.logger.Info("queried audit events", zap.Int("count", len(events)), zap.Strings("tenants", scope))

	return events, nil
}

// QueryEntityAudit gets the complete audit timeline for an entity
func (r *AuditGraphResolver) QueryEntityAudit(ctx context.Context, entityType, entityId string, tenantIds []string, from, to time.Time) (*EntityAuditResponse, error) {
	// Validate tenant scope
	allowedTenants := r.getTenantScope(ctx)
	scope := intersectTenants(allowedTenants, tenantIds)

	if len(scope) == 0 {
		return nil, fmt.Errorf("access denied: no valid tenant scope")
	}

	// Query events affecting this entity
	query := fmt.Sprintf(`
		SELECT
		  e.event_id,
		  e.node_type,
		  e.event_time,
		  e.event_status,
		  e.severity,
		  e.properties,
		  e.tenant_id
		FROM audit.entity_timeline e
		WHERE e.entity_id = '%s'
		  AND e.entity_type = '%s'
		  AND e.event_time >= TIMESTAMP '%s'
		  AND e.event_time <= TIMESTAMP '%s'
		  AND e.tenant_id IN (%s)
		ORDER BY e.event_time DESC
	`, entityId, entityType, from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05"), formatTenantList(scope))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to query entity audit", zap.Error(err), zap.String("entity", entityId))
		return nil, err
	}

	// Build response with statistics
	response := &EntityAuditResponse{
		EntityType: entityType,
		EntityId:   entityId,
		Events:     parseAuditEventResults(results),
	}

	// Calculate summary
	response.Summary = calculateEntitySummary(response.Events)

	// Build impact analysis
	response.ImpactAnalysis = r.analyzeEntityImpact(ctx, entityType, entityId, scope)

	return response, nil
}

// QueryIncidents gets incidents with root cause and impact information
func (r *AuditGraphResolver) QueryIncidents(ctx context.Context, tenantIds []string, statuses, severities []string, from, to time.Time, limit, offset int) ([]*IncidentResponse, error) {
	// Validate tenant scope
	allowedTenants := r.getTenantScope(ctx)
	scope := intersectTenants(allowedTenants, tenantIds)

	if len(scope) == 0 {
		return nil, fmt.Errorf("access denied: no valid tenant scope")
	}

	// Query incidents
	query := r.buildIncidentsQuery(scope, statuses, severities, from, to, limit, offset)

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to query incidents", zap.Error(err))
		return nil, err
	}

	incidents := parseIncidentResults(results)

	// Enrich with root cause and AI analysis
	for i := range incidents {
		incidents[i].RootCauseAnalysis = r.generateRootCauseAnalysis(ctx, incidents[i].ID)
	}

	return incidents, nil
}

// ExplainAudit generates AI-powered explanation for audit events
func (r *AuditGraphResolver) ExplainAudit(ctx context.Context, eventId, eventType string, tenantIds []string) (*AIExplanationResponse, error) {
	// Validate tenant scope
	allowedTenants := r.getTenantScope(ctx)
	scope := intersectTenants(allowedTenants, tenantIds)

	if len(scope) == 0 {
		return nil, fmt.Errorf("access denied: no valid tenant scope")
	}

	// Fetch the event
	event, err := r.getEventDetails(ctx, eventId, scope)
	if err != nil {
		return nil, err
	}

	// Fetch related events for context
	relatedEvents, err := r.getRelatedEvents(ctx, eventId, scope)
	if err != nil {
		return nil, err
	}

	// Build AI prompt with graph context
	prompt := r.buildGraphAwarePrompt(event, relatedEvents, eventType)

	// Call LLM (Gemini/Claude)
	explanation, err := r.callLLMForExplanation(ctx, prompt)
	if err != nil {
		r.logger.Error("failed to generate AI explanation", zap.Error(err))
		return nil, err
	}

	return &AIExplanationResponse{
		WhatHappened:       explanation.Summary,
		RootCause:          explanation.RootCause,
		Severity:           explanation.Severity,
		BlastRadius:        explanation.BlastRadius,
		AffectedEntities:   explanation.AffectedEntities,
		RecommendedActions: explanation.RecommendedActions,
		ProposedChangeSet:  explanation.ProposedChangeSet,
		Confidence:         explanation.Confidence,
		RelatedEvents:      relatedEvents,
	}, nil
}

// QueryChangeSetImpact analyzes the impact of a ChangeSet on downstream entities
func (r *AuditGraphResolver) QueryChangeSetImpact(ctx context.Context, changeSetId string, tenantIds []string) (*ChangeSetImpactResponse, error) {
	allowedTenants := r.getTenantScope(ctx)
	scope := intersectTenants(allowedTenants, tenantIds)

	if len(scope) == 0 {
		return nil, fmt.Errorf("access denied: no valid tenant scope")
	}

	// Query changeset impact view
	query := fmt.Sprintf(`
		SELECT
		  cs.changeset_id,
		  cs.changeset_status,
		  cs.risk_score,
		  cs.impacted_semantic_term_id,
		  cs.impacted_term_type
		FROM audit.changeset_impact cs
		WHERE cs.changeset_id = '%s'
		  AND cs.tenant_id IN (%s)
	`, changeSetId, formatTenantList(scope))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	// Build response
	response := parseChangeSetImpactResults(results)

	// Analyze downstream impacts
	response.DownstreamImpacts = r.analyzeDownstreamImpacts(ctx, response.AffectedTerms, scope)

	// Find related incidents
	response.PotentialIncidents = r.findRelatedIncidents(ctx, response.AffectedTerms, scope)

	return response, nil
}

// QueryComplianceStatus gets compliance status and violation summary
func (r *AuditGraphResolver) QueryComplianceStatus(ctx context.Context, tenantIds []string, from, to time.Time) (*ComplianceStatusResponse, error) {
	allowedTenants := r.getTenantScope(ctx)
	scope := intersectTenants(allowedTenants, tenantIds)

	if len(scope) == 0 {
		return nil, fmt.Errorf("access denied: no valid tenant scope")
	}

	// Query compliance summary
	query := fmt.Sprintf(`
		SELECT
		  COUNT(*) as total_checks,
		  SUM(CASE WHEN severity = 'CRITICAL' THEN 1 ELSE 0 END) as critical,
		  SUM(CASE WHEN severity = 'HIGH' THEN 1 ELSE 0 END) as high,
		  SUM(CASE WHEN status = 'VIOLATION' THEN 1 ELSE 0 END) as violations,
		  SUM(CASE WHEN status IN ('REMEDIATED', 'PASSED') THEN 1 ELSE 0 END) as passing,
		  MAX(event_time) as last_violation
		FROM audit.semantic_events
		WHERE node_type = 'compliance_event'
		  AND event_time >= TIMESTAMP '%s'
		  AND event_time <= TIMESTAMP '%s'
		  AND tenant_id IN (%s)
	`, from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05"), formatTenantList(scope))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	return parseComplianceStatusResults(results), nil
}

// QueryCriticalEventsRealtime returns critical events from the last N hours
func (r *AuditGraphResolver) QueryCriticalEventsRealtime(ctx context.Context, tenantIds []string, hoursBack int) ([]*AuditEventResponse, error) {
	allowedTenants := r.getTenantScope(ctx)
	scope := intersectTenants(allowedTenants, tenantIds)

	if len(scope) == 0 {
		return nil, fmt.Errorf("access denied: no valid tenant scope")
	}

	// Query critical events view
	query := fmt.Sprintf(`
		SELECT
		  event_id,
		  node_type,
		  event_time,
		  event_status,
		  severity,
		  error_message,
		  properties,
		  tenant_id,
		  seconds_ago
		FROM audit.critical_events_realtime
		WHERE seconds_ago <= %d
		  AND tenant_id IN (%s)
		ORDER BY event_time DESC
	`, hoursBack*3600, formatTenantList(scope))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	return parseAuditEventResults(results), nil
}

// QueryAuditEventStats gets statistics for dashboard
func (r *AuditGraphResolver) QueryAuditEventStats(ctx context.Context, tenantIds []string, from, to time.Time) (*AuditEventStatsResponse, error) {
	allowedTenants := r.getTenantScope(ctx)
	scope := intersectTenants(allowedTenants, tenantIds)

	if len(scope) == 0 {
		return nil, fmt.Errorf("access denied: no valid tenant scope")
	}

	// Build comprehensive stats query
	query := fmt.Sprintf(`
		SELECT
		  COUNT(*) as total_events,
		  node_type,
		  event_status,
		  severity
		FROM audit.semantic_events
		WHERE event_time >= TIMESTAMP '%s'
		  AND event_time <= TIMESTAMP '%s'
		  AND tenant_id IN (%s)
		GROUP BY node_type, event_status, severity
	`, from.Format("2006-01-02 15:04:05"), to.Format("2006-01-02 15:04:05"), formatTenantList(scope))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	return parseAuditEventStatsResults(results), nil
}

// ============================================================================
// Helper Methods
// ============================================================================

// getTenantScope extracts allowed tenants from auth context
func (r *AuditGraphResolver) getTenantScope(ctx context.Context) []string {
	// This would typically come from JWT or auth middleware
	// For now, return empty - caller must provide tenant IDs
	return []string{}
}

// buildAuditEventsQuery constructs the Trino SQL query for audit events
func (r *AuditGraphResolver) buildAuditEventsQuery(tenantIds, types, statuses, severities []string, from, to time.Time, limit, offset int) string {
	// Build WHERE clause dynamically
	whereClause := fmt.Sprintf(
		`WHERE tenant_id IN (%s) AND event_time >= TIMESTAMP '%s' AND event_time <= TIMESTAMP '%s'`,
		formatTenantList(tenantIds),
		from.Format("2006-01-02 15:04:05"),
		to.Format("2006-01-02 15:04:05"),
	)

	if len(types) > 0 {
		whereClause += fmt.Sprintf(` AND node_type IN (%s)`, formatStringList(types))
	}
	if len(statuses) > 0 {
		whereClause += fmt.Sprintf(` AND event_status IN (%s)`, formatStringList(statuses))
	}
	if len(severities) > 0 {
		whereClause += fmt.Sprintf(` AND severity IN (%s)`, formatStringList(severities))
	}

	query := fmt.Sprintf(`
		SELECT * FROM audit.semantic_events
		%s
		ORDER BY event_time DESC
		LIMIT %d OFFSET %d
	`, whereClause, limit, offset)

	return query
}

// buildIncidentsQuery builds the query for incidents
func (r *AuditGraphResolver) buildIncidentsQuery(tenantIds, statuses, severities []string, from, to time.Time, limit, offset int) string {
	whereClause := fmt.Sprintf(
		`WHERE tenant_id IN (%s) AND incident_time >= TIMESTAMP '%s' AND incident_time <= TIMESTAMP '%s'`,
		formatTenantList(tenantIds),
		from.Format("2006-01-02 15:04:05"),
		to.Format("2006-01-02 15:04:05"),
	)

	if len(statuses) > 0 {
		whereClause += fmt.Sprintf(` AND incident_status IN (%s)`, formatStringList(statuses))
	}
	if len(severities) > 0 {
		whereClause += fmt.Sprintf(` AND incident_severity IN (%s)`, formatStringList(severities))
	}

	query := fmt.Sprintf(`
		SELECT * FROM audit.incident_graph
		%s
		ORDER BY incident_time DESC
		LIMIT %d OFFSET %d
	`, whereClause, limit, offset)

	return query
}

// Helper functions for formatting SQL

func formatTenantList(tenants []string) string {
	var result string
	for i, t := range tenants {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("'%s'", t)
	}
	return result
}

func formatStringList(items []string) string {
	var result string
	for i, item := range items {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("'%s'", item)
	}
	return result
}

// intersectTenants finds common tenants between allowed and requested
func intersectTenants(allowed, requested []string) []string {
	if len(requested) == 0 {
		return allowed
	}
	allowedMap := make(map[string]bool)
	for _, t := range allowed {
		allowedMap[t] = true
	}
	var result []string
	for _, t := range requested {
		if allowedMap[t] {
			result = append(result, t)
		}
	}
	return result
}

// Response types for GraphQL queries

type AuditEventResponse struct {
	ID            string
	Type          string
	Timestamp     time.Time
	Status        string
	Severity      string
	ErrorMessage  string
	Properties    map[string]any
	TenantID      string
	RelatedEntity *AuditEntityResponse
	AINarratives  []*AISuggestionResponse
	CatalogNodeID string
	RelatedEvents []*AuditEventRelationshipResponse
}

type AuditEntityResponse struct {
	Type string
	ID   string
	Name string
}

type AuditEventRelationshipResponse struct {
	Type             string
	RelatedEventID   string
	RelatedEventType string
	Properties       map[string]any
}

type EntityAuditResponse struct {
	EntityType     string
	EntityId       string
	Events         []*AuditEventResponse
	Summary        *EntityAuditSummaryResponse
	ImpactAnalysis *ImpactAnalysisResponse
}

type EntityAuditSummaryResponse struct {
	TotalEvents    int
	EventsByType   []*EventTypeCountResponse
	EventsByStatus []*EventStatusCountResponse
	LastEvent      time.Time
	FirstEvent     time.Time
}

type EventTypeCountResponse struct {
	Type  string
	Count int
}

type EventStatusCountResponse struct {
	Status string
	Count  int
}

type ImpactAnalysisResponse struct {
	AffectedTerms      []string
	RelatedIncidents   []*IncidentResponse
	RiskScore          float64
	DownstreamEntities []*AuditEntityResponse
}

type IncidentResponse struct {
	ID                 string
	Title              string
	Description        string
	Status             string
	Severity           string
	DetectedAt         time.Time
	ResolvedAt         *time.Time
	RootCauseEvents    []*AuditEventResponse
	AffectedTerms      []string
	RootCauseAnalysis  *AISuggestionResponse
	RecommendedActions []string
	BlastRadius        string
}

type AISuggestionResponse struct {
	ID                 string
	Type               string
	Narrative          string
	Confidence         float64
	GeneratedBy        string
	GeneratedAt        time.Time
	RecommendedActions []string
	RelatedEventID     string
	RelatedEventType   string
}

type AIExplanationResponse struct {
	WhatHappened       string
	RootCause          string
	Severity           string
	BlastRadius        string
	AffectedEntities   []*AuditEntityResponse
	RecommendedActions []string
	ProposedChangeSet  string
	Confidence         float64
	RelatedEvents      []*AuditEventResponse
}

type ChangeSetImpactResponse struct {
	ChangeSetID        string
	Summary            string
	RiskScore          float64
	AffectedTerms      []string
	DownstreamImpacts  []*AuditEntityResponse
	PotentialIncidents []*IncidentResponse
}

type ComplianceStatusResponse struct {
	TotalChecks   int
	Violations    int
	Passing       int
	Critical      int
	High          int
	LastViolation *time.Time
	AffectedTerms []string
}

type AuditEventStatsResponse struct {
	TotalEvents         int
	EventsByType        []*EventTypeCountResponse
	EventsByStatus      []*EventStatusCountResponse
	EventsBySeverity    []*EventSeverityCountResponse
	TopImpactedEntities []*ImpactedEntityStatsResponse
	IncidentCount       int
	ViolationCount      int
	AvgDurationMs       *int
	CriticalCount       int
}

type EventSeverityCountResponse struct {
	Severity string
	Count    int
}

type ImpactedEntityStatsResponse struct {
	Type       string
	ID         string
	EventCount int
	LastEvent  time.Time
}

// Helper methods for graph traversal and AI integration

func (r *AuditGraphResolver) getEventDetails(ctx context.Context, eventId string, tenantIds []string) (*AuditEventResponse, error) {
	query := fmt.Sprintf(`
		SELECT event_id, node_type, event_time, event_status, severity, properties, tenant_id
		FROM audit.semantic_events
		WHERE event_id = '%s' AND tenant_id IN (%s)
		LIMIT 1
	`, eventId, formatTenantList(tenantIds))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	events := parseAuditEventResults(results)
	if len(events) == 0 {
		return nil, fmt.Errorf("event not found: %s", eventId)
	}

	return events[0], nil
}

func (r *AuditGraphResolver) getRelatedEvents(ctx context.Context, eventId string, tenantIds []string) ([]*AuditEventResponse, error) {
	// Query catalog edges to find related events
	query := fmt.Sprintf(`
		SELECT DISTINCT
		  e.event_id,
		  e.node_type,
		  e.event_time,
		  e.event_status,
		  e.severity,
		  e.properties,
		  e.tenant_id
		FROM audit.semantic_events e
		INNER JOIN catalog.catalog_edge edge ON (
		  edge.from_node_id = 'audit_event:%s' AND edge.to_node_id = e.event_id
		  OR edge.to_node_id = 'audit_event:%s' AND edge.from_node_id = e.event_id
		)
		WHERE e.tenant_id IN (%s)
		AND edge.edge_type IN ('has_impact_on', 'causes', 'has_ai_narrative', 'has_compliance_context')
		ORDER BY e.event_time DESC
		LIMIT 50
	`, eventId, eventId, formatTenantList(tenantIds))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	return parseAuditEventResults(results), nil
}

func (r *AuditGraphResolver) buildGraphAwarePrompt(event *AuditEventResponse, relatedEvents []*AuditEventResponse, eventType string) string {
	prompt := fmt.Sprintf(`Analyze the following audit event and its related events:

PRIMARY EVENT:
Type: %s
Status: %s
Severity: %s
Time: %s
Properties: %+v

RELATED EVENTS (%d):
`, event.Type, event.Status, event.Severity, event.Timestamp, event.Properties, len(relatedEvents))

	for i, re := range relatedEvents {
		if i < 10 { // Limit context
			prompt += fmt.Sprintf("- %s [%s] at %s\n", re.Type, re.Status, re.Timestamp)
		}
	}

	prompt += fmt.Sprintf(`

Based on this audit event graph, provide:
1. A concise summary of what happened
2. The likely root cause
3. The blast radius and affected entities
4. Recommended actions to resolve or prevent recurrence
5. Suggested compliance remediation if applicable

Format your response as structured JSON.
`)

	return prompt
}

func (r *AuditGraphResolver) callLLMForExplanation(ctx context.Context, prompt string) (*LLMExplanation, error) {
	// In production, integrate with AI service (Gemini, Claude, or O1)
	// Return structured response indicating AI service integration needed
	r.logger.Info("AI explanation requested", zap.Int("prompt_length", len(prompt)))

	return &LLMExplanation{
		Summary:            "AI-generated explanation pending integration with LLM service",
		RootCause:          "Requires AI service integration to determine root cause",
		Severity:           "MEDIUM",
		BlastRadius:        "Localized to specific tenant/workflow",
		AffectedEntities:   []*AuditEntityResponse{},
		RecommendedActions: []string{"Review event details", "Check related events", "Integrate AI service for deeper analysis"},
		ProposedChangeSet:  "",
		Confidence:         0.0,
	}, nil
}

type LLMExplanation struct {
	Summary            string
	RootCause          string
	Severity           string
	BlastRadius        string
	AffectedEntities   []*AuditEntityResponse
	RecommendedActions []string
	ProposedChangeSet  string
	Confidence         float64
}

func (r *AuditGraphResolver) analyzeEntityImpact(ctx context.Context, entityType, entityId string, tenantIds []string) *ImpactAnalysisResponse {
	// Query Trino for impact analysis
	query := fmt.Sprintf(`
		SELECT
		  COUNT(*) as total_events,
		  COUNT(DISTINCT tenant_id) as affected_tenants,
		  MAX(severity) as max_severity
		FROM audit.entity_timeline
		WHERE entity_id = '%s'
		  AND entity_type = '%s'
		  AND tenant_id IN (%s)
		  AND event_time >= CURRENT_TIMESTAMP - INTERVAL '7' DAY
	`, entityId, entityType, formatTenantList(tenantIds))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to analyze entity impact", zap.Error(err))
		return &ImpactAnalysisResponse{
			AffectedTerms:      []string{},
			RelatedIncidents:   []*IncidentResponse{},
			RiskScore:          0.0,
			DownstreamEntities: []*AuditEntityResponse{},
		}
	}

	// Parse results and compute impact score
	downstreamEntities := r.analyzeDownstreamImpacts(ctx, []string{entityId}, tenantIds)
	relatedIncidents := r.findRelatedIncidents(ctx, []string{entityId}, tenantIds)

	// Calculate risk score based on event count and severity
	var totalEvents int
	if resultMap, ok := results.(map[string]interface{}); ok {
		totalEvents, _ = resultMap["total_events"].(int)
	}

	riskScore := float64(totalEvents) * 0.1 // Simple scoring model
	if riskScore > 100.0 {
		riskScore = 100.0
	}

	return &ImpactAnalysisResponse{
		AffectedTerms:      []string{entityId},
		RelatedIncidents:   relatedIncidents,
		RiskScore:          riskScore,
		DownstreamEntities: downstreamEntities,
	}
}

func (r *AuditGraphResolver) generateRootCauseAnalysis(ctx context.Context, incidentId string) *AISuggestionResponse {
	// Query incident details and related events
	query := fmt.Sprintf(`
		SELECT DISTINCT
		  e.event_id,
		  e.node_type,
		  e.properties
		FROM audit.semantic_events e
		INNER JOIN catalog.catalog_edge edge ON edge.from_node_id = 'incident:%s'
		WHERE edge.edge_type = 'causes'
		ORDER BY e.event_time DESC
		LIMIT 20
	`, incidentId)

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to generate root cause", zap.Error(err))
		return &AISuggestionResponse{
			ID:               fmt.Sprintf("ai_suggestion:rca_%s", incidentId),
			Type:             "root_cause",
			Narrative:        "Unable to generate root cause analysis",
			Confidence:       0.0,
			GeneratedBy:      "system",
			GeneratedAt:      time.Now(),
			RelatedEventID:   incidentId,
			RelatedEventType: "incident",
		}
	}

	// In production, build prompt and call AI service
	events := parseAuditEventResults(results)
	_ = fmt.Sprintf("Analyze root cause for incident %s with %d related events", incidentId, len(events))

	return &AISuggestionResponse{
		ID:               fmt.Sprintf("ai_suggestion:rca_%s", incidentId),
		Type:             "root_cause",
		Narrative:        "Root cause analysis requires AI service integration. Related events identified: " + fmt.Sprint(len(events)),
		Confidence:       0.5,
		GeneratedBy:      "system",
		GeneratedAt:      time.Now(),
		RelatedEventID:   incidentId,
		RelatedEventType: "incident",
		RecommendedActions: []string{
			"Review related events",
			"Check semantic term lineage",
			"Integrate AI service for deeper analysis",
		},
	}
}

func (r *AuditGraphResolver) analyzeDownstreamImpacts(ctx context.Context, affectedTerms []string, tenantIds []string) []*AuditEntityResponse {
	if len(affectedTerms) == 0 {
		return []*AuditEntityResponse{}
	}

	// Query for entities affected by the semantic terms
	termList := formatStringList(affectedTerms)
	query := fmt.Sprintf(`
		SELECT DISTINCT
		  entity_type,
		  entity_id,
		  entity_name,
		  COUNT(*) as impact_count
		FROM audit.entity_timeline
		WHERE semantic_term_id IN (%s)
		  AND tenant_id IN (%s)
		  AND event_time >= CURRENT_TIMESTAMP - INTERVAL '30' DAY
		GROUP BY entity_type, entity_id, entity_name
		ORDER BY impact_count DESC
		LIMIT 100
	`, termList, formatTenantList(tenantIds))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to analyze downstream impacts", zap.Error(err))
		return []*AuditEntityResponse{}
	}

	// Simple parsing - in production would use proper result parser
	entities := []*AuditEntityResponse{}
	if resultRows, ok := results.([]map[string]interface{}); ok {
		for _, row := range resultRows {
			entities = append(entities, &AuditEntityResponse{
				Type: fmt.Sprint(row["entity_type"]),
				ID:   fmt.Sprint(row["entity_id"]),
				Name: fmt.Sprint(row["entity_name"]),
			})
		}
	}

	return entities
}

func (r *AuditGraphResolver) findRelatedIncidents(ctx context.Context, affectedTerms []string, tenantIds []string) []*IncidentResponse {
	if len(affectedTerms) == 0 {
		return []*IncidentResponse{}
	}

	termList := formatStringList(affectedTerms)
	query := fmt.Sprintf(`
		SELECT
		  incident_id,
		  affected_tenants,
		  affected_jobs,
		  affected_dags,
		  start_time,
		  end_time,
		  status,
		  event_count,
		  ai_root_cause
		FROM audit.incident_graph
		WHERE ARRAY_INTERSECT(affected_semantic_terms, ARRAY[%s]) IS NOT NULL
		  AND ARRAY_INTERSECT(affected_tenants, ARRAY[%s]) IS NOT NULL
		ORDER BY start_time DESC
		LIMIT 50
	`, termList, formatTenantList(tenantIds))

	results, err := r.trinoQuerier.Query(ctx, query)
	if err != nil {
		r.logger.Error("failed to find related incidents", zap.Error(err))
		return []*IncidentResponse{}
	}

	return parseIncidentResults(results)
}

// Parsing functions for Trino query results

func parseAuditEventResults(results interface{}) []*AuditEventResponse {
	events := []*AuditEventResponse{}

	if resultRows, ok := results.([]map[string]interface{}); ok {
		for _, row := range resultRows {
			evt := &AuditEventResponse{
				ID:       fmt.Sprint(row["event_id"]),
				Type:     fmt.Sprint(row["node_type"]),
				Status:   fmt.Sprint(row["event_status"]),
				Severity: fmt.Sprint(row["severity"]),
				TenantID: fmt.Sprint(row["tenant_id"]),
			}

			if ts, ok := row["event_time"].(time.Time); ok {
				evt.Timestamp = ts
			}

			if props, ok := row["properties"].(map[string]interface{}); ok {
				evt.Properties = props
			}

			events = append(events, evt)
		}
	}

	return events
}

func parseIncidentResults(results interface{}) []*IncidentResponse {
	incidents := []*IncidentResponse{}

	if resultRows, ok := results.([]map[string]interface{}); ok {
		for _, row := range resultRows {
			inc := &IncidentResponse{
				ID:          fmt.Sprint(row["incident_id"]),
				Status:      fmt.Sprint(row["status"]),
				Severity:    fmt.Sprint(row["severity"]),
				BlastRadius: fmt.Sprint(row["blast_radius"]),
			}

			if ts, ok := row["detected_at"].(time.Time); ok {
				inc.DetectedAt = ts
			}

			if affectedTerms, ok := row["affected_semantic_terms"].([]string); ok {
				inc.AffectedTerms = affectedTerms
			}

			if rootCause, ok := row["ai_root_cause"].(string); ok && rootCause != "" {
				inc.RootCauseAnalysis = &AISuggestionResponse{
					Narrative: rootCause,
				}
			}

			incidents = append(incidents, inc)
		}
	}

	return incidents
}

func parseChangeSetImpactResults(results interface{}) *ChangeSetImpactResponse {
	response := &ChangeSetImpactResponse{
		AffectedTerms:      []string{},
		DownstreamImpacts:  []*AuditEntityResponse{},
		PotentialIncidents: []*IncidentResponse{},
	}

	if resultRow, ok := results.(map[string]interface{}); ok {
		response.ChangeSetID = fmt.Sprint(resultRow["changeset_id"])
		response.Summary = fmt.Sprint(resultRow["summary"])

		if riskScore, ok := resultRow["risk_score"].(float64); ok {
			response.RiskScore = riskScore
		}

		if affectedTerms, ok := resultRow["affected_terms"].([]string); ok {
			response.AffectedTerms = affectedTerms
		}
	}

	return response
}

func parseComplianceStatusResults(results interface{}) *ComplianceStatusResponse {
	response := &ComplianceStatusResponse{
		AffectedTerms: []string{},
	}

	if resultRow, ok := results.(map[string]interface{}); ok {
		if totalChecks, ok := resultRow["total_checks"].(int); ok {
			response.TotalChecks = totalChecks
		}
		if violations, ok := resultRow["violations"].(int); ok {
			response.Violations = violations
		}
		if passing, ok := resultRow["passing"].(int); ok {
			response.Passing = passing
		}
		if critical, ok := resultRow["critical"].(int); ok {
			response.Critical = critical
		}
		if high, ok := resultRow["high"].(int); ok {
			response.High = high
		}
	}

	return response
}

func parseAuditEventStatsResults(results interface{}) *AuditEventStatsResponse {
	response := &AuditEventStatsResponse{
		EventsByType:        []*EventTypeCountResponse{},
		EventsByStatus:      []*EventStatusCountResponse{},
		EventsBySeverity:    []*EventSeverityCountResponse{},
		TopImpactedEntities: []*ImpactedEntityStatsResponse{},
	}

	if resultRow, ok := results.(map[string]interface{}); ok {
		if totalEvents, ok := resultRow["total_events"].(int); ok {
			response.TotalEvents = totalEvents
		}
		if incidentCount, ok := resultRow["incident_count"].(int); ok {
			response.IncidentCount = incidentCount
		}
		if violationCount, ok := resultRow["violation_count"].(int); ok {
			response.ViolationCount = violationCount
		}
		if criticalCount, ok := resultRow["critical_count"].(int); ok {
			response.CriticalCount = criticalCount
		}
	}

	return response
}

func calculateEntitySummary(events []*AuditEventResponse) *EntityAuditSummaryResponse {
	summary := &EntityAuditSummaryResponse{
		TotalEvents:    len(events),
		EventsByType:   []*EventTypeCountResponse{},
		EventsByStatus: []*EventStatusCountResponse{},
	}

	typeMap := make(map[string]int)
	statusMap := make(map[string]int)

	for _, evt := range events {
		typeMap[evt.Type]++
		statusMap[evt.Status]++

		if evt.Timestamp.After(summary.LastEvent) {
			summary.LastEvent = evt.Timestamp
		}
		if summary.FirstEvent.IsZero() || evt.Timestamp.Before(summary.FirstEvent) {
			summary.FirstEvent = evt.Timestamp
		}
	}

	// Convert maps to response structures
	for typeName, count := range typeMap {
		summary.EventsByType = append(summary.EventsByType, &EventTypeCountResponse{
			Type:  typeName,
			Count: count,
		})
	}

	for status, count := range statusMap {
		summary.EventsByStatus = append(summary.EventsByStatus, &EventStatusCountResponse{
			Status: status,
			Count:  count,
		})
	}

	return summary
}

// Helper functions for query building
// (formatStringList already defined above, removed duplicate)

// Interfaces for external dependencies

type TrinoQuerier interface {
	Query(ctx context.Context, sql string) (interface{}, error)
}

type CatalogReader interface {
	GetNode(ctx context.Context, nodeID string) (interface{}, error)
	GetEdges(ctx context.Context, fromNodeID, edgeType string) ([]interface{}, error)
}
