import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import type { Action, CellSpec, ColumnDef, ConditionNode } from '../appModel';
import '../../../../features/catalog/studio';

const cond = (field: string, operator: string, value?: unknown): ConditionNode => ({ type: 'condition', field, operator, value });
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });

type Spec = Record<string, { type: 'Row' | 'Column' | 'Dialog'; children: string[]; style?: Record<string, string>; props?: Record<string, unknown> }>;
const layout = (root: string, spec: Spec): PageLayout => ({ root, nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])) });
const col = (id: string, header: string, cell: CellSpec | CellSpec[], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell, ...more };

export function catalogNodeTypesBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    header: {
      id: 'header',
      type: 'PageHeader',
      props: {
        icon: 'schema',
        title: 'Node Types',
        subtitle: 'Metadata structures and node types in the catalog graph',
      },
    },
    search: {
      id: 'search',
      type: 'SearchInput',
      props: { variable: 'search', placeholder: 'Search node types...' },
      style: { flex: '1 1 300px' },
    },
    create_btn: {
      id: 'create_btn',
      type: 'ActionButton',
      props: {
        label: 'New Node Type',
        variant: 'contained',
        onClick: [set('createOpen', true)],
      },
      style: { flex: '0 0 auto' },
    },
    grid: {
      id: 'grid',
      type: 'DataGrid',
      props: {
        query: 'nodeTypes',
        rowKey: 'id',
        enableSearch: true,
        searchVariable: 'search',
        searchFields: ['catalog_type_name', 'description', 'type'],
        columns: [
          col('name', 'Catalog Type Name', { kind: 'text', value: '{{row.catalog_type_name}}' }),
          col('type', 'Origin', {
            kind: 'chip',
            value: '{{row.type}}',
            colorMap: { core: 'primary', custom: 'success' },
          }),
          col('desc', 'Description', { kind: 'text', value: '{{row.description}}' }),
          col('status', 'Status', {
            kind: 'chip',
            value: '{{row.is_active}}',
            labelKey: '{{row.is_active ? "Active" : "Inactive"}}',
            colorMap: { true: 'success', false: 'default' },
          }),
          col('act', '', {
            kind: 'actions',
            buttons: [
              {
                label: 'Delete',
                color: 'error',
                visibleWhen: cond('row.type', 'equals', 'custom'),
                onClick: [{
                  kind: 'runOperation',
                  operation: 'catalog.deleteNodeType',
                  params: { id: '{{row.id}}' },
                  successMessage: 'Node type deleted',
                }],
              },
            ],
          }, { align: 'right' }),
        ],
      },
    },
    create_form: {
      id: 'create_form',
      type: 'Form',
      props: {
        variable: 'createForm',
        fields: [
          { name: 'catalog_type_name', label: 'Node Type Name', kind: 'text', required: true },
          { name: 'description', label: 'Description', kind: 'multiline', required: false },
        ],
        submitLabel: 'Create',
        onSubmit: [
          {
            kind: 'runOperation',
            operation: 'catalog.createNodeType',
            params: {
              catalog_type_name: '{{form.catalog_type_name}}',
              description: '{{form.description}}',
            },
            successMessage: 'Node type created successfully',
            onSuccess: [set('createOpen', false)],
          },
        ],
      },
    },
  };

  const l = layout('root', {
    root: { type: 'Column', children: ['controls_row', 'grid', 'create_dialog'], style: { gap: '16px' } },
    controls_row: { type: 'Row', children: ['search', 'create_btn'], style: { alignItems: 'center', gap: '12px' } },
    create_dialog: {
      type: 'Dialog',
      children: ['create_form'],
      props: {
        title: 'Create Custom Node Type',
        openWhen: cond('vars.createOpen', 'is_true'),
        onClose: [set('createOpen', false)],
      },
    },
  });

  const fb = layout('fb_root', {
    fb_root: { type: 'Row', children: ['header'], style: { alignItems: 'center', width: '100%' } },
  });

  return {
    name: 'Node Types',
    slug: 'catalog-node-types',
    description: 'Metadata structures and node types in the catalog graph',
    version: 1,
    isCore: true,
    status: 'published',
    layout: l,
    filterBar: fb,
    tabs: [],
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none',
      surface: { maxWidth: 1400, padding: 3 },
      variables: [
        { name: 'search', default: '', url: true },
        { name: 'createOpen', default: false },
        { name: 'createForm', default: { catalog_type_name: '', description: '' } },
      ],
      queries: [
        { id: 'nodeTypes', operation: 'catalog.listNodeTypes' },
      ],
    },
  };
}

