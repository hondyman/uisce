module.exports = {
  locales: ['en', 'es', 'fr', 'de', 'pt-BR', 'ja', 'zh-CN', 'ar'],
  defaultLocale: 'en',
  updateMissing: true,
  src: ['./src/**/*.{ts,tsx}'],
  output: './src/locales/$LOCALE.json',
  defaultNamespace: 'translation',
  namespaceSeparator: false,
  keySeparator: '.',
  // Look at: react-i18next `t()`, `i18n.t()`, our local `t()` wrapper, and
  // `Trans` children props (i18nKey).
  func: {
    list: ['t', 'i18n.t'],
    extensions: ['.ts', '.tsx'],
  },
  trans: {
    component: 'Trans',
    i18nKey: 'i18nKey',
  },
  // Don't replace values for `ar` / `ja` / `zh-CN` — they're frozen until
  // after Phase 2. Re-enable when the human-translation pass is ordered.
  failOnWarnings: false,
  failOnUpdate: false,
  verbose: false,
};
