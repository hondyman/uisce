import { ReactNode } from 'react'
import { SnackbarProvider } from 'notistack'
import { MemoryRouter } from 'react-router-dom'
import { ConfirmProvider } from '@/components/ConfirmProvider'

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

// ExpressionBuilder mock (heavy dependency)
vi.mock('@/components/ExpressionBuilder', () => ({
  ExpressionBuilder: () => <div data-testid="expression-builder" />
}))

vi.mock('@/components/ExpressionBuilder/ExpressionBuilder', () => ({
  default: () => <div data-testid="expression-builder" />
}))

vi.mock('@/components/validation/ConditionBuilder', () => ({
  default: () => <div data-testid="condition-builder" />
}))

vi.mock('@/components/validation/RuleTemplatesSelector', () => ({
  default: () => <div data-testid="rule-templates-selector" />
}))

vi.mock('@/components/validation/LivePreview', () => ({
  default: () => <div data-testid="live-preview" />
}))

vi.mock('@/components/validation/ImpactAnalysis', () => ({
  default: () => <div data-testid="impact-analysis" />
}))

vi.mock('@/components/validation/AdvancedFieldSelector', () => ({
  default: () => <div data-testid="advanced-field-selector" />
}))

vi.mock('@/components/validation/RuleCloneAndConflict', () => ({
  default: () => <div data-testid="rule-clone-conflict" />
}))

vi.mock('@/components/validation/SampleDataGenerator', () => ({
  default: () => <div data-testid="sample-data-generator" />
}))

// Apollo hooks mock to avoid cache initialization in tests
vi.mock('@apollo/client', async () => {
  const actual = await vi.importActual<any>('@apollo/client')
  return {
    ...actual,
    useQuery: () => ({ data: null, loading: false, error: null }),
    useMutation: () => [vi.fn(), { data: null, loading: false, error: null }]
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
