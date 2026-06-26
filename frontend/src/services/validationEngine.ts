/**
 * Validation Engine Service
 * TypeScript client for Wealth Management Validation Engine
 * Handles validation rule execution, results caching, and UI interaction
 */

import { fetchAPI } from '../api';
import { devWarn, devError, devLog } from '../utils/devLogger';

// Local narrowers/helpers to avoid `(xxx as any)` casts around errors and records
const asRecord = (v: unknown): Record<string, unknown> => (v && typeof v === 'object' ? (v as Record<string, unknown>) : {});
const getErrorMessage = (err: unknown): string => {
  if (!err) return '';
  if (err instanceof Error) return err.message;
  const r = asRecord(err);
  const m = r['message'] ?? r['msg'] ?? r['error'];
  return m == null ? String(err) : String(m);
};

// ============================================================================
// TYPE DEFINITIONS - matching backend services/validation_engine.go
// ============================================================================

export enum RuleSeverity {
  WARNING = 'WARNING',
  BLOCK = 'BLOCK',
  INFO = 'INFO',
}

export enum RuleScope {
  INDIVIDUAL_ACCOUNT = 'INDIVIDUAL_ACCOUNT',
  JOINT_ACCOUNT = 'JOINT_ACCOUNT',
  TRUST_ACCOUNT = 'TRUST_ACCOUNT',
  IRA_ACCOUNT = 'IRA_ACCOUNT',
  ALL_ACCOUNTS = 'ALL_ACCOUNTS',
}

export enum RuleFrequency {
  CONTINUOUS = 'CONTINUOUS',
  DAILY = 'DAILY',
  WEEKLY = 'WEEKLY',
  ON_TRADE = 'ON_TRADE',
  ON_REBALANCE = 'ON_REBALANCE',
}

export interface ValidationRule {
  id: string;
  name: string;
  description: string;
  ruleType: string;
  scope: string[];
  severity: RuleSeverity;
  isActive: boolean;
  effectiveFrom: Date;
  effectiveTo?: Date;
  frequency: RuleFrequency;
  evaluationOrder: number;
  overrideConditions?: string[];
  requiredAuthority?: string;
  parameters: Record<string, any>;
  createdAt: Date;
  updatedAt: Date;
  tenantId: string;
  datasourceId: string;
}

export interface ValidationContext {
  accountId: string;
  accountType: string;
  clientId: string;
  portfolioData?: Record<string, any>;
  transactionData?: Record<string, any>;
  clientProfile?: Record<string, any>;
  timestamp: Date;
  userId?: string;
  overrideAuthorization?: string;
  tenantId: string;
  datasourceId: string;
}

export interface ValidationResult {
  ruleId: string;
  ruleName: string;
  passed: boolean;
  severity: RuleSeverity;
  message: string;
  details?: Record<string, any>;
  timestamp: Date;
  requiresOverride?: boolean;
  allowedOverrideAuthority?: string;
  failedValue?: any;
  threshold?: any;
}

export interface ValidationExecutionResult {
  contextId: string;
  accountId: string;
  passed: boolean;
  timestamp: Date;
  results: ValidationResult[];
  blockedRules: ValidationResult[];
  warningRules: ValidationResult[];
  infoRules: ValidationResult[];
  executionTimeMs: number;
  tenantId: string;
  datasourceId: string;
}

// ============================================================================
// VALIDATION ENGINE SERVICE
// ============================================================================

export class InvestmentValidationEngine {
  private tenantId: string;
  private datasourceId: string;
  private resultsCache: Map<string, ValidationExecutionResult> = new Map();
  private cacheDuration: number = 5 * 60 * 1000; // 5 minutes

  constructor(tenantId: string, datasourceId: string) {
    this.tenantId = tenantId;
    this.datasourceId = datasourceId;
  }

  /**
   * Execute validations against an account/portfolio
   */
  async executeValidations(
    context: ValidationContext
  ): Promise<ValidationExecutionResult> {
    const cacheKey = `${context.accountId}-${context.timestamp.getTime()}`;

    // Check cache
    const cached = this.resultsCache.get(cacheKey);
    if (cached && Date.now() - cached.timestamp.getTime() < this.cacheDuration) {
      return cached;
    }

    try {
      const response: unknown = await fetchAPI('/validate', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': this.tenantId,
          'X-Tenant-Datasource-ID': this.datasourceId,
        },
        body: JSON.stringify({
          ...context,
          tenantId: this.tenantId,
          datasourceId: this.datasourceId,
        }),
      });

