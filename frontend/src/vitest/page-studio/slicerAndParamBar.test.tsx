import React, { useState } from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { SlicerWidget } from '../../pages/page-studio/SlicerWidget';
import { ReportParameterBar } from '../../pages/page-studio/ReportParameterBar';
import { GridLayoutRenderer } from '../../pages/page-studio/GridLayoutRenderer';
import { CrossFilterProvider, useCrossFilterBus, useActiveCrossFilters } from '../../features/query-builder/utils/crossFilterBus';
import type {
  ComponentDefinition,
  PageGridLayout,
  SlicerTileConfig,
  ReportParameterBarConfig,
} from '../../types/pageStudio';
import * as savedQueryApi from '../../features/query-builder/services/savedQueryApi';

vi.mock('../../features/query-builder/services/savedQueryApi', async () => {
  const actual = await vi.importActual('../../features/query-builder/services/savedQueryApi');
  return {
    ...actual,
    batchExecuteSavedQueries: vi.fn(),
  };
});

// Helper component to inspect crossFilterBus state
const BusInspector: React.FC = () => {
  const bus = useCrossFilterBus();
  const active = useActiveCrossFilters(bus);
  return (
    <div data-testid="bus-inspector">
      {active.map((f) => (
        <span key={f.termNodeId} data-testid={`filter-${f.termNodeId}`}>
          {f.termNodeId}:{f.operator}:{JSON.stringify(f.value)}
        </span>
      ))}
    </div>
  );
};

