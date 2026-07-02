import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Get backend host from environment or use defaults
// Backend + Auth run in LOCAL compose on MacBook
// Hasura (GraphQL) runs on REMOTE server - NEVER on localhost
const BACKEND_HOST = process.env.VITE_BACKEND_HOST || process.env.BACKEND_HOST || 'http://localhost:8082';
const PLATFORM_BACKEND_HOST = process.env.VITE_PLATFORM_BACKEND_HOST || process.env.PLATFORM_BACKEND_HOST || 'http://localhost:8083';
const GRAPHQL_HOST = process.env.VITE_GRAPHQL_HOST || process.env.GRAPHQL_HOST || 'http://100.84.126.19:8085';

console.log('[Vite Config] Backend Host:', BACKEND_HOST);
console.log('[Vite Config] Platform Backend Host:', PLATFORM_BACKEND_HOST);
console.log('[Vite Config] GraphQL Host:', GRAPHQL_HOST);

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
      'vscode-languageserver-types/lib/esm/main.js': 'vscode-languageserver-types',
    },
  },
  optimizeDeps: {},
  server: {
    proxy: {
      // Platform/auth/admin routes are served by the full backend (ABAC, auth,
      // tenant management).  Semantic/rules routes stay on the dedicated
      // semantic-rules-api for local dev.
      '/api/auth': {
        target: PLATFORM_BACKEND_HOST,
        changeOrigin: true,
        secure: false,
      },
      '/api/tenants': {
        target: PLATFORM_BACKEND_HOST,
        changeOrigin: true,
        secure: false,
      },
      '/api/admin': {
        target: PLATFORM_BACKEND_HOST,
        changeOrigin: true,
        secure: false,
      },
      '/api': {
        target: BACKEND_HOST,
        changeOrigin: true,
        secure: false,
      },
      '/v1/graphql': {
        target: GRAPHQL_HOST,
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