      const resp = response as Record<string, unknown>;
      const mapResults = (arr: unknown): ValidationResult[] => {
        if (!Array.isArray(arr)) return [];
        return (arr as unknown[]).map((it) => {
          const r = it as Record<string, unknown>;
          return {
            ruleId: String(r.ruleId ?? r.ruleId ?? ''),
            ruleName: String(r.ruleName ?? r.ruleName ?? ''),
            passed: Boolean(r.passed),
            severity: (String(r.severity || '') as RuleSeverity) || RuleSeverity.INFO,
            message: String(r.message ?? ''),
            details: (r.details as Record<string, any>) ?? undefined,
            timestamp: r.timestamp ? new Date(String(r.timestamp)) : new Date(),
            requiresOverride: Boolean(r.requiresOverride),
            allowedOverrideAuthority: r.allowedOverrideAuthority ? String(r.allowedOverrideAuthority) : undefined,
            failedValue: r.failedValue,
            threshold: r.threshold,
          } as ValidationResult;
        });
      };

      const result: ValidationExecutionResult = {
        contextId: String(resp.contextId ?? ''),
        accountId: String(resp.accountId ?? ''),
        passed: Boolean(resp.passed),
        timestamp: resp.timestamp ? new Date(String(resp.timestamp)) : new Date(),
        results: mapResults(resp.results),
        blockedRules: mapResults(resp.blockedRules),
        warningRules: mapResults(resp.warningRules),
        infoRules: mapResults(resp.infoRules),
        executionTimeMs: Number(resp.executionTimeMs) || 0,
        tenantId: String(resp.tenantId ?? ''),
        datasourceId: String(resp.datasourceId ?? ''),
      };

      // Cache result
      this.resultsCache.set(cacheKey, result);

