import React from 'react';
import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../../i18n';

// Mock hooks used in BusinessTermsTab
vi.mock('../../../api/glossary', () => ({
  useBusinessTerms: () => ({ data: [{ id: 'bt-1', node_name: 'Business A', is_active: true }], isLoading: false }),
  useSemanticTerms: () => ({ data: [], isLoading: false }),
  useGlossaryEdges: () => ({ data: [], isLoading: false })
}));

// Also mock GET_ALL_SEMANTIC_DATA query
vi.mock('@apollo/client', () => ({
  useQuery: () => ({ data: { business_terms: [{ id: 'bt-1', node_name: 'Business A' }], semantic_terms: [], semantic_columns: [], semantic_edges: [] }, loading: false, error: null })
}));

import BusinessTermsTab from '../BusinessTermsTab';

describe('BusinessTermsTab i18n smoke test', () => {
  afterEach(async () => await i18n.changeLanguage('en'));

  test('renders tab title and responds to language changes', async () => {
    const { rerender } = render(
      <I18nextProvider i18n={i18n}>
        <BusinessTermsTab />
      </I18nextProvider>
    );

    expect(screen.getByText(/Business Terms/)).toBeTruthy();

    i18n.addResourceBundle('xx', 'translation', { tab: { business_terms: 'Términos de Negocio' } }, true, true);
    await i18n.changeLanguage('xx');

    rerender(
      <I18nextProvider i18n={i18n}>
        <BusinessTermsTab />
      </I18nextProvider>
    );

    expect(screen.getByText(/Términos de Negocio/)).toBeTruthy();
  });
});
