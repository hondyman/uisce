import React from 'react';
import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';

// Mock useQuery so we can simulate a missing-field GraphQL error without wiring up MockedProvider
vi.mock('@apollo/client', async () => {
  const actual = await vi.importActual<typeof import('@apollo/client')>('@apollo/client');
  return {
    ...actual,
    useQuery: (query: any, options?: any) => {
      const qtext = String(query?.loc?.source?.body || '').toLowerCase();
      const datasourceId = options?.variables?.datasourceId;

      // Simulate missing tenant_chart for the combined chart query
      if (qtext.includes('query getcombinedchart') || qtext.includes('tenant_chart') && qtext.includes('in: ["enhanced_erd_chart"')) {
        return { loading: false, error: new Error("field 'tenant_chart' not found in type: 'query_root'"), data: undefined };
      }

      // For other queries return neutral, empty results
      return { loading: false, error: undefined, data: {} };
    }
  };
});
import TabbedModal from '../TabbedModal';
import { GET_COMBINED_CHART, GET_ALL_SEMANTIC_DATA, GET_TECHNICAL_LINEAGE_CHART, GET_SEMANTIC_LINEAGE_CHART } from '../../../graphql/queries/semantic';

describe('TabbedModal - missing tenant_chart field', () => {
  it('shows helpful message when tenant_chart field is missing from GraphQL schema', async () => {
    // we mock useQuery above — no explicit MockedProvider mocks are needed

    render(<TabbedModal datasourceId="ds-123" onClose={() => {}} isModal={false} />);

    // Wait for the component to render the missing-field message
    expect(await screen.findByText(/Lineage\/ERD data not available/i)).toBeInTheDocument();
  });
});