      return result;
    } catch (error) {
      devError('Validation execution failed:', error);
      throw error;
    }
  }

  /**
   * Get all validation rules for the tenant/datasource
   */
  async getRules(filters?: {
    ruleType?: string;
    scope?: RuleScope;
  }): Promise<ValidationRule[]> {
    try {
      const params = new URLSearchParams();
      if (filters?.ruleType) {
        params.append('ruleType', filters.ruleType);
      }
      if (filters?.scope) {
        params.append('scope', filters.scope);
      }

      const response: any = await fetchAPI(`/validation-rules?${params.toString()}`, {
        method: 'GET',
        headers: {
          'X-Tenant-ID': this.tenantId,
          'X-Tenant-Datasource-ID': this.datasourceId,
        },
      });

      // Backend returns response with rules property
      return (response.rules || []).map((r: any) => ({
        id: r.id,
        name: r.rule_name,
        description: r.description,
        ruleType: r.rule_type,
        scope: [r.target_entity], // Convert target_entity to scope array
        severity: (String(r.severity || '') as RuleSeverity) || RuleSeverity.INFO,
        isActive: r.is_active,
        effectiveFrom: new Date(r.created_at),
        effectiveTo: undefined, // Backend doesn't have effectiveTo
        frequency: RuleFrequency.CONTINUOUS, // Default value
        evaluationOrder: 0, // Default value
        parameters: r.condition_json || {},
        createdAt: new Date(r.created_at),
        updatedAt: new Date(r.updated_at),
        tenantId: this.tenantId,
        datasourceId: this.datasourceId,
      }));
    } catch (error) {
      devError('Failed to fetch validation rules:', error);
      return [];
    }
  }

  /**
   * Helper: quick UUID format check
   */
  private isUUID(id: string): boolean {
    if (!id) return false;
    const re = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    return re.test(id);
  }

  /**
   * Helper: resolve a slug/name to the canonical rule UUID by searching the list endpoint.
   * If the provided id already looks like a UUID it is returned as-is.
   */
  private async resolveRuleId(maybeId: string): Promise<string> {
    if (this.isUUID(maybeId)) return maybeId;

    // Try searching by name/slug using the list endpoint's search param
    try {
      const params = new URLSearchParams();
      params.append('search', maybeId);
      // include tenant/datasource so the list is scoped correctly (tenant shim also adds these, but include for clarity)
      params.append('tenant_id', this.tenantId);
      params.append('datasource_id', this.datasourceId);

      const response: any = await fetchAPI(`/validation-rules?${params.toString()}`, {
        method: 'GET',
        headers: {
          'X-Tenant-ID': this.tenantId,
          'X-Tenant-Datasource-ID': this.datasourceId,
        },
      });

      const candidates = response.rules || [];
      if (candidates.length === 0) {
        throw new Error(`No validation rule found matching '${maybeId}'`);
      }

      // If exactly one candidate, use it. Otherwise, try to find an exact match on rule_name or name.
      if (candidates.length === 1) return candidates[0].id;

      for (const c of candidates) {
        if (!c) continue;
        if (c.id === maybeId) return c.id;
        if (c.rule_name === maybeId) return c.id;
        if (c.name === maybeId) return c.id;
      }

      // Fall back to first candidate if no exact match
      return candidates[0].id;
    } catch (err) {
      const msg = getErrorMessage(err);
      throw new Error(`Failed to resolve rule id '${maybeId}': ${msg}`);
    }
  }

  /**
   * Public helper to attempt to find a rule id by slug or name. Returns undefined when not found.
   */
  public async findRuleIdByName(maybeId: string): Promise<string | undefined> {
    try {
      return await this.resolveRuleId(maybeId);
    } catch (e) {
      return undefined;
    }
  }

  /**
   * Normalize frontend rule type values to backend-expected snake_case types.
   * Returns undefined if no value provided.
   * Maps all WEALTH_VALIDATION_RULES domain types to backend-accepted types.
   */
  private normalizeRuleType(rt?: string): string | undefined {
    if (!rt) return undefined;
    const v = String(rt).trim().toLowerCase();
    if (!v) return undefined;
    const valid = new Set([
      'field_format',
      'cardinality',
      'uniqueness',
      'referential_integrity',
      'business_logic',
    ]);
    if (valid.has(v)) return v;
    // Minimal legacy mappings for compatibility
    const compact = v.replace(/[_\-\s]/g, '');
    switch (compact) {
      case 'fieldformat':
        return 'field_format';
      case 'referentialintegrity':
        return 'referential_integrity';
      case 'businesslogic':
      case 'validation':
        return 'business_logic';
      default:
        // Log unmapped types for debugging but default to business_logic to avoid 400 errors
        devWarn(`[validationEngine] Invalid rule type: "${v}" - defaulting to business_logic`);
        return 'business_logic';
    }
  }

  /**
   * Create a new validation rule
   */
  async createRule(rule: Omit<ValidationRule, 'createdAt' | 'updatedAt'>): Promise<ValidationRule> {
    try {
      // Normalize frontend camelCase rule object to backend snake_case shape
      const toSnakePayload = (r: any) => {
        const payload: any = {};
        // Basic mapping of commonly used fields
        if (r.id) payload.id = r.id;
        if (r.name) payload.rule_name = r.name;
        // Normalize rule type to backend-expected snake_case
        const ruleTypeInput = r.rule_type || r.ruleType || r.RuleType || r.RuleTypeCC;
        devLog(`[toSnakePayload] Normalizing rule_type for "${r.name}":`, { input: ruleTypeInput });
        const normalizedRt = this.normalizeRuleType(ruleTypeInput);
        if (normalizedRt) payload.rule_type = normalizedRt;
        else devWarn(`[toSnakePayload] No valid rule_type for "${r.name}"`);
        if (r.description) payload.description = r.description;
        if (typeof r.isActive !== 'undefined') payload.is_active = r.isActive;
        if (r.effectiveFrom) payload.effective_from = r.effectiveFrom;
        if (r.effectiveTo) payload.effective_to = r.effectiveTo;
        if (r.frequency) payload.frequency = r.frequency;
        if (typeof r.evaluationOrder !== 'undefined') payload.evaluation_order = r.evaluationOrder;
        if (r.overrideConditions) payload.override_conditions = r.overrideConditions;
        if (r.requiredAuthority) payload.required_authority = r.requiredAuthority;
        if (typeof r.is_core !== 'undefined') payload.is_core = r.is_core;
        // Map parameters to condition_json for backend compatibility
        if (r.parameters) payload.condition_json = { ...r.parameters };
        if (r.condition_json) payload.condition_json = { ...r.condition_json };
        // Map scope/scope array -> target_entities and ensure legacy single target_entity
        if (r.scope) payload.target_entities = r.scope;
        if (r.scopes && !payload.target_entities) payload.target_entities = r.scopes;
        if (r.target_entities) payload.target_entities = r.target_entities;
        if (r.target_entity) payload.target_entity = r.target_entity;
        if (!payload.target_entity && Array.isArray(payload.target_entities) && payload.target_entities.length > 0) {
          payload.target_entity = payload.target_entities[0];
        }
        // Also include camelCase alias for targetEntity for maximum compatibility
        if (payload.target_entity) payload.targetEntity = payload.target_entity;
        // Normalize severity
        if (r.severity) {
          const severityMap: Record<string, string> = {
            BLOCK: 'error',
            WARNING: 'warning',
            INFO: 'info',
          };
          payload.severity = severityMap[r.severity.toUpperCase()] || r.severity.toLowerCase();
        }
        // Attach tenant/datasource metadata expected by the backend
        payload.tenant_id = this.tenantId;
        payload.datasource_id = this.datasourceId;
        devLog(`[toSnakePayload] Final payload for "${r.name}":`, payload);
        return payload;
      };

      const payload = toSnakePayload(rule);
      devLog('[validationEngine.createRule] Sending payload:', { ruleName: String(rule.name ?? ''), ruleType: payload.rule_type, payload });

      const response: any = await fetchAPI('/validation-rules', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': this.tenantId,
          'X-Tenant-Datasource-ID': this.datasourceId,
        },
        body: JSON.stringify(payload),
      });

      devLog('[validationEngine.createRule] Success:', { ruleName: String(rule.name ?? ''), ruleId: response.id });

      return {
        id: response.id,
        name: response.rule_name,
        description: response.description,
        ruleType: response.rule_type,
        scope: [response.target_entity],
        severity: (String(response.severity || '') as RuleSeverity) || RuleSeverity.INFO,
        isActive: response.is_active,
        effectiveFrom: new Date(response.created_at),
        effectiveTo: undefined,
        frequency: RuleFrequency.CONTINUOUS,
        evaluationOrder: 0,
        parameters: {},
        createdAt: new Date(response.created_at),
        updatedAt: new Date(response.created_at),
        tenantId: this.tenantId,
        datasourceId: this.datasourceId,
      };
    } catch (error) {
      devError('Failed to create validation rule:', error);
      throw error;
    }
  }

  /**
   * Update an existing validation rule
   */
  async updateRule(ruleId: string, updates: Partial<ValidationRule>): Promise<ValidationRule> {
    try {
      // Ensure we use the canonical UUID for the rule - resolve slugs/names if needed
      const resolvedId = await this.resolveRuleId(ruleId);

      // Map updates to backend-expected snake_case shape where applicable
      const toSnakeUpdates = (u: any) => {
        const out: any = { ...u };
        const ruleTypeInput = u.rule_type || u.ruleType || u.RuleType || u.RuleTypeCC;
        devLog(`[toSnakeUpdates] Normalizing rule_type:`, { input: ruleTypeInput });
        if (typeof ruleTypeInput !== 'undefined') {
          out.rule_type = this.normalizeRuleType(ruleTypeInput) || ruleTypeInput;
          delete out.ruleType;
          delete out.rule_type;
        }
        if (typeof out.name !== 'undefined') {
          out.rule_name = out.name;
          delete out.name;
        }
        if (typeof out.isActive !== 'undefined') {
          out.is_active = out.isActive;
          delete out.isActive;
        }
        if (typeof out.effectiveFrom !== 'undefined') {
          out.effective_from = out.effectiveFrom;
          delete out.effectiveFrom;
        }
        if (typeof out.effectiveTo !== 'undefined') {
          out.effective_to = out.effectiveTo;
          delete out.effectiveTo;
        }
        if (typeof out.evaluationOrder !== 'undefined') {
          out.evaluation_order = out.evaluationOrder;
          delete out.evaluationOrder;
        }
        if (typeof out.requiredAuthority !== 'undefined') {
          out.required_authority = out.requiredAuthority;
          delete out.requiredAuthority;
        }
        if (typeof out.is_core !== 'undefined') {
          out.is_core = out.is_core;
        }
        // Map parameters to condition_json for backend compatibility
        if (out.parameters) out.condition_json = { ...out.parameters };
        if (out.condition_json) out.condition_json = { ...out.condition_json };
        // Map scope/scope array -> target_entities
        if (out.scope) out.target_entities = out.scope;
        if (out.scopes && !out.target_entities) out.target_entities = out.scopes;
        if (out.target_entity) out.target_entity = out.target_entity;
        if (!out.target_entity && Array.isArray(out.target_entities) && out.target_entities.length > 0) {
          out.target_entity = out.target_entities[0];
        }
        // Normalize severity
        if (out.severity) {
          const severityMap: Record<string, string> = {
            BLOCK: 'error',
            WARNING: 'warning',
            INFO: 'info',
          };
          out.severity = severityMap[out.severity.toUpperCase()] || out.severity.toLowerCase();
        }
        // Ensure tenant/datasource ids present for backend routing/validation
        out.tenant_id = this.tenantId;
        out.datasource_id = this.datasourceId;
        devLog(`[toSnakeUpdates] Final updates payload:`, out);
        return out;
      };

      const response: any = await fetchAPI(`/validation-rules/${resolvedId}`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': this.tenantId,
          'X-Tenant-Datasource-ID': this.datasourceId,
        },
        body: JSON.stringify(toSnakeUpdates(updates)),
      });

      return {
        id: response.id,
        name: response.name,
        description: response.description,
        ruleType: response.ruleType,
        scope: response.scope,
        severity: response.severity,
        isActive: response.isActive,
        effectiveFrom: new Date(response.effectiveFrom),
        effectiveTo: response.effectiveTo ? new Date(response.effectiveTo) : undefined,
        frequency: response.frequency,
        evaluationOrder: response.evaluationOrder,
        overrideConditions: response.overrideConditions,
        requiredAuthority: response.requiredAuthority,
        parameters: response.parameters,
        createdAt: new Date(response.createdAt),
        updatedAt: new Date(response.updatedAt),
        tenantId: response.tenantId,
        datasourceId: response.datasourceId,
      };
    } catch (error) {
      devError('Failed to update validation rule:', error);
      throw error;
    }
  }

  /**
   * Delete a validation rule
   */
  async deleteRule(ruleId: string): Promise<void> {
    try {
      await fetchAPI(`/validation-rules/${ruleId}`, {
        method: 'DELETE',
        headers: {
          'X-Tenant-ID': this.tenantId,
          'X-Tenant-Datasource-ID': this.datasourceId,
        },
      });
    } catch (error) {
      devError('Failed to delete validation rule:', error);
      throw error;
    }
  }

  /**
   * Get validation history for an account
   */
  async getValidationHistory(
    accountId: string,
    from?: Date,
    to?: Date
  ): Promise<ValidationExecutionResult[]> {
    try {
      const params = new URLSearchParams();
      if (from) {
        params.append('from', from.toISOString());
      }
      if (to) {
        params.append('to', to.toISOString());
      }

      const response: any = await fetchAPI(`/validation-results/${accountId}?${params.toString()}`, {
        method: 'GET',
        headers: {
          'X-Tenant-ID': this.tenantId,
          'X-Tenant-Datasource-ID': this.datasourceId,
        },
      });

      return (response.results || []).map((r: any) => ({
        contextId: r.contextId || '',
        accountId: r.accountId,
        passed: r.passed,
        timestamp: new Date(r.timestamp),
        results: (r.results || []).map((res: any) => ({
          ...res,
          timestamp: new Date(res.timestamp),
        })),
        blockedRules: (r.blockedRules || []).map((bl: any) => ({
          ...bl,
          timestamp: new Date(bl.timestamp),
        })),
        warningRules: (r.warningRules || []).map((w: any) => ({
          ...w,
          timestamp: new Date(w.timestamp),
        })),
        infoRules: (r.infoRules || []).map((i: any) => ({
          ...i,
          timestamp: new Date(i.timestamp),
        })),
        executionTimeMs: r.executionTimeMs || 0,
        tenantId: r.tenantId,
        datasourceId: r.datasourceId,
      }));
    } catch (error) {
      devError('Failed to fetch validation history:', error);
      return [];
    }
  }

  /**
   * Clear results cache
   */
  clearCache(): void {
    this.resultsCache.clear();
  }

  /**
   * Get cache statistics
   */
  getCacheStats(): { size: number; entries: string[] } {
    return {
      size: this.resultsCache.size,
      entries: Array.from(this.resultsCache.keys()),
    };
  }
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

