import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { act } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { LineageGraph } from '@/features/glossary/components/LineageGraph';

const focalTerm = {
  id: 'focal-1',
  node_name: 'Account Code',
  description: 'The unique identifier for an account in the system',
  qualified_path: 'business_term/Account Code',
  properties: { category: 'Finance' },
};

const upstreamBO = {
  id: 'bo-1',
  node_name: 'Account Object',
  description: 'BO description',
  qualified_path: 'business_object/Account',
  catalog_type_name: 'business_object',
  relLabel: 'member_of',
};

const upstreamCalc = {
  id: 'calc-1',
  node_name: 'Account Net',
  description: 'Calculated',
  qualified_path: 'calculated_term/Account_Net',
  catalog_type_name: 'calculated_term',
  relLabel: 'depends_on',
};

const upstreamStandard = {
  id: 'bt-1',
  node_name: 'Account Standard',
  description: 'Standard BT',
  qualified_path: 'business_term/Account_Standard',
  catalog_type_name: 'business_term',
  relLabel: 'describes',
};

const downstreamSemantic = {
  id: 'sem-1',
  node_name: 'Account ID',
  description: 'Semantic term',
  qualified_path: 'semantic_term/Account_ID',
};

const datasources = [
  {
    name: 'CRM Database',
    type: 'Postgres',
    host: 'crm.example.com',
    totalColumns: 1,
    totalTables: 1,
    schemas: {
      public: {
        name: 'public',
        tables: {
          accounts: {
            name: 'accounts',
            columns: [
              { id: 'col-1', name: 'account_code', dataType: 'varchar' },
            ],
          },
        },
      },
    },
  },
];

function renderLineage(props: any = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onNavigate = vi.fn();
  const utils = render(
    <QueryClientProvider client={qc}>
      <LineageGraph
        focalTerm={focalTerm}
        focalLabel="Business Term (Focal)"
        upstreamNodes={[upstreamBO, upstreamCalc, upstreamStandard]}
        downstreamNodes={[downstreamSemantic]}
        edges={[]}
        datasources={datasources}
        showDatasourceLayer
        onNavigate={onNavigate}
        height={400}
        {...props}
      />
    </QueryClientProvider>
  );
  return { ...utils, onNavigate };
}

describe('LineageGraph', () => {
  beforeEach(() => {
    // ReactFlow renders nodes via portals; clean up between tests
    document.body.innerHTML = '';
  });

  it('renders the focal term label', () => {
    renderLineage();
    expect(screen.getByText('Account Code')).toBeInTheDocument();
  });

  it('renders upstream business objects, calculations, and standard terms', () => {
    renderLineage();
    expect(screen.getByText('Account Object')).toBeInTheDocument();
    expect(screen.getByText('Account Net')).toBeInTheDocument();
    expect(screen.getByText('Account Standard')).toBeInTheDocument();
  });

  it('renders downstream semantic terms', () => {
    renderLineage();
    expect(screen.getByText('Account ID')).toBeInTheDocument();
  });

  it('renders datasources when the datasource layer is enabled', () => {
    renderLineage();
    expect(screen.getByText('CRM Database')).toBeInTheDocument();
  });

  it('hides datasources when showDatasourceLayer is false', () => {
    renderLineage({ showDatasourceLayer: false });
    expect(screen.queryByText('CRM Database')).not.toBeInTheDocument();
  });

  it('toggles the Business Objects layer on click', () => {
    renderLineage();
    const boToggle = screen.getByText(/Business Objects \(ON\)/);
    fireEvent.click(boToggle);
    expect(screen.getByText(/Business Objects \(OFF\)/)).toBeInTheDocument();
  });

  it('toggles the Dependencies (Calculations) layer on click', () => {
    renderLineage();
    const calcToggle = screen.getByText(/Dependencies \(ON\)/);
    fireEvent.click(calcToggle);
    expect(screen.getByText(/Dependencies \(OFF\)/)).toBeInTheDocument();
  });

  it('toggles the Datasources layer on click', () => {
    renderLineage();
    const dsToggle = screen.getByText(/Datasources \(ON\)/);
    fireEvent.click(dsToggle);
    expect(screen.getByText(/Datasources \(OFF\)/)).toBeInTheDocument();
  });

  it('renders the fullscreen toggle button', () => {
    renderLineage();
    expect(screen.getByText(/Fullscreen/)).toBeInTheDocument();
  });

  it('renders the Expand All Tables button when datasources are present', () => {
    renderLineage();
    expect(screen.getByText(/Expand All Tables/)).toBeInTheDocument();
  });

  it('expands a datasource when the Expand button is clicked', async () => {
    renderLineage();
    // The datasource node has a "▼ Expand" button (BO toggles say "Expand All Tables")
    const expandBtns = screen.getAllByText('▼ Expand');
    expect(expandBtns.length).toBeGreaterThan(0);
    fireEvent.click(expandBtns[0]);

    // After expansion the datasource collapses button should appear.
    // (ReactFlow renders the node internals in a portal that jsdom doesn't
    // expose via getByText, so we verify state via the toggle button label.)
    const collapseBtns = await screen.findAllByText('▲ Collapse');
    expect(collapseBtns.length).toBeGreaterThan(0);
  });

  it('calls onNavigate when a node is clicked', () => {
    const { onNavigate } = renderLineage();

    // The LineageGraph wires each upstream node's onClick to its onSelectNode,
    // which forwards to onNavigate. Click the ℹ️ button on the BO first to open
    // the inspector and confirm the callback was wired (inspector is the visible
    // proof that the upstream node rendered correctly).
    const infoButtons = screen.getAllByText('ℹ️');
    fireEvent.click(infoButtons[0]);
    expect(screen.getByText('Node Inspector')).toBeInTheDocument();

    // Click the first "Focus 🔍" button (each upstream node has one; click the
    // BO first). The onSelectNode handler forwards to onNavigate.
    const focusButtons = screen.getAllByText('Focus 🔍');
    fireEvent.click(focusButtons[0]);
    expect(onNavigate).toHaveBeenCalledWith('bo-1');
  });

  it('opens the node inspector drawer when the ℹ️ button is clicked', () => {
    renderLineage();
    // Each node has an ℹ️ button. Click the first one.
    const infoButtons = screen.getAllByText('ℹ️');
    fireEvent.click(infoButtons[0]);
    expect(screen.getByText('Node Inspector')).toBeInTheDocument();
  });

  it('renders an empty state when no nodes are provided', () => {
    renderLineage({
      upstreamNodes: [],
      downstreamNodes: [],
      datasources: undefined,
      edges: [],
    });
    expect(screen.getByText(/No upstream or downstream nodes/)).toBeInTheDocument();
  });
});
