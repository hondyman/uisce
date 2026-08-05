// ── Uisce Brand Palette ──────────────────────────────────────────────────────────
// Theme: Rich chocolate + warm amber. Teal is ONLY in the logo.
// No blue anywhere in the UI.

// ── Brand Colors ────────────────────────────────────────────────────────────────
export const uisceAmber = {
  main:   '#D4A017',  // primary UI accent — melted gold
  light:  '#F5C518',  // bright gold — hover/text highlights
  dark:   '#B8860B',  // dark gold — pressed states
  glow:   'rgba(212, 160, 23, 0.35)',
  glowLight: 'rgba(212, 160, 23, 0.15)',
} as const;

export const uisceChocolate = {
  900: '#1A0F08',  // espresso — darkest bg
  800: '#2D1A0E',  // deep chocolate
  700: '#3D2314',  // rich brown — surfaces
  600: '#5C3D2E',  // warm brown
  text:  '#F5F0E8',  // cream — text on dark
} as const;

export const uisceCream = {
  DEFAULT: '#F5F0E8',
  100: '#FAF7F2',
  200: '#EDE6DB',
  dark:  '#8B7D6B',  // warm grey — secondary text
} as const;

export const uisceTeal = {
  // Tertiary — ONLY in the logo. Not used in UI.
  main:   '#00c9c8',
  light:  '#4dd9d8',
  dark:   '#00bcd4',
  glow:   'rgba(0, 201, 200, 0.35)',
  glowLight: 'rgba(0, 201, 200, 0.15)',
} as const;

export const uisceWhite = {
  DEFAULT: '#ffffff',
  12: 'rgba(255, 255, 255, 0.12)',
  20: 'rgba(255, 255, 255, 0.20)',
  60: 'rgba(255, 255, 255, 0.60)',
  80: 'rgba(255, 255, 255, 0.80)',
} as const;

export const uisceInk = {
  DEFAULT: '#1A0F08',
  soft:   'rgba(26, 15, 8, 0.65)',
  dim:    'rgba(26, 15, 8, 0.45)',
} as const;

// ── Semantic Colors ─────────────────────────────────────────────────────────────
export const semanticSuccess = {
  light:   '#d1fae5',
  DEFAULT: '#10b981',
  dark:    '#34d399',
  dark2:   '#065f46',
} as const;

export const semanticWarning = {
  light:   '#fef3c7',
  DEFAULT: '#ea580c',   // orange — warm complement
  dark:    '#fb923c',
  dark2:   '#7c2d12',
} as const;

export const semanticError = {
  light:   '#fce7f3',
  DEFAULT: '#e11d48',
  dark:    '#fb7185',
  dark2:   '#9f1239',
} as const;

// ── Category Accent Colors ─────────────────────────────────────────────────────
export const categoryColors = {
  platform: {
    light: '#f1f5f9',
    main:  '#64748b',
    dark:  '#94a3b8',
    bg:    'rgba(100, 116, 139, 0.08)',
    bgLight: 'rgba(100, 116, 139, 0.12)',
  },
  catalog: {
    light: '#e0f2fe',
    main:  '#0891b2',
    dark:  '#22d3ee',
    bg:    'rgba(8, 145, 178, 0.08)',
    bgLight: 'rgba(8, 145, 178, 0.12)',
  },
  build: {
    light: '#ede9fe',
    main:  '#7c3aed',
    dark:  '#a78bfa',
    bg:    'rgba(124, 58, 237, 0.08)',
    bgLight: 'rgba(124, 58, 237, 0.12)',
  },
  studio: {
    light: '#fce7f3',
    main:  '#db2777',
    dark:  '#f472b6',
    bg:    'rgba(219, 39, 119, 0.08)',
    bgLight: 'rgba(219, 39, 119, 0.12)',
  },
  operations: {
    light: '#ccfbf1',
    main:  '#0d9488',
    dark:  '#2dd4bf',
    bg:    'rgba(13, 148, 136, 0.08)',
    bgLight: 'rgba(13, 148, 136, 0.12)',
  },
  intelligence: {
    light: '#e0e7ff',
    main:  '#4f46e5',
    dark:  '#818cf8',
    bg:    'rgba(79, 70, 229, 0.08)',
    bgLight: 'rgba(79, 70, 229, 0.12)',
  },
  consume: {
    light: '#fef3c7',
    main:  '#d97706',
    dark:  '#fbbf24',
    bg:    'rgba(217, 119, 6, 0.08)',
    bgLight: 'rgba(217, 119, 6, 0.12)',
  },
  calendar: {
    light: '#e0f2fe',
    main:  '#0284c7',
    dark:  '#38bdf8',
    bg:    'rgba(2, 132, 199, 0.08)',
    bgLight: 'rgba(2, 132, 199, 0.12)',
  },
} as const;

