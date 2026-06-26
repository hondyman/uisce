import React from 'react';
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
// Mock heavy editor imports to keep Vitest lightweight
vi.mock('@monaco-editor/react', () => ({
  default: (props: any) => <div data-testid="monaco-editor">{props.children}</div>,
}));
vi.mock('monaco-editor', () => ({}));

import ValidationRuleEditor from '@/components/validation/ValidationRuleEditor';
import { ConfirmProvider } from '@/components/ConfirmProvider';
import { SnackbarProvider } from 'notistack';

// Mock useNotification to avoid notistack provider complexity
vi.mock('@/hooks/useNotification', () => ({
  useNotification: () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn() }),
}));

// Mock apiClient used for metadata fetches
vi.mock('@/utils/apiClient', () => ({
  __esModule: true,
  default: async () => ({ ok: true, json: async () => ({ data: [] }) }),
}));

// Mock ExpressionBuilder which relies on Apollo Mutations in real app
vi.mock('@/components/ExpressionBuilder/ExpressionBuilder', () => ({
  default: (props: any) => <div data-testid="expression-builder" />,
}));

// Mock rulesApi
vi.mock('@/services/rulesApi', () => ({
  rulesApi: {
    getRules: vi.fn(),
    getRule: vi.fn(),
    overrideRule: vi.fn(),
  },
}));

import { rulesApi } from '@/services/rulesApi';

beforeEach(() => {
  // Provide tenant context required by the component
  localStorage.setItem('selected_tenant', JSON.stringify({ id: 't1', gold_copy: false }));
  localStorage.setItem('selected_datasource', JSON.stringify({ id: 'd1' }));
  vi.resetAllMocks();
});

test('shows core banner and Create Tenant Override button when rule is core', async () => {
  (rulesApi.getRules as unknown as vi.Mock).mockResolvedValue([
    { id: 'r1', name: 'CoreRule', bp_name: 'b1', step_name: 'name', condition_json: '{}' }
  ]);

  (rulesApi.getRule as unknown as vi.Mock).mockResolvedValue({ id: 'r1', name: 'CoreRule', is_core: true, can_override: true });

  render(
    <SnackbarProvider>
      <ConfirmProvider>
        <ValidationRuleEditor />
      </ConfirmProvider>
    </SnackbarProvider>
  );

  // Wait for rule row to render
  expect(await screen.findByText('CoreRule')).toBeInTheDocument();

  // Click the edit button for the rule
  const editButtons = await screen.findAllByTitle('Edit');
  fireEvent.click(editButtons[0]);

  // Expect the core banner to be visible and the override button
  expect(await screen.findByText(/core \(gold copy\)/i)).toBeInTheDocument();
  expect(await screen.findByRole('button', { name: /Create Tenant Override/i })).toBeInTheDocument();
});

test('shows override banner and View Core / Revert buttons when rule is override', async () => {
  (rulesApi.getRules as unknown as vi.Mock).mockResolvedValue([
    { id: 'r2', name: 'OverrideRule', bp_name: 'b1', step_name: 'name', condition_json: '{}' }
  ]);

  (rulesApi.getRule as unknown as vi.Mock).mockResolvedValue({ id: 'r2', name: 'OverrideRule', is_override: true, core_rule_id: 'core-123', can_delete: true });

  render(
    <ConfirmProvider>
      <ValidationRuleEditor />
    </ConfirmProvider>
  );

  expect(await screen.findByText('OverrideRule')).toBeInTheDocument();
  const editButtons = await screen.findAllByTitle('Edit');
  fireEvent.click(editButtons[0]);

  expect(await screen.findByText(/tenant override/i)).toBeInTheDocument();
  expect(await screen.findByRole('button', { name: /View Core Rule/i })).toBeInTheDocument();
  expect(await screen.findByRole('button', { name: /Revert to Core/i })).toBeInTheDocument();
});

test('creating override calls rulesApi.overrideRule and opens new rule', async () => {
  (rulesApi.getRules as unknown as vi.Mock).mockResolvedValue([
    { id: 'r1', name: 'CoreRule', bp_name: 'b1', step_name: 'name', condition_json: '{}' }
  ]);

  (rulesApi.getRule as unknown as vi.Mock)
    .mockResolvedValueOnce({ id: 'r1', name: 'CoreRule', is_core: true, can_override: true })
    .mockResolvedValueOnce({ id: 'new-override', name: 'Override (new)', is_override: true, core_rule_id: 'r1' });

  (rulesApi.overrideRule as unknown as vi.Mock).mockResolvedValue({ id: 'new-override' });

  render(
    <SnackbarProvider>
      <ConfirmProvider>
        <ValidationRuleEditor />
      </ConfirmProvider>
    </SnackbarProvider>
  );

  expect(await screen.findByText('CoreRule')).toBeInTheDocument();
  const editButtons = await screen.findAllByTitle('Edit');
  fireEvent.click(editButtons[0]);

  const overrideButton = await screen.findByRole('button', { name: /Create Tenant Override/i });
  fireEvent.click(overrideButton);

  await waitFor(() => expect(rulesApi.overrideRule).toHaveBeenCalledWith('r1', 't1', 'd1'));

  // After override, the editor should fetch the new rule (second getRule call)
  await waitFor(() => expect(rulesApi.getRule).toHaveBeenCalledWith('new-override'));
});

test('reverting override deletes override and reopens core rule', async () => {
  (rulesApi.getRules as unknown as vi.Mock).mockResolvedValue([
    { id: 'r2', name: 'OverrideRule', bp_name: 'b1', step_name: 'name', condition_json: '{}' }
  ]);

  (rulesApi.getRule as unknown as vi.Mock)
    .mockResolvedValueOnce({ id: 'r2', name: 'OverrideRule', is_override: true, core_rule_id: 'core-123', can_delete: true })
    .mockResolvedValueOnce({ id: 'core-123', name: 'CoreRule', is_core: true });

  // Stub global fetch for delete endpoint
  vi.stubGlobal('fetch', vi.fn(async (url) => ({ ok: true } as any)));

  render(
    <SnackbarProvider>
      <ConfirmProvider>
        <ValidationRuleEditor />
      </ConfirmProvider>
    </SnackbarProvider>
  );

  expect(await screen.findByText('OverrideRule')).toBeInTheDocument();
  const editButtons = await screen.findAllByTitle('Edit');
  fireEvent.click(editButtons[0]);

  const revertButton = await screen.findByRole('button', { name: /Revert to Core/i });
  fireEvent.click(revertButton);

  // Confirm dialog appears; click confirm
  const confirmBtn = await screen.findByRole('button', { name: 'Confirm' });
  fireEvent.click(confirmBtn);

  // After deletion, the editor should re-open the core rule
  await waitFor(() => expect(rulesApi.getRule).toHaveBeenCalledWith('core-123'));

  // Restore fetch
  vi.unstubAllGlobals();
});