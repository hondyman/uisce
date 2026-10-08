import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { CodeEditorWidget } from '../../pages/page-studio/app/moreWidgets';
import { AppRuntimeProvider } from '../../pages/page-studio/app/AppRuntime';
import type { PageAppModel } from '../../pages/page-studio/app/appModel';

vi.mock('@monaco-editor/react', () => ({
  default: ({ value, language, onChange }: { value?: string; language?: string; onChange?: (val: string) => void }) => (
    <div data-testid="monaco-editor-mock">
      <span data-testid="monaco-lang">{language}</span>
      <textarea
        data-testid="monaco-input"
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
      />
    </div>
  ),
}));

function renderWithProviders(ui: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        {ui}
      </MemoryRouter>
    </QueryClientProvider>
  );
}

describe('CodeEditorWidget', () => {
  it('renders Monaco editor mock with specified language and bound value', () => {
    const app: PageAppModel = {
      chrome: 'none',
      variables: [{ name: 'sqlCode', default: 'SELECT * FROM accounts;' }],
    };

    renderWithProviders(
      <AppRuntimeProvider app={app} tenantId="tenant-1">
        <CodeEditorWidget
          p={{
            variable: 'sqlCode',
            language: 'sql',
          }}
          scope={{ vars: { sqlCode: 'SELECT * FROM accounts;' } }}
        />
      </AppRuntimeProvider>
    );

    expect(screen.getByTestId('monaco-editor-mock')).toBeDefined();
    expect(screen.getByTestId('monaco-lang').textContent).toBe('sql');
    const input = screen.getByTestId('monaco-input') as HTMLTextAreaElement;
    expect(input.value).toBe('SELECT * FROM accounts;');
  });

  it('initializes from binding when variable is empty', () => {
    const app: PageAppModel = {
      chrome: 'none',
      variables: [],
    };

    renderWithProviders(
      <AppRuntimeProvider app={app} tenantId="tenant-1">
        <CodeEditorWidget
          p={{
            initFrom: '{{data.defaultCode}}',
            language: 'json',
          }}
          scope={{ data: { defaultCode: '{"key": "value"}' } }}
        />
      </AppRuntimeProvider>
    );

    expect(screen.getByTestId('monaco-lang').textContent).toBe('json');
    const input = screen.getByTestId('monaco-input') as HTMLTextAreaElement;
    expect(input.value).toBe('{"key": "value"}');
  });
});
