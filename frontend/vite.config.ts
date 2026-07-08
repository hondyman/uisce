import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// Backend routing — single Go execution binary owns all /api/* routes.
// The fresh Go backend listens on http://localhost:8080 and serves the unified
// surface: auth, tenants, admin, catalog, glossary, node-types, edge-types,
// semantic-bundles, business-objects, etc. The previous indirection through
// PLATFORM_BACKEND_HOST (default :8083) was wired for a multi-service compose
// layout where the platform backend was on :8083 and the semantic-rules-api
// was on :8080/:8082. With the unified binary on :8080, ALL /api/* traffic
// must route to http://localhost:8080 — otherwise the SPA ends up hitting
// its own dev server on :5173 (404) or an unreachable port (400/401/403).
//
// `BACKEND_HOST` is preserved as a derived alias for any code that still
// reads it (semantic-rules surface), and `GRAPHQL_HOST` keeps its env hook
// because /v1/graphql is a placeholder that returns 503 on the unified
// binary and is not user-visible.
const PLATFORM_BACKEND_HOST = 'http://localhost:8080';
const BACKEND_HOST = process.env.VITE_BACKEND_HOST || process.env.BACKEND_HOST || PLATFORM_BACKEND_HOST;
const GRAPHQL_HOST = process.env.VITE_GRAPHQL_HOST || process.env.GRAPHQL_HOST || PLATFORM_BACKEND_HOST;

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
