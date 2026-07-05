import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Get backend host from environment or use defaults
// Backend + Auth run in LOCAL compose on MacBook
// GraphQL (/v1/graphql) is served by the local platform backend on PLATFORM_BACKEND_HOST
// (semlayer-backend, default :8083) — the platform mounts /v1/graphql itself. The
// previous default of 100.84.126.19:8085 was a remote Hasura on a Tailscale IP that is
// no longer reachable; use the local platform backend instead.
//
// IMPORTANT: All /api/* routes (auth, tenants, admin, catalog, glossary,
// node-types, edge-types, semantic-bundles, etc.) are served by the "full"
// backend (semlayer-backend on PLATFORM_BACKEND_HOST, default :8083). The
// semantic-rules-api (BACKEND_HOST on :8080/:8082) only handles a dedicated
// semantic-rules surface — it does NOT implement catalog / glossary routes,
// so the catch-all /api proxy MUST point at PLATFORM_BACKEND_HOST, otherwise
// the BusinessTermsTab / Glossary / Catalog pages all 404.
const BACKEND_HOST = process.env.VITE_BACKEND_HOST || process.env.BACKEND_HOST || 'http://localhost:8082';
const PLATFORM_BACKEND_HOST = process.env.VITE_PLATFORM_BACKEND_HOST || process.env.PLATFORM_BACKEND_HOST || 'http://localhost:8080';
const GRAPHQL_HOST = process.env.VITE_GRAPHQL_HOST || process.env.GRAPHQL_HOST || 'http://localhost:8080';

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
      // tenant management). The semantic-rules-api is a separate dedicated
      // service that does not own these routes.
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
      // Catch-all /api proxy → PLATFORM_BACKEND_HOST (semlayer-backend,
      // :8083). It owns the catalog, glossary, node-types, edge-types,
      // semantic-bundles etc. routes used by the frontend. The semantic-
      // rules-api (BACKEND_HOST on :8080/:8082) does not implement these
      // endpoints and returns 404, which is what made the BusinessTermsTab
      // show "nodeTypes count: 0".
      '/api': {
        target: PLATFORM_BACKEND_HOST,
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
