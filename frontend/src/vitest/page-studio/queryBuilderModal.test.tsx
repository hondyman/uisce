import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import QueryBuilderModal, {
  CoreStatusBadge,
  validateParamBinding,
} from '../../pages/page-studio/app/QueryBuilderModal';
import type { SavedQuery, SavedQueryParameter } from '../../features/query-builder/types/queryDef';
import type { PageVariable } from '../../pages/page-studio/app/appModel';

vi.mock('../../features/query-builder/services/savedQueryApi', () => ({
  listSavedQueries: vi.fn(),
  createSavedQuery: vi.fn(),
  runSavedQuery: vi.fn(),
}));

vi.mock('../../features/query-builder/services/queryBuilderApi', () => ({
  previewQuery: vi.fn(),
}));

vi.mock('../../contexts/TenantContext', () => ({
  useTenant: () => ({
    tenant: { id: 'test-tenant', name: 'Test Tenant' },
  }),
}));

vi.mock('../../studio-core/binding/useBusinessObjectSelector', () => ({
  useBusinessObjectSelector: () => ({
    primary: null,
    related: [],
    objects: [],
    setPrimaryBinding: vi.fn(),
    toggleField: vi.fn(),
  }),
}));

import { listSavedQueries } from '../../features/query-builder/services/savedQueryApi';

describe('QueryBuilderModal - validateParamBinding', () => {
  const pageVariables: PageVariable[] = [
    { name: 'fundId', default: 'fund-123' },
    { name: 'minAmount', default: 5000 },
    { name: 'isActive', default: true },
  ];

  it('validates required vs optional parameters without binding', () => {
    const reqParam: SavedQueryParameter = { name: 'param1', type: 'string', required: true };
    const optParam: SavedQueryParameter = { name: 'param2', type: 'string', required: false };

    expect(validateParamBinding(reqParam, undefined, pageVariables).valid).toBe(false);
    expect(validateParamBinding(reqParam, undefined, pageVariables).message).toBe(
      'Required parameter has no binding'
    );
    expect(validateParamBinding(optParam, undefined, pageVariables).valid).toBe(true);
  });

  it('validates pageVar bindings', () => {
    const numParam: SavedQueryParameter = { name: 'amount', type: 'number', required: true };

    // Valid number variable
    expect(
      validateParamBinding(
        numParam,
        { mode: 'pageVar', varName: 'minAmount' },
        pageVariables
      ).valid
    ).toBe(true);

    // Missing variable
    const missingRes = validateParamBinding(
      numParam,
      { mode: 'pageVar', varName: 'unknownVar' },
      pageVariables
    );
    expect(missingRes.valid).toBe(false);
    expect(missingRes.message).toContain('not found');

    // Type mismatch (number expected, string given)
    const mismatchRes = validateParamBinding(
      numParam,
      { mode: 'pageVar', varName: 'fundId' },
      pageVariables
    );
    expect(mismatchRes.valid).toBe(false);
    expect(mismatchRes.message).toContain('Type mismatch');
  });

  it('validates staticLiteral bindings', () => {
    const numParam: SavedQueryParameter = { name: 'limit', type: 'number', required: true };

    expect(
      validateParamBinding(numParam, { mode: 'staticLiteral', value: 100 }, pageVariables).valid
    ).toBe(true);
    expect(
      validateParamBinding(numParam, { mode: 'staticLiteral', value: '100' }, pageVariables).valid
    ).toBe(true);
    expect(
      validateParamBinding(numParam, { mode: 'staticLiteral', value: 'not_a_number' }, pageVariables).valid
    ).toBe(false);
  });
});

describe('QueryBuilderModal - CoreStatusBadge', () => {
  it('renders correct labels for all core statuses', () => {
    const { rerender } = render(<CoreStatusBadge status="core" />);
    expect(screen.getByText('CORE (Master)')).toBeTruthy();

    rerender(<CoreStatusBadge status="vanilla" />);
    expect(screen.getByText('CORE (Vanilla)')).toBeTruthy();

    rerender(<CoreStatusBadge status="extended" />);
    expect(screen.getByText('CORE (Extended)')).toBeTruthy();

    rerender(<CoreStatusBadge status="upgrade_available" />);
    expect(screen.getByText('UPGRADE AVAILABLE')).toBeTruthy();

    rerender(<CoreStatusBadge status="cloned" />);
    expect(screen.getByText('CLONED')).toBeTruthy();

    rerender(<CoreStatusBadge status="custom" />);
    expect(screen.getByText('CUSTOM')).toBeTruthy();
  });
});

describe('QueryBuilderModal - Component Integration', () => {
  const mockQueries: SavedQuery[] = [
    {
      id: 'core-aum',
      name: 'Core AUM Query',
      description: 'Standard institutional AUM breakdown',
      isCore: true,
      coreStatus: 'core',
      boId: 'bo-aum',
      chartType: 'bar',
      state: {
        dimensions: [{ termNodeId: 'term-dim-1', alias: 'strategy' }],
        measures: [{ termNodeId: 'term-m-1', alias: 'total_aum', agg: 'SUM' }],
        filters: [],
        parameters: [
          { name: 'fundFilter', label: 'Fund Filter', type: 'string', required: true },
        ],
      },
    },
    {
      id: 'custom-query',
      name: 'Custom Revenue Query',
      isCore: false,
      coreStatus: 'custom',
      boId: 'bo-rev',
      chartType: 'line',
      state: {
        dimensions: [],
        measures: [],
        filters: [],
        parameters: [],
      },
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
    (listSavedQueries as any).mockResolvedValue(mockQueries);
  });

  it('renders tabs and allows switching between visual and library tabs', async () => {
    render(
      <QueryBuilderModal
        open={true}
        onClose={vi.fn()}
        onSave={vi.fn()}
        pageVariables={[{ name: 'fundId', default: 'fund-1' }]}
      />
    );

    expect(screen.getByText('Query Studio')).toBeTruthy();
    expect(screen.getByRole('tab', { name: 'Visual Query Designer' })).toBeTruthy();
    expect(screen.getByRole('tab', { name: 'Public & Core Query Library' })).toBeTruthy();

    // Switch to library tab
    await userEvent.click(screen.getByRole('tab', { name: 'Public & Core Query Library' }));

    const matches = await screen.findAllByText('Core AUM Query');
    expect(matches.length).toBeGreaterThanOrEqual(1);
    expect(screen.getByText('Custom Revenue Query')).toBeTruthy();
    expect(screen.getAllByText('CORE (Master)').length).toBeGreaterThanOrEqual(1);
  });

  it('binds parameters and saves savedQuery PageQuery shape', async () => {
    const onSave = vi.fn();
    const onClose = vi.fn();

    render(
      <QueryBuilderModal
        open={true}
        onClose={onClose}
        onSave={onSave}
        pageVariables={[{ name: 'selectedFund', default: 'alpha' }]}
      />
    );

    // Switch to library
    await userEvent.click(screen.getByRole('tab', { name: 'Public & Core Query Library' }));
    await screen.findAllByText('Core AUM Query');

    // Save and bind query
    await userEvent.click(screen.getByRole('button', { name: 'Select & Bind Query' }));

    expect(onSave).toHaveBeenCalledWith({
      id: 'core_aum_query',
      kind: 'savedQuery',
      operation: '',
      savedQueryId: 'core-aum',
      paramBindings: {},
    });
    expect(onClose).toHaveBeenCalled();
  });
});
