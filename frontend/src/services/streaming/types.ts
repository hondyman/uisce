/**
 * Market Data Streaming Types & Interfaces.
 * 
 * Strict Architectural Boundary:
 * High-frequency tick data, L2/L3 quotes, and execution fills MUST NEVER
 * travel over the FDC3 context bus.
 * They flow directly into memory structures outside React component state.
 */

export interface MarketTick {
  symbol: string;
  price: number;
  size: number;
  bid: number;
  ask: number;
  timestamp: number;
}

export interface QuoteRow {
  symbol: string;
  price: number;
  prevPrice: number;
  change: number;
  changePct: number;
  bid: number;
  ask: number;
  volume: number;
  high: number;
  low: number;
  lastUpdated: number;
  tickDirection: 'up' | 'down' | 'flat';
}

export type TickCallback = (tick: MarketTick) => void;

/**
 * Transport-pluggable tick stream interface.
 * Implemented by SyntheticTickSource in the spike, and WebSocketTickSource in production.
 */
export interface TickSource {
  readonly name: string;
  connect(): void;
  disconnect(): void;
  isConnected(): boolean;
  onTick(callback: TickCallback): () => void;
}

/**
 * IngestionMetrics provides operational metrics on high-throughput tick stream.
 * Note: p50/p95/p99 latency percentiles use approximate reservoir sampling
 * over a bounded 1,000-element window to preserve zero-GC performance.
 */
export interface IngestionMetrics {
  totalTicks: number;
  coalescedTicks: number;
  tableUpdates: number;
  ticksPerSecond: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  isDegraded: boolean;
}
