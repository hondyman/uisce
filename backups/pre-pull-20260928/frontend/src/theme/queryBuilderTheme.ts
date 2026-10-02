import { createTheme, alpha, PaletteOptions } from '@mui/material/styles';
import { darkPalette } from './palette';

// Tone tokens — sourced from the UISCE platform palette so this page fits in.
const qb = {
  primary:        '#0d9488',
  primaryFixed:   '#2dd4bf',
  onPrimary:      '#ffffff',
  secondary:      '#38bdf8',
  tertiary:       '#93ccff',
  warning:        '#fb923c',
  purple:         '#a78bfa',
  error:          '#ffb4ab',
  surface:        '#031427',
  bgDefault:      '#031427',
  bgPaper:        '#0b1c30',
  bgSubtle:       '#102034',
  bgElevated:     '#162a42',
  bgLowest:       '#000f21',
  surfaceContainerLowest: '#000f21',
  surfaceContainerLow:    '#031427',
  surfaceContainer:       '#0b1c30',
  surfaceContainerHigh:   '#162a42',
  surfaceContainerHighest:'#102843',
  onSurface:      '#d3e4fe',
  onSurfaceVariant: '#879391',
  outline:        '#879391',
  outlineVariant: 'rgba(135, 147, 145, 0.22)',
  borderLight:    'rgba(255, 255, 255, 0.08)',
};

// ── CSS variables for page-scoped consumption ─────────────────────────────────
//
// IMPORTANT: MuiCssBaseline.styleOverrides are only injected if a <CssBaseline />
// component mounts under that theme. Mounting CssBaseline here would re-apply
// body-level global styles over the whole app while this page is open, which
// is wrong. Instead we expose the tokens as a plain object and apply them via
// the page root Box's `sx` prop — they cascade to every child, they vanish
// when the page unmounts, and they never touch the platform shell.
export const qbCssVars = {
  // Colors (mockup mui-* aligned)
  '--mui-bg-default':             qb.bgDefault,
  '--mui-bg-paper':               qb.bgPaper,
  '--mui-bg-subtle':              qb.bgSubtle,
  '--mui-bg-elevated':            qb.bgElevated,
  '--mui-bg-lowest':              qb.bgLowest,
  '--mui-border':                 qb.outlineVariant,
  '--mui-border-light':           qb.borderLight,
  '--mui-primary-main':           qb.primary,
  '--mui-primary-light':          qb.primaryFixed,
  '--mui-primary-dark':           '#0f766e',
  '--mui-primary-contrastText':   qb.onPrimary,
  '--mui-secondary-main':         qb.secondary,
  '--mui-secondary-contrastText': '#002c47',
  '--mui-warning-main':           qb.warning,
  '--mui-purple-main':            qb.purple,
  '--mui-text-primary':           qb.onSurface,
  '--mui-text-secondary':         qb.onSurfaceVariant,
  '--mui-text-disabled':          '#54656f',
  // Backwards-compat (old page tokens still referenced)
  '--qb-primary':                 qb.primary,
  '--qb-primary-fixed':           qb.primaryFixed,
  '--qb-on-primary':              qb.onPrimary,
  '--qb-secondary':               qb.secondary,
  '--qb-tertiary':                qb.tertiary,
  '--qb-error':                   qb.error,
  '--qb-warning':                 qb.warning,
  '--qb-purple':                  qb.purple,
  '--qb-surface':                 qb.surface,
  '--qb-surface-container-lowest':qb.surfaceContainerLowest,
  '--qb-surface-container-low':   qb.surfaceContainerLow,
  '--qb-surface-container':       qb.surfaceContainer,
  '--qb-surface-container-high':  qb.surfaceContainerHigh,
  '--qb-surface-container-highest':qb.surfaceContainerHighest,
  '--qb-on-surface':              qb.onSurface,
  '--qb-on-surface-variant':      qb.onSurfaceVariant,
  '--qb-outline':                 qb.outline,
  '--qb-outline-variant':         qb.outlineVariant,
  // Fonts
  '--font-headline-md':           'Inter, sans-serif',
  '--font-data-label':            '"Archivo Narrow", sans-serif',
  '--font-mono-label':            '"JetBrains Mono", monospace',
} as const;

export const qbTheme = createTheme({
  palette: {
    ...darkPalette,
    mode: 'dark',
  },
  shape: { borderRadius: 4 },
} as PaletteOptions, {
  cssVariables: false,
  typography: {
    fontFamily: ['Inter', 'Outfit', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'Helvetica Neue', 'Arial', 'sans-serif'].join(', '),
    h1: { fontFamily: 'Inter, sans-serif', fontWeight: 700, letterSpacing: '-0.02em' },
    h2: { fontFamily: 'Inter, sans-serif', fontWeight: 700, letterSpacing: '-0.02em' },
    h3: { fontFamily: 'Inter, sans-serif', fontWeight: 600, letterSpacing: '-0.01em' },
    h4: { fontFamily: 'Inter, sans-serif', fontWeight: 600, letterSpacing: '-0.01em' },
    h5: { fontFamily: 'Inter, sans-serif', fontWeight: 600 },
    h6: { fontFamily: 'Inter, sans-serif', fontWeight: 600 },
    subtitle1: { fontFamily: 'Inter, sans-serif', fontWeight: 500 },
    subtitle2: { fontFamily: 'Inter, sans-serif', fontWeight: 600 },
    body1: { fontFamily: 'Inter, sans-serif' },
    body2: { fontFamily: 'Inter, sans-serif' },
    caption: { fontFamily: 'Inter, sans-serif', letterSpacing: '0.02em' },
    overline: { fontFamily: 'Inter, sans-serif', fontWeight: 700, letterSpacing: '0.1em' },
    button: { fontFamily: 'Inter, sans-serif', fontWeight: 600, letterSpacing: '-0.01em', textTransform: 'none' },
  },
  components: {
    // No MuiCssBaseline here — see qbCssVars comment above.
    MuiTypography: {
      styleOverrides: {
        root: { color: darkPalette.text.primary },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          backgroundImage: 'none',
          backgroundColor: qb.surfaceContainer,
          color: qb.onSurface,
        },
      },
    },
    MuiButton: {
      styleOverrides: {
        root: { borderRadius: 6, fontWeight: 600, fontSize: '0.875rem', textTransform: 'none' },
        contained: {
          backgroundColor: qb.primary,
          color: qb.onPrimary,
          boxShadow: 'none',
          '&:hover': { backgroundColor: qb.primaryFixed, boxShadow: 'none' },
        },
        outlined: {
          borderColor: alpha(qb.outline, 0.5),
          color: qb.onSurface,
          '&:hover': { borderColor: qb.outline, backgroundColor: alpha(qb.onSurface, 0.04) },
        },
        text: {
          color: qb.primary,
          '&:hover': { backgroundColor: alpha(qb.primary, 0.08) },
        },
      },
    },
  },
});
