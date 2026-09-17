import { describe, it, expect, vi, beforeEach } from 'vitest';
import React from 'react';
import { render, fireEvent } from '@testing-library/react';
import { WorkstationCommandBar } from '../../components/docking/WorkstationCommandBar';
import { fdc3Agent } from '../../services/fdc3';

describe('WorkstationCommandBar Component', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
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
});
