import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../../i18n';
import LanguageSelector from '../../LanguageSelector';
import useUserAPI from '../../../hooks/useUserAPI';

jest.mock('../../../hooks/useUserAPI');
const mockUseUserAPI = useUserAPI as unknown as jest.MockedFunction<typeof useUserAPI>;
const mockUpdate = jest.fn(async (userId: string, prefs: any) => ({ language: prefs.language }));
mockUseUserAPI.mockReturnValue({ getUserPreferences: async () => ({ language: 'en' }), updateUserPreferences: mockUpdate } as any);

describe('LanguageSelector', () => {
  afterEach(async () => await i18n.changeLanguage('en'));

  test('renders and changes language', async () => {
    render(
      <I18nextProvider i18n={i18n}>
        <LanguageSelector />
      </I18nextProvider>
    );

    const button = screen.getByRole('button', { name: /Language selector/i });
    expect(button).toBeTruthy();

    // open menu
    fireEvent.click(button);
    const esItem = await screen.findByText(/Español/);
    expect(esItem).toBeTruthy();

    // Click Spanish and ensure language changes
    fireEvent.click(esItem);
    // Wait for language change
    expect(i18n.resolvedLanguage).toBe('es');
    expect(localStorage.getItem('selected_language')).toBe('es');
    expect(mockUpdate).toHaveBeenCalled();
  });
});
