import {
  ChannelMessageEnvelope,
  ContextHandler,
  Fdc3Context,
  IFdc3Transport,
  UserChannelId,
} from './types';
import { BroadcastChannelTransport } from './transports/BroadcastChannelTransport';
import { devWarn } from '../../utils/devLogger';
import { IntentRegistry } from './IntentRegistry';
import { IntentHandler, IntentResolution, IntentTarget, StandardIntent } from './intentTypes';

/**
 * Local channel instance representing a single FDC3 context channel (e.g. 'blue', 'red').
 * 
 * Features:
 * - Transport decoupling (BroadcastChannel or Wails Relay)
 * - Echo suppression via `sourceWindowId`
 * - Late-joiner hydration and write-through persistence via shared `localStorage`
 * - Context caching and instant replay to new subscribers
 */
export class LocalFdc3Channel {
  public readonly id: string;
  private transport: IFdc3Transport;
  private currentWindowId: string;
  private listeners: Set<(context: Fdc3Context) => void> = new Set();
  private lastContext: Fdc3Context | null = null;
  private lastTimestamp: number = 0;
  private unsubscribeTransport?: () => void;

  constructor(channelId: string, currentWindowId: string, transport: IFdc3Transport) {
    this.id = channelId;
    this.currentWindowId = currentWindowId;
    this.transport = transport;

    // 1. Hydrate lastContext from shared localStorage for late-joining windows
    this.hydrateFromLocalStorage();

    // 2. Subscribe to transport
    this.bindTransport();
  }

  private hydrateFromLocalStorage(): void {
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        const raw = window.localStorage.getItem(`fdc3_last_${this.id}`);
        if (raw) {
          const parsed = JSON.parse(raw);
          if (parsed && parsed.context && typeof parsed.timestamp === 'number') {
            this.lastContext = parsed.context;
            this.lastTimestamp = parsed.timestamp;
          }
        }
      }
    } catch (err) {
      // Corrupt or restricted storage must never crash channel init
      devWarn(`[LocalFdc3Channel ${this.id}] Failed to hydrate lastContext from localStorage:`, err);
    }
  }

  private persistToLocalStorage(envelope: ChannelMessageEnvelope): void {
    try {
      if (typeof window !== 'undefined' && window.localStorage) {
        window.localStorage.setItem(`fdc3_last_${this.id}`, JSON.stringify(envelope));
      }
    } catch (err) {
      devWarn(`[LocalFdc3Channel ${this.id}] Failed to persist to localStorage:`, err);
    }
  }

  private bindTransport(): void {
    if (this.unsubscribeTransport) {
      this.unsubscribeTransport();
    }

    this.unsubscribeTransport = this.transport.subscribe(this.id, (envelope: ChannelMessageEnvelope) => {
      // Drop echo if message was emitted by this window
      if (envelope.sourceWindowId === this.currentWindowId) {
        return;
      }

      // Live message wins over older cached/hydrated context
      if (envelope.timestamp >= this.lastTimestamp) {
        this.lastContext = envelope.context;
        this.lastTimestamp = envelope.timestamp;
        this.persistToLocalStorage(envelope);
      }

      // Notify registered listeners
      this.listeners.forEach((listener) => {
        try {
          listener(envelope.context);
        } catch (err) {
          console.error(`[LocalFdc3Channel ${this.id}] Error in context listener:`, err);
        }
      });
    });
  }

  public setTransport(transport: IFdc3Transport): void {
    this.transport = transport;
    this.bindTransport();
  }

  public broadcast(context: Fdc3Context): void {
    this.lastContext = context;
    this.lastTimestamp = Date.now();

    const envelope: ChannelMessageEnvelope = {
      sourceWindowId: this.currentWindowId,
      channelId: this.id,
      context,
      timestamp: this.lastTimestamp,
    };

    // Write-through to localStorage for late-joiner windows
    this.persistToLocalStorage(envelope);

    // Broadcast across transport to other windows
    this.transport.broadcast(envelope);

    // Notify in-window subscribers (e.g. sibling docked panels in the same window)
    this.listeners.forEach((listener) => {
      try {
        listener(context);
      } catch (err) {
        console.error(`[LocalFdc3Channel ${this.id}] Error in local context listener:`, err);
      }
    });
  }

  public getCurrentContext<T extends Fdc3Context = Fdc3Context>(): T | null {
    return this.lastContext as T | null;
  }

  public addContextListener<T extends Fdc3Context = Fdc3Context>(
    contextType: string | null,
    handler: ContextHandler<T>
  ): () => void {
    const filterHandler = (ctx: Fdc3Context) => {
      if (!contextType || contextType === '*' || ctx.type === contextType) {
        handler(ctx as T);
      }
    };

    this.listeners.add(filterHandler);

    // Replay latest context upon initial subscription
    if (
      this.lastContext &&
      (!contextType || contextType === '*' || this.lastContext.type === contextType)
    ) {
      try {
        filterHandler(this.lastContext);
      } catch (err) {
        console.error(`[LocalFdc3Channel ${this.id}] Replay error:`, err);
      }
    }

    return () => {
      this.listeners.delete(filterHandler);
    };
  }

  public destroy(): void {
    if (this.unsubscribeTransport) {
      this.unsubscribeTransport();
      delete this.unsubscribeTransport;
    }
    this.listeners.clear();
  }
}

/**
 * Main FDC3 Desktop Agent orchestrating user channels, broadcasts, and context listeners.
 */
export class Fdc3DesktopAgent {
  public readonly windowId: string;
  private transport: IFdc3Transport;
  private channels: Map<string, LocalFdc3Channel> = new Map();
  private currentChannel: LocalFdc3Channel | null = null;
  private channelChangeListeners: Set<(channelId: UserChannelId) => void> = new Set();
  private intentRegistry: IntentRegistry;

  constructor(
    windowId: string = `win_${Math.random().toString(36).substring(2, 9)}`,
    transport: IFdc3Transport = new BroadcastChannelTransport()
  ) {
    this.windowId = windowId;
    this.transport = transport;
    this.intentRegistry = new IntentRegistry(this.windowId, this.transport);

    // Default to the Blue user channel (standard trading convention)
    this.joinUserChannel('blue');
  }

  public getTransport(): IFdc3Transport {
    return this.transport;
  }

  public setTransport(transport: IFdc3Transport): void {
    this.transport = transport;
    this.channels.forEach((channel) => channel.setTransport(transport));
    this.intentRegistry.setTransport(transport);
  }

  public setIntentResolverPrompt(
    prompt: (intent: StandardIntent, targets: IntentTarget[], context?: Fdc3Context) => Promise<IntentTarget>
  ): void {
    this.intentRegistry.setResolverPrompt(prompt);
  }

  public registerIntentHandler(
    intent: string,
    viewId: string,
    title: string,
    handler: IntentHandler
  ): () => void {
    return this.intentRegistry.registerIntentHandler(intent, viewId, title, handler);
  }

  public raiseIntent(
    intent: string,
    context: Fdc3Context,
    targetViewId?: string
  ): Promise<IntentResolution> {
    return this.intentRegistry.raiseIntent(intent, context, targetViewId);
  }

  public findIntentTargets(intent: string): IntentTarget[] {
    return this.intentRegistry.findIntentTargets(intent);
  }

  public getOrCreateChannel(channelId: string): LocalFdc3Channel {
    let chan = this.channels.get(channelId);
    if (!chan) {
      chan = new LocalFdc3Channel(channelId, this.windowId, this.transport);
      this.channels.set(channelId, chan);
    }
    return chan;
  }

  public joinUserChannel(channelId: UserChannelId): LocalFdc3Channel {
    const channel = this.getOrCreateChannel(channelId);
    this.currentChannel = channel;
    this.channelChangeListeners.forEach((cb) => {
      try {
        cb(channelId);
      } catch (err) {
        console.error('[Fdc3DesktopAgent] Channel change listener error:', err);
      }
    });
    return channel;
  }

  public getCurrentChannel(): LocalFdc3Channel | null {
    return this.currentChannel;
  }

  public getCurrentChannelId(): UserChannelId {
    return (this.currentChannel?.id as UserChannelId) || 'blue';
  }

  public onChannelChanged(callback: (channelId: UserChannelId) => void): () => void {
    this.channelChangeListeners.add(callback);
    callback(this.getCurrentChannelId());
    return () => {
      this.channelChangeListeners.delete(callback);
    };
  }

  public broadcast(context: Fdc3Context): void {
    if (!this.currentChannel) {
      devWarn('[Fdc3DesktopAgent] No active FDC3 channel joined. Broadcasting aborted.');
      return;
    }
    this.currentChannel.broadcast(context);
  }

  public addContextListener<T extends Fdc3Context = Fdc3Context>(
    contextType: string | null,
    handler: ContextHandler<T>
  ): () => void {
    if (!this.currentChannel) {
      return () => {};
    }
    return this.currentChannel.addContextListener(contextType, handler);
  }

  public destroy(): void {
    this.channelChangeListeners.clear();
    this.channels.forEach((chan) => chan.destroy());
    this.channels.clear();
    this.intentRegistry.destroy();
    this.transport.destroy?.();
  }
}

// Global default agent instance
export const fdc3Agent = new Fdc3DesktopAgent();

/**
 * Deliberate automation & test hook: window.__fdc3Agent
 * 
 * To reduce attack surface and prevent accidental script tampering in institutional
 * production deployments, this global is gated STRICTLY to explicit test/dev signals.
 * 
 * NEVER gate on origin hostname (`localhost` or `127.0.0.1`)! In Wails v3 desktop,
 * production WebViews run on custom schemes (`wails://localhost` on macOS,
 * `http://wails.localhost` on Windows), which matches `localhost` hostnames and would
 * unintentionally expose this automation hook in production desktop binaries.
 */
export function isFdc3AutomationHookAllowed(
  win: Window = window,
  env: { DEV?: boolean; MODE?: string } = import.meta.env
): boolean {
  try {
    const isDev = Boolean(env?.DEV);
    const isTestMode = Boolean(env?.MODE === 'test');
    const isExplicitFlag = Boolean((win as any).__ENABLE_FDC3_AUTOMATION_HOOK__);
    const search = win.location?.search || '';
    const hasTestQuery = search.includes('verify=1') || search.includes('test=1');

    return isDev || isTestMode || isExplicitFlag || hasTestQuery;
  } catch {
    return false;
  }
}

export function initFdc3AutomationHook(win: Window = window): void {
  if (typeof win !== 'undefined' && isFdc3AutomationHookAllowed(win)) {
    (win as any).__fdc3Agent = fdc3Agent;
  }
}

if (typeof window !== 'undefined') {
  (window as any).__initFdc3AutomationHook = () => initFdc3AutomationHook(window);
  initFdc3AutomationHook(window);
}

