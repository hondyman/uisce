/**
 * Rules API Service
 *
 * Client-side service for interacting with the Rules Engine API.
 * Handles all CRUD operations for semantic rules and simulations.
 */

const API_BASE = process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1';

/**
 * Semantic term from business object catalog
 */
export interface SemanticTerm {
  id: string;
  name: string;
  nodeType: string;
  dataType: 'STRING' | 'NUMBER' | 'BOOLEAN' | 'DATE';
  businessDefinition: string;
  sampleValues: string[];
  governanceStatus: 'APPROVED' | 'DRAFT' | 'DEPRECATED';
  category: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateRuleRequest {
  businessObject: string;
  name: string;
  description: string;
  steps: Array<{
    priority: number;
    condition: {
      semanticTerm: string;
      operator: string;
      value: string;
    };
    action: {
      useField: string;
      confidence: number;
    };
    description: string;
  }>;
  defaultAction?: string;
}

export interface Rule extends CreateRuleRequest {
  id: string;
  version: number;
  status: 'draft' | 'testing' | 'staging' | 'production';
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}

export interface SimulateRuleRequest {
  ruleId: string;
  testData: any;
}

export interface SimulateRuleResponse {
  executionTrace: any[];
  impactedDates: number;
  changedDates: number;
  avgConfidence: number;
}

export interface ApprovalRequest {
  ruleId: string;
  version: number;
  role: string;
  action: 'approve' | 'reject';
  comments?: string;
}

/**
 * Create a new rule
 */
export const createRule = async (request: CreateRuleRequest): Promise<Rule> => {
  const response = await fetch(`${API_BASE}/rules`, {
    method: 'POST',
    headers: buildHeaders(),
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to create rule' }));
    throw new Error(error.message || 'Failed to create rule');
  }

  return response.json();
};

/**
 * Get rule by ID
 */
export const getRule = async (ruleId: string): Promise<Rule> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}`, {
    headers: buildHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to fetch rule' }));
    throw new Error(error.message || 'Failed to fetch rule');
  }

  return response.json();
};

/**
 * Update rule (draft only)
 */
export const updateRule = async (ruleId: string, updates: Partial<CreateRuleRequest>): Promise<Rule> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}`, {
    method: 'PUT',
    headers: buildHeaders(),
    body: JSON.stringify(updates),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to update rule' }));
    throw new Error(error.message || 'Failed to update rule');
  }

  return response.json();
};

/**
 * Delete rule (draft only)
 */
export const deleteRule = async (ruleId: string): Promise<void> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}`, {
    method: 'DELETE',
    headers: buildHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to delete rule' }));
    throw new Error(error.message || 'Failed to delete rule');
  }
};

/**
 * List rules for a business object
 */
export const listRules = async (businessObject: string, status?: string): Promise<Rule[]> => {
  const params = new URLSearchParams({ businessObject });
  if (status) params.append('status', status);

  const response = await fetch(`${API_BASE}/rules?${params}`, {
    headers: buildHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to fetch rules' }));
    throw new Error(error.message || 'Failed to fetch rules');
  }

  return response.json();
};

/**
 * Publish rule to testing stage
 */
export const publishRule = async (
  ruleId: string,
  version: number,
  description: string
): Promise<Rule> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}/publish`, {
    method: 'POST',
    headers: buildHeaders(),
    body: JSON.stringify({ version, description }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to publish rule' }));
    throw new Error(error.message || 'Failed to publish rule');
  }

  return response.json();
};

/**
 * Promote rule between stages
 */
export const promoteRule = async (
  ruleId: string,
  version: number,
  toStage: string
): Promise<Rule> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}/promote`, {
    method: 'POST',
    headers: buildHeaders(),
    body: JSON.stringify({ version, toStage }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to promote rule' }));
    throw new Error(error.message || 'Failed to promote rule');
  }

  return response.json();
};

/**
 * Simulate rule against test data
 */
export const simulateRule = async (
  ruleId: string,
  testData: any
): Promise<SimulateRuleResponse> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}/simulate`, {
    method: 'POST',
    headers: buildHeaders(),
    body: JSON.stringify({ testData }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Simulation failed' }));
    throw new Error(error.message || 'Simulation failed');
  }

  return response.json();
};

/**
 * Get rule version history
 */
export const getRuleVersions = async (ruleId: string): Promise<any[]> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}/versions`, {
    headers: buildHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to fetch rule versions' }));
    throw new Error(error.message || 'Failed to fetch rule versions');
  }

  return response.json();
};

/**
 * Get version diff
 */
export const getVersionDiff = async (ruleId: string, v1: number, v2: number): Promise<any> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}/diff?v1=${v1}&v2=${v2}`, {
    headers: buildHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to fetch version diff' }));
    throw new Error(error.message || 'Failed to fetch version diff');
  }

  return response.json();
};

/**
 * Request approval for rule
 */
export const requestApproval = async (request: ApprovalRequest): Promise<void> => {
  const response = await fetch(`${API_BASE}/rules/${request.ruleId}/approve`, {
    method: 'POST',
    headers: buildHeaders(),
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to submit approval' }));
    throw new Error(error.message || 'Failed to submit approval');
  }
};

/**
 * Get pending approvals
 */
export const getPendingApprovals = async (): Promise<any[]> => {
  const response = await fetch(`${API_BASE}/approvals/pending`, {
    headers: buildHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to fetch approvals' }));
    throw new Error(error.message || 'Failed to fetch approvals');
  }

  return response.json();
};

/**
 * Rollback to previous rule version
 */
export const rollbackRule = async (ruleId: string, toVersion: number): Promise<Rule> => {
  const response = await fetch(`${API_BASE}/rules/${ruleId}/rollback`, {
    method: 'POST',
    headers: buildHeaders(),
    body: JSON.stringify({ toVersion }),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Rollback failed' }));
    throw new Error(error.message || 'Rollback failed');
  }

  return response.json();
};

/**
 * Get tenant ID from context/auth
 */
function getTenantId(): string {
  // In production, get from auth context or session store
  // For now, try localStorage first, then environment variable, then default
  return (
    localStorage.getItem('tenantId') ||
    process.env.REACT_APP_TENANT_ID ||
    '00000000-0000-0000-0000-000000000001'
  );
}

/**
 * Get authorization token from storage
 */
function getAuthToken(): string | null {
  return localStorage.getItem('authToken') || sessionStorage.getItem('authToken');
}

/**
 * Build common fetch headers with auth and tenant
 */
function buildHeaders(contentType = 'application/json'): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': contentType,
    'X-Tenant-ID': getTenantId(),
  };

  const token = getAuthToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  return headers;
}

/**
 * Get semantic terms for a business object
 */
export const getSemanticTerms = async (businessObject: string): Promise<SemanticTerm[]> => {
  const response = await fetch(`${API_BASE}/semantic-terms?businessObject=${encodeURIComponent(businessObject)}`, {
    headers: buildHeaders(),
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: 'Failed to fetch semantic terms' }));
    throw new Error(error.message || 'Failed to fetch semantic terms');
  }

  return response.json();
};
