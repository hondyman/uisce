// i18next-parser reads `i18next-parser.config.{js,mjs,json,ts,yaml,yml}`.
// lilconfig supports `.js` and `.mjs`; `.cjs` is not in its loader list.
// This file is ESM (project root package.json sets "type": "module").

export default {
  locales: ['en', 'es', 'fr', 'de', 'pt-BR', 'ja', 'zh-CN', 'ar'],
  defaultLocale: 'en',

  // CRITICAL: do NOT remove keys absent from source. Template-literal keys
  // (e.g. `scheduler.days.${key}`) and the I18nContext namespacing route
  // look absent to the parser. Removing them would erase the Step-0.5 triage
  // harvest. Reconciliation handles them by hand.
  // NOTE: the actual flag name in i18next-parser@9 is `keepRemoved` —
  // `removeUnusedKeys` is a different package's option name and is silently
  // ignored here.
  keepRemoved: true,

  // Existing values must be preserved — the parser only ADDS keys, never
  // overwrites or removes existing entries.
  updateMissing: false,

  // Don't fail when new keys are missing from non-default locales — that's
  // expected until Phase 2 freeze.
  failOnMissing: false,
  failOnWarnings: false,
  failOnUpdate: false,
  verbose: false,

  input: [
    'src/**/*.{ts,tsx}',
    '!**/__tests__/**',
    '!**/*.test.{ts,tsx}',
    '!**/*.spec.{ts,tsx}',
  ],

  // Single-file locale layout — matches the project's existing structure
  // (one JSON file per locale, no namespace subdirectories).
  output: 'src/locales/$LOCALE.json',

  defaultNamespace: 'translation',
  namespaceSeparator: false,
  keySeparator: '.',

  // Default values from the second argument of t(key, 'default') are
  // harvested into en.json automatically — that's how Step-0.5 reconciliation
  // closes. Literal strings and template literals both feed the harvest.
  // Empty string in source = empty string in output = human copy pending.
  func: {
    list: ['t', 'i18n.t'],
    extensions: ['.ts', '.tsx'],
  },

  trans: {
    component: 'Trans',
    i18nKey: 'i18nKey',
  },

  lexers: {
    js: ['JsxLexer'],
    ts: ['JsxLexer'],
    tsx: ['JsxLexer'],
    default: ['JsxLexer'],
  },
};
