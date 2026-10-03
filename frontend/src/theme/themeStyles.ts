import type { Theme } from '@mui/material/styles';
import { createUisceTheme } from './uisceTheme';
import { createIvyTheme } from './ivyTheme';
import { createPaletteTheme, frostSpec, obsidianSpec, slateSpec } from './paletteThemes';

export type ThemeStyle = 'uisce' | 'ivy' | 'slate' | 'obsidian' | 'frost';
export type ThemeMode = 'light' | 'dark';

/** Selectable visual styles. Add an entry here (and a factory) to offer another. */
export const THEME_STYLES: { id: ThemeStyle; label: string; description: string; accent: string }[] = [
  { id: 'uisce', label: 'Uisce', description: 'Teal and gold on deep navy', accent: '#00C9C8' },
  { id: 'ivy', label: 'Ivy', description: 'Green and teal, clean surfaces', accent: '#2E7D32' },
  { id: 'slate', label: 'Slate', description: 'Modern fintech: blue on cool grey', accent: '#2563EB' },
  { id: 'obsidian', label: 'Obsidian', description: 'Wealth management: amber on charcoal', accent: '#B45309' },
  { id: 'frost', label: 'Frost', description: 'Minimal and flat: indigo on white', accent: '#4F46E5' },
];

export const DEFAULT_THEME_STYLE: ThemeStyle = 'uisce';

export function isThemeStyle(v: unknown): v is ThemeStyle {
  return THEME_STYLES.some((s) => s.id === v);
}

export function createThemeForStyle(style: ThemeStyle, mode: ThemeMode): Theme {
  switch (style) {
    case 'ivy': return createIvyTheme(mode);
    case 'slate': return createPaletteTheme(mode, slateSpec);
    case 'obsidian': return createPaletteTheme(mode, obsidianSpec);
    case 'frost': return createPaletteTheme(mode, frostSpec);
    default: return createUisceTheme(mode);
  }
}
