import { describe, it, expect, beforeEach, vi } from 'vitest';
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { LayoutManager, DetachedWindowInfo } from '../../services/docking/LayoutManager';
import { UniversalWorkspaceHub } from '../../components/docking/UniversalWorkspaceHub';
import { PanelErrorBoundary } from '../../components/docking/PanelErrorBoundary';
import { useFdc3 } from '../../services/fdc3/useFdc3';
import { fdc3Agent } from '../../services/fdc3/Fdc3DesktopAgent';
import { MemoryRouter } from 'react-router-dom';

// Mock dockview-react component
vi.mock('dockview-react', () => {
  return {
    DockviewReact: (props: any) => {
      const readyRef = React.useRef(false);
      React.useEffect(() => {
        if (!readyRef.current && props.onReady) {
          readyRef.current = true;
          props.onReady({
            api: {
              fromJSON: vi.fn(),
              toJSON: () => ({ activeGroup: 'group-1' }),
              addPanel: vi.fn((opts: any) => ({ id: opts.id })),
              clear: vi.fn(),
            },
          });
        }
      }, []);
      return <div data-testid="mock-dockview-container">Dockview Container</div>;
    },
  };
});

// Mock PageBrowser StandalonePageRenderer
vi.mock('../../pages/PageBrowser', () => {
  return {
    StandalonePageRenderer: ({ slug }: { slug: string }) => (
      <div data-testid={`page-renderer-${slug}`}>Page Renderer: {slug}</div>
    ),
  };
});

// Mock child components
vi.mock('../../components/AIPortfolioRebalancer', () => ({
  default: () => <div data-testid="rebalancer-mock">Rebalancer Mock</div>,
}));

vi.mock('../../components/ScenarioAnalysisPro', () => ({
  default: () => <div data-testid="scenario-mock">Scenario Mock</div>,
}));

vi.mock('../../components/FixedIncomeDashboard', () => ({
  default: () => <div data-testid="fixed-income-mock">Fixed Income Mock</div>,
}));

