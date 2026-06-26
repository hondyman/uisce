import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useConfirm } from '../../components/ConfirmProvider';
import { useNotification } from '../../hooks/useNotification';
import './ValidationRulesWithFacets.css';
import { WEALTH_VALIDATION_RULES } from '../../data/wealthValidationRules';
import { MARKETPLACE_VALIDATION_RULES } from '../../data/marketplaceValidationRules';
import InvestmentValidationEngine from '../../services/validationEngine';
import { ValidationRuleCreator } from './ValidationRuleCreator';
import { RuleJsonViewer } from './RuleJsonViewer';
import { devError, devLog, devWarn } from '../../utils/devLogger';
import type { ValidationRule as SharedValidationRule } from '../../components/validation/types';

interface FacetOption {
  value: string;
  label: string;
  count: number;
}

interface _FacetResponse {
  rules: SharedValidationRule[];
  total: number;
  page: number;
  limit: number;
  has_more: boolean;
  entity_facets: FacetOption[];
  sub_entity_facets: FacetOption[];
  rule_type_facets: FacetOption[];
  severity_facets: FacetOption[];
}

interface FilterState {
  selectedEntities: string[];
  selectedRuleTypes: string[];
  selectedSeverities: string[];
  selectedScopes: string[]; // 'global', 'specific'
  selectedTypes: string[]; // 'core', 'custom'
  searchQuery: string;
}

interface ValidationRulesProps {
  tenantId: string;
  datasourceId: string;
  entities?: string[];
  // Entity schema can be a runtime object; prefer unknown and narrow locally
  entitySchema?: Record<string, unknown>;
}

// Debounce helper
function useDebounce<T>(value: T, delay: number): T {
  const [debouncedValue, setDebouncedValue] = useState<T>(value);

  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedValue(value);
    }, delay);

    return () => clearTimeout(handler);
  }, [value, delay]);

  return debouncedValue;
}

