import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

// Pin the zone so tests that read the machine's time zone (Intl default, Date
// formatting) give the same answer on a laptop and on the UTC CI runner. Set
// here, before vitest spawns its workers, so they inherit it.
process.env.TZ = 'UTC'

export default defineConfig({
  plugins: [react()],
  resolve: {
    dedupe: ['react', 'react-dom'],
    alias: [
      { find: '@', replacement: path.resolve(__dirname, 'src') },
      { find: /^react$/, replacement: path.resolve(__dirname, 'node_modules/react') },
      { find: /^react-dom$/, replacement: path.resolve(__dirname, 'node_modules/react-dom') },
      // Wave A: monaco-editor/esm alias — handle deep imports through the same mock.
      // The bare 'monaco-editor' alias above only matches the package root, not
      // subpath imports like 'monaco-editor/esm/vs/editor/editor.api'.
      { find: /^monaco-editor(\/esm)?/, replacement: path.resolve(__dirname, 'src/vitest/__mocks__/monaco-editor.js') },
      // Wave A: RTL renderHook import swap. @testing-library/react-hooks is
      // deprecated; its renderHook/waitFor/etc. live on @testing-library/react
      // in v13+. Aliasing lets existing imports keep working without touching
      // each test file.
      { find: '@testing-library/react-hooks', replacement: '@testing-library/react' }
    ]
  },

  test: {
    globals: true,
    root: path.resolve(__dirname),
    // Only include tests that live under src/vitest/**
    include: ['src/vitest/**/*.test.ts', 'src/vitest/**/*.test.tsx'],

    // Explicitly exclude legacy Jest tests and spec files
    exclude: [
      '**/__tests__/**',
      '**/*.spec.ts',
      '**/*.spec.tsx',

      'src/components/**/*.test.ts',
      'src/components/**/*.test.tsx',
      'src/components/**/__tests__/**',

      'src/pages/**/*.test.ts',
      'src/pages/**/*.test.tsx',
      'src/pages/**/__tests__/**',

      'src/features/**/*.test.ts',
      'src/features/**/*.test.tsx',
      'src/features/**/__tests__/**',

      'src/hooks/**/*.test.ts',
      'src/hooks/**/*.test.tsx',
      'src/hooks/**/__tests__/**',

      'src/api/**/*.test.ts',
      'src/api/**/*.test.tsx',
      'src/api/**/__tests__/**'
    ],

    deps: {
      optimizer: {
        web: {
          include: []
        }
      }
    },

    environment: 'jsdom',
    setupFiles: path.resolve(__dirname, 'vitest.setup.ts')
  }
})
