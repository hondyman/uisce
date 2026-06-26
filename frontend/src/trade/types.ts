export interface WorkflowDefinition {
    id: string;
    tenant_id: string;
    name: string;
    description?: string;
    status: 'draft' | 'active' | 'retired';
    stages: WorkflowStage[];
    created_at: string;
}

export interface WorkflowStage {
    id: string;
    workflow_id: string;
    name: string;
    order_index: number;
    config: StageConfig;
    created_at: string;
}

export interface StageConfig {
    fields: FieldDefinition[];
    actions?: string[];
}

export interface FieldDefinition {
    field: string;
    label: string;
    type: 'string' | 'number' | 'enum' | 'boolean' | 'date';
    required?: boolean;
    options?: string[]; // For enum
    validation?: string;
}

export interface TradeInput {
    tenant_id: string;
    workflow_name: string;
    data: Record<string, any>;
}

export interface WorkflowResult {
    workflow_id: string;
    run_id: string;
}
