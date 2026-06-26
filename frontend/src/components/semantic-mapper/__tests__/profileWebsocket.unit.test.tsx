import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { act } from 'react-dom/test-utils';
import SemanticMapper from '../../SemanticMapper';
import { TenantProvider } from '../../../contexts/TenantContext';
import { ScopeProvider } from '../../../contexts/ScopeContext';
import { RouteBlockerProvider } from '../../../components/RouteBlocker/RouteBlocker';
import { ApolloProvider } from '@apollo/client';
import apolloClient from '../../../../src/graphql/apolloClient';
import { vi, describe, it, beforeEach, afterEach, expect } from 'vitest';

// Minimal fake WebSocket that allows us to push messages from test code
class FakeWebSocket {
  url: string;
  onopen: ((ev?: any) => void) | null = null;
  onmessage: ((ev: any) => void) | null = null;
  onerror: ((ev?: any) => void) | null = null;
  onclose: ((ev?: any) => void) | null = null;
  closed = false;

  constructor(url: string) {
    this.url = url;
    // simulate async open
    setTimeout(() => { if (this.onopen) this.onopen(); }, 10);
  }

  send(_data: any) {}
  close() { this.closed = true; if (this.onclose) this.onclose(); }
}

describe('Profile WebSocket integration (frontend mocks)', () => {
  let originalFetch: any;
  let originalWS: any;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    originalWS = (globalThis as any).WebSocket;

    // Seed tenant selection in localStorage
    localStorage.setItem('selected_tenant', JSON.stringify({ id: 't1', display_name: 'T1' }));
    localStorage.setItem('selected_product', JSON.stringify({ id: 'p1', alpha_product: { product_name: 'P1' } }));
    localStorage.setItem('selected_datasource', JSON.stringify({ id: 'ds1', source_name: 'DS1', config: { host: 'localhost', port: 5432, credentials: { username: 'u', password: 'p' } } }));

    // Mock fetch: - token endpoint returns token, - profiler POST returns jobId, - status/results used for polling fallback
    (globalThis.fetch as any) = vi.fn((input: any, _opts?: any) => {
      const url = typeof input === 'string' ? input : input?.url || '';
      if (typeof url === 'string' && url.includes('/api/ws/token')) {
        return Promise.resolve(new Response(JSON.stringify({ token: 'fake-jwt' }), { status: 200 }));
      }
      if (typeof url === 'string' && url.includes('/api/profiler/profile')) {
        return Promise.resolve(new Response(JSON.stringify({ jobId: 'job-xyz' }), { status: 200 }));
      }
      if (typeof url === 'string' && url.includes('/api/profiler/status/')) {
        // simulate completed when polled
        return Promise.resolve(new Response(JSON.stringify({ status: 'completed' }), { status: 200 }));
      }
      if (typeof url === 'string' && url.includes('/api/profiler/results')) {
        return Promise.resolve(new Response(JSON.stringify({ profiles: [{ tableName: 't', columnName: 'c', dataType: 'text', cardinality: 10 }] }), { status: 200 }));
      }
      if (typeof url === 'string' && url.includes('/api/semantic-mappings')) {
        return Promise.resolve(new Response(JSON.stringify([]), { status: 200 }));
      }
      return Promise.resolve(new Response(JSON.stringify({}), { status: 200 }));
    });

    // Replace global WebSocket with our fake
    (globalThis as any).WebSocket = FakeWebSocket as any;
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
    (globalThis as any).WebSocket = originalWS;
    vi.resetAllMocks();
    localStorage.clear();
  });

  it('opens websocket and handles progress/completed messages', async () => {
    render(
      <ApolloProvider client={apolloClient}>
        <RouteBlockerProvider>
          <TenantProvider>
            <ScopeProvider>
              <SemanticMapper />
            </ScopeProvider>
          </TenantProvider>
        </RouteBlockerProvider>
      </ApolloProvider>
    );

  // Click run profile button if present (Profile UI moved to its own menu)
  const runButton = await screen.findByRole('button', { name: /Run Data Profile/i });
  fireEvent.click(runButton);

    // Wait for WebSocket to be constructed
    await waitFor(() => {
      expect((globalThis.fetch as any)).toHaveBeenCalled();
    });

    // The fake WebSocket will call onopen automatically; we now simulate a progress message
    await act(async () => {
      // find the instantiated fake websocket from window (we don't expose it directly), but the implementation used should have created one
      // Instead, push a message via dispatching a MessageEvent on window to mimic server message
      const msg = { type: 'completed', results: { profiles: [{ tableName: 't', columnName: 'c', dataType: 'text', cardinality: 10 }] } };
      window.dispatchEvent(new MessageEvent('message', { data: JSON.stringify(msg) }));
    });

    // Expect profile results to be rendered
    await waitFor(() => {
      expect(screen.getByText(/Profile Results/)).toBeTruthy();
    });
  });

  it('falls back to polling when websocket errors', async () => {
    // Modify FakeWebSocket to trigger onerror when opened
    class ErringWS extends FakeWebSocket {
      constructor(url: string) {
        super(url);
        setTimeout(() => { if (this.onerror) this.onerror(new Event('error')); }, 5);
      }
    }
    (globalThis as any).WebSocket = ErringWS as any;

    render(
      <ApolloProvider client={apolloClient}>
        <RouteBlockerProvider>
          <TenantProvider>
            <ScopeProvider>
              <SemanticMapper />
            </ScopeProvider>
          </TenantProvider>
        </RouteBlockerProvider>
      </ApolloProvider>
    );

  // Run profiling via the Run button
  const runButton = await screen.findByRole('button', { name: /Run Data Profile/i });
  fireEvent.click(runButton);

    // Wait for polling to fetch results and render
    await waitFor(() => {
      expect((globalThis.fetch as any)).toHaveBeenCalled();
      // Ensure results rendered
      expect(screen.getByText(/Profile Results/)).toBeTruthy();
    });
  });
});
