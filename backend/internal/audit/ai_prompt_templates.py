"""
AI Prompt Templates for Audit Semantic Graph
These prompts guide LLMs (Gemini, Claude, etc.) to reason over the catalog graph
and provide rich explanations of audit events, incidents, and compliance issues.

All prompts are designed to:
1. Understand semantic relationships
2. Trace root causes across systems
3. Identify blast radius
4. Recommend remediation
5. Generate actionable ChangeSets
"""

# ============================================================================
# 1. Explain Audit Event Prompt
# ============================================================================
# Used for: job runs, DAG runs, changesets, any single event
# Output: Narrative, root cause, blast radius, recommendations

PROMPT_EXPLAIN_AUDIT_EVENT = """You are an expert in semantic graph reasoning and operational intelligence.

You will be given:
- An audit event node (job_run, dag_run, changeset_event, compliance_event, etc.)
- Its connected nodes in the semantic graph (related entities, semantic terms, tenants)
- Its edges (impact relationships, context relationships, etc.)
- Its properties (status, timestamps, error messages, metadata)

Your task:
1. Explain what happened in clear, non-technical language
2. Identify the root cause using graph context (traverse backward through CAUSES edges)
3. Identify the blast radius by traversing downstream IMPACTS and HAS_SEMANTIC_CONTEXT edges
4. Identify which semantic terms, business objects, or APIs were affected
5. Recommend immediate actions and longer-term fixes
6. If appropriate, propose a ChangeSet summary

Format your response as JSON with these exact fields:
{
  "what_happened": "Clear narrative of what occurred",
  "root_cause": "Identified root cause, traced through graph",
  "severity": "CRITICAL|HIGH|MEDIUM|LOW",
  "blast_radius": "Description of affected systems and data",
  "affected_entities": [
    {"type": "semantic_term", "id": "...", "name": "..."},
    {"type": "job", "id": "...", "name": "..."},
    {"type": "api", "id": "...", "name": "..."}
  ],
  "recommended_actions": [
    "Immediate: ...",
    "Short-term: ...",
    "Long-term: ..."
  ],
  "proposed_changeset": {
    "title": "Fix for ...",
    "description": "...",
    "impact_summary": "..."
  },
  "confidence": 0.85,
  "explanation": "Why you believe this analysis is correct based on the graph"
}

Context from the semantic graph:
{graph_context}

Event being analyzed:
{event_json}

Think carefully through the graph relationships before responding.
"""

# ============================================================================
# 2. Explain Incident Prompt
# ============================================================================
# Used for: incidents, clusters of related failures
# Output: Root cause, affected systems, remediation, changeset proposal

PROMPT_EXPLAIN_INCIDENT = """You are analyzing an INCIDENT in a semantic graph.

The incident node has these connected relationships:
- CAUSES edges pointing TO job_run/dag_run/audit_event nodes (root causes)
- HAS_SEMANTIC_CONTEXT edges pointing TO semantic_term nodes (affected terms)
- HAS_TENANT edges pointing TO tenant nodes (affected organizations)

Your task:
1. Summarize the incident based on its properties and connected graph
2. Trace CAUSES edges backward to identify root cause events
3. Identify all impacted semantic terms (entities, data, APIs)
4. Identify all impacted tenants (for multi-tenant isolation verification)
5. Estimate likelihood of recurrence based on root cause
6. Recommend remediation (immediate, short-term, long-term)
7. Propose a ChangeSet to prevent recurrence

Format your response as JSON:
{
  "incident_summary": "Narrative of what happened and impact",
  "root_cause_analysis": {
    "primary_cause": "Main root cause identified",
    "contributing_factors": ["Factor 1", "Factor 2"],
    "root_cause_events": [
      {"event_id": "...", "type": "job_run|dag_run|...", "timestamp": "..."}
    ]
  },
  "impact_assessment": {
    "impacted_tenants": ["Tenant-1", "Tenant-2"],
    "impacted_semantic_terms": ["Term-1", "Term-2", "Term-3"],
    "blast_radius": "Description of scope",
    "affected_users": "Estimated number of users",
    "business_impact": "Estimated business loss/risk"
  },
  "severity": "CRITICAL|HIGH|MEDIUM|LOW",
  "recurrence_likelihood": {
    "probability": "high|medium|low",
    "reasoning": "Why this might happen again"
  },
  "recommended_remediation": {
    "immediate": ["Action 1", "Action 2"],
    "short_term": ["Action 1", "Action 2"],
    "long_term": ["Action 1", "Action 2"]
  },
  "proposed_changeset": {
    "title": "Prevent recurrence of ...",
    "description": "Detailed description",
    "affected_terms": ["Term-1", "Term-2"],
    "risk_mitigation": "How this fixes the issue"
  },
  "confidence": 0.92,
  "notes": "Additional context or caveats"
}

Incident data:
{incident_json}

Connected graph:
{graph_context}

Reason through the graph relationships carefully.
"""

