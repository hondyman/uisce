/**
 * Marketplace Validation Rules Import Handler
 * 
 * Provides functions to import validation rules from the marketplace
 * into the current tenant/datasource context.
 */

import { MARKETPLACE_VALIDATION_RULES } from '../data/marketplaceValidationRules';

// Minimal local type describing the marketplace rule shape we rely on here.
export interface MarketplaceValidationRule {
  id: string
  name: string
  description?: string
  rule_type?: string
  scope?: string[]
  severity?: 'BLOCK' | 'WARNING' | 'INFO' | string
  isActive?: boolean
  effectiveFrom?: string
  frequency?: string
  evaluationOrder?: number
  parameters?: Record<string, any>
  category?: string
  [key: string]: any
}

export interface ImportResult {
  created: number;
  updated: number;
  skipped: number;
  failed: Array<{ ruleName: string; error: string }>;
}

/**
 * Import marketplace validation rules
 * 
 * Converts marketplace rules to the format expected by the validation rules engine
 * and creates them via API calls.
 */
export async function importMarketplaceValidationRules(
  tenantId: string,
  datasourceId: string,
  selectedRuleIds?: string[] // If undefined, import all
): Promise<ImportResult> {
  const result: ImportResult = {
    created: 0,
    updated: 0,
    skipped: 0,
    failed: [],
  };

  // Determine which rules to import
  const rulesToImport = selectedRuleIds
    ? MARKETPLACE_VALIDATION_RULES.filter((r) => selectedRuleIds.includes(r.id))
    : MARKETPLACE_VALIDATION_RULES;

  // Import each rule
  for (const marketplaceRule of rulesToImport) {
    try {
      // Convert marketplace rule to API payload format
      const payload = {
        id: marketplaceRule.id,
        name: marketplaceRule.name,
        description: marketplaceRule.description,
        rule_type: marketplaceRule.rule_type,
        scope: marketplaceRule.scope,
        severity: marketplaceRule.severity,
        isActive: marketplaceRule.isActive,
        effectiveFrom: marketplaceRule.effectiveFrom,
        frequency: marketplaceRule.frequency,
        evaluationOrder: marketplaceRule.evaluationOrder,
        condition_json: marketplaceRule.parameters,
        tenantId: tenantId,
        datasourceId: datasourceId,
      };

      // Call API to create or update rule
      const response = await fetch('/api/validation-rules', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
        },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        const error = await response.text();
        result.failed.push({
          ruleName: marketplaceRule.name,
          error: `HTTP ${response.status}: ${error}`,
        });
        result.skipped++;
      } else {
        result.created++;
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : String(error);
      result.failed.push({
        ruleName: marketplaceRule.name,
        error: errorMsg,
      });
      result.skipped++;
    }
  }

  return result;
}

/**
 * Get a summary of marketplace rules for display
 */
export function getMarketplaceRulesSummary() {
  // Treat the imported data as MarketplaceValidationRule[] and avoid blanket `any` casts.
  const rules = MARKETPLACE_VALIDATION_RULES as unknown as MarketplaceValidationRule[]
  const byCategory = new Map<string, MarketplaceValidationRule[]>();

  for (const rule of rules) {
    // marketplace data doesn't always include category; default to 'Uncategorized'
    const category = rule.category ?? 'Uncategorized'
    if (!byCategory.has(category)) {
      byCategory.set(category, []);
    }
    byCategory.get(category)!.push(rule);
  }

  const summary = Array.from(byCategory.entries())
    .map(([category, rules]) => ({
      category,
      count: rules.length,
      byRisk: {
        block: rules.filter((r) => r.severity === 'BLOCK').length,
        warning: rules.filter((r) => r.severity === 'WARNING').length,
        info: rules.filter((r) => r.severity === 'INFO').length,
      },
    }))
    .sort((a, b) => a.category.localeCompare(b.category));

  return {
    totalRules: MARKETPLACE_VALIDATION_RULES.length,
    totalCategories: byCategory.size,
    bySeverity: {
      block: MARKETPLACE_VALIDATION_RULES.filter((r) => r.severity === 'BLOCK').length,
      warning: MARKETPLACE_VALIDATION_RULES.filter((r) => r.severity === 'WARNING').length,
      info: MARKETPLACE_VALIDATION_RULES.filter((r) => r.severity === 'INFO').length,
    },
    byCategory: summary,
  };
}

export default importMarketplaceValidationRules;
