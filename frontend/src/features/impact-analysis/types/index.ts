export type NodeType =
    | 'business_object'
    | 'BO_FIELD'
    | 'semantic_term'
    | 'DB_COLUMN'
    | 'API_ENDPOINT'
    | 'BI_ARTIFACT'
    | 'AI_ARTIFACT'
    | 'ACCESS_RULE'
    | 'calculation_term';

export interface ImpactNode {
    id: string;
    type: NodeType;
    label: string;
    properties: Record<string, any>;
}

export interface ImpactEdge {
    id: string;
    source: string;
    target: string;
    type: string;
    properties: Record<string, any>;
}

export interface ImpactGraphData {
    nodes: ImpactNode[];
    edges: ImpactEdge[];
}

export interface ImpactSummary {
    totalNodes: number;
    nodesByType: Record<string, number>; // using string key because Record<NodeType, number> might be strict on missing keys
    affectedArtifacts: Record<string, ImpactNode[]>;
    explanation: string;
    recommendations?: string[];
}
