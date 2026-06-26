import React from 'react';
import { render } from '@testing-library/react';
import JsonMonacoEditor from '../JsonMonacoEditor';
import { vi } from 'vitest';

// Mock @monaco-editor/react to call beforeMount with a fake monaco object
vi.mock('@monaco-editor/react', () => ({
  __esModule: true,
  default: ({ beforeMount, value }: any) => {
    const setDiagnosticsOptions = vi.fn();
    // Expose the spy so tests can assert upon it
    // @ts-ignore - attach to window for test inspection
    (window as any).__monaco_setDiagnosticsOptions = setDiagnosticsOptions;
    // Call beforeMount with a fake monaco object that contains json.jsonDefaults
    if (beforeMount) beforeMount({ languages: { json: { jsonDefaults: { setDiagnosticsOptions } } } });

    return <textarea aria-label="monaco-mock" defaultValue={value} />;
  }
}));

describe('JsonMonacoEditor diagnostics', () => {
  afterEach(() => {
    delete (window as any).__monaco_setDiagnosticsOptions;
  });

  test('registers diagnostics options when schema is passed', () => {
    const testSchema = { type: 'object', properties: { a: { type: 'string' } } };
    render(<JsonMonacoEditor value="{}" onChange={() => {}} schema={testSchema} schemaUrn="inmemory://test" />);

    const spy = (window as any).__monaco_setDiagnosticsOptions;
    expect(spy).toBeDefined();
    expect(spy.mock.calls.length).toBeGreaterThan(0);
    const called = spy.mock.calls[0][0];
    expect(called).toHaveProperty('validate', true);
    expect(called.schemas[0]).toMatchObject({ uri: 'inmemory://test', schema: testSchema });
  });

  test('does not call diagnostics when schema is not passed', () => {
    render(<JsonMonacoEditor value="{}" onChange={() => {}} />);

    const spy = (window as any).__monaco_setDiagnosticsOptions;
    // spy may be undefined because beforeMount may not be called with setDiagnostics - in our mock it will not create the spy if beforeMount isn't invoked.
    if (spy) {
      // if it exists ensure it wasn't used
      expect(spy.mock.calls.length).toBe(0);
    }
  });
});
