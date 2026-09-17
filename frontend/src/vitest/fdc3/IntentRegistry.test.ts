import { describe, it, expect, vi, beforeEach } from 'vitest';
import { IntentRegistry } from '../../services/fdc3/IntentRegistry';
import { ChannelMessageEnvelope, Fdc3Context, IFdc3Transport } from '../../services/fdc3/types';
import { IntentTarget, StandardIntent } from '../../services/fdc3/intentTypes';

class MockTransport implements IFdc3Transport {
  public name = 'MockTransport';
  public messages: ChannelMessageEnvelope[] = [];
  public listeners: Map<string, Set<(env: ChannelMessageEnvelope) => void>> = new Map();

  public broadcast(envelope: ChannelMessageEnvelope): void {
    this.messages.push(envelope);
    const set = this.listeners.get(envelope.channelId);
    if (set) {
      set.forEach((cb) => cb(envelope));
    }
  }

  public subscribe(channelId: string, onMessage: (envelope: ChannelMessageEnvelope) => void): () => void {
    if (!this.listeners.has(channelId)) {
      this.listeners.set(channelId, new Set());
    }
    this.listeners.get(channelId)!.add(onMessage);
    return () => {
      this.listeners.get(channelId)?.delete(onMessage);
    };
  }

  public destroy(): void {
    this.listeners.clear();
    this.messages = [];
  }
}

describe('IntentRegistry & Router', () => {
  let transport: MockTransport;
  let registry: IntentRegistry;

  beforeEach(() => {
    transport = new MockTransport();
    registry = new IntentRegistry('win_test_1', transport);
  });

  it('rejects unknown intent names (closed vocabulary)', () => {
    expect(() => {
      registry.registerIntentHandler('CustomNonStandardIntent', 'view_1', 'Title', vi.fn());
    }).toThrow(/Unknown intent/);

    return expect(
      registry.raiseIntent('NonStandardIntent', { type: 'fdc3.instrument', id: { ticker: 'AAPL' } })
    ).rejects.toThrow(/Unknown intent/);
  });

  it('registers handler, discovers targets, and unregisters cleanly (StrictMode safe)', () => {
    const handler = vi.fn();
    const unregister = registry.registerIntentHandler('ViewInstrument', 'view_depth', 'Market Depth', handler);

    let targets = registry.findIntentTargets('ViewInstrument');
    expect(targets).toHaveLength(1);
    expect(targets[0].viewId).toBe('view_depth');

    // StrictMode unmount
    unregister();
    targets = registry.findIntentTargets('ViewInstrument');
    expect(targets).toHaveLength(0);

    // StrictMode second mount
    const unregister2 = registry.registerIntentHandler('ViewInstrument', 'view_depth', 'Market Depth', handler);
    targets = registry.findIntentTargets('ViewInstrument');
    expect(targets).toHaveLength(1);
    unregister2();
  });

  it('routes directly to single local handler without resolver modal', async () => {
    const handler = vi.fn();
    registry.registerIntentHandler('ViewChart', 'view_chart', 'Yield Curve Chart', handler);

    const context: Fdc3Context = { type: 'fdc3.instrument', id: { ticker: 'US10Y' } };
    const resolution = await registry.raiseIntent('ViewChart', context);

    expect(resolution.success).toBe(true);
    expect(resolution.target.viewId).toBe('view_chart');
    expect(handler).toHaveBeenCalledTimes(1);
    expect(handler).toHaveBeenCalledWith(context);
  });

  it('rejects when zero matching handlers exist (no silent failure)', async () => {
    await expect(
      registry.raiseIntent('ViewOrders', { type: 'fdc3.instrument', id: { ticker: 'MSFT' } })
    ).rejects.toThrow(/No active handlers found for intent: "ViewOrders"/);
  });

  it('surfaces resolver modal when multiple targets exist and routes to user selection', async () => {
    const rebalancerHandler = vi.fn();
    const scenarioHandler = vi.fn();

    registry.registerIntentHandler('ViewAnalysis', 'rebalancer', 'AI Portfolio Rebalancer', rebalancerHandler);
    registry.registerIntentHandler('ViewAnalysis', 'scenario', 'Scenario Analysis Pro', scenarioHandler);

    // Resolver prompt mock simulating trader selecting Scenario Analysis
    const promptMock = vi.fn(async (intent: StandardIntent, targets: IntentTarget[]) => {
      expect(targets).toHaveLength(2);
      return targets.find((t) => t.viewId === 'scenario')!;
    });

    registry.setResolverPrompt(promptMock);

    const context = { type: 'fdc3.portfolio', id: { portfolioId: 'PORT-1' } };
    const resolution = await registry.raiseIntent('ViewAnalysis', context);

    expect(promptMock).toHaveBeenCalledTimes(1);
    expect(resolution.success).toBe(true);
    expect(resolution.target.viewId).toBe('scenario');
    expect(scenarioHandler).toHaveBeenCalledWith(context);
    expect(rebalancerHandler).not.toHaveBeenCalled();
  });

  it('enforces intent loop guard (intent handler cannot raise another intent)', async () => {
    registry.registerIntentHandler('ViewOrders', 'blotter', 'OMS Blotter', async () => {
      // Malicious or accidental recursive intent trigger
      await registry.raiseIntent('ViewChart', { type: 'fdc3.instrument', id: { ticker: 'AAPL' } });
    });

    registry.registerIntentHandler('ViewChart', 'chart', 'Chart', vi.fn());

    await expect(
      registry.raiseIntent('ViewOrders', { type: 'fdc3.order', id: { orderId: 'ORD-1' } })
    ).rejects.toThrow(/Intent handler is currently executing. Raising another intent is forbidden/);
  });

  it('handles cross-window invocation and acknowledgment across the transport', async () => {
    const remoteRegistry = new IntentRegistry('win_remote_2', transport);
    const remoteHandler = vi.fn();
    remoteRegistry.registerIntentHandler('ViewExecution', 'exec_blotter', 'Execution Blotter', remoteHandler);

    const context = { type: 'fdc3.order', id: { orderId: 'ORD-999' } };
    const resolution = await registry.raiseIntent('ViewExecution', context);

    expect(resolution.success).toBe(true);
    expect(resolution.target.windowId).toBe('win_remote_2');
    expect(remoteHandler).toHaveBeenCalledWith(context);

    remoteRegistry.destroy();
  });

  it('detects dead-handler timeout, purges dead target, and falls back to live target', async () => {
    // Register a remote handler that never acknowledges (simulating a hung or killed window)
    const deadTarget: IntentTarget = {
      intent: 'ViewInstrument',
      viewId: 'dead_view',
      windowId: 'win_crashed',
      title: 'Crashed Market Depth',
    };

    // Fake dead window announcement
    transport.broadcast({
      sourceWindowId: 'win_crashed',
      channelId: '__fdc3_system_intents__',
      context: {
        type: 'fdc3.intent.registered',
        target: deadTarget,
        timestamp: Date.now(),
      } as any,
      timestamp: Date.now(),
    });

    // Also register a healthy local handler
    const healthyHandler = vi.fn();
    registry.registerIntentHandler('ViewInstrument', 'live_view', 'Live Security Master', healthyHandler);

    // Initial find shows both
    expect(registry.findIntentTargets('ViewInstrument')).toHaveLength(2);

    // Explicitly target the dead view
    // Should time out on ack, purge dead target, and throw or re-resolve
    await expect(
      registry.raiseIntent('ViewInstrument', { type: 'fdc3.instrument', id: { ticker: 'NVDA' } }, 'dead_view')
    ).rejects.toThrow();

    // Verify the dead target was purged from the mesh registry!
    const remaining = registry.findIntentTargets('ViewInstrument');
    expect(remaining).toHaveLength(1);
    expect(remaining[0].viewId).toBe('live_view');

    // Now raising without specific target directly routes to the remaining live handler
    const res = await registry.raiseIntent('ViewInstrument', { type: 'fdc3.instrument', id: { ticker: 'NVDA' } });
    expect(res.success).toBe(true);
    expect(res.target.viewId).toBe('live_view');
    expect(healthyHandler).toHaveBeenCalledTimes(1);
  });
});