// ── Complete light-mode palette ────────────────────────────────────────────────
export const lightPalette = {
  primary: {
    main:         uisceAmber.main,
    light:       uisceAmber.light,
    dark:        uisceAmber.dark,
    contrastText: uisceInk.DEFAULT,
  },
  secondary: {
    main:         uisceChocolate[700],
    light:       uisceChocolate[600],
    dark:        uisceChocolate[800],
    contrastText: uisceCream.DEFAULT,
  },
  tertiary: {
    main:         uisceTeal.main,
    light:       uisceTeal.light,
    dark:        uisceTeal.dark,
    contrastText: uisceInk.DEFAULT,
  },
  success: {
    main:         semanticSuccess.DEFAULT,
    light:       semanticSuccess.light,
    dark:        semanticSuccess.dark,
    contrastText: semanticSuccess.dark2,
  },
  warning: {
    main:         semanticWarning.DEFAULT,
    light:       semanticWarning.light,
    dark:        semanticWarning.dark,
    contrastText: semanticWarning.dark2,
  },
  error: {
    main:         semanticError.DEFAULT,
    light:       semanticError.light,
    dark:        semanticError.dark,
    contrastText: semanticError.dark2,
  },
  background: {
    default: uisceCream[100],
    paper:   uisceCream.DEFAULT,
  },
  text: {
    primary:   uisceInk.DEFAULT,
    secondary: uisceInk.soft,
    disabled:  uisceInk.dim,
  },
  divider: 'rgba(26, 15, 8, 0.08)',
  action: {
    hover:         'rgba(212, 160, 23, 0.06)',
    selected:      'rgba(212, 160, 23, 0.10)',
    hoverOpacity:   0.06,
    selectedOpacity: 0.10,
    focus:         'rgba(212, 160, 23, 0.12)',
    disabled:      'rgba(26, 15, 8, 0.26)',
    disabledBackground: 'rgba(26, 15, 8, 0.12)',
    active:        'rgba(212, 160, 23, 0.14)',
    activeOpacity:  0.14,
  },
} as const;

// ── Charcoal dark-mode palette ────────────────────────────────────────────────
export const darkPalette = {
  primary: {
    main:         uisceAmber.light,
    light:       uisceAmber.light,
    dark:        uisceAmber.main,
    contrastText: '#1A0F08',
  },
  secondary: {
    main:         uisceChocolate[700],
    light:       uisceChocolate[600],
    dark:        uisceChocolate[900],
    contrastText: uisceCream.DEFAULT,
  },
  tertiary: {
    main:         uisceTeal.main,
    light:       uisceTeal.light,
    dark:        uisceTeal.dark,
    contrastText: uisceCream.DEFAULT,
  },
  success: {
    main:         semanticSuccess.dark,
    light:       semanticSuccess.DEFAULT,
    dark:        semanticSuccess.light,
    contrastText: semanticSuccess.dark2,
  },
  warning: {
    main:         semanticWarning.dark,
    light:       semanticWarning.DEFAULT,
    dark:        semanticWarning.light,
    contrastText: semanticWarning.dark2,
  },
  error: {
    main:         semanticError.dark,
    light:       semanticError.DEFAULT,
    dark:        semanticError.light,
    contrastText: semanticError.dark2,
  },
  background: {
    default: '#0A0C12',
    paper:   '#13161E',
  },
  text: {
    primary:   '#E2E8F0',
    secondary: '#8892A4',
    disabled:  'rgba(226, 232, 240, 0.40)',
  },
  divider: 'rgba(255, 255, 255, 0.07)',
  action: {
    hover:         'rgba(212, 160, 23, 0.08)',
    selected:      'rgba(212, 160, 23, 0.15)',
    hoverOpacity:   0.08,
    selectedOpacity: 0.15,
    focus:         'rgba(212, 160, 23, 0.20)',
    disabled:      'rgba(226, 232, 240, 0.26)',
    disabledBackground: 'rgba(226, 232, 240, 0.12)',
    active:        'rgba(212, 160, 23, 0.22)',
    activeOpacity:  0.22,
  },
} as const;
