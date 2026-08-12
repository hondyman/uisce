// ── Uisce Brand Palette ──────────────────────────────────────────────────────────
// Theme: Water & Liquidity — the essence of Uisce (Gaelic for water).
// Financial markets are liquid; capital flows like water. This theme embodies
// that fluidity — deep ocean depth in dark mode, tidal light in light mode.
//
// DARK:  Abyssal navy + bioluminescent aqua + seafoam phosphorescence
// LIGHT: Pearl tide + morning ocean teal + sunlit amber warmth

// ── Core Brand Colors ────────────────────────────────────────────────────────────

// PRIMARY — Ocean Teal (the water itself)
export const uisceOcean = {
  50:   '#E6FAFA',
  100:  '#B3F0EF',
  200:  '#80E6E5',
  300:  '#4DD9D8',
  400:  '#26CFCE',
  main: '#00C9C8',   // bioluminescent teal — primary action
  600:  '#00B5B4',
  700:  '#009E9D',
  800:  '#008786',
  dark: '#00706F',   // deep teal — pressed/active
  light:'#4DD9D8',   // aqua highlight
  glow:  'rgba(0, 201, 200, 0.35)',
  glowLight: 'rgba(0, 201, 200, 0.15)',
  glowStrong: 'rgba(0, 201, 200, 0.55)',
} as const;

// SECONDARY — Seafoam (the edge where water meets shore)
export const uisceSeafoam = {
  main:  '#0AF5B0',  // phosphorescent seafoam green
  light: '#5DFED0',
  dark:  '#00C98A',
  glow:  'rgba(10, 245, 176, 0.30)',
} as const;

// ACCENT — Liquid Gold (sunlight through shallow water / Core indicator)
export const uisceGold = {
  main:  '#F5A623',  // warm amber — sunlight on water surface
  light: '#FFC94D',
  dark:  '#D4880A',
  glow:  'rgba(245, 166, 35, 0.35)',
  glowLight: 'rgba(245, 166, 35, 0.15)',
} as const;

// Kept for backward compat (mapped to gold)
export const uisceAmber = uisceGold;

// ABYSS — Dark mode depth layers (deep ocean)
export const uisceAbyss = {
  950:  '#020810',  // hadal zone — deepest bg
  900:  '#050D1A',  // abyssal — canvas
  800:  '#071526',  // midnight ocean — surface
  700:  '#0A1E35',  // deep sea — elevated
  600:  '#0D2847',  // ocean floor — overlay
  500:  '#153558',  // twilight zone
  border: 'rgba(0, 201, 200, 0.12)',  // aqua border in dark
  divider:'rgba(255, 255, 255, 0.06)',
} as const;

// TIDE — Light mode surface layers (tidal shore)
export const uisceTide = {
  50:  '#F0FAFC',   // seafoam white — default bg
  100: '#E4F6F8',   // light tide — paper
  200: '#C8EDF2',   // tidal pool — elevated
  300: '#A8E1E9',   // shallow water
  400: '#75CED9',
  border: 'rgba(0, 149, 155, 0.15)',  // ocean border in light
  divider:'rgba(0, 100, 110, 0.08)',
} as const;

// Backward compat
export const uisceChocolate = {
  900: uisceAbyss[950],
  800: uisceAbyss[900],
  700: uisceAbyss[800],
  600: uisceAbyss[700],
  text: '#E8F4FF',
} as const;

export const uisceCream = {
  DEFAULT: uisceTide[100],
  100:     uisceTide[50],
  200:     uisceTide[200],
  dark:    '#4A7A8A',
} as const;

// Custom brand (delta/computed indicator)
export const uisceDelta = {
  main:  '#8B5CF6',  // violet — computed/delta
  light: '#A78BFA',
  dark:  '#6D28D9',
  glow:  'rgba(139, 92, 246, 0.25)',
} as const;

