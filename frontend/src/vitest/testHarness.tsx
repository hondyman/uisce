import { ReactNode } from 'react'
import { SnackbarProvider } from 'notistack'
import { MemoryRouter } from 'react-router-dom'
import { ConfirmProvider } from '@/components/ConfirmProvider'

import { vi } from 'vitest';

const ensureTenantScope = () => {
  try {
    if (!localStorage.getItem('selected_tenant')) {
      localStorage.setItem(
        'selected_tenant',
        JSON.stringify({ id: 't1', display_name: 'Test Tenant' })
      )
    }
    if (!localStorage.getItem('selected_product')) {
      localStorage.setItem(
        'selected_product',
        JSON.stringify({ id: 'p1', alpha_product: { product_name: 'Test Product' } })
      )
    }
    if (!localStorage.getItem('selected_datasource')) {
      localStorage.setItem(
        'selected_datasource',
        JSON.stringify({ id: 'd1', source_name: 'Test Datasource' })
      )
    }
  } catch (error) {
    // Ignore storage errors in test environment
  }
}

// Global API context mock
vi.mock('@/contexts/ApiContext', () => ({
  useApi: () => ({
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn()
  })
}))

vi.mock('@/components/validation/ConditionBuilder', () => ({
  default: () => <div data-testid="condition-builder" />
}))

// Apollo hooks mock - Apollo has been removed, these are no-ops
// Keeping the mock structure in case any test files still reference Apollo
vi.mock('@apollo/client', async () => {
  return {
    ApolloProvider: ({ children }: { children: React.ReactNode }) => children,
    useQuery: () => ({ data: null, loading: false, error: null }),
    useMutation: () => [vi.fn(), { data: null, loading: false, error: null }],
    useSubscription: () => ({ data: null, loading: false, error: null }),
    useLazyQuery: () => [vi.fn(), { data: null, loading: false, error: null }],
    gql: (strings: TemplateStringsArray) => strings[0],
    InMemoryCache: vi.fn(),
    HttpLink: vi.fn(),
  }
})

// Monaco mock (DiffEditor + Editor)
vi.mock('@monaco-editor/react', () => ({
  DiffEditor: () => <div data-testid="monaco-diff" />,
  default: () => <div data-testid="monaco-editor" />
}))

// notistack mock
vi.mock('notistack', async () => {
  const actual = await vi.importActual<any>('notistack')
  return {
    ...actual,
    useSnackbar: () => ({
      enqueueSnackbar: vi.fn()
    })
  }
})

export function TestHarness({ children }: { children: ReactNode }) {
  ensureTenantScope()
  return (
    <SnackbarProvider maxSnack={1}>
      <ConfirmProvider>
        <MemoryRouter>
          {children}
        </MemoryRouter>
      </ConfirmProvider>
    </SnackbarProvider>
  )
}
