import { MarketTick } from './types';

/**
 * Pre-allocated Circular Ring Buffer for raw tick history (e.g. 10,000 ticks).
 * Useful for fast time-series inspection, recent trades blurs, and tick blotters.
 */
export class TickRingBuffer {
  private readonly capacity: number;
  private readonly buffer: (MarketTick | null)[];
  private head: number = 0;
  private count: number = 0;

  constructor(capacity: number = 10000) {
    this.capacity = capacity;
    this.buffer = new Array(capacity).fill(null);
  }

  public push(tick: MarketTick): void {
    this.buffer[this.head] = tick;
    this.head = (this.head + 1) % this.capacity;
    if (this.count < this.capacity) {
      this.count++;
    }
  }

  public getRecent(limit?: number): MarketTick[] {
    const take = limit ? Math.min(limit, this.count) : this.count;
    const result: MarketTick[] = [];
    result.length = take;

    let idx = (this.head - 1 + this.capacity) % this.capacity;
    for (let i = 0; i < take; i++) {
      result[i] = this.buffer[idx]!;
      idx = (idx - 1 + this.capacity) % this.capacity;
    }
    return result;
  }

  public size(): number {
    return this.count;
  }

  public clear(): void {
    this.head = 0;
    this.count = 0;
    this.buffer.fill(null);
  }
}
