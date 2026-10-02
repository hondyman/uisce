/**
 * Validation Rule & Rulefabric API Client
 * Targets the canonical backend endpoints under /api/validation-rule-nodes
 */

export interface ValidationRuleDescriptor {
  id: string;
  rule_key?: string;
  tenant_id: string;
  bo_name: string;
  name: string;
  description?: string;
  severity: 'BLOCK' | 'WARN';
  timing: 'pre_write' | 'reconcile';
  category?: string;
  governance_status?: string;
  domain?: string;
  is_active: boolean;
  origin?: 'core' | 'custom';
  rule_ast: any;
  binding_ids?: string[];
  created_at?: string;
  updated_at?: string;
}

export interface RuleDiffSummary {
  rule_key: string;
  name: string;
  bo_name: string;
  action: 'CREATE' | 'UPDATE' | 'NO_OP' | 'DELETE';
  changes?: string[];
  rule_errors?: string[];
}

export interface RuleImportErrorDetail {
  rule_key: string;
  code: string;
  reason: string;
}

export interface RuleImportReport {
  success: boolean;
  dry_run: boolean;
  target_tenant_id: string;
  total_rules: number;
  created: string[];
  updated: string[];
  deleted: string[];
  unchanged: string[];
  diff_summary: RuleDiffSummary[];
  errors: RuleImportErrorDetail[];
  warnings: string[];
  imported_at: string;
  duration_ms: number;
}

export interface ViolationRecordItem {
  id: string;
  tenant_id: string;
  rule_id: string;
  rule_name: string;
  bo_key: string;
  severity: string;
  record_id: string;
  message: string;
  context: any;
  write_blocked: boolean;
  rule_error: boolean;
  created_at: string;
}

export interface RecordEvaluationResponse {
  valid: boolean;
  blocked: boolean;
  evaluated_rules_count: number;
  snapshot_id?: string;
  violations: Array<{
    rule_id: string;
    rule_name: string;
    bo_name: string;
    severity: string;
    record_id: string;
    message: string;
    error: string;
  }>;
  rule_errors: Array<{
    rule_id: string;
    rule_key: string;
    rule_name: string;
    message: string;
  }>;
  error?: string;
  missing_context_fields?: string[];
  hint?: string;
}

export interface BatchEvaluationResponse {
  summary: {
    snapshot_id: string;
    total_records: number;
    valid_count: number;
    invalid_count: number;
    rule_error_count: number;
    error_count: number;
    duration_ms: number;
  };
  records: RecordEvaluationResponse[];
  errors: Array<{
    index: number;
    code: string;
    message: string;
  }>;
}

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {});
  if (!headers.has('Content-Type') && !(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json');
  }

  const token = localStorage.getItem('token');
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const activeTenant = localStorage.getItem('active_tenant_id');
  if (activeTenant && !headers.has('X-Tenant-ID')) {
    headers.set('X-Tenant-ID', activeTenant);
  }

  const res = await fetch(url, { ...options, headers });
  if (!res.ok) {
    let errMsg = `Request failed: ${res.status} ${res.statusText}`;
    try {
      const errBody = await res.json();
      errMsg = errBody.message || errBody.error || errMsg;
    } catch {
      try {
        const text = await res.text();
        if (text) errMsg = text;
      } catch {}
    }
    throw new Error(errMsg);
  }
  return res.json();
}

