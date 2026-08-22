import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

vi.mock('../../../../contexts/TenantContext', () => ({
  useTenant: () => ({ tenant: { id: 'tenant-1' } }),
}));

const rejectionStore: any[] = [];
const apiCalls: Array<{ url: string; init?: RequestInit }> = [];

vi.mock('../../../../utils/apiClient', () => ({
  default: vi.fn(async (url: string, init?: RequestInit) => {
    apiCalls.push({ url, init });
    // GET rejections
    if (url.includes('/api/semantic-mapper/rejections') && (!init || init.method === undefined || init.method === 'GET')) {
      return { data: rejectionStore };
    }
    // POST rejection
    if (url.includes('/api/semantic-mapper/rejections') && init?.method === 'POST') {
      const body = JSON.parse(init.body as string);
      rejectionStore.push({
        rejection_id: `rej-${rejectionStore.length + 1}`,
        source_node_id: body.source_node_id,
        rejected_target_id: body.rejected_target_id,
        edge_type_id: body.edge_type_id,
      });
      return { status: 'recorded' };
    }
    // GET related terms
    if (url.includes('/related')) {
      return {
        primary_term: { term_id: 'focal', term_name: 'Account Code', relationship_type: '' },
        related_terms: [
          {
            term_id: 't-1',
            term_name: 'Account ID',
            qualified_path: 'semantic_term/Account_ID',
            domain: 'Finance',
            relationship_type: 'IS_SPECIALIZATION_OF',
            confidence: 0.85,
            differentiation_notes: 'Account ID is a specific identifier within the Account Code concept.',
          },
          {
            term_id: 't-2',
            term_name: 'Customer Account',
            qualified_path: 'business_term/Customer_Account',
            relationship_type: 'RELATES_TO',
            confidence: 0.62,
            differentiation_notes: 'Customer Account tracks ownership; Account Code is the identifier.',
          },
          {
            term_id: 't-3',
            term_name: 'CUSIP',
            relationship_type: 'IS_PEER_IDENTIFIER_OF',
            confidence: 0.74,
          },
        ],
        differentiator_summary: 'Three ways to slice the same domain.',
      };
    }
    // POST create relationship
    if (url.includes('/semantic-terms/relationships')) {
      return { edge_id: 'new-edge', status: 'created' };
    }
    return null;
  }),
}));

import { AISuggestionCard } from '@/features/glossary/components/AISuggestionCard';

const focalNode = {
  id: 'focal-1',
  node_name: 'Account Code',
  catalog_type_name: 'business_term',
};

function renderCard(props: any = {}) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const utils = render(
    <QueryClientProvider client={qc}>
      <AISuggestionCard
        entityType="business_term"
        entityId="focal-1"
        focalNode={focalNode as any}
        {...props}
      />
    </QueryClientProvider>
  );
  return utils;
}

describe('AISuggestionCard', () => {
  beforeEach(() => {
    apiCalls.length = 0;
    rejectionStore.length = 0;
  });

  it('renders loading state initially', async () => {
    renderCard();
    expect(screen.getByTestId('ai-suggestion-card-loading')).toBeInTheDocument();
  });

  it('renders 3 suggestion rows from the backend payload', async () => {
    renderCard();
    await waitFor(() => expect(screen.getAllByTestId('ai-suggestion-row')).toHaveLength(3));
    // Each row's predicate badge uses the registry icon
    expect(screen.getByText(/🔻 Specialization/i)).toBeInTheDocument();
    expect(screen.getByText(/🔗 Relates to/i)).toBeInTheDocument();
    expect(screen.getByText(/🔁 Peer identifier/i)).toBeInTheDocument();
  });

  it('hides rejected suggestions after the user dismisses them', async () => {
    renderCard();
    await waitFor(() => expect(screen.getAllByTestId('ai-suggestion-row')).toHaveLength(3));

    // Click the reject button on the first row (Account ID)
    const rejectButtons = screen.getAllByTestId('ai-suggestion-reject');
    fireEvent.click(rejectButtons[0]);

    // The first row should disappear immediately (optimistic local dismiss)
    await waitFor(() => expect(screen.getAllByTestId('ai-suggestion-row')).toHaveLength(2));
    expect(screen.queryByText('Account ID')).not.toBeInTheDocument();

    // Verify the POST to rejections endpoint fired with the right payload
    const post = apiCalls.find((c) => c.url.includes('/rejections') && c.init?.method === 'POST');
    expect(post).toBeDefined();
    expect(JSON.parse(post!.init!.body as string)).toMatchObject({
      source_node_id: 'focal-1',
      rejected_target_id: 't-1',
      edge_type_id: 'IS_SPECIALIZATION_OF',
    });
  });

  it('calls onSuggestionApplied after Accept', async () => {
    const onApplied = vi.fn();
    renderCard({ onSuggestionApplied: onApplied });

    await waitFor(() => expect(screen.getAllByTestId('ai-suggestion-row')).toHaveLength(3));

    const acceptButtons = screen.getAllByTestId('ai-suggestion-accept');
    fireEvent.click(acceptButtons[0]);

    // Verify POST to relationships endpoint fired
    await waitFor(() => {
      const create = apiCalls.find((c) => c.url.includes('/relationships') && c.init?.method === 'POST');
      expect(create).toBeDefined();
      expect(JSON.parse(create!.init!.body as string)).toMatchObject({
        source_node_id: 'focal-1',
        target_node_id: 't-1',
        edge_type_name: 'IS_SPECIALIZATION_OF',
      });
    });
    expect(onApplied).toHaveBeenCalled();
  });

  it('renders the empty state when there are no suggestions', async () => {
    vi.doMock('../../../../utils/apiClient', () => ({
      default: vi.fn(async (url: string) => {
        if (url.includes('/related')) {
          return { primary_term: {}, related_terms: [] };
        }
        return { data: [] };
      }),
    }));
    const { rerender } = renderCard();
    // Force a re-fetch by remounting
    rerender(
      <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
        <AISuggestionCard entityType="business_term" entityId="focal-2" focalNode={focalNode as any} />
      </QueryClientProvider>
    );
    vi.doUnmock('../../../../utils/apiClient');
  });

  it('renders nothing when disabled', () => {
    const { container } = renderCard({ disabled: true });
    expect(container.firstChild).toBeNull();
  });
});