export const uisceWhite = {
  DEFAULT: '#ffffff',
  12: 'rgba(255, 255, 255, 0.12)',
  20: 'rgba(255, 255, 255, 0.20)',
  60: 'rgba(255, 255, 255, 0.60)',
  80: 'rgba(255, 255, 255, 0.80)',
} as const;

// Kept for backward compat
export const uisceTeal = {
  main:      uisceOcean.main,
  light:     uisceOcean.light,
  dark:      uisceOcean.dark,
  glow:      uisceOcean.glow,
  glowLight: uisceOcean.glowLight,
} as const;

export const uisceInk = {
  DEFAULT: '#071526',
  soft:    'rgba(7, 21, 38, 0.65)',
  dim:     'rgba(7, 21, 38, 0.40)',
} as const;

// ── Semantic Status Colors ───────────────────────────────────────────────────────
export const semanticSuccess = {
  light:   '#CCFBF1',
  DEFAULT: '#10B981',
  dark:    '#34D399',
  dark2:   '#064E3B',
} as const;

export const semanticWarning = {
  light:   '#FEF3C7',
  DEFAULT: '#F59E0B',
  dark:    '#FBBF24',
  dark2:   '#78350F',
} as const;

export const semanticError = {
  light:   '#FEE2E2',
  DEFAULT: '#EF4444',
  dark:    '#F87171',
  dark2:   '#7F1D1D',
} as const;

// ── Category Accent Colors (water-tinted) ────────────────────────────────────────
export const categoryColors = {
  platform: {
    light:   '#E0F2F7',
    main:    '#0077A8',
    dark:    '#38BDF8',
    bg:      'rgba(0, 119, 168, 0.08)',
    bgLight: 'rgba(0, 119, 168, 0.12)',
  },
  catalog: {
    light:   '#CCFBF1',
    main:    '#0D9488',
    dark:    '#2DD4BF',
    bg:      'rgba(13, 148, 136, 0.08)',
    bgLight: 'rgba(13, 148, 136, 0.12)',
  },
  build: {
    light:   '#EDE9FE',
    main:    '#7C3AED',
    dark:    '#A78BFA',
    bg:      'rgba(124, 58, 237, 0.08)',
    bgLight: 'rgba(124, 58, 237, 0.12)',
  },
  studio: {
    light:   '#FCE7F3',
    main:    '#DB2777',
    dark:    '#F472B6',
    bg:      'rgba(219, 39, 119, 0.08)',
    bgLight: 'rgba(219, 39, 119, 0.12)',
  },
  operations: {
    light:   '#D1FAE5',
    main:    '#059669',
    dark:    '#34D399',
    bg:      'rgba(5, 150, 105, 0.08)',
    bgLight: 'rgba(5, 150, 105, 0.12)',
  },
  intelligence: {
    light:   '#E0E7FF',
    main:    '#4F46E5',
    dark:    '#818CF8',
    bg:      'rgba(79, 70, 229, 0.08)',
    bgLight: 'rgba(79, 70, 229, 0.12)',
  },
  consume: {
    light:   '#FEF9C3',
    main:    '#D97706',
    dark:    '#FBBF24',
    bg:      'rgba(217, 119, 6, 0.08)',
    bgLight: 'rgba(217, 119, 6, 0.12)',
  },
  calendar: {
    light:   '#DBEAFE',
    main:    '#2563EB',
    dark:    '#60A5FA',
    bg:      'rgba(37, 99, 235, 0.08)',
    bgLight: 'rgba(37, 99, 235, 0.12)',
  },
} as const;

