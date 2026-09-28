import React from 'react';
import { beforeAll, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { loadRuleEngine } from './loadRuleEngine';
import { registerOperations } from '../../studio-core/operations/registry';
import type { ComponentDefinition, PageLayout } from '../../types/pageStudio';
import type { PageAppModel } from '../../pages/page-studio/app/appModel';
import RuntimePage from '../../pages/page-studio/app/RuntimePage';
import { graphEdits } from '../../pages/page-studio/app/canvas';

/**
 * The Canvas widget: a graph held in a page variable, drawn from bindings
 * (title, subtitle, icon, category colour and handles, problems, chips), a
 * grouped palette whose items run onAdd with {{item}} / {{graph}} /
 * {{selected}}, selection in a variable, and pure graph edits (remove,
 * connect with one input per node, move).
 */

// React Flow measures its viewport; jsdom has no layout.
class RO { observe() {} unobserve() {} disconnect() {} }
(globalThis as { ResizeObserver?: unknown }).ResizeObserver ??= RO;

const kinds = [
  { type: 'file_source', label: 'Read a file', category: 'source', icon: 'file', available: true },
  { type: 'map', label: 'Map fields', category: 'step', icon: 'swap', available: true },
  { type: 'bo_sink', label: 'Write records', category: 'destination', icon: 'business', available: false },
];
const addStep = vi.fn(async (p: Record<string, unknown>) => {
  const g = p.graph as { nodes: { id: string }[]; edges: unknown[] };
  const item = p.item as { type: string; label: string };
  const id = `${item.type}_${g.nodes.length + 1}`;
  const edges = p.selected ? [...g.edges, { from: p.selected, to: id }] : g.edges;
  return { graph: { ...g, nodes: [...g.nodes, { id, type: item.type, label: item.label }], edges }, id };
});
registerOperations([
  { id: 'test.kinds', domain: 'test', kind: 'query', label: 'Kinds', params: [], run: async () => kinds },
  { id: 'test.issues', domain: 'test', kind: 'query', label: 'Issues', params: [], run: async () => ({ map_2: { issues: ['Map at least one field'], stats: [{ label: 'in 10' }] } }) },
  {
    id: 'test.addStep', domain: 'test', kind: 'mutation', label: 'Add a step', invalidates: [],
    params: [{ name: 'item', type: 'object' }, { name: 'graph', type: 'object' }, { name: 'selected', type: 'string' }], run: addStep,
  },
]);

beforeAll(loadRuleEngine, 30000);

const graph = {
  nodes: [
    { id: 'file_1', type: 'file_source', category: 'source', icon: 'file', label: 'Vendor file', summary: 'prices.csv · 4 columns', position: { x: 0, y: 0 } },
    { id: 'map_2', type: 'map', category: 'step', icon: 'swap', label: '', summary: '', position: { x: 300, y: 0 } },
  ],
  edges: [{ from: 'file_1', to: 'map_2' }],
};

function mount() {
  const layout: PageLayout = { root: 'root', nodes: { root: { id: 'root', type: 'Column', children: ['canvas', 'sel'] } } as PageLayout['nodes'] };
  const components: Record<string, ComponentDefinition> = {
    canvas: {
      id: 'canvas', type: 'Canvas', props: {
        variable: 'spec', selectedVariable: 'selected', singleInput: true, height: 400,
        node: {
          title: '{{node.label}}', subtitle: '{{node.summary}}', placeholder: 'Click to configure', icon: '{{node.icon}}',
          category: '{{node.category}}', errors: '{{extra.issues}}', chips: '{{extra.stats}}',
        },
        nodeData: '{{queries.issues.data}}',
        categories: { source: { color: 'info', inputs: false }, step: { color: 'secondary' }, destination: { color: 'success', outputs: false } },
        palette: {
          query: 'kinds', groupBy: 'category', label: '{{item.label}}', icon: '{{item.icon}}', category: '{{item.category}}',
          groups: [{ id: 'source', label: 'Read from' }, { id: 'step', label: 'Check & shape' }, { id: 'destination', label: 'Write to' }],
          disabledWhen: { type: 'condition', field: 'item.available', operator: 'is_false' },
        },
        onAdd: [{
          kind: 'runOperation', operation: 'test.addStep', params: { item: '{{item}}', graph: '{{graph}}', selected: '{{selected}}' },
          onSuccess: [{ kind: 'setVariable', name: 'spec', value: '{{result.graph}}' }, { kind: 'setVariable', name: 'selected', value: '{{result.id}}' }],
        }],
        emptyText: 'Start with a source on the left',
      },
    },
    sel: { id: 'sel', type: 'TextBlock', props: { text: 'selected:{{vars.selected}} edges:{{vars.spec.edges.length}} last:{{vars.spec.edges.1.from}}->{{vars.spec.edges.1.to}}' } },
  };
  const app: PageAppModel = {
    chrome: 'none',
    variables: [{ name: 'spec', default: graph }, { name: 'selected' }],
    queries: [{ id: 'kinds', operation: 'test.kinds', params: {} }, { id: 'issues', operation: 'test.issues', params: {} }],
  };
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}><MemoryRouter>
      <RuntimePage name="t" slug="t" tabs={[{ id: 'main', label: 'Main', layout }]} components={components} dataSources={[]} tenantId="t1" app={app} />
    </MemoryRouter></QueryClientProvider>,
  );
}

