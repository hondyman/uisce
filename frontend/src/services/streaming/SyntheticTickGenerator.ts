import { MarketTick, TickCallback, TickSource } from './types';

export interface SyntheticGeneratorOptions {
  symbols?: string[];
  targetRatePerSec?: number;
  burstDurationMs?: number;
  basePrices?: Record<string, number>;
}

export const DEFAULT_TICK_UNIVERSE = [
  'AAPL', 'MSFT', 'GOOGL', 'AMZN', 'NVDA', 'META', 'TSLA', 'BRK.B', 'UNH', 'JNJ',
  'JPM', 'V', 'PG', 'XOM', 'HD', 'CVX', 'MA', 'BAC', 'ABBV', 'PFE',
  'AVGO', 'COST', 'DIS', 'KO', 'PEP', 'TMO', 'CSCO', 'WMT', 'MRK', 'ABT',
  'ADBE', 'MCD', 'CRM', 'ACN', 'LIN', 'VZ', 'NFLX', 'NKE', 'AMD', 'INTC',
  'CMCSA', 'DHR', 'TXN', 'WFC', 'PM', 'NEE', 'BMY', 'UNP', 'HON', 'QCOM'
];

/**
 * SyntheticTickGenerator generates synthetic market ticks up to 5,000+ msg/sec
 * across a defined instrument universe.
 * 
 * Implements the TickSource interface.
 */
export class SyntheticTickGenerator implements TickSource {
  public readonly name = 'SyntheticTickSource';
  private symbols: string[];
  private prices: Map<string, number> = new Map();
  private listeners: Set<TickCallback> = new Set();
  private timerId: number | NodeJS.Timeout | null = null;
  private running: boolean = false;
  private targetRate: number;

  constructor(options: SyntheticGeneratorOptions = {}) {
    this.symbols = options.symbols || DEFAULT_TICK_UNIVERSE;
    this.targetRate = options.targetRatePerSec || 5000;

    for (const sym of this.symbols) {
      const initial = options.basePrices?.[sym] || (100 + Math.random() * 400);
      this.prices.set(sym, +initial.toFixed(2));
    }
  }

  public connect(): void {
    if (this.running) return;
    this.running = true;

    // Dispatch batches every 10ms to achieve targetRatePerSec
    const batchIntervalMs = 10;
    const ticksPerBatch = Math.max(1, Math.round((this.targetRate * batchIntervalMs) / 1000));

    const emitBatch = () => {
      if (!this.running) return;
      const now = Date.now();

      for (let i = 0; i < ticksPerBatch; i++) {
        const sym = this.symbols[Math.floor(Math.random() * this.symbols.length)];
        const curPrice = this.prices.get(sym) || 150.0;
        const delta = (Math.random() - 0.49) * 0.5;
        const newPrice = Math.max(1.0, +(curPrice + delta).toFixed(2));
        this.prices.set(sym, newPrice);

        const spread = +(0.02 + Math.random() * 0.04).toFixed(2);
        const tick: MarketTick = {
          symbol: sym,
          price: newPrice,
          size: Math.floor(Math.random() * 50) * 10 + 10,
          bid: +(newPrice - spread / 2).toFixed(2),
          ask: +(newPrice + spread / 2).toFixed(2),
          timestamp: now,
        };

        this.listeners.forEach((cb) => cb(tick));
      }

      this.timerId = setTimeout(emitBatch, batchIntervalMs);
    };

    emitBatch();
  }

  public disconnect(): void {
    this.running = false;
    if (this.timerId !== null) {
      clearTimeout(this.timerId);
      this.timerId = null;
    }
  }

  public isConnected(): boolean {
    return this.running;
  }

  public onTick(callback: TickCallback): () => void {
    this.listeners.add(callback);
    return () => {
      this.listeners.delete(callback);
    };
  }

  public setRate(ratePerSec: number): void {
    this.targetRate = ratePerSec;
  }
}
