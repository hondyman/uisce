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
      handles its own RTL via `theme.direction`. ~172 hand-written CSS
      files ship LTR-in-Arabic until this lands. **`postcss-rtlcss` is
      declared in package.json (^4.0.9) — activation is a Phase 2
      precondition, not an assumption.**
- [ ] **`@mui/lab@5.0.0-alpha.177` version skew** — lab@5 is paired with
      `@mui/system@5.18.0` (nested under `node_modules/@mui/lab/`). When
      material's RtlProvider from `@mui/system@7.3.8` runs, lab-rendered
      components reading RtlProvider's React context read from the old
      5.18.0 context, which is a different object identity — lab components
      stay LTR. **Practical impact** is confined to `SimulationPanel.tsx`'s
      `TabPanel` (Timeline/TreeView aren't direction-sensitive). Bumping
      lab to v7+ would break Timeline (removed in lab@6) and require
      migration. Deferred to Phase 2; documented as known `ar` defect.
- [ ] **Pseudolocale** — add `en-XA` (accented) for QA once the key set
      stabilizes post-Phase-2 freeze.
- [ ] **Dynamic-key sweep** — `i18next-parser` will surface every `t(...)`
      call. The `typeof en` augmentation now enforces key shape; current
      `tsc` reports 0 dynamic-key sites.
- [ ] **5 new locales** — `de`, `pt-BR`, `ja`, `zh-CN`, `ar`: skeleton
      files with English defaults present, awaiting translation. Phase 3.
      **Coverage after Phase 1: 1–2.5% per skeleton (essentially "file
      present, no human copy").** See `scripts/out/i18n-coverage.json`.
- [ ] **Phase 2 key-set freeze** — `en.json` now has 512 keys (159 original
      + 353 harvested from `t('key', 'default')` patterns). After Phase 2
      review, freeze and order human translation for the 5 new locales.
- [ ] **Empty-value reconciliation** — 2 keys in en.json, 3 each in es/fr,
      are empty (parser couldn't resolve from a defaultValue conflict).
      Audit and fill in Phase 2.

## Coverage numbers (Phase 2 kickoff input)

Output of `node scripts/i18n-diff.mjs`:

| Locale | Present | Missing | Identity (untranslated) | Real translated | Translated % |
|--------|---------|---------|--------------------------|------------------|---------------|
| es     | 723     | 33      | 407                      | 316              | **61.7%**     |
| fr     | 723     | 33      | 413                      | 310              | **60.5%**     |
| ar     | 458     | 62      | 445                      | 13               | 2.5% (skeleton) |
| de     | 450     | 62      | 445                      | 5                | 1.0% (skeleton) |
| ja     | 448     | 64      | 443                      | 5                | 1.0% (skeleton) |
| pt-BR  | 452     | 62      | 445                      | 7                | 1.4% (skeleton) |
| zh-CN  | 448     | 64      | 443                      | 5                | 1.0% (skeleton) |

es/fr ~60% translated is the *current* state — same as pre-batch (no
translations were lost). The remaining ~40% of UI renders English for
Spanish/French users today; not a regression, but now *measured*.

Skeleton locales (5) have all-key identity with English (parser
harvested the inline English defaults). They appear 100% covered to a
naive coverage check; the `i18n-diff.mjs` script distinguishes identity
from real translation.

**Translation work to complete all locales**: ~3,480 strings across 7
locales. Of that, ~440 strings each for es/fr (filling in the 40%
gap), ~510 strings each for de/ar/pt-BR/zh-CN/ja (full skeleton). Vendor
order sizing input for Phase 2 kickoff.

### Vendor plural notes (per CLDR plural categories)

Native suffix plurals mean the number of suffix variants per key depends
on the target language's CLDR categories. Vendor brief MUST specify
this so the order isn't six keys per string for languages that only
have one:

| Locale | Required suffixes | Notes |
|--------|-------------------|-------|
| en      | `_one`, `_other`    | CLDR: 2 categories |
| es      | `_one`, `_other`    | CLDR: 2 categories |
| fr      | `_one`, `_other`    | CLDR: 2 categories (≥2) |
| de      | `_one`, `_other`    | CLDR: 2 categories |
| ja      | `_other`            | CLDR: **1 category** — no plural distinction |
| zh-CN   | `_other`            | CLDR: 1 category |
| pt-BR   | `_one`, `_other`    | CLDR: 2 categories |
| **ar**  | **`_zero`, `_one`, `_two`, `_few`, `_many`, `_other`** | CLDR: **6 categories** — must ship all six |

This prevents paying for six keys per Japanese/Chinese string (which only
need one) while ensuring Arabic gets all six variants it needs for correct
plural rendering.

## Files

**Modified**

- `src/i18n.ts` — rewritten (i18next-icu plugin, ICU parse patched via direct
  `IntlMessageFormat` constructor resolution, `selected_language` migration,
  URL-driven initial language, deterministic `<html lang/dir>` block,
  typed `resources` augmentation)
- `src/i18n/DirectionProvider.tsx` — now imports the `RtlProvider` default
  export from `@mui/system/RtlProvider` (was a pass-through stub)
- `src/App.tsx` — DirectionProvider hoisted above LocaleShell, `stripLocale`
  in `hideNav`
- `src/main.tsx` — `theme.direction` threaded from `i18n.language`
- `src/theme/uisceTheme.ts` — `direction` parameter, `prefers-reduced-motion`
  media block (placed AFTER `html scrollBehavior` for equal-specificity
  precedence), Noto Sans font stack for script-aware fallback
- `src/locales/en.json`, `src/locales/es.json`, `src/locales/fr.json` —
  ICU migration of `search.results_count`, 3 chrome keys (`app`, `a11y`,
  `nav`) added to top, parser-harvested defaultValues, `nav.languageComingSoon`
- `src/locales/{de,pt-BR,ja,zh-CN,ar}.json` — new files with English
  defaults, awaiting translation
- `src/i18n/icu-plural.test.ts` — new test file (4 tests)
- `src/i18n/locales.ts` — `LOCALES`, `Locale`, `normalizeLocale`,
  `matchLocale`, `localePath`, `stripLocale`, `isRtl`, `NATIVE_NAMES`
- `src/i18n/useLocale.ts` — location-derived locale
- `src/routes/localeShell.tsx` — `<Routes>` wrapper, un-prefixed routes
  static, `:locale/*` splat, search/hash preservation
- `src/routes/Link.tsx` — locale-aware Link / NavLink
- `src/components/LanguageSelector.tsx` — gated to 3 active locales + 5
  disabled "Coming soon" entries
- `src/index.css` — stripped 3 inert `@tailwind` directives
- `src/AppRoutes.tsx` — codemod: 158 paths stripped, 11 `<Navigate>`
  localized, 4 `useBlockableNavigate` calls localized, 6 un-prefixed
  routes relocated to LocaleShell
- `src/components/MainNavigation.tsx`, `src/components/AccessAwareNavigation.tsx`,
  `src/components/MobileResponsiveNavigation.tsx`,
  `src/components/ConfigurableNavigationSidebar.tsx`,
  `src/admin/layout/AdminLayout.tsx`,
  `src/admin-v2/layout/AdminLayout.tsx` — `stripLocale()` in path
  comparisons; admin layouts use locale-aware `<Link>`
- `eslint.config.cjs` — `no-restricted-imports` for `Link`/`NavLink`
- `package.json` — 5 new deps (i18next-icu, i18next-browser-languagedetector,
  i18next-parser, postcss-rtlcss, @axe-core/playwright), 2 new scripts
  (`i18n:extract`, `i18n:diff`), `build` wired through ts-ratchet.mjs
- `vitest.config.ts` — include `src/i18n/**/*.test.{ts,tsx}`

**Created**

- `src/i18n/locales.ts`
- `src/i18n/useLocale.ts`
- `src/i18n/DirectionProvider.tsx`
- `src/i18n/icu-plural.test.ts`
- `src/routes/localeShell.tsx`
- `src/routes/Link.tsx`
- `e2e/a11y/auth.setup.ts`
- `e2e/a11y/baseline.spec.ts`
- `scripts/aggregate-a11y-baseline.mjs`
- `scripts/i18n-diff.mjs`
- `scripts/ts-ratchet.mjs` — typecheck ratchet gate (exit non-zero on new tsc errors)
- `scripts/i18n-merge-harvest.mjs` — merge parser harvest into existing locale files
- `i18next-parser.config.js` — ESM (was `.cjs`, lilconfig can't load `.cjs`)
- `scripts/out/locale-triage.json` — Step 0.5a classification
- `scripts/out/i18n-merge-summary.json` — Step 1c reconciliation summary
- `docs/ts-baseline.txt` — 957 pre-existing tsc error blocks (anchor for ratchet)
- `docs/a11y/PHASE_1_NOTES.md` — this file

**Deleted**

- `src/utils/i18n.ts` — dead wrapper, zero consumers
- `src/contexts/I18nContext.tsx` — dead duplicate i18n context, zero consumers
- `src/i18n/index.tsx` — dead duplicate i18n module, zero consumers
- 41 orphan CSS files (no live imports anywhere in src/)

## i18next-parser trap (avoided)

i18next-parser@9 takes its config from `i18next-parser.config.{js,mjs,json,ts,...}`
(via lilconfig). **`.cjs` is not in lilconfig's loader list** — using `.cjs`
silently produces an empty config and writes to the package's default
`locales/$LOCALE/$NAMESPACE.json` instead of your custom output path. The
correct flag for "preserve unused keys" is `keepRemoved: true`, not
`removeUnusedKeys: false` (latter is silently ignored). Without these,
the parser will:

1. Write to wrong path (overwriting JSON files you didn't intend)
2. Wipe keys that look "absent" (e.g. template-literal keys like
   `scheduler.days.${key}` and the I18nContext namespacing route)

## ICU plugin: dropped in favor of native i18next suffix plurals

`i18next-icu@2.4.4` (latest published — `^3.0.0` does NOT exist on npm; the
project appears abandoned after 2.4.4) has a broken ESM
`import IntlMessageFormat from 'intl-messageformat'` that resolves to the
namespace object rather than the constructor in our Vite/vitest
environment. The plugin's `parse()` then throws "IntlMessageFormat is
not a constructor" and silently returns the raw ICU string via its
`parseErrorHandler`.

### Decision: drop ICU plugin entirely, use native i18next suffix plurals

**Usage audit (grep across all locales):**

```
ICU plural messages: 3   (search.results_count × en/es/fr)
ICU select messages: 0
```

Three plural messages. Zero select. i18next's native suffix plurals
(`_one` / `_other` driven by `Intl.PluralRules`) cover 100% of this
usage with zero dependencies. Migration cost: 1 key with ICU syntax
across 8 locales → 2 native suffix keys per locale.

**Migration applied**:
- `search.results_count` (ICU syntax) → `search.results_count_one` /
  `search.results_count_other` (native).
- Placeholders use i18next default `{{count}}` syntax (single-brace
  `{count}` doesn't interpolate under i18next's default settings).
- `i18next-icu` and `intl-messageformat` uninstalled (no longer in
  `package.json`).
- `src/i18n.ts` no longer imports `i18next-icu` or patches the plugin.

**Why this beats the patch-from-the-previous-batch**:

1. **Zero interop fragility** — native suffix plurals work through i18next's
   built-in `Intl.PluralRules` integration. No constructor resolution,
   no plugin post-init monkey-patching, no silent error swallow.
2. **Vendor-tool-friendly** — both Crowdin and Lokalise (the two TMS
   options flagged for Phase 2) treat i18next-JSON as a first-class
   format. ICU syntax would require a vendor-specific TMS configuration.
3. **6-category Arabic support** — `Intl.PluralRules('ar').select(0)`
   returns `'zero'`, `.select(1)` returns `'one'`, etc. Native suffix
   plurals map each category to its own key (`_zero`, `_one`, `_two`,
   `_few`, `_many`, `_other`). Same coverage as ICU, less machinery.
4. **CLDR-aligned** — i18next uses `Intl.PluralRules` which is the
   same source ICU uses. Zero is zero in both.

**Smoke test updated** (`src/i18n/plural.test.ts`): 4 tests, all
passing. Same tripwire shape: if suffix plurals break, the test fails
loudly with the raw key vs. expected `'1 result'`.

**Counterfactual had we shipped the previous batch's patch**:
`search.results_count` would render **literally** as
`{count, plural, =0 {# results} one {# result} other {# results}}` in
the search UI. Build green, ratchet green, no test caught it. The smoke
test caught it — that's the static-orchestra ceiling and why runtime
tests must stay paired.

## Test results

- `vitest`: 8 files failed / 41 passed; 26 tests failed / 204 passed.
  Same 26 failures pre/post. **Zero new test failures from this batch.**
- `ts-ratchet`: 957 baseline error blocks. Ratchet OK, 0 new.
- Plural smoke test (4 tests, all passing): `search.results_count`
  renders `0 results`, `1 result`, `2 results`, `5 results` via native
  i18next suffix plurals.
- Locale-shell routing matrix (10 tests, all passing): exercises
  `resolveLocaleRoute` — the single pure function the shell calls.
