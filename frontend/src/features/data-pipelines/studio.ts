import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { type Definition, pipelinesApi } from './api';
import { categoryOf } from './editorStudio';

/**
 * The data-pipelines domain's Page Studio surface: the pipeline list (the
 * Data pipelines core page) and the editor's operations (editorStudio.ts -
 * the Data pipeline editor core page is built from studio blocks, its graph
 * on the Canvas widget, the assistant on the Chat widget, the schedule
 * the studio schedule editor - features/schedules/studio.ts).
 */

/** "Sources → destinations", from the pipeline's own nodes. */
export function describePipeline(d: Definition): string {
  const src = d.spec.nodes.filter((n) => categoryOf(n.type) === 'source').map((n) => n.label || n.type);
  const dst = d.spec.nodes.filter((n) => categoryOf(n.type) === 'destination').map((n) => n.label || n.type);
  if (!src.length && !dst.length) return 'Empty pipeline';
  return `${src.join(', ') || '?'} → ${dst.join(', ') || '?'}`;
}

const operations: OperationDef[] = [
  {
    id: 'dataPipelines.list', domain: 'dp', kind: 'query', label: 'Data pipelines',
    description: 'Every pipeline definition; summary reads sources → destinations.',
    params: [],
    fields: [
      { name: 'id' }, { name: 'name' }, { name: 'summary' }, { name: 'steps', type: 'number' },
      { name: 'last_modified_at', type: 'datetime' },
    ],
    run: async () => (await pipelinesApi.list()).map((d) => ({
      ...d, summary: describePipeline(d), steps: d.spec.nodes.length,
    })),
  },
  {
    id: 'dataPipelines.delete', domain: 'dp', kind: 'mutation', label: 'Delete a pipeline',
    description: 'Run history is kept.',
    params: [{ name: 'id', type: 'string', required: true }],
    run: (p) => pipelinesApi.remove(String(p.id ?? '')),
  },
];

registerOperations(operations);