# ============================================================================
# 3. Explain Compliance Violation Prompt
# ============================================================================
# Used for: compliance events, regulatory violations, policy breaches
# Output: Violation details, risk assessment, remediation

PROMPT_EXPLAIN_COMPLIANCE_VIOLATION = """You are analyzing a COMPLIANCE VIOLATION in a semantic graph.

The compliance event has these properties:
- rule: The compliance rule that was violated
- severity: CRITICAL|HIGH|MEDIUM|LOW
- affected_terms: Semantic terms involved in the violation
- description: Details of the violation

Connected graph:
- HAS_COMPLIANCE_CONTEXT edges to semantic_term/business_term nodes
- HAS_TENANT edges to affected tenants
- CAUSES edges from incident nodes (if this triggered incidents)

Your task:
1. Explain the compliance violation in business terms
2. Identify which data/systems/users were impacted
3. Assess regulatory risk and remediation timeline
4. Recommend immediate containment and long-term fix
5. Propose ChangeSet to prevent recurrence
6. Verify multi-tenant isolation wasn't breached

Format your response as JSON:
{
  "violation_summary": "Business-friendly explanation",
  "compliance_rule": "Rule being violated",
  "regulatory_context": {
    "regulation": "GDPR|CCPA|SOC2|...",
    "risk_level": "CRITICAL|HIGH|MEDIUM|LOW",
    "remediation_deadline": "Hours|Days|Weeks"
  },
  "affected_scope": {
    "affected_tenants": ["Tenant-1"],
    "affected_data_types": ["PII", "Health", "Financial"],
    "affected_entities": ["Customer data", "Transaction logs"],
    "estimated_records": 1000,
    "data_sensitivity": "HIGH|MEDIUM|LOW"
  },
  "root_cause": "Why this violation occurred",
  "containment": {
    "immediate_steps": ["Step 1", "Step 2"],
    "isolation_status": "Data is isolated|Data may be exposed|Unknown"
  },
  "remediation": {
    "immediate": ["Within 24h: ..."],
    "short_term": ["Within 1 week: ..."],
    "long_term": ["Permanent fix: ..."],
    "notification_required": true,
    "regulatory_reporting": true
  },
  "proposed_changeset": {
    "title": "Remediate compliance violation: ...",
    "description": "...",
    "priority": "CRITICAL"
  },
  "confidence": 0.88
}

Compliance event:
{compliance_event_json}

Graph context:
{graph_context}

Be thorough and conservative in risk assessment.
"""

# ============================================================================
# 4. Analyze Impact of ChangeSet Prompt
# ============================================================================
# Used for: analyzing impact of proposed changes
# Output: Risk assessment, affected entities, recommended safeguards

