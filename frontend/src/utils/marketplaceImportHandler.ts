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
  _tenantId: string,
  _datasourceId: string,
  selectedRuleIds?: string[] // If undefined, all rules are reported skipped
): Promise<ImportResult> {
  // Import is retired: POST /api/validation-rules (rule creation) is
  // 410 Gone, and the replacement endpoint (POST /validation-rule-nodes)
  // needs a rule_ast, not the flat condition shape this fixture data is
  // in. Unlike the facets create path, this isn't a translation that
  // could be added later without content work: none of these templates
  // carry a target_entity, and their `scope` values (ALL_ACCOUNTS,
  // WEALTH_ACCOUNT, IRA_ACCOUNT, ...) don't match any of the 5 real BOs
  // in the catalog (execution, execution_allocation, order,
  // order_allocation, placement) - the same shape as the already-retired
  // catalog_validation_rules corpus (0/233 field-resolvable). Each
  // template would need a human to hand-author it against a real BO;
  // report every rule as skipped with that reason rather than attempt a
  // mechanical translation that can't exist.
  const rulesToImport = selectedRuleIds
    ? MARKETPLACE_VALIDATION_RULES.filter((r) => selectedRuleIds.includes(r.id))
    : MARKETPLACE_VALIDATION_RULES;

  const result: ImportResult = {
    created: 0,
    updated: 0,
    skipped: rulesToImport.length,
    failed: rulesToImport.map((marketplaceRule) => ({
      ruleName: marketplaceRule.name,
      error:
        'Marketplace import is retired - these templates target account scopes that don\'t map to any real business object. Author rules directly in the Validation Rules editor instead.',
    })),
  };

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
