import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { TickStreamConsumer } from '../../services/streaming/TickStreamConsumer';
import { LatestQuoteTable } from '../../services/streaming/LatestQuoteTable';
import { TickRingBuffer } from '../../services/streaming/TickRingBuffer';
import { TickSource, MarketTick, TickCallback } from '../../services/streaming/types';

class MockTickSource implements TickSource {
  public readonly name = 'MockTickSource';
  public connected = false;
  public listeners: Set<TickCallback> = new Set();

  public connect(): void {
    this.connected = true;
  }
  public disconnect(): void {
    this.connected = false;
  }
  public isConnected(): boolean {
    return this.connected;
  }
  public onTick(cb: TickCallback): () => void {
    this.listeners.add(cb);
    return () => this.listeners.delete(cb);
  }
  public emit(tick: MarketTick): void {
    this.listeners.forEach((cb) => cb(tick));
  }
}

describe('TickStreamConsumer & Hysteresis Backpressure', () => {
  let table: LatestQuoteTable;
  let ring: TickRingBuffer;
  let source: MockTickSource;
  let consumer: TickStreamConsumer;

  beforeEach(() => {
    table = new LatestQuoteTable(['AAPL', 'MSFT']);
    ring = new TickRingBuffer(100);
    source = new MockTickSource();
    consumer = new TickStreamConsumer({
      source,
      quoteTable: table,
      ringBuffer: ring,
    });
  });

  afterEach(() => {
    consumer.stop();
  });

  it('ingests ticks and updates quote table without React state', () => {
    consumer.start();

    const tick: MarketTick = {
      symbol: 'AAPL',
      price: 180.25,
      size: 100,
      bid: 180.20,
      ask: 180.30,
      timestamp: Date.now(),
    };

    source.emit(tick);

    expect(table.getRow('AAPL')?.price).toBe(180.25);
    expect(ring.size()).toBe(1);

    const metrics = consumer.getMetrics();
    expect(metrics.totalTicks).toBe(1);
    expect(metrics.isDegraded).toBe(false);
  });

  it('transitions to degraded coalescing mode after 10 consecutive slow frames (>16.6ms)', () => {
    consumer.start();
    const degradedSpy = vi.fn();
    consumer.setDegradedCallback(degradedSpy);

    // 9 slow frames -> still normal
    for (let i = 0; i < 9; i++) {
      consumer.reportFrameTime(25.0);
    }
    expect(consumer.getMetrics().isDegraded).toBe(false);
    expect(degradedSpy).not.toHaveBeenCalled();

    // 10th slow frame -> enters degraded
    consumer.reportFrameTime(22.4);
    expect(consumer.getMetrics().isDegraded).toBe(true);
    expect(degradedSpy).toHaveBeenCalledWith(true);

    // In degraded mode: first tick updates table, immediate subsequent ticks for same symbol are dropped
    const tick1: MarketTick = {
      symbol: 'MSFT',
      price: 420.0,
      size: 50,
      bid: 419.9,
      ask: 420.1,
      timestamp: Date.now(),
    };
    source.emit(tick1);

    expect(consumer.getMetrics().tableUpdates).toBe(1);
    expect(consumer.getMetrics().coalescedTicks).toBe(0);
    expect(table.getRow('MSFT')?.price).toBe(420.0);
    expect(ring.size()).toBe(0); // Not pushed to ring buffer in degraded mode!

    // Rapid successive tick within degradedSampleIntervalMs (50ms) for MSFT
    const tick2: MarketTick = {
      symbol: 'MSFT',
      price: 421.5,
      size: 100,
      bid: 421.4,
      ask: 421.6,
      timestamp: Date.now(),
    };
    source.emit(tick2);

    // Assert real backpressure reduction: table update was skipped and coalescedTicks incremented!
    expect(consumer.getMetrics().tableUpdates).toBe(1); // Unchanged!
    expect(consumer.getMetrics().coalescedTicks).toBe(1);
    expect(table.getRow('MSFT')?.price).toBe(420.0); // Table was shielded from rapid mutation!
    expect(ring.size()).toBe(0);
  });

  it('recovers from degraded mode only after 60 consecutive fast frames (<=16.6ms)', () => {
    consumer.start();
    // Force into degraded mode
    for (let i = 0; i < 10; i++) {
      consumer.reportFrameTime(20.0);
    }
    expect(consumer.getMetrics().isDegraded).toBe(true);

    const degradedSpy = vi.fn();
    consumer.setDegradedCallback(degradedSpy);

    // 59 fast frames -> still degraded
    for (let i = 0; i < 59; i++) {
      consumer.reportFrameTime(14.0);
    }
    expect(consumer.getMetrics().isDegraded).toBe(true);

    // 1 slow frame interrupts recovery streak
    consumer.reportFrameTime(18.0);

    // Another 59 fast frames
    for (let i = 0; i < 59; i++) {
      consumer.reportFrameTime(12.0);
    }
    expect(consumer.getMetrics().isDegraded).toBe(true);

    // 60th consecutive fast frame recovers
    consumer.reportFrameTime(13.0);
    expect(consumer.getMetrics().isDegraded).toBe(false);
    expect(degradedSpy).toHaveBeenCalledWith(false);
  });
});