/**
 * Determine if validation should block an action
 */
export function shouldBlock(result: ValidationExecutionResult): boolean {
  return !result.passed && result.blockedRules.length > 0;
}

/**
 * Get severity color for UI display
 */
export function getSeverityColor(severity: RuleSeverity): string {
  switch (severity) {
    case RuleSeverity.BLOCK:
      return 'text-red-600 dark:text-red-400';
    case RuleSeverity.WARNING:
      return 'text-yellow-600 dark:text-yellow-400';
    case RuleSeverity.INFO:
      return 'text-blue-600 dark:text-blue-400';
    default:
      return 'text-gray-600 dark:text-gray-400';
  }
}

/**
 * Get severity badge color
 */
export function getSeverityBadgeColor(severity: RuleSeverity): string {
  switch (severity) {
    case RuleSeverity.BLOCK:
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300';
    case RuleSeverity.WARNING:
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300';
    case RuleSeverity.INFO:
      return 'bg-blue-100 text-blue-800 dark:bg-blue-900/30 dark:text-blue-300';
    default:
      return 'bg-gray-100 text-gray-800 dark:bg-gray-900/30 dark:text-gray-300';
  }
}

/**
 * Get severity icon
 */
export function getSeverityIcon(severity: RuleSeverity): string {
  switch (severity) {
    case RuleSeverity.BLOCK:
      return '🚫';
    case RuleSeverity.WARNING:
      return '⚠️';
    case RuleSeverity.INFO:
      return 'ℹ️';
    default:
      return '•';
  }
}

