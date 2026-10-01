import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { KpiTileWidget, formatMetricValue } from '../../pages/page-studio/KpiTileWidget';
import { GridLayoutRenderer } from '../../pages/page-studio/GridLayoutRenderer';
import { CrossFilterProvider } from '../../features/query-builder/utils/crossFilterBus';
import type { ComponentDefinition, PageGridLayout, KpiTileConfig } from '../../types/pageStudio';
import * as savedQueryApi from '../../features/query-builder/services/savedQueryApi';

vi.mock('../../features/query-builder/services/savedQueryApi', async () => {
  const actual = await vi.importActual('../../features/query-builder/services/savedQueryApi');
  return {
    ...actual,
    batchExecuteSavedQueries: vi.fn(),
  };
});

describe('KpiTileWidget - Contract & Rendering', () => {
  it('formats metric values correctly across currency, percentage, compact, and number types', () => {
    expect(formatMetricValue(1234567.89, { type: 'currency', currencySymbol: '$', precision: 2 })).toBe('$1,234,567.89');
    expect(formatMetricValue(0.158, { type: 'percentage', precision: 1 })).toBe('0.2%');
    expect(formatMetricValue(15.8, { type: 'percentage', precision: 1 })).toBe('15.8%');
    expect(formatMetricValue(2500000, { type: 'compact' })).toBe('2.5M');
    expect(formatMetricValue(45200, { type: 'compact' })).toBe('45.2K');
    expect(formatMetricValue(null, { type: 'number' })).toBe('—');
  });

  it('renders single-measure KPI tile with target comparison and delta variance', () => {
    const config: KpiTileConfig = {
      measureAlias: 'revenue',
      format: { type: 'currency', precision: 0 },
      comparison: {
        enabled: true,
        type: 'target_literal',
        targetValue: 80000,
        deltaFormat: 'percentage',
      },
    };

    const data = {
      rows: [{ revenue: 100000 }],
      columns: [{ name: 'revenue', type: 'numeric' }],
    };

    render(<KpiTileWidget title="Total Revenue" config={config} data={data} />);

    expect(screen.getByText('Total Revenue')).toBeDefined();
    expect(screen.getByText('$100,000')).toBeDefined();
    expect(screen.getByText('+25.0%')).toBeDefined();
  });

  it('renders trend series sparkline and previous period delta when multi-row data is provided', () => {
    const config: KpiTileConfig = {
      measureAlias: 'activeUsers',
      trendDimensionAlias: 'month',
      format: { type: 'number', precision: 0 },
      comparison: {
        enabled: true,
        type: 'previous_period',
        deltaFormat: 'percentage',
      },
      sparkline: {
        enabled: true,
        type: 'line',
      },
    };

    const data = {
      rows: [
        { month: '2026-01', activeUsers: 1000 },
        { month: '2026-02', activeUsers: 1200 },
        { month: '2026-03', activeUsers: 1500 },
      ],
      columns: [
        { name: 'month', type: 'string' },
        { name: 'activeUsers', type: 'numeric' },
      ],
    };

    const { container } = render(<KpiTileWidget title="Monthly Active Users" config={config} data={data} />);

    expect(screen.getByText('1,500')).toBeDefined();
    // Delta vs previous month (1500 vs 1200 = +25.0%)
    expect(screen.getByText('+25.0%')).toBeDefined();
    expect(container.querySelector('svg')).not.toBeNull();
  });

  it('supports polarity inversion for negative metrics (e.g. expenses, latency)', () => {
    const config: KpiTileConfig = {
      measureAlias: 'operatingExpenses',
      format: { type: 'currency', precision: 0 },
      comparison: {
        enabled: true,
        type: 'target_literal',
        targetValue: 50000,
        deltaFormat: 'percentage',
        invertPolarity: true, // Lower expense is positive
      },
    };

    const data = {
      rows: [{ operatingExpenses: 40000 }], // 40k < 50k target -> diff -10k -> inverted = positive
    };

    render(<KpiTileWidget title="Operating Expenses" config={config} data={data} />);

    expect(screen.getByText('-20.0%')).toBeDefined();
    expect(screen.getByTestId('TrendingUpIcon')).toBeDefined();
  });

  it('shows designer-time warning chip when comparison is enabled without trend dimension', () => {
    const config: KpiTileConfig = {
      measureAlias: 'revenue',
      format: { type: 'currency' },
      comparison: {
        enabled: true,
        type: 'previous_period', // previous_period needs a trend/period dimension
        deltaFormat: 'percentage',
      },
    };

    render(<KpiTileWidget title="Revenue" config={config} mode="design" />);
    expect(screen.getByText('Config Warning')).toBeDefined();
  });

  it('emits cross-filter on tile click when interactionConfig is enabled', () => {
    const onEmit = vi.fn();
    const config: KpiTileConfig = {
      measureAlias: 'revenue',
      format: { type: 'currency' },
      interactionConfig: {
        crossFilter: {
          enabled: true,
          mode: 'emit',
          termNodeId: 'term-sales-kpi',
        },
      },
    };

    const data = {
      rows: [{ revenue: 75000 }],
    };

    render(<KpiTileWidget title="Sales" config={config} data={data} onEmitCrossFilter={onEmit} />);

    fireEvent.click(screen.getByText('Sales'));
    expect(onEmit).toHaveBeenCalledWith('term-sales-kpi', 75000);
  });
});

