import { createTheme, type Theme, type ThemeOptions } from '@mui/material/styles';
import { createUisceTheme } from './uisceTheme';

const ivy = {
  300: '#81C784', 400: '#66BB6A', 500: '#4CAF50', 600: '#43A047',
  700: '#388E3C', 800: '#2E7D32', 900: '#1B5E20',
};
const teal = { 300: '#4DB6AC', 400: '#26A69A', 600: '#00897B', 800: '#00695C' };

/** Per-mode surface tokens; everything else in the Ivy theme is shared. */
const tokens = {
  light: {
    bg: '#F5F7FA', paper: '#FFFFFF', border: '#E5E7EB', tableHead: '#F9FAFB',
    text: '#1A1A2E', textSecondary: '#5A6072', textDisabled: '#9CA3AF',
    primary: ivy[800], primaryLight: ivy[400], primaryDark: ivy[900],
    secondary: teal[600], secondaryLight: teal[400], secondaryDark: teal[800],
    glow: 'rgba(46,125,50,0.25)',
  },
  dark: {
    bg: '#0D1117', paper: '#161B22', border: '#21262D', tableHead: '#0D1117',
    text: '#FFFFFF', textSecondary: '#B0B8C4', textDisabled: '#6B7280',
    primary: ivy[400], primaryLight: ivy[300], primaryDark: ivy[600],
    secondary: teal[400], secondaryLight: teal[300], secondaryDark: teal[600],
    glow: 'rgba(76,175,80,0.35)',
  },
} as const;

/**
 * Ivy: green/teal, Inter, rounded surfaces. Built on the Uisce theme only to
 * inherit the custom palette keys (tertiary, category*) the app shell reads;
 * everything visible is Ivy's.
 */
export function createIvyTheme(mode: 'light' | 'dark'): Theme {
  const t = tokens[mode];
  const base = createUisceTheme(mode);

  const options: ThemeOptions = {
    palette: {
      mode,
      primary: { main: t.primary, light: t.primaryLight, dark: t.primaryDark, contrastText: '#FFFFFF' },
      secondary: { main: t.secondary, light: t.secondaryLight, dark: t.secondaryDark, contrastText: '#FFFFFF' },
      success: { main: mode === 'light' ? ivy[500] : ivy[400] },
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
      fontFamily: '"Inter", "Roboto", "Helvetica", "Arial", sans-serif',
      h1: { fontSize: '2.25rem', fontWeight: 700, letterSpacing: '-0.02em' },
      h2: { fontSize: '1.875rem', fontWeight: 700, letterSpacing: '-0.01em' },
      h3: { fontSize: '1.5rem', fontWeight: 600 },
      h4: { fontSize: '1.25rem', fontWeight: 600 },
      h5: { fontSize: '1.125rem', fontWeight: 600 },
      h6: { fontSize: '1rem', fontWeight: 600 },
      button: { fontWeight: 600, textTransform: 'none', letterSpacing: '0.01em' },
    },
    shape: { borderRadius: 12 },
    components: {
      MuiCssBaseline: { styleOverrides: { body: { backgroundColor: t.bg, colorScheme: mode } } },
      MuiButton: {
        styleOverrides: {
          root: { borderRadius: 10, boxShadow: 'none', '&:hover': { boxShadow: `0 2px 8px ${t.glow}` } },
        },
      },
      MuiCard: {
        styleOverrides: { root: { borderRadius: 16, border: `1px solid ${t.border}`, backgroundImage: 'none' } },
      },
      MuiPaper: { styleOverrides: { root: { backgroundImage: 'none' } } },
      MuiAppBar: {
        styleOverrides: {
          root: { backgroundColor: t.paper, color: t.text, backgroundImage: 'none', boxShadow: 'none', borderBottom: `1px solid ${t.border}` },
        },
      },
      MuiDrawer: { styleOverrides: { paper: { backgroundColor: t.paper, borderRight: `1px solid ${t.border}` } } },
      MuiChip: { styleOverrides: { root: { borderRadius: 8, fontWeight: 500 } } },
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
