import { describe, it, expect } from 'vitest';
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { PageEventBusProvider, usePageEventBus } from '../../../components/pagedesigner/PageEventBusContext';

const PublisherComponent = () => {
  const { setParameter } = usePageEventBus();
  return (
    <button onClick={() => setParameter('selected_region', 'EMEA')}>
      Select EMEA
    </button>
  );
};

const SubscriberComponent = () => {
  const { subscribeToParameter } = usePageEventBus();
  const region = subscribeToParameter('selected_region') || 'ALL';
  return <div data-testid="region-display">Active Region: {region}</div>;
};

describe('PageEventBus Reactive Coordination', () => {
  it('propagates published channel updates to downstream subscribers without re-render loops', () => {
    render(
      <PageEventBusProvider initialParams={{ selected_region: 'ALL' }}>
        <SubscriberComponent />
        <PublisherComponent />
      </PageEventBusProvider>
    );

    expect(screen.getByTestId('region-display').textContent).toBe('Active Region: ALL');

    // Fire Click Event
    fireEvent.click(screen.getByText('Select EMEA'));

    expect(screen.getByTestId('region-display').textContent).toBe('Active Region: EMEA');
  });
});
