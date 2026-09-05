const pluginImport = require('eslint-plugin-import');
const pluginJsxA11y = require('eslint-plugin-jsx-a11y');

module.exports = [
  {
    ignores: ["dist/**"],
  },
  {
    files: ["**/*.{js,jsx,mjs,cjs,ts,tsx}"],
    plugins: {
      "@typescript-eslint": require("@typescript-eslint/eslint-plugin"),
      "react": require("eslint-plugin-react"),
      "react-hooks": require("eslint-plugin-react-hooks"),
      import: pluginImport,
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
      // Locale-prefixed routes: forbid direct imports of `Link`/`NavLink` from
      // react-router-dom so every internal navigation is prefixed with the
      // current locale. Use the wrappers in src/routes/Link.tsx instead.
      "no-restricted-imports": [
        "error",
        {
          "paths": [
            {
              "name": "react-router-dom",
              "importNames": ["Link"],
              "message": "Import Link from 'src/routes/Link' so paths are locale-prefixed."
            },
            {
              "name": "react-router-dom",
              "importNames": ["NavLink"],
              "message": "Import NavLink from 'src/routes/Link' so paths are locale-prefixed."
            }
          ]
        }
      ],
      // Disallow console usage except console.error
      "no-restricted-syntax": [
        "error",
        {
          selector: "CallExpression[callee.object.name='console'][callee.property.name!= 'error']",
          message: "Use the devLogger (devLog/devDebug/devWarn) for non-error console output or remove console statements. Only console.error is allowed."
        }
      ],
      // jsx-a11y gate — Phase 2 accessibility enforcement.
      // Set to "warn" initially so existing violations surface in CI output without
      // blocking the build. Upgrade to "error" rule-by-rule as violations are fixed.
      //
      // axe-core → jsx-a11y mapping (rules confirmed present in eslint-plugin-jsx-a11y 6.x):
      //   aria-input-field-name → label-has-associated-control
      //   label                → label-has-associated-control + control-has-associated-label
      //   scrollable-region     → no-static-element-interactions + no-noninteractive-tabindex
      //   aria-prohibited-attr → aria-props
      //   nested-interactive   → no-static-element-interactions
      //   aria-command-name    → anchor-has-content (for anchors used as commands)
      //
      // axe-core rules with NO jsx-a11y equivalent in this version:
      //   button-name, select-name, aria-progressbar-name, list, listitem
      // These are handled by the Playwright axe-core crawler (the authoritative source).
      "jsx-a11y/label-has-associated-control": "warn",
      "jsx-a11y/control-has-associated-label": "warn",
      "jsx-a11y/no-static-element-interactions": "warn",
      "jsx-a11y/click-events-have-key-events": "warn",
      "jsx-a11y/aria-props": "warn",
      "jsx-a11y/anchor-has-content": "warn",
      "jsx-a11y/no-noninteractive-tabindex": "warn",
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
      // Turn off jsx-a11y rules for test files and dev tools — these are
      // harnesses, not production components; the a11y crawler handles them.
      "jsx-a11y/label-has-associated-control": "off",
      "jsx-a11y/control-has-associated-label": "off",
      "jsx-a11y/no-static-element-interactions": "off",
      "jsx-a11y/click-events-have-key-events": "off",
      "jsx-a11y/aria-props": "off",
      "jsx-a11y/anchor-has-content": "off",
      "jsx-a11y/no-noninteractive-tabindex": "off",
    }
  },
];
