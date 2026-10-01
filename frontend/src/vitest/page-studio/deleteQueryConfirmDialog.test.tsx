import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import DeleteQueryConfirmDialog from '../../features/query-builder/components/DeleteQueryConfirmDialog';
import * as savedQueryApi from '../../features/query-builder/services/savedQueryApi';
import type { SavedQuery } from '../../features/query-builder/types/queryDef';

describe('DeleteQueryConfirmDialog', () => {
  const dummyQuery: SavedQuery = {
    id: 'sq-test-1',
    tenantId: 'tenant-1',
    userId: 'user-1',
    name: 'Revenue by Region',
    description: 'Quarterly breakdown',
    boId: 'bo-account',
    bindingId: 'b-1',
    relatedBoIds: [],
    chartType: 'bar',
    state: {
      dimensions: [{ termNodeId: 'region', alias: 'region' }],
      measures: [{ termNodeId: 'revenue', alias: 'revenue', agg: 'SUM' }],
      filters: [],
      parameters: [],
    },
    tags: [],
    isFavorite: false,
    visibility: 'shared',
    isCore: false,
    editable: true,
    canCustomize: false,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  };

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it('blocks deletion and displays active references when query is in use (409/inUse)', async () => {
    vi.spyOn(savedQueryApi, 'getSavedQueryUsage').mockResolvedValueOnce({
      inUse: true,
      references: [
        {
          type: 'page_published',
          id: 'pg-live-1',
          name: 'Executive Dashboard',
          location: 'app.queries[1]',
          version: 2,
        },
        {
          type: 'drill_through_target',
          sourceQueryId: 'sq-summary',
          name: 'Summary Table',
          target: 'sq-test-1',
        },
      ],
    });

    render(
      <DeleteQueryConfirmDialog
        query={dummyQuery}
        open={true}
        onClose={vi.fn()}
        onSuccess={vi.fn()}
      />
    );

    // Initial loading or immediate render of usage
    await waitFor(() => {
      expect(screen.getByText('Cannot Delete Query in Use')).toBeTruthy();
    });

    expect(screen.getByText(/Query is Referenced Across the Workspace/i)).toBeTruthy();
    expect(screen.getByText('Executive Dashboard (v2)')).toBeTruthy();

    // Delete button should be disabled
    const deleteBtn = screen.getByRole('button', { name: 'Delete Query' });
    expect(deleteBtn).toBeDisabled();
  });

  it('allows soft-delete when query is unused', async () => {
    vi.spyOn(savedQueryApi, 'getSavedQueryUsage').mockResolvedValueOnce({
      inUse: false,
      references: [],
    });
    const deleteSpy = vi.spyOn(savedQueryApi, 'deleteSavedQuery').mockResolvedValueOnce();
    const onSuccess = vi.fn();
    const onClose = vi.fn();

    render(
      <DeleteQueryConfirmDialog
        query={dummyQuery}
        open={true}
        onClose={onClose}
        onSuccess={onSuccess}
      />
    );

    await waitFor(() => {
      expect(screen.getByText('Delete "Revenue by Region"?')).toBeTruthy();
    });

    const deleteBtn = screen.getByRole('button', { name: 'Delete Query' });
    expect(deleteBtn).toBeEnabled();

    await userEvent.click(deleteBtn);
    expect(deleteSpy).toHaveBeenCalledWith('sq-test-1');
    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalled();
      expect(onClose).toHaveBeenCalled();
    });
  });
});
