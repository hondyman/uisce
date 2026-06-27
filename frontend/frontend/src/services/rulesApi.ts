import { Rule, RulePreviewResult, RuleSimulationResult, ValidationPreviewResult, ValidationRule, RuleDiff, RuleVersion, RuleLineage } from "../types/rules";

const getHeaders = (tenantId?: string, datasourceId?: string) => {
    const headers: HeadersInit = {
        'Content-Type': 'application/json'
    };
    if (tenantId) {
        headers['X-Tenant-ID'] = tenantId;
    }
    if (datasourceId) {
        headers['X-Tenant-Instance-ID'] = datasourceId;
        headers['X-Tenant-Datasource-ID'] = datasourceId;
    }
    return headers;
};

const getUrl = (path: string, datasourceId?: string) => {
    const url = new URL(path, window.location.origin);
    if (datasourceId) {
        url.searchParams.append('datasource_id', datasourceId);
    }
    return url.toString();
};

export const rulesApi = {
    // Retrieval & Scoping
    fetchBOValidations: async (boId: string, tenantId?: string, datasourceId?: string): Promise<ValidationRule[]> => {
        const url = getUrl(`/api/business-objects/${boId}/validations`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to fetch BO validations");
        return res.json();
    },

    fetchRulesBySemanticTerm: async (termId: string): Promise<Rule[]> => {
        const res = await fetch(`/api/semantic-terms/${termId}/rules`);
        if (!res.ok) throw new Error("Failed to fetch rules for semantic term");
        return res.json();
    },

    // Governance Extensions
    fetchRuleDiff: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string): Promise<RuleDiff> => {
        const url = getUrl(`/api/business-objects/${boId}/validations/${ruleId}/diff`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to fetch rule diff");
        return res.json();
    },

    fetchRuleHistory: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string): Promise<RuleVersion[]> => {
        const url = getUrl(`/api/business-objects/${boId}/validations/${ruleId}/history`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to fetch rule history");
        return res.json();
    },

    fetchRuleLineage: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string): Promise<RuleLineage> => {
        // Lineage is rule-centric, use validation-rules endpoint
        const url = getUrl(`/api/validation-rules/${ruleId}/lineage`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to fetch rule lineage");
        return res.json();
    },

    fetchRuleImpact: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string): Promise<import("../types/rules").ImpactResult> => {
        // New contract: the handler is exposed at /api/rules/{id}/impact
        const url = getUrl(`/api/rules/${ruleId}/impact`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to fetch rule impact");
        return res.json();
    },

    // Create a tenant override of a core rule
    overrideRule: async (coreRuleId: string, tenantId?: string, datasourceId?: string): Promise<Rule> => {
        const url = getUrl(`/api/rules/${coreRuleId}/override`, datasourceId);
        const res = await fetch(url, { method: 'POST', headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error('Failed to override rule');
        return res.json();
    },

    // Preview impact using the test/compile endpoint (useful for unsaved rules)
    previewRuleImpact: async (dsl: string, tenantId?: string, boId?: string): Promise<import("../types/rules").ImpactResult> => {
        const body: any = { dsl };
        if (boId) body.business_object_id = boId;
        const res = await fetch(`/api/rules/test`, {
            method: 'POST',
            headers: getHeaders(tenantId),
            body: JSON.stringify(body),
        });
        if (!res.ok) throw new Error('Failed to preview rule impact');
        const testRes = await res.json();
        // Map RuleTestResponse -> ImpactResult (fields only)
        const fields = (testRes.referenced_fields || []).map((f: any) => ({
            semantic_term_id: f.semantic_term_id,
            bo_field_id: f.bo_field_id,
            business_object_id: f.business_object_id,
            field_path: f.field_path,
        }));
        return {
            rule_id: '',
            fields,
            semantic_terms: [],
            business_objects: [],
            dependent_rules: [],
            overrides: [],
        } as import("../types/rules").ImpactResult;
    },

    fetchRuleTerms: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string): Promise<any> => {
        const url = getUrl(`/api/validation-rules/${ruleId}/terms`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to fetch related terms");
        return res.json();
    },

    // SQL Preview
    fetchValidationPreviewSQL: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string): Promise<ValidationPreviewResult> => {
        const url = getUrl(`/api/business-objects/${boId}/validations/${ruleId}/preview-sql`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId) });
        if (!res.ok) throw new Error("Failed to fetch validation SQL preview");
        return res.json();
    },

    fetchUnsavedValidationPreviewSQL: async (boId: string, expression: string, tenantId?: string, datasourceId?: string): Promise<ValidationPreviewResult> => {
        const url = getUrl(`/api/business-objects/${boId}/validations/preview-sql`, datasourceId);
        const res = await fetch(url, {
            method: "POST",
            headers: getHeaders(tenantId, datasourceId),
            body: JSON.stringify({ expression }),
        });
        if (!res.ok) throw new Error("Failed to fetch validation SQL preview");
        return res.json();
    },

    // Promotion & Approval
    promoteValidationRuleToCore: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string): Promise<void> => {
        const url = getUrl(`/api/business-objects/${boId}/validations/${ruleId}/promote`, datasourceId);
        const res = await fetch(url, { method: "POST", headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to promote validation rule");
    },

    fetchPromotionStatus: async (boId: string, ruleId: string, tenantId?: string, datasourceId?: string): Promise<{ promotionStatus: string; workflowId?: string; lastUpdatedAt: string }> => {
        const url = getUrl(`/api/business-objects/${boId}/validations/${ruleId}/promotion-status`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to fetch promotion status");
        return res.json();
    },

    // Promotion impact summary for a rule (used before promoting a change to core)
    fetchPromotionImpact: async (ruleId: string, tenantId?: string, datasourceId?: string): Promise<{ business_objects: number; fields: number; semantic_terms: number; related_rules: number; overrides: number }> => {
        const url = getUrl(`/api/rules/${ruleId}/promotion-impact`, datasourceId);
        const res = await fetch(url, { headers: getHeaders(tenantId, datasourceId) });
        if (!res.ok) throw new Error("Failed to fetch promotion impact summary");
        return res.json();
    },

    approveRule: async (boId: string, ruleId: string, comment?: string, tenantId?: string, datasourceId?: string): Promise<void> => {
        const url = getUrl(`/api/business-objects/${boId}/validations/${ruleId}/approve`, datasourceId);
        const res = await fetch(url, {
            method: "POST",
            headers: getHeaders(tenantId, datasourceId),
            body: JSON.stringify({ comment })
        });
        if (!res.ok) throw new Error("Failed to approve rule");
    },

    denyRule: async (boId: string, ruleId: string, comment?: string, tenantId?: string, datasourceId?: string): Promise<void> => {
        const url = getUrl(`/api/business-objects/${boId}/validations/${ruleId}/deny`, datasourceId);
        const res = await fetch(url, {
            method: "POST",
            headers: getHeaders(tenantId, datasourceId),
            body: JSON.stringify({ comment })
        });
        if (!res.ok) throw new Error("Failed to deny rule");
    },

    // Simulation
    simulateRule: async (
        entity: string,
        expression: string,
        sampleSize: number = 50,
        filter: Record<string, any> = {},
        tenantId?: string
    ): Promise<RuleSimulationResult> => {
        const res = await fetch(`/api/validation-rules/simulate`, {
            method: "POST",
            headers: getHeaders(tenantId),
            body: JSON.stringify({ entity, expression, sampleSize, filter }),
        });
        if (!res.ok) throw new Error("Failed to simulate rule");
        return res.json();
    },

    // Legacy / Generic (to be deprecated or used for global views)
    getRules: async (): Promise<Rule[]> => {
        const res = await fetch("/api/validation-rules");
        if (!res.ok) throw new Error("Failed to fetch rules");
        return res.json();
    },

    getRule: async (id: string): Promise<Rule> => {
        const res = await fetch(`/api/validation-rules/${id}`);
        if (!res.ok) throw new Error("Failed to fetch rule");
        return res.json();
    },

    testRule: async (dsl: string, tenantId: string, boId: string, sample: any): Promise<import("../types/rules").RuleTestResponse> => {
        const res = await fetch(`/api/rules/test`, {
            method: 'POST',
            headers: getHeaders(tenantId),
            body: JSON.stringify({ dsl, tenant_id: tenantId, business_object_id: boId, sample }),
        });
        if (!res.ok) {
            const text = await res.text();
            throw new Error(`Failed to test rule: ${res.status} ${text}`);
        }
        return res.json();
    }
};

// Named exports for convenience
// Rule Engine base for specialized routes
const RULES_API_BASE = '/api/validation-rules';

export const fetchBOValidations = async (bpName?: string, stepName?: string, tenantId?: string) => {
    try {
        const params = new URLSearchParams();
        if (bpName) params.append('bp_name', bpName);
        if (stepName) params.append('step_name', stepName);

        const url = `${RULES_API_BASE}/?${params.toString()}`;
        const response = await fetch(url, { method: 'GET', headers: getHeaders(tenantId) });
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(`Failed to fetch BO validations: ${response.status} ${errorText}`);
        }
        return await response.json();
    } catch (error) {
        console.error('Error in fetchBOValidations:', error);
        throw error;
    }
};

export const getRules = rulesApi.getRules;
export const getRule = rulesApi.getRule;

// Flexible simulateRule wrapper:
// - If called with (ruleId: string, testData: any) -> POST /api/rules/evaluate
// - Otherwise delegates to the entity/expression simulator (existing implementation)
export const simulateRule = async (
    a: string,
    b?: any,
    c?: number,
    d?: Record<string, any>,
    tenantId?: string
): Promise<RuleSimulationResult> => {
    // Heuristic: if second arg is an object/array treat as test data for rule evaluation
    if (b && (Array.isArray(b) || typeof b === 'object')) {
        const ruleId = a;
        const testData = b;
        const res = await fetch(`${RULES_API_BASE}/evaluate`, {
            method: 'POST',
            headers: getHeaders(tenantId),
            body: JSON.stringify({ rule_id: ruleId, test_data: testData }),
        });
        if (!res.ok) {
            const text = await res.text();
            throw new Error(`Failed to simulate rule by id: ${res.status} ${text}`);
        }
        return res.json();
    }

    // Fallback to original signature: (entity, expression, sampleSize, filter, tenantId)
    return rulesApi.simulateRule(a, b, c, d, tenantId);
};

export const previewRule = async (rule: Partial<Rule>, tenantId?: string, datasourceId?: string): Promise<RulePreviewResult> => {
    // Validate/preview via the Rule Engine service
    const url = getUrl('/api/rules/validate', datasourceId);
    const res = await fetch(url, {
        method: 'POST',
        headers: getHeaders(tenantId, datasourceId),
        body: JSON.stringify(rule),
    });
    if (!res.ok) {
        const text = await res.text();
        throw new Error(`Failed to preview rule: ${res.status} ${text}`);
    }
    return res.json();
};

// Creation helpers (convenience wrappers around backend endpoints)
export const createRule = async (rule: Partial<Rule>, tenantId?: string, datasourceId?: string): Promise<Rule> => {
    const url = getUrl('/api/rules', datasourceId);
    const res = await fetch(url, { method: 'POST', headers: getHeaders(tenantId, datasourceId), body: JSON.stringify(rule) });
    if (!res.ok) throw new Error('Failed to create rule');
    return res.json();
};

export const createBusinessObjectRule = async (boId: string, rule: Partial<Rule>, tenantId?: string, datasourceId?: string): Promise<Rule> => {
    const url = getUrl(`/api/business-objects/${boId}/validations`, datasourceId);
    const res = await fetch(url, { method: 'POST', headers: getHeaders(tenantId, datasourceId), body: JSON.stringify(rule) });
    if (!res.ok) throw new Error('Failed to create BO rule');
    return res.json();
};

export const createSemanticRule = async (termId: string, rule: Partial<Rule>, tenantId?: string): Promise<Rule> => {
    const res = await fetch(`/api/semantic-terms/${termId}/rules`, { method: 'POST', headers: getHeaders(tenantId), body: JSON.stringify(rule) });
    if (!res.ok) throw new Error('Failed to create semantic rule');
    return res.json();
};

export const createFieldRule = async (boId: string, fieldPath: string, rule: Partial<Rule>, tenantId?: string, datasourceId?: string): Promise<Rule> => {
    const body = { ...rule, target_field: fieldPath };
    const url = getUrl(`/api/business-objects/${boId}/validations`, datasourceId);
    const res = await fetch(url, { method: 'POST', headers: getHeaders(tenantId, datasourceId), body: JSON.stringify(body) });
    if (!res.ok) throw new Error('Failed to create field rule');
    return res.json();
};
