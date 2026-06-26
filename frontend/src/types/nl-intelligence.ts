export interface NLRequest {
    question: string;
    tenant_scope: string[];
}

export interface NLResponse {
    intent: string;
    query_plan: QueryPlan;
    explanation?: string;
}

export interface QueryPlan {
    type: string;
    engine: string;
    sql?: string;
    cypher?: string;
    graph_name?: string;
    parameters: Record<string, any>;
    dialect?: string;
}

export interface IncidentExplanation {
    incident_id: string;
    narrative: string;
    root_cause_nodes: string[];
    affected_nodes: string[];
    severity: string;
}

export interface ChangeSetProposal {
    id: string;
    title: string;
    description: string;
    changes: Change[];
    impact_score: number;
}

export interface Change {
    action: 'CREATE' | 'UPDATE' | 'DELETE' | 'REPLACE';
    entity_type: string;
    entity_id: string;
    properties: Record<string, any>;
}

export interface ForecastResult {
    target_id: string;
    failure_probability: number;
    predicted_failure_time?: string;
    contributing_factors: string[];
    confidence_score: number;
}
