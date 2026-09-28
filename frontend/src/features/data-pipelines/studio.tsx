import React, { useEffect } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { type Definition, pipelinesApi } from './api';
import { NODE_META } from './PipelineNode';
import PipelineEditorPage from './PipelineEditorPage';

/**
 * The data-pipelines domain's Page Studio surface: the pipeline list as an
 * operation (the Data pipelines core page) and the visual pipeline editor as
 * a domain component (the Data pipeline editor core page). The editor is a
 * canvas - drag, connect, configure, preview, run - so it is placed whole,
 * with its declared contract, rather than approximated with generic widgets.
 */

/** "Sources → destinations", from the pipeline's own nodes. */
export function describePipeline(d: Definition): string {
  const src = d.spec.nodes.filter((n) => NODE_META[n.type]?.category === 'source').map((n) => n.label || n.type);
  const dst = d.spec.nodes.filter((n) => NODE_META[n.type]?.category === 'destination').map((n) => n.label || n.type);
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

// The editor refreshes its own list key on save; the list page's queries live
// under the domain prefix, so refresh those when the editor goes away.
function EditorView() {
  const qc = useQueryClient();
  useEffect(() => () => { qc.invalidateQueries({ queryKey: ['dp'] }); }, [qc]);
  return <PipelineEditorPage />;
}

registerDomainComponents([
  {
    id: 'dataPipelines.Editor', domain: 'dp', label: 'Pipeline editor',
    description: 'The visual pipeline editor for the pipeline in the route (/data/pipelines/:id; "new" starts a blank one): canvas, step settings, preview, assistant, runs. Fills the page.',
    inputs: [],
    events: [],
    render: EditorView,
  },
]);
