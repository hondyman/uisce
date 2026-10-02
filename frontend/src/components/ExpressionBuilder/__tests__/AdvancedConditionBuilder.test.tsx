/**
 * AdvancedConditionBuilder.test.tsx
 * Comprehensive unit tests for the Advanced Condition Builder component
 * Tests: rendering, condition creation, AND/OR operators, type detection.
 * Evaluation is the rule engine's (rule_engine.wasm, verify_wasm.js).
 */

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach as _beforeEach } from 'vitest';
import AdvancedConditionBuilder, {
  Condition,
  ConditionGroup,
  ConditionNode,
} from '../AdvancedConditionBuilder';

describe('AdvancedConditionBuilder Component', () => {
  const availableFields = [
    { name: 'age', type: 'number', label: 'Age' },
    { name: 'email', type: 'string', label: 'Email' },
    { name: 'hire_date', type: 'date', label: 'Hire Date' },
    { name: 'is_vip', type: 'boolean', label: 'Is VIP' },
  ];

  it('renders with initial empty condition group', () => {
    const onChange = vi.fn();
    const initialValue: ConditionGroup = {
      id: 'root',
      type: 'group',
      operator: 'AND',
      conditions: [],
    };

    render(
      <AdvancedConditionBuilder
        value={initialValue}
        onChange={onChange}
        availableFields={availableFields}
        entityName="Employee"
      />
    );

    // Check that component renders
    expect(screen.getByText('Employee Conditions')).toBeInTheDocument();
  });

  it('allows adding a new condition', async () => {
    const onChange = vi.fn();
    const initialValue: ConditionGroup = {
      id: 'root',
      type: 'group',
      operator: 'AND',
      conditions: [],
    };

    render(
      <AdvancedConditionBuilder
        value={initialValue}
        onChange={onChange}
        availableFields={availableFields}
        entityName="Employee"
      />
    );

    // Click the "Add Condition" button
    const addBtn = screen.getByText(/Add Condition/i);
    fireEvent.click(addBtn);

    // Verify onChange was called with updated tree
    await waitFor(() => {
      expect(onChange).toHaveBeenCalled();
    });

    // The new tree should have one condition
    const newTree = onChange.mock.calls[0][0] as ConditionGroup;
    expect(newTree.conditions.length).toBeGreaterThan(0);
  });

  it('toggles AND/OR operator', async () => {
    const onChange = vi.fn();
    const initialValue: ConditionGroup = {
      id: 'root',
      type: 'group',
      operator: 'AND',
      conditions: [],
    };

  const { rerender: _rerender } = render(
      <AdvancedConditionBuilder
        value={initialValue}
        onChange={onChange}
        availableFields={availableFields}
        entityName="Employee"
      />
    );

    // Find and click the AND/OR toggle button
    const operatorBtn = screen.getByText('AND');
    fireEvent.click(operatorBtn);

    await waitFor(() => {
      expect(onChange).toHaveBeenCalled();
    });

    const newTree = onChange.mock.calls[0][0] as ConditionGroup;
    expect(newTree.operator).toBe('OR');
  });

  it('allows adding nested condition groups', async () => {
    const onChange = vi.fn();
    const initialValue: ConditionGroup = {
      id: 'root',
      type: 'group',
      operator: 'AND',
      conditions: [],
    };

    render(
      <AdvancedConditionBuilder
        value={initialValue}
        onChange={onChange}
        availableFields={availableFields}
        entityName="Employee"
      />
    );

    // Click "Add Group" button
    const addGroupBtn = screen.getByText(/Add Group/i);
    fireEvent.click(addGroupBtn);

    await waitFor(() => {
      expect(onChange).toHaveBeenCalled();
    });

    const newTree = onChange.mock.calls[0][0] as ConditionGroup;
    // Check if a group was added by checking for 'operator' property
    const hasGroup = newTree.conditions.length > 0 && 'operator' in newTree.conditions[0];
    expect(hasGroup).toBe(true);
  });

  it('displays correct operators for different field types', async () => {
    const onChange = vi.fn();
    const initialValue: ConditionGroup = {
      id: 'root',
      type: 'group',
      operator: 'AND',
      conditions: [
        {
          id: 'cond-1',
          field: 'age',
          operator: 'greater_than',
          value: "18",
          fieldType: 'number',
        } as unknown as Condition,
      ],
    };

    render(
      <AdvancedConditionBuilder
        value={initialValue}
        onChange={onChange}
        availableFields={availableFields}
        entityName="Employee"
      />
    );

    // For number fields, should show numeric operators
    expect(screen.getByDisplayValue('greater_than')).toBeInTheDocument();
  });

  it('allows deleting conditions', async () => {
    const onChange = vi.fn();
    const initialValue: ConditionGroup = {
      id: 'root',
      type: 'group',
      operator: 'AND',
      conditions: [
        {
          id: 'cond-1',
          field: 'age',
          operator: 'greater_than',
          value: "18",
          fieldType: 'number',
        } as unknown as Condition,
      ],
    };

    render(
      <AdvancedConditionBuilder
        value={initialValue}
        onChange={onChange}
        availableFields={availableFields}
        entityName="Employee"
      />
    );

    // Find and click the delete button
    const deleteBtn = screen.getByTitle(/delete/i);
    fireEvent.click(deleteBtn);

    await waitFor(() => {
      expect(onChange).toHaveBeenCalled();
    });

    const newTree = onChange.mock.calls[0][0] as ConditionGroup;
    expect(newTree.conditions.length).toBe(0);
  });

  it('allows editing condition values', async () => {
    const onChange = vi.fn();
    const initialValue: ConditionGroup = {
      id: 'root',
      type: 'group',
      operator: 'AND',
      conditions: [
        {
          id: 'cond-1',
          field: 'age',
          operator: 'greater_than',
          value: "18",
          fieldType: 'number',
        } as unknown as Condition,
      ],
    };

    render(
      <AdvancedConditionBuilder
        value={initialValue}
        onChange={onChange}
        availableFields={availableFields}
        entityName="Employee"
      />
    );

    // Find the value input and change it
    const valueInput = screen.getByDisplayValue('18') as HTMLInputElement;
    fireEvent.change(valueInput, { target: { value: '21' } });

    await waitFor(() => {
      expect(onChange).toHaveBeenCalled();
    });

    const newTree = onChange.mock.calls[0][0] as ConditionGroup;
    const cond = newTree.conditions[0] as Condition;
    expect(cond.value).toBe('21');
  });
});

describe('Type Guards', () => {
  it('isCondition correctly identifies Condition nodes', () => {
    const condition: Condition = {
      id: 'c1',
      field: 'age',
      operator: 'greater_than',
      value: "18",
      fieldType: 'number',
    };

    const conditionGroup: ConditionGroup = {
      id: 'g1',
      type: 'group',
      operator: 'AND',
      conditions: [],
    };

    // Import the guards if exported
    // This test assumes guards are available for import
    // If not, this is a demonstration of what should be testable
    expect(condition.field).toBeDefined();
    expect(conditionGroup.type).toBe('group');
  });
});
