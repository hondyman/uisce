import React from 'react';
import { beforeAll, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import { arrange } from './pipelineEditorScenario';
import { assistantScenario, START_SPEC } from './assistantScenario';

/**
 * The pipeline assistant, built in Page Studio on the Chat widget (the
 * conversation in a page variable, dataPipelines.assist doing the asking),
 * against what the hand-built AssistantPanel showed and sent for the same
 * conversation - recorded below before it was retired: the empty state and
 * starters, the request (message, spec, selected step, history, the last
 * preview's rejects), a proposal with its changes and remaining issues,
 * Apply (the spec and its first new step), and a failed request.
 */

class RO { observe() {} unobserve() {} disconnect() {} }
(globalThis as { ResizeObserver?: unknown }).ResizeObserver ??= RO;

const assist = vi.hoisted(() => vi.fn());
vi.mock('../../utils/apiClient', () => ({
  default: (url: string, init?: unknown) => (url.includes('/data-pipelines/assist') ? assist(url, init) : Promise.reject(new Error(`unmocked ${url}`))),
}));
const dp = vi.hoisted(() => ({ get: vi.fn(), nodeTypes: vi.fn(), validate: vi.fn(), stagingTables: vi.fn(), files: vi.fn(), preview: vi.fn(), runs: vi.fn(), run: vi.fn(), update: vi.fn(), suggestMapping: vi.fn(), create: vi.fn(), startRun: vi.fn(), profile: vi.fn(), upload: vi.fn() }));
const pf = vi.hoisted(() => ({ businessObjects: vi.fn(), boSchema: vi.fn(), rules: vi.fn() }));
vi.mock('../../features/data-pipelines/api', async (orig) => {
  const real = await orig<typeof import('../../features/data-pipelines/api')>();
  return { ...real, pipelinesApi: { ...real.pipelinesApi, ...dp }, platformApi: { ...real.platformApi, ...pf } };
});
const sched = vi.hoisted(() => ({ list: vi.fn() }));
vi.mock('../../features/schedules/api', async (orig) => {
  const real = await orig<typeof import('../../features/schedules/api')>();
  return { ...real, schedulesApi: { ...real.schedulesApi, ...sched } };
});
const mast = vi.hoisted(() => ({ profiles: vi.fn() }));
vi.mock('../../features/mastering/api', async (orig) => {
  const real = await orig<typeof import('../../features/mastering/api')>();
  return { ...real, masteringApi: { ...real.masteringApi, ...mast } };
});

import '../../studio-core/registerDomains';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { dataPipelineEditorBlueprint } from '../../pages/page-studio/app/blueprints/dataPipelines';

beforeAll(loadRuleEngine, 30000);

/** What the hand-built assistant showed and sent. */
const HAND_BUILT = {
  "empty": "PipelineassistantDescribewhatyouwanttoloadorchange.Ionlyusebusinessobjects,rules,filesandtablesthatexistforyou,andnothingchangesuntilyoupressApply.LoadtheuploadedFactSetfileintotheFundbusinessobject,updatingfundsthatalreadyexistAddastepthatappliesthefundvalidationrulesbeforewritingLoadthisfileintoastagingtableinsteadofabusinessobjectWhywererowsrejectedinthepreview?",
  "request": [
    "/api/data-pipelines/assist",
    {
      "method": "POST",
      "body": "{\"message\":\"Load the uploaded FactSet file into the Fund business object, updating funds that already exist\",\"spec\":{\"version\":1,\"nodes\":[{\"id\":\"file_1\",\"type\":\"file_source\",\"label\":\"Vendor file\",\"config\":{\"uri\":\"a.csv\",\"format\":\"csv\",\"columns\":[]}}],\"edges\":[]},\"selected_node_id\":\"file_1\",\"history\":[],\"preview_rejects\":[\"row 3 (file_1): bad date\"]}",
      "headers": {
        "Content-Type": "application/json"
      }
    }
  ],
  "proposal": "PipelineassistantLoadtheuploadedFactSetfileintotheFundbusinessobject,updatingfundsthatalreadyexistAddedastepthatwritesfunds.•AddedWritefunds•ConnectedVendorfile→WritefundsStillneedsattention:bo_sink_2:PickkeyfieldsApplytocanvas",
  "applied": {
    "nodes": [
      "file_1",
      "bo_sink_2"
    ],
    "focus": "bo_sink_2"
  },
  "appliedButtonDisabled": true,
  "request2": [
    "/api/data-pipelines/assist",
    {
      "method": "POST",
      "body": "{\"message\":\"Why were rows rejected?\",\"spec\":{\"version\":1,\"nodes\":[{\"id\":\"file_1\",\"type\":\"file_source\",\"label\":\"Vendor file\",\"config\":{\"uri\":\"a.csv\",\"format\":\"csv\",\"columns\":[]}}],\"edges\":[]},\"selected_node_id\":\"file_1\",\"history\":[{\"role\":\"user\",\"text\":\"Load the uploaded FactSet file into the Fund business object, updating funds that already exist\"},{\"role\":\"assistant\",\"text\":\"Added a step that writes funds.\"}],\"preview_rejects\":[\"row 3 (file_1): bad date\"]}",
      "headers": {
        "Content-Type": "application/json"
      }
    }
  ],
  "error": "PipelineassistantLoadtheuploadedFactSetfileintotheFundbusinessobject,updatingfundsthatalreadyexistAddedastepthatwritesfunds.•AddedWritefunds•ConnectedVendorfile→WritefundsStillneedsattention:bo_sink_2:PickkeyfieldsAppliedWhywererowsrejected?TheAIassistantisnotsetuphereyet-anadministratorcanconfigureitunderSystem>LLMConfig."
} as const;

type Req = [string, { method: string; body: string; headers: Record<string, string> }];
// The editor loads a pipeline with the default error policy (the hand-built editor did too); the recording
// handed the panel a bare spec, so compare without it.
const body = (r: unknown) => {
  const b = JSON.parse((r as Req)[1].body);
  delete b.spec.error_policy;
  return b;
};

describe('pipeline assistant built in Page Studio', () => {
  it('shows, sends, proposes, applies and fails as the hand-built assistant did', async () => {
    arrange({ dp, pf, sched, mast });
    dp.get.mockResolvedValue({ id: 'p1', name: 'Funds', description: '', spec: structuredClone(START_SPEC), created_by: 'u', created_at: '', last_modified_at: '' });
    dp.validate.mockResolvedValue({ valid: true, issues: [] });
    dp.preview.mockResolvedValue({ rejects: [{ node_id: 'file_1', row: 3, reason: 'bad date', kind: 'error' }], summary: { Nodes: [], RecordsIn: 3, RecordsOut: 2, Errors: 1 } });
    const bp = dataPipelineEditorBlueprint();
    const { container } = render(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <MemoryRouter initialEntries={['/data/pipelines/p1']}>
          <Routes>
            <Route path="/data/pipelines/:id" element={
              <RuntimePage name={bp.name} slug={bp.slug} tabs={bp.tabs!} components={bp.components} dataSources={[]} tenantId="t1" app={bp.app} />
            } />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>,
    );
    // As in the recording: the file step selected, a preview run (its rejects go to the assistant).
    await screen.findByText('Vendor file');
    fireEvent.click(container.querySelector('[data-id="file_1"]') as HTMLElement);
    await screen.findByDisplayValue('Vendor file');
    const preview = screen.getByRole('button', { name: /Preview/ });
    await waitFor(() => expect((preview as HTMLButtonElement).disabled).toBe(false));
    fireEvent.click(preview);
    await waitFor(() => expect(dp.preview).toHaveBeenCalled());
    fireEvent.click(screen.getByRole('button', { name: 'Assistant' }));

    const nodes = () => Array.from(container.querySelectorAll('.react-flow__node')).map((n) => (n as HTMLElement).dataset.id);
    const got = await assistantScenario(assist, () => ({ nodes: nodes(), focus: (container.querySelector('.react-flow__node.selected') as HTMLElement | null)?.dataset.id }));

    expect(got.empty).toBe(HAND_BUILT.empty);
    expect(body(got.request)).toEqual(body(HAND_BUILT.request));
    expect(got.proposal).toBe(HAND_BUILT.proposal);
    expect(got.applied).toEqual(HAND_BUILT.applied);
    expect(got.appliedButtonDisabled).toBe(true);
    // The second request: the same conversation. The page's spec and selection are now the applied ones, and
    // applying cleared the preview (the hand-built editor's update() did too) - the recording rendered the
    // panel alone, so its props did not follow the apply.
    const second = body(got.request2);
    const recorded = body(HAND_BUILT.request2);
    const rest = (b: Record<string, unknown>) => ({ ...b, spec: undefined, selected_node_id: undefined, preview_rejects: undefined });
    expect(rest(second)).toEqual(rest(recorded));
    expect(second.preview_rejects).toBeUndefined();
    expect(second.spec.nodes.map((n: { id: string }) => n.id)).toEqual(['file_1', 'bo_sink_2']);
    expect(second.selected_node_id).toBe('bo_sink_2');
    expect(got.error).toBe(HAND_BUILT.error);
  }, 60000);
});
