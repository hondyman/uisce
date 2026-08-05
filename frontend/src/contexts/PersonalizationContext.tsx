import React, { createContext, useContext, useState, useCallback, ReactNode } from 'react';
import { useAuth } from './AuthContext';
import { getEnv } from '../utils/getEnv';
import { useAuthFetch } from '../utils/authFetch';

export type MenuDisplayMode = 'standard' | 'cards';

interface PersonalizationContextType {
  menuDisplayMode: MenuDisplayMode;
  setMenuDisplayMode: (mode: MenuDisplayMode) => void;
  toggleMenuDisplayMode: () => void;
}

const PersonalizationContext = createContext<PersonalizationContextType | undefined>(undefined);

const MENU_DISPLAY_MODE_KEY = 'menu-display-mode';

export const PersonalizationProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const auth = useAuth();
  const { authFetch } = useAuthFetch();
  const API_BASE_URL = getEnv('', 'VITE_API_BASE_URL', 'http://localhost:29080') as string;

  const [menuDisplayMode, setMenuDisplayModeState] = useState<MenuDisplayMode>(() => {
    if (typeof localStorage !== 'undefined') {
      const stored = localStorage.getItem(MENU_DISPLAY_MODE_KEY);
      if (stored === 'standard' || stored === 'cards') {
        return stored;
      }
    }
    return 'standard';
  });

  const setMenuDisplayMode = useCallback(async (mode: MenuDisplayMode) => {
    setMenuDisplayModeState(mode);
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(MENU_DISPLAY_MODE_KEY, mode);
    }

    if (auth?.user?.id) {
      try {
        await authFetch(`${API_BASE_URL}/api/users/${auth.user.id}/preferences`, {
          method: 'PUT',
          json: { menu_display_mode: mode },
        });
      } catch (err) {
        console.error('Failed to sync menu display preference to server', err);
      }
    }
  }, [auth?.user?.id, authFetch, API_BASE_URL]);

  const toggleMenuDisplayMode = useCallback(() => {
    const newMode = menuDisplayMode === 'cards' ? 'standard' : 'cards';
    setMenuDisplayMode(newMode);
  }, [menuDisplayMode, setMenuDisplayMode]);

  const value: PersonalizationContextType = {
    menuDisplayMode,
    setMenuDisplayMode,
    toggleMenuDisplayMode,
  };

  return (
    <PersonalizationContext.Provider value={value}>
      {children}
    </PersonalizationContext.Provider>
  );
};

export const usePersonalization = () => {
  const context = useContext(PersonalizationContext);
  if (context === undefined) {
    throw new Error('usePersonalization must be used within a PersonalizationProvider');
  }
  return context;
};
