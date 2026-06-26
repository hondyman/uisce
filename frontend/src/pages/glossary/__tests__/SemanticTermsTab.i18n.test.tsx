import React from 'react';
import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../../i18n';

// Mock Apollo useQuery
vi.mock('@apollo/client', () => ({
  useQuery: () => ({ data: { semantic_terms: [{ id: 'st-1', node_name: 'Semantic A', description: 'desc' }], semantic_edges: [] }, loading: false, error: null })
}));

// Mock useEdgeTypes and useNodeTypes
vi.mock('../../../api/edgeTypes', () => ({ useEdgeTypes: () => ({ data: [] }) }));
vi.mock('../../../api/nodeTypes', () => ({ useNodeTypes: () => ({ data: [] }) }));

import SemanticTermsTab from '../SemanticTermsTab';

describe('SemanticTermsTab i18n smoke test', () => {
  afterEach(async () => await i18n.changeLanguage('en'));

  test('renders relationships header and updates when language changes', async () => {
    const { rerender } = render(
      <I18nextProvider i18n={i18n}>
        <SemanticTermsTab />
      </I18nextProvider>
    );

    // relationships table header 'ID' (from translations)
    expect(screen.getByText(/ID/)).toBeTruthy();

    i18n.addResourceBundle('xx', 'translation', { relationships: { id: 'Identificador' } }, true, true);
    await i18n.changeLanguage('xx');

    rerender(
      <I18nextProvider i18n={i18n}>
        <SemanticTermsTab />
      </I18nextProvider>
    );

    expect(screen.getByText(/Identificador/)).toBeTruthy();
  });
});