describe('Canvas widget', () => {
  it('draws the graph from bindings: titles, summaries or a placeholder, problems, chips, handles by category', async () => {
    const { container } = mount();
    expect(await screen.findByText('Vendor file')).toBeTruthy();
    expect(screen.getByText('prices.csv · 4 columns')).toBeTruthy();
    expect(screen.getByText('Click to configure')).toBeTruthy();
    // Extra data per node: the map step has a problem and a stats chip.
    expect(await screen.findByText('in 10')).toBeTruthy();
    expect(container.querySelectorAll('[data-testid="ErrorOutlineIcon"]').length).toBe(1);
    // A source takes no input; a step has both handles.
    const file = container.querySelector('[data-id="file_1"]') as HTMLElement;
    const step = container.querySelector('[data-id="map_2"]') as HTMLElement;
    expect(file.querySelectorAll('.react-flow__handle-left').length).toBe(0);
    expect(step.querySelectorAll('.react-flow__handle-left, .react-flow__handle-right').length).toBe(2);
    // Icons by name.
    expect(file.querySelector('[data-testid="InsertDriveFileIcon"]')).toBeTruthy();
    // (Edges draw once React Flow has measured the nodes - not in jsdom; the graph holds them.)
    expect(screen.getByText(/edges:1/)).toBeTruthy();
  }, 30000);

  it('selecting a node sets the variable; the palette is grouped and adds through the domain, chained after the selection', async () => {
    const { container } = mount();
    await screen.findByText('Vendor file');
    fireEvent.click(container.querySelector('[data-id="map_2"]') as HTMLElement);
    expect(await screen.findByText(/selected:map_2 /)).toBeTruthy();

    expect(screen.getByText('Read from')).toBeTruthy();
    expect(screen.getByText('Write to')).toBeTruthy();
    expect((screen.getByRole('button', { name: 'Write records' }) as HTMLElement).getAttribute('aria-disabled')).toBe('true');
    fireEvent.click(screen.getByRole('button', { name: 'Map fields' }));
    await waitFor(() => expect(addStep).toHaveBeenCalled());
    const args = addStep.mock.calls[0][0];
    expect(args.item).toMatchObject({ type: 'map' });
    expect(args.selected).toBe('map_2');
    expect((args.graph as typeof graph).nodes).toHaveLength(2);
    // The result becomes the graph and the new node is selected.
    expect(await screen.findByText('selected:map_3 edges:2 last:map_2->map_3')).toBeTruthy();
    // The new node is drawn (its title next to the palette item's).
    await waitFor(() => expect(screen.getAllByText('Map fields')).toHaveLength(2));
  }, 30000);

  it('shows the empty text for an empty graph', async () => {
    const layout: PageLayout = { root: 'root', nodes: { root: { id: 'root', type: 'Column', children: ['canvas'] } } as PageLayout['nodes'] };
    render(
      <QueryClientProvider client={new QueryClient()}><MemoryRouter>
        <RuntimePage name="t" slug="t" tabs={[{ id: 'main', label: 'Main', layout }]} dataSources={[]} tenantId="t1"
          components={{ canvas: { id: 'canvas', type: 'Canvas', props: { variable: 'spec', node: { title: '{{node.label}}' }, emptyText: 'Start with a source on the left' } } }}
          app={{ chrome: 'none', variables: [{ name: 'spec', default: { nodes: [], edges: [] } }] }} />
      </MemoryRouter></QueryClientProvider>,
    );
    expect(await screen.findByText('Start with a source on the left')).toBeTruthy();
  }, 30000);
});

describe('graph edits', () => {
  const shape = { nodesPath: 'nodes', edgesPath: 'edges', from: 'from', to: 'to' };
  const g = { nodes: [{ id: 'a' }, { id: 'b' }, { id: 'c' }], edges: [{ from: 'a', to: 'b' }, { from: 'b', to: 'c' }], other: 1 };
  it('removing a node removes its edges and keeps the rest of the graph', () => {
    expect(graphEdits.removeNodes(g, ['b'], shape)).toEqual({ nodes: [{ id: 'a' }, { id: 'c' }], edges: [], other: 1 });
  });
  it('connecting with one input per node replaces the old input; self-loops are ignored', () => {
    expect(graphEdits.connect(g, 'a', 'c', shape, true).edges).toEqual([{ from: 'a', to: 'b' }, { from: 'a', to: 'c' }]);
    expect(graphEdits.connect(g, 'a', 'c', shape, false).edges).toHaveLength(3);
    expect(graphEdits.connect(g, 'a', 'a', shape, true)).toBe(g);
  });
  it('removes edges by id and moves nodes', () => {
    expect(graphEdits.removeEdges(g, ['a->b'], shape).edges).toEqual([{ from: 'b', to: 'c' }]);
    expect((graphEdits.move(g, new Map([['a', { x: 5, y: 6 }]]), shape).nodes as unknown[])[0]).toEqual({ id: 'a', position: { x: 5, y: 6 } });
  });
  it('works with nested paths and other edge field names', () => {
    const nested = { spec: { steps: [{ id: 'x' }, { id: 'y' }], links: [{ src: 'x', dst: 'y' }] } };
    const s = { nodesPath: 'spec.steps', edgesPath: 'spec.links', from: 'src', to: 'dst' };
    expect(graphEdits.removeNodes(nested, ['x'], s)).toEqual({ spec: { steps: [{ id: 'y' }], links: [] } });
  });
});
