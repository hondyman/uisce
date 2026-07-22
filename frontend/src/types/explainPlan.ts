export interface ExplainStep {
  step_id: number;
  operation: string;
  target_table: string;
  alias: string;
  condition?: string;
  cost_weight: number;
}

export interface ExplainPlanResult {
  bo_key: string;
  tenant_id: string;
  dialect: string;
  complexity_score: number;
  has_cross_source_join: boolean;
  has_aggregations: boolean;
  ast_depth: number;
  generated_sql: string;
  execution_steps: ExplainStep[];
  tenant_scoped: boolean;
  timestamp: string;
}
