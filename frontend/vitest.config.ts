import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    dedupe: ['react', 'react-dom'],
    alias: {
      '@': path.resolve(__dirname, 'src'),
      react: path.resolve(__dirname, 'node_modules/react'),
      'react-dom': path.resolve(__dirname, 'node_modules/react-dom'),
      // Prevent Monaco from being prebundled in tests; we provide a lightweight mock in src/vitest/__mocks__
      'monaco-editor': path.resolve(__dirname, 'src/vitest/__mocks__/monaco-editor.js')
    }
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

      'src/vitest/components/**',

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
