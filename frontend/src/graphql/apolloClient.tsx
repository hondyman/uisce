import { ApolloClient, InMemoryCache, HttpLink, from, ApolloLink, Observable } from '@apollo/client';
import { onError } from '@apollo/client/link/error';
import NotificationService from '../services/NotificationService';
import { setContext } from '@apollo/client/link/context';
import { devDebug, devError } from '../utils/devLogger';

// The HTTP link is the connection to your GraphQL endpoint.
// CENTRALIZED PORT ALLOCATION: Load endpoint from .env.ports via VITE_GRAPHQL_ENDPOINT
// The endpoint is set in frontend/.env and frontend/.env.local as:
//   VITE_GRAPHQL_ENDPOINT=http://localhost:${PORT_HASURA_GRAPHQL}/v1/graphql
//
// NEVER hardcode the port number here. Always use the environment variable.
// If the endpoint is not set in env, fall back to relative path /v1/graphql.
const envEndpoint = (import.meta.env.VITE_GRAPHQL_ENDPOINT as string) || '';
let graphqlEndpoint = envEndpoint || '';
try {
  if (!graphqlEndpoint) {
    // Fallback: use relative path (works when proxying via dev server)
    graphqlEndpoint = '/v1/graphql';
    devDebug && devDebug('[apollo] No VITE_GRAPHQL_ENDPOINT in env; using relative fallback:', graphqlEndpoint);
  }
} catch (e) {}
if (!graphqlEndpoint) graphqlEndpoint = '/v1/graphql';

const httpLink = new HttpLink({
  uri: graphqlEndpoint,
});

// DEV: log chosen endpoint to browser console so runtime requests can be traced easily
try {
  if (typeof window !== 'undefined' && import.meta.env.DEV) {
    devDebug('[apollo] graphqlEndpoint =', graphqlEndpoint);
    devDebug('[apollo] VITE_GRAPHQL_ENDPOINT =', import.meta.env.VITE_GRAPHQL_ENDPOINT);
    devDebug('[apollo] import.meta.env.PROD =', import.meta.env.PROD);
  }
} catch (e) {}

// Fallback link: if the network request fails (gateway/Hasura down),
// return a minimal empty-data response so the UI can handle it gracefully
// without crashing. We log the original error for diagnostics.
const fallbackLink = new ApolloLink((operation, forward) => {
  if (!forward) {
    // No downstream handler — return empty observable
    return Observable.of({ data: {} } as any);
  }

  return new Observable(observer => {
    let sub: any = null;
    try {
      sub = forward(operation).subscribe({
        next: (result: any) => observer.next(result),
        error: (err: any) => {
          // Log the network error silently, then provide a safe fallback result that
          // won't crash components expecting `data` to be present.
          try {
            // Only log in verbose debug mode to avoid console spam
            if (typeof window !== 'undefined' && (window as any).__DEBUG_APOLLO) {
              devError('[apollo][fallback] network error for', operation.operationName, err);
            }
          } catch (e) {}
          // If fallback on down is explicitly enabled, return safe empty data.
          // Otherwise, propagate the error so components enter an error state.
          const fallbackEnabled = String(import.meta.env.VITE_GRAPHQL_FALLBACK_ON_DOWN || '') === 'true';
          if (!fallbackEnabled) {
            observer.error(err);
            return;
          }
          // If the operation appears to be fetching catalog_node rows, return
          // an empty catalog_node array so consumers (like useEnhancedSemanticTerms)
          // can detect the empty result and attempt REST fallbacks.
          const opName = operation.operationName || '';
          const isCatalogNodeQuery = opName.includes('GetSemanticTermsWithMetadata') || opName.includes('catalog_node') || String(operation.query).includes('catalog_node');
          if (isCatalogNodeQuery) {
            observer.next({ data: { catalog_node: [] } } as any);
          } else {
            // For tenants and other queries, return empty data
            observer.next({ data: {} } as any);
          }
          observer.complete();
        },
        complete: () => observer.complete(),
      });
    } catch (err) {
      try {
        if (typeof window !== 'undefined' && (window as any).__DEBUG_APOLLO) {
          devError('[apollo][fallback] unexpected error', err);
        }
      } catch (e) {}
      observer.next({ data: {} } as any);
      observer.complete();
    }

    return () => {
      if (sub) sub.unsubscribe();
    };
  });
});

import { getRequiredTenantScope, hasTenantScope } from '../utils/tenantScope';

