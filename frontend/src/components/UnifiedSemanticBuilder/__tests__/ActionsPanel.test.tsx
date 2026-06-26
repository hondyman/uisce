// React import removed (automatic JSX runtime)
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import ActionsPanel from '../ActionsPanel';

const mockColumn = {
  nodeId: 'node-1',
  tableName: 'orders',
  column: { name: 'order_id', type: 'string' },
  id: 'col-1'
};

describe('ActionsPanel', () => {
  it('renders nothing when selectedColumn is null', () => {
    const { container } = render(
      <ActionsPanel selectedColumn={null} addDimension={() => {}} addMeasure={() => {}} addFilter={() => {}} getBusinessTermForColumn={() => {}} />
    );
    expect(container).toBeTruthy();
    // Should render no actions-panel section
    expect(container.querySelector('.actions-panel')).toBeNull();
  });

  it('renders ColumnActionsPanel when selectedColumn is provided', () => {
    render(
      <ActionsPanel selectedColumn={mockColumn as any} addDimension={() => {}} addMeasure={() => {}} addFilter={() => {}} getBusinessTermForColumn={() => {}} />
    );
    // ColumnActionsPanel renders buttons with class 'btn' typically; check existence
    const el = document.querySelector('.actions-panel');
    expect(el).not.toBeNull();
  });
});