describe('SlicerWidget - Categorical, Multi-select, DateRange & Cascading', () => {
  const sampleData = {
    rows: [
      { region: 'North America', country: 'USA' },
      { region: 'North America', country: 'Canada' },
      { region: 'Europe', country: 'UK' },
      { region: 'Europe', country: 'Germany' },
      { region: 'APAC', country: 'Japan' },
    ],
    columns: [
      { name: 'region', type: 'string' },
      { name: 'country', type: 'string' },
    ],
  };

  it('renders dropdown slicer and emits single EQ filter on selection', () => {
    const config: SlicerTileConfig = {
      dimensionAlias: 'region',
      termNodeId: 'term-region',
      display: 'dropdown',
    };

    render(
      <CrossFilterProvider>
        <SlicerWidget id="slicer-region" title="Region Slicer" config={config} data={sampleData} />
        <BusInspector />
      </CrossFilterProvider>
    );

    expect(screen.getAllByText('Region Slicer').length).toBeGreaterThan(0);

    // Select "Europe"
    const select = screen.getByRole('combobox');
    fireEvent.mouseDown(select);

    const option = screen.getByRole('option', { name: 'Europe' });
    fireEvent.click(option);

    expect(screen.getByTestId('filter-term-region').textContent).toBe('term-region:eq:"Europe"');
  });

  it('renders multi-select slicer and emits IN filter with array value on selection', () => {
    const config: SlicerTileConfig = {
      dimensionAlias: 'region',
      termNodeId: 'term-region',
      display: 'multiSelect',
    };

    render(
      <CrossFilterProvider>
        <SlicerWidget id="slicer-region" title="Region Multi" config={config} data={sampleData} />
        <BusInspector />
      </CrossFilterProvider>
    );

    const select = screen.getByRole('combobox');
    fireEvent.mouseDown(select);

    // Click "North America" and "APAC"
    fireEvent.click(screen.getByRole('option', { name: 'North America' }));
    fireEvent.click(screen.getByRole('option', { name: 'APAC' }));

    expect(screen.getByTestId('filter-term-region').textContent).toBe(
      'term-region:in:["North America","APAC"]'
    );

    // Close select popup by pressing Escape
    fireEvent.keyDown(screen.getByRole('listbox'), { key: 'Escape', code: 'Escape' });

    // Clear selections
    const clearBtn = screen.getByRole('button', { name: /clear/i });
    fireEvent.click(clearBtn);

    expect(screen.queryByTestId('filter-term-region')).toBeNull();
  });

  it('renders list checklist slicer and updates filters on toggle', () => {
    const config: SlicerTileConfig = {
      dimensionAlias: 'country',
      termNodeId: 'term-country',
      display: 'list',
    };

    render(
      <CrossFilterProvider>
        <SlicerWidget id="slicer-country" title="Country Checklist" config={config} data={sampleData} />
        <BusInspector />
      </CrossFilterProvider>
    );

    // Click USA
    fireEvent.click(screen.getByText('USA'));
    expect(screen.getByTestId('filter-term-country').textContent).toBe('term-country:eq:"USA"');

    // Click Canada -> emits IN filter
    fireEvent.click(screen.getByText('Canada'));
    expect(screen.getByTestId('filter-term-country').textContent).toBe('term-country:in:["USA","Canada"]');
  });

  it('routes dateRange slicer to page variables (paramBinding) and NOT to crossFilterBus', () => {
    const onUpdateVar = vi.fn();
    const config: SlicerTileConfig = {
      dimensionAlias: 'orderDate',
      termNodeId: 'term-order-date',
      display: 'dateRange',
      paramBinding: { varName: 'reportDateRange' },
    };

    render(
      <CrossFilterProvider>
        <SlicerWidget
          id="slicer-date"
          title="Date Range"
          config={config}
          onUpdatePageVar={onUpdateVar}
        />
        <BusInspector />
      </CrossFilterProvider>
    );

    const startDateInput = screen.getByLabelText('Start Date');
    const endDateInput = screen.getByLabelText('End Date');

    fireEvent.change(startDateInput, { target: { value: '2026-01-01' } });
    fireEvent.change(endDateInput, { target: { value: '2026-03-31' } });

    expect(onUpdateVar).toHaveBeenCalledWith('reportDateRange', {
      start: '2026-01-01',
      end: '2026-03-31',
    });

    // CrossFilterBus must remain clean (no continuous range values in cross filter bus)
    expect(screen.queryByTestId('filter-term-order-date')).toBeNull();
  });

  it('shows authoring warning in design mode when dateRange lacks a paramBinding', () => {
    const config: SlicerTileConfig = {
      dimensionAlias: 'orderDate',
      termNodeId: 'term-order-date',
      display: 'dateRange',
      // paramBinding missing intentionally
    };

    render(
      <CrossFilterProvider>
        <SlicerWidget
          id="slicer-date-invalid"
          title="Date Range"
          config={config}
          mode="design"
        />
      </CrossFilterProvider>
    );

    expect(screen.getByText('Param Required')).toBeDefined();
  });

  it('cascading slicer scopes its member list when upstream slicer emits a filter', () => {
    const regionConfig: SlicerTileConfig = {
      dimensionAlias: 'region',
      termNodeId: 'term-region',
      display: 'dropdown',
    };

    const countryConfig: SlicerTileConfig = {
      dimensionAlias: 'country',
      termNodeId: 'term-country',
      display: 'dropdown',
      cascadingFrom: ['slicer-region'], // Cascading from region slicer
    };

    render(
      <CrossFilterProvider>
        <SlicerWidget id="slicer-region" title="Region" config={regionConfig} data={sampleData} />
        <SlicerWidget id="slicer-country" title="Country" config={countryConfig} data={sampleData} />
      </CrossFilterProvider>
    );

    // Upstream region emits "Europe"
    const regionSelect = screen.getAllByRole('combobox')[0];
    fireEvent.mouseDown(regionSelect);
    fireEvent.click(screen.getByRole('option', { name: 'Europe' }));

    // Downstream country dropdown should only show Europe countries (UK, Germany)
    const countrySelect = screen.getAllByRole('combobox')[1];
    fireEvent.mouseDown(countrySelect);

    expect(screen.getByRole('option', { name: 'UK' })).toBeDefined();
    expect(screen.getByRole('option', { name: 'Germany' })).toBeDefined();
    expect(screen.queryByRole('option', { name: 'USA' })).toBeNull();
    expect(screen.queryByRole('option', { name: 'Japan' })).toBeNull();
  });
});