export function catalogEdgeTypesBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    header: {
      id: 'header',
      type: 'PageHeader',
      props: {
        icon: 'accountTree',
        title: 'Edge Types',
        subtitle: 'Relationship types and connections in the catalog graph',
      },
    },
    search: {
      id: 'search',
      type: 'SearchInput',
      props: { variable: 'search', placeholder: 'Search edge types...' },
      style: { flex: '1 1 300px' },
    },
    grid: {
      id: 'grid',
      type: 'DataGrid',
      props: {
        query: 'edgeTypes',
        rowKey: 'id',
        enableSearch: true,
        searchVariable: 'search',
        searchFields: ['edge_type_name', 'description', 'type'],
        columns: [
          col('name', 'Edge Type Name', { kind: 'text', value: '{{row.edge_type_name}}' }),
          col('type', 'Origin', {
            kind: 'chip',
            value: '{{row.type}}',
            colorMap: { core: 'primary', custom: 'success' },
          }),
          col('desc', 'Description', { kind: 'text', value: '{{row.description}}' }),
          col('status', 'Status', {
            kind: 'chip',
            value: '{{row.is_active}}',
            labelKey: '{{row.is_active ? "Active" : "Inactive"}}',
            colorMap: { true: 'success', false: 'default' },
          }),
        ],
      },
    },
  };

  const l = layout('root', {
    root: { type: 'Column', children: ['controls_row', 'grid'], style: { gap: '16px' } },
    controls_row: { type: 'Row', children: ['search'], style: { alignItems: 'center', gap: '12px' } },
  });

  const fb = layout('fb_root', {
    fb_root: { type: 'Row', children: ['header'], style: { alignItems: 'center', width: '100%' } },
  });

  return {
    name: 'Edge Types',
    slug: 'catalog-edge-types',
    description: 'Relationship types and connections in the catalog graph',
    version: 1,
    isCore: true,
    status: 'published',
    layout: l,
    filterBar: fb,
    tabs: [],
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none',
      surface: { maxWidth: 1400, padding: 3 },
      variables: [
        { name: 'search', default: '', url: true },
      ],
      queries: [
        { id: 'edgeTypes', operation: 'catalog.listEdgeTypes' },
      ],
    },
  };
}

export function abbreviationsBlueprint(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    header: {
      id: 'header',
      type: 'PageHeader',
      props: {
        icon: 'textFields',
        title: 'Abbreviations',
        subtitle: 'Standard business abbreviations and their full semantic expansions',
      },
    },
    search: {
      id: 'search',
      type: 'SearchInput',
      props: { variable: 'search', placeholder: 'Search abbreviations...' },
      style: { flex: '1 1 300px' },
    },
    grid: {
      id: 'grid',
      type: 'DataGrid',
      props: {
        query: 'abbreviations',
        rowKey: 'id',
        enableSearch: true,
        searchVariable: 'search',
        searchFields: ['abbreviation', 'expansion', 'domain'],
        columns: [
          col('abbrev', 'Abbreviation', { kind: 'text', value: '{{row.abbreviation}}' }),
          col('expansion', 'Expansion', { kind: 'text', value: '{{row.expansion}}' }),
          col('domain', 'Domain', { kind: 'chip', value: '{{row.domain}}' }),
          col('status', 'Status', {
            kind: 'chip',
            value: '{{row.is_active}}',
            labelKey: '{{row.is_active ? "Active" : "Inactive"}}',
            colorMap: { true: 'success', false: 'default' },
          }),
        ],
      },
    },
  };

  const l = layout('root', {
    root: { type: 'Column', children: ['controls_row', 'grid'], style: { gap: '16px' } },
    controls_row: { type: 'Row', children: ['search'], style: { alignItems: 'center', gap: '12px' } },
  });

  const fb = layout('fb_root', {
    fb_root: { type: 'Row', children: ['header'], style: { alignItems: 'center', width: '100%' } },
  });

  return {
    name: 'Abbreviations',
    slug: 'core-abbreviations',
    description: 'Standard business abbreviations and their full semantic expansions',
    version: 1,
    isCore: true,
    status: 'published',
    layout: l,
    filterBar: fb,
    tabs: [],
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none',
      surface: { maxWidth: 1400, padding: 3 },
      variables: [
        { name: 'search', default: '', url: true },
      ],
      queries: [
        { id: 'abbreviations', operation: 'catalog.listAbbreviations' },
      ],
    },
  };
}
