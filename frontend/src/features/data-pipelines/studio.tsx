import React from 'react';
import { registerOperations, type OperationDef } from '../../studio-core/operations/registry';
import { registerDomainComponents } from '../../studio-core/components/registry';
import { type Definition, type PreviewResult, type Spec, pipelinesApi } from './api';
import { AssistantPanel } from './AssistantPanel';
import { categoryOf } from './editorStudio';
import ScheduleEditor from '../schedules/ScheduleEditor';
import { useTargetSchedule } from '../schedules/useTargetSchedule';
import { schedulesApi } from '../schedules/api';

/**
 * The data-pipelines domain's Page Studio surface: the pipeline list (the
 * Data pipelines core page) and the editor's operations (editorStudio.ts -
 * the Data pipeline editor core page is built from studio blocks, its graph
 * on the Canvas widget). Two parts are still placed as domain components:
 * the AI assistant (a chat) and the pipeline's schedule (the shared
 * schedule editor).
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

registerOperations([{
  id: 'schedules.forTarget', domain: 'sched-list', kind: 'query', label: 'A target\'s schedule',
  description: 'The (at most one) schedule of a product target: scheduled (enabled), label for its button.',
  params: [{ name: 'kind', type: 'string', required: true }, { name: 'ref', type: 'string', required: true }],
  fields: [{ name: 'scheduled', type: 'boolean' }, { name: 'label' }],
  run: async (p) => {
    const ref = String(p.ref ?? '');
    if (!ref || ref === 'new') return { scheduled: false, label: 'Schedule' };
    const sc = (await schedulesApi.list({ kind: String(p.kind), ref })).schedules?.[0];
    return { scheduled: !!sc?.enabled, label: sc?.enabled ? 'Scheduled' : 'Schedule' };
  },
}]);

/** The shared schedule editor, for one target (kind + ref). */
function TargetSchedule({ inputs, emit }: { inputs: Record<string, unknown>; emit: (e: string) => void }) {
  const kind = String(inputs.kind ?? '');
  const ref = String(inputs.ref ?? '');
  const s = useTargetSchedule(kind, ref, !!ref && ref !== 'new');
  if (!inputs.open || !ref || ref === 'new' || s.isLoading) return null;
  return (
    <ScheduleEditor key={s.schedule?.id ?? 'new'} open schedule={s.schedule}
      fixedTarget={{ kind, ref, name: String(inputs.name ?? '') }} onClose={() => emit('close')} />
  );
}

registerDomainComponents([
  {
    id: 'dataPipelines.Assistant', domain: 'dp', label: 'Pipeline assistant',
    description: 'Describe a change in words; the assistant proposes a spec, applied on request. apply carries spec and focus (the first new step).',
    inputs: [{ name: 'spec', type: 'object', required: true }, { name: 'selected', type: 'string' }, { name: 'preview', type: 'object' }],
    events: [{ name: 'apply', payload: ['spec', 'focus'] }, { name: 'close' }],
    render: ({ inputs, emit }) => (
      <AssistantPanel spec={inputs.spec as Spec} selectedNodeId={(inputs.selected as string) || null} preview={(inputs.preview as PreviewResult) ?? null}
        onApply={(spec, focus) => emit('apply', { spec, focus: focus ?? null })} onClose={() => emit('close')} />
    ),
  },
  {
    id: 'schedules.TargetSchedule', domain: 'sched-list', label: 'Schedule a target', overlay: true,
    description: 'The schedule editor for one target (kind data_pipeline, ref its id).',
    inputs: [{ name: 'kind', type: 'string', required: true }, { name: 'ref', type: 'string', required: true }, { name: 'name', type: 'string' }, { name: 'open', type: 'boolean' }],
    events: [{ name: 'close' }],
    render: TargetSchedule,
  },
]);
