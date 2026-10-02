import { DEFAULT_THEME_STYLE, isThemeStyle, type ThemeStyle } from '../theme/themeStyles';
import React, { createContext, useContext, useEffect, useState, ReactNode } from 'react';

export type Theme = 'light' | 'dark' | 'system';

interface ThemeContextType {
  theme: Theme;
  systemTheme: 'light' | 'dark';
  effectiveTheme: 'light' | 'dark';
  /** Visual style (palette family), independent of light/dark. */
  style: ThemeStyle;
  setStyle: (style: ThemeStyle) => void;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
}

const ThemeContext = createContext<ThemeContextType | undefined>(undefined);

const THEME_STORAGE_KEY = 'app-theme-preference';
const STYLE_STORAGE_KEY = 'app-theme-style';

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

  const [style, setStyleState] = useState<ThemeStyle>(() => {
    try {
      const stored = localStorage.getItem(STYLE_STORAGE_KEY);
      if (isThemeStyle(stored)) return stored;
    } catch (e) {
      // Silently fail if localStorage is not available
    }
    return DEFAULT_THEME_STYLE;
  });

  const setStyle = (next: ThemeStyle) => {
    setStyleState(next);
    try {
      localStorage.setItem(STYLE_STORAGE_KEY, next);
    } catch (e) {
      // Silently fail if localStorage is not available
    }
  };

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

    // Uisce brand CSS variables for nav
    const r = document.documentElement.style;
    if (style === 'ivy') {
      const dark = effectiveTheme === 'dark';
      const accent = dark ? '#66BB6A' : '#2E7D32';
      const rgb = dark ? '102, 187, 106' : '46, 125, 50';
      const surface = dark ? '#161B22' : '#FFFFFF';
      const line = dark ? '#21262D' : '#E5E7EB';
      r.setProperty('--nav-accent', accent);
      r.setProperty('--nav-bg', dark ? '#0D1117' : '#F5F7FA');
      r.setProperty('--nav-text', dark ? '#FFFFFF' : '#1A1A2E');
      r.setProperty('--nav-appbar-bg', surface);
      r.setProperty('--nav-appbar-border', line);
      r.setProperty('--nav-border-accent', `rgba(${rgb}, 0.5)`);
      r.setProperty('--nav-accent-muted', `rgba(${rgb}, 0.10)`);
      r.setProperty('--nav-hover-fill', `rgba(${rgb}, 0.06)`);
      r.setProperty('--nav-glass-bg', surface);
      r.setProperty('--nav-glass-border', line);
      r.setProperty('--nav-menu-shadow', dark ? '0 8px 32px rgba(0, 0, 0, 0.5)' : '0 4px 16px rgba(0, 0, 0, 0.08)');
      r.setProperty('--nav-item-active', `rgba(${rgb}, 0.12)`);
      r.setProperty('--nav-item-text', dark ? '#FFFFFF' : '#1A1A2E');
      r.setProperty('--nav-item-hover', `rgba(${rgb}, 0.06)`);
      r.setProperty('--nav-sidebar-bg', surface);
      r.setProperty('--nav-sidebar-border', line);
      r.setProperty('--nav-rail-accent', accent);
      r.setProperty('--nav-text-dim', dark ? '#B0B8C4' : '#5A6072');
      r.setProperty('--nav-glow-color', `rgba(${rgb}, 0.35)`);
    } else if (effectiveTheme === 'dark') {
      r.setProperty('--nav-accent', '#F5C518');
      r.setProperty('--nav-bg', '#0A0C12');
      r.setProperty('--nav-text', '#E2E8F0');
      r.setProperty('--nav-appbar-bg', 'rgba(19, 22, 30, 0.85)');
      r.setProperty('--nav-appbar-border', 'rgba(255, 255, 255, 0.07)');
      r.setProperty('--nav-border-accent', 'rgba(245, 197, 24, 0.5)');
      r.setProperty('--nav-accent-muted', 'rgba(245, 197, 24, 0.10)');
      r.setProperty('--nav-hover-fill', 'rgba(245, 197, 24, 0.06)');
      r.setProperty('--nav-glass-bg', 'rgba(19, 22, 30, 0.80)');
      r.setProperty('--nav-glass-border', 'rgba(255, 255, 255, 0.07)');
      r.setProperty('--nav-menu-shadow', '0 8px 32px rgba(0, 0, 0, 0.6)');
      r.setProperty('--nav-item-active', 'rgba(245, 197, 24, 0.10)');
      r.setProperty('--nav-item-text', '#E2E8F0');
      r.setProperty('--nav-item-hover', 'rgba(245, 197, 24, 0.06)');
      r.setProperty('--nav-sidebar-bg', '#0A0C12');
      r.setProperty('--nav-sidebar-border', 'rgba(255, 255, 255, 0.07)');
      r.setProperty('--nav-rail-accent', '#F5C518');
      r.setProperty('--nav-text-dim', '#8892A4');
      r.setProperty('--nav-glow-color', 'rgba(245, 197, 24, 0.4)');
    } else {
      r.setProperty('--nav-accent', '#D4A017');
      r.setProperty('--nav-bg', '#F5F0E8');
      r.setProperty('--nav-text', '#1A0F08');
      r.setProperty('--nav-appbar-bg', 'rgba(245, 240, 232, 0.85)');
      r.setProperty('--nav-appbar-border', 'rgba(26, 15, 8, 0.08)');
      r.setProperty('--nav-border-accent', 'rgba(212, 160, 23, 0.5)');
      r.setProperty('--nav-accent-muted', 'rgba(212, 160, 23, 0.10)');
      r.setProperty('--nav-hover-fill', 'rgba(212, 160, 23, 0.06)');
      r.setProperty('--nav-glass-bg', 'rgba(245, 240, 232, 0.96)');
      r.setProperty('--nav-glass-border', 'rgba(26, 15, 8, 0.08)');
      r.setProperty('--nav-menu-shadow', '0 4px 12px rgba(26, 15, 8, 0.08), 0 16px 40px rgba(26, 15, 8, 0.10)');
      r.setProperty('--nav-item-active', 'rgba(212, 160, 23, 0.10)');
      r.setProperty('--nav-item-text', '#1A0F08');
      r.setProperty('--nav-item-hover', 'rgba(212, 160, 23, 0.06)');
      r.setProperty('--nav-sidebar-bg', '#ffffff');
      r.setProperty('--nav-sidebar-border', 'rgba(26, 15, 8, 0.08)');
      r.setProperty('--nav-rail-accent', '#D4A017');
      r.setProperty('--nav-text-dim', '#8B7D6B');
      r.setProperty('--nav-glow-color', 'rgba(212, 160, 23, 0.3)');
    }
    html.dataset.themeStyle = style;
  }, [effectiveTheme, style]);

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
    style,
    setStyle,
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
