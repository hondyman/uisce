import { describe, it, expect, vi, beforeEach } from 'vitest';
import React, { act } from 'react';
import { render } from '@testing-library/react';
import { HighDensityCanvasGrid } from '../../components/docking/HighDensityCanvasGrid';
import { TickSource, MarketTick, TickCallback } from '../../services/streaming/types';

class MockTickSource implements TickSource {
  public readonly name = 'MockTickSource';
  public isConnectedState = false;
  public callback?: TickCallback;

  public connect() {
    this.isConnectedState = true;
  }
  public disconnect() {
    this.isConnectedState = false;
  }
  public isConnected() {
    return this.isConnectedState;
  }
  public onTick(cb: TickCallback) {
    this.callback = cb;
    return () => {
      this.callback = undefined;
    };
  }
  public emit(tick: MarketTick) {
    this.callback?.(tick);
  }
}

describe('HighDensityCanvasGrid Component', () => {
  let mockSource: MockTickSource;

  beforeEach(() => {
    mockSource = new MockTickSource();
  });

  it('mounts canvas without error and registers 0 React re-renders during high-volume ticks', () => {
    const { getByTestId, container } = render(
      <HighDensityCanvasGrid
        customSource={mockSource}
        universe={['AAPL', 'MSFT', 'NVDA']}
      />
    );

    const renderCounter = getByTestId('render-counter');
    const initialRenders = parseInt(renderCounter.textContent || '0', 10);
    expect(initialRenders).toBeGreaterThanOrEqual(1);

    // Simulate 500 ticks arriving rapidly
    act(() => {
      for (let i = 0; i < 500; i++) {
        mockSource.emit({
          symbol: 'AAPL',
          price: 180 + (i % 10),
          size: 100,
          bid: 179.9,
          ask: 180.1,
          timestamp: Date.now(),
        });
      }
    });

    // Verify React render counter did NOT increase from incoming ticks
    const postTickRenders = parseInt(getByTestId('render-counter').textContent || '0', 10);
    expect(postTickRenders).toBe(initialRenders);

    // Verify canvas element is rendered in DOM
    const canvas = container.querySelector('canvas');
    expect(canvas).toBeTruthy();
  });
});
