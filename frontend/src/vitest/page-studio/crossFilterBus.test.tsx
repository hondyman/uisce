import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import {
  CrossFilterBus,
  CrossFilterProvider,
  useCrossFilterBus,
} from '../../features/query-builder/utils/crossFilterBus';
import SavedQueryWidget from '../../pages/page-studio/SavedQueryWidget';
import { AppRuntimeProvider, useAppRuntime } from '../../pages/page-studio/app/AppRuntime';
import type { PageAppModel } from '../../pages/page-studio/app/appModel';

// Partial mock: keep the module's real pure helpers (the widget's roll-up gate
// imports isSafeToRollUpAcrossRows / isAdditiveSafe) and stub only the network calls.
vi.mock('../../features/query-builder/services/savedQueryApi', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../../features/query-builder/services/savedQueryApi')>()),
  runSavedQuery: vi.fn(),
  executeSavedQuery: vi.fn(),
}));

import { runSavedQuery } from '../../features/query-builder/services/savedQueryApi';

describe('CrossFilterBus - Unit Tests', () => {
  it('manages filter emission, toggle, self-exclusion, and term matching', () => {
    const bus = new CrossFilterBus();
    const listener = vi.fn();
    bus.subscribe(listener);

    // 1. Emit filter from widget-1 on term 'region'
    bus.emit('widget-1', 'region', 'North America');
    expect(bus.count).toBe(1);
    expect(listener).toHaveBeenCalledTimes(1);
    expect(bus.getAll()).toEqual([
      { sourceWidgetId: 'widget-1', termNodeId: 'region', value: 'North America' },
    ]);

    // 2. Self-exclusion: widget-1 asking for its own filters gets empty array
    const w1Filters = bus.getFiltersForWidget('widget-1', ['region', 'sales']);
    expect(w1Filters).toEqual([]);

    // 3. Term matching: widget-2 that queries 'region' receives the filter
    const w2Filters = bus.getFiltersForWidget('widget-2', ['region', 'sales']);
    expect(w2Filters).toEqual([
      { termNodeId: 'region', operator: 'eq', value: 'North America' },
    ]);

    // 4. Term matching: widget-3 that does NOT query 'region' receives nothing
    const w3Filters = bus.getFiltersForWidget('widget-3', ['account_id', 'balance']);
    expect(w3Filters).toEqual([]);

    // 5. Config mode 'emit' ignores incoming filters
    const w4Filters = bus.getFiltersForWidget('widget-4', ['region'], { mode: 'emit' });
    expect(w4Filters).toEqual([]);

    // 6. Config enabled: false ignores incoming filters
    const w5Filters = bus.getFiltersForWidget('widget-5', ['region'], { enabled: false });
    expect(w5Filters).toEqual([]);

    // 7. Toggle behavior: clicking same value again on widget-1 removes the filter
    bus.emit('widget-1', 'region', 'North America');
    expect(bus.count).toBe(0);
    expect(listener).toHaveBeenCalledTimes(2);
    expect(bus.getFiltersForWidget('widget-2', ['region'])).toEqual([]);

    // 8. Clear resets all filters
    bus.emit('widget-1', 'region', 'EMEA');
    bus.emit('widget-2', 'sector', 'Technology');
    expect(bus.count).toBe(2);
    bus.clear();
    expect(bus.count).toBe(0);
    expect(listener).toHaveBeenCalledTimes(5);
  });
});

