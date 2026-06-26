import React from 'react';
import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../../i18n';

// Mock glossary hooks so the component mounts without hitting network
vi.mock('../../../api/glossary', () => ({
  useBusinessTerms: () => ({ data: [{ id: 'bt-1', description: 'Business A', catalog_type_name: 'Business Term', is_active: true }], isLoading: false }),
  useSemanticTerms: () => ({ data: [{ id: 'st-1', description: 'Semantic A', catalog_type_name: 'Semantic Term', is_active: true }], isLoading: false }),
  useGlossaryEdges: () => ({ data: [], isLoading: false }),
  useCreateTermEdge: () => ({})
}));

import GlossaryFlowView from '../GlossaryFlowView';

describe('GlossaryFlowView i18n smoke test', () => {
  afterEach(async () => await i18n.changeLanguage('en'));

  test('renders glossary title and updates when language changes', async () => {
    const { rerender } = render(
      <I18nextProvider i18n={i18n}>
        <GlossaryFlowView focus="business" />
      </I18nextProvider>
    );

    expect(screen.getByText(/Business Glossary - Relationships/)).toBeTruthy();

    i18n.addResourceBundle('xx', 'translation', { glossary: { title: 'Glosario - Relaciones' } }, true, true);
    await i18n.changeLanguage('xx');

    rerender(
      <I18nextProvider i18n={i18n}>
        <GlossaryFlowView focus="business" />
      </I18nextProvider>
    );

    expect(screen.getByText(/Glosario - Relaciones/)).toBeTruthy();
  });
});
