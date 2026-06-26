module.exports = {
  root: true,
  parser: '@typescript-eslint/parser',
  plugins: ['react', 'react-hooks', '@typescript-eslint'],
  extends: [
    'eslint:recommended',
    'plugin:react/recommended',
    'plugin:@typescript-eslint/recommended'
  ],
  plugins: ['jsx-a11y'],
  rules: {
    // Weakening this rule avoids static false-positives for dynamic ARIA values in some components
    'jsx-a11y/aria-proptypes': 'warn'
  },
  settings: {
    react: {
      version: 'detect'
    }
  },
  rules: {
    // Enforce rules of hooks
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'warn',
    // Allow some leeway for JSX in TSX
    'react/jsx-uses-react': 'off',
    'react/react-in-jsx-scope': 'off'
  }
};