PROMPT_ANALYZE_CHANGESET_IMPACT = """You are analyzing the IMPACT of a proposed ChangeSet in a semantic graph.

The ChangeSet has:
- affected_terms: Semantic terms being modified
- impact_edges: HAS_IMPACT_ON edges to semantic_term nodes
- risk_score: 0-100 risk assessment

Your task:
1. Identify all directly affected semantic terms
2. Trace HAS_IMPACT_ON edges to find downstream impacts
3. Identify business processes that depend on affected terms
4. Identify potential job failures (via JOB_RUN -> HAS_SEMANTIC_CONTEXT edges)
5. Assess risk to data quality and user-facing APIs
6. Recommend testing strategy and rollback plan
7. Identify tenants that will be impacted

Format your response as JSON:
{
  "change_summary": "What is being changed and why",
  "directly_affected": {
    "semantic_terms": ["Term-1", "Term-2"],
    "definitions": "How definitions are changing"
  },
  "downstream_impacts": {
    "impacted_jobs": ["job-1", "job-2"],
    "impacted_dags": ["dag-1"],
    "impacted_apis": ["api-1"],
    "impacted_reports": ["report-1", "report-2"],
    "impacted_dashboards": ["dashboard-1"]
  },
  "risk_assessment": {
    "data_quality_risk": "HIGH|MEDIUM|LOW",
    "user_impact_risk": "HIGH|MEDIUM|LOW",
    "revenue_impact_risk": "HIGH|MEDIUM|LOW",
    "compliance_risk": "HIGH|MEDIUM|LOW",
    "technical_risk": "HIGH|MEDIUM|LOW",
    "overall_risk_score": 65
  },
  "tenant_impact": {
    "all_tenants_affected": true,
    "critical_tenants": ["Tenant-A"],
    "impact_distribution": "How tenants are affected differently"
  },
  "recommended_safeguards": [
    "Enable gradual rollout to 10% of tenants first",
    "Monitor job failure rate on Term-1 for 48h",
    "Have rollback plan ready",
    "Notify affected teams 24h in advance"
  ],
  "testing_strategy": {
    "unit_tests": "Test recalculation logic",
    "integration_tests": "Test dependent jobs",
    "user_acceptance": "Required for User-Facing APIs",
    "shadow_mode": "Run parallel calculations for 24h"
  },
  "rollback_plan": "How to undo if something goes wrong",
  "confidence": 0.91
}

ChangeSet being analyzed:
{changeset_json}

Graph context (affected entities and relationships):
{graph_context}

Be conservative with risk assessment - assume worst case until proven otherwise.
"""

# ============================================================================
# 5. Root Cause Analysis for Job Failure Prompt
# ============================================================================
# Used for: job_run failures, explaining why jobs are failing
# Output: Root cause, contributing factors, fix recommendations

PROMPT_ROOT_CAUSE_JOB_FAILURE = """You are analyzing a JOB FAILURE in a semantic graph.

The job_run node shows:
- status: FAILED
- error_message: The error that occurred
- semantic_terms: Terms the job depends on
- changeset_id: If caused by a recent change

Connected graph:
- RUNS_JOB edge to the job definition
- HAS_SEMANTIC_CONTEXT edges to semantic terms the job depends on
- Potentially CAUSED_BY edges from incident/changeset nodes

Your task:
1. Classify the failure (data quality|schema change|data volume|permission|timeout|etc)
2. Identify if caused by a recent ChangeSet
3. Identify semantic term issues (definition changes, value changes)
4. Check if incident involved related jobs
5. Assess if this is isolated or pattern
6. Recommend immediate fix and prevention

Format your response as JSON:
{
  "failure_classification": "Schema change|Data quality|Timeout|Permission|Logic error|...",
  "immediate_cause": "The direct cause of failure",
  "root_causes": [
    {"cause": "Change to Term-X definition", "evidence": "..."},
    {"cause": "Data volume spike in Table-Y", "evidence": "..."}
  ],
  "is_recent_changeset_related": true,
  "related_changeset": {
    "id": "changeset-123",
    "what_changed": "Definition of Term-X",
    "likelihood_this_caused_failure": "HIGH|MEDIUM|LOW"
  },
  "similar_failures": {
    "pattern_detected": true,
    "other_jobs_failing": ["job-2", "job-3"],
    "failure_frequency": "First time|Recurring|Daily",
    "affected_tenants": ["Tenant-A", "Tenant-B"]
  },
  "immediate_actions": [
    "Roll back changeset-123",
    "Increase timeout for this job",
    "Manual data quality check for Term-X"
  ],
  "prevention": {
    "short_term": "Add data validation before processing",
    "long_term": "Add automated testing for schema changes",
    "monitoring": "Alert on Term-X anomalies"
  },
  "confidence": 0.87
}

Job failure data:
{job_run_json}

Related events and context:
{graph_context}

Use all available graph relationships to build a comprehensive picture.
"""

# ============================================================================
# 6. Multi-Tenant Impact Assessment Prompt
# ============================================================================
# Used for: assessing cross-tenant impact of events
# Output: Which tenants affected, isolation verification

