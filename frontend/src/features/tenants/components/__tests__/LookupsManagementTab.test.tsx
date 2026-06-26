// @vitest-environment jsdom
import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import LookupsManagementTab from '../LookupsManagementTab';
import { waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { test, expect, vi } from 'vitest';

vi.mock('../../../api/lookups', () => ({
  useLookups: (tenantId?: string, q?: string, limit?: number) => ({ data: [{ id: 'l1', name: 'domains', description: 'Domains' }], isLoading: false }),
  useLookupValues: (tenantId?: string, lookupId?: string) => ({ data: [{ id: 'v1', name: 'Finance', parent_id: null }], isLoading: false }),
  createLookup: vi.fn(async () => ({ id: 'l2', name: 'new' })),
  updateLookup: vi.fn(async () => ({})),
  deleteLookup: vi.fn(async () => ({})),
  createLookupValue: vi.fn(async () => ({ id: 'v2' })),
  updateLookupValue: vi.fn(async () => ({})),
  deleteLookupValue: vi.fn(async () => ({})),
}));

test('LookupsManagementTab shows lookup rows and opens values dialog', () => {
  const qc = new QueryClient();
  render(
    <QueryClientProvider client={qc}>
      <LookupsManagementTab tenantId="t1" />
    </QueryClientProvider>
  );

  expect(screen.getByText('domains')).toBeTruthy();
  const btn = screen.getByRole('button', { name: 'View Values' });
  fireEvent.click(btn);

  expect(screen.getByRole('dialog')).toBeTruthy();
  expect(screen.getByText('Finance')).toBeTruthy();
  // New lookup button should be present
  expect(screen.getByRole('button', { name: 'New Lookup' })).toBeTruthy();
});

test('create lookup opens dialog and calls API', async () => {
  const qc = new QueryClient();
  const createLookup = (await import('../../../api/lookups')).createLookup as any;

  render(
    <QueryClientProvider client={qc}>
      <LookupsManagementTab tenantId="t1" />
    </QueryClientProvider>
  );

  fireEvent.click(screen.getByRole('button', { name: 'New Lookup' }));
  expect(screen.getByText('New Lookup')).toBeTruthy();
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'test-lookup' } });
  fireEvent.click(screen.getByRole('button', { name: 'Create' }));

  await waitFor(() => expect(createLookup).toHaveBeenCalledWith('t1', { name: 'test-lookup', description: '' }));
});

test('create value uses New Value dialog to create entry', async () => {
  const qc = new QueryClient();
  const createLookupValue = (await import('../../../api/lookups')).createLookupValue as any;
  render(
    <QueryClientProvider client={qc}>
      <LookupsManagementTab tenantId="t1" />
    </QueryClientProvider>
  );

  fireEvent.click(screen.getByRole('button', { name: 'View Values' }));
  fireEvent.click(screen.getByRole('button', { name: 'New Value' }));
  fireEvent.change(screen.getByLabelText('Value'), { target: { value: 'finance-2' } });
  fireEvent.change(screen.getByLabelText('Label'), { target: { value: 'Finance 2' } });
  fireEvent.click(screen.getByRole('button', { name: 'Create' }));

  await waitFor(() => expect(createLookupValue).toHaveBeenCalledWith('t1', 'l1', { value: 'finance-2', label: 'Finance 2', parent_id: null }));
});

test('edit lookup opens edit dialog and calls update API', async () => {
  const qc = new QueryClient();
  const updateLookup = (await import('../../../api/lookups')).updateLookup as any;
  render(
    <QueryClientProvider client={qc}>
      <LookupsManagementTab tenantId="t1" />
    </QueryClientProvider>
  );

  fireEvent.click(screen.getByRole('button', { name: 'Edit' }));
  expect(screen.getByText('Edit Lookup')).toBeTruthy();
  fireEvent.change(screen.getByLabelText('Name'), { target: { value: 'domains-updated' } });
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() => expect(updateLookup).toHaveBeenCalledWith('t1', 'l1', { name: 'domains-updated', description: 'Domains' }));
});

test('edit value opens dialog and calls update value API', async () => {
  const qc = new QueryClient();
  const updateLookupValue = (await import('../../../api/lookups')).updateLookupValue as any;
  render(
    <QueryClientProvider client={qc}>
      <LookupsManagementTab tenantId="t1" />
    </QueryClientProvider>
  );

  fireEvent.click(screen.getByRole('button', { name: 'View Values' }));
    const dialog = screen.getByRole('dialog');
    const dialogWithin = within(dialog);
    fireEvent.click(dialogWithin.getByRole('button', { name: 'Edit' }));
  expect(screen.getByText('Edit Lookup Value')).toBeTruthy();
  fireEvent.change(screen.getByLabelText('Value'), { target: { value: 'finance-updated' } });
  fireEvent.click(screen.getByRole('button', { name: 'Save' }));

  await waitFor(() => expect(updateLookupValue).toHaveBeenCalledWith('t1', 'l1', 'v1', { value: 'finance-updated', label: 'Finance', parent_id: null }));
});
