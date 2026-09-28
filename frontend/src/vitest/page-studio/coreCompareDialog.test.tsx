import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { vi } from 'vitest';

vi.mock('@/api/pageStudio', () => ({
  PageStudioApi: {
    compareWithCore: vi.fn(),
    upgradeExtension: vi.fn(),
  },
}));

import { PageStudioApi } from '@/api/pageStudio';
import CoreCompareDialog from '@/pages/page-studio/CoreCompareDialog';
import type { CorePageDefinition } from '@/types/pageStudio';

const page = { id: 'core-1', name: 'Mastering console' } as CorePageDefinition;

const comparison = {
  baseVersion: 3,
  coreVersion: 5,
  upgradeAvailable: true,
  customizations: [
    {
      id: 'component:note', kind: 'component', label: 'Widget “Desk notes” (Text)', summary: 'added',
      changes: [{ path: 'components.note', op: 'add' }], conflict: false, inCore: false,
    },
    {
      id: 'tab:runs', kind: 'tab', label: 'Tab “Runs”', summary: 'changed',
      changes: [{ path: 'tabs[runs].label', op: 'change', old: 'Runs', new: 'Our runs' }], conflict: true, inCore: false,
    },
  ],
  coreUpdates: [
    {
      id: 'tab:runs', kind: 'tab', label: 'Tab “Runs”', summary: 'changed',
      changes: [{ path: 'tabs[runs].label', op: 'change', old: 'Runs', new: 'Mastering runs' }], conflict: false, inCore: false,
    },
  ],
};

describe('CoreCompareDialog', () => {
  it('upgrades keeping everything except what the tenant marks Remove', async () => {
    (PageStudioApi.compareWithCore as any).mockResolvedValue(comparison);
    (PageStudioApi.upgradeExtension as any).mockResolvedValue({ ...page, version: 5 });
    const onChanged = vi.fn();
    const onClose = vi.fn();
    render(<CoreCompareDialog page={page} onClose={onClose} onChanged={onChanged} />);

    await screen.findByText('Widget “Desk notes” (Text)');
    expect(screen.getByText(/core is now v5/)).toBeTruthy();
    expect(screen.getByText('Core also changed this')).toBeTruthy();

    // Keep the widget, take the core's tab label.
    const removeButtons = screen.getAllByRole('button', { name: 'Remove' });
    await userEvent.click(removeButtons[1]);
    await userEvent.click(screen.getByRole('button', { name: 'Upgrade to v5 and remove 1' }));

    await waitFor(() => expect(PageStudioApi.upgradeExtension).toHaveBeenCalledWith('core-1', ['tab:runs']));
    expect(onChanged).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it('without a new core version it only removes, and needs something marked', async () => {
    (PageStudioApi.compareWithCore as any).mockResolvedValue({ ...comparison, coreVersion: 3, upgradeAvailable: false, coreUpdates: [] });
    render(<CoreCompareDialog page={page} onClose={vi.fn()} onChanged={vi.fn()} />);
    const apply = await screen.findByRole('button', { name: 'Remove 0 customizations' });
    expect((apply as HTMLButtonElement).disabled).toBe(true);
  });
});
