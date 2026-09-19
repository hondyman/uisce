import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { WebSocketService } from '@/services/WebSocketService';

class MockWebSocket {
  static instances: MockWebSocket[] = [];
  url: string;
  readyState: number = 0; // CONNECTING
  onopen: (() => void) | null = null;
  onclose: ((event: { code: number; reason: string }) => void) | null = null;
  onerror: ((error: any) => void) | null = null;
  onmessage: ((event: { data: string }) => void) | null = null;
  sentMessages: string[] = [];

  constructor(url: string) {
    this.url = url;
    MockWebSocket.instances.push(this);
    // Simulate auto-open on next tick
    setTimeout(() => {
      this.readyState = 1; // OPEN
      if (this.onopen) this.onopen();
    }, 10);
  }

  send(data: string) {
    this.sentMessages.push(data);
  }

  close(code: number = 1000, reason: string = '') {
    this.readyState = 3; // CLOSED
    if (this.onclose) this.onclose({ code, reason });
  }
}

describe('WebSocketService Ticket Authentication', () => {
  let originalWebSocket: any;
  let originalFetch: any;

  beforeEach(() => {
    originalWebSocket = globalThis.WebSocket;
    originalFetch = globalThis.fetch;
    MockWebSocket.instances = [];
    (globalThis as any).WebSocket = MockWebSocket;
    localStorage.setItem('auth_token', 'test.valid.jwt');
  });

  afterEach(() => {
    globalThis.WebSocket = originalWebSocket;
    globalThis.fetch = originalFetch;
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it('acquires an ephemeral ticket prior to opening the WebSocket connection', async () => {
    let ticketCallCount = 0;
    globalThis.fetch = vi.fn().mockImplementation(async (url: string) => {
      if (url === '/api/ws/ticket') {
        ticketCallCount++;
        return {
          ok: true,
          status: 200,
          json: async () => ({ ticket: `ticket_seq_${ticketCallCount}`, expires_in: 30 }),
        };
      }
      return { ok: false, status: 404 };
    });

    const service = new WebSocketService('ws://localhost:8081/api/ws');
    await service.connect();

    expect(ticketCallCount).toBe(1);
    expect(MockWebSocket.instances.length).toBe(1);
    expect(MockWebSocket.instances[0].url).toBe('ws://localhost:8081/api/ws?ticket=ticket_seq_1');

    service.disconnect();
  });

  it('triggers onAuthError on 401 when fetching ticket and does not establish connection', async () => {
    globalThis.fetch = vi.fn().mockImplementation(async (url: string) => {
      if (url === '/api/ws/ticket') {
        return {
          ok: false,
          status: 401,
          json: async () => ({ error: 'unauthorized' }),
        };
      }
      return { ok: false, status: 404 };
    });

    const authErrorSpy = vi.fn();
    const service = new WebSocketService('ws://localhost:8081/api/ws');
    service.onAuthError(authErrorSpy);

    await expect(service.connect()).rejects.toThrow('Unauthorized');
    expect(authErrorSpy).toHaveBeenCalledTimes(1);
    expect(MockWebSocket.instances.length).toBe(0);

    service.disconnect();
  });

  it('acquires a fresh ticket on reconnect rather than reusing the old one', async () => {
    vi.useFakeTimers();
    let ticketSequence = 0;
    globalThis.fetch = vi.fn().mockImplementation(async (url: string) => {
      if (url === '/api/ws/ticket') {
        ticketSequence++;
        return {
          ok: true,
          status: 200,
          json: async () => ({ ticket: `fresh_ticket_${ticketSequence}`, expires_in: 30 }),
        };
      }
      return { ok: false, status: 404 };
    });

    const service = new WebSocketService('ws://localhost:8081/api/ws');
    const connectPromise = service.connect();
    await vi.advanceTimersByTimeAsync(20);
    await connectPromise;

    expect(MockWebSocket.instances.length).toBe(1);
    expect(MockWebSocket.instances[0].url).toContain('ticket=fresh_ticket_1');

    // Simulate connection drop
    MockWebSocket.instances[0].close(1006, 'Abnormal closure');

    // Wait for reconnect timer (1000ms * 1st attempt)
    await vi.advanceTimersByTimeAsync(1100);

    expect(MockWebSocket.instances.length).toBe(2);
    // Ticket MUST be fresh_ticket_2, NOT fresh_ticket_1!
    expect(MockWebSocket.instances[1].url).toContain('ticket=fresh_ticket_2');

    service.disconnect();
    vi.useRealTimers();
  });
});
