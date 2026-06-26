import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Get backend host from environment or use defaults
// Backend + Auth run in LOCAL compose on MacBook
// Hasura (GraphQL) runs on REMOTE server - NEVER on localhost
const BACKEND_HOST = process.env.VITE_BACKEND_HOST || process.env.BACKEND_HOST || 'http://localhost:8082';
const AUTH_SERVICE_HOST = process.env.VITE_AUTH_HOST || process.env.AUTH_HOST || 'http://localhost:3001';
const GRAPHQL_HOST = process.env.VITE_GRAPHQL_HOST || process.env.GRAPHQL_HOST || 'http://100.84.126.19:8085';

console.log('[Vite Config] Backend Host:', BACKEND_HOST);
console.log('[Vite Config] Auth Service Host:', AUTH_SERVICE_HOST);
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
      '/api/auth': {
        target: AUTH_SERVICE_HOST,
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