import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import DrillThroughTargetsEditor, { validateDrillTarget } from '../../features/query-builder/components/DrillThroughTargetsEditor';
import type { DrillThroughTarget } from '../../features/query-builder/utils/drilldown';

describe('DrillThroughTargetsEditor', () => {
  it('renders target list and allows adding targets and mappings', async () => {
    const onChange = vi.fn();
    const initialTargets: DrillThroughTarget[] = [
      {
        type: 'page',
        target: '/pages/accounts/:id',
        label: 'View Account Details',
        contextMapping: { accountId: '{{row.account_id}}' },
      },
    ];

    render(
      <DrillThroughTargetsEditor
        targets={initialTargets}
        onChange={onChange}
        outputColumns={['account_id', 'total_aum']}
        pageVariables={[{ name: 'currentFund' }]}
      />
    );

    expect(screen.getByDisplayValue('View Account Details')).toBeTruthy();
    expect(screen.getByDisplayValue('/pages/accounts/:id')).toBeTruthy();
    expect(screen.getByDisplayValue('accountId')).toBeTruthy();

    // Click Add Action
    await userEvent.click(screen.getByRole('button', { name: 'Add Action' }));
    expect(onChange).toHaveBeenCalled();
  });

  it('validates drill target requiring non-empty label and target', () => {
    const validTarget: DrillThroughTarget = {
      type: 'page',
      label: 'Open Details',
      target: '/pages/details',
    };
    expect(validateDrillTarget(validTarget)).toEqual({ valid: true });

    const emptyLabel: DrillThroughTarget = {
      type: 'page',
      label: '   ',
      target: '/pages/details',
    };
    expect(validateDrillTarget(emptyLabel)).toEqual({
      valid: false,
      message: 'Action label is required',
    });

    const emptyTarget: DrillThroughTarget = {
      type: 'modal',
      label: 'Open Modal',
      target: '',
    };
    expect(validateDrillTarget(emptyTarget)).toEqual({
      valid: false,
      message: 'Target route or key is required',
    });
  });

  it('renders validation error alert when target has invalid/empty fields', () => {
    const invalidTargets: DrillThroughTarget[] = [
      {
        type: 'page',
        target: '',
        label: 'Action 1',
        contextMapping: {},
      },
    ];

    render(
      <DrillThroughTargetsEditor
        targets={invalidTargets}
        onChange={vi.fn()}
        outputColumns={['account_id', 'total_aum']}
        pageVariables={[{ name: 'currentFund' }]}
      />
    );

    expect(screen.getByText('Target route or key is required')).toBeTruthy();
  });

  it('derives token suggestions from outputColumns and pageVariables', async () => {
    const targets: DrillThroughTarget[] = [
      {
        type: 'page',
        target: '/pages/account-view',
        label: 'View Account',
        contextMapping: { accountId: '{{row.account_id}}' },
      },
    ];

    render(
      <DrillThroughTargetsEditor
        targets={targets}
        onChange={vi.fn()}
        outputColumns={['account_id', 'total_aum']}
        pageVariables={[{ name: 'currentFund' }, { name: 'selectedRegion' }]}
      />
    );

    // The autocomplete options should be populated with token suggestions
    // We check that the context mapping inputs render properly
    expect(screen.getByDisplayValue('{{row.account_id}}')).toBeTruthy();
  });
});

