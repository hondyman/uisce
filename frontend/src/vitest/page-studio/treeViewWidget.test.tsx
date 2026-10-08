import React from 'react';
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { TreeViewWidget } from '../../pages/page-studio/app/moreWidgets';
import { AppRuntimeProvider } from '../../pages/page-studio/app/AppRuntime';
import type { PageAppModel } from '../../pages/page-studio/app/appModel';

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

describe('TreeViewWidget', () => {
  const treeData = [
    {
      id: 'root-1',
      label: 'Root Node 1',
      children: [
        { id: 'child-1-1', label: 'Child 1.1' },
        { id: 'child-1-2', label: 'Child 1.2' },
      ],
    },
    {
      id: 'root-2',
      label: 'Root Node 2',
      children: [],
    },
  ];

  it('renders root nodes and nested children', () => {
    const app: PageAppModel = {
      chrome: 'none',
      variables: [{ name: 'selectedId', default: '' }],
    };

    renderWithProviders(
      <AppRuntimeProvider app={app} tenantId="tenant-1">
        <TreeViewWidget
          p={{
            items: '{{data.tree}}',
            selectedVariable: 'selectedId',
          }}
          scope={{ data: { tree: treeData } }}
        />
      </AppRuntimeProvider>
    );

    expect(screen.getByText('Root Node 1')).toBeDefined();
    expect(screen.getByText('Child 1.1')).toBeDefined();
    expect(screen.getByText('Child 1.2')).toBeDefined();
    expect(screen.getByText('Root Node 2')).toBeDefined();
  });

  it('handles node selection and clicks without error', () => {
    const app: PageAppModel = {
      chrome: 'none',
      variables: [{ name: 'selectedId', default: '' }],
    };

    renderWithProviders(
      <AppRuntimeProvider app={app} tenantId="tenant-1">
        <TreeViewWidget
          p={{
            items: '{{data.tree}}',
            selectedVariable: 'selectedId',
          }}
          scope={{ data: { tree: treeData } }}
        />
      </AppRuntimeProvider>
    );

    const childNode = screen.getByText('Child 1.1');
    fireEvent.click(childNode);
    expect(childNode).toBeDefined();
  });

  it('renders empty message when no items', () => {
    renderWithProviders(
      <AppRuntimeProvider app={{ chrome: 'none' }} tenantId="tenant-1">
        <TreeViewWidget
          p={{
            items: '{{data.tree}}',
            emptyText: 'No hierarchy data found',
          }}
          scope={{ data: { tree: [] } }}
        />
      </AppRuntimeProvider>
    );

    expect(screen.getByText('No hierarchy data found')).toBeDefined();
  });
});