describe('ReportParameterBar - Hydration, URL Sync & Bulk Binding', () => {
  const barConfig: ReportParameterBarConfig = {
    enabled: true,
    parameters: [
      {
        varName: 'asOfDate',
        label: 'As Of Date',
        type: 'date',
        control: 'date',
        default: '2026-09-01',
      },
      {
        varName: 'portfolioRegion',
        label: 'Region',
        type: 'string',
        control: 'select',
        default: 'GLOBAL',
        selectOptions: {
          source: 'static',
          options: [
            { label: 'Global', value: 'GLOBAL' },
            { label: 'North America', value: 'NA' },
            { label: 'EMEA', value: 'EMEA' },
          ],
        },
      },
    ],
  };

  it('hydrates variables from URL search params on mount', () => {
    const onUpdateVar = vi.fn();
    const searchParams = new URLSearchParams('asOfDate=2026-08-15&portfolioRegion=EMEA');

    render(
      <ReportParameterBar
        config={barConfig}
        variables={{}}
        onUpdateVariable={onUpdateVar}
        searchParams={searchParams}
      />
    );

    expect(onUpdateVar).toHaveBeenCalledWith('asOfDate', '2026-08-15');
    expect(onUpdateVar).toHaveBeenCalledWith('portfolioRegion', 'EMEA');
  });

  it('updates variable and syncs URL when parameter changes', () => {
    const onUpdateVar = vi.fn();
    const onUpdateUrl = vi.fn();

    render(
      <ReportParameterBar
        config={barConfig}
        variables={{ asOfDate: '2026-09-01', portfolioRegion: 'GLOBAL' }}
        onUpdateVariable={onUpdateVar}
        onUpdateSearchParams={onUpdateUrl}
      />
    );

    const dateInput = screen.getByLabelText('As Of Date');
    fireEvent.change(dateInput, { target: { value: '2026-09-30' } });

    expect(onUpdateVar).toHaveBeenCalledWith('asOfDate', '2026-09-30');
    expect(onUpdateUrl).toHaveBeenCalledWith({ asOfDate: '2026-09-30' });
  });

  it('bulk-bind modal inspects tile query parameters and writes paramBindings across tiles', () => {
    const onBulkBind = vi.fn();

    const components: Record<string, ComponentDefinition> = {
      'tile-1': {
        id: 'tile-1',
        type: 'KpiTile',
        label: 'AUM KPI',
        props: {
          savedQueryId: 'sq-aum',
        },
      },
      'tile-2': {
        id: 'tile-2',
        type: 'LineChart',
        label: 'Performance Chart',
        props: {
          savedQueryId: 'sq-perf',
          savedQueryParams: {
            asOfDate: { mode: 'pageVar', varName: 'asOfDate' },
          },
        },
      },
    };

    render(
      <ReportParameterBar
        config={barConfig}
        components={components}
        variables={{ asOfDate: '2026-09-01' }}
        onUpdateVariable={vi.fn()}
        onBulkBindTiles={onBulkBind}
        mode="design"
      />
    );

    // Click link icon to open bulk bind modal for asOfDate
    const bindIcons = screen.getAllByTitle(/Bind "asOfDate"/i);
    fireEvent.click(bindIcons[0]);

    expect(screen.getByText(/Bind Parameter: As Of Date/i)).toBeDefined();
    expect(screen.getByText('Bound to PageVar')).toBeDefined(); // tile-2
    expect(screen.getByText('Unbound')).toBeDefined(); // tile-1

    // Apply bulk bind
    const applyBtn = screen.getByRole('button', { name: /Bulk Bind All Tiles/i });
    fireEvent.click(applyBtn);

    expect(onBulkBind).toHaveBeenCalledWith(
      'asOfDate',
      expect.objectContaining({
        'tile-1': expect.objectContaining({
          props: expect.objectContaining({
            savedQueryParams: expect.objectContaining({
              asOfDate: { mode: 'pageVar', varName: 'asOfDate' },
            }),
          }),
        }),
      })
    );
  });
});

