import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import SavedQueryWidget from '../../pages/page-studio/SavedQueryWidget';
import { SUPPORTED_CHART_TYPES } from '../../features/query-builder/charts';
import * as savedQueryApi from '../../features/query-builder/services/savedQueryApi';

vi.mock('echarts-for-react', () => ({
  default: (props: any) => (
    <div data-testid="echarts-mock" data-option={JSON.stringify(props.option)} data-has-echarts={String(!!props.echarts)}>
      {props.option?.series?.[0]?.type || 'chart-rendered'}
    </div>
  ),
}));

describe('SavedQueryWidget - ECharts Registration & All 8 Chart Types Post Code-Splitting', () => {
  const sampleData: Record<string, any> = {
    bar: {
      columns: [{ name: 'region', type: 'string' }, { name: 'revenue', type: 'numeric' }],
      rows: [{ region: 'North America', revenue: 1000 }],
    },
    stackedBar: {
      columns: [{ name: 'region', type: 'string' }, { name: 'tier', type: 'string' }, { name: 'revenue', type: 'numeric' }],
      rows: [{ region: 'North America', tier: 'Enterprise', revenue: 1000 }],
    },
    line: {
      columns: [{ name: 'month', type: 'string' }, { name: 'sales', type: 'numeric' }],
      rows: [{ month: '2026-01', sales: 500 }],
    },
    area: {
      columns: [{ name: 'month', type: 'string' }, { name: 'sales', type: 'numeric' }],
      rows: [{ month: '2026-01', sales: 500 }],
    },
    pie: {
      columns: [{ name: 'status', type: 'string' }, { name: 'count', type: 'numeric' }],
      rows: [{ status: 'Active', count: 42 }],
    },
    donut: {
      columns: [{ name: 'status', type: 'string' }, { name: 'count', type: 'numeric' }],
      rows: [{ status: 'Active', count: 42 }],
    },
    scatter: {
      columns: [{ name: 'entity', type: 'string' }, { name: 'x', type: 'numeric' }, { name: 'y', type: 'numeric' }],
      rows: [{ entity: 'Alpha', x: 10, y: 20 }],
    },
    combo: {
      columns: [{ name: 'month', type: 'string' }, { name: 'sales', type: 'numeric' }, { name: 'margin', type: 'numeric' }],
      rows: [{ month: '2026-01', sales: 500, margin: 25 }],
    },
  };

  it.each(SUPPORTED_CHART_TYPES)('renders chart type "%s" correctly with echarts instance provided', async (chartType) => {
    const runSpy = vi.spyOn(savedQueryApi, 'runSavedQuery').mockResolvedValue({
      name: `sq-${chartType}`,
      chartType: chartType as any,
      columns: sampleData[chartType].columns,
      rows: sampleData[chartType].rows,
      rowCount: 1,
    });

    render(
      <SavedQueryWidget
        savedQueryId={`sq-${chartType}`}
        widgetType="chart"
      />
    );

    await waitFor(() => {
      const chart = screen.getByTestId('echarts-mock');
      expect(chart).toBeDefined();
      expect(chart.getAttribute('data-has-echarts')).toBe('true');
      const option = JSON.parse(chart.getAttribute('data-option') || '{}');
      expect(option).toBeDefined();
      expect(option.series).toBeDefined();
      expect(option.series.length).toBeGreaterThan(0);
    });

    runSpy.mockRestore();
  });
});
