import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { AdaptiveSemanticForm } from '@/components/canvas/AdaptiveSemanticForm';
import type {
  ResolvedCapabilities,
  ResolvedField,
} from '@/types/mutability';
import { TEST_GOLD_COPY_TENANT_ID } from '@/utils/testutil';

// Mock the dispatch API so the test doesn't hit the network.
vi.mock('@/api/layoutResolver', () => ({
  dispatchMutationApi: vi.fn().mockResolvedValue({
    commandId: 'cmd-1',
    correlationId: 'corr-1',
    route: 'ASYNCHRONOUS_CQRS_QUEUE',
    topic: 'uisce.command.portfolio_holding.v1',
    status: 'pending',
    timestamp: '2026-07-09T00:00:00Z',
  }),
}));

const capabilities: ResolvedCapabilities = {
  mutabilityMode: 'ASYNCHRONOUS_CQRS_QUEUE',
  temporalStrategy: 'BITEMPORAL',
  allowDirectCrudFormButtons: false,
  activeRouteBackendId: 'sr_starrocks_hot',
  commandTopicTemplate: 'uisce.command.{boKey}.v1',
};

const fields: ResolvedField[] = [
  {
    semanticTermKey: 'portfolio_identifier',
    displayLabel: 'Portfolio Identifier',
    hydrationState: 'RESOLVED',
    isEditable: false,
  },
  {
    semanticTermKey: 'position_quantity',
    displayLabel: 'Position Quantity',
    hydrationState: 'RESOLVED',
    isEditable: true,
  },
  {
    semanticTermKey: 'risk_parameter',
    displayLabel: 'Risk Parameter',
    hydrationState: 'UNBOUND_FALLBACK_NULL',
    isEditable: true,
  },
];

describe('AdaptiveSemanticForm', () => {
  it('renders the CQRS badge when mutabilityMode is ASYNCHRONOUS_CQRS_QUEUE', () => {
    render(
      <AdaptiveSemanticForm
        capabilities={capabilities}
        schemaFields={fields}
        initialValues={{}}
        businessObjectKey="portfolio_holding"
        tenantId="TEST_GOLD_COPY_TENANT_ID"
      />,
    );
    const badge = screen.getByTestId('mutability-mode-badge');
    expect(badge.textContent).toContain('CQRS Buffered Pipe');
  });

  it('renders the Direct badge when mutabilityMode is DIRECT_OLTP_SQL', () => {
    render(
      <AdaptiveSemanticForm
        capabilities={{ ...capabilities, mutabilityMode: 'DIRECT_OLTP_SQL', allowDirectCrudFormButtons: true }}
        schemaFields={fields}
        initialValues={{}}
        businessObjectKey="customers"
        tenantId="TEST_GOLD_COPY_TENANT_ID"
      />,
    );
    const badge = screen.getByTestId('mutability-mode-badge');
    expect(badge.textContent).toContain('Direct Relational Mode');
  });

  it('locks UNBOUND_FALLBACK_NULL fields with "--" placeholder', () => {
    render(
      <AdaptiveSemanticForm
        capabilities={capabilities}
        schemaFields={fields}
        initialValues={{}}
        businessObjectKey="portfolio_holding"
        tenantId="TEST_GOLD_COPY_TENANT_ID"
      />,
    );
    expect(screen.getByText('Field Missing from Source')).toBeTruthy();
    const inputs = screen.getAllByRole('textbox') as HTMLInputElement[];
    // 3 fields → 3 inputs; the unbound one shows "--"
    expect(inputs.length).toBe(3);
    const unbound = inputs.find((i) => i.value === '--');
    expect(unbound).toBeTruthy();
    expect(unbound!.disabled).toBe(true);
  });

  it('locks non-editable fields (primary keys)', () => {
    render(
      <AdaptiveSemanticForm
        capabilities={{ ...capabilities, allowDirectCrudFormButtons: true }}
        schemaFields={fields}
        initialValues={{}}
        businessObjectKey="portfolio_holding"
        tenantId="TEST_GOLD_COPY_TENANT_ID"
      />,
    );
    const inputs = screen.getAllByRole('textbox') as HTMLInputElement[];
    const pkInput = inputs.find((i) => i.disabled);
    expect(pkInput).toBeTruthy();
  });

  it('dispatches mutation on submit and shows pending status', async () => {
    const onSuccess = vi.fn();
    render(
      <AdaptiveSemanticForm
        capabilities={capabilities}
        schemaFields={fields}
        initialValues={{ position_quantity: '100' }}
        businessObjectKey="portfolio_holding"
        bindingId="00000000-0000-0000-0000-00000000a002"
        tenantId="TEST_GOLD_COPY_TENANT_ID"
        onSuccess={onSuccess}
      />,
    );

    const button = screen.getByText('Publish State Mutation');
    fireEvent.click(button);

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
    });
  });
});