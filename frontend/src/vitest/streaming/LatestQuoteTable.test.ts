import { describe, it, expect } from 'vitest';
import { LatestQuoteTable } from '../../services/streaming/LatestQuoteTable';
import { MarketTick } from '../../services/streaming/types';

describe('LatestQuoteTable (Zero-GC State Table)', () => {
  it('initializes pre-allocated rows for instrument universe', () => {
    const table = new LatestQuoteTable(['AAPL', 'MSFT', 'GOOGL']);
    expect(table.size()).toBe(3);

    const aapl = table.getRow('AAPL');
    expect(aapl).toBeDefined();
    expect(aapl?.symbol).toBe('AAPL');
    expect(aapl?.price).toBe(100.0);
    expect(aapl?.volume).toBe(0);
    expect(aapl?.tickDirection).toBe('flat');
  });

  it('mutates existing row in-place without object reallocation', () => {
    const table = new LatestQuoteTable(['AAPL']);
    const initialRow = table.getRow('AAPL')!;

    const tick: MarketTick = {
      symbol: 'AAPL',
      price: 105.50,
      size: 200,
      bid: 105.48,
      ask: 105.52,
      timestamp: 1600000000000,
    };

    const updatedRow = table.update(tick);
    // Strict reference identity check: proves zero memory reallocation!
    expect(updatedRow).toBe(initialRow);

    expect(updatedRow.price).toBe(105.50);
    expect(updatedRow.prevPrice).toBe(100.0);
    expect(updatedRow.change).toBe(5.50);
    expect(updatedRow.volume).toBe(200);
    expect(updatedRow.tickDirection).toBe('up');
  });

  it('correctly tracks tickDirection on downtick and uptick', () => {
    const table = new LatestQuoteTable(['NVDA']);
    table.update({ symbol: 'NVDA', price: 120, size: 50, bid: 119.9, ask: 120.1, timestamp: 1 });
    expect(table.getRow('NVDA')?.tickDirection).toBe('up');

    table.update({ symbol: 'NVDA', price: 118, size: 50, bid: 117.9, ask: 118.1, timestamp: 2 });
    expect(table.getRow('NVDA')?.tickDirection).toBe('down');

    table.update({ symbol: 'NVDA', price: 118, size: 25, bid: 117.9, ask: 118.1, timestamp: 3 });
    expect(table.getRow('NVDA')?.tickDirection).toBe('flat');
  });

  it('handles unknown symbol lazily by pre-allocating slot', () => {
    const table = new LatestQuoteTable();
    expect(table.size()).toBe(0);

    table.update({ symbol: 'TSLA', price: 250, size: 100, bid: 249.9, ask: 250.1, timestamp: 10 });
    expect(table.size()).toBe(1);
    expect(table.getRow('TSLA')?.price).toBe(250);
  });
});
