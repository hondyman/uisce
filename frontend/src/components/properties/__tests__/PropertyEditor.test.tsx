import React from 'react';
import { render, screen } from '@testing-library/react';
import PropertyEditor from '../PropertyEditor';
import { vi } from 'vitest';

// Mock JsonMonacoEditor to capture schema prop
let capturedProps: any = null;
vi.mock('../../editors/JsonMonacoEditor', () => ({ __esModule: true, default: (props: any) => {
  capturedProps = props;
  return <textarea aria-label="monaco-mock" data-testid="monaco-mock" defaultValue={props.value} />;
}}));

// Mock tenant context since PropertyEditor uses useTenant hook
vi.mock('../../../contexts/TenantContext', () => ({ useTenant: () => ({ tenant: { id: 't-1' }, datasource: { id: 'ds-1' } }) }));
vi.mock('../../../api/lookups', () => ({ useLookupValues: () => ({ data: [], isLoading: false }) }));

describe('PropertyEditor JSON schema wiring', () => {
  beforeEach(() => { capturedProps = null; });

  test('passes json schema to JsonMonacoEditor from property metadata', () => {
    const prop = { name: 'meta', label: 'Meta', data_type: 'json', input_type: 'json-editor', validation: { jsonSchema: { type: 'object', properties: { a: { type: 'string' } } } } } as any;
    render(<PropertyEditor property={prop} value={{}} onChange={() => {}} allProperties={{}} />);
    expect(capturedProps).not.toBeNull();
    expect(capturedProps.schema).toBeDefined();
    expect(capturedProps.schema.properties).toHaveProperty('a');
  });
});