export const validationRulesApi = {
  listRules: async (boName?: string, domain?: string): Promise<ValidationRuleDescriptor[]> => {
    const params = new URLSearchParams();
    if (boName) params.set('bo_name', boName);
    if (domain) params.set('domain', domain);
    const qs = params.toString();
    return request<ValidationRuleDescriptor[]>(`/api/validation-rule-nodes${qs ? `?${qs}` : ''}`);
  },

  getRule: async (id: string): Promise<ValidationRuleDescriptor> => {
    return request<ValidationRuleDescriptor>(`/api/validation-rule-nodes/${id}`);
  },

  upsertRule: async (rule: Partial<ValidationRuleDescriptor>): Promise<ValidationRuleDescriptor> => {
    return request<ValidationRuleDescriptor>('/api/validation-rule-nodes', {
      method: 'POST',
      body: JSON.stringify(rule),
    });
  },

  setActive: async (id: string, active: boolean): Promise<{ success: boolean; is_active: boolean }> => {
    return request<{ success: boolean; is_active: boolean }>(`/api/validation-rule-nodes/${id}/active`, {
      method: 'PATCH',
      body: JSON.stringify({ is_active: active }),
    });
  },

  exportBundle: async (boName?: string, domain?: string, origin?: string, format: 'json' | 'yaml' = 'json'): Promise<any> => {
    const params = new URLSearchParams();
    if (boName) params.set('bo_name', boName);
    if (domain) params.set('domain', domain);
    if (origin) params.set('origin', origin);
    params.set('format', format);

    const token = localStorage.getItem('token');
    const headers: Record<string, string> = {};
    if (token) headers['Authorization'] = `Bearer ${token}`;
    if (format === 'yaml') headers['Accept'] = 'application/x-yaml';

    const res = await fetch(`/api/validation-rule-nodes/export?${params.toString()}`, { headers });
    if (!res.ok) throw new Error(`Export failed: ${res.statusText}`);
    return format === 'yaml' ? res.text() : res.json();
  },

  preflightImport: async (bundle: any, overwritePolicy = 'fail', preserveStatus = true, prune = false): Promise<RuleImportReport> => {
    const params = new URLSearchParams({
      overwrite_policy: overwritePolicy,
      preserve_status: String(preserveStatus),
      prune_missing: String(prune),
    });
    return request<RuleImportReport>(`/api/validation-rule-nodes/preflight?${params.toString()}`, {
      method: 'POST',
      body: typeof bundle === 'string' ? bundle : JSON.stringify({ bundle }),
      headers: typeof bundle === 'string' ? { 'Content-Type': 'application/x-yaml' } : undefined,
    });
  },

  importBundle: async (bundle: any, overwritePolicy = 'fail', preserveStatus = true, prune = false, idemKey?: string): Promise<RuleImportReport> => {
    const params = new URLSearchParams({
      overwrite_policy: overwritePolicy,
      preserve_status: String(preserveStatus),
      prune_missing: String(prune),
    });
    const headers: Record<string, string> = {};
    if (idemKey) headers['X-Idempotency-Key'] = idemKey;
    if (typeof bundle === 'string') headers['Content-Type'] = 'application/x-yaml';

    return request<RuleImportReport>(`/api/validation-rule-nodes/import?${params.toString()}`, {
      method: 'POST',
      body: typeof bundle === 'string' ? bundle : JSON.stringify({ bundle }),
      headers,
    });
  },

  evaluateRecord: async (boName: string, record: Record<string, any>, domain?: string, timing?: string): Promise<RecordEvaluationResponse> => {
    return request<RecordEvaluationResponse>('/api/validation-rule-nodes/evaluate-record', {
      method: 'POST',
      body: JSON.stringify({ bo_name: boName, record, domain, timing }),
    });
  },

  evaluateBatch: async (boName: string, records: Array<Record<string, any>>, domain?: string, timing?: string): Promise<BatchEvaluationResponse> => {
    return request<BatchEvaluationResponse>('/api/validation-rule-nodes/evaluate-batch', {
      method: 'POST',
      body: JSON.stringify({ bo_name: boName, records, domain, timing }),
    });
  },

  listViolations: async (boKey?: string, limit = 100): Promise<ViolationRecordItem[]> => {
    const params = new URLSearchParams();
    if (boKey) params.set('bo_key', boKey);
    params.set('limit', String(limit));
    return request<ViolationRecordItem[]>(`/api/validation-rule-nodes/violations?${params.toString()}`);
  },
};
