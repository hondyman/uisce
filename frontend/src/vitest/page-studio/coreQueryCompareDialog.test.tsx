import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import CoreQueryCompareDialog from '../../features/query-builder/components/CoreQueryCompareDialog';
import type { SavedQuery } from '../../features/query-builder/types/queryDef';
import type { QueryComparisonResult } from '../../features/query-builder/services/savedQueryApi';

vi.mock('../../features/query-builder/services/savedQueryApi', () => ({
  compareSavedQuery: vi.fn(),
  upgradeSavedQuery: vi.fn(),
  revertSavedQuery: vi.fn(),
}));

import {
  compareSavedQuery,
  upgradeSavedQuery,
  revertSavedQuery,
} from '../../features/query-builder/services/savedQueryApi';

const query: SavedQuery = {
  id: 'sq-aum',
  name: 'Institutional AUM',
  isCore: true,
  coreStatus: 'upgrade_available',
  boId: 'bo-1',
  chartType: 'bar',
  state: { dimensions: [], measures: [], filters: [], parameters: [] },
};

const comparison: QueryComparisonResult = {
  baseVersion: 1,
  coreVersion: 2,
  upgradeAvailable: true,
  customizations: [
    {
      id: 'dim:custom_region',
      kind: 'dimension',
      label: 'Dimension “custom_region”',
      changes: [{ path: [{ kind: 'dimensions', value: 'custom_region' }], op: 'add', new: 'region' }],
    },
  ],
  coreUpdates: [
    {
      id: 'meas:net_flows',
      kind: 'measure',
      label: 'Measure “net_flows”',
      changes: [{ path: [{ kind: 'measures', value: 'net_flows' }], op: 'add', new: 'net_flows' }],
    },
  ],
  conflicts: [],
};

describe('CoreQueryCompareDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (compareSavedQuery as any).mockResolvedValue(comparison);
    (upgradeSavedQuery as any).mockResolvedValue({ ...query, coreStatus: 'extended' });
    (revertSavedQuery as any).mockResolvedValue({ ...query, coreStatus: 'vanilla' });
  });

  it('renders comparison details and upgrades', async () => {
    const onChanged = vi.fn();
    const onClose = vi.fn();

    render(<CoreQueryCompareDialog query={query} onClose={onClose} onChanged={onChanged} />);

    await screen.findByText(/core is now v2/);
    expect(screen.getByText('Dimension “custom_region”')).toBeTruthy();
    expect(screen.getByText('Measure “net_flows”')).toBeTruthy();

    await userEvent.click(screen.getByRole('button', { name: 'Upgrade to v2' }));

    await waitFor(() => expect(upgradeSavedQuery).toHaveBeenCalledWith('sq-aum', { remove: [] }));
    expect(onChanged).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });

  it('handles revert to vanilla core with confirmation flow', async () => {
    const onChanged = vi.fn();
    const onClose = vi.fn();

    render(<CoreQueryCompareDialog query={query} onClose={onClose} onChanged={onChanged} />);

    await screen.findByText(/core is now v2/);

    await userEvent.click(screen.getByRole('button', { name: 'Revert to Core' }));
    expect(screen.getByText(/Reverting switches this query back to vanilla core/)).toBeTruthy();

    await userEvent.click(screen.getByRole('button', { name: 'Confirm Revert' }));

    await waitFor(() => expect(revertSavedQuery).toHaveBeenCalledWith('sq-aum'));
    expect(onChanged).toHaveBeenCalled();
    expect(onClose).toHaveBeenCalled();
  });
});
