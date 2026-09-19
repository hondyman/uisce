import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import React, { act } from 'react';
import { render, fireEvent } from '@testing-library/react';
import { WorkstationCommandBar } from '../../components/docking/WorkstationCommandBar';
import { fdc3Agent } from '../../services/fdc3';
import * as instrumentsApi from '../../services/api/instrumentsApi';

describe('WorkstationCommandBar Component', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('renders correctly when open and filters commands on input', () => {
    const onClose = vi.fn();
    const { getByTestId, queryByTestId, getByPlaceholderText } = render(
      <WorkstationCommandBar isOpen={true} onClose={onClose} />
    );

    expect(getByTestId('command-bar-modal')).toBeTruthy();
    expect(getByTestId('command-item-intent-rebalancer')).toBeTruthy();

    const input = getByPlaceholderText('Type a command, intent, channel, or ticker...');
    fireEvent.change(input, { target: { value: 'Fixed Income' } });

    expect(getByTestId('command-item-intent-fixed-income')).toBeTruthy();
    expect(queryByTestId('command-item-intent-rebalancer')).toBeNull();
  });

  it('closes on Escape key', () => {
    const onClose = vi.fn();
    const { getByTestId } = render(
      <WorkstationCommandBar isOpen={true} onClose={onClose} />
    );

    const input = getByTestId('command-bar-input');
    fireEvent.keyDown(input, { key: 'Escape' });
    expect(onClose).toHaveBeenCalled();
  });

  it('executes channel switch on selection', () => {
    const joinChannelSpy = vi.spyOn(fdc3Agent, 'joinUserChannel').mockImplementation((() => null) as any);
    const onClose = vi.fn();
    const { getByTestId } = render(
      <WorkstationCommandBar isOpen={true} onClose={onClose} />
    );

    const redChannelItem = getByTestId('command-item-channel-red');
    fireEvent.click(redChannelItem);

    expect(joinChannelSpy).toHaveBeenCalledWith('red');
    expect(onClose).toHaveBeenCalled();
  });

  it('invokes intent callback on intent selection', () => {
    const onSelectIntent = vi.fn();
    const onClose = vi.fn();
    const { getByTestId } = render(
      <WorkstationCommandBar isOpen={true} onClose={onClose} onSelectIntent={onSelectIntent} />
    );

    const rebalancerIntent = getByTestId('command-item-intent-rebalancer');
    fireEvent.click(rebalancerIntent);

    expect(onSelectIntent).toHaveBeenCalledWith('ViewAnalysis', 'AAPL');
    expect(onClose).toHaveBeenCalled();
  });

  it('debounces live instrument search and broadcasts FDC3 instrument with ticker and ISIN', async () => {
    vi.useFakeTimers();

    const searchSpy = vi.spyOn(instrumentsApi, 'searchInstruments').mockResolvedValue([
      {
        symbol: 'NVDA',
        name: 'NVIDIA Corporation',
        isin: 'US67066G1040',
        asset_class: 'equity',
        internal_id: '11111111-1111-1111-1111-111111111111',
      },
    ]);
    const broadcastSpy = vi.spyOn(fdc3Agent, 'broadcast').mockImplementation((() => null) as any);
    const onClose = vi.fn();

    const { getByPlaceholderText, findByTestId } = render(
      <WorkstationCommandBar isOpen={true} onClose={onClose} />
    );

    const input = getByPlaceholderText('Type a command, intent, channel, or ticker...');
    fireEvent.change(input, { target: { value: 'NV' } });

    // Search not triggered immediately
    expect(searchSpy).not.toHaveBeenCalled();

    // Advance 100ms: still within 200ms debounce
    act(() => {
      vi.advanceTimersByTime(100);
    });
    expect(searchSpy).not.toHaveBeenCalled();

    // Type additional character: resets debounce
    fireEvent.change(input, { target: { value: 'NVDA' } });

    // Advance 210ms: debounce fires
    await act(async () => {
      vi.advanceTimersByTime(210);
    });

    expect(searchSpy).toHaveBeenCalledTimes(1);
    expect(searchSpy).toHaveBeenCalledWith('NVDA', expect.objectContaining({ limit: 10 }));

    // Switch back to real timers to await react render
    vi.useRealTimers();

    const nvdaItem = await findByTestId('command-item-inst-NVDA');
    expect(nvdaItem).toBeTruthy();
    expect(nvdaItem.textContent).toContain('US67066G1040');

    // Click result and verify FDC3 broadcast payload has ticker AND ISIN
    fireEvent.click(nvdaItem);
    expect(broadcastSpy).toHaveBeenCalledWith({
      type: 'fdc3.instrument',
      id: {
        ticker: 'NVDA',
        ISIN: 'US67066G1040',
      },
      name: 'NVIDIA Corporation',
    });
    expect(onClose).toHaveBeenCalled();
  });

  it('aborts prior in-flight request when user continues typing', async () => {
    const signals: AbortSignal[] = [];
    const searchSpy = vi.spyOn(instrumentsApi, 'searchInstruments').mockImplementation(
      async (_q, opts) => {
        if (opts?.signal) {
          signals.push(opts.signal);
        }
        return new Promise((resolve) => setTimeout(resolve, 500));
      }
    );

    vi.useFakeTimers();

    const { getByPlaceholderText } = render(
      <WorkstationCommandBar isOpen={true} onClose={vi.fn()} />
    );

    const input = getByPlaceholderText('Type a command, intent, channel, or ticker...');
    fireEvent.change(input, { target: { value: 'APP' } });

    act(() => {
      vi.advanceTimersByTime(210);
    });
    expect(searchSpy).toHaveBeenCalledTimes(1);
    expect(signals).toHaveLength(1);
    expect(signals[0].aborted).toBe(false);

    // User types additional character, triggering new debounce cycle and aborting first
    fireEvent.change(input, { target: { value: 'APPL' } });
    expect(signals[0].aborted).toBe(true);

    act(() => {
      vi.advanceTimersByTime(210);
    });

    expect(searchSpy).toHaveBeenCalledTimes(2);
    expect(signals).toHaveLength(2);
    expect(signals[1].aborted).toBe(false);
  });

  it('gracefully degrades to static fallback instruments if backend search fails', async () => {
    vi.useFakeTimers();

    vi.spyOn(instrumentsApi, 'searchInstruments').mockRejectedValue(new Error('Network offline'));
    const broadcastSpy = vi.spyOn(fdc3Agent, 'broadcast').mockImplementation((() => null) as any);
    const onClose = vi.fn();

    const { getByPlaceholderText, findByTestId } = render(
      <WorkstationCommandBar isOpen={true} onClose={onClose} />
    );

    const input = getByPlaceholderText('Type a command, intent, channel, or ticker...');
    fireEvent.change(input, { target: { value: 'AAPL' } });

    await act(async () => {
      vi.advanceTimersByTime(210);
    });

    vi.useRealTimers();

    // Fallback item for AAPL is still displayed and clickable without crash
    const aaplItem = await findByTestId('command-item-inst-AAPL');
    expect(aaplItem).toBeTruthy();

    fireEvent.click(aaplItem);
    expect(broadcastSpy).toHaveBeenCalledWith({
      type: 'fdc3.instrument',
      id: {
        ticker: 'AAPL',
        ISIN: 'US0378331005',
      },
      name: 'Apple Inc.',
    });
    expect(onClose).toHaveBeenCalled();
  });
});
