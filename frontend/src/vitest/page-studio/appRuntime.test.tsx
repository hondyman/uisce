import React from 'react';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { AppRuntimeProvider, useAppRuntime } from '../../pages/page-studio/app/AppRuntime';
import type { PageAppModel } from '../../pages/page-studio/app/appModel';

vi.mock('../../features/query-builder/services/savedQueryApi', () => ({
  runSavedQuery: vi.fn(),
}));

import { runSavedQuery } from '../../features/query-builder/services/savedQueryApi';

function TestRuntimeConsumer() {
  const { scope, setVariable } = useAppRuntime();
  const queryData = (scope.queries as any)?.accountQuery?.data;

  return (
    <div>
      <div data-testid="var-value">
        {(scope.vars as any)?.selectedAccountId}
      </div>
      <div data-testid="query-result">
        {queryData ? JSON.stringify(queryData) : 'no-data'}
      </div>
      <button
        onClick={() => setVariable('selectedAccountId', 'ACC-2')}
      >
        Switch Account
      </button>
    </div>
  );
}

describe('AppRuntime - Variable Refresh Semantics', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (runSavedQuery as any).mockImplementation((id: string, params: Record<string, string>) => {
      return Promise.resolve({
        columns: [{ name: 'account_id', type: 'string' }],
        rows: [{ account_id: params?.accountId || 'default' }],
        rowCount: 1,
        chartType: 'bar',
        name: 'Account Query',
      });
    });
  });

  it('re-resolves bound params and refetches when a pageVar changes', async () => {
    const app: PageAppModel = {
      variables: [{ name: 'selectedAccountId', default: 'ACC-1' }],
      queries: [
        {
          id: 'accountQuery',
          kind: 'savedQuery',
          operation: '',
          savedQueryId: 'sq-acc-1',
          paramBindings: {
            accountId: { mode: 'pageVar', varName: 'selectedAccountId' },
          },
        },
      ],
    };

    const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });

    render(
      <QueryClientProvider client={qc}>
        <MemoryRouter>
          <AppRuntimeProvider app={app} mode="preview">
            <TestRuntimeConsumer />
          </AppRuntimeProvider>
        </MemoryRouter>
      </QueryClientProvider>
    );

    // Initial render executes with ACC-1
    await waitFor(() => {
      expect(runSavedQuery).toHaveBeenCalledWith(
        'sq-acc-1',
        expect.objectContaining({ accountId: 'ACC-1' })
      );
    });
    expect(screen.getByTestId('var-value').textContent).toBe('ACC-1');

    // Mutate variable to ACC-2
    await userEvent.click(screen.getByRole('button', { name: 'Switch Account' }));

    // Expect variable updated and saved query refetched with ACC-2
    await waitFor(() => {
      expect(screen.getByTestId('var-value').textContent).toBe('ACC-2');
      expect(runSavedQuery).toHaveBeenCalledWith(
        'sq-acc-1',
        expect.objectContaining({ accountId: 'ACC-2' })
      );
    });
  });
});
