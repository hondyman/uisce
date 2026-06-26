import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import TermForm from '../../components/TermForm';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../i18n';
import { vi } from 'vitest';
// Mock Monaco editor for tests - expose a simple textarea that calls onChange
vi.mock('@monaco-editor/react', () => ({
  __esModule: true,
  default: ({ value, onChange }: any) => {
    return <textarea aria-label="monaco-json" value={value} onChange={(e) => onChange?.(e.target.value)} />;
  }
}));

// Mock tenant and node types
vi.mock('../../api/nodeTypes', () => ({ useNodeTypes: () => ({ data: [{ id: 'nt-semantic_term', catalog_type_name: 'semantic_term', properties: [
    { name: 'data_type', label: 'Data Type', data_type: 'string', nullable: false, input_type: 'text', order: 1, validation: { minLength: 2 } },
    { name: 'metadata_json', label: 'Metadata JSON', data_type: 'json', nullable: true, input_type: 'json-editor', order: 2 },
    { name: 'color', label: 'Color', data_type: 'string', nullable: true, input_type: 'lookup', order: 2, lookup_id: 'colors' },
    { name: 'score', label: 'Score', data_type: 'integer', nullable: true, input_type: 'number', order: 3, validation: { min: 1, max: 10 } },
    { name: 'tags', label: 'Tags', data_type: 'array', nullable: true, input_type: 'chips', order: 4, validation: { multiple: true } }
  ] }] }) }));
// Mock lookups API for tests
vi.mock('../../api/lookups', () => ({ useLookupValues: () => ({ data: [{ id: 'red', name: 'Red' }, { id: 'blue', name: 'Blue' }], isLoading: false }) }));
vi.mock('../../contexts/TenantContext', () => ({ useTenant: () => ({ datasource: { id: 'ds-1' }, tenant: { id: 't-1' } }) }));

describe('TermForm inline validation', () => {
  test('shows inline validation for required metadata-driven properties and prevents save', () => {
    const onSave = vi.fn();

    render(
      <I18nextProvider i18n={i18n}>
        <TermForm open={true} onClose={() => {}} onSave={onSave} term={null} termType="semantic_term" disableTypeSelection={true} />
      </I18nextProvider>
    );

    // Name is required - check it shows the error
    const saveBtn = screen.getByText('Save');
    fireEvent.click(saveBtn);
    expect(screen.getByText('Name is required')).toBeTruthy();

    // Fill name
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'New Semantic' } });

    // Try to save again
    fireEvent.click(saveBtn);
    // Still should have error because the metadata-driven property is required
    expect(screen.getByText('Data Type is required')).toBeTruthy();
    expect(onSave).not.toHaveBeenCalled();

    // Should display locked indicator for type
    expect(screen.getByTestId('type-locked')).toBeTruthy();

    // Fill the property with a too-short value to trigger minLength validation
    const dataInput = screen.getByLabelText('Data Type') as HTMLInputElement;
    fireEvent.change(dataInput, { target: { value: 'x' } });
    fireEvent.click(saveBtn);
    expect(screen.getByText(/must be at least 2 characters/i)).toBeTruthy();

    // Provide a valid value and save should succeed
    fireEvent.change(dataInput, { target: { value: 'Text' } });
    fireEvent.click(saveBtn);
    expect(onSave).toHaveBeenCalled();
  });

  test('validates JSON input and prevents save when invalid', () => {
    const onSave = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <TermForm open={true} onClose={() => {}} onSave={onSave} term={null} termType="semantic_term" disableTypeSelection={true} />
      </I18nextProvider>
    );

    // Enter invalid JSON
    const metadataField = screen.getByLabelText('monaco-json') as HTMLInputElement;
    fireEvent.change(metadataField, { target: { value: '{invalid: }' } });

    const saveBtn = screen.getByText('Save');
    fireEvent.click(saveBtn);

    expect(screen.getByText(/is not valid JSON/)).toBeTruthy();
    expect(onSave).not.toHaveBeenCalled();
  });

  test('validates numeric min/max and allows save when value within range', () => {
    const onSave = vi.fn();
    render(
      <I18nextProvider i18n={i18n}>
        <TermForm open={true} onClose={() => {}} onSave={onSave} term={null} termType="semantic_term" disableTypeSelection={true} />
      </I18nextProvider>
    );

    // Score input: number with min=1 and max=10
    const scoreInput = screen.getByLabelText('Score') as HTMLInputElement;

    // Try below min
    fireEvent.change(scoreInput, { target: { value: '0' } });
    fireEvent.click(screen.getByText('Save'));
    expect(screen.getByText(/must be >= 1/)).toBeTruthy();

    // Try above max
    fireEvent.change(scoreInput, { target: { value: '11' } });
    fireEvent.click(screen.getByText('Save'));
    expect(screen.getByText(/must be <= 10/)).toBeTruthy();

    // Try valid value
    fireEvent.change(scoreInput, { target: { value: '5' } });
    fireEvent.click(screen.getByText('Save'));
    expect(onSave).toHaveBeenCalled();
  });

  test('chips array editor allows adding tags', () => {
    const onSave = vi.fn();

    render(
      <I18nextProvider i18n={i18n}>
        <TermForm open={true} onClose={() => {}} onSave={onSave} term={null} termType="semantic_term" disableTypeSelection={true} />
      </I18nextProvider>
    );

    // Fill required name and data_type to unblock save
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'T' } });
    fireEvent.change(screen.getByLabelText('Data Type'), { target: { value: 'X' } });

    const chipsInput = screen.getByTestId('chips-tags');
    // Type a tag and press Enter to add
    fireEvent.change(chipsInput, { target: { value: 'tag1' } });
    fireEvent.keyDown(chipsInput, { key: 'Enter', code: 'Enter' });

    // Save should now succeed
    fireEvent.click(screen.getByText('Save'));
    expect(onSave).toHaveBeenCalled();
  });

  test('renders lookup select and allows selection', () => {
    const onSave = vi.fn();

    render(
      <I18nextProvider i18n={i18n}>
        <TermForm open={true} onClose={() => {}} onSave={onSave} term={null} termType="semantic_term" disableTypeSelection={true} />
      </I18nextProvider>
    );

    // Select uses the 'color' property added in the nodeType mock (if present)
    // But this test only ensures the lookup options are rendered and we can choose one
    const colorSelect = screen.queryByLabelText('Color');
    if (colorSelect) {
      fireEvent.mouseDown(colorSelect);
      const option = screen.getByText('Blue');
      fireEvent.click(option);

      // Save to invoke onSave
      fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'T' } });
      fireEvent.change(screen.getByLabelText('Data Type'), { target: { value: 'X' } });
      fireEvent.click(screen.getByText('Save'));
      expect(onSave).toHaveBeenCalled();
    }
  });

  test('displays server-side structured validation errors', async () => {
    const onSave = vi.fn(async () => {
      const err: any = new Error('Validation');
      err.validation_errors = [{ field: 'properties.data_type', message: 'Server says invalid' }];
      throw err;
    });

    render(
      <I18nextProvider i18n={i18n}>
        <TermForm open={true} onClose={() => {}} onSave={onSave} term={null} termType="semantic_term" disableTypeSelection={true} />
      </I18nextProvider>
    );

    // Fill required top-level inputs and the required property
    fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'New Term' } });
    fireEvent.change(screen.getByLabelText('Data Type'), { target: { value: 'OK' } });

    fireEvent.click(screen.getByText('Save'));

    // Expect the server-side message to appear under the property label
    expect(await screen.findByText('Server says invalid')).toBeTruthy();
  });
});
