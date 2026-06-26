/**
 * Business Process Service
 * TypeScript client for Business Process API
 * Handles BP CRUD operations, execution, and integration with existing backend
 */

import { apiGet, apiPost } from '../utils/api';
import { devError } from '../utils/devLogger';

// ============================================================================
// TYPE DEFINITIONS - matching backend business_process_api.go
// ============================================================================

export interface BusinessProcess {
  id: string;
  processName: string;
  description: string;
  isActive: boolean;
  version: number;
  stepCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface CreateBPRequest {
  process_name: string;
  description: string;
  steps: CreateBPStep[];
}

export interface CreateBPStep {
  step_order: number;
  step_type: string;
  step_name: string;
  duration_hours: number;
  assignee_role?: string;
  assignee_user?: string;
  trigger_ids?: string[];
  condition_json?: any;
  action_config?: any;
}

export interface BPExecution {
  instance_id: string;
  process_id: string;
  process_name: string;
  entity_id: string;
  entity_type: string;
  current_step: number;
  status: string;
  instance_data: Record<string, any>;
  started_at: string;
  current_step_started_at: string;
  current_step_due_at: string;
  temporal_workflow_id?: string;
  created_at: string;
}

export interface StartBPRequest {
  entity_id: string;
  entity_type: string;
  data: Record<string, any>;
}

// ============================================================================
// BUSINESS PROCESS CRUD OPERATIONS
// ============================================================================

/**
 * List all business processes for the current tenant
 */
export const getBusinessProcesses = async (): Promise<BusinessProcess[]> => {
  try {
    const response = await apiGet('bp');
    return response.bps || [];
  } catch (error) {
    devError('Failed to fetch business processes:', error);
    throw error;
  }
};

/**
 * Get a specific business process by ID
 */
export const getBusinessProcess = async (id: string): Promise<BusinessProcess> => {
  try {
    return await apiGet(`bp/${id}`);
  } catch (error) {
    devError('Failed to fetch business process:', error);
    throw error;
  }
};

/**
 * Create a new business process
 */
export const createBusinessProcess = async (bp: CreateBPRequest): Promise<{ id: string }> => {
  try {
    return await apiPost('bp', bp);
  } catch (error) {
    devError('Failed to create business process:', error);
    throw error;
  }
};

/**
 * Start a business process execution
 */
export const startBusinessProcess = async (processId: string, request: StartBPRequest): Promise<BPExecution> => {
  try {
    return await apiPost(`bp/${processId}/start`, request);
  } catch (error) {
    devError('Failed to start business process:', error);
    throw error;
  }
};

/**
 * Get business process execution status
 */
export const getBPExecution = async (instanceId: string): Promise<BPExecution> => {
  try {
    return await apiGet(`bp/instance/${instanceId}`);
  } catch (error) {
    devError('Failed to fetch BP execution:', error);
    throw error;
  }
};

// ============================================================================
// VALIDATION RULES INTEGRATION
// ============================================================================

/**
 * Get validation rules for rule builder
 */
export const getValidationRules = async (): Promise<any[]> => {
  try {
    return await apiGet('validation-rules');
  } catch (error) {
    devError('Failed to fetch validation rules:', error);
    // Return empty array as fallback
    return [];
  }
};

// ============================================================================
// BUSINESS OBJECT SCHEMA INTEGRATION
// ============================================================================

/**
 * Get business object schemas from bundles
 */
export const getBusinessObjectSchemas = async (): Promise<any[]> => {
  try {
    return await apiGet('bundles');
  } catch (error) {
    devError('Failed to fetch business object schemas:', error);
    // Return empty array as fallback
    return [];
  }
};

// ============================================================================
// EVENT/TRIGGER INTEGRATION
// ============================================================================

/**
 * Get available workflow events
 */
export const getWorkflowEvents = async (): Promise<any[]> => {
  try {
    return await apiGet('workflow-events');
  } catch (error) {
    devError('Failed to fetch workflow events:', error);
    // Return empty array as fallback
    return [];
  }
};

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/**
 * Convert ReactFlow process to backend format
 */
export const convertProcessToBackend = (process: any): CreateBPRequest => {
  return {
    process_name: process.processName || 'New Process',
    description: process.description || '',
    steps: (process.steps || []).map((step: any, index: number) => ({
      step_order: index + 1,
      step_type: step.stepType,
      step_name: step.stepName || `${step.stepType} Step ${index + 1}`,
      duration_hours: step.durationHours || 24,
      assignee_role: step.assigneeRole,
      assignee_user: step.assigneeUser,
      trigger_ids: step.triggerIds || [],
      condition_json: step.conditionJson,
      action_config: step.actionConfig
    }))
  };
};

/**
 * Convert backend process to ReactFlow format
 */
export const convertProcessFromBackend = (bp: BusinessProcess): any => {
  return {
    id: bp.id,
    processName: bp.processName,
    description: bp.description,
    isActive: bp.isActive,
    version: bp.version,
    steps: [] // Steps would need to be fetched separately
  };
};