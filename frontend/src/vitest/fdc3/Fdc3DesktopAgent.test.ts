import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import {
  Fdc3DesktopAgent,
  LocalFdc3Channel,
  BroadcastChannelTransport,
  WailsRelayTransport,
  Fdc3InstrumentContext,
  Fdc3PortfolioContext,
  ChannelMessageEnvelope,
  IFdc3Transport,
} from '../../services/fdc3';

// Mock BroadcastChannel for deterministic cross-channel simulation in Node/Vitest
class MockBroadcastChannel {
  public name: string;
  public onmessage: ((event: MessageEvent<any>) => void) | null = null;
  private static buses: Map<string, Set<MockBroadcastChannel>> = new Map();

  constructor(name: string) {
    this.name = name;
    let set = MockBroadcastChannel.buses.get(name);
    if (!set) {
      set = new Set();
      MockBroadcastChannel.buses.set(name, set);
    }
    set.add(this);
  }

  public postMessage(data: any): void {
    const set = MockBroadcastChannel.buses.get(this.name);
    if (set) {
      set.forEach((ch) => {
        // BroadcastChannel does NOT echo to self
        if (ch !== this && ch.onmessage) {
          ch.onmessage({ data } as MessageEvent);
        }
      });
    }
  }

  public close(): void {
    const set = MockBroadcastChannel.buses.get(this.name);
    if (set) {
      set.delete(this);
      if (set.size === 0) {
        MockBroadcastChannel.buses.delete(this.name);
      }
    }
  }

  public static resetAll(): void {
    MockBroadcastChannel.buses.clear();
  }
}

