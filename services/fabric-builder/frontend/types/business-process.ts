// Business Process-related type definitions
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