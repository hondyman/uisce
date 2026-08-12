import {
  createTheme,
  Theme,
  alpha,
  PaletteOptions,
} from '@mui/material/styles';
import {
  lightPalette,
  darkPalette,
  uisceAmber,
  uisceOcean,
  uisceAbyss,
  uisceTide,
  uisceGold,
  uisceInk,
  categoryColors,
  semanticSuccess,
  semanticWarning,
  semanticError,
} from './palette';

// ── Type Augmentation ───────────────────────────────────────────────────────────
// Allow theme.palette.categoryXXX in any component without casting.
declare module '@mui/material/styles' {
  interface Palette {
    tertiary: Palette['primary'];
    // 8 category accent colors
    categoryPlatform:    Palette['primary'];
    categoryCatalog:    Palette['primary'];
    categoryBuild:     Palette['primary'];
    categoryStudio:     Palette['primary'];
    categoryOperations: Palette['primary'];
    categoryIntelligence: Palette['primary'];
    categoryConsume:   Palette['primary'];
    categoryCalendar:   Palette['primary'];
  }
  interface PaletteOptions {
    tertiary?: PaletteOptions['primary'];
    categoryPlatform?:    PaletteOptions['primary'];
    categoryCatalog?:    PaletteOptions['primary'];
    categoryBuild?:     PaletteOptions['primary'];
    categoryStudio?:    PaletteOptions['primary'];
    categoryOperations?: PaletteOptions['primary'];
    categoryIntelligence?: PaletteOptions['primary'];
    categoryConsume?:   PaletteOptions['primary'];
    categoryCalendar?:  PaletteOptions['primary'];
  }
}

// ── Category palette builders ───────────────────────────────────────────────────

function categoryLight(key: keyof typeof categoryColors) {
  const c = categoryColors[key];
  return {
    main: c.main,
    light: c.light,
    dark: c.dark,
    contrastText: '#ffffff',
  };
}
function categoryDark(key: keyof typeof categoryColors) {
  const c = categoryColors[key];
  return {
    main: c.dark,
    light: c.main,
    dark: c.light,
    contrastText: '#ffffff',
  };
}

// ── Shared component style helpers ───────────────────────────────────────────

const BASE_FONT_FAMILY = [
  'Inter',
  'Outfit',
  '-apple-system',
  'BlinkMacSystemFont',
  '"Segoe UI"',
  'Roboto',
  'Helvetica Neue',
  'Arial',
  'sans-serif',
].join(', ');

function oceanFocusRing(hex: string) {
  return {
    '&:focus-visible': {
      outline: `2px solid ${hex}`,
      outlineOffset: 2,
    },
  };
}

// ── Theme factory ─────────────────────────────────────────────────────────────

export function createUisceTheme(mode: 'light' | 'dark'): Theme {
  const p = mode === 'light' ? lightPalette : darkPalette;

  const palette: PaletteOptions = {
    mode,
    primary: p.primary,
    secondary: p.secondary,
    tertiary: p.tertiary,
    success: p.success,
    warning: p.warning,
    error: p.error,
    background: p.background,
    text: p.text,
    divider: p.divider,
    action: p.action,
    // Category colors
    categoryPlatform:     mode === 'light' ? categoryLight('platform')     : categoryDark('platform'),
    categoryCatalog:     mode === 'light' ? categoryLight('catalog')     : categoryDark('catalog'),
    categoryBuild:       mode === 'light' ? categoryLight('build')       : categoryDark('build'),
    categoryStudio:      mode === 'light' ? categoryLight('studio')     : categoryDark('studio'),
    categoryOperations:  mode === 'light' ? categoryLight('operations') : categoryDark('operations'),
    categoryIntelligence:mode === 'light' ? categoryLight('intelligence'): categoryDark('intelligence'),
    categoryConsume:     mode === 'light' ? categoryLight('consume')     : categoryDark('consume'),
    categoryCalendar:    mode === 'light' ? categoryLight('calendar')    : categoryDark('calendar'),
  };

  // ── Water theme color variables ──
  const oceanMain   = uisceOcean.main;           // #00C9C8 — teal primary
  const oceanDark   = uisceOcean.dark;           // #00706F
  const oceanGlow   = mode === 'light' ? uisceOcean.glowLight : uisceOcean.glow;
  const goldMain    = uisceGold.main;            // #F5A623 — warm amber/gold
  const goldDark    = uisceGold.dark;

  // Backward compat aliases
  const amberMain   = oceanMain;
  const amberDark   = oceanDark;
  const amberGlow   = oceanGlow;

  // Surface backgrounds
  const surfaceBorder = mode === 'light' ? uisceTide.border : uisceAbyss.border;
  const surfaceBg    = mode === 'light' ? '#FFFFFF'         : uisceAbyss[800];  // #071526
  const deepBg       = mode === 'light' ? uisceTide[50]     : uisceAbyss[900];  // #050D1A
  const charcoalBg   = mode === 'light' ? uisceTide[100]    : uisceAbyss[950]; // #020810

  return createTheme({
    palette,
    shape: { borderRadius: 10 },
    typography: {
      fontFamily: BASE_FONT_FAMILY,
      button: {
        textTransform: 'none',
        fontWeight: 600,
        letterSpacing: '-0.01em',
      },
      h1: { fontWeight: 800, letterSpacing: '-0.03em' },
      h2: { fontWeight: 700, letterSpacing: '-0.02em' },
      h3: { fontWeight: 700, letterSpacing: '-0.02em' },
      h4: { fontWeight: 700, letterSpacing: '-0.01em' },
      h5: { fontWeight: 600, letterSpacing: '-0.01em' },
      h6: { fontWeight: 600 },
      subtitle1: { fontWeight: 500, letterSpacing: '-0.01em' },
      subtitle2: { fontWeight: 600 },
      body1: { letterSpacing: '-0.005em' },
      body2: { letterSpacing: '0' },
      caption: { letterSpacing: '0.02em' },
      overline: { fontWeight: 700, letterSpacing: '0.1em' },
    },
    shadows: (() => {
      const shadowBase = mode === 'light' ? surfaceBg : '#000000';
      const a = (opacity: number) => alpha(shadowBase, opacity);
      return [
        'none',
        `0 1px 2px ${a(0.06)}`,
        `0 1px 3px ${a(0.08)}, 0 2px 6px ${a(0.05)}`,
        `0 2px 4px ${a(0.08)}, 0 4px 12px ${a(0.06)}`,
        `0 4px 6px ${a(0.08)}, 0 6px 16px ${a(0.07)}`,
        `0 6px 8px ${a(0.09)}, 0 8px 24px ${a(0.08)}`,
        `0 8px 10px ${a(0.10)}, 0 10px 32px ${a(0.09)}`,
        `0 10px 14px ${a(0.11)}, 0 12px 40px ${a(0.10)}`,
        `0 12px 16px ${a(0.12)}, 0 14px 48px ${a(0.11)}`,
        `0 14px 18px ${a(0.13)}, 0 16px 56px ${a(0.12)}`,
        `0 16px 20px ${a(0.14)}, 0 18px 64px ${a(0.13)}`,
        `0 18px 24px ${a(0.15)}, 0 20px 72px ${a(0.14)}`,
        `0 20px 28px ${a(0.16)}, 0 22px 80px ${a(0.15)}`,
        `0 22px 32px ${a(0.17)}, 0 24px 88px ${a(0.16)}`,
        `0 24px 36px ${a(0.18)}, 0 26px 96px ${a(0.17)}`,
        `0 26px 40px ${a(0.19)}, 0 28px 104px ${a(0.18)}`,
        `0 28px 44px ${a(0.20)}, 0 30px 112px ${a(0.19)}`,
        `0 30px 48px ${a(0.21)}, 0 32px 120px ${a(0.20)}`,
        `0 32px 52px ${a(0.22)}, 0 34px 128px ${a(0.21)}`,
        `0 34px 56px ${a(0.23)}, 0 36px 136px ${a(0.22)}`,
        `0 36px 60px ${a(0.24)}, 0 38px 144px ${a(0.23)}`,
        `0 38px 64px ${a(0.25)}, 0 40px 152px ${a(0.24)}`,
        `0 40px 68px ${a(0.26)}, 0 42px 160px ${a(0.25)}`,
        `0 42px 72px ${a(0.27)}, 0 44px 168px ${a(0.26)}`,
        `0 44px 76px ${a(0.28)}, 0 46px 176px ${a(0.27)}`,
      ];
    })(),

    components: {
      // ── Global ────────────────────────────────────────────────────────────
      MuiCssBaseline: {
        styleOverrides: {
          ':root': {
            colorScheme: mode,
            // ── Uisce design tokens ──────────────────────────────────────
            '--uisce-ocean':     uisceOcean.main,
            '--uisce-ocean-glow': uisceOcean.glow,
            '--uisce-gold':      uisceGold.main,
            '--uisce-gold-glow': uisceGold.glow,
            '--uisce-seafoam':   '#0AF5B0',
            '--uisce-abyss':     uisceAbyss[900],
            '--uisce-tide':      uisceTide[50],
            // ── Core / Custom indicators ─────────────────────────────────
            '--color-core':      uisceGold.main,
            '--color-custom':    uisceOcean.main,
            '--color-core-glow': uisceGold.glow,
            '--color-custom-glow': uisceOcean.glow,
          },
          html: {
            scrollBehavior: 'smooth',
          },
          // ── Aurora wave animation (dark) / Tidal shimmer (light) ──────
          '@keyframes uisce-aurora': {
            '0%':   { backgroundPosition: '0% 50%' },
            '50%':  { backgroundPosition: '100% 50%' },
            '100%': { backgroundPosition: '0% 50%' },
          },
          '@keyframes uisce-ripple': {
            '0%':   { transform: 'scale(1)', opacity: 0.4 },
            '50%':  { transform: 'scale(1.04)', opacity: 0.6 },
            '100%': { transform: 'scale(1)', opacity: 0.4 },
          },
          '@keyframes uisce-flow': {
            '0%':   { backgroundPosition: '200% center' },
            '100%': { backgroundPosition: '-200% center' },
          },
          // Body background — deep ocean aurora in dark, tidal wash in light
          body: {
            background: mode === 'dark'
              ? `radial-gradient(ellipse at 20% 20%, ${alpha(uisceOcean.main, 0.06)} 0%, transparent 50%),
                 radial-gradient(ellipse at 80% 80%, ${alpha('#0AF5B0', 0.04)} 0%, transparent 50%),
                 radial-gradient(ellipse at 60% 40%, ${alpha(uisceGold.main, 0.03)} 0%, transparent 40%),
                 ${uisceAbyss[900]}`
              : `radial-gradient(ellipse at 30% 0%, ${alpha(uisceOcean.main, 0.06)} 0%, transparent 60%),
                 radial-gradient(ellipse at 70% 100%, ${alpha(uisceOcean[300], 0.08)} 0%, transparent 50%),
                 ${uisceTide[50]}`,
            backgroundAttachment: 'fixed',
          },
          '::-webkit-scrollbar': {
            width: 6,
            height: 6,
          },
          '::-webkit-scrollbar-track': {
            background: mode === 'light' ? uisceTide[100] : uisceAbyss[900],
          },
          '::-webkit-scrollbar-thumb': {
            background: mode === 'light'
              ? `${uisceOcean.main}55`
              : `${uisceOcean.main}44`,
            borderRadius: 3,
            '&:hover': {
              background: `${uisceOcean.main}99`,
            },
          },
          '::selection': {
            background: mode === 'light'
              ? alpha(uisceOcean.main, 0.20)
              : alpha(uisceOcean.main, 0.35),
            color: mode === 'light' ? uisceAbyss[900] : '#E8F4FF',
          },
          ':focus-visible': {
            outline: `2px solid ${oceanMain}`,
            outlineOffset: 2,
          },
          // ── Glassmorphism utility class ────────────────────────────────
          '.uisce-glass': {
            background: mode === 'dark'
              ? `rgba(7, 21, 38, 0.75)`
              : `rgba(255, 255, 255, 0.80)`,
            backdropFilter: 'blur(16px) saturate(180%)',
            WebkitBackdropFilter: 'blur(16px) saturate(180%)',
            border: `1px solid ${mode === 'dark' ? uisceAbyss.border : uisceTide.border}`,
          },
          // ── Ocean glow utilities ───────────────────────────────────────
          '.uisce-glow-ocean': {
            boxShadow: `0 0 20px ${alpha(uisceOcean.main, 0.4)}, 0 0 40px ${alpha(uisceOcean.main, 0.2)}`,
          },
          '.uisce-glow-gold': {
            boxShadow: `0 0 20px ${alpha(uisceGold.main, 0.4)}, 0 0 40px ${alpha(uisceGold.main, 0.2)}`,
          },
          // ── Core (gold) / Custom (teal) indicator strips ───────────────
          '.indicator-core': {
            borderLeft: `3px solid ${uisceGold.main}`,
            boxShadow: `inset 3px 0 12px ${alpha(uisceGold.main, 0.15)}`,
          },
          '.indicator-custom': {
            borderLeft: `3px solid ${uisceOcean.main}`,
            boxShadow: `inset 3px 0 12px ${alpha(uisceOcean.main, 0.15)}`,
          },
        },
      },

      // ── AppBar & Toolbar ─────────────────────────────────────────────────
      MuiAppBar: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            background: mode === 'light'
              ? 'rgba(240,250,252,0.88)'
              : `linear-gradient(180deg, ${uisceAbyss[950]} 0%, ${uisceAbyss[900]} 100%)`,
            backdropFilter: 'blur(20px) saturate(180%)',
            WebkitBackdropFilter: 'blur(20px) saturate(180%)',
            borderBottom: `1px solid ${surfaceBorder}`,
            boxShadow: 'none',
            color: p.text.primary,
            // Teal light seam at base — like light refracted through water
            '&::after': {
              content: '""',
              position: 'absolute',
              bottom: 0,
              left: 0,
              right: 0,
              height: '1px',
              background: mode === 'dark'
                ? `linear-gradient(90deg, transparent 0%, ${uisceOcean.main} 30%, ${alpha('#0AF5B0', 0.8)} 70%, transparent 100%)`
                : `linear-gradient(90deg, transparent 0%, ${uisceOcean.main} 30%, ${uisceOcean[400]} 70%, transparent 100%)`,
              opacity: mode === 'dark' ? 0.45 : 0.30,
            },
          },
        },
      },
      MuiToolbar: {
        styleOverrides: {
          root: {
            minHeight: '56px !important',
            paddingLeft: '16px !important',
            paddingRight: '16px !important',
          },
        },
      },

      // ── Buttons ────────────────────────────────────────────────────────────
      MuiButton: {
        defaultProps: { disableElevation: true },
        styleOverrides: {
          root: {
            borderRadius: 8,
            padding: '7px 18px',
            fontSize: '0.875rem',
            fontWeight: 600,
            letterSpacing: '-0.01em',
            transition: 'all 150ms cubic-bezier(0.4,0,0.2,1)',
            '&:hover': {
              transform: 'translateY(-1px)',
            },
          },
          contained: {
            background: `linear-gradient(135deg, ${oceanMain} 0%, ${oceanDark} 100%)`,
            color: '#FFFFFF',
            boxShadow: `0 2px 8px ${alpha(oceanMain, 0.35)}`,
            '&:hover': {
              background: `linear-gradient(135deg, ${alpha(oceanMain, 0.85)} 0%, ${oceanMain} 100%)`,
              boxShadow: `0 4px 20px ${alpha(oceanMain, 0.50)}`,
            },
            '&:active': {
              transform: 'translateY(0)',
            },
          },
          containedSecondary: {
            background: `linear-gradient(135deg, ${goldMain} 0%, ${goldDark} 100%)`,
            color: mode === 'light' ? '#1A0A00' : '#050D1A',
            boxShadow: `0 2px 8px ${alpha(goldMain, 0.35)}`,
            '&:hover': {
              boxShadow: `0 4px 20px ${alpha(goldMain, 0.50)}`,
            },
          },
          outlined: {
            borderColor: alpha(oceanMain, 0.60),
            color: oceanMain,
            '&:hover': {
              borderColor: oceanMain,
              background: alpha(oceanMain, 0.06),
              boxShadow: `0 0 12px ${alpha(oceanMain, 0.15)}`,
            },
          },
          outlinedSecondary: {
            borderColor: alpha(goldMain, 0.50),
            color: goldMain,
            '&:hover': {
              borderColor: goldMain,
              background: alpha(goldMain, 0.06),
            },
          },
          text: {
            color: oceanMain,
            '&:hover': {
              background: alpha(oceanMain, 0.06),
            },
          },
          textSecondary: {
            color: p.text.secondary,
            '&:hover': {
              background: alpha(oceanMain, 0.04),
              color: p.text.primary,
            },
          },
          sizeSmall: {
            fontSize: '0.8125rem',
            padding: '4px 12px',
          },
          sizeLarge: {
            fontSize: '0.9375rem',
            padding: '10px 24px',
          },
          ...oceanFocusRing(oceanMain),
        },
      },
      MuiIconButton: {
        styleOverrides: {
          root: {
            borderRadius: 8,
            transition: 'all 150ms ease',
            color: p.text.secondary,
            '&:hover': {
              background: alpha(oceanMain, 0.08),
              color: oceanMain,
            },
            ...oceanFocusRing(oceanMain),
          },
        },
      },
      MuiButtonBase: {
        defaultProps: { disableRipple: false },
      },

      // ── Chips ──────────────────────────────────────────────────────────────
      MuiChip: {
        styleOverrides: {
          root: {
            fontWeight: 600,
            fontSize: '0.75rem',
            letterSpacing: '0.01em',
            borderRadius: 6,
          },
          filled: {
            backgroundColor: alpha(oceanMain, 0.10),
            color: mode === 'dark' ? uisceOcean.light : oceanMain,
            border: `1px solid ${alpha(oceanMain, 0.30)}`,
            '&:hover': {
              backgroundColor: alpha(oceanMain, 0.18),
              boxShadow: `0 0 8px ${alpha(oceanMain, 0.25)}`,
            },
          },
          outlined: {
            borderColor: alpha(oceanMain, 0.45),
            color: mode === 'dark' ? uisceOcean.light : oceanMain,
            '&:hover': {
              backgroundColor: alpha(oceanMain, 0.08),
              borderColor: oceanMain,
            },
          },
          sizeSmall: {
            height: 22,
            fontSize: '0.6875rem',
          },
        },
      },

      // ── Paper / Card ─────────────────────────────────────────────────────
      MuiPaper: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            backgroundImage: 'none',
            border: `1px solid ${surfaceBorder}`,
            borderRadius: 12,
            // Subtle water-glass sheen in dark mode
            ...(mode === 'dark' && {
              background: `linear-gradient(135deg, ${uisceAbyss[800]} 0%, ${uisceAbyss[700]} 100%)`,
            }),
          },
          elevation1: {
            boxShadow: mode === 'dark'
              ? `0 2px 8px rgba(0,0,0,0.3), 0 0 0 1px ${uisceAbyss.border}`
              : `0 1px 3px ${alpha(uisceOcean.main, 0.06)}, 0 2px 8px ${alpha(uisceOcean.main, 0.04)}`,
          },
        },
      },
      MuiCard: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            backgroundImage: 'none',
            border: `1px solid ${surfaceBorder}`,
            borderRadius: 14,
            boxShadow: mode === 'light'
              ? `0 1px 3px ${alpha(uisceOcean.main, 0.06)}, 0 2px 8px ${alpha(uisceOcean.main, 0.04)}`
              : `0 2px 8px rgba(0,0,0,0.30), 0 4px 16px rgba(0,0,0,0.20)`,
            transition: 'box-shadow 200ms ease, border-color 200ms ease, transform 200ms ease',
            '&:hover': {
              borderColor: alpha(oceanMain, mode === 'light' ? 0.30 : 0.40),
              boxShadow: mode === 'light'
                ? `0 4px 16px ${alpha(oceanMain, 0.12)}, 0 8px 32px ${alpha(oceanMain, 0.06)}`
                : `0 4px 20px rgba(0,0,0,0.50), 0 8px 40px ${alpha(oceanMain, 0.15)}`,
            },
          },
        },
      },

      // ── Menus ──────────────────────────────────────────────────────────────
      MuiMenu: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          paper: {
            background: mode === 'light'
              ? 'rgba(240,250,252,0.96)'
              : `rgba(7, 21, 38, 0.95)`,
            backdropFilter: 'blur(20px) saturate(180%)',
            WebkitBackdropFilter: 'blur(20px) saturate(180%)',
            border: `1px solid ${surfaceBorder}`,
            borderRadius: 12,
            boxShadow: mode === 'light'
              ? `0 4px 16px ${alpha(uisceOcean.main, 0.08)}, 0 16px 48px ${alpha(uisceOcean.main, 0.06)}`
              : `0 8px 40px rgba(0,0,0,0.65), 0 0 0 1px ${uisceAbyss.border}`,
            padding: '4px 0',
          },
        },
      },
      MuiMenuItem: {
        styleOverrides: {
          root: {
            borderRadius: 6,
            margin: '0 6px',
            padding: '8px 12px',
            fontSize: '0.875rem',
            fontWeight: 500,
            color: p.text.primary,
            borderLeft: `3px solid transparent`,
            transition: 'all 150ms ease',
            '&:hover': {
              background: alpha(oceanMain, 0.08),
              color: mode === 'dark' ? uisceOcean.light : oceanMain,
            },
            '&.Mui-selected': {
              background: alpha(oceanMain, 0.12),
              color: mode === 'dark' ? uisceOcean.light : oceanMain,
              borderLeftColor: oceanMain,
              fontWeight: 600,
              '&:hover': {
                background: alpha(oceanMain, 0.16),
              },
            },
          },
        },
      },

      // ── List / ListItem ───────────────────────────────────────────────────
      MuiList: {
        styleOverrides: {
          root: { padding: '4px 0' },
        },
      },
      MuiListItemButton: {
        styleOverrides: {
          root: {
            borderRadius: 6,
            margin: '0 6px',
            padding: '8px 12px',
            color: p.text.secondary,
            transition: 'all 150ms ease',
            borderLeft: `3px solid transparent`,
            '&:hover': {
              background: alpha(oceanMain, 0.08),
              color: mode === 'dark' ? uisceOcean.light : oceanMain,
            },
            '&.Mui-selected': {
              background: alpha(oceanMain, 0.12),
              color: mode === 'dark' ? uisceOcean.light : oceanMain,
              borderLeftColor: oceanMain,
              fontWeight: 600,
              '&:hover': { background: alpha(oceanMain, 0.16) },
            },
          },
        },
      },

      // ── Drawer ─────────────────────────────────────────────────────────────
      MuiDrawer: {
        styleOverrides: {
          paper: {
            background: mode === 'light'
              ? `linear-gradient(180deg, ${uisceTide[50]} 0%, ${uisceTide[100]} 100%)`
              : `linear-gradient(180deg, ${uisceAbyss[950]} 0%, ${uisceAbyss[900]} 100%)`,
            borderRight: `1px solid ${surfaceBorder}`,
            boxShadow: mode === 'dark'
              ? `2px 0 20px rgba(0,0,0,0.4), 1px 0 0 ${uisceAbyss.border}`
              : `2px 0 12px ${alpha(uisceOcean.main, 0.06)}`,
          },
        },
      },

      // ── Tabs ──────────────────────────────────────────────────────────────
      MuiTabs: {
        styleOverrides: {
          indicator: {
            height: 3,
            borderRadius: '3px 3px 0 0',
            background: `linear-gradient(90deg, ${oceanMain} 0%, ${alpha('#0AF5B0', 0.9)} 100%)`,
            boxShadow: `0 0 8px ${alpha(oceanMain, 0.5)}`,
          },
        },
      },
      MuiTab: {
        styleOverrides: {
          root: {
            textTransform: 'none',
            fontWeight: 600,
            fontSize: '0.875rem',
            minHeight: 44,
            color: p.text.secondary,
            transition: 'all 150ms ease',
            '&.Mui-selected': {
              color: mode === 'dark' ? uisceOcean.light : oceanMain,
              fontWeight: 700,
            },
            '&:hover': {
              color: mode === 'dark' ? uisceOcean.light : oceanMain,
              background: alpha(oceanMain, 0.05),
            },
          },
        },
      },

      // ── Switch ────────────────────────────────────────────────────────────
      MuiSwitch: {
        styleOverrides: {
          root: { padding: 8 },
          switchBase: {
            '&.Mui-checked': {
              color: oceanMain,
              '& + .MuiSwitch-track': {
                background: oceanMain,
                opacity: 0.55,
              },
            },
          },
          track: {
            borderRadius: 12,
            background: mode === 'light'
              ? alpha(uisceAbyss[700], 0.20)
              : alpha('#ffffff', 0.18),
          },
          thumb: {
            width: 18,
            height: 18,
            boxShadow: '0 1px 4px rgba(0,0,0,0.25)',
          },
        },
      },

      // ── Checkbox / Radio ─────────────────────────────────────────────────
      MuiCheckbox: {
        styleOverrides: {
          root: {
            color: alpha(oceanMain, 0.40),
            padding: 6,
            borderRadius: 4,
            transition: 'all 150ms ease',
            '&:hover': { background: alpha(oceanMain, 0.07) },
            '&.Mui-checked': {
              color: oceanMain,
              filter: mode === 'dark' ? `drop-shadow(0 0 4px ${alpha(oceanMain, 0.5)})` : 'none',
            },
          },
        },
      },
      MuiRadio: {
        styleOverrides: {
          root: {
            color: alpha(oceanMain, 0.40),
            padding: 6,
            transition: 'all 150ms ease',
            '&:hover': { background: alpha(oceanMain, 0.07) },
            '&.Mui-checked': { color: oceanMain },
          },
        },
      },

      // ── TextField / Input / Select ───────────────────────────────────────
      MuiTextField: {
        defaultProps: { variant: 'outlined', size: 'medium' },
      },
      MuiOutlinedInput: {
        styleOverrides: {
          root: {
            borderRadius: 8,
            background: mode === 'light'
              ? 'rgba(255,255,255,0.9)'
              : alpha(uisceAbyss[700], 0.60),
            transition: 'all 150ms ease',
            '& fieldset': {
              borderColor: surfaceBorder,
              borderWidth: 1,
              transition: 'border-color 150ms ease, box-shadow 150ms ease',
            },
            '&:hover fieldset': {
              borderColor: alpha(oceanMain, 0.55),
            },
            '&.Mui-focused fieldset': {
              borderColor: oceanMain,
              borderWidth: 2,
            },
            '&.Mui-focused': {
              boxShadow: `0 0 0 3px ${alpha(oceanMain, 0.15)}, 0 0 12px ${alpha(oceanMain, 0.08)}`,
            },
          },
          input: {
            padding: '10px 14px',
            fontSize: '0.875rem',
          },
          notchedOutline: {
            borderColor: surfaceBorder,
          },
        },
      },
      MuiInputBase: {
        styleOverrides: {
          root: {
            fontSize: '0.875rem',
          },
          input: {
            '&::placeholder': {
              color: p.text.disabled,
              opacity: 1,
            },
          },
        },
      },
      MuiInputLabel: {
        styleOverrides: {
          root: {
            fontSize: '0.875rem',
            fontWeight: 500,
            color: p.text.secondary,
            '&.Mui-focused': { color: oceanMain },
          },
        },
      },
      MuiSelect: {
        styleOverrides: {
          select: { padding: '10px 14px' },
        },
      },

      // ── Slider ────────────────────────────────────────────────────────────
      MuiSlider: {
        styleOverrides: {
          root: {
            height: 4,
            '& .MuiSlider-track': {
              background: `linear-gradient(90deg, ${oceanMain} 0%, ${alpha('#0AF5B0', 0.9)} 100%)`,
              border: 'none',
            },
            '& .MuiSlider-rail': {
              background: mode === 'light'
                ? alpha(uisceAbyss[700], 0.12)
                : alpha('#ffffff', 0.15),
              opacity: 1,
            },
            '& .MuiSlider-thumb': {
              width: 16,
              height: 16,
              background: oceanMain,
              boxShadow: `0 1px 4px ${alpha(oceanMain, 0.5)}, 0 0 0 3px ${alpha(oceanMain, 0.15)}`,
              '&:hover, &.Mui-focusVisible': {
                boxShadow: `0 2px 10px ${alpha(oceanMain, 0.6)}, 0 0 0 5px ${alpha(oceanMain, 0.15)}`,
              },
            },
          },
        },
      },

      // ── Progress ────────────────────────────────────────────────────────
      MuiLinearProgress: {
        styleOverrides: {
          root: {
            borderRadius: 4,
            background: mode === 'light'
              ? alpha(uisceOcean.main, 0.10)
              : alpha(uisceOcean.main, 0.12),
          },
          bar: {
            borderRadius: 4,
            background: `linear-gradient(90deg, ${oceanMain} 0%, ${alpha('#0AF5B0', 0.9)} 100%)`,
            boxShadow: mode === 'dark' ? `0 0 8px ${alpha(oceanMain, 0.5)}` : 'none',
          },
        },
      },
      MuiCircularProgress: {
        styleOverrides: {
          root: {
            color: oceanMain,
            filter: mode === 'dark' ? `drop-shadow(0 0 6px ${alpha(oceanMain, 0.5)})` : 'none',
          },
        },
      },

      // ── Alert ────────────────────────────────────────────────────────────
      MuiAlert: {
        styleOverrides: {
          root: {
            borderRadius: 8,
            fontSize: '0.875rem',
            fontWeight: 500,
            border: '1px solid',
          },
          standardSuccess: {
            backgroundColor: alpha(semanticSuccess.light, 0.7),
            borderColor: alpha(semanticSuccess.DEFAULT, 0.30),
            color: mode === 'light' ? semanticSuccess.dark2 : semanticSuccess.dark,
          },
          standardWarning: {
            backgroundColor: alpha(semanticWarning.light, 0.7),
            borderColor: alpha(semanticWarning.DEFAULT, 0.30),
            color: mode === 'light' ? semanticWarning.dark2 : semanticWarning.dark,
          },
          standardError: {
            backgroundColor: alpha(semanticError.light, 0.7),
            borderColor: alpha(semanticError.DEFAULT, 0.30),
            color: mode === 'light' ? semanticError.dark2 : semanticError.dark,
          },
          outlinedSuccess: {
            borderColor: alpha(semanticSuccess.DEFAULT, 0.40),
            color: mode === 'light' ? semanticSuccess.DEFAULT : semanticSuccess.dark,
          },
          outlinedWarning: {
            borderColor: alpha(semanticWarning.DEFAULT, 0.40),
            color: mode === 'light' ? semanticWarning.DEFAULT : semanticWarning.dark,
          },
          outlinedError: {
            borderColor: alpha(semanticError.DEFAULT, 0.40),
            color: mode === 'light' ? semanticError.DEFAULT : semanticError.dark,
          },
          icon: { fontSize: 20 },
          message: { padding: '2px 0' },
        },
      },

      // ── Tooltip ──────────────────────────────────────────────────────────
      MuiTooltip: {
        styleOverrides: {
          tooltip: {
            background: mode === 'light'
              ? `linear-gradient(135deg, ${uisceAbyss[800]} 0%, ${uisceAbyss[900]} 100%)`
              : `linear-gradient(135deg, ${alpha(uisceAbyss[700], 0.98)} 0%, ${alpha(uisceAbyss[800], 0.95)} 100%)`,
            color: '#E8F4FF',
            fontSize: '0.75rem',
            fontWeight: 500,
            borderRadius: 8,
            padding: '6px 10px',
            boxShadow: `0 4px 16px rgba(0,0,0,0.4), 0 0 0 1px ${alpha(oceanMain, 0.20)}`,
            border: `1px solid ${alpha(oceanMain, 0.25)}`,
            backdropFilter: 'blur(8px)',
          },
          arrow: {
            color: uisceAbyss[800],
          },
        },
      },

      // ── Dialog ───────────────────────────────────────────────────────────
      MuiDialog: {
        styleOverrides: {
          paper: {
            background: mode === 'light'
              ? 'rgba(250,253,254,0.98)'
              : `linear-gradient(145deg, ${uisceAbyss[800]} 0%, ${uisceAbyss[700]} 100%)`,
            border: `1px solid ${surfaceBorder}`,
            borderRadius: 16,
            boxShadow: mode === 'light'
              ? `0 20px 60px ${alpha(uisceOcean.main, 0.12)}, 0 8px 24px ${alpha(uisceOcean.main, 0.08)}`
              : `0 24px 80px rgba(0,0,0,0.75), 0 8px 40px ${alpha(oceanMain, 0.15)}, 0 0 0 1px ${uisceAbyss.border}`,
            backdropFilter: 'blur(20px)',
            WebkitBackdropFilter: 'blur(20px)',
          },
        },
      },
      MuiDialogTitle: {
        styleOverrides: {
          root: {
            fontSize: '1.1rem',
            fontWeight: 700,
            padding: '20px 24px 12px',
          },
        },
      },
      MuiDialogContent: {
        styleOverrides: {
          root: { padding: '12px 24px 16px' },
        },
      },
      MuiDialogActions: {
        styleOverrides: {
          root: { padding: '12px 24px 20px', gap: 8 },
        },
      },

      // ── Divider ─────────────────────────────────────────────────────────
      MuiDivider: {
        styleOverrides: {
          root: { borderColor: surfaceBorder },
          vertical: { margin: '4px 8px' },
        },
      },

      // ── Badge ───────────────────────────────────────────────────────────
      MuiBadge: {
        styleOverrides: {
          badge: {
            fontWeight: 700,
            fontSize: '0.65rem',
          },
          colorSuccess: {
            backgroundColor: semanticSuccess.DEFAULT,
            color: '#ffffff',
          },
          colorWarning: {
            backgroundColor: semanticWarning.DEFAULT,
            color: '#ffffff',
          },
          colorError: {
            backgroundColor: semanticError.DEFAULT,
            color: '#ffffff',
          },

        },
      },

      // ── Snackbar ─────────────────────────────────────────────────────────
      MuiSnackbar: {
        styleOverrides: {
          root: { bottom: 24 },
        },
      },
      MuiSnackbarContent: {
        styleOverrides: {
          root: {
            borderRadius: 8,
            fontSize: '0.875rem',
            fontWeight: 500,
            boxShadow: mode === 'light'
              ? `0 8px 24px ${alpha(surfaceBg, 0.20)}`
              : '0 8px 24px rgba(0,0,0,0.5)',
          },
        },
      },

      // ── Stepper ─────────────────────────────────────────────────────────
      MuiStepIcon: {
        styleOverrides: {
          root: {
            color: alpha(oceanMain, 0.25),
            '&.Mui-completed': {
              color: oceanMain,
              filter: mode === 'dark' ? `drop-shadow(0 0 4px ${alpha(oceanMain, 0.5)})` : 'none',
            },
            '&.Mui-active': {
              color: oceanMain,
              filter: mode === 'dark' ? `drop-shadow(0 0 6px ${alpha(oceanMain, 0.6)})` : 'none',
            },
          },
        },
      },
      MuiStepLabel: {
        styleOverrides: {
          label: {
            fontWeight: 500,
            fontSize: '0.875rem',
            '&.Mui-completed': { fontWeight: 600, color: mode === 'dark' ? uisceOcean.light : oceanMain },
            '&.Mui-active': { fontWeight: 700, color: mode === 'dark' ? uisceOcean.light : oceanMain },
          },
        },
      },

      // ── Table ───────────────────────────────────────────────────────────
      MuiTableHead: {
        styleOverrides: {
          root: {
            background: mode === 'light'
              ? uisceAbyss[800]   // deep ocean header in light too
              : uisceAbyss[950],  // hadal zone in dark
            '& .MuiTableCell-root': {
              color: mode === 'dark' ? uisceOcean.light : '#E8F4FF',
              fontWeight: 700,
              fontSize: '0.8125rem',
              letterSpacing: '0.06em',
              textTransform: 'uppercase',
              borderBottom: `1px solid ${alpha(oceanMain, 0.25)}`,
              padding: '10px 16px',
            },
          },
        },
      },
      MuiTableCell: {
        styleOverrides: {
          root: {
            fontSize: '0.875rem',
            borderColor: surfaceBorder,
            padding: '10px 16px',
          },
          head: {
            fontWeight: 700,
            background: 'transparent',
          },
        },
      },
      MuiTableRow: {
        styleOverrides: {
          root: {
            transition: 'background 120ms ease',
            '&:hover': {
              background: alpha(oceanMain, 0.05),
            },
            '&.Mui-selected': {
              background: alpha(oceanMain, 0.09),
              '&:hover': { background: alpha(oceanMain, 0.13) },
            },
          },
        },
      },

      // ── Accordion ──────────────────────────────────────────────────────
      MuiAccordion: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            border: `1px solid ${surfaceBorder}`,
            borderRadius: '10px !important',
            '&:before': { display: 'none' },
            '&.Mui-expanded': {
              margin: '0 0 8px 0',
            },
          },
        },
      },
      MuiAccordionSummary: {
        styleOverrides: {
          root: {
            fontWeight: 600,
            minHeight: 48,
            '&.Mui-expanded': { minHeight: 48 },
          },
          content: {
            margin: '12px 0',
            '&.Mui-expanded': { margin: '12px 0' },
          },
        },
      },
      MuiAccordionDetails: {
        styleOverrides: {
          root: { padding: '0 16px 16px' },
        },
      },

      // ── Tabs (MuiTab is from @mui/material; MuiTabList/TabPanel/TabContext are from @mui/lab) ─
      // Applied via sx prop where used, not via Components overrides.

      // ── Pagination ────────────────────────────────────────────────────────
      MuiPagination: {
        styleOverrides: {
          root: { marginTop: 16 },
        },
      },
      MuiPaginationItem: {
        styleOverrides: {
          root: {
            fontWeight: 500,
            borderRadius: 6,
            transition: 'all 150ms ease',
            '&:hover': { background: alpha(amberMain, 0.06) },
            '&.Mui-selected': {
              background: amberMain,
              color: mode === 'light' ? uisceInk.DEFAULT : '#ffffff',
              fontWeight: 700,
              '&:hover': { background: amberDark },
            },
          },
        },
      },

      // ── Skeleton ────────────────────────────────────────────────────────
      MuiSkeleton: {
        styleOverrides: {
          root: {
            background: mode === 'light'
              ? alpha(uisceOcean.main, 0.08)
              : alpha('#ffffff', 0.06),
          },
        },
      },

      // ── Backdrop ────────────────────────────────────────────────────────
      MuiBackdrop: {
        styleOverrides: {
          root: {
            background: mode === 'light'
              ? alpha(uisceAbyss[900], 0.35)
              : alpha(charcoalBg, 0.80),
            backdropFilter: 'blur(4px)',
          },
        },
      },

    },
  });
}