describe('FDC3 Interoperability Bus & Desktop Agent', () => {
  const originalBC = globalThis.BroadcastChannel;

  beforeEach(() => {
    MockBroadcastChannel.resetAll();
    (globalThis as any).BroadcastChannel = MockBroadcastChannel;
    localStorage.clear();
  });

  afterEach(() => {
    (globalThis as any).BroadcastChannel = originalBC;
    localStorage.clear();
    vi.restoreAllMocks();
  });

  describe('BroadcastChannelTransport', () => {
    it('delivers messages between different transport instances on the same channel', () => {
      const transportA = new BroadcastChannelTransport();
      const transportB = new BroadcastChannelTransport();

      const received: ChannelMessageEnvelope[] = [];
      const unsubscribe = transportB.subscribe('blue', (env) => {
        received.push(env);
      });

      const envelope: ChannelMessageEnvelope = {
        sourceWindowId: 'win_1',
        channelId: 'blue',
        context: { type: 'fdc3.instrument', id: { ticker: 'NVDA' } },
        timestamp: Date.now(),
      };

      transportA.broadcast(envelope);

      expect(received).toHaveLength(1);
      expect(received[0].context.type).toBe('fdc3.instrument');
      expect((received[0].context as Fdc3InstrumentContext).id.ticker).toBe('NVDA');

      unsubscribe();
      transportA.destroy();
      transportB.destroy();
    });

    it('isolates messages across different channel IDs', () => {
      const transportA = new BroadcastChannelTransport();
      const transportB = new BroadcastChannelTransport();

      const redMessages: ChannelMessageEnvelope[] = [];
      transportB.subscribe('red', (env) => redMessages.push(env));

      transportA.broadcast({
        sourceWindowId: 'win_1',
        channelId: 'blue',
        context: { type: 'fdc3.instrument', id: { ticker: 'AAPL' } },
        timestamp: Date.now(),
      });

      expect(redMessages).toHaveLength(0);

      transportA.destroy();
      transportB.destroy();
    });
  });

  describe('LocalFdc3Channel', () => {
    it('filters out echoes matching currentWindowId', () => {
      const transport = new BroadcastChannelTransport();
      const channel = new LocalFdc3Channel('blue', 'win_self', transport);

      const received: any[] = [];
      channel.addContextListener(null, (ctx) => received.push(ctx));

      // Simulate incoming message with SAME windowId
      const echoEnvelope: ChannelMessageEnvelope = {
        sourceWindowId: 'win_self',
        channelId: 'blue',
        context: { type: 'fdc3.instrument', id: { ticker: 'ECHO' } },
        timestamp: Date.now(),
      };

      // Manually trigger transport message
      transport.broadcast(echoEnvelope);

      expect(received).toHaveLength(0);

      channel.destroy();
      transport.destroy();
    });

    it('filters context by contextType properly', () => {
      const transportA = new BroadcastChannelTransport();
      const transportB = new BroadcastChannelTransport();

      const channelA = new LocalFdc3Channel('blue', 'win_A', transportA);
      const channelB = new LocalFdc3Channel('blue', 'win_B', transportB);

      const instrumentReceived: Fdc3InstrumentContext[] = [];
      const portfolioReceived: Fdc3PortfolioContext[] = [];
      const allReceived: any[] = [];

      channelB.addContextListener<Fdc3InstrumentContext>('fdc3.instrument', (ctx) =>
        instrumentReceived.push(ctx)
      );
      channelB.addContextListener<Fdc3PortfolioContext>('fdc3.portfolio', (ctx) =>
        portfolioReceived.push(ctx)
      );
      channelB.addContextListener('*', (ctx) => allReceived.push(ctx));

      // 1. Broadcast instrument
      channelA.broadcast({
        type: 'fdc3.instrument',
        name: 'Microsoft',
        id: { ticker: 'MSFT' },
      });

      expect(instrumentReceived).toHaveLength(1);
      expect(instrumentReceived[0].id.ticker).toBe('MSFT');
      expect(portfolioReceived).toHaveLength(0);
      expect(allReceived).toHaveLength(1);

      // 2. Broadcast portfolio
      channelA.broadcast({
        type: 'fdc3.portfolio',
        name: 'Global Growth',
        id: { portfolioId: 'PORT-99' },
      });

      expect(instrumentReceived).toHaveLength(1);
      expect(portfolioReceived).toHaveLength(1);
      expect(portfolioReceived[0].id.portfolioId).toBe('PORT-99');
      expect(allReceived).toHaveLength(2);

      channelA.destroy();
      channelB.destroy();
      transportA.destroy();
      transportB.destroy();
    });

    it('hydrates lastContext from localStorage for late-joining windows', () => {
      // Simulate Monitor 1 setting context in localStorage
      const cachedEnvelope: ChannelMessageEnvelope = {
        sourceWindowId: 'win_monitor1',
        channelId: 'blue',
        context: { type: 'fdc3.instrument', id: { ticker: 'TSLA' } },
        timestamp: Date.now(),
      };
      localStorage.setItem('fdc3_last_blue', JSON.stringify(cachedEnvelope));

      // Monitor 2 opens 30 seconds later
      const transport2 = new BroadcastChannelTransport();
      const channel2 = new LocalFdc3Channel('blue', 'win_monitor2', transport2);

      // Should have hydrated context immediately
      expect(channel2.getCurrentContext()).toEqual(cachedEnvelope.context);

      // Listener subscribing should receive immediate replay
      const replayed: any[] = [];
      channel2.addContextListener('fdc3.instrument', (ctx) => replayed.push(ctx));

      expect(replayed).toHaveLength(1);
      expect(replayed[0].id.ticker).toBe('TSLA');

      channel2.destroy();
      transport2.destroy();
    });

    it('safely handles corrupted localStorage data without crashing initialization', () => {
      localStorage.setItem('fdc3_last_blue', 'INVALID_JSON_%%%');

      const transport = new BroadcastChannelTransport();
      expect(() => {
        const channel = new LocalFdc3Channel('blue', 'win_err', transport);
        expect(channel.getCurrentContext()).toBeNull();
        channel.destroy();
      }).not.toThrow();

      transport.destroy();
    });

    it('write-through persists broadcast context to localStorage', () => {
      const transport = new BroadcastChannelTransport();
      const channel = new LocalFdc3Channel('orange', 'win_writer', transport);

      channel.broadcast({
        type: 'fdc3.instrument',
        id: { ticker: 'AMZN' },
      });

      const stored = localStorage.getItem('fdc3_last_orange');
      expect(stored).not.toBeNull();
      const parsed = JSON.parse(stored!);
      expect(parsed.context.id.ticker).toBe('AMZN');

      channel.destroy();
      transport.destroy();
    });
  });

  describe('Fdc3DesktopAgent', () => {
    it('manages channel switching and channel change callbacks', () => {
      const agent = new Fdc3DesktopAgent('agent_1');

      expect(agent.getCurrentChannelId()).toBe('blue');

      const channelChanges: string[] = [];
      agent.onChannelChanged((ch) => channelChanges.push(ch));

      agent.joinUserChannel('green');
      expect(agent.getCurrentChannelId()).toBe('green');

      expect(channelChanges).toEqual(['blue', 'green']);

      agent.destroy();
    });

    it('supports dynamic transport switching to WailsRelayTransport', () => {
      const agentA = new Fdc3DesktopAgent('agent_a');
      const agentB = new Fdc3DesktopAgent('agent_b');

      const relayTransportA = new WailsRelayTransport();
      const relayTransportB = new WailsRelayTransport();

      agentA.setTransport(relayTransportA);
      agentB.setTransport(relayTransportB);

      expect(agentA.getTransport().name).toBe('WailsRelay');

      const received: any[] = [];
      agentB.addContextListener('fdc3.instrument', (ctx) => received.push(ctx));

      agentA.broadcast({
        type: 'fdc3.instrument',
        id: { ticker: 'GOOGL' },
      });

      expect(received).toHaveLength(1);
      expect(received[0].id.ticker).toBe('GOOGL');

      agentA.destroy();
      agentB.destroy();
    });

    it('delivers messages through mocked Wails Go bridge and event emission', async () => {
      // Simulate Wails v3 runtime environment
      let wailsEventHandler: ((payload: unknown) => void) | null = null;

      const mockWailsEvents = {
        On: vi.fn((eventName: string, cb: (payload: unknown) => void) => {
          if (eventName === 'fdc3_relay_message') {
            wailsEventHandler = cb;
          }
          return () => {
            wailsEventHandler = null;
          };
        }),
        Emit: vi.fn(),
      };

      const mockDeskWindowManager = {
        RelayMessage: vi.fn(async (channel: string, payloadStr: string) => {
          // Go backend relays message to all windows via Wails Event
          if (wailsEventHandler) {
            wailsEventHandler(payloadStr);
          }
        }),
      };

      (window as any).wails = { Events: mockWailsEvents };
      (window as any).go = { main: { DeskWindowManager: mockDeskWindowManager } };

      const agentA = new Fdc3DesktopAgent('agent_wails_a', new WailsRelayTransport());
      const agentB = new Fdc3DesktopAgent('agent_wails_b', new WailsRelayTransport());

      const received: any[] = [];
      agentB.addContextListener<Fdc3InstrumentContext>('fdc3.instrument', (ctx) => {
        received.push(ctx);
      });

      agentA.broadcast({
        type: 'fdc3.instrument',
        id: { ticker: 'AAPL', ISIN: 'US0378331005' },
      });

      expect(mockDeskWindowManager.RelayMessage).toHaveBeenCalledWith(
        'blue',
        expect.stringContaining('US0378331005')
      );

      expect(received).toHaveLength(1);
      expect(received[0].id.ticker).toBe('AAPL');
      expect(received[0].id.ISIN).toBe('US0378331005');

      delete (window as any).wails;
      delete (window as any).go;

      agentA.destroy();
      agentB.destroy();
    });
  });
});

