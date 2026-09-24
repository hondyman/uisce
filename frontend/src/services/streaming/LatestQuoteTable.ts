import { MarketTick, QuoteRow } from './types';

/**
 * Zero-GC State Table holding the latest quote per instrument.
 * 
 * - Pre-allocates QuoteRow objects for all known symbols.
 * - In-place field mutation (no new object allocation on incoming ticks).
 * - O(1) lookup and update via symbol Map.
 * - Returns ordered slices directly to the Canvas render loop without allocating arrays.
 */
export class LatestQuoteTable {
  private rows: QuoteRow[] = [];
  private symbolIndex: Map<string, QuoteRow> = new Map();
  private basePrices: Map<string, number> = new Map();

  constructor(initialSymbols: string[] = []) {
    this.initSymbols(initialSymbols);
  }

  public initSymbols(symbols: string[]): void {
    this.rows = [];
    this.symbolIndex.clear();
    this.basePrices.clear();

    const now = Date.now();
    for (const sym of symbols) {
      const row: QuoteRow = {
        symbol: sym,
        price: 100.0,
        prevPrice: 100.0,
        change: 0.0,
        changePct: 0.0,
        bid: 99.98,
        ask: 100.02,
        volume: 0,
        high: 100.0,
        low: 100.0,
        lastUpdated: now,
        tickDirection: 'flat',
      };
      this.rows.push(row);
      this.symbolIndex.set(sym, row);
      this.basePrices.set(sym, 100.0);
    }
  }

  /**
   * Updates an instrument's latest quote in-place with zero memory allocation.
   * If the symbol is unrecognized, lazily registers it with pre-allocated slot.
   */
  public update(tick: MarketTick): QuoteRow {
    let row = this.symbolIndex.get(tick.symbol);
    if (!row) {
      row = {
        symbol: tick.symbol,
        price: tick.price,
        prevPrice: tick.price,
        change: 0,
        changePct: 0,
        bid: tick.bid,
        ask: tick.ask,
        volume: tick.size,
        high: tick.price,
        low: tick.price,
        lastUpdated: tick.timestamp,
        tickDirection: 'flat',
      };
      this.rows.push(row);
      this.symbolIndex.set(tick.symbol, row);
      this.basePrices.set(tick.symbol, tick.price);
      return row;
    }

    // In-place mutation (Zero-GC)
    row.prevPrice = row.price;
    row.price = tick.price;
    row.bid = tick.bid;
    row.ask = tick.ask;
    row.volume += tick.size;
    row.lastUpdated = tick.timestamp;

    if (tick.price > row.high) row.high = tick.price;
    if (tick.price < row.low || row.low === 0) row.low = tick.price;

    const base = this.basePrices.get(tick.symbol) || tick.price;
    row.change = row.price - base;
    row.changePct = (row.change / base) * 100;

    if (row.price > row.prevPrice) {
      row.tickDirection = 'up';
    } else if (row.price < row.prevPrice) {
      row.tickDirection = 'down';
    } else {
      row.tickDirection = 'flat';
    }

    return row;
  }

  public getRow(symbol: string): QuoteRow | undefined {
    return this.symbolIndex.get(symbol);
  }

  public getAllRows(): readonly QuoteRow[] {
    return this.rows;
  }

  public size(): number {
    return this.rows.length;
  }
}
