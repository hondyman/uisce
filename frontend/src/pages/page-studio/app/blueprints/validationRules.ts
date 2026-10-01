import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { Action, CellSpec, ChipColor, ColumnDef, ConditionNode } from '../appModel';

/**
 * Validation Rules & Rulefabric Page Studio Blueprint:
 * 
 * Provides centralized management for:
 * 1. Rule Catalog & Explorer (Core vs Custom, Active/Draft, Rule Key)
 * 2. Rule Bundle Porter (GitOps JSON/YAML Export & Preflight Dry-Run Import)
 * 3. Interactive Rule Evaluator & Simulator (Single payload & batch testing)
 * 4. Violations Live Monitor (Queryable telemetric sink on alpha.public.validation_rule_violations)
 */

const fit = { flex: '0 0 auto' };

type Spec = Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>;
const layout = (root: string, spec: Spec): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };

export function validationRulesBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const reg = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}): string => {
    components[id] = { id, type, props, ...extra };
    return id;
  };

  // Header
  reg('hdr', 'PageHeader', {
    icon: 'rule',
    title: 'Validation Rules & Rulefabric Studio',
    subtitle: 'Centralized single-store validation catalog, portable AST bundles, and live evaluation engine',
  }, { style: { flex: '1 1 320px' } });

  // Top Controls
  reg('bo_select', 'VariableSelect', {
    variable: 'selected_bo',
    label: 'Business Object',
    minWidth: 180,
    options: [
      { value: 'order', label: 'Order (orm.order)' },
      { value: 'execution', label: 'Execution (orm.execution)' },
      { value: 'security', label: 'Security (oms.security)' },
      { value: 'account', label: 'Account (oms.account)' },
      { value: 'party', label: 'Party / Customer (master.customer)' },
    ],
  }, { style: fit });

  reg('domain_select', 'VariableSelect', {
    variable: 'domain_filter',
    label: 'Domain Scope',
    emptyLabel: 'All Domains',
    minWidth: 160,
    options: [
      { value: 'validation', label: 'Business Validation' },
      { value: 'mdm', label: 'MDM Data Quality' },
      { value: 'compliance', label: 'Regulatory & Compliance' },
    ],
  }, { style: fit });

  // Tabs
  const tabs: PageTab[] = [
    {
      id: 'catalog',
      label: 'Rule Catalog & GitOps Porter',
      layout: layout('catalog_root', {
        catalog_root: { type: 'Column', children: ['porter_widget', 'rules_grid'], style: { gap: '16px' } },
      }),
    },
    {
      id: 'evaluator',
      label: 'Interactive Rule Tester',
      layout: layout('eval_root', {
        eval_root: { type: 'Column', children: ['eval_tester_widget'], style: { gap: '16px' } },
      }),
    },
    {
      id: 'violations',
      label: 'Live Violations Monitor',
      layout: layout('violation_root', {
        violation_root: { type: 'Column', children: ['violations_widget'], style: { gap: '16px' } },
      }),
    },
  ];

  // Tab 1 Components
  reg('porter_widget', 'validationRules.RuleBundleManager', {
    bo_name: '{{vars.selected_bo}}',
    domain: '{{vars.domain_filter}}',
  });

  reg('rules_grid', 'DataGrid', {
    query: 'rules_list',
    rowsPath: 'rules',
    rowKey: 'id',
    enableSearch: true,
    searchFields: ['name', 'rule_key', 'description'],
    emptyText: 'No validation rules found for this Business Object.',
    columns: [
      col('rule_key', 'Rule Key', { kind: 'text', value: '{{row.rule_key}}' }),
      col('name', 'Rule Name', { kind: 'text', value: '{{row.name}}' }),
      col('bo', 'Business Object', { kind: 'chip', value: '{{row.bo_name}}', variant: 'outlined' }),
      col('severity', 'Severity', {
        kind: 'chip',
        value: '{{row.severity}}',
        colorMap: { BLOCK: 'error', WARN: 'warning' },
      }),
      col('timing', 'Timing', { kind: 'text', value: '{{row.timing}}' }),
      col('origin', 'Origin', {
        kind: 'chip',
        value: '{{row.origin}}',
        colorMap: { core: 'primary', custom: 'default' },
      }),
      col('status', 'Active', {
        kind: 'chip',
        value: '{{row.is_active}}',
        colorMap: { true: 'success', false: 'default' },
      }),
    ],
  });

  // Tab 2 Components
  reg('eval_tester_widget', 'validationRules.RuleTesterPanel', {
    default_bo_name: '{{vars.selected_bo}}',
  });

  // Tab 3 Components
  reg('violations_widget', 'validationRules.ViolationsLiveViewer', {
    limit: 100,
  });

  return {
    name: 'Validation Rules & Rule Studio',
    slug: 'validation-rules',
    description: 'Centralized single-store validation catalog, portable AST bundles, and live evaluation engine.',
    version: 1,
    isCore: true,
    status: 'published',
    layout: layout('page_root', {
      page_root: { type: 'Column', children: ['rules_grid'], style: { gap: '16px' } },
    }),
    tabs,
    components,
    dataSources: [],
    presentationEvents: [],
    filterBar: layout('filter_root', {
      filter_root: { type: 'Column', children: ['header_row'], style: { gap: '12px' } },
      header_row: { type: 'Row', children: ['hdr', 'bo_select', 'domain_select'], style: { alignItems: 'center', gap: '8px', flexWrap: 'wrap' } },
    }),
    app: {
      chrome: 'none',
      surface: { maxWidth: 1400, padding: 3 },
      tabVariable: 'tab',
      variables: [
        { name: 'selected_bo', default: 'order', url: true },
        { name: 'domain_filter', default: '', url: true },
        { name: 'tab', default: 'catalog', url: true },
      ],
      queries: [
        {
          id: 'rules_list',
          operation: 'validationRules.list',
          params: {
            bo_name: '{{vars.selected_bo}}',
            domain: '{{vars.domain_filter}}',
          },
        },
      ],
    },
  };
}
