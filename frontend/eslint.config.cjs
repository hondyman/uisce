const pluginImport = require('eslint-plugin-import');

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
      // Disallow console usage except console.error
      "no-restricted-syntax": [
        "error",
        {
          selector: "CallExpression[callee.object.name='console'][callee.property.name!= 'error']",
          message: "Use the devLogger (devLog/devDebug/devWarn) for non-error console output or remove console statements. Only console.error is allowed."
        }
      ],
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
      "no-restricted-syntax": "off"
    }
  },
];
