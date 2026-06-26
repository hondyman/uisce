import { fetchAPI } from '../api';

export type AccessLevel = 'NONE' | 'READ' | 'WRITE';
export type RuleStatus = 'DRAFT' | 'REVIEW' | 'APPROVED' | 'DEPRECATED';
export type MaskType = 'HIDE' | 'MASK' | 'NONE';

export interface ColumnMask {
  semanticTermId: string;
  maskType: MaskType;
}

export interface AccessRule {
  ruleId: string;
  tenantId: string;
  businessObjectId: string;
  groupDn: string;
  accessLevel: AccessLevel;
  status: RuleStatus;
  rowFilterDsl?: string;
  columnMasks: ColumnMask[];
  scope?: {
    appliesToApis?: boolean;
    appliesToBi?: boolean;
    appliesToAi?: boolean;
  };
  metadata?: Record<string, unknown>;
}

export interface AccessRuleInput extends Omit<AccessRule, 'ruleId' | 'metadata'> {
  metadata?: Record<string, unknown>;
  ruleId?: string;
}

export interface AccessRuleImpact {
  ruleId: string;
  businessObjectId: string;
  semanticTerms: string[];
  apis: string[];
  biArtifacts: string[];
  aiArtifacts: string[];
}

export const accessRulesApi = {
  list: async (params: { businessObjectId?: string; groupDn?: string; status?: RuleStatus } = {}): Promise<AccessRule[]> => {
    const qs = new URLSearchParams();
    if (params.businessObjectId) qs.set('businessObjectId', params.businessObjectId);
    if (params.groupDn) qs.set('groupDn', params.groupDn);
    if (params.status) qs.set('status', params.status);
    const suffix = qs.toString() ? `?${qs.toString()}` : '';
    return fetchAPI(`/security/rules${suffix}`);
  },

  get: async (ruleId: string): Promise<AccessRule> => fetchAPI(`/security/rules/${ruleId}`),

  create: async (input: AccessRuleInput): Promise<AccessRule> =>
    fetchAPI(`/security/rules`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    }),

  update: async (ruleId: string, input: AccessRuleInput): Promise<AccessRule> =>
    fetchAPI(`/security/rules/${ruleId}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    }),

  delete: async (ruleId: string): Promise<void> =>
    fetchAPI(`/security/rules/${ruleId}`, {
      method: 'DELETE',
    }),

  validate: async (body: { rowFilterDsl: string; businessObjectId?: string }): Promise<{ valid: boolean; error?: string; sql?: string }> =>
    fetchAPI(`/security/rules/validate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),

  impact: async (ruleId: string): Promise<AccessRuleImpact> => fetchAPI(`/security/rules/${ruleId}/impact`),
};