describe('LayoutManager & Universal Workspace Hub', () => {
  let layoutMgr: LayoutManager;

  beforeEach(() => {
    localStorage.clear();
    layoutMgr = new LayoutManager();
  });

  describe('LayoutManager Persistence', () => {
    it('saves and loads docking layout and detached windows', () => {
      const mockLayout = { panels: ['orders', 'rebalancer'] };
      const win1: DetachedWindowInfo = {
        windowId: 'win_orders',
        route: '/view/page/orders',
        title: 'OMS Orders',
        targetScreenIndex: 0,
        width: 1400,
        height: 900,
      };

      layoutMgr.registerDetachedWindow(win1);
      layoutMgr.saveLayout(mockLayout);

      const loaded = layoutMgr.loadLayout();
      expect(loaded).not.toBeNull();
      expect(loaded?.dockviewLayout).toEqual(mockLayout);
      expect(loaded?.detachedWindows).toHaveLength(1);
      expect(loaded?.detachedWindows[0].windowId).toBe('win_orders');
    });

    it('unregisters detached windows and updates persistent storage', () => {
      layoutMgr.registerDetachedWindow({
        windowId: 'win_rebalancer',
        route: '/view/rebalancer',
        title: 'Rebalancer',
        targetScreenIndex: 1,
        width: 1280,
        height: 800,
      });

      expect(layoutMgr.getDetachedWindows()).toHaveLength(1);

      layoutMgr.unregisterDetachedWindow('win_rebalancer');
      expect(layoutMgr.getDetachedWindows()).toHaveLength(0);

      const loaded = layoutMgr.loadLayout();
      expect(loaded?.detachedWindows).toHaveLength(0);
    });

    it('clears layout and detached window state completely', () => {
      layoutMgr.registerDetachedWindow({
        windowId: 'win_1',
        route: '/view/page/orders',
        title: 'Test',
        targetScreenIndex: 0,
        width: 1000,
        height: 800,
      });

      layoutMgr.clearLayout();
      expect(layoutMgr.getDetachedWindows()).toHaveLength(0);
      expect(layoutMgr.loadLayout()).toBeNull();
    });
  });

  describe('UniversalWorkspaceHub Component', () => {
    it('renders workspace title, platform mode, and dockview container', async () => {
      render(
        <MemoryRouter>
          <UniversalWorkspaceHub />
        </MemoryRouter>
      );

      expect(screen.getByText('UNIVERSAL WORKSPACE')).toBeDefined();
      expect(screen.getByText('Web Browser')).toBeDefined();
      expect(screen.getByTestId('mock-dockview-container')).toBeDefined();
      expect(screen.getByText(/Distribute to Multi-Monitor/i)).toBeDefined();
      expect(screen.getByText('+ Orders')).toBeDefined();
      expect(screen.getByText('+ Rebalancer')).toBeDefined();
      expect(screen.getByText('+ Scenario')).toBeDefined();
      expect(screen.getByText('+ Fixed Income')).toBeDefined();
    });

    it('triggers save layout action when clicking Save button', async () => {
      render(
        <MemoryRouter>
          <UniversalWorkspaceHub />
        </MemoryRouter>
      );

      const saveBtn = screen.getByRole('button', { name: /save/i });
      fireEvent.click(saveBtn);

      await waitFor(() => {
        expect(screen.getByText(/Workspace layout saved/i)).toBeDefined();
      });
    });

    it('triggers reset layout action when clicking Reset button', async () => {
      render(
        <MemoryRouter>
          <UniversalWorkspaceHub />
        </MemoryRouter>
      );

      const resetBtn = screen.getByRole('button', { name: /reset/i });
      fireEvent.click(resetBtn);

      await waitFor(() => {
        expect(screen.getByText(/Layout reset to default/i)).toBeDefined();
      });
    });
  });

  describe('PanelErrorBoundary', () => {
    it('renders children when no error occurs', () => {
      render(
        <PanelErrorBoundary panelTitle="Safe Panel">
          <div data-testid="safe-child">Normal content</div>
        </PanelErrorBoundary>
      );
      expect(screen.getByTestId('safe-child')).toBeDefined();
    });

    it('renders error card when child component throws', () => {
      const Bomb = () => {
        throw new Error('Component boom!');
      };
      const spy = vi.spyOn(console, 'error').mockImplementation(() => {});

      render(
        <PanelErrorBoundary panelTitle="Crashing Panel">
          <Bomb />
        </PanelErrorBoundary>
      );

      expect(screen.getByText(/Panel Error: Crashing Panel/i)).toBeDefined();
      expect(screen.getByText(/Component boom!/i)).toBeDefined();
      expect(screen.getByRole('button', { name: /Reload Panel/i })).toBeDefined();
      spy.mockRestore();
    });
  });

  describe('useFdc3 Hook', () => {
    it('subscribes to context changes and broadcasts without feedback loops', async () => {
      let receivedCtx: any = null;
      const TestSubscriber = () => {
        const { activeChannel, broadcast } = useFdc3('fdc3.instrument', (ctx) => {
          receivedCtx = ctx;
        });
        return (
          <div>
            <span data-testid="active-channel">{activeChannel}</span>
            <button
              data-testid="broadcast-btn"
              onClick={() => broadcast({ type: 'fdc3.instrument', id: { ticker: 'NVDA' } })}
            >
              Broadcast
            </button>
          </div>
        );
      };

      render(<TestSubscriber />);
      expect(screen.getByTestId('active-channel').textContent).toBe('blue');

      // 1. In-window user broadcast
      fireEvent.click(screen.getByTestId('broadcast-btn'));
      expect(receivedCtx?.id?.ticker).toBe('NVDA');

      // 2. Direct external context update
      fdc3Agent.broadcast({
        type: 'fdc3.instrument',
        id: { ticker: 'MSFT' },
        name: 'Microsoft Corp.',
      });
      expect(receivedCtx?.id?.ticker).toBe('MSFT');
    });
  });
});
