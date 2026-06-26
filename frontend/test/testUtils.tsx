import React from 'react';
import { render } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ApolloClient, InMemoryCache, ApolloProvider } from '@apollo/client';
import { SnackbarProvider } from 'notistack';
import { ConfirmProvider } from '../src/components/ConfirmProvider';

type RenderOptions = {
  queryClient?: QueryClient;
  apolloClient?: ApolloClient<any>;
};

export function createQueryClient() {
  return new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
}

export function createApolloClient() {
  return new ApolloClient({ cache: new InMemoryCache() });
}

export function renderWithProviders(ui: React.ReactElement, options: RenderOptions = {}) {
  const qc = options.queryClient ?? createQueryClient();
  const ac = options.apolloClient ?? createApolloClient();

  return render(
    <ApolloProvider client={ac}>
      <QueryClientProvider client={qc}>
        <SnackbarProvider maxSnack={3}>
          <ConfirmProvider>
            {ui}
          </ConfirmProvider>
        </SnackbarProvider>
      </QueryClientProvider>
    </ApolloProvider>
  );
}

export default renderWithProviders;
