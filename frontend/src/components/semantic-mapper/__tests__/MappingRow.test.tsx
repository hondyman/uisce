import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import { vi } from 'vitest';
import { MappingRow } from '../MappingRow';

const makeMapping = (overrides: any = {}) => ({
  id: overrides.id ?? undefined,
  database_column: {
    schema: overrides.schema ?? 'public',
    table: overrides.table ?? 'orders',
    column: overrides.column ?? 'order_id',
    node_id: overrides.node_id ?? null,
    tenant_id: overrides.tenant_id ?? '00000000-0000-0000-0000-000000000000',
    tenant_tenant_instance_id: overrides.tenant_tenant_instance_id ?? '11111111-1111-1111-1111-111111111111',
  },
  semantic_term: overrides.semantic_term ?? null,
  semantic_term_id: overrides.semantic_term_id ?? null,
  confidence: overrides.confidence ?? 0,
  selected: false,
  ignored: false,
  override: overrides.override ?? true,
  edge_exists: overrides.edge_exists ?? false,
  is_new_term: false,
  match_reason: overrides.match_reason ?? '',
});

describe('MappingRow override/create flows', () => {
  it('calls handleCreateAndSelectTerm with generated unique id and term', async () => {
  const mapping = makeMapping({ id: undefined, override: true }); // generated mapping with override enabled
    const handleCreateAndSelectTerm = vi.fn().mockResolvedValue({ term_name: 'CUSTOM_TERM', node_id: 'newid' });
    const setOverride = vi.fn();
    const setIgnored = vi.fn();
    const toggleMapping = vi.fn();

    render(
      <MappingRow
        mapping={mapping}
        idx={0}
        savedRows={new Set()}
        compactRows={false}
        keyboardExpanded={false}
        setKeyboardExpanded={() => {}}
        toggleMapping={toggleMapping}
        confirmEditing={() => {}}
        searchSemanticTerms={async () => []}
        selectSemanticTerm={() => {}}
        handleCreateAndSelectTerm={handleCreateAndSelectTerm}
        setOverride={setOverride}
        setIgnored={setIgnored}
        openReplaceConfirm={() => {}}
        openLineageModal={() => {}}
      />
    );

    // Type into the input
    const input = screen.getByPlaceholderText('Search semantic terms...');
    fireEvent.change(input, { target: { value: 'my custom' } });

    // Wait for create button to appear and click it
    await waitFor(() => expect(screen.getByRole('button', { name: /Create New/i })).toBeInTheDocument());
    const createBtn = screen.getByRole('button', { name: /Create New/i });
    fireEvent.click(createBtn);

    await waitFor(() => expect(handleCreateAndSelectTerm).toHaveBeenCalled());
    const [calledId, calledName] = handleCreateAndSelectTerm.mock.calls[0];
    expect(calledName).toMatch(/my custom/i);
    expect(calledId).toMatch(/public-orders-order_id/);
  });

  it('calls selectSemanticTerm when selecting an existing term', async () => {
  const mapping = makeMapping({ id: undefined, override: true });
    const selectSemanticTerm = vi.fn();

    render(
      <MappingRow
        mapping={mapping}
        idx={0}
        savedRows={new Set()}
        compactRows={false}
        keyboardExpanded={false}
        setKeyboardExpanded={() => {}}
        toggleMapping={() => {}}
        confirmEditing={() => {}}
        searchSemanticTerms={async () => [{ term_name: 'EXISTING', node_id: 'e1' }]}
        selectSemanticTerm={selectSemanticTerm}
        handleCreateAndSelectTerm={async () => null}
        setOverride={() => {}}
        setIgnored={() => {}}
        openReplaceConfirm={() => {}}
        openLineageModal={() => {}}
      />
    );

    const input = screen.getByPlaceholderText('Search semantic terms...');
    fireEvent.change(input, { target: { value: 'ex' } });

    await waitFor(() => expect(screen.getByText('EXISTING')).toBeInTheDocument());
    const option = screen.getByText('EXISTING');
    fireEvent.click(option);

    await waitFor(() => expect(selectSemanticTerm).toHaveBeenCalled());
    const [term, id] = selectSemanticTerm.mock.calls[0];
    expect(term.term_name).toBe('EXISTING');
    expect(id).toMatch(/public-orders-order_id/);
  });

  it('does not show Create New button when semantic term is already selected', async () => {
    const mapping = makeMapping({ 
      id: undefined, 
      override: true, 
      semantic_term_id: 'existing-term-id',
      semantic_term: 'SELECTED_TERM'
    });
    
    render(
      <MappingRow
        mapping={mapping}
        idx={0}
        savedRows={new Set()}
        compactRows={false}
        keyboardExpanded={false}
        setKeyboardExpanded={() => {}}
        toggleMapping={() => {}}
        confirmEditing={() => {}}
        searchSemanticTerms={async () => []}
        selectSemanticTerm={() => {}}
        handleCreateAndSelectTerm={async () => null}
        setOverride={() => {}}
        setIgnored={() => {}}
        openReplaceConfirm={() => {}}
        openLineageModal={() => {}}
      />
    );

    // Type into the input - even with no results, Create New button should not appear
    const input = screen.getByPlaceholderText('Search semantic terms...');
    fireEvent.change(input, { target: { value: 'some term' } });

    // Wait a bit and ensure Create New button does NOT appear
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: /Create New/i })).not.toBeInTheDocument();
    });
  });

  it('shows Create New button when user types a term and no semantic term is selected', async () => {
    const mapping = makeMapping({ 
      id: undefined, 
      override: true, 
      semantic_term: 'ORIGINAL_TERM'
    });
    
    render(
      <MappingRow
        mapping={mapping}
        idx={0}
        savedRows={new Set()}
        compactRows={false}
        keyboardExpanded={false}
        setKeyboardExpanded={() => {}}
        toggleMapping={() => {}}
        confirmEditing={() => {}}
        searchSemanticTerms={async () => [{ term_name: 'DIFFERENT_TERM', node_id: 'd1' }]}
        selectSemanticTerm={() => {}}
        handleCreateAndSelectTerm={async () => null}
        setOverride={() => {}}
        setIgnored={() => {}}
        openReplaceConfirm={() => {}}
        openLineageModal={() => {}}
      />
    );

    // Type into the input
    const input = screen.getByPlaceholderText('Search semantic terms...');
    fireEvent.change(input, { target: { value: 'country' } });

    // Wait for Create New button to appear
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /Create New/i })).toBeInTheDocument();
    });
    
    // Verify the button text includes the typed term
    expect(screen.getByRole('button', { name: /Create New: "COUNTRY"/i })).toBeInTheDocument();
  });
});
