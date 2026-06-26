import React from 'react';
import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../../i18n';
import BusinessTermsTree from '../../BusinessTermsTree';

describe('BusinessTermsTree i18n smoke test', () => {
  afterEach(async () => await i18n.changeLanguage('en'));

  test('renders view toggle and updates when language changes', async () => {
    const businessTerms = [{ id: 'bt1', node_name: 'Term 1', properties: {} }];

    const { rerender } = render(
      <I18nextProvider i18n={i18n}>
        <BusinessTermsTree
          businessTerms={businessTerms}
          semanticTerms={[]}
          semanticViews={[]}
          selectedAsset={null}
          onAssetSelect={() => {}}
          highlightedItem={null}
          searchTerm=""
        />
      </I18nextProvider>
    );

    // Should show the English toggle
    expect(screen.getByText(/Flat View|Tree View/)).toBeTruthy();

    // Add another language and verify change
    i18n.addResourceBundle('xx', 'translation', { view: { tree: 'Arbol', flat: 'Plano' }, no_results: { title: 'Nada', description: 'No hay resultados' } }, true, true);
    await i18n.changeLanguage('xx');

    // Rerender to pick up the new language
    rerender(
      <I18nextProvider i18n={i18n}>
        <BusinessTermsTree
          businessTerms={businessTerms}
          semanticTerms={[]}
          semanticViews={[]}
          selectedAsset={null}
          onAssetSelect={() => {}}
          highlightedItem={null}
          searchTerm=""
        />
      </I18nextProvider>
    );

    expect(screen.getByText(/Arbol|Plano/)).toBeTruthy();
  });
});