PROMPT_ASSESS_MULTI_TENANT_IMPACT = """You are assessing MULTI-TENANT IMPACT of an audit event.

This is critical for multi-tenant security and isolation verification.

The event has:
- affected_terms: Semantic terms involved
- affected_tenants: Tenants connected via HAS_TENANT edges
- incident_scope: If incident, which entities are affected

Your task:
1. Verify that HAS_TENANT edges correctly isolate by tenant
2. Verify that affected semantic terms don't accidentally leak cross-tenant
3. Assess if isolation was breached
4. Identify all tenants that could be affected
5. Recommend isolation reinforcement if needed

Format your response as JSON:
{
  "event_isolation_status": "PROPERLY_ISOLATED|POTENTIALLY_LEAKED|BREACHED",
  "isolation_verification": {
    "has_tenant_edges_present": true,
    "edges_correctly_scoped": true,
    "data_flow_between_tenants": "None detected|Potential leak|Confirmed leak"
  },
  "affected_tenants": {
    "confirmed": ["Tenant-A", "Tenant-B"],
    "potentially_exposed": ["Tenant-C"],
    "isolated": ["Tenant-D", "Tenant-E"]
  },
  "breach_assessment": {
    "breach_detected": false,
    "data_exposure_type": "None|Metadata|PII|Full data",
    "regulatory_impact": "None|Notification required|Incident report required"
  },
  "recommendations": [
    "Audit ChangeSet-X for tenant-scoping issues",
    "Add data lineage validation to prevent cross-tenant flows",
    "Review query execution to ensure row-level security"
  ],
  "confidence": 0.94
}

Event data:
{event_json}

Tenant isolation graph:
{tenant_isolation_context}

This is security-critical - err on the side of caution.
"""

# ============================================================================
# 7. System Health Summary Prompt
# ============================================================================
# Used for: high-level health summaries
# Output: Overall system health, key issues, trends

PROMPT_SYSTEM_HEALTH_SUMMARY = """You are generating a SYSTEM HEALTH SUMMARY from audit graph data.

You have:
- Recent event statistics (job success rates, failure types, incident counts)
- Trending data (is system getting more/less stable?)
- Critical incidents and compliance violations
- Semantic term health (are definitions stable?)

Your task:
1. Summarize overall system health (Healthy|Degraded|Critical)
2. Identify top issues (what's failing most)
3. Assess trends (getting better or worse?)
4. Highlight critical areas needing attention
5. Provide executive summary (1-2 paragraphs)

Format your response as JSON:
{
  "overall_health": "HEALTHY|DEGRADED|CRITICAL",
  "health_score": 78,
  "summary": "Executive summary of system health in 2 paragraphs",
  "top_issues": [
    {"issue": "Job X failing 40% of time", "severity": "HIGH", "trend": "Worsening"},
    {"issue": "Compliance violations on Data-Y", "severity": "CRITICAL", "trend": "New"}
  ],
  "trends": {
    "job_failure_rate": "Increasing from 2% to 5%",
    "incident_frequency": "Stable",
    "compliance_violations": "Decreasing",
    "system_stability": "Degrading"
  },
  "recommendations": [
    "Immediate: Investigate Job X failures",
    "This week: Audit compliance violations on Data-Y",
    "This month: Address systematic data quality issues"
  ],
  "confidence": 0.85
}

Event statistics:
{stats_json}

Trend data:
{trends_json}

Historical context:
{historical_context}

Provide insights a CTO would care about.
"""

# ============================================================================
# Helper function for building complete context
# ============================================================================

def build_graph_context(event_id: str, catalog_reader) -> str:
    """
    Build the complete semantic graph context for a prompt.
    Fetches the event node, all connected nodes, and edge relationships.
    """
    # Get the main event node
    event_node = catalog_reader.get_node(event_id)
    
    # Get all outgoing edges (impact, causes, context)
    outgoing_edges = catalog_reader.get_edges(event_id, None)
    
    # Get all incoming edges (what caused this event)
    incoming_edges = catalog_reader.get_edges(None, event_id)
    
    # Build connected node list
    connected_nodes = []
    for edge in outgoing_edges + incoming_edges:
        node = catalog_reader.get_node(edge.target_node_id or edge.source_node_id)
        connected_nodes.append(node)
    
    # Format as JSON context
    context = {
        "main_event": event_node,
        "connected_nodes": connected_nodes,
        "outgoing_relationships": outgoing_edges,
        "incoming_relationships": incoming_edges
    }
    
    import json
    return json.dumps(context, indent=2)
