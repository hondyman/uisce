// Additional generated types for rule engine
// This file contains supporting types for the ASL rule system

export interface RuleEngineConfig {
  tenantId: string;
  ruleVersion: number;
  enableTracing: boolean;
  maxExecutionTime: number;
}

export interface RuleExecutionContext {
  tenantId: string;
  entityId: string;
  entityType: string;
  data: Record<string, any>;
  metadata: Record<string, any>;
}

export interface RuleExecutionResult {
  ruleId: string;
  passed: boolean;
  message?: string;
  details?: any;
  executionTime: number;
}

export interface ValidationError {
  field: string;
  message: string;
  severity: "error" | "warning" | "info";
  code?: string;
}

export interface RuleMetrics {
  totalRules: number;
  executedRules: number;
  failedRules: number;
  averageExecutionTime: number;
  cacheHitRate: number;
}