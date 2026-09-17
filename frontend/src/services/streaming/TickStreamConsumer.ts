import { IngestionMetrics, MarketTick, TickSource } from './types';
import { LatestQuoteTable } from './LatestQuoteTable';
import { TickRingBuffer } from './TickRingBuffer';

export interface TickStreamConsumerOptions {
  source: TickSource;
  quoteTable: LatestQuoteTable;
  ringBuffer?: TickRingBuffer;
  /** Degraded mode per-symbol sample interval in ms (default: 50ms) */
  degradedSampleIntervalMs?: number;
}

/**
 * TickStreamConsumer routes incoming market ticks into LatestQuoteTable
 * and TickRingBuffer without triggering React state dispatch.
 * 
 * Backpressure Hysteresis & Degraded Mode Contract (Option A):
 * - Normal Mode:
 *   Every incoming tick updates the LatestQuoteTable immediately and appends to TickRingBuffer.
 * - Degraded Mode (triggered when frame render time exceeds 16.6ms for 10 consecutive frames):
 *   (a) History recording to TickRingBuffer is bypassed to save memory allocations & CPU.
 *   (b) Per-symbol sampling throttling: ticks arriving within `degradedSampleIntervalMs` (50ms)
 *       for a symbol are coalesced/discarded at the consumer boundary, reducing mutations
 *       to LatestQuoteTable and shielding the render loop under heavy backpressure.
 * - Normal Recovery:
 *   When frame render time stays under 16.6ms for 60 consecutive frames, exits degraded mode.
 */
export class TickStreamConsumer {
  private source: TickSource;
  private quoteTable: LatestQuoteTable;
  private ringBuffer?: TickRingBuffer;
  private unsubscribe?: () => void;
  private degradedSampleIntervalMs: number;

  // Latency & Throughput Metrics
  private totalTicks: number = 0;
  private coalescedTicks: number = 0;
  private tableUpdates: number = 0;
  private windowTicks: number = 0;
  private ticksPerSecond: number = 0;
  private latencies: number[] = [];
  private rateIntervalId: number | NodeJS.Timeout | null = null;

  // Per-symbol throttle timestamp map for degraded mode
  private lastSymbolUpdate: Map<string, number> = new Map();

  // Hysteresis Controller State
  private consecutiveSlowFrames: number = 0;
  private consecutiveFastFrames: number = 0;
  private isDegraded: boolean = false;
  private onDegradedChange?: (degraded: boolean) => void;

  constructor(options: TickStreamConsumerOptions) {
    this.source = options.source;
    this.quoteTable = options.quoteTable;
    this.ringBuffer = options.ringBuffer;
    this.degradedSampleIntervalMs = options.degradedSampleIntervalMs ?? 50;
  }

  public start(): void {
    if (this.unsubscribe) return;

    this.unsubscribe = this.source.onTick((tick: MarketTick) => {
      this.handleTick(tick);
    });

    if (!this.source.isConnected()) {
      this.source.connect();
    }

    // Rate calculation every 1 second
    this.rateIntervalId = setInterval(() => {
      this.ticksPerSecond = this.windowTicks;
      this.windowTicks = 0;
    }, 1000);
  }

  public stop(): void {
    if (this.unsubscribe) {
      this.unsubscribe();
      this.unsubscribe = undefined;
    }
    if (this.rateIntervalId !== null) {
      clearInterval(this.rateIntervalId);
      this.rateIntervalId = null;
    }
    this.source.disconnect();
    this.lastSymbolUpdate.clear();
  }

  private handleTick(tick: MarketTick): void {
    const now = Date.now();
    const latency = Math.max(0, now - tick.timestamp);
    this.totalTicks++;
    this.windowTicks++;

    // Approximate reservoir sampling over bounded 1,000-element window
    if (this.latencies.length < 1000) {
      this.latencies.push(latency);
    } else {
      this.latencies[Math.floor(Math.random() * 1000)] = latency;
    }

    if (this.isDegraded) {
      // Degraded mode: per-symbol sampling throttle
      const last = this.lastSymbolUpdate.get(tick.symbol) || 0;
      if (now - last < this.degradedSampleIntervalMs) {
        // Drop intermediate tick before updating the table — real backpressure reduction!
        this.coalescedTicks++;
        return;
      }
      this.lastSymbolUpdate.set(tick.symbol, now);
      this.tableUpdates++;
      this.quoteTable.update(tick);
      // Skip ring buffer history retention in degraded mode
    } else {
      // Normal mode: update quote table and append to historical ring
      this.tableUpdates++;
      this.quoteTable.update(tick);
      if (this.ringBuffer) {
        this.ringBuffer.push(tick);
      }
    }
  }

  /**
   * Reports an animation frame render duration from the canvas loop.
   * Manages hysteresis transitions:
   * - 10 consecutive frames > 16.6ms -> degraded
   * - 60 consecutive frames <= 16.6ms -> healthy
   */
  public reportFrameTime(frameDurationMs: number): void {
    const BUDGET_MS = 16.67;

    if (frameDurationMs > BUDGET_MS) {
      this.consecutiveSlowFrames++;
      this.consecutiveFastFrames = 0;

      if (!this.isDegraded && this.consecutiveSlowFrames >= 10) {
        this.isDegraded = true;
        this.onDegradedChange?.(true);
      }
    } else {
      this.consecutiveFastFrames++;
      this.consecutiveSlowFrames = 0;

      if (this.isDegraded && this.consecutiveFastFrames >= 60) {
        this.isDegraded = false;
        this.onDegradedChange?.(false);
      }
    }
  }

  public setDegradedCallback(cb: (degraded: boolean) => void): void {
    this.onDegradedChange = cb;
  }

  public getMetrics(): IngestionMetrics {
    const sorted = [...this.latencies].sort((a, b) => a - b);
    const p50 = sorted.length ? sorted[Math.floor(sorted.length * 0.50)] : 0;
    const p95 = sorted.length ? sorted[Math.floor(sorted.length * 0.95)] : 0;
    const p99 = sorted.length ? sorted[Math.floor(sorted.length * 0.99)] : 0;

    return {
      totalTicks: this.totalTicks,
      coalescedTicks: this.coalescedTicks,
      tableUpdates: this.tableUpdates,
      ticksPerSecond: this.ticksPerSecond,
      p50LatencyMs: p50,
      p95LatencyMs: p95,
      p99LatencyMs: p99,
      isDegraded: this.isDegraded,
    };
  }

  public getQuoteTable(): LatestQuoteTable {
    return this.quoteTable;
  }
}
