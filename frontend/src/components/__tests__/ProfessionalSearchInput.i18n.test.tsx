import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../../i18n';
import ProfessionalSearchInput from '../../ProfessionalSearchInput';

const sampleData = [
  { id: '1', text: 'Apple', subtext: 'Fruit', payload: { id: '1' } },
  { id: '2', text: 'Banana', subtext: 'Fruit', payload: { id: '2' } }
];

describe('ProfessionalSearchInput i18n', () => {
  afterEach(async () => await i18n.changeLanguage('en'));

  test('renders default placeholder and updates with language change', async () => {
    render(
      <I18nextProvider i18n={i18n}>
        <ProfessionalSearchInput data={sampleData} onSelect={() => {}} />
      </I18nextProvider>
    );

    const input = screen.getByPlaceholderText(/Type to search.../);
    expect(input).toBeTruthy();

    i18n.addResourceBundle('xx', 'translation', { search: { placeholder: 'Buscar...' } }, true, true);
    await i18n.changeLanguage('xx');

    expect(screen.getByPlaceholderText(/Buscar.../)).toBeTruthy();
  });

  test('shows localized no-results text', async () => {
    render(
      <I18nextProvider i18n={i18n}>
        <ProfessionalSearchInput data={[]} onSelect={() => {}} />
      </I18nextProvider>
    );

    const input = screen.getByPlaceholderText(/Type to search.../);
    fireEvent.change(input, { target: { value: 'zzz' } });

    const noResults = await screen.findByText(/No results found/);
    expect(noResults).toBeTruthy();

    i18n.addResourceBundle('xx', 'translation', { search: { no_results: 'Nada' } }, true, true);
    await i18n.changeLanguage('xx');

    expect(await screen.findByText(/Nada/)).toBeTruthy();
  });
});
