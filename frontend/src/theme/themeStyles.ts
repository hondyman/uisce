import type { Theme } from '@mui/material/styles';
import { createUisceTheme } from './uisceTheme';
import { createIvyTheme } from './ivyTheme';

export type ThemeStyle = 'uisce' | 'ivy';
export type ThemeMode = 'light' | 'dark';

/** Selectable visual styles. Add an entry here (and a factory) to offer another. */
export const THEME_STYLES: { id: ThemeStyle; label: string; description: string; accent: string }[] = [
  { id: 'uisce', label: 'Uisce', description: 'Teal and gold on deep navy', accent: '#00C9C8' },
  { id: 'ivy', label: 'Ivy', description: 'Green and teal, clean surfaces', accent: '#2E7D32' },
];

export const DEFAULT_THEME_STYLE: ThemeStyle = 'uisce';

export function isThemeStyle(v: unknown): v is ThemeStyle {
  return THEME_STYLES.some((s) => s.id === v);
}

export function createThemeForStyle(style: ThemeStyle, mode: ThemeMode): Theme {
  return style === 'ivy' ? createIvyTheme(mode) : createUisceTheme(mode);
}
