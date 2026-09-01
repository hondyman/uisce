import { describe, it, expect, vi } from 'vitest';
import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { PageEventBusProvider, usePageEventBus } from '../../../components/pagedesigner/PageEventBusContext';
import { DynamicBOFormWidget } from '../../../components/pagedesigner/widgets/DynamicBOFormWidget';
import { PageWidgetDef } from '../../../components/pagedesigner/PageDesignerTypes';

const TestEventBusConsumer: React.FC = () => {
  const { parameters, setParameter, setParametersBatch } = usePageEventBus();
  return (
    <div>
      <span data-testid="param-account">{parameters.selected_account_id || 'none'}</span>
      <span data-testid="param-region">{parameters.filter_region || 'none'}</span>
      <button
        onClick={() => setParameter('selected_account_id', 'ACC-999')}
        data-testid="btn-set-account"
      >
        Set Account
      </button>
      <button
        onClick={() => setParametersBatch({ selected_account_id: 'ACC-BATCH', filter_region: 'APAC' })}
        data-testid="btn-set-batch"
      >
        Set Batch
      </button>
    </div>
  );
};

describe('PageEventBusContext', () => {
  it('updates parameter state across pub/sub consumers', () => {
    render(
      <PageEventBusProvider initialParams={{ selected_account_id: 'ACC-001' }}>
        <TestEventBusConsumer />
      </PageEventBusProvider>
    );

    expect(screen.getByTestId('param-account').textContent).toBe('ACC-001');

    fireEvent.click(screen.getByTestId('btn-set-account'));
    expect(screen.getByTestId('param-account').textContent).toBe('ACC-999');

    fireEvent.click(screen.getByTestId('btn-set-batch'));
    expect(screen.getByTestId('param-account').textContent).toBe('ACC-BATCH');
    expect(screen.getByTestId('param-region').textContent).toBe('APAC');
  });
});

describe('DynamicBOFormWidget', () => {
  it('hydrates and renders form fields when subscribed parameter is active', async () => {
    const mockRecord = { id: 'ACC-101', name: 'Global Alpha Fund', status: 'ACTIVE' };
    global.fetch = vi.fn().mockImplementation((url) => {
      if (url.includes('/api/v1/bo/account/records/ACC-101')) {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve(mockRecord),
        });
      }
      return Promise.resolve({ ok: false, text: () => Promise.resolve('not found') });
    });

    const widgetDef: PageWidgetDef = {
      id: 'w_form',
      type: 'BO_FORM',
      title: 'Account Detail',
      boKey: 'account',
      gridSpan: { xs: 12, md: 6, lg: 6 },
      subscribedParams: ['selected_account_id'],
      entitlements: { allowCreate: true, allowUpdate: true, allowDelete: false },
    };

    render(
      <PageEventBusProvider initialParams={{ selected_account_id: 'ACC-101' }}>
        <DynamicBOFormWidget widget={widgetDef} />
      </PageEventBusProvider>
    );

    await waitFor(() => {
      expect(screen.getByText('Account Detail')).toBeInTheDocument();
      expect(screen.getByDisplayValue('Global Alpha Fund')).toBeInTheDocument();
    });
  });
});