// Main Component
export const ValidationRulesWithFacets: React.FC<ValidationRulesProps> = ({
  tenantId,
  datasourceId,
  entities = [],
  entitySchema = {},
}) => {
  // Debug logging
  devLog('ValidationRulesWithFacets received entities:', entities);
  // State
  const [rules, setRules] = useState<SharedValidationRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [totalCount, setTotalCount] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const [filters, setFilters] = useState<FilterState>({
    selectedEntities: [],
    selectedRuleTypes: [],
    selectedSeverities: [],
    selectedScopes: [],
    selectedTypes: [],
    searchQuery: '',
  });

  const [totalFacetCounts, setTotalFacetCounts] = useState<{
    entities: Record<string, number>;
    ruleTypes: Record<string, number>;
    severities: Record<string, number>;
  }>({
    entities: {},
    ruleTypes: {},
    severities: {},
  });

  const [editingRule, setEditingRule] = useState<SharedValidationRule | null>(null);
  const [creatorOpen, setCreatorOpen] = useState(false);

  // JSON Viewer state
  const [jsonViewerOpen, setJsonViewerOpen] = useState(false);
  const [viewingRule, setViewingRule] = useState<SharedValidationRule | null>(null);

  const [engine, setEngine] = useState<InvestmentValidationEngine | null>(null);

  // Initialize engine
  useEffect(() => {
    if (tenantId && datasourceId) {
      const newEngine = new InvestmentValidationEngine(tenantId, datasourceId);
      setEngine(newEngine);
    }
  }, [tenantId, datasourceId]);

  // Build entity facets using total counts for stable display
  const entityFacets = useCallback(() => {
    return Object.entries(totalFacetCounts.entities)
      .map(([entity, count]) => ({ entity, count }))
      .sort((a, b) => a.entity.localeCompare(b.entity));
  }, [totalFacetCounts.entities]);

  // Build scope facets (global vs specific)
  const scopeFacets = useCallback(() => {
    const scopeMap: Record<string, number> = { global: 0, specific: 0 };

    rules.forEach((rule) => {
      const isGlobal = Array.isArray(rule.target_entities) &&
                      rule.target_entities.includes('global');
      if (isGlobal) {
        scopeMap.global += 1;
      } else {
        scopeMap.specific += 1;
      }
    });

    return Object.entries(scopeMap)
      .filter(([, count]) => count > 0)
      .map(([scope, count]) => ({ scope, count }));
  }, [rules]);

  // Build type facets (core vs custom)
  const typeFacets = useCallback(() => {
    const typeMap: Record<string, number> = { core: 0, custom: 0 };

    rules.forEach((rule) => {
      if (rule.is_core) {
        typeMap.core += 1;
      } else {
        typeMap.custom += 1;
      }
    });

    return Object.entries(typeMap)
      .filter(([, count]) => count > 0)
      .map(([type, count]) => ({ type, count }));
  }, [rules]);

  // Build rule type facets using total counts for stable display
  const ruleTypeFacets = useCallback(() => {
    return Object.entries(totalFacetCounts.ruleTypes)
      .map(([ruleType, count]) => ({ ruleType, count }))
      .sort((a, b) => a.ruleType.localeCompare(b.ruleType));
  }, [totalFacetCounts.ruleTypes]);

  // Build severity facets using total counts for stable display
  const severityFacets = useCallback(() => {
    return Object.entries(totalFacetCounts.severities)
      .map(([severity, count]) => ({ severity, count }))
      .sort((a, b) => a.severity.localeCompare(b.severity));
  }, [totalFacetCounts.severities]);

  const searchInputRef = useRef<HTMLInputElement>(null);
  const debouncedSearchQuery = useDebounce(filters.searchQuery, 300); // Correctly debounces the search query

  // Debug search query
  // devDebug('Search query:', filters.searchQuery, 'Debounced:', debouncedSearchQuery);

  // Build query string
  const buildFilterQuery = useCallback(
    (pageNum: number): string => {
      const params = new URLSearchParams();

      params.append('page', pageNum.toString());
      params.append('limit', '20');
      params.append('tenant_id', tenantId);
      params.append('tenant_instance_id', datasourceId);

      // Send each entity as a separate parameter for proper backend parsing
      if (filters.selectedEntities.length > 0) {
        filters.selectedEntities.forEach(entity => {
          params.append('target_entity', entity);
        });
      }
      if (filters.selectedRuleTypes.length > 0) {
        filters.selectedRuleTypes.forEach(ruleType => {
          params.append('rule_type', ruleType);
        });
      }
      if (filters.selectedSeverities.length > 0) {
        filters.selectedSeverities.forEach(severity => {
          params.append('severity', severity);
        });
      }
      if (filters.selectedScopes.length > 0) {
        filters.selectedScopes.forEach(scope => {
          params.append('scope', scope);
        });
      }
      if (filters.selectedTypes.length > 0) {
        filters.selectedTypes.forEach(type => {
          params.append('type', type);
        });
      }
      if (debouncedSearchQuery) {
        params.append('search', debouncedSearchQuery);
      }

      return params.toString();
    },
    [tenantId, datasourceId, filters, debouncedSearchQuery]
  );

  // Fetch rules
  const fetchRules = useCallback(
    async (pageNum: number, isLoadMore: boolean = false, signal?: AbortSignal) => {
      // Guard clauses for required props
      if (!tenantId || !datasourceId) {
          devWarn('[ValidationRules] Missing tenantId or datasourceId, skipping fetch');
          return;
      }

      setLoading(true);
      setError(null);

      try {
        const queryStr = buildFilterQuery(pageNum);
        const response = await fetch(`/api/validation-rules?${queryStr}`, {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
          signal,
        });

        if (!response.ok) {
          throw new Error(`Failed to fetch rules: ${response.statusText}`);
        }

        const dataJson = await response.json();
        const dataObj = dataJson && typeof dataJson === 'object' ? (dataJson as Record<string, unknown>) : {};

        // normalize response: map backend shape into shared ValidationRule
        const rawRules: unknown[] = Array.isArray(dataJson)
          ? (dataJson as unknown[])
          : (Array.isArray(dataObj.rules as unknown) ? (dataObj.rules as unknown[]) : []);

        const rulesArr: SharedValidationRule[] = rawRules.map((r: unknown) => {
          const rec = (r && typeof r === 'object') ? (r as Record<string, unknown>) : {};
          const id = String(rec['id'] ?? '');
          const ruleName = (rec['rule_name'] ?? rec['name']) as string | undefined;
          const targetEntities = rec['target_entities'];

          const conditionJson = rec['condition_json'];
          const conditions = (conditionJson && typeof conditionJson === 'object' && Array.isArray((conditionJson as Record<string, unknown>)['conditions']))
            ? ((conditionJson as Record<string, unknown>)['conditions'] as unknown[])
            : (Array.isArray(rec['conditions']) ? (rec['conditions'] as unknown[]) : undefined);

          return ({
            id,
            name: ruleName ?? undefined,
            rule_name: ruleName ?? undefined,
            rule_type: String(rec['rule_type'] ?? ''),
            entity: String(rec['entity'] ?? rec['target_entity'] ?? ''),
            target_entity: String(rec['target_entity'] ?? rec['entity'] ?? ''),
            target_entities: Array.isArray(targetEntities) ? (targetEntities as string[]) : (typeof targetEntities === 'string' ? [String(targetEntities)] : []),
            sub_entity_type: String(rec['sub_entity_type'] ?? ''),
            severity: String(rec['severity'] ?? ''),
            description: String(rec['description'] ?? ''),
            is_active: Boolean(rec['is_active']),
            is_global: Boolean(rec['is_global']),
            is_core: Boolean(rec['is_core']),
            conditions: Array.isArray(conditions) ? (conditions as unknown as SharedValidationRule['conditions']) : undefined,
            condition_json: rec['condition_json'] ?? undefined,
            dependent_rule_ids: Array.isArray(rec['dependent_rule_ids']) ? (rec['dependent_rule_ids'] as string[]) : [],
            created_at: String(rec['created_at'] ?? ''),
            updated_at: String(rec['updated_at'] ?? ''),
            script_content: String(rec['script_content'] ?? ''), // Ensure script_content is mapped
          } as SharedValidationRule);
        });

        setRules((prev) => (isLoadMore ? [...prev, ...rulesArr] : rulesArr));
        setPage(pageNum);
        const dj = dataObj;
        setTotalCount(Number(dj.total ?? 0));
        setHasMore(Boolean(dj.has_more));

        // Update facets from new shape or fallback to legacy fields
        const facetSource = (dj['facets'] as Record<string, unknown> | undefined) ?? {
          entities: dj['entity_facets'] ?? [],
          rule_types: dj['rule_type_facets'] ?? [],
          severities: dj['severity_facets'] ?? [],
        } as Record<string, unknown>;

        // Update total facet counts when no filters are applied (for stable counts)
        const hasNoFilters = !filters.selectedEntities.length &&
                            !filters.selectedRuleTypes.length &&
                            !filters.selectedSeverities.length &&
                            !filters.selectedScopes.length &&
                            !filters.selectedTypes.length &&
                            !debouncedSearchQuery;

        if (hasNoFilters) {
          const entityCounts: Record<string, number> = {};
          const ruleTypeCounts: Record<string, number> = {};
          const severityCounts: Record<string, number> = {};

          // Build counts from facet data
          ((facetSource.entities as unknown[]) || []).forEach((facet: unknown) => {
            const f = (facet && typeof facet === 'object') ? (facet as FacetOption) : null;
            if (f) entityCounts[f.value] = f.count;
          });
          ((facetSource.rule_types as unknown[]) || []).forEach((facet: unknown) => {
            const f = (facet && typeof facet === 'object') ? (facet as FacetOption) : null;
            if (f) ruleTypeCounts[f.value] = f.count;
          });
          ((facetSource.severities as unknown[]) || []).forEach((facet: unknown) => {
            const f = (facet && typeof facet === 'object') ? (facet as FacetOption) : null;
            if (f) severityCounts[f.value] = f.count;
          });

          setTotalFacetCounts({
            entities: entityCounts,
            ruleTypes: ruleTypeCounts,
            severities: severityCounts,
          });
        }
      } catch (err: any) {
        if (err.name === 'AbortError') return;
        setError(err instanceof Error ? err.message : 'Unknown error occurred');
        devError('Error fetching validation rules:', err);
      } finally {
        if (!signal?.aborted) {
            setLoading(false);
        }
      }
    },
    [buildFilterQuery, tenantId, datasourceId, filters, debouncedSearchQuery]
  );

  // Initial load and when search/filters change
  useEffect(() => {
    const controller = new AbortController();
    fetchRules(1, false, controller.signal);
    return () => controller.abort();
  }, [
    fetchRules, // Depend on fetchRules which updates with filters
    // Explicit dependencies for clarity, though included in fetchRules dependency
    targetDependenciesString(filters, debouncedSearchQuery, tenantId, datasourceId) 
  ]);

  // Helper for stable dependency string
  function targetDependenciesString(f: FilterState, q: string, t: string, d: string) {
      return `${JSON.stringify(f.selectedEntities)}-${JSON.stringify(f.selectedRuleTypes)}-${JSON.stringify(f.selectedSeverities)}-${JSON.stringify(f.selectedScopes)}-${JSON.stringify(f.selectedTypes)}-${q}-${t}-${d}`;
  }

  // Handlers
  const handleFacetChange = (
    category: keyof FilterState,
    value: string,
    checked: boolean
  ) => {
    const newFilters = { ...filters };

    if (
      category === 'selectedEntities' ||
      category === 'selectedRuleTypes' ||
      category === 'selectedSeverities' ||
      category === 'selectedScopes' ||
      category === 'selectedTypes'
    ) {
      const current = filters[category] as string[];
      newFilters[category] = checked
        ? [...current, value]
        : current.filter((v) => v !== value);
    }

    setFilters(newFilters);
    setPage(1);
  };

  const handleSearchChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setFilters({ ...filters, searchQuery: e.target.value });
    setPage(1);
  };

  const handleLoadMore = () => {
    fetchRules(page + 1, true);
  };

  const handleToggleRuleActive = async (ruleId: string, isActive: boolean) => {
    try {
      const response = await fetch(`/api/validation-rules/${ruleId}`, {
        method: 'PATCH',
        headers: {
          'X-Tenant-ID': tenantId,
          'X-Tenant-Datasource-ID': datasourceId,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ is_active: isActive }),
      });

      if (!response.ok) {
        throw new Error(`Failed to update rule: ${response.statusText}`);
      }

      // The optimistic update already happened in the checkbox onChange
      devLog('Rule active status updated:', ruleId, isActive);
    } catch (err) {
      devError('Error updating rule active status:', err);
      // Revert the optimistic update by refetching
      fetchRules(page, false);
    }
  };

  const handleDeleteRule = async (ruleId: string) => {
      const confirm = useConfirm();
      if (!(await confirm({ title: 'Delete validation rule', description: 'Are you sure you want to delete this validation rule? This action cannot be undone.' }))) {
        return;
      }

      try {
        const response = await fetch(`/api/validation-rules/${ruleId}?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`, {
          method: 'DELETE',
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        });

        if (!response.ok) {
          throw new Error(`Failed to delete rule: ${response.statusText}`);
        }

        setRules((prev) => prev.filter((r) => r.id !== ruleId));
        setTotalCount((prev) => prev - 1);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error occurred');
        devError('Error deleting rule:', err);
      }
  };

  const clearAllFilters = () => {
    setFilters({
      selectedEntities: [],
      selectedRuleTypes: [],
      selectedSeverities: [],
      selectedScopes: [],
      selectedTypes: [],
      searchQuery: '',
    });
    if (searchInputRef.current) {
      searchInputRef.current.value = '';
    }
    setPage(1);
  };

  const handleImportRules = async () => {
    if (!engine) { return; }
    const confirm = useConfirm();
    const notification = useNotification();
    if (!(await confirm({ title: 'Import sample rules', description: 'Import sample wealth-management validation rules into the current tenant/datasource?' }))) return;
    setLoading(true);
    try {
      let created = 0;
      let updated = 0;
      let skipped = 0;
      const failedRules: Array<{ ruleName: string; error: string }> = [];

      for (const r of (WEALTH_VALIDATION_RULES as any)) {
        const payload: any = {
          id: r.id,
          name: r.name,
          description: r.description,
          rule_type: r.rule_type, // Use rule_type directly
          scope: r.scope,
          severity: r.severity,
          isActive: r.isActive,
          is_core: r.isCore, // Add is_core field
          effectiveFrom: r.effectiveFrom,
          ...(r.effectiveTo ? { effectiveTo: r.effectiveTo } : {}),
          frequency: r.frequency,
          evaluationOrder: r.evaluationOrder,
          overrideConditions: r.overrideConditions,
          requiredAuthority: r.requiredAuthority,
          parameters: r.parameters,
          tenantId: tenantId,
          datasourceId: datasourceId,
        };

        devLog(`[handleImportRules] Processing rule: "${r.name}" (id: "${r.id}", type: "${r.rule_type}")`);

        // Pre-flight check: search for existing rule by name to decide create vs update
        let existingRuleId: string | undefined;
        try {
          const searchResp = await fetch(
            `/api/validation-rules?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&search=${encodeURIComponent(r.name)}`,
            {
              headers: {
                'X-Tenant-ID': tenantId,
                'X-Tenant-Datasource-ID': datasourceId,
              },
            }
          );
          const searchData = await searchResp.json();
          const existingRules = (searchData.rules || []).filter((rule: any) => rule.rule_name === r.name);
          if (existingRules.length > 0) {
            existingRuleId = existingRules[0].id;
            devLog(`[handleImportRules] Found existing rule: "${r.name}" (id: "${existingRuleId}")`);
          }
        } catch (searchErr) {
          devWarn(`[handleImportRules] Search for existing rule failed for "${r.name}":`, searchErr);
        }

        try {
          if (existingRuleId) {
            // Rule exists, attempt update
            devLog(`[handleImportRules] Updating existing rule: "${r.name}"`);
            await engine.updateRule(existingRuleId, payload);
            updated += 1;
            devLog(`[handleImportRules] Rule updated successfully: "${r.name}"`);
          } else {
            // Rule does not exist, create it
            devLog(`[handleImportRules] Creating new rule: "${r.name}"`);
            await engine.createRule(payload);
            created += 1;
            devLog(`[handleImportRules] Rule created successfully: "${r.name}"`);
          }
        } catch (err: any) {
          const errorMsg = err?.message || String(err);
          console.error(`[handleImportRules] Failed to create/update rule "${r.name}":`, errorMsg);

          // If creation failed due to invalid rule_type, skip and do not attempt update
          if (errorMsg.includes('Invalid rule_type')) {
            devWarn(`[handleImportRules] Skipping rule "${r.name}" due to invalid rule_type`);
            skipped += 1;
            failedRules.push({ ruleName: r.name, error: 'Invalid rule_type' });
            continue;
          }

          // If creation failed due to duplicate key error, attempt fallback update
          if (errorMsg.includes('duplicate key') || errorMsg.includes('unique constraint')) {
            devLog(`[handleImportRules] Duplicate key detected for "${r.name}", attempting update via name search...`);
            try {
              const fallbackId = await engine.findRuleIdByName(r.name || r.id).catch(() => undefined);
              if (fallbackId) {
                await engine.updateRule(fallbackId, payload);
                updated += 1;
                devLog(`[handleImportRules] Rule updated successfully via fallback: "${r.name}"`);
              } else {
                console.error(`[handleImportRules] Could not find rule for fallback update: "${r.name}"`);
                skipped += 1;
                failedRules.push({ ruleName: r.name, error: 'Duplicate key but rule not found for update' });
              }
            } catch (fallbackErr) {
              console.error(`[handleImportRules] Fallback update failed for "${r.name}":`, fallbackErr);
              skipped += 1;
              failedRules.push({ ruleName: r.name, error: 'Fallback update failed' });
            }
          } else {
            skipped += 1;
            failedRules.push({ ruleName: r.name, error: errorMsg });
          }
        }
      }

      devLog('[handleImportRules] Import summary:', { created, updated, skipped, totalRules: WEALTH_VALIDATION_RULES.length });
      if (failedRules.length > 0) {
        devWarn('[handleImportRules] Failed rules:', failedRules);
        notification.error(`Import completed with issues: Created ${created}, Updated ${updated}, Skipped ${skipped}`);
      } else {
        notification.success(`Import completed successfully: Created ${created}, Updated ${updated}`);
      }
      fetchRules(1, false);
    } catch (e) {
      devError('Import failed', e);
      notification.error(`Import failed: ${e}`);
    } finally {
      setLoading(false);
    }
  };

  const handleImportMarketplaceRules = async () => {
    if (!engine) {
      devError('Import failed: Engine not initialized');
      notification.error('Import failed: Engine not initialized');
      return;
    }
    if (!(await confirm({ title: 'Import marketplace rules', description: 'Import marketplace validation rules into the current tenant/datasource?' }))) return;
    setLoading(true);
    try {
      let created = 0;
      let updated = 0;
      let skipped = 0;
      const failedRules: Array<{ ruleName: string; error: string }> = [];

      for (const r of MARKETPLACE_VALIDATION_RULES) {
        const payload: any = {
          id: r.id,
          name: r.name,
          description: r.description,
          rule_type: r.rule_type,
          scope: r.scope,
          severity: r.severity,
          isActive: r.isActive,
          effectiveFrom: r.effectiveFrom,
          frequency: r.frequency,
          evaluationOrder: r.evaluationOrder,
          parameters: r.parameters,
          tenantId: tenantId,
          datasourceId: datasourceId,
        };

        // Use existing upsert logic from handleImportRules
        let existingRuleId: string | undefined;
        try {
          const searchResp = await fetch(
            `/api/validation-rules?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}&search=${encodeURIComponent(r.name)}`,
            {
              headers: {
                'X-Tenant-ID': tenantId,
                'X-Tenant-Datasource-ID': datasourceId,
              },
            }
          );
          const searchData = await searchResp.json();
          const existingRules = (searchData.rules || []).filter((rule: any) => rule.rule_name === r.name);
          if (existingRules.length > 0) {
            existingRuleId = existingRules[0].id;
          }
        } catch (searchErr) {
          // ignore
        }

        try {
          if (existingRuleId) {
            await engine.updateRule(existingRuleId, payload);
            updated += 1;
          } else {
            await engine.createRule(payload);
            created += 1;
          }
        } catch (err: any) {
            skipped += 1;
            failedRules.push({ ruleName: r.name, error: err?.message || String(err) });
        }
      }

      if (failedRules.length > 0) {
        notification.info(`Import completed with issues:\n✓ Created: ${created}\n✓ Updated: ${updated}\n✗ Skipped: ${skipped}\n\nFailed rules: ${failedRules.map(f => `${f.ruleName} (${f.error})`).join(', ')}`);
      } else {
        notification.success(`Import completed successfully!\n✓ Created: ${created}\n✓ Updated: ${updated}`);
      }
      fetchRules(1, false);
    } catch (e) {
      devError('Marketplace import failed', e);
      notification.error(`Marketplace import failed with error: ${e}`);
    } finally {
      setLoading(false);
    }
  };

  const getRuleTypeLabel = (value: string): string => {
    const labels: Record<string, string> = {
      field_format: 'Field Format',
      business_logic: 'Business Logic',
      cardinality: 'Cardinality',
      uniqueness: 'Uniqueness',
      referential_integrity: 'Referential Integrity',
    };
    return labels[value] || value;
  };

  const getSeverityLabel = (value: string): string => {
    const labels: Record<string, string> = {
      error: 'Error',
      warning: 'Warning',
      info: 'Info',
    };
    return labels[value] || value;
  };

  const getEntityLabel = (value: string): string => {
    // Entities typically don't need transformation, but keeping for consistency
    return value;
  };

  const getRuleTypeIcon = (ruleType: string): string => {
    const iconMap: Record<string, string> = {
      business_logic: '⚙️',
      field_format: '📝',
      cardinality: '📊',
      referential: '🔗',
      uniqueness: '🔑',
    };
    return iconMap[ruleType] || '📋';
  };

  const getSeverityIcon = (severity: string): string => {
    const iconMap: Record<string, string> = {
      error: '🔴',
      warning: '🟠',
      info: '🔵',
    };
    return iconMap[severity] || '⚪';
  };

  const hasActiveFilters =
    filters.selectedEntities.length > 0 ||
    filters.selectedRuleTypes.length > 0 ||
    filters.selectedSeverities.length > 0 ||
    filters.selectedScopes.length > 0 ||
    filters.selectedTypes.length > 0 ||
    filters.searchQuery.length > 0;

  return (
    <div className="validation-rules-container">
      {error && (
        <div className="error-notification">
          <span>✗ {error}</span>
        </div>
      )}

      <div className="rules-layout">
        {/* Sidebar Facets */}
        <aside className="facets-sidebar">
          <div className="facets-section header-section">
            <h3>FILTERS</h3>
            {/* Clear Filters Icon Button */}
            {hasActiveFilters && (
              <button
                className="clear-filters-icon"
                onClick={clearAllFilters}
                title="Clear all filters"
              >
                ✕
              </button>
            )}
          </div>

          {/* Entity Facets - Flat List */}
          <div className="facets-section">
            <h4>📁 ENTITIES</h4>
            <div className="facet-list">
              {entityFacets().map((entityItem) => (
                <label key={entityItem.entity} className="facet-item">
                  <input
                    type="checkbox"
                    checked={filters.selectedEntities.includes(entityItem.entity)}
                    onChange={(e) =>
                      handleFacetChange(
                        'selectedEntities',
                        entityItem.entity,
                        e.target.checked
                      )
                    }
                  />
                  <span className="facet-label">
                    {getEntityLabel(entityItem.entity)}
                  </span>
                  <span className="facet-count">({entityItem.count})</span>
                </label>
              ))}
            </div>
          </div>

          {/* Rule Type Facets */}
          <div className="facets-section">
            <h4>📋 RULE TYPES</h4>
            <div className="facet-list">
              {ruleTypeFacets().map((facet) => (
                <label key={facet.ruleType} className="facet-item">
                  <input
                    type="checkbox"
                    checked={filters.selectedRuleTypes.includes(facet.ruleType)}
                    onChange={(e) =>
                      handleFacetChange(
                        'selectedRuleTypes',
                        facet.ruleType,
                        e.target.checked
                      )
                    }
                  />
                  <span className="facet-label">{getRuleTypeLabel(facet.ruleType)}</span>
                  <span className="facet-count">({facet.count})</span>
                </label>
              ))}
            </div>
          </div>

          {/* Severity Facets */}
          <div className="facets-section">
            <h4>⚠️ SEVERITY</h4>
            <div className="facet-list">
              {severityFacets().map((facet) => (
                <label key={facet.severity} className="facet-item">
                  <input
                    type="checkbox"
                    checked={filters.selectedSeverities.includes(facet.severity)}
                    onChange={(e) =>
                      handleFacetChange(
                        'selectedSeverities',
                        facet.severity,
                        e.target.checked
                      )
                    }
                  />
                  <span className="facet-label">{getSeverityLabel(facet.severity)}</span>
                  <span className="facet-count">({facet.count})</span>
                </label>
              ))}
            </div>
          </div>

          {/* Scope Facets */}
          <div className="facets-section">
            <h4>🌍 SCOPE</h4>
            <div className="facet-list">
              {scopeFacets().map((scopeItem) => (
                <label key={scopeItem.scope} className="facet-item">
                  <input
                    type="checkbox"
                    checked={filters.selectedScopes.includes(scopeItem.scope)}
                    onChange={(e) =>
                      handleFacetChange(
                        'selectedScopes',
                        scopeItem.scope,
                        e.target.checked
                      )
                    }
                  />
                  <span className="facet-label">
                    {scopeItem.scope === 'global' ? '🌍 Global' : '🎯 Specific'}
                  </span>
                  <span className="facet-count">({scopeItem.count})</span>
                </label>
              ))}
            </div>
          </div>

          {/* Type Facets */}
          <div className="facets-section">
            <h4>🏢 TYPE</h4>
            <div className="facet-list">
              {typeFacets().map((typeItem) => (
                <label key={typeItem.type} className="facet-item">
                  <input
                    type="checkbox"
                    checked={filters.selectedTypes.includes(typeItem.type)}
                    onChange={(e) =>
                      handleFacetChange(
                        'selectedTypes',
                        typeItem.type,
                        e.target.checked
                      )
                    }
                  />
                  <span className="facet-label">
                    {typeItem.type === 'core' ? '🔒 Core' : '✏️ Custom'}
                  </span>
                  <span className="facet-count">({typeItem.count})</span>
                </label>
              ))}
            </div>
          </div>
        </aside>

        {/* Main Content */}
        <main className="rules-main">
          {/* Search Bar and Add Button */}
          <div className="search-bar-container">
            <div className="search-input-wrapper">
              <input
                ref={searchInputRef}
                type="text"
                placeholder="🔍 Search rules by name, description..."
                className="search-input"
                value={filters.searchQuery}
                onChange={handleSearchChange}
              />
            </div>
            <button
              className="add-rule-btn"
              onClick={() => setCreatorOpen(true)}
              title="Create a new validation rule"
            >
              + Add Rule
            </button>
            <button
              className="import-rules-btn"
              onClick={handleImportRules}
              title="Import Wealth Management Rules"
            >
              Import Wealth Rules
            </button>
            <button
              className="import-rules-btn"
              onClick={handleImportMarketplaceRules}
              title="Import Marketplace Validation Rules"
            >
              Import Marketplace Rules
            </button>
          </div>

          {/* Rules List */}
          <div className="rules-list">
            {loading && rules.length === 0 ? (
              <div className="loading-state">
                <span>⏳ Loading rules...</span>
              </div>
            ) : rules.length === 0 ? (
              <div className="empty-state">
                <p>No rules found matching your filters.</p>
                <button onClick={clearAllFilters} className="secondary-btn">
                  Clear Filters
                </button>
              </div>
            ) : (
              <>
                {rules.map((rule) => (
                  <div key={rule.id} className={`rule-item ${!rule.is_active ? 'inactive' : ''}`}>
                    <div className="rule-header">
                      <div className="rule-title">
                        <input
                          type="checkbox"
                          checked={rule.is_active !== false}
                          onChange={(e) => {
                            if (!rule.id) return;
                            const updated = { ...rule, is_active: e.target.checked };
                            setRules(prevRules =>
                              prevRules.map(r => r.id === rule.id ? updated : r)
                            );
                            // Optimistically update, then make API call
                            handleToggleRuleActive(rule.id, e.target.checked);
                          }}
                          title={rule.is_active ? "Active" : "Inactive"}
                          className="rule-active-checkbox"
                        />
                        <span className="rule-type-icon">
                          {getRuleTypeIcon(rule.rule_type || '')}
                        </span>
                        <span className="rule-name">{rule.rule_name}</span>
                        <span className="rule-entity">{rule.target_entity}</span>
                        {rule.sub_entity_type && (
                          <span className="rule-sub-entity">
                            → {rule.sub_entity_type}
                          </span>
                        )}
                      </div>
                      <div className="rule-meta">
                        <span className="severity-badge">
                          {getSeverityIcon(rule.severity)} {rule.severity}
                        </span>
                        {/* Rule Type Chips */}
                        <div className="rule-chips">
                          {Array.isArray(rule.target_entities) && rule.target_entities.includes('global') && (
                            <span className="rule-chip global-chip">🌍 Global</span>
                          )}
                          <span className={`rule-chip type-chip ${rule.is_core ? 'core-chip' : 'custom-chip'}`}>
                            {rule.is_core ? '🔒 Core' : '✏️ Custom'}
                          </span>
                          <span className="rule-chip rule-type-chip">
                            {getRuleTypeLabel(rule.rule_type || '')}
                          </span>
                        </div>
                      </div>
                    </div>
                    <div className="rule-description">{rule.description}</div>
                    <div className="rule-actions">
                      <button
                        className="icon-btn"
                        title="View JSON Configuration"
                        onClick={() => {
                          setViewingRule(rule);
                          setJsonViewerOpen(true);
                        }}
                      >
                        👁
                      </button>
                      <button
                        className="icon-btn"
                        title="Edit Rule"
                        onClick={() => {
                          setEditingRule(rule);
                          setCreatorOpen(true);
                        }}
                      >
                        ✎
                      </button>
                      <button className="icon-btn" title="Copy">
                        📋
                      </button>
                      <button 
                        className="icon-btn delete" 
                        title="Delete"
                        onClick={() => handleDeleteRule(rule.id!)}
                      >
                        🗑
                      </button>
                    </div>
                  </div>
                ))}

                {/* Load More Button */}
                {hasMore && (
                  <div className="load-more-container">
                    <button
                      onClick={handleLoadMore}
                      disabled={loading}
                      className="load-more-btn"
                    >
                      {loading ? (
                        <span>⏳ Loading...</span>
                      ) : (
                        <span>
                          ⏬ Load 20 more rules ({totalCount - rules.length}{' '}
                          remaining)
                        </span>
                      )}
                    </button>
                  </div>
                )}

                {/* Results Summary */}
                <div className="results-summary">
                  {!hasMore && (
                    <p>
                      Showing all {rules.length} of {totalCount} rules
                    </p>
                  )}
                </div>
              </>
            )}
          </div>
        </main>
      </div>

      <RuleJsonViewer
        rule={viewingRule}
        isOpen={jsonViewerOpen}
        onClose={() => {
          setJsonViewerOpen(false);
          setViewingRule(null);
        }}
      />

      <ValidationRuleCreator
        isOpen={creatorOpen || !!editingRule}
        onClose={() => {
          setCreatorOpen(false);
          setEditingRule(null);
        }}
        onSave={(newRule) => {
          if (editingRule) {
            // Update mode: refresh to reflect changes
            setRules(prevRules =>
              prevRules.map(rule =>
                rule.id === newRule.id ? newRule : rule
              )
            );
          } else {
            // Create mode: add new rule
            setRules(prevRules => [newRule, ...prevRules]);
            fetchRules(1, false);
          }
        }}
        tenantId={tenantId}
        datasourceId={datasourceId}
        availableEntities={entities}
        entitySchema={entitySchema}
        editingRule={editingRule}
      />
    </div>
  );
};

export default ValidationRulesWithFacets;