// ── Light Mode Palette — Tidal Shore ─────────────────────────────────────────────
// Morning light on the water. Clean, airy, premium. Teal primary, gold warmth.
export const lightPalette = {
  primary: {
    main:         uisceOcean.main,
    light:        uisceOcean.light,
    dark:         uisceOcean.dark,
    contrastText: '#FFFFFF',
  },
  secondary: {
    main:         uisceGold.main,
    light:        uisceGold.light,
    dark:         uisceGold.dark,
    contrastText: '#1A0A00',
  },
  tertiary: {
    main:         uisceSeafoam.main,
    light:        uisceSeafoam.light,
    dark:         uisceSeafoam.dark,
    contrastText: uisceInk.DEFAULT,
  },
  success: {
    main:         semanticSuccess.DEFAULT,
    light:        semanticSuccess.light,
    dark:         semanticSuccess.dark,
    contrastText: '#FFFFFF',
  },
  warning: {
    main:         semanticWarning.DEFAULT,
    light:        semanticWarning.light,
    dark:         semanticWarning.dark,
    contrastText: '#FFFFFF',
  },
  error: {
    main:         semanticError.DEFAULT,
    light:        semanticError.light,
    dark:         semanticError.dark,
    contrastText: '#FFFFFF',
  },
  background: {
    default: uisceTide[50],   // #F0FAFC — seafoam white
    paper:   '#FFFFFF',
  },
  text: {
    primary:   '#071526',        // deep ocean ink
    secondary: 'rgba(7,21,38,0.60)',
    disabled:  'rgba(7,21,38,0.36)',
  },
  divider: uisceTide.divider,
  action: {
    hover:              'rgba(0, 201, 200, 0.06)',
    selected:           'rgba(0, 201, 200, 0.10)',
    hoverOpacity:        0.06,
    selectedOpacity:     0.10,
    focus:              'rgba(0, 201, 200, 0.12)',
    disabled:           'rgba(7, 21, 38, 0.26)',
    disabledBackground: 'rgba(7, 21, 38, 0.08)',
    active:             'rgba(0, 201, 200, 0.14)',
    activeOpacity:       0.14,
  },
} as const;

// ── Dark Mode Palette — Abyssal Ocean ─────────────────────────────────────────────
// Bioluminescent life in the deep. Obsidian canvas, aqua glow, seafoam pulse.
export const darkPalette = {
  primary: {
    main:         uisceOcean.main,   // #00C9C8 bioluminescent teal
    light:        uisceOcean.light,  // #4DD9D8
    dark:         uisceOcean[600],   // #00B5B4
    contrastText: '#050D1A',
  },
  secondary: {
    main:         uisceGold.main,    // #F5A623 amber — like the gold copy badge
    light:        uisceGold.light,
    dark:         uisceGold.dark,
    contrastText: '#050D1A',
  },
  tertiary: {
    main:         uisceSeafoam.main, // #0AF5B0 seafoam phosphorescence
    light:        uisceSeafoam.light,
    dark:         uisceSeafoam.dark,
    contrastText: '#050D1A',
  },
  success: {
    main:         semanticSuccess.dark,
    light:        semanticSuccess.DEFAULT,
    dark:         semanticSuccess.light,
    contrastText: '#050D1A',
  },
  warning: {
    main:         semanticWarning.dark,
    light:        semanticWarning.DEFAULT,
    dark:         semanticWarning.light,
    contrastText: '#050D1A',
  },
  error: {
    main:         semanticError.dark,
    light:        semanticError.DEFAULT,
    dark:         semanticError.light,
    contrastText: '#050D1A',
  },
  background: {
    default: uisceAbyss[900],   // #050D1A — abyssal canvas
    paper:   uisceAbyss[800],   // #071526 — midnight surface
  },
  text: {
    primary:   '#E8F4FF',        // moonlit blue-white
    secondary: '#7BA8C4',        // ocean mist
    disabled:  'rgba(232,244,255,0.36)',
  },
  divider: uisceAbyss.divider,
  action: {
    hover:              'rgba(0, 201, 200, 0.08)',
    selected:           'rgba(0, 201, 200, 0.15)',
    hoverOpacity:        0.08,
    selectedOpacity:     0.15,
    focus:              'rgba(0, 201, 200, 0.20)',
    disabled:           'rgba(232, 244, 255, 0.26)',
    disabledBackground: 'rgba(232, 244, 255, 0.08)',
    active:             'rgba(0, 201, 200, 0.22)',
    activeOpacity:       0.22,
  },
} as const;
