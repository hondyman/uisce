const pluginImport = require('eslint-plugin-import');
const pluginJsxA11y = require('eslint-plugin-jsx-a11y');

module.exports = [
  {
    ignores: ["dist/**", "src/backups/**", "**/backups/**"],
  },
  {
    files: ["**/*.{js,jsx,mjs,cjs,ts,tsx}"],
    plugins: {
      "@typescript-eslint": require("@typescript-eslint/eslint-plugin"),
      "react": require("eslint-plugin-react"),
      "react-hooks": require("eslint-plugin-react-hooks"),
      "import": pluginImport,
      "jsx-a11y": pluginJsxA11y,
    },
    languageOptions: {
      parser: require("@typescript-eslint/parser"),
      parserOptions: {
        ecmaFeatures: {
          jsx: true,
        },
      },
      globals: {
        browser: true,
        es2017: true,
        node: true,
      },
    },
    rules: {
      // Use the TypeScript-aware rule and ignore variables/args that start with an underscore
      "no-unused-vars": "off",
      "@typescript-eslint/no-unused-vars": [
        "warn",
        {
          "vars": "all",
          "args": "after-used",
          "ignoreRestSiblings": true,
          "varsIgnorePattern": "^_",
          "argsIgnorePattern": "^_"
        }
      ],
      "import/no-named-as-default": "off",
      // Disallow console usage except console.error — tracked separately from a11y
      // errors. The a11y cap is a11y-only (59); this rule contributes only to
      // the warning floor and its own cleanliness counter, not the a11y gate.
      "no-restricted-syntax": [
        "warn",
        {
          selector: "CallExpression[callee.object.name='console'][callee.property.name!= 'error']",
          message: "Use the devLogger (devLog/devDebug/devWarn) for non-error console output or remove console statements. Only console.error is allowed."
        }
      ],
      // jsx-a11y — static a11y lint gate. Maps to the axe baseline:
      //   button-name (21)              ← click-events-have-key-events +
      //                                   interactive-supports-focus (partial);
      //                                   IconButton wrappers are the real fix,
      //                                   this catches the easy misses.
      //   aria-input-field-name (10),
      //     label (7),
      //     select-name (5)             ← label-has-associated-control +
      //                                   label-has-for
      //   aria-progressbar-name (26)    ← aria-role (catches unlabelled
      //                                   progressbar role misuse; tracks the
      //                                   A11yCircularProgress sweep)
      //   scrollable-region-focusable   ← (no static equivalent; axe-runtime)
      //   list (3), listitem (1)        ← no-redundant-roles
      //   aria-command-name (2)         ← (no static equivalent; axe-runtime)
      //   nested-interactive (2)        ← no-static-element-interactions +
      //                                   no-noninteractive-element-interactions
      //   aria-prohibited-attr (2)      ← aria-proptypes
      //
      // Phase-2 gate: WARN on the rules that map to baseline violations
      // (so existing code doesn't fail CI before the sweep drives them down),
      // ERROR on the cheap structural rules where axe-runtime is the only
      // other signal (so new code can't add them undetected). The sweep
      // flips WARN→ERROR per rule as it drives counts to zero — each PR
      // turns one rule from WARN to ERROR with a corresponding commit
      // message naming the rule.
      "jsx-a11y/alt-text": "error",
      "jsx-a11y/anchor-has-content": "error",
      "jsx-a11y/anchor-is-valid": "warn",
      "jsx-a11y/aria-activedescendant-has-tabindex": "warn",
      "jsx-a11y/aria-props": "warn",
      "jsx-a11y/aria-proptypes": "warn",
      "jsx-a11y/aria-role": "warn",
      "jsx-a11y/aria-unsupported-elements": "warn",
      "jsx-a11y/click-events-have-key-events": "warn",
      "jsx-a11y/heading-has-content": "error",
      "jsx-a11y/html-has-lang": "error",
      "jsx-a11y/iframe-has-title": "error",
      "jsx-a11y/img-redundant-alt": "error",
      "jsx-a11y/interactive-supports-focus": "warn",
      "jsx-a11y/label-has-associated-control": "warn",
      "jsx-a11y/label-has-for": "warn",
      "jsx-a11y/media-has-caption": "warn",
      "jsx-a11y/mouse-events-have-key-events": "warn",
      "jsx-a11y/no-access-key": "error",
      "jsx-a11y/no-autofocus": "error",
      "jsx-a11y/no-distracting-elements": "error",
      "jsx-a11y/no-interactive-element-to-noninteractive-role": "warn",
      "jsx-a11y/no-noninteractive-element-interactions": "warn",
      "jsx-a11y/no-noninteractive-element-to-interactive-role": "warn",
      "jsx-a11y/no-onchange": "warn",
      "jsx-a11y/no-redundant-roles": "warn",
      "jsx-a11y/no-static-element-interactions": "warn",
      "jsx-a11y/role-has-required-aria-props": "warn",
      "jsx-a11y/role-supports-aria-props": "warn",
      "jsx-a11y/scope": "error",
      "jsx-a11y/tabindex-no-positive": "error",
    },
    settings: {
      react: {
        version: "detect",
      },
    },
  },
  // Allow console usage in development tooling and tests. The global rule above
  // enforces using `devLogger` in application code, but dev-tools and test
  // harnesses intentionally use console for simple diagnostics.
  {
    files: [
      "dev-tools/**",
      "scripts/**",
      "tests/**",
      "**/__tests__/**",
      "**/*.test.{js,ts,tsx,jsx}",
      "**/*.mjs"
    ],
    rules: {
      // Turn off the console restriction for these files so dev scripts can use
      // console.log/debug/warn freely without violating the app-wide rule.
      "no-restricted-syntax": "off",
      // Test files don't need a11y lint coverage — they're testing components,
      // not shipping UI. Production lint catches real regressions.
      "jsx-a11y/alt-text": "off",
      "jsx-a11y/anchor-has-content": "off",
      "jsx-a11y/anchor-is-valid": "off",
      "jsx-a11y/click-events-have-key-events": "off",
      "jsx-a11y/heading-has-content": "off",
      "jsx-a11y/html-has-lang": "off",
      "jsx-a11y/iframe-has-title": "off",
      "jsx-a11y/img-redundant-alt": "off",
      "jsx-a11y/interactive-supports-focus": "off",
      "jsx-a11y/label-has-associated-control": "off",
      "jsx-a11y/label-has-for": "off",
      "jsx-a11y/media-has-caption": "off",
      "jsx-a11y/mouse-events-have-key-events": "off",
      "jsx-a11y/no-access-key": "off",
      "jsx-a11y/no-autofocus": "off",
      "jsx-a11y/no-distracting-elements": "off",
      "jsx-a11y/no-interactive-element-to-noninteractive-role": "off",
      "jsx-a11y/no-noninteractive-element-interactions": "off",
      "jsx-a11y/no-noninteractive-element-to-interactive-role": "off",
      "jsx-a11y/no-onchange": "off",
      "jsx-a11y/no-redundant-roles": "off",
      "jsx-a11y/no-static-element-interactions": "off",
      "jsx-a11y/role-has-required-aria-props": "off",
      "jsx-a11y/role-supports-aria-props": "off",
      "jsx-a11y/scope": "off",
      "jsx-a11y/tabindex-no-positive": "off",
    }
  },
];
