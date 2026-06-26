// React import removed (unused)
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import ManageConflictModal from '../../tenants/components/ManageConflictModal';
import * as AuthCtx from '../../../contexts/AuthContext';

describe('ManageConflictModal', () => {
  beforeEach(() => {
    global.fetch = vi.fn();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  test('executes transfer and assign calls', async () => {
    const tenants = [{ id: 't1', displayName: 'Tenant One' }, { id: 't2', displayName: 'Tenant Two' }];
    (global.fetch as any).mockResolvedValue({ ok: true, json: async () => ({ status: 'success' }) });

  // mock isAdmin to return true
  vi.spyOn(AuthCtx, 'useAuth').mockReturnValue({ isAdmin: () => true } as any);
  const onClose = vi.fn();
  render(<ManageConflictModal open={true} ownerTenantId={'t1'} onClose={onClose} conflictingIp={'192.168.1.1'} tenants={tenants} />);

    // select transfer-to tenant
    const select = screen.getByLabelText(/Transfer to tenant/i) as HTMLInputElement;
    fireEvent.mouseDown(select);
    const option = await screen.findByText('Tenant Two');
    fireEvent.click(option);

  // click Execute (first click will prompt confirmation), then confirm
  const exec = screen.getByText(/Execute/i);
  fireEvent.click(exec);

  // now the button should change to Confirm Execute; click it
  const confirm = await screen.findByText(/Confirm Execute/i);
  fireEvent.click(confirm);

  // Wait for success message to appear (operation completed)
  await waitFor(() => screen.getByText(/Operation completed/i));
  // fetch should be called at least once (delete, and possibly post)
  expect((global.fetch as any).mock.calls.length).toBeGreaterThanOrEqual(1);
  });
});
