import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AdaptiveSemanticCanvas } from '@/components/canvas/AdaptiveSemanticCanvas';
import type {
  ResolvedCapabilities,
  ResolvedField,
} from '@/types/mutability';

const fields: ResolvedField[] = [
  {
    semanticTermKey: 'portfolio_id',
    displayLabel: 'Portfolio Id',
    hydrationState: 'RESOLVED',
    isEditable: false,
    bindingBackendId: 'pg_postgres_oltp',
  },
  {
    semanticTermKey: 'live_volatility',
    displayLabel: 'Real-time Volatility Index',
    hydrationState: 'UNBOUND_FALLBACK_NULL',
    isEditable: true,
    bindingBackendId: 'sr_starrocks_hot',
  },
  {
    semanticTermKey: 'historical_cost',
    displayLabel: 'Historical Cost Basis',
    hydrationState: 'RESOLVED',
    isEditable: true,
    bindingBackendId: 'ib_iceberg_coldstore',
  },
];

const directCap: ResolvedCapabilities = {
  mutabilityMode: 'DIRECT_OLTP_SQL',
  temporalStrategy: 'NONE',
  allowDirectCrudFormButtons: true,
  activeRouteBackendId: 'pg_postgres_oltp',
};

describe('AdaptiveSemanticCanvas', () => {
  it('renders CQRS badge when mutabilityMode is ASYNCHRONOUS_CQRS_QUEUE', () => {
    render(
      <AdaptiveSemanticCanvas
        fields={fields}
        dataset={[]}
        capabilities={{
          ...directCap,
          mutabilityMode: 'ASYNCHRONOUS_CQRS_QUEUE',
          allowDirectCrudFormButtons: false,
        }}
      />,
    );
    expect(screen.getByText('CQRS Event Stream Node')).toBeTruthy();
  });

  it('renders Direct badge when mutabilityMode is DIRECT_OLTP_SQL', () => {
    render(<AdaptiveSemanticCanvas fields={fields} dataset={[]} capabilities={directCap} />);
    expect(screen.getByText('Direct Storage Mode')).toBeTruthy();
  });

  it('renders the empty message for UNBOUND_FALLBACK_NULL fields', () => {
    render(
      <AdaptiveSemanticCanvas
        fields={fields}
        dataset={[{ portfolio_id: 'p1', historical_cost: '100' }]}
        capabilities={directCap}
      />,
    );
    expect(screen.getByText('-- field unavailable --')).toBeTruthy();
  });

  it('renders Hot Tier Only badge for hot-bound fields', () => {
    render(<AdaptiveSemanticCanvas fields={fields} dataset={[]} capabilities={directCap} />);
    expect(screen.getByText('Hot Tier Only')).toBeTruthy();
  });

  it('renders Cold Tier badge for cold-bound fields', () => {
    render(<AdaptiveSemanticCanvas fields={fields} dataset={[]} capabilities={directCap} />);
    expect(screen.getByText('Cold Tier')).toBeTruthy();
  });

  it('renders empty state when dataset is empty', () => {
    render(<AdaptiveSemanticCanvas fields={fields} dataset={[]} capabilities={directCap} />);
    expect(screen.getByText('No rows to display')).toBeTruthy();
  });

  it('renders row values for RESOLVED fields', () => {
    render(
      <AdaptiveSemanticCanvas
        fields={fields}
        dataset={[{ portfolio_id: 'p1', historical_cost: '100.5' }]}
        capabilities={directCap}
      />,
    );
    expect(screen.getByText('p1')).toBeTruthy();
    expect(screen.getByText('100.5')).toBeTruthy();
  });
});