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
  uisceChocolate,
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

function amberFocusRing(hex: string) {
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

  const amberMain   = mode === 'light' ? uisceAmber.main : uisceAmber.light;
  const amberDark   = mode === 'light' ? uisceAmber.dark  : uisceAmber.main;
  const amberGlow  = mode === 'light' ? uisceAmber.glowLight : uisceAmber.glow;
  const surfaceBorder = mode === 'light' ? 'rgba(26,15,8,0.08)'  : 'rgba(255,255,255,0.07)';
  const surfaceBg    = mode === 'light' ? uisceChocolate[700] : '#13161E';
  const charcoalBg   = '#0A0C12';

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
          },
          html: {
            scrollBehavior: 'smooth',
          },
          '::-webkit-scrollbar': {
            width: 6,
            height: 6,
          },
          '::-webkit-scrollbar-track': {
            background: mode === 'light' ? '#f1f5f9' : charcoalBg,
          },
          '::-webkit-scrollbar-thumb': {
            background: mode === 'light'
              ? `${uisceAmber.main}55`
              : `${uisceAmber.main}44`,
            borderRadius: 3,
            '&:hover': {
              background: mode === 'light'
                ? `${uisceAmber.main}88`
                : `${uisceAmber.main}88`,
            },
          },
          '::selection': {
            background: amberGlow,
            color: mode === 'light' ? uisceChocolate[900] : '#ffffff',
          },
          ':focus-visible': {
            outline: `2px solid ${amberMain}`,
            outlineOffset: 2,
          },
        },
      },

      // ── AppBar & Toolbar ─────────────────────────────────────────────────
      MuiAppBar: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            background: mode === 'light'
              ? 'rgba(255,255,255,0.85)'
              : `linear-gradient(180deg, ${charcoalBg} 0%, ${surfaceBg} 100%)`,
            backdropFilter: mode === 'light' ? 'blur(12px) saturate(160%)' : 'blur(16px) saturate(140%)',
            borderBottom: `1px solid ${surfaceBorder}`,
            boxShadow: 'none',
            color: p.text.primary,
            ...(mode === 'dark' && {
              '&::after': {
                content: '""',
                position: 'absolute',
                bottom: 0,
                left: 0,
                right: 0,
                height: '1px',
                background: `linear-gradient(90deg, transparent 0%, ${amberMain} 40%, ${amberMain} 60%, transparent 100%)`,
                opacity: 0.35,
              },
            }),
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
            background: `linear-gradient(135deg, ${amberMain} 0%, ${amberDark} 100%)`,
            color: mode === 'light' ? uisceInk.DEFAULT : '#ffffff',
            boxShadow: `0 2px 8px ${alpha(amberMain, 0.35)}`,
            '&:hover': {
              background: `linear-gradient(135deg, ${amberDark} 0%, ${amberMain} 100%)`,
              boxShadow: `0 4px 16px ${alpha(amberMain, 0.45)}`,
            },
            '&:active': {
              transform: 'translateY(0)',
            },
          },
          containedSecondary: {
            background: mode === 'light'
              ? uisceChocolate[700]
              : `linear-gradient(135deg, ${uisceChocolate[700]} 0%, ${uisceChocolate[600]} 100%)`,
            color: '#ffffff',
            '&:hover': {
              background: mode === 'light' ? uisceChocolate[600] : uisceChocolate[900],
              boxShadow: `0 4px 12px ${alpha(uisceChocolate[700], 0.4)}`,
            },
          },
          outlined: {
            borderColor: amberMain,
            color: amberMain,
            '&:hover': {
              borderColor: amberDark,
              background: alpha(amberMain, 0.06),
            },
          },
          outlinedSecondary: {
            borderColor: surfaceBorder,
            color: p.text.primary,
            '&:hover': {
              borderColor: mode === 'light' ? uisceChocolate[600] : uisceChocolate[600],
              background: alpha(surfaceBg, 0.04),
            },
          },
          text: {
            color: amberMain,
            '&:hover': {
              background: alpha(amberMain, 0.06),
            },
          },
          textSecondary: {
            color: p.text.secondary,
            '&:hover': {
              background: alpha(surfaceBg, 0.04),
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
          ...amberFocusRing(amberMain),
        },
      },
      MuiIconButton: {
        styleOverrides: {
          root: {
            borderRadius: 8,
            transition: 'all 150ms ease',
            color: p.text.secondary,
            '&:hover': {
              background: alpha(amberMain, 0.08),
              color: amberMain,
            },
            ...amberFocusRing(amberMain),
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
            backgroundColor: alpha(amberMain, 0.10),
            color: amberMain,
            border: `1px solid ${alpha(amberMain, 0.25)}`,
            '&:hover': {
              backgroundColor: alpha(amberMain, 0.16),
            },
          },
          outlined: {
            borderColor: alpha(amberMain, 0.40),
            color: amberMain,
            '&:hover': {
              backgroundColor: alpha(amberMain, 0.08),
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
          },
          elevation1: {
            boxShadow: 'none',
            border: `1px solid ${surfaceBorder}`,
          },
        },
      },
      MuiCard: {
        defaultProps: { elevation: 0 },
        styleOverrides: {
          root: {
            backgroundImage: 'none',
            border: `1px solid ${surfaceBorder}`,
            borderRadius: 12,
            boxShadow: mode === 'light'
              ? '0 1px 3px rgba(13,27,110,0.06), 0 2px 8px rgba(13,27,110,0.04)'
              : '0 2px 8px rgba(0,0,0,0.3), 0 4px 16px rgba(0,0,0,0.2)',
            transition: 'box-shadow 200ms ease, border-color 200ms ease',
            '&:hover': {
              borderColor: mode === 'light'
                ? 'rgba(13,27,110,0.16)'
                : alpha(amberMain, 0.40),
              boxShadow: mode === 'light'
                ? '0 4px 12px rgba(13,27,110,0.10), 0 8px 24px rgba(13,27,110,0.06)'
                : `0 4px 16px rgba(0,0,0,0.5), 0 8px 32px ${alpha(amberMain, 0.12)}`,
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
              ? 'rgba(255,255,255,0.96)'
              : surfaceBg,
            backdropFilter: mode === 'light' ? 'blur(12px)' : 'blur(20px)',
            WebkitBackdropFilter: mode === 'light' ? 'blur(12px)' : 'blur(20px)',
            border: `1px solid ${surfaceBorder}`,
            borderRadius: 10,
            boxShadow: mode === 'light'
              ? '0 4px 12px rgba(13,27,110,0.08), 0 16px 40px rgba(13,27,110,0.10)'
              : '0 8px 32px rgba(0,0,0,0.6)',
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
              background: alpha(amberMain, 0.06),
              color: amberMain,
            },
            '&.Mui-selected': {
              background: alpha(amberMain, 0.10),
              color: amberMain,
              borderLeftColor: amberMain,
              fontWeight: 600,
              '&:hover': {
                background: alpha(amberMain, 0.14),
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
              background: alpha(amberMain, 0.06),
              color: amberMain,
            },
            '&.Mui-selected': {
              background: alpha(amberMain, 0.10),
              color: amberMain,
              borderLeftColor: amberMain,
              fontWeight: 600,
              '&:hover': { background: alpha(amberMain, 0.14) },
            },
          },
        },
      },

      // ── Drawer ─────────────────────────────────────────────────────────────
      MuiDrawer: {
        styleOverrides: {
          paper: {
            background: mode === 'light' ? '#ffffff' : charcoalBg,
            borderRight: `1px solid ${surfaceBorder}`,
            boxShadow: mode === 'dark' ? '2px 0 8px rgba(0,0,0,0.3)' : 'none',
          },
        },
      },

      // ── Tabs ──────────────────────────────────────────────────────────────
      MuiTabs: {
        styleOverrides: {
          indicator: {
            height: 3,
            borderRadius: '3px 3px 0 0',
            background: `linear-gradient(90deg, ${amberMain} 0%, ${amberDark} 100%)`,
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
            transition: 'color 150ms ease',
            '&.Mui-selected': {
              color: amberMain,
              fontWeight: 700,
            },
            '&:hover': {
              color: amberMain,
              background: alpha(amberMain, 0.04),
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
              color: amberMain,
              '& + .MuiSwitch-track': {
                background: amberMain,
                opacity: 0.5,
              },
            },
          },
          track: {
            borderRadius: 12,
            background: mode === 'light'
              ? uisceChocolate[700]
              : alpha('#ffffff', 0.20),
          },
          thumb: {
            width: 18,
            height: 18,
            boxShadow: '0 1px 3px rgba(0,0,0,0.2)',
          },
        },
      },

      // ── Checkbox / Radio ─────────────────────────────────────────────────
      MuiCheckbox: {
        styleOverrides: {
          root: {
            color: surfaceBorder,
            padding: 6,
            borderRadius: 4,
            transition: 'all 150ms ease',
            '&:hover': { background: alpha(amberMain, 0.06) },
            '&.Mui-checked': { color: amberMain },
          },
        },
      },
      MuiRadio: {
        styleOverrides: {
          root: {
            color: surfaceBorder,
            padding: 6,
            transition: 'all 150ms ease',
            '&:hover': { background: alpha(amberMain, 0.06) },
            '&.Mui-checked': { color: amberMain },
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
            background: mode === 'light' ? '#ffffff' : alpha(uisceChocolate[700], 0.4),
            transition: 'all 150ms ease',
            '& fieldset': {
              borderColor: surfaceBorder,
              borderWidth: 1,
            },
            '&:hover fieldset': {
              borderColor: alpha(amberMain, 0.5),
            },
            '&.Mui-focused fieldset': {
              borderColor: amberMain,
              borderWidth: 2,
            },
            '&.Mui-focused': {
              boxShadow: `0 0 0 3px ${alpha(amberMain, 0.15)}`,
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
            '&.Mui-focused': { color: amberMain },
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
              background: amberMain,
              border: 'none',
            },
            '& .MuiSlider-rail': {
              background: mode === 'light'
                ? uisceChocolate[700]
                : alpha('#ffffff', 0.15),
              opacity: 1,
            },
            '& .MuiSlider-thumb': {
              width: 16,
              height: 16,
              background: amberMain,
              boxShadow: `0 1px 4px ${alpha(amberMain, 0.4)}`,
              '&:hover, &.Mui-focusVisible': {
                boxShadow: `0 2px 8px ${alpha(amberMain, 0.5)}`,
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
              ? alpha(uisceChocolate[700], 0.08)
              : alpha('#ffffff', 0.10),
          },
          bar: {
            borderRadius: 4,
            background: `linear-gradient(90deg, ${amberMain} 0%, ${amberDark} 100%)`,
          },
        },
      },
      MuiCircularProgress: {
        styleOverrides: {
          root: {
            color: amberMain,
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
              ? `linear-gradient(135deg, ${uisceChocolate[700]} 0%, ${uisceChocolate[900]} 100%)`
              : `linear-gradient(135deg, ${alpha(surfaceBg, 0.98)} 0%, ${alpha(charcoalBg, 0.95)} 100%)`,
            color: '#ffffff',
            fontSize: '0.75rem',
            fontWeight: 500,
            borderRadius: 6,
            padding: '5px 10px',
            boxShadow: `0 4px 12px rgba(0,0,0,0.4)`,
            border: `1px solid ${alpha(amberMain, 0.15)}`,
          },
          arrow: {
            color: mode === 'light' ? uisceChocolate[700] : surfaceBg,
          },
        },
      },

      // ── Dialog ───────────────────────────────────────────────────────────
      MuiDialog: {
        styleOverrides: {
          paper: {
            background: mode === 'light' ? '#ffffff' : surfaceBg,
            border: `1px solid ${surfaceBorder}`,
            borderRadius: 14,
            boxShadow: mode === 'light'
              ? '0 20px 60px rgba(13,27,110,0.15), 0 8px 24px rgba(13,27,110,0.08)'
              : `0 24px 80px rgba(0,0,0,0.7), 0 8px 32px ${alpha(amberMain, 0.10)}`,
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
            color: surfaceBorder,
            '&.Mui-completed': { color: amberMain },
            '&.Mui-active': { color: amberMain },
          },
        },
      },
      MuiStepLabel: {
        styleOverrides: {
          label: {
            fontWeight: 500,
            fontSize: '0.875rem',
            '&.Mui-completed': { fontWeight: 600, color: amberMain },
            '&.Mui-active': { fontWeight: 700 },
          },
        },
      },

      // ── Table ───────────────────────────────────────────────────────────
      MuiTableHead: {
        styleOverrides: {
          root: {
            background: mode === 'light' ? uisceChocolate[900] : surfaceBg,
            '& .MuiTableCell-root': {
              color: '#ffffff',
              fontWeight: 700,
              fontSize: '0.8125rem',
              letterSpacing: '0.04em',
              textTransform: 'uppercase',
              borderBottom: `1px solid ${alpha(amberMain, 0.20)}`,
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
            '&:hover': { background: alpha(amberMain, 0.04) },
            '&.Mui-selected': {
              background: alpha(amberMain, 0.08),
              '&:hover': { background: alpha(amberMain, 0.12) },
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
              ? alpha(uisceChocolate[700], 0.06)
              : alpha('#ffffff', 0.06),
          },
        },
      },

      // ── Backdrop ────────────────────────────────────────────────────────
      MuiBackdrop: {
        styleOverrides: {
          root: {
            background: mode === 'light'
              ? alpha(uisceChocolate[900], 0.40)
              : alpha(charcoalBg, 0.80),
            backdropFilter: 'blur(4px)',
          },
        },
      },

    },
  });
}
