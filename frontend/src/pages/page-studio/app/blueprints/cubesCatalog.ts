import type { ComponentDefinition, CorePageDefinition, PageLayout } from '../../../../types/pageStudio';
import type { Action, ColumnDef, ConditionNode } from '../appModel';

/**
 * Cubes catalog (slug cubes-catalog): list/filter by scope, open designer,
 * deploy/refresh via cubes.* operations, Impact drawer (A3). Served at
 * /build/cubes via route_aliases (PR5).
 */

const op = (
  operation: string,
  params: Record<string, unknown>,
  onSuccess: Action[] = [],
  more: Record<string, unknown> = {},
): Action => ({ kind: 'runOperation', operation, params, onSuccess, ...more } as Action);
const set = (name: string, value: unknown = null): Action => ({ kind: 'setVariable', name, value });
const cond = (field: string, operator: string, value?: unknown): ConditionNode =>
  ({ type: 'condition', field, operator, value });
const col = (id: string, header: string, cell: ColumnDef['cell'] | ColumnDef['stack'], more: Partial<ColumnDef> = {}): ColumnDef =>
  Array.isArray(cell) ? { id, header, stack: cell, ...more } : { id, header, cell: cell as ColumnDef['cell'], ...more };

const fit = { flex: '0 0 auto' };

type NodeSpec = {
  type: 'Row' | 'Column' | 'Drawer';
  children: string[];
  style?: Record<string, string>;
  props?: Record<string, unknown>;
};
const layout = (root: string, spec: Record<string, NodeSpec>): PageLayout => ({
  root,
  nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])),
});

function cubesCatalogPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {};
  const w = (id: string, type: string, props: Record<string, unknown>, extra: Partial<ComponentDefinition> = {}) => {
    components[id] = { id, type, props, ...extra };
  };

  w(
    'hdr',
    'PageHeader',
    {
      icon: 'cube',
      title: 'Cubes',
      subtitle:
        'Published aggregation contracts — dimensions, governed metrics, grains, materialization, and impact.',
    },
    { style: { flex: '1 1 320px' } },
  );

  w(
    'scope',
    'VariableSelect',
    {
      variable: 'scope',
      label: 'Scope',
      minWidth: 160,
      options: [
        { value: 'all', label: 'All' },
        { value: 'core', label: 'Core' },
        { value: 'custom', label: 'Custom' },
        { value: 'adopted', label: 'Adopted' },
      ],
    },
    { style: fit },
  );

  w(
    'new_btn',
    'ActionButton',
    {
      label: 'New cube',
      icon: 'add',
      variant: 'contained',
      onClick: [{ kind: 'navigate', to: '/build/cubes/new' }],
    },
    { style: fit },
  );

  w('grid', 'DataGrid', {
    query: 'cubes',
    rowsPath: 'rows',
    rowKey: 'id',
    emptyText: '{{queries.cubes.data.empty_text}}',
    onRowClick: [{ kind: 'navigate', to: '/build/cubes/{{row.id}}' }],
    columns: [
      col('name', 'Cube', [
        { kind: 'text', value: '{{row.name}}', bold: true },
        { kind: 'text', value: '{{row.boId}}', caption: true },
      ]),
      col('contract', 'Contract', { kind: 'number', value: '{{row.contractVersion}}' }, { nowrap: true }),
      col(
        'core',
        'Core',
        {
          kind: 'chip',
          value: '{{row.isCore}}',
          label: '{{row.isCore}}',
          variant: 'outlined',
          colorMap: { true: 'primary', false: 'default', '*': 'default' },
        },
        { nowrap: true },
      ),
      col(
        'status',
        'Status',
        {
          kind: 'chip',
          value: '{{row.status}}',
          label: '{{row.status}}',
          variant: 'outlined',
          colorMap: { active: 'success', draft: 'default', '*': 'default' },
        },
        { nowrap: true },
      ),
      col('updated', 'Updated', { kind: 'datetime', value: '{{row.updatedAt}}' }, { nowrap: true }),
      col(
        'act',
        '',
        {
          kind: 'actions',
          buttons: [
            { label: 'Open', onClick: [{ kind: 'navigate', to: '/build/cubes/{{row.id}}' }] },
            {
              label: 'Impact',
              icon: 'insights',
              onClick: [
                set('impactCubeId', '{{row.id}}'),
                set('impactCubeName', '{{row.name}}'),
              ],
            },
            {
              label: 'Deploy',
              icon: 'play',
              onClick: [
                op(
                  'cubes.deploy',
                  { id: '{{row.id}}', force: false },
                  [],
                  {
                    confirm: {
                      title: 'Deploy "{{row.name}}"?',
                      text: 'Start materialization for this cube contract (force=false).',
                      confirmLabel: 'Deploy',
                    },
                    successMessage: 'Deploy started',
                  },
                ),
              ],
            },
            {
              label: 'Refresh',
              icon: 'refresh',
              onClick: [
                op(
                  'cubes.refresh',
                  { id: '{{row.id}}', force: false },
                  [],
                  {
                    confirm: {
                      title: 'Refresh "{{row.name}}"?',
                      text: 'Refresh materialization when already dual-committed (force=false may no-op).',
                      confirmLabel: 'Refresh',
                    },
                    successMessage: 'Refresh requested',
                  },
                ),
              ],
            },
          ],
        },
        { align: 'right', nowrap: true },
      ),
    ] satisfies ColumnDef[],
  });

  w('impact_dc', 'DomainComponent', {
    component: 'cubes.ImpactPanel',
    inputs: {
      cubeId: '{{vars.impactCubeId}}',
      title: 'Impact — {{vars.impactCubeName}}',
    },
  });

  const closeImpact = [set('impactCubeId'), set('impactCubeName')];

  const main = layout('page_root', {
    page_root: {
      type: 'Column',
      children: ['top', 'grid', 'impact_drawer'],
      style: { gap: '16px' },
    },
    top: {
      type: 'Row',
      children: ['hdr', 'scope', 'new_btn'],
      style: { alignItems: 'center', gap: '12px', flexWrap: 'wrap' },
    },
    impact_drawer: {
      type: 'Drawer',
      children: ['impact_dc'],
      props: {
        title: 'Cube impact',
        subtitle: '{{vars.impactCubeName}}',
        width: 720,
        openWhen: cond('vars.impactCubeId', 'is_not_empty'),
        onClose: closeImpact,
      },
    },
  });

  return {
    name: 'Cubes',
    slug: 'cubes-catalog',
    description:
      'Aggregation contracts: dimensions, governed metrics, grains, materialization, and impact. Built in Page Studio.',
    version: 2,
    isCore: true,
    status: 'published',
    layout: main,
    tabs: [],
    filterBar: layout('fb_root', { fb_root: { type: 'Column', children: [], style: { gap: '0px' } } }),
    components,
    dataSources: [],
    presentationEvents: [],
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1200, padding: 3 },
      variables: [
        { name: 'scope', default: 'all', url: true, description: 'Cube list scope filter' },
        { name: 'impactCubeId', description: 'Cube id open in the Impact drawer' },
        { name: 'impactCubeName', description: 'Cube name shown in the Impact drawer title' },
      ],
      queries: [
        {
          id: 'cubes',
          operation: 'cubes.list',
          params: { scope: '{{vars.scope}}', limit: 50 },
          keepPrevious: true,
        },
      ],
    },
  };
}

export const cubesCatalogBlueprint = () => structuredClone(cubesCatalogPage());
