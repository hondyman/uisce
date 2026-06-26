export interface ChangeReview {
    id: string;
    change_set_id: string;
    status: 'pending' | 'approved' | 'rejected' | 'promoted';
    diff_summary: Record<string, SemanticDiffDTO>;
    lineage_impact: Record<string, ImpactReport>;
    test_results: TestResult[];
    created_at: string;
    updated_at: string;
    approved_by?: string;
    approved_at?: string;
    ai_summary?: string;
    ai_risk_score?: number;
    ai_risk_level?: string;
    // ai_risk_level string // 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
}

export interface ChangeSet {
    id: string;
    tenant_id: string;
    name: string;
    description: string;
    status: 'open' | 'submitted' | 'approved' | 'rejected' | 'applied';
    created_by: string;
    created_at: string;
}

export interface SemanticDiffDTO {
    [key: string]: {
        changes: SemanticDiffChange[];
    };
}

export interface SemanticDiffChange {
    path: string;
    old?: any;
    new?: any;
    type: 'idx' | 'modified' | 'added' | 'removed';
}

export interface ImpactReport {
    object_id: string;
    impact_score: number;
    affected_nodes: ImpactNode[];
}

export interface ImpactNode {
    node_id: string;
    node_type: string;
    is_direct: boolean; // direct dependency vs downstream
}

export interface TestResult {
    test_name: string;
    passed: boolean;
    duration_ms: number;
    error?: string;
    logs: string[];
}
