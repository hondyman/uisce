import React from 'react';
import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../../../i18n';

// Mock the Apollo useQuery used inside the page
vi.mock('@apollo/client', () => ({
  useQuery: () => ({ data: { business_terms: [], semantic_terms: [], semantic_columns: [] }, loading: false, error: null })
}));

import BusinessGlossaryPage from '../BusinessGlossaryPage';

describe('BusinessGlossaryPage i18n smoke test', () => {
  afterEach(async () => await i18n.changeLanguage('en'));

  test('renders search placeholder and tab names and updates when language changes', async () => {
    const { rerender } = render(
      <I18nextProvider i18n={i18n}>
        <BusinessGlossaryPage />
      </I18nextProvider>
    );

    expect(screen.getByPlaceholderText(/Search business & semantic terms/)).toBeTruthy();
    expect(screen.getByText(/Business Terms/)).toBeTruthy();
    expect(screen.getByText(/Semantic Terms/)).toBeTruthy();

    i18n.addResourceBundle('xx', 'translation', { global_search: { placeholder: 'Buscar...' }, tab: { business_terms: 'Negocios', semantic_terms: 'Semánticos' } }, true, true);
    await i18n.changeLanguage('xx');

    rerender(
      <I18nextProvider i18n={i18n}>
        <BusinessGlossaryPage />
      </I18nextProvider>
    );

    expect(screen.getByPlaceholderText(/Buscar.../)).toBeTruthy();
    expect(screen.getByText(/Negocios/)).toBeTruthy();
    expect(screen.getByText(/Semánticos/)).toBeTruthy();
  });
});