/**
 * Format validation result message for display
 */
export function formatResultMessage(result: ValidationResult): string {
  if (result.failedValue !== undefined && result.threshold !== undefined) {
    return `${result.message} (Value: ${result.failedValue}, Threshold: ${result.threshold})`;
  }
  return result.message;
}

/**
 * Group results by severity
 */
export function groupResultsBySeverity(results: ValidationResult[]): {
  blocked: ValidationResult[];
  warnings: ValidationResult[];
  info: ValidationResult[];
} {
  return {
    blocked: results.filter((r) => r.severity === RuleSeverity.BLOCK),
    warnings: results.filter((r) => r.severity === RuleSeverity.WARNING),
    info: results.filter((r) => r.severity === RuleSeverity.INFO),
  };
}

/**
 * Get compliance status badge
 */
export function getComplianceStatus(result: ValidationExecutionResult): {
  status: 'pass' | 'warn' | 'fail';
  label: string;
  color: string;
} {
  if (result.passed) {
    return {
      status: 'pass',
      label: 'Compliant',
      color: 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-300',
    };
  }

  if (result.blockedRules.length > 0) {
    return {
      status: 'fail',
      label: `${result.blockedRules.length} Blocker(s)`,
      color: 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-300',
    };
  }

  return {
    status: 'warn',
    label: `${result.warningRules.length} Warning(s)`,
    color: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300',
  };
}

