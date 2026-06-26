import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';

export type Theme = 'light' | 'dark' | 'system';

interface ThemeContextType {
  theme: Theme;
  systemTheme: 'light' | 'dark';
  effectiveTheme: 'light' | 'dark';
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

const THEME_STORAGE_KEY = 'app-theme-preference';

export const ThemeProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  // Detect system preference
  const [systemTheme, setSystemTheme] = useState<'light' | 'dark'>('light');
  
  // Get stored theme preference
  const [theme, setThemeState] = useState<Theme>(() => {
    try {
      const stored = localStorage.getItem(THEME_STORAGE_KEY);
      if (stored === 'light' || stored === 'dark' || stored === 'system') {
        return stored;
      }
    } catch (e) {
      // Silently fail if localStorage is not available
    }
    return 'system';
  });

  // Detect system preference on mount and when it changes
  useEffect(() => {
    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    
    const handleChange = (e: MediaQueryListEvent | MediaQueryList) => {
      setSystemTheme(e.matches ? 'dark' : 'light');
    };

    // Set initial system theme
    handleChange(mediaQuery);

    // Listen for changes
    mediaQuery.addEventListener('change', handleChange);
    return () => mediaQuery.removeEventListener('change', handleChange);
  }, []);

  // Calculate effective theme
  const effectiveTheme: 'light' | 'dark' = theme === 'system' ? systemTheme : theme;

  // Apply theme to DOM
  useEffect(() => {
    const html = document.documentElement;
    
    if (effectiveTheme === 'dark') {
      html.classList.add('dark');
      document.body.style.colorScheme = 'dark';
    } else {
      html.classList.remove('dark');
      document.body.style.colorScheme = 'light';
    }
  }, [effectiveTheme]);

  // Save theme preference to localStorage
  const setTheme = (newTheme: Theme) => {
    setThemeState(newTheme);
    try {
      localStorage.setItem(THEME_STORAGE_KEY, newTheme);
    } catch (e) {
      // Silently fail if localStorage is not available
    }
  };

  // Toggle between light and dark (respects system preference as "off" state)
  const toggleTheme = () => {
    if (theme === 'light') {
      setTheme('dark');
    } else if (theme === 'dark') {
      setTheme('system');
    } else {
      setTheme('light');
    }
  };

  const value: ThemeContextType = {
    theme,
    systemTheme,
    effectiveTheme,
    setTheme,
    toggleTheme,
  };

  return (
    <ThemeContext.Provider value={value}>
      {children}
    </ThemeContext.Provider>
  );
};

export const useTheme = () => {
  const context = useContext(ThemeContext);
  if (context === undefined) {
    throw new Error('useTheme must be used within a ThemeProvider');
  }
  return context;
};
