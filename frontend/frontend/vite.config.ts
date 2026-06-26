import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, '../src'),
    },
  },
  optimizeDeps: {
    exclude: ['vscode-languageserver-types', 'monaco-yaml'],
  },
  build: {
    rollupOptions: {
      external: ['vscode-languageserver-types', 'monaco-yaml'],
      commonjsOptions: {
        ignore: ['vscode-languageserver-types'],
      },
    },
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
      },
      '/v1/graphql': {
        target: 'http://100.84.126.19:8080',
        changeOrigin: true,
        secure: false,
        ws: true,
        rewrite: (path) => {
          console.log('[GraphQL Proxy] Rewriting path:', path);
          return path;
        },
      },
    },
  },
})