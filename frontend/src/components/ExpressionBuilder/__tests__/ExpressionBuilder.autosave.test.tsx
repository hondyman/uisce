// default React import removed; test uses named imports only
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { act } from 'react';
import { vi, describe, it, expect, beforeEach, afterEach } from 'vitest';
import { gql } from '@apollo/client';
import * as ApolloClientModule from '@apollo/client';
import ExpressionBuilder from '../ExpressionBuilder';

const INSERT_DRAFT = gql`
  mutation InsertDraftValidationRule($object: catalog_validation_rules_insert_input!) {
    insert_catalog_validation_rules_one(object: $object) { 
      id
      __typename
    }
  }
`;

const _UPDATE_BY_PK = gql`
  mutation UpdateValidationRuleByPk($id: uuid!, $changes: catalog_validation_rules_set_input!) {
    update_catalog_validation_rules_by_pk(pk_columns: { id: $id }, _set: $changes) { 
      id
      __typename
    }
  }
`;

// Setup fake localStorage for tenant context
beforeEach(() => {
  // Use real timers so Apollo MockedProvider promise microtasks resolve reliably in tests.
  vi.useRealTimers();
  
  // Mock localStorage
  const store: Record<string, string> = {};
  // Seed tenant scope expected by persistNow
  store['selected_tenant'] = JSON.stringify({ id: '00000000-0000-0000-0000-000000000000', display_name: 'Test Tenant' });
  store['selected_product'] = JSON.stringify({ id: '00000000-0000-0000-0000-000000000001', alpha_product: { product_name: 'Test' } });
  store['selected_datasource'] = JSON.stringify({ id: '00000000-0000-0000-0000-000000000002', source_name: 'TestDS' });
  Storage.prototype.getItem = vi.fn((key: string) => store[key] || null);
  Storage.prototype.setItem = vi.fn((key: string, value: string) => {
    store[key] = value.toString();
  });
  Storage.prototype.removeItem = vi.fn((key: string) => {
    delete store[key];
  });
  Storage.prototype.clear = vi.fn(() => {
    Object.keys(store).forEach(key => delete store[key]);
  });
});

afterEach(() => {
  // Ensure timers are reset to real timers after each test
  vi.useRealTimers();
  vi.clearAllMocks();
});

