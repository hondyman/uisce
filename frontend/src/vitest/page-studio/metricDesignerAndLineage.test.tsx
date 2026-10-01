import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MetricDesignerModal } from '../../features/query-builder/components/MetricDesignerModal';
import { MetricLineageDrawer } from '../../features/query-builder/components/MetricLineageDrawer';
import { KpiTileWidget } from '../../pages/page-studio/KpiTileWidget';

describe('Phase 7.3: Metric Designer, Tile Bindings, and Lineage Drawer', () => {
  it('surfaces inline grain-422 error when query requests unallowed grains', () => {
    render(
      <MetricDesignerModal
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
        initialMetric={{
          name: 'Regional Margin',
          description: 'Desc',
          boId: 'order',
          expression: { kind: 'aggregation', fn: 'sum', termNodeId: 'margin' },
          grainAllowlist: ['region', 'desk'],
          variables: [],
          formatConfig: { type: 'currency' },
          materializationConfig: { strategy: 'on_the_fly' },
        }}
        activeQueryGrains={['region', 'trader_ssn']} // 'trader_ssn' is unallowed
      />
    );

    expect(screen.getByText(/ErrGrainNotAllowed/i)).toBeInTheDocument();
    expect(screen.getByText(/Requested grain\(s\) \[trader_ssn\] are not permitted/i)).toBeInTheDocument();
  });

  it('validates AST formula and rejects unauthorized dangerous SQL keywords', () => {
    render(
      <MetricDesignerModal
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
        initialMetric={{
          name: 'Formula Metric',
          description: 'Desc',
          boId: 'order',
          expression: {
            kind: 'formula',
            formula: 'SUM(amount); DROP TABLE orders; --',
          },
          grainAllowlist: ['region'],
          variables: [],
          formatConfig: { type: 'number' },
          materializationConfig: { strategy: 'on_the_fly' },
        }}
      />
    );

    expect(screen.getByText(/Formula contains invalid or unauthorized SQL keywords/i)).toBeInTheDocument();
  });

  it('renders KPI tile from metricId with format precedence and legacy raw field designer hint', () => {
    const { container, rerender } = render(
      <KpiTileWidget
        title="Active Volume"
        mode="design"
        config={{
          metricId: 'm_vol_001',
          measureAlias: 'raw_volume',
          format: { type: 'currency', currencySymbol: '€', precision: 2 },
        }}
        data={{
          rows: [{ m_vol_001: 54000.5, raw_volume: 12000 }],
        }}
      />
    );

    // metricId takes precedence over measureAlias (54,000.50 with € symbol)
    expect(screen.getByText('€54,000.50')).toBeInTheDocument();
    expect(screen.queryByText('Raw Field')).not.toBeInTheDocument();

    // Rerender with legacy measureAlias without metricId -> surfaces design hint
    rerender(
      <KpiTileWidget
        title="Legacy Volume"
        mode="design"
        config={{
          measureAlias: 'raw_volume',
          format: { type: 'currency', currencySymbol: '$', precision: 0 },
        }}
        data={{
          rows: [{ raw_volume: 12000 }],
        }}
      />
    );

    expect(screen.getByText('$12,000')).toBeInTheDocument();
    expect(screen.getByText('Raw Field')).toBeInTheDocument();
  });

  it('renders MetricLineageDrawer displaying METRIC_OF, DERIVED_FROM, and USES_TERM graph edges', async () => {
    render(
      <MetricLineageDrawer
        open={true}
        onClose={vi.fn()}
        metricId="m_compound_ratio"
        metricName="Adjusted Margin Ratio"
      />
    );

    expect(await screen.findByText('Metric Lineage Graph')).toBeInTheDocument();
    expect(screen.getByText('Adjusted Margin Ratio')).toBeInTheDocument();
    expect(screen.getByText('Business Object (METRIC_OF)')).toBeInTheDocument();
    expect(screen.getByText('Base Metrics (DERIVED_FROM)')).toBeInTheDocument();
    expect(screen.getByText('Underlying Terms & Columns (USES_TERM)')).toBeInTheDocument();
  });
});