describe('SavedQueryWidget - Cross-Filtering & Runtime Execution', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (runSavedQuery as any).mockImplementation((id: string, params: any, runtimeFilters: any) => {
      if (id === 'sq-slicer') {
        return Promise.resolve({
          columns: [{ name: 'region', type: 'string' }],
          rows: [{ region: 'US' }, { region: 'EMEA' }, { region: 'APAC' }],
          rowCount: 3,
          chartType: 'bar',
          name: 'Regions',
        });
      }
      return Promise.resolve({
        columns: [
          { name: 'region', type: 'string' },
          { name: 'sales', type: 'number' },
        ],
        rows: [{ region: 'US', sales: 1000 }],
        rowCount: 1,
        chartType: 'bar',
        name: 'Regional Sales',
      });
    });
  });

  it('emits cross-filter from slicer chip click and injects runtimeFilters into receiving widget', async () => {
    const bus = new CrossFilterBus();

    render(
      <CrossFilterProvider bus={bus}>
        <div>
          {/* Emitting slicer widget */}
          <SavedQueryWidget
            widgetId="slicer-widget"
            savedQueryId="sq-slicer"
            widgetType="slicer"
            crossFilterBus={bus}
          />
          {/* Receiving chart widget */}
          <SavedQueryWidget
            widgetId="chart-widget"
            savedQueryId="sq-chart"
            widgetType="gauge"
            queryTerms={['region', 'sales']}
            crossFilterBus={bus}
          />
        </div>
      </CrossFilterProvider>
    );

    // Initial render runs both queries with empty runtime filters
    await waitFor(() => {
      expect(runSavedQuery).toHaveBeenCalledWith('sq-slicer', {}, []);
      expect(runSavedQuery).toHaveBeenCalledWith('sq-chart', {}, []);
    });

    // Slicer renders chips for US, EMEA, APAC
    const emeaChip = await screen.findByText('EMEA');
    await userEvent.click(emeaChip);

    // Expect bus to hold EMEA filter from slicer-widget
    expect(bus.getAll()).toEqual([
      { sourceWidgetId: 'slicer-widget', termNodeId: 'region', value: 'EMEA' },
    ]);

    // Receiving widget should now re-fetch with runtimeFilters: [{ termNodeId: 'region', operator: 'eq', value: 'EMEA' }]
    await waitFor(() => {
      expect(runSavedQuery).toHaveBeenCalledWith(
        'sq-chart',
        {},
        [{ termNodeId: 'region', operator: 'eq', value: 'EMEA' }]
      );
    });

    // Clear filters chip is visible and clicking it clears cross-filters
    const clearChip = await screen.findAllByText(/Clear filters \(1\)/);
    expect(clearChip.length).toBeGreaterThan(0);
    await userEvent.click(clearChip[0]);

    // Bus is cleared
    expect(bus.count).toBe(0);
    await waitFor(() => {
      expect(runSavedQuery).toHaveBeenLastCalledWith('sq-chart', {}, []);
    });
  });

  it('composes multiple simultaneous emitters and leaves non-matching widgets untouched', async () => {
    const bus = new CrossFilterBus();

    render(
      <CrossFilterProvider bus={bus}>
        <div>
          {/* Slicer 1: emits region */}
          <SavedQueryWidget
            widgetId="slicer-region"
            savedQueryId="sq-slicer-region"
            widgetType="slicer"
            crossFilterBus={bus}
          />
          {/* Slicer 2: emits sector */}
          <SavedQueryWidget
            widgetId="slicer-sector"
            savedQueryId="sq-slicer-sector"
            widgetType="slicer"
            crossFilterBus={bus}
          />
          {/* Receiver: matches both region and sector */}
          <SavedQueryWidget
            widgetId="chart-composite"
            savedQueryId="sq-chart-comp"
            widgetType="gauge"
            queryTerms={['region', 'sector', 'revenue']}
            crossFilterBus={bus}
          />
          {/* Non-matching: has no region or sector terms */}
          <SavedQueryWidget
            widgetId="chart-untouched"
            savedQueryId="sq-chart-untouched"
            widgetType="gauge"
            queryTerms={['account_id', 'balance']}
            crossFilterBus={bus}
          />
        </div>
      </CrossFilterProvider>
    );

    // Initial load
    await waitFor(() => {
      expect(runSavedQuery).toHaveBeenCalledWith('sq-chart-comp', {}, []);
      expect(runSavedQuery).toHaveBeenCalledWith('sq-chart-untouched', {}, []);
    });

    const untouchedCallCountBefore = (runSavedQuery as any).mock.calls.filter((c: any) => c[0] === 'sq-chart-untouched').length;

    // 1. Emit filter 1 from region slicer
    act(() => {
      bus.emit('slicer-region', 'region', 'EMEA');
    });

    // 2. Emit filter 2 from sector slicer
    act(() => {
      bus.emit('slicer-sector', 'sector', 'Financials');
    });

    // Receiver should have received both filters composed
    await waitFor(() => {
      expect(runSavedQuery).toHaveBeenCalledWith('sq-chart-comp', {}, [
        { termNodeId: 'region', operator: 'eq', value: 'EMEA' },
        { termNodeId: 'sector', operator: 'eq', value: 'Financials' },
      ]);
    });

    // Non-matching widget should remain untouched with empty runtimeFilters
    const untouchedCallsAfter = (runSavedQuery as any).mock.calls.filter((c: any) => c[0] === 'sq-chart-untouched');
    const lastUntouchedCall = untouchedCallsAfter[untouchedCallsAfter.length - 1];
    expect(lastUntouchedCall[2]).toEqual([]);
  });

  it('composes hierarchical drill filters with incoming cross-filters', async () => {
    const bus = new CrossFilterBus();
    bus.emit('external-slicer', 'sector', 'Healthcare');

    render(
      <CrossFilterProvider bus={bus}>
        <SavedQueryWidget
          widgetId="drilled-chart"
          savedQueryId="sq-drilled"
          widgetType="gauge"
          queryTerms={['region', 'sector', 'revenue']}
          crossFilterBus={bus}
        />
      </CrossFilterProvider>
    );

    // Runs with incoming cross-filter on sector
    await waitFor(() => {
      expect(runSavedQuery).toHaveBeenCalledWith('sq-drilled', {}, [
        { termNodeId: 'sector', operator: 'eq', value: 'Healthcare' },
      ]);
    });
  });
});