describe('End-to-End Grid Report Integration: Parameter Bar + Slicer + Tiles', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('executes 1 batch query on load and re-executes batch on parameter or slicer change with resolved params and IN filter', async () => {
    const mockBatchResponse: savedQueryApi.BatchExecuteSavedQueryResponse = {
      results: [
        {
          id: 'slicer-region',
          savedQueryId: 'sq-regions',
          columns: [{ name: 'region', type: 'string' }],
          rows: [{ region: 'EMEA' }, { region: 'US' }, { region: 'APAC' }],
          rowCount: 3,
          cacheHit: true,
          resolvedTier: 'hot',
        },
        {
          id: 'tile-revenue',
          savedQueryId: 'sq-rev',
          columns: [{ name: 'revenue', type: 'numeric' }],
          rows: [{ revenue: 750000 }],
          rowCount: 1,
          cacheHit: false,
          resolvedTier: 'hot',
        },
      ],
      totalDurationMs: 12,
    };

    vi.mocked(savedQueryApi.batchExecuteSavedQueries).mockResolvedValue(mockBatchResponse);

    const layout: PageGridLayout & { parameterBar?: ReportParameterBarConfig } = {
      cols: { lg: 12, md: 8, sm: 4 },
      items: [
        { i: 'slicer-region', x: 0, y: 0, w: 4, h: 4 },
        { i: 'tile-revenue', x: 4, y: 0, w: 8, h: 4 },
      ],
      parameterBar: {
        enabled: true,
        parameters: [
          {
            varName: 'asOfDate',
            label: 'As Of Date',
            type: 'date',
            control: 'date',
            default: '2026-09-01',
          },
        ],
      },
    };

    const components: Record<string, ComponentDefinition> = {
      'slicer-region': {
        id: 'slicer-region',
        type: 'SlicerTile',
        label: 'Region Filter',
        props: {
          config: {
            savedQueryId: 'sq-regions',
            dimensionAlias: 'region',
            termNodeId: 'term-region',
            display: 'multiSelect',
          },
        },
      },
      'tile-revenue': {
        id: 'tile-revenue',
        type: 'KpiTile',
        label: 'Revenue',
        props: {
          config: {
            savedQueryId: 'sq-rev',
            measureAlias: 'revenue',
            format: { type: 'currency' },
          },
          savedQueryParams: {
            asOfDate: { mode: 'pageVar', varName: 'asOfDate' },
          },
        },
      },
    };

    const Harness: React.FC = () => {
      const [vars, setVars] = useState<Record<string, any>>({ asOfDate: '2026-09-01' });

      return (
        <CrossFilterProvider>
          <GridLayoutRenderer
            layout={layout}
            components={components}
            variables={vars}
            onUpdateVariable={(name, val) => setVars((prev) => ({ ...prev, [name]: val }))}
            renderComponent={(comp, data) => (
              <div data-testid={`tile-${comp.id}`}>
                {comp.label}: {data?.rowCount ?? 0} rows
              </div>
            )}
          />
        </CrossFilterProvider>
      );
    };

    render(<Harness />);

    // Wait for initial batch call on load
    await waitFor(() => {
      expect(savedQueryApi.batchExecuteSavedQueries).toHaveBeenCalledTimes(1);
    });

    expect(savedQueryApi.batchExecuteSavedQueries).toHaveBeenLastCalledWith([
      expect.objectContaining({
        id: 'slicer-region',
        savedQueryId: 'sq-regions',
      }),
      expect.objectContaining({
        id: 'tile-revenue',
        savedQueryId: 'sq-rev',
        params: { asOfDate: '2026-09-01' }, // Resolved from page variable
      }),
    ]);

    // Update Date in Parameter Bar
    const dateInput = screen.getByLabelText('As Of Date');
    fireEvent.change(dateInput, { target: { value: '2026-09-30' } });

    // Triggers consolidated batch query with updated param
    await waitFor(() => {
      expect(savedQueryApi.batchExecuteSavedQueries).toHaveBeenCalledTimes(2);
    });

    expect(savedQueryApi.batchExecuteSavedQueries).toHaveBeenLastCalledWith([
      expect.objectContaining({
        id: 'slicer-region',
        savedQueryId: 'sq-regions',
      }),
      expect.objectContaining({
        id: 'tile-revenue',
        savedQueryId: 'sq-rev',
        params: { asOfDate: '2026-09-30' },
      }),
    ]);
  });
});