const authLink = setContext((_, { headers }) => {
  const adminSecret = (import.meta.env.VITE_GRAPHQL_ADMIN_SECRET as string) || '';
  const token = localStorage.getItem('auth_token');

  const outHeaders: Record<string, any> = { ...headers };

  // Inject Tenant Scope
  if (hasTenantScope()) {
    const { tenantId, datasourceId } = getRequiredTenantScope();
    outHeaders['X-Tenant-ID'] = tenantId;
    outHeaders['X-Tenant-Datasource-ID'] = datasourceId;
  }

  // Always attach the admin secret when configured (needed for development)
  // Production environments should use JWT tokens instead
  if (adminSecret) {
    outHeaders['x-hasura-admin-secret'] = adminSecret;
  }

  // Validate token before sending - must have 3 parts (header.payload.signature)
  if (token) {
    const parts = token.split('.');
    if (parts.length === 3) {
      outHeaders['Authorization'] = `Bearer ${token}`;
      devDebug('[apollo] JWT token attached, length:', token.length);
    } else {
      devError('[apollo] Invalid token format, expected 3 parts, got:', parts.length, 'token:', token.substring(0, 50));
    }
  }

  return { headers: outHeaders };
});

// Error link: intercept and optionally suppress or log GraphQL / network errors.
// By default we suppress noisy logs (like connection refused when Hasura is down)
// unless the __DEBUG_APOLLO flag is set in the browser. This keeps dev console
// output clean during local development when Hasura isn't running.
// throttle notifications to avoid repeat spamming
let lastGraphqlNotify = 0;
const GRAPHQL_NOTIFY_TTL = 60 * 1000; // 1 minute
let isGraphqlDown = false;
let recoveryPolling = false;
const GRAPHQL_RECOVERY_POLL_MS = Number(import.meta.env.VITE_GRAPHQL_RECOVERY_POLL || '15000');

async function startRecoveryPolling() {
  if (recoveryPolling) return;
  recoveryPolling = true;
  const endpoint = graphqlEndpoint;

  while (recoveryPolling) {
    try {
      // Use OPTIONS to check endpoint availability without running a GraphQL query.
      const resp = await fetch(endpoint, { method: 'OPTIONS', cache: 'no-cache' });
      if (resp && resp.ok) {
        window.dispatchEvent(new CustomEvent('semlayer.graphqlOutage', { detail: { down: false } }));
        isGraphqlDown = false;
        recoveryPolling = false;
        break;
      }
    } catch (e) {
      // still down
    }

    // Sleep until next try
    await new Promise(res => setTimeout(res, GRAPHQL_RECOVERY_POLL_MS));
  }
}

const errorLink = onError(({ graphQLErrors, networkError, operation }) => {
  try {
    const debug = typeof window !== 'undefined' && (window as any).__DEBUG_APOLLO;
    if (graphQLErrors && graphQLErrors.length) {
      // Only print GraphQL errors in debug mode
      if (debug) {
        graphQLErrors.forEach(err => devError('[apollo][gql-error]', operation.operationName, err));
      }
    }

    if (networkError) {
      // The network 'Failed to fetch' / ECONNREFUSED messages are noisy during dev.
      // If debug is enabled, show them; otherwise swallow silently since we
      // already have a fallback link that returns safe empty data.
      const now = Date.now();
      if (debug) {
        devError('[apollo][network-error]', operation.operationName, networkError);
      }

      // Notify user that GraphQL is not available. Only notify once every
      // GRAPHQL_NOTIFY_TTL milliseconds to avoid spamming on repeated queries.
      if (now - lastGraphqlNotify > GRAPHQL_NOTIFY_TTL) {
        lastGraphqlNotify = now;
        try {
          NotificationService.warn('GraphQL endpoint unreachable. Some features may be unavailable.');
        } catch (e) {}
      }

      // dispatch a global outage event so a persistent banner can show for the outage
      try {
        if (!isGraphqlDown && String(import.meta.env.VITE_GRAPHQL_NOTIFY_ON_DOWN || '') !== 'false') {
          isGraphqlDown = true;
          window.dispatchEvent(new CustomEvent('semlayer.graphqlOutage', { detail: { down: true } }));
          startRecoveryPolling();
        }
      } catch (e) {}
    }
  } catch (e) {
    // Defensive: don't allow error handlers to throw during error handling.
  }
});

// The client is created by chaining the authLink and the httpLink.
// The authLink runs first, adding the header, then the httpLink sends the request.
const client = new ApolloClient({
  // Order: authLink -> errorLink -> fallbackLink -> httpLink
  // fallbackLink intercepts network failures from httpLink and returns
  // a safe empty data payload so UI components can render without fatal
  // uncaught exceptions. Keep authLink first so admin headers are applied.
  link: from([authLink, errorLink, fallbackLink, httpLink]),
  cache: new InMemoryCache({
    // Disable all caching to force fresh schema fetches
    resultCaching: false,
  }),
  // Force all queries to go to network
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'network-only',
      errorPolicy: 'all',
    },
    query: {
      fetchPolicy: 'network-only',
      errorPolicy: 'all',
    },
    mutate: {
      errorPolicy: 'all',
    },
  },
  // Enable Apollo Client DevTools in development
  devtools: { enabled: import.meta.env.DEV },
});

export default client;