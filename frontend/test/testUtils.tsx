import React from 'react';
import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SnackbarProvider } from 'notistack';
import { ConfirmProvider } from '../src/components/ConfirmProvider';

type RenderOptions = {
  queryClient?: QueryClient;
};

export function createQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } });
}

export function renderWithProviders(ui: React.ReactElement, options: RenderOptions = {}) {
  const qc = options.queryClient ?? createQueryClient();

  return render(
    <QueryClientProvider client={qc}>
      <SnackbarProvider maxSnack={3}>
        <ConfirmProvider>
          {ui}
        </ConfirmProvider>
      </SnackbarProvider>
    </QueryClientProvider>
  );
}

export default renderWithProviders;
