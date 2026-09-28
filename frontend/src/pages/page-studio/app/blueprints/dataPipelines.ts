import type { ComponentDefinition, CorePageDefinition, PageLayout, PageTab } from '../../../../types/pageStudio';
import type { ColumnDef } from '../appModel';

/**
 * Data pipelines and the pipeline editor as Page Studio pages (they replaced
 * the hand-built list and editor pages) - the ingest
 * step of mastering. The list is studio widgets over the dataPipelines.list
 * operation; the editor page places the domain's visual editor.
 */

type Spec = Record<string, { type: 'Row' | 'Column'; children: string[]; style?: Record<string, string> }>;
const layout = (root: string, spec: Spec): PageLayout => ({ root, nodes: Object.fromEntries(Object.entries(spec).map(([id, n]) => [id, { id, ...n }])) });
const fit = { flex: '0 0 auto' };

function listPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const components: Record<string, ComponentDefinition> = {
    hdr: {
      id: 'hdr', type: 'PageHeader', style: { flex: '1 1 320px' }, props: {
        icon: 'pipeline', title: 'Data pipelines',
        subtitle: 'Load files and business objects, check them against your rules, and write them to business objects, staging tables or files - no code.',
      },
    },
    new_btn: {
      id: 'new_btn', type: 'ActionButton', style: fit,
      props: { label: 'New pipeline', icon: 'add', variant: 'contained', onClick: [{ kind: 'navigate', to: '/data/pipelines/new' }] },
    },
    grid: {
      id: 'grid', type: 'DataGrid', props: {
        query: 'pipelines',
        emptyText: 'No pipelines yet. Start from a blank canvas with New pipeline, or describe what you want to the assistant.',
        onRowClick: [{ kind: 'navigate', to: '/data/pipelines/{{row.id}}' }],
        columns: [
          { id: 'name', header: 'Pipeline', cell: { kind: 'twoLine', primary: '{{row.name}}', secondary: '{{row.summary}}' } },
          { id: 'steps', header: 'Steps', align: 'right', cell: { kind: 'number', value: '{{row.steps}}' } },
          { id: 'edited', header: 'Edited', nowrap: true, cell: { kind: 'datetime', value: '{{row.last_modified_at}}' } },
          {
            id: 'act', align: 'right', nowrap: true, cell: {
              kind: 'actions', buttons: [
                { label: 'Open', onClick: [{ kind: 'navigate', to: '/data/pipelines/{{row.id}}' }] },
                {
                  label: 'Delete', color: 'error', onClick: [{
                    kind: 'runOperation', operation: 'dataPipelines.delete', params: { id: '{{row.id}}' },
                    confirm: { title: 'Delete "{{row.name}}"?', text: 'The pipeline is removed; its run history is kept.', confirmLabel: 'Delete' },
                    successMessage: 'Pipeline deleted',
                  }],
                },
              ],
            },
          },
        ] satisfies ColumnDef[],
      },
    },
  };
  const tabs: PageTab[] = [{
    id: 'pipelines', label: 'Pipelines',
    layout: layout('root', {
      root: { type: 'Column', children: ['top', 'grid'], style: { gap: '16px' } },
      top: { type: 'Row', children: ['hdr', 'new_btn'], style: { alignItems: 'center' } },
    }),
  }];
  return {
    name: 'Data pipelines',
    slug: 'data-pipelines',
    description: 'Every data pipeline: load files and business objects, check them, write them where mastering reads them. Built in Page Studio.',
    layout: tabs[0].layout,
    tabs,
    components,
    dataSources: [],
    status: 'draft' as const,
    app: {
      chrome: 'none' as const,
      surface: { maxWidth: 1200, padding: 3 },
      queries: [{ id: 'pipelines', operation: 'dataPipelines.list', params: {} }],
    },
  };
}

function editorPage(): Omit<CorePageDefinition, 'id' | 'createdAt' | 'updatedAt'> {
  const tabs: PageTab[] = [{
    id: 'editor', label: 'Editor',
    layout: layout('root', { root: { type: 'Column', children: ['editor'] } }),
  }];
  return {
    name: 'Data pipeline editor',
    slug: 'data-pipeline-editor',
    description: 'The visual pipeline editor (/data/pipelines/:id): canvas, step settings, preview, assistant and runs. Built in Page Studio around the domain editor.',
    layout: tabs[0].layout,
    tabs,
    components: {
      editor: { id: 'editor', type: 'DomainComponent', props: { component: 'dataPipelines.Editor', inputs: {}, events: {} } },
    },
    dataSources: [],
    status: 'draft' as const,
    // The editor paints its own surface and fills the height it is given.
    app: { chrome: 'none' as const, surface: { padding: 0 } },
  };
}

export const dataPipelinesBlueprint = () => structuredClone(listPage());
export const dataPipelineEditorBlueprint = () => structuredClone(editorPage());
