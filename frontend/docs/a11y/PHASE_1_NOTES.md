# Internationalization & Accessibility — Phase 1

This file describes the i18n + locale-prefix routing layer added in the
WCAG 2.1 AA initiative. It is the foundation every later phase compiles
against.

## What's wired

### Single source of truth for locales

`src/i18n/locales.ts` exports:

- `LOCALES` — the 8 locked locales: `en, es, fr, de, pt-BR, ja, zh-CN, ar`
- `Locale` — the literal-union type
- `normalizeLocale(raw)` — best-effort matcher; tolerates `es-MX` → `es`
- `matchLocale(candidates)` — first hit wins over a `navigator.languages`-style array
- `localePath(locale, rest)` — `localePath('es', '/dashboard')` → `/es/dashboard`
- `stripLocale(pathname)` — `/es/dashboard` → `/dashboard`, used by all path comparisons
- `isRtl(locale)` — `ar` (and `he`, `fa`) → `true`

The canonical list is what `i18n.ts` registers as `supportedLngs`, so unknown
locales fall back to `DEFAULT_LOCALE` (`en`).

### Routing shell

`src/routes/localeShell.tsx` is mounted in `App.tsx`. Two route tiers:

1. **Un-prefixed, top-level** — auth, deep links, embed surfaces. v6 ranks
   static routes above `/:locale/*` regardless of declaration order:
   - `/login`, `/auth/callback`, `/api-studio`, `/page-studio`, `/app/:slug`, `/change-review`
2. **Locale-prefixed** — `:locale/*` matches everything else. `LocaleLayout`
   heals un-prefixed requests to the user's preferred locale (preserving
   `search` and `hash`), and `LocaleSync` keeps `i18n.language` in lockstep
   with the URL.

`<DirectionProvider>` is hoisted above `LocaleShell` in `App.tsx` so RTL
applies to un-prefixed routes too — no shim components.

### `Link` and `NavLink`

`src/routes/Link.tsx` exports wrappers around `react-router-dom`'s `Link`
and `NavLink` that prepend the current locale prefix. ESLint
(`eslint.config.cjs`) forbids direct imports of `Link`/`NavLink` from
`react-router-dom`, so future internal navigations are forced through the
wrappers.

### Theme direction

`createUisceTheme(mode, direction)` accepts an optional `direction` arg
(defaults to `'ltr'`). `main.tsx` reads `i18n.language` and computes
`direction` via `isRtl()`.

> **Stubbed**: `DirectionProvider` is currently a pass-through. After the
> `ar` smoke test, it will install an Emotion cache with
> `stylis-plugin-rtl` for sx/emotion custom styles, and plain `.css`
> files will go through `postcss-rtlcss` (installed; pipeline config
> pending). MUI v7 handles MUI-generated styles via `theme.direction`
> alone.

### ICU + Language detection

`src/i18n.ts` registers i18next with:
- `i18next-icu` for ICU message format (handles Arabic plurals etc.)
- `i18next-browser-languagedetector` reading `appLocale` from localStorage
- Custom dynamic-import backend: `en` is bundled (typed keys, instant),
  other locales load as separate chunks on demand (visible in build output
  as `es-*.js`, `fr-*.js`).
- A `languageChanged` handler that updates `document.documentElement.lang`
  and `dir` so screen readers announce the right language.

The `selected_language` → `appLocale` migration runs **before**
LanguageDetector's first read (it reads during init).

## ICU migration

Existing `{{count}}` in `search.results_count` / `search.results_count_plural`
was rewritten as a single ICU key:

```jsonc
"search": {
  "results_count": "{count, plural, =0 {# results} one {# result} other {# results}}"
}
```

The `_plural` sibling was deleted; the only caller
(`ProfessionalSearchInput.tsx:206`) already used `results_count`.

## Triage of es/fr drift

The 250 keys present in `es.json` / `fr.json` but missing in `en.json` are
process drift from different tooling at different times:

- **9 keys (a)** — referenced in source (`scheduler.jobs.*`,
  `scheduler.title`); promote to en with real translations in Phase 2.
- **241 keys (c)** — dead; safe to delete from es/fr.

See `frontend/scripts/out/locale-triage.json` for the full classification.

## Commands

```bash
# Extract i18n keys from source into locale JSONs
pnpm i18n:extract

# Diff locales vs en; exits non-zero on missing keys (CI gate)
pnpm i18n:diff

# Aggregate per-route axe results into a baseline
node scripts/aggregate-a11y-baseline.mjs
```

## Known follow-ups (Phase 2+)

- [ ] **Plain CSS RTL** — activate postcss-rtlcss pipeline; only MUI v7
      handles its own RTL via `theme.direction`.
- [ ] **Pseudolocale** — add `en-XA` (accented) for QA once the key set
      stabilizes post-Phase-2 freeze.
- [ ] **Dynamic-key sweep** — `i18next-parser` will surface every `t(...)`
      call. The `typeof en` augmentation now enforces key shape; current
      `tsc` reports 1 dynamic-key site (`src/utils/i18n.ts:32`, fixed).
- [ ] **5 new locales** — de, pt-BR, ja, zh-CN, ar: ordered once the en key
      set is frozen (after Phase 2 sweep).
- [ ] **Theme stub** — `<DirectionProvider>` needs the Emotion cache once we
      know exactly what `ar` flips incorrectly without it.

## Files

**Modified**

- `src/i18n.ts` — rewritten
- `src/App.tsx` — DirectionProvider hoist + LocaleShell mount + stripLocale
- `src/main.tsx` — theme.direction threaded from i18n.language
- `src/AppRoutes.tsx` — codemod: 158 paths stripped, 11 `<Navigate>` localized,
  4 `useBlockableNavigate` localized, 6 un-prefixed routes moved to LocaleShell
- `src/components/LanguageSelector.tsx` — navigates with search/hash preservation,
  persists to user profile, in-place when on un-prefixed routes
- `src/components/MainNavigation.tsx` — stripLocale in active-tab highlight
- `src/components/AccessAwareNavigation.tsx` — stripLocale in active-tab
- `src/components/MobileResponsiveNavigation.tsx` — stripLocale
- `src/components/ConfigurableNavigationSidebar.tsx` — stripLocale
- `src/admin/layout/AdminLayout.tsx` — stripLocale + locale-aware Link
- `src/admin-v2/layout/AdminLayout.tsx` — stripLocale + locale-aware Link
- `src/utils/i18n.ts` — `as never` cast for dynamic keys
- `src/theme/uisceTheme.ts` — direction param added
- `src/locales/{en,es,fr}.json` — ICU migration, 3 shell keys added at top
- `eslint.config.cjs` — restrict-imports for Link/NavLink
- `package.json` — 5 new deps, 2 new scripts

**Created**

- `src/i18n/locales.ts`
- `src/i18n/useLocale.ts`
- `src/i18n/DirectionProvider.tsx` (stubbed)
- `src/routes/localeShell.tsx`
- `src/routes/Link.tsx`
- `e2e/a11y/auth.setup.ts`
- `e2e/a11y/baseline.spec.ts`
- `scripts/aggregate-a11y-baseline.mjs`
- `scripts/i18n-diff.mjs`
- `i18next-parser.config.cjs`
- `scripts/out/locale-triage.json`

**Deleted** — none; existing dead code under `src/i18n/index.tsx` will be
removed in a follow-up PR alongside the 241 dead keys in es/fr.