describe('ExpressionBuilder autosave', () => {
  it('creates a draft on first autosave and calls onDraftCreated', async () => {
    const draftId = '1111-2222-3333-4444';
    const onDraftCreated = vi.fn();

    // Mock useMutation to return an insert mutation function that resolves with the draft id
    const insertMock = vi.fn().mockResolvedValue({ data: { insert_catalog_validation_rules_one: { id: draftId } } });
    const useMutationSpy = vi.spyOn(ApolloClientModule, 'useMutation')
      .mockImplementationOnce(() => [insertMock, {} as any]);

    render(
      <ExpressionBuilder 
        autosave={true} 
        debounceMs={500} 
        onDraftCreated={onDraftCreated} 
        ruleName="Test Rule"
      />
    );
    // restore spy so other tests can set their own implementations
    useMutationSpy.mockRestore();

    // Builder should render (AdvancedConditionBuilder header visible)
    await waitFor(() => {
      expect(screen.getByText(/Advanced Condition Builder/i)).toBeInTheDocument();
    });

  // Trigger a change by clicking the 'Condition' add button to add a condition
  const addConditionBtn = screen.getByRole('button', { name: /^Condition$/i });
  fireEvent.click(addConditionBtn);

    // Wait past debounce delay so the autosave scheduler runs
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 600));
    });

    // Wait for GraphQL response and callback
    await waitFor(() => {
      expect(onDraftCreated).toHaveBeenCalledWith(draftId, expect.any(String));
    }, { timeout: 2000 });
  });

  it('uses update_by_pk after draft exists', async () => {
    const draftId = '3333-4444-5555-6666';
    let insertCallCount = 0;
    const updateCallCount = 0;

    const onDraftCreated = vi.fn();

    // Mock useMutation to provide insert and update functions that increment counters
    const insertMock = vi.fn().mockResolvedValue({ data: { insert_catalog_validation_rules_one: { id: draftId } } });
    const updateMock = vi.fn().mockResolvedValue({ data: { update_catalog_validation_rules_by_pk: { id: draftId } } });
    const useMutationSpy = vi.spyOn(ApolloClientModule, 'useMutation')
      .mockImplementationOnce(() => [insertMock, {} as any])
      .mockImplementationOnce(() => [updateMock, {} as any]);

    render(
      <ExpressionBuilder 
        autosave={true} 
        debounceMs={300}
        onDraftCreated={onDraftCreated}
      />
    );
    useMutationSpy.mockRestore();

    // Wait for initial render (AdvancedConditionBuilder header visible)
    await waitFor(() => {
      expect(screen.getByText(/Advanced Condition Builder/i)).toBeInTheDocument();
    });

  // Trigger first change to create draft by adding a condition
  const addConditionBtn = screen.getByRole('button', { name: /^Condition$/i });
  fireEvent.click(addConditionBtn);

    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 400));
    });

    // Wait for draft creation
    await waitFor(() => {
      expect(insertCallCount).toBeGreaterThan(0);
    }, { timeout: 2000 });

    // Reset call count to verify subsequent saves use update
    insertCallCount = 0;

  // Trigger another change by adding another condition
  const addConditionBtn2 = screen.getByRole('button', { name: /^Condition$/i });
  fireEvent.click(addConditionBtn2);

    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 400));
    });

    // Verify update was called, not insert
    await waitFor(() => {
      expect(updateCallCount).toBeGreaterThan(0);
    }, { timeout: 2000 });
  });

  it('renders AdvancedConditionBuilder component', async () => {
    // For this test, we just need to ensure useMutation is available; provide a noop insert
  const insertMock = vi.fn().mockResolvedValue({ data: { insert_catalog_validation_rules_one: { id: 'test-id' } } });
  const useMutationSpy = vi.spyOn(ApolloClientModule, 'useMutation').mockImplementation(() => [insertMock, {} as any]);

    render(<ExpressionBuilder autosave={false} />);
    useMutationSpy.mockRestore();

    // Verify builder components render
    await waitFor(() => {
      expect(screen.getByText(/Condition/i)).toBeInTheDocument();
    });
  });

  it('flushes pending save on unmount', async () => {
    const draftId = '5555-6666-7777-8888';
    const saveSpy = vi.fn();

  const _mocks = [
      {
        request: { query: INSERT_DRAFT },
        result: () => {
          saveSpy();
          return {
            data: { 
              insert_catalog_validation_rules_one: { 
                id: draftId,
                __typename: 'catalog_validation_rules'
              } 
            }
          };
        }
      }
    ];

  const saveMock = vi.fn().mockResolvedValue({ data: { insert_catalog_validation_rules_one: { id: draftId } } });
  // Spy useMutation to return a mock that calls our saveSpy when executed
  const useMutationSpy2 = vi.spyOn(ApolloClientModule, 'useMutation').mockImplementation(() => [saveMock, {} as any]);

    const { unmount } = render(<ExpressionBuilder autosave={true} debounceMs={1000} />);
    useMutationSpy2.mockRestore();

    // Wait for initial render (AdvancedConditionBuilder header visible)
    await waitFor(() => {
      expect(screen.getByText(/Advanced Condition Builder/i)).toBeInTheDocument();
    });

  // Trigger a change by adding a condition
  const addConditionBtn = screen.getByRole('button', { name: /^Condition$/i });
  fireEvent.click(addConditionBtn);

    // Unmount without waiting for the debounce; flush pending save on unmount
    unmount();

    // Give the flush a short moment to run
    await act(async () => {
      await new Promise(resolve => setTimeout(resolve, 50));
    });

    // Give async operations time to complete
    await waitFor(() => {
      // Save should have been flushed before unmount
      expect(saveSpy).toHaveBeenCalled();
    }, { timeout: 2000 });
  });
});