describe('AppRuntime - CrossFilter Scope (Tab vs Page)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (runSavedQuery as any).mockResolvedValue({
      columns: [{ name: 'id', type: 'string' }],
      rows: [{ id: '1' }],
      rowCount: 1,
      chartType: 'bar',
      name: 'Query',
    });
  });

  function TabSwitchConsumer() {
    const { setVariable } = useAppRuntime();
    const bus = useCrossFilterBus();
    const [, setTick] = React.useState(0);

    React.useEffect(() => {
      return bus.subscribe(() => setTick((t) => t + 1));
    }, [bus]);

    return (
      <div>
        <div data-testid="filter-count">{bus.count}</div>
        <button onClick={() => bus.emit('w1', 'region', 'US')}>Emit Filter</button>
        <button onClick={() => setVariable('activeTab', 'tab-2')}>Switch Tab</button>
      </div>
    );
  }

  it('clears cross-filters on tab switch when crossFilterScope is tab', async () => {
    const app: PageAppModel = {
      tabVariable: 'activeTab',
      crossFilterScope: 'tab',
      variables: [{ name: 'activeTab', default: 'tab-1' }],
    };

    const qc = new QueryClient();

    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <AppRuntimeProvider app={app} mode="preview">
            <TabSwitchConsumer />
          </AppRuntimeProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );

    // Emit filter
    await userEvent.click(screen.getByRole('button', { name: 'Emit Filter' }));
    expect(screen.getByTestId('filter-count').textContent).toBe('1');

    // Switch tab
    await userEvent.click(screen.getByRole('button', { name: 'Switch Tab' }));

    // In 'tab' scope, switching tabs clears the bus
    await waitFor(() => {
      expect(screen.getByTestId('filter-count').textContent).toBe('0');
    });
  });

  it('preserves cross-filters on tab switch when crossFilterScope is page', async () => {
    const app: PageAppModel = {
      tabVariable: 'activeTab',
      crossFilterScope: 'page',
      variables: [{ name: 'activeTab', default: 'tab-1' }],
    };

    const qc = new QueryClient();

    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <AppRuntimeProvider app={app} mode="preview">
            <TabSwitchConsumer />
          </AppRuntimeProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );

    // Emit filter
    await userEvent.click(screen.getByRole('button', { name: 'Emit Filter' }));
    expect(screen.getByTestId('filter-count').textContent).toBe('1');

    // Switch tab
    await userEvent.click(screen.getByRole('button', { name: 'Switch Tab' }));

    // In 'page' scope, switching tabs keeps the cross-filter
    expect(screen.getByTestId('filter-count').textContent).toBe('1');
  });
});