describe('GridLayoutRenderer - 4-Tile Report Batch Loading & Interaction', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders 4 tiles and loads all queries in a single batchExecuteSavedQueries call on mount', async () => {
    const mockBatchResponse: savedQueryApi.BatchExecuteSavedQueryResponse = {
      results: [
        {
          id: 'kpi-1',
          savedQueryId: 'sq-kpi-1',
          columns: [{ name: 'revenue', type: 'numeric' }],
          rows: [{ revenue: 500000 }],
          rowCount: 1,
          cacheHit: true,
          resolvedTier: 'hot',
        },
        {
          id: 'kpi-2',
          savedQueryId: 'sq-kpi-2',
          columns: [{ name: 'orders', type: 'numeric' }],
          rows: [{ orders: 1250 }],
          rowCount: 1,
          cacheHit: true,
          resolvedTier: 'hot',
        },
        {
          id: 'chart-1',
          savedQueryId: 'sq-chart-1',
          columns: [{ name: 'region', type: 'string' }, { name: 'sales', type: 'numeric' }],
          rows: [{ region: 'EMEA', sales: 200000 }, { region: 'US', sales: 300000 }],
          rowCount: 2,
          cacheHit: false,
          resolvedTier: 'hot',
        },
        {
          id: 'table-1',
          savedQueryId: 'sq-table-1',
          columns: [{ name: 'id', type: 'string' }, { name: 'customer', type: 'string' }],
          rows: [{ id: '1', customer: 'Acme Corp' }],
          rowCount: 1,
          cacheHit: false,
          resolvedTier: 'hot',
        },
      ],
      totalDurationMs: 15,
    };

    vi.mocked(savedQueryApi.batchExecuteSavedQueries).mockResolvedValue(mockBatchResponse);

    const layout: PageGridLayout = {
      cols: { lg: 12, md: 8, sm: 4 },
      items: [
        { i: 'kpi-1', x: 0, y: 0, w: 6, h: 2 },
        { i: 'kpi-2', x: 6, y: 0, w: 6, h: 2 },
        { i: 'chart-1', x: 0, y: 2, w: 8, h: 4 },
        { i: 'table-1', x: 8, y: 2, w: 4, h: 4 },
      ],
    };

    const components: Record<string, ComponentDefinition> = {
      'kpi-1': {
        id: 'kpi-1',
        type: 'KpiTile',
        label: 'Total Revenue',
        props: {
          config: {
            savedQueryId: 'sq-kpi-1',
            measureAlias: 'revenue',
            format: { type: 'currency', precision: 0 },
          },
        },
      },
      'kpi-2': {
        id: 'kpi-2',
        type: 'KpiTile',
        label: 'Total Orders',
        props: {
          config: {
            savedQueryId: 'sq-kpi-2',
            measureAlias: 'orders',
            format: { type: 'number', precision: 0 },
          },
        },
      },
      'chart-1': {
        id: 'chart-1',
        type: 'LineChart',
        label: 'Sales by Region',
        props: {
          savedQueryId: 'sq-chart-1',
        },
      },
      'table-1': {
        id: 'table-1',
        type: 'Table',
        label: 'Top Customers',
        props: {
          savedQueryId: 'sq-table-1',
        },
      },
    };

    render(
      <CrossFilterProvider>
        <GridLayoutRenderer
          layout={layout}
          components={components}
          mode="preview"
          renderComponent={(comp, data) => (
            <div data-testid={`tile-${comp.id}`}>
              <span>{comp.label}</span>
              {data && <span data-testid={`rows-${comp.id}`}>{data.rowCount} rows</span>}
            </div>
          )}
        />
      </CrossFilterProvider>
    );

    await waitFor(() => {
      expect(screen.getByTestId('rows-kpi-1').textContent).toBe('1 rows');
    });

    expect(savedQueryApi.batchExecuteSavedQueries).toHaveBeenCalledWith([
      expect.objectContaining({ id: 'kpi-1', savedQueryId: 'sq-kpi-1' }),
      expect.objectContaining({ id: 'kpi-2', savedQueryId: 'sq-kpi-2' }),
      expect.objectContaining({ id: 'chart-1', savedQueryId: 'sq-chart-1' }),
      expect.objectContaining({ id: 'table-1', savedQueryId: 'sq-table-1' }),
    ]);

    expect(screen.getByTestId('rows-kpi-2').textContent).toBe('1 rows');
    expect(screen.getByTestId('rows-chart-1').textContent).toBe('2 rows');
    expect(screen.getByTestId('rows-table-1').textContent).toBe('1 rows');
  });

  it('supports authoring mode resize handles', () => {
    const onUpdate = vi.fn();
    const layout: PageGridLayout = {
      cols: { lg: 12, md: 8, sm: 4 },
      items: [{ i: 'kpi-1', x: 0, y: 0, w: 12, h: 2 }],
    };

    const components: Record<string, ComponentDefinition> = {
      'kpi-1': {
        id: 'kpi-1',
        type: 'KpiTile',
        label: 'Revenue',
        props: { savedQueryId: 'sq-1' },
      },
    };

    render(
      <CrossFilterProvider>
        <GridLayoutRenderer
          layout={layout}
          components={components}
          mode="design"
          onUpdateLayoutItem={onUpdate}
          renderComponent={(comp) => <div>{comp.label}</div>}
        />
      </CrossFilterProvider>
    );

    const resizeBtn = screen.getByTitle('Resize tile');
    fireEvent.click(resizeBtn);

    expect(onUpdate).toHaveBeenCalledWith(expect.objectContaining({ i: 'kpi-1', w: 6 }));
  });

  it('triggers onRefresh polling when refreshInterval >= 60 and document is visible', () => {
    vi.useFakeTimers();
    const onRefresh = vi.fn();
    const config: KpiTileConfig = {
      measureAlias: 'revenue',
      format: { type: 'currency' },
      refreshInterval: 60, // 60s
    };

    render(
      <KpiTileWidget
        title="Live KPI"
        config={config}
        data={{ rows: [{ revenue: 100 }] }}
        onRefresh={onRefresh}
      />
    );

    // Fast-forward 60s
    vi.advanceTimersByTime(60000);
    expect(onRefresh).toHaveBeenCalledTimes(1);

    // Fast-forward another 60s
    vi.advanceTimersByTime(60000);
    expect(onRefresh).toHaveBeenCalledTimes(2);

    vi.useRealTimers();
  });
});

