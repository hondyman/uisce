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
const restTranslationLink = new ApolloLink((operation) => {
  return new Observable((observer) => {
    const { query, variables } = operation;
    const queryStr = query.loc?.source.body || '';
    const headers = operation.getContext().headers || {};

    const performRestRequest = async () => {
      try {
        let url = '';
        let method = 'GET';
        let body: any = null;
        let transform = (data: any) => data;

        if (queryStr.includes('GetAllProducts') || queryStr.includes('alpha_product')) {
          url = '/api/rest/products';
          transform = (data) => ({ alpha_product: data });
        } else if (queryStr.includes('GetSemanticModels') || queryStr.includes('semantic_models')) {
          url = '/api/rest/semantic-models';
          transform = (data) => ({ semantic_models: data });
        } else if (queryStr.includes('GetTechnicalLineageChart') || queryStr.includes('GetSemanticLineageChart') || queryStr.includes('GetCombinedChart') || queryStr.includes('tenant_chart')) {
          const dsId = variables.datasourceId;
          url = `/api/rest/charts?tenant_datasource_id=${dsId}`;
          if (variables.chartName) {
            url += `&chart_name=${variables.chartName}`;
          }
          transform = (data) => ({ tenant_chart: data });
        } else if (queryStr.includes('GetSemanticAssets') || queryStr.includes('semantic_assets')) {
          const beId = variables.businessEntityId;
          const dsId = variables.datasourceId;
          url = `/api/rest/semantic-assets?business_entity_id=${beId}&tenant_instance_id=${dsId}`;
          transform = (data) => ({ semantic_assets: data });
        } else if (queryStr.includes('GetRelationshipSuggestions') || queryStr.includes('relationship_suggestions')) {
          const beId = variables.businessEntityId;
          const dsId = variables.datasourceId;
          const limit = variables.limit || 100;
          url = `/api/rest/relationship-suggestions?source_entity_id=${beId}&tenant_instance_id=${dsId}&limit=${limit}`;
          transform = (data) => ({ relationship_suggestions: data });
        } else if (queryStr.includes('GetLinkedModels') || queryStr.includes('catalog_edge')) {
          const modelId = variables.modelId;
          const dsId = variables.datasourceId;
          url = `/api/rest/catalog-edges?source_node_id=${modelId}&tenant_datasource_id=${dsId}&relationship_types=joins,references,extends`;
          transform = (data) => ({ catalog_edge: data });
        } else if (queryStr.includes('GetAvailableDatasources') || queryStr.includes('alpha_datasource')) {
          url = '/api/rest/datasources';
          transform = (data) => ({ alpha_datasource: data });
        } else if (queryStr.includes('GetSemanticTerms') || queryStr.includes('catalog_node')) {
          if (queryStr.includes('business_terms') && queryStr.includes('semantic_terms')) {
            const dsId = variables.datasourceId;
            const [nodes, edges] = await Promise.all([
              fetch(`/api/rest/catalog-nodes?tenant_datasource_id=${dsId}`, { headers }).then(r => r.json()),
              fetch(`/api/rest/catalog-edges?tenant_datasource_id=${dsId}&relationship_types=SemanticToView,SemanticViewToColumn`, { headers }).then(r => r.json())
            ]);
            const businessTerms = Array.isArray(nodes) ? nodes.filter((n: any) => n.node_type_id === '21645d21-de5f-4feb-af99-99273ea75626') : [];
            const semanticTerms = Array.isArray(nodes) ? nodes.filter((n: any) => n.node_type_id === '820b942a-9c9e-4abc-acdc-84616db33098') : [];
            const semanticViews = Array.isArray(nodes) ? nodes.filter((n: any) => n.node_type_id === 'c53f9e99-8d02-4dfb-bc1b-914747d35edb') : [];
            observer.next({
              data: {
                business_terms: businessTerms,
                semantic_terms: semanticTerms,
                semantic_views: semanticViews,
                business_edges: edges
              }
            });
            observer.complete();
            return;
          } else {
            const dsId = variables.datasourceId;
            const nodeTypeId = variables.nodeTypeId || '820b942a-9c9e-4abc-acdc-84616db33098';
            url = `/api/rest/catalog-nodes?tenant_datasource_id=${dsId}&node_type_id=${nodeTypeId}`;
            transform = (data) => ({ catalog_node: data });
          }
        } else if (queryStr.includes('GetTablesForDatasource') || queryStr.includes('catalog_node_vw')) {
          const dsId = variables.datasourceId;
          const q = variables.q || '';
          const limit = variables.limit || 100;
          url = `/api/rest/catalog-nodes?tenant_datasource_id=${dsId}&q=${encodeURIComponent(q)}&limit=${limit}&use_view=true`;
          if (queryStr.includes('GetColumnsForTable')) {
            const parentId = variables.parentId;
            url = `/api/rest/catalog-nodes?tenant_datasource_id=${dsId}&parent_id=${parentId}&q=${encodeURIComponent(q)}&limit=${limit}&use_view=true`;
          }
          transform = (data) => ({ catalog_node_vw: data });
        } else if (queryStr.includes('GetSchemaTables')) {
          const dsId = variables.datasourceId;
          const nodes = await fetch(`/api/rest/catalog-nodes?tenant_datasource_id=${dsId}`, { headers }).then(r => r.json());
          const tables = Array.isArray(nodes) ? nodes.filter((n: any) => n.node_type_id === '49a50271-ae58-4d3e-ae1c-2f5b89d89192') : [];
          const columns = Array.isArray(nodes) ? nodes.filter((n: any) => n.node_type_id === 'a64c1011-16e8-4ddf-b447-363bf8e15c9a') : [];
          observer.next({
            data: {
              tables,
              columns
            }
          });
          observer.complete();
          return;
        } else if (queryStr.includes('GetCatalogNodeById')) {
          const dsId = variables.datasourceId;
          const nodeId = variables.nodeId;
          url = `/api/rest/catalog-nodes?tenant_datasource_id=${dsId}&id=${nodeId}&use_view=true`;
          transform = (data) => ({ catalog_node_vw: data });
        } else if (queryStr.includes('CreateDraft') || queryStr.includes('insert_fabric_defn_one')) {
          url = '/api/rest/fabric-defn';
          method = 'POST';
          body = JSON.stringify({ input: variables.input });
          transform = (data) => data;
        } else {
          console.warn('[restTranslationLink] Unhandled query, falling back to GraphQL endpoint:', queryStr);
          const response = await fetch(graphqlEndpoint, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              ...headers
            },
            body: JSON.stringify({ query: queryStr, variables })
          });
          const gqlRes = await response.json();
          observer.next(gqlRes);
          observer.complete();
          return;
        }

        const fetchOptions: RequestInit = {
          method,
          headers: {
            'Content-Type': 'application/json',
            ...headers
          }
        };
        if (body) {
          fetchOptions.body = body;
        }

        const resp = await fetch(url, fetchOptions);
        if (!resp.ok) {
          throw new Error(`HTTP error ${resp.status}: ${await resp.text()}`);
        }
        const data = await resp.json();
        observer.next({ data: transform(data) });
        observer.complete();
      } catch (err: any) {
        observer.error(err);
      }
    };

    performRestRequest();
  });
});

const client = new ApolloClient({
  // Order: authLink -> errorLink -> fallbackLink -> restTranslationLink
  link: from([authLink, errorLink, fallbackLink, restTranslationLink]),
  cache: new InMemoryCache({
    resultCaching: false,
  }),
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
  devtools: { enabled: import.meta.env.DEV },
});

export default client;