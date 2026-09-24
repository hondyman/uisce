import { ChannelMessageEnvelope, IFdc3Transport } from '../types';

/**
 * Primary FDC3 Transport using the browser's native BroadcastChannel API.
 * 
 * Delivers sub-millisecond, zero-broker messaging across browser tabs and
 * separate Wails WebviewWindow instances sharing the same origin partition.
 */
export class BroadcastChannelTransport implements IFdc3Transport {
  public readonly name = 'BroadcastChannel';
  private channels: Map<string, BroadcastChannel> = new Map();
  private subscribers: Map<string, Set<(envelope: ChannelMessageEnvelope) => void>> = new Map();

  private getOrCreateChannel(channelId: string): BroadcastChannel {
    let bc = this.channels.get(channelId);
    if (!bc) {
      bc = new BroadcastChannel(`fdc3_channel_${channelId}`);
      bc.onmessage = (event: MessageEvent<ChannelMessageEnvelope>) => {
        const envelope = event.data;
        if (!envelope || !envelope.channelId) return;
        const set = this.subscribers.get(envelope.channelId);
        if (set) {
          set.forEach((handler) => {
            try {
              handler(envelope);
            } catch (err) {
              console.error(`[FDC3 BroadcastChannelTransport] Handler error on channel ${envelope.channelId}:`, err);
            }
          });
        }
      };
      this.channels.set(channelId, bc);
    }
    return bc;
  }

  public broadcast(envelope: ChannelMessageEnvelope): void {
    const bc = this.getOrCreateChannel(envelope.channelId);
    bc.postMessage(envelope);
  }

  public subscribe(
    channelId: string,
    onMessage: (envelope: ChannelMessageEnvelope) => void
  ): () => void {
    this.getOrCreateChannel(channelId); // ensure initialized

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
          const bc = this.channels.get(channelId);
          if (bc) {
            bc.close();
            this.channels.delete(channelId);
          }
        }
      }
    };
  }

  public destroy(): void {
    this.subscribers.clear();
    this.channels.forEach((bc) => bc.close());
    this.channels.clear();
  }
}