/**
 * Create sample validation context for testing
 */
export function createSampleValidationContext(
  accountId: string,
  accountType: string = 'INDIVIDUAL_ACCOUNT',
  tenantId: string,
  datasourceId: string
): ValidationContext {
  return {
    accountId,
    accountType,
    clientId: `client-${accountId}`,
    timestamp: new Date(),
    tenantId,
    datasourceId,
    portfolioData: {
      totalValue: 1000000,
      cash: 50000,
      positions: [
        {
          ticker: 'AAPL',
          marketValue: 350000,
          assetType: 'EQUITY',
          costBasis: 320000,
        },
        {
          ticker: 'VTSAX',
          marketValue: 400000,
          assetType: 'MUTUAL_FUND',
          costBasis: 380000,
        },
        {
          ticker: 'BOND_ETF',
          marketValue: 200000,
          assetType: 'FIXED_INCOME',
          costBasis: 195000,
        },
      ],
    },
    clientProfile: {
      fullName: 'John Doe',
      dateOfBirth: new Date('1975-05-15'),
      riskTolerance: 'MODERATE',
      investmentObjective: 'GROWTH',
      netWorth: 2500000,
      accreditedInvestorStatus: true,
      pepStatus: 'CLEAR',
    },
    transactionData: {
      type: 'BUY',
      amount: 50000,
      feePercentage: 0.005,
    },
  };
}

export default InvestmentValidationEngine;
