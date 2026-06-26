import { defineConfig } from 'vitest/config'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: './vitest.setup.ts',

    // Only run tests placed explicitly for Vitest
    include: ['src/vitest/**/*.test.ts', 'src/vitest/**/*.test.tsx'],

    // Keep Vitest from touching Jest and Playwright suites
    exclude: [
      '**/__tests__/**',
      '**/*.spec.ts',
      '**/*.spec.tsx',
      'src/components/**/__tests__/**',
      'src/pages/**/__tests__/**',
      'src/hooks/**/__tests__/**',
      'src/features/**/__tests__/**',
      'src/**/playwright/**',
    ],

    // Prevent Vite from trying to pre-bundle Monaco; use mock alias
    deps: {
      external: ['monaco-editor'],
      inline: ['@monaco-editor/react'],
    }
  },

  resolve: {
    alias: {
      '@': path.resolve(__dirname, 'src'),
      'monaco-editor': path.resolve(__dirname, 'src/vitest/__mocks__/monaco-editor.js')
    },
  }
})
