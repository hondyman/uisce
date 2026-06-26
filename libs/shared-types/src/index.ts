// Shared Types Library
// Common types used across all services

// Wealth Management Types
export interface UMA {
  id: string;
  aum: number;
  tax_saved: number;
  status: 'active' | 'rebalancing' | 'alpha_rebalanced';
  holdings: Holding[];
}

export interface DirectIndex {
  id: string;
  aum: number;
  drift: number;
  tax_saved: number;
  esg_score: number;
  status: 'active' | 'optimizing' | 'alpha_optimized';
  holdings: Holding[];
}

export interface Holding {
  symbol: string;
  quantity: number;
  price: number;
  value: number;
  weight: number;
}

export interface AttributionResult {
  total_return: number;
  benchmark_return: number;
  alpha: number;
  factors: FactorAttribution[];
}

export interface FactorAttribution {
  factor: string;
  contribution: number;
  t_stat: number;
}

export interface TaxOptimization {
  lots_selected: TaxLot[];
  tax_saved: number;
  esg_score: number;
  wash_sale_risk: number;
}

export interface TaxLot {
  symbol: string;
  quantity: number;
  basis: number;
  gain_loss: number;
  holding_period: number;
}

// Fabric Builder Types
export interface FabricModel {
  id: string;
  name: string;
  description: string;
  tenant_id: string;
  datasource_id: string;
  schema: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface Extension {
  id: string;
  name: string;
  type: string;
  tenant_id: string;
  datasource_id: string;
  config: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface ValidationResult {
  valid: boolean;
  errors: string[];
  model: FabricModel;
}

export interface CompatibilityReport {
  datasource_id: string;
  compatible: boolean;
  report: string;
}

// Business Process Types
export interface BusinessProcess {
  id: string;
  tenant_id: string;
  datasource_id: string;
  processName: string;
  entity: string;
  description: string;
  steps: BPStep[];
  isActive: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt?: string;
  version: number;
  tags: string[];
}

export interface BPStep {
  id: string;
  stepOrder: number;
  stepType: string;
  stepName: string;
  durationHours: number;
  assigneeRole?: string;
  assigneeUser?: string;
  validationRules?: string[];
  notificationTemplate?: string;
  conditionLogic?: ConditionBranch;
  description?: string;
  status?: string;
  escalationThresholdHours?: number;
}

export interface ConditionBranch {
  condition: string;
  trueStepId: string;
  falseStepId: string;
}

export interface ProcessStepType {
  id: string;
  key: string;
  label: string;
  description?: string;
  icon_svg?: string;
  default_data: any;
  created_at: string;
  updated_at: string;
}

export interface ValidationOperator {
  id: string;
  key: string;
  label: string;
  description?: string;
  value_type: string;
  config?: any;
  created_at: string;
  updated_at: string;
}

export interface WorkflowEvent {
  id: string;
  key: string;
  label: string;
  description?: string;
  event_type: string;
  config?: any;
  created_at: string;
  updated_at: string;
}

export interface ProcessExecution {
  id: string;
  execution_id: string;
  status: string;
  started_at: string;
  current_step?: string;
  progress?: number;
}

// Enhanced Temporal Types
export interface TemporalWorkflowOptions {
  taskQueue: string;
  workflowId?: string;
  timeout?: number;
  retryPolicy?: {
    initialInterval: number;
    backoffCoefficient: number;
    maximumInterval: number;
    maximumAttempts: number;
  };
}

export interface TemporalActivityOptions {
  startToCloseTimeout: number;
  retryPolicy?: {
    initialInterval: number;
    backoffCoefficient: number;
    maximumInterval: number;
    maximumAttempts: number;
  };
}

// Database Types
export interface DatabaseConnection {
  host: string;
  port: number;
  database: string;
  username: string;
  password: string;
  ssl: boolean;
}

export interface QueryResult<T = any> {
  rows: T[];
  count: number;
  executionTime: number;
}

// Configuration Types
export interface ServiceConfig {
  name: string;
  version: string;
  port: number;
  database: DatabaseConnection;
  temporal?: {
    host: string;
    namespace: string;
  };
  hasura?: {
    endpoint: string;
    adminSecret: string;
  };
  ai?: {
    apiKey: string;
    baseURL: string;
  };
}

// Event Types
export interface DomainEvent {
  id: string;
  type: string;
  aggregateId: string;
  aggregateType: string;
  data: Record<string, any>;
  metadata: {
    timestamp: string;
    userId?: string;
    tenantId?: string;
    correlationId?: string;
  };
}

// Error Types
export interface ServiceError {
  code: string;
  message: string;
  details?: Record<string, any>;
  stack?: string;
}

// Health Check Types
export interface HealthStatus {
  service: string;
  status: 'healthy' | 'unhealthy' | 'degraded';
  timestamp: string;
  checks: Array<{
    name: string;
    status: 'pass' | 'fail' | 'warn';
    message?: string;
  }>;
}