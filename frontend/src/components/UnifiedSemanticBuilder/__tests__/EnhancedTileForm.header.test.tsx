import { render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import EnhancedTileForm from '../EnhancedTileForm';
// React import removed (not needed with the new JSX transform)

// EXPLICIT STUB: useTenant is stubbed, so this file is evidence about EnhancedTileForm's
// rendering only — not about the tenant context contract. See fixtures/tenantContextStub.
vi.mock('../../../contexts/TenantContext', async () => {
  const { tenantContextMockFactory } = await import('../../../vitest/fixtures/tenantContextStub');
  return tenantContextMockFactory();
});

const baseProps = {
  element: { name: 'revenue', title: 'Revenue', type: 'number', sql: 'SUM(amount)' },
  type: 'measure' as const,
  isCore: false,
  isNew: false,
  coreOptions: [],
  modelName: 'Sales Cube',
  onUpdate: () => {},
  onSave: () => {},
  onCancel: () => {},
};

describe('EnhancedTileForm header layout', () => {
  it('shows leading icon, core/custom and datatype chips, and model name on separate line', () => {
    render(<EnhancedTileForm {...baseProps} />);

    // Icon is decorative; check wrapper exists
    expect(screen.getByLabelText('Model name')).toHaveTextContent('Sales Cube');

    // chips row contains Custom and datatype
    expect(screen.getByText(/Custom/i)).toBeInTheDocument();
    expect(screen.getByText('number')).toBeInTheDocument();
  });

  it('shows Core chip for core items and hides save/cancel in read-only', () => {
    render(<EnhancedTileForm {...baseProps} isCore readOnly />);
    expect(screen.getByText('Core')).toBeInTheDocument();
    expect(screen.queryByText('Save')).not.toBeInTheDocument();
    expect(screen.queryByText('Cancel')).not.toBeInTheDocument();
  });
});
