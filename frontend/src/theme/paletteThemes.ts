import { createTheme, type PaletteColorOptions, type Theme, type ThemeOptions } from '@mui/material/styles';
import { createUisceTheme } from './uisceTheme';

interface Brand { main: string; light: string; dark: string; contrastText: string }

interface Tokens {
  bg: string; paper: string; border: string; tableHead: string;
  text: string; textSecondary: string; textDisabled: string;
  primary: Brand; secondary: Brand;
}

export interface PaletteThemeSpec {
  light: Tokens;
  dark: Tokens;
  /** Base corner radius; cards use cardRadius. */
  radius: number;
  cardRadius: number;
  /** Flat cards (border only) vs a soft shadow. */
  flatCards?: boolean;
  fontFamily?: string;
  headingWeight?: number;
}

/**
 * Builds a full theme from a small token spec. Like Ivy it borrows the Uisce
 * theme only for the custom palette keys the shell reads (tertiary, category*).
 */
export function createPaletteTheme(mode: 'light' | 'dark', spec: PaletteThemeSpec): Theme {
  const t = spec[mode];
  const base = createUisceTheme(mode);
  const shadow = mode === 'light' ? '0 1px 3px rgba(0,0,0,0.06)' : '0 1px 3px rgba(0,0,0,0.4)';
  const heading = spec.headingWeight ?? 600;

  const options: ThemeOptions = {
    palette: {
      mode,
      primary: t.primary as PaletteColorOptions,
      secondary: t.secondary as PaletteColorOptions,
      background: { default: t.bg, paper: t.paper },
      text: { primary: t.text, secondary: t.textSecondary, disabled: t.textDisabled },
      divider: t.border,
      tertiary: base.palette.tertiary,
      categoryPlatform: base.palette.categoryPlatform,
      categoryCatalog: base.palette.categoryCatalog,
      categoryBuild: base.palette.categoryBuild,
      categoryStudio: base.palette.categoryStudio,
      categoryOperations: base.palette.categoryOperations,
      categoryIntelligence: base.palette.categoryIntelligence,
      categoryConsume: base.palette.categoryConsume,
      categoryCalendar: base.palette.categoryCalendar,
    },
    typography: {
      fontFamily: spec.fontFamily ?? '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
      h1: { fontSize: '2.25rem', fontWeight: heading + 100, letterSpacing: '-0.02em' },
      h2: { fontSize: '1.875rem', fontWeight: heading + 100, letterSpacing: '-0.01em' },
      h3: { fontSize: '1.5rem', fontWeight: heading },
      h4: { fontSize: '1.25rem', fontWeight: heading },
      h5: { fontSize: '1.125rem', fontWeight: heading },
      h6: { fontSize: '1rem', fontWeight: heading },
      button: { fontWeight: 600, textTransform: 'none' },
    },
    shape: { borderRadius: spec.radius },
    components: {
      MuiCssBaseline: { styleOverrides: { body: { backgroundColor: t.bg, colorScheme: mode } } },
      MuiButton: { styleOverrides: { root: { borderRadius: spec.radius, boxShadow: 'none' } } },
      MuiCard: {
        styleOverrides: {
          root: {
            borderRadius: spec.cardRadius,
            border: `1px solid ${t.border}`,
            boxShadow: spec.flatCards ? 'none' : shadow,
            backgroundImage: 'none',
          },
        },
      },
      MuiPaper: { styleOverrides: { root: { backgroundImage: 'none' } } },
      MuiAppBar: {
        styleOverrides: {
          root: { backgroundColor: t.paper, color: t.text, backgroundImage: 'none', boxShadow: 'none', borderBottom: `1px solid ${t.border}` },
        },
      },
      MuiDrawer: { styleOverrides: { paper: { backgroundColor: t.paper, borderRight: `1px solid ${t.border}` } } },
      MuiChip: { styleOverrides: { root: { borderRadius: Math.max(4, spec.radius - 2), fontWeight: 500 } } },
      MuiTableCell: {
        styleOverrides: {
          root: { borderBottom: `1px solid ${t.border}` },
          head: { fontWeight: 600, color: t.textSecondary, backgroundColor: t.tableHead },
        },
      },
    },
  };

  return createTheme(options);
}

// Accent is the primary colour in all three, so nav, buttons and the Core icon
// read as the theme's brand; neutrals carry the surfaces. Dark-mode accents are
// lightened and use dark contrast text to stay readable.

export const slateSpec: PaletteThemeSpec = {
  radius: 8, cardRadius: 8, headingWeight: 600,
  light: {
    bg: '#F8FAFC', paper: '#FFFFFF', border: '#E2E8F0', tableHead: '#F1F5F9',
    text: '#0F172A', textSecondary: '#64748B', textDisabled: '#94A3B8',
    primary: { main: '#2563EB', light: '#60A5FA', dark: '#1D4ED8', contrastText: '#FFFFFF' },
    secondary: { main: '#334155', light: '#64748B', dark: '#0F172A', contrastText: '#FFFFFF' },
  },
  dark: {
    bg: '#020617', paper: '#0F172A', border: '#1E293B', tableHead: '#020617',
    text: '#F8FAFC', textSecondary: '#94A3B8', textDisabled: '#64748B',
    primary: { main: '#60A5FA', light: '#93C5FD', dark: '#3B82F6', contrastText: '#0B1220' },
    secondary: { main: '#94A3B8', light: '#CBD5E1', dark: '#64748B', contrastText: '#0B1220' },
  },
};

export const obsidianSpec: PaletteThemeSpec = {
  radius: 4, cardRadius: 4, headingWeight: 600,
  light: {
    bg: '#FAFAF9', paper: '#FFFFFF', border: '#E7E5E4', tableHead: '#F5F5F4',
    text: '#1C1917', textSecondary: '#57534E', textDisabled: '#A8A29E',
    primary: { main: '#B45309', light: '#D97706', dark: '#92400E', contrastText: '#FFFFFF' },
    secondary: { main: '#44403C', light: '#78716C', dark: '#1C1917', contrastText: '#FFFFFF' },
  },
  dark: {
    bg: '#0C0A09', paper: '#1C1917', border: '#292524', tableHead: '#0C0A09',
    text: '#FAFAF9', textSecondary: '#A8A29E', textDisabled: '#78716C',
    primary: { main: '#F59E0B', light: '#FBBF24', dark: '#D97706', contrastText: '#1C1917' },
    secondary: { main: '#A8A29E', light: '#D6D3D1', dark: '#78716C', contrastText: '#1C1917' },
  },
};

export const frostSpec: PaletteThemeSpec = {
  radius: 12, cardRadius: 12, flatCards: true, headingWeight: 600,
  light: {
    bg: '#FFFFFF', paper: '#F9FAFB', border: '#E5E7EB', tableHead: '#F3F4F6',
    text: '#111827', textSecondary: '#6B7280', textDisabled: '#9CA3AF',
    primary: { main: '#4F46E5', light: '#818CF8', dark: '#3730A3', contrastText: '#FFFFFF' },
    secondary: { main: '#374151', light: '#6B7280', dark: '#111827', contrastText: '#FFFFFF' },
  },
  dark: {
    bg: '#030712', paper: '#111827', border: '#1F2937', tableHead: '#030712',
    text: '#F9FAFB', textSecondary: '#9CA3AF', textDisabled: '#6B7280',
    primary: { main: '#818CF8', light: '#A5B4FC', dark: '#6366F1', contrastText: '#0B1020' },
    secondary: { main: '#9CA3AF', light: '#D1D5DB', dark: '#6B7280', contrastText: '#0B1020' },
  },
};
