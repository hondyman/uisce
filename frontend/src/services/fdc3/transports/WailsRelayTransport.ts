import { ChannelMessageEnvelope, IFdc3Transport } from '../types';

/**
 * Fallback FDC3 Transport using Go-side IPC relay in Wails v3.
 * 
 * Activated if running under an environment where separate WebViews do not share
 * the native BroadcastChannel bus (such as certain Windows WebView2 partitions).
 * 
 * NOTE: The Go Relay intentionally broadcasts to ALL open windows including
 * the sender; the agent's `sourceWindowId` echo-filter contract drops the echo.
 * Do NOT attempt to add sender-exclusion logic in Go.
 */
export class WailsRelayTransport implements IFdc3Transport {
  public readonly name = 'WailsRelay';
  private subscribers: Map<string, Set<(envelope: ChannelMessageEnvelope) => void>> = new Map();
  private unhookWailsEvents?: () => void;
  private windowEventHandler?: (e: Event) => void;

  constructor() {
    this.setupIncomingListener();
  }

  private setupIncomingListener(): void {
    // 1. Wails v3 Event listener if available
    const wailsEvents = (window as unknown as { wails?: { Events?: { On?: (name: string, cb: (payload: unknown) => void) => () => void } } })?.wails?.Events;
    if (typeof wailsEvents?.On === 'function') {
      this.unhookWailsEvents = wailsEvents.On('fdc3_relay_message', (payload: unknown) => {
        this.handleIncomingRaw(payload);
      });
    }

    // 2. DOM CustomEvent fallback (dispatched via win.ExecJS from Go)
    this.windowEventHandler = (e: Event) => {
      const customEvt = e as CustomEvent<{ channelId: string; envelope: ChannelMessageEnvelope | string }>;
      if (customEvt.detail) {
        this.handleIncomingRaw(customEvt.detail.envelope);
      }
    };
    window.addEventListener('fdc3_relay_message', this.windowEventHandler);

    // 3. Expose global hook for direct win.ExecJS invocation
    (window as unknown as { __onFdc3RelayMessage?: (data: unknown) => void }).__onFdc3RelayMessage = (data: unknown) => {
      this.handleIncomingRaw(data);
    };
  }

  private handleIncomingRaw(payload: unknown): void {
    if (!payload) return;
    let envelope: ChannelMessageEnvelope | null = null;
    if (typeof payload === 'string') {
      try {
        envelope = JSON.parse(payload);
      } catch (err) {
        console.error('[WailsRelayTransport] Failed to parse relay payload string:', err);
        return;
      }
    } else if (typeof payload === 'object' && 'channelId' in (payload as object)) {
      envelope = payload as ChannelMessageEnvelope;
    }

    if (envelope && envelope.channelId) {
      const set = this.subscribers.get(envelope.channelId);
      if (set) {
        set.forEach((handler) => {
          try {
            handler(envelope!);
          } catch (err) {
            console.error(`[WailsRelayTransport] Handler error on channel ${envelope?.channelId}:`, err);
          }
        });
      }
    }
  }

  public broadcast(envelope: ChannelMessageEnvelope): void {
    const wailsDeskManager = (window as unknown as { go?: { main?: { DeskWindowManager?: { RelayMessage?: (channel: string, payload: string) => Promise<void> } } } })
      ?.go?.main?.DeskWindowManager;

    const payloadStr = JSON.stringify(envelope);

    if (typeof wailsDeskManager?.RelayMessage === 'function') {
      wailsDeskManager.RelayMessage(envelope.channelId, payloadStr).catch((err: unknown) => {
        console.error('[WailsRelayTransport] DeskWindowManager.RelayMessage failed:', err);
      });
    } else {
      // If Wails DeskWindowManager is not bound, fallback to local dispatch or event emit
      const wailsEmit = (window as unknown as { wails?: { Events?: { Emit?: (name: string, payload: unknown) => void } } })
        ?.wails?.Events?.Emit;
      if (typeof wailsEmit === 'function') {
        wailsEmit('fdc3_relay_message', payloadStr);
      } else {
        // In local/mock environments, dispatch DOM event
        window.dispatchEvent(new CustomEvent('fdc3_relay_message', { detail: { channelId: envelope.channelId, envelope } }));
      }
    }
  }

  public subscribe(
    channelId: string,
    onMessage: (envelope: ChannelMessageEnvelope) => void
  ): () => void {
    let set = this.subscribers.get(channelId);
    if (!set) {
      set = new Set();
      this.subscribers.set(channelId, set);
    }
    set.add(onMessage);

    return () => {
      const currentSet = this.subscribers.get(channelId);
      if (currentSet) {
        currentSet.delete(onMessage);
        if (currentSet.size === 0) {
          this.subscribers.delete(channelId);
        }
      }
    };
  }

  public destroy(): void {
    if (this.unhookWailsEvents) {
      this.unhookWailsEvents();
    }
    if (this.windowEventHandler) {
      window.removeEventListener('fdc3_relay_message', this.windowEventHandler);
    }
    delete (window as unknown as { __onFdc3RelayMessage?: unknown }).__onFdc3RelayMessage;
    this.subscribers.clear();
  }
}
