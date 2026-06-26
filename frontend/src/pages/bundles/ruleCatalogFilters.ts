/**
 * Rules Catalog Filtering Logic
 *
 * Functions for filtering and sorting rules in the catalog
 */

import { FilterOptions, RuleCatalogItem, RULE_CATEGORIES } from './ruleCatalogConstants';
import { getSeverityOrder } from './ruleCatalogUtils';
import WEALTH_VALIDATION_RULES from '@/data/wealthValidationRules';
import { getParametersForRule } from '@/data/ValidationRuleParametersRegistry';

export const filterAndSortRules = (filters: FilterOptions): RuleCatalogItem[] => {
  let results = WEALTH_VALIDATION_RULES.map(rule => {
    const ruleCategories = RULE_CATEGORIES.filter(cat => cat.ruleIds.includes(rule.id));
    const parameters = getParametersForRule(rule.name);
    return { rule, categories: ruleCategories, parameters };
  });

  // Apply search
  if (filters.search) {
    const searchLower = filters.search.toLowerCase();
    results = results.filter(item =>
      item.rule.name.toLowerCase().includes(searchLower) ||
      item.rule.description.toLowerCase().includes(searchLower) ||
      item.categories.some(cat => cat.name.toLowerCase().includes(searchLower))
    );
  }

  // Apply category filter
  if (filters.categories.length > 0) {
    results = results.filter(item =>
      item.categories.some(cat => filters.categories.includes(cat.id))
    );
  }

  // Apply severity filter
  if (filters.severities.length > 0) {
    results = results.filter(item => filters.severities.includes(item.rule.severity));
  }

  // Apply frequency filter
  if (filters.frequencies.length > 0) {
    results = results.filter(item => filters.frequencies.includes(item.rule.frequency));
  }

  // Apply rule type filter
  if (filters.ruleTypes.length > 0) {
    results = results.filter(item => filters.ruleTypes.includes(item.rule.rule_type));
  }

  // Apply core rule filter
  if (filters.isCore !== undefined) {
    results = results.filter(item => item.rule.isCore === filters.isCore);
  }

  // Sort
  results.sort((a, b) => {
    switch (filters.sortBy) {
      case 'name':
        return a.rule.name.localeCompare(b.rule.name);
      case 'severity':
        return getSeverityOrder(a.rule.severity) - getSeverityOrder(b.rule.severity);
      case 'evaluationOrder':
      default:
        return a.rule.evaluationOrder - b.rule.evaluationOrder;
    }
  });

  return results;
};

export const getUniqueValues = () => {
  const uniqueSeverities = [...new Set(WEALTH_VALIDATION_RULES.map(r => r.severity))];
  const uniqueFrequencies = [...new Set(WEALTH_VALIDATION_RULES.map(r => r.frequency))];
  const uniqueRuleTypes = [...new Set(WEALTH_VALIDATION_RULES.map(r => r.rule_type))];

  return {
    uniqueSeverities,
    uniqueFrequencies,
    uniqueRuleTypes
  };
};