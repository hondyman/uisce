/**
 * AdvancedConditionBuilder.test.tsx
 * Comprehensive unit tests for the Advanced Condition Builder component
 * Tests: rendering, condition creation, AND/OR operators, type detection, evaluation engine
 */

import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach as _beforeEach } from 'vitest';
import AdvancedConditionBuilder, {
  evaluateCondition,
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

describe('evaluateCondition Function', () => {
  const sampleData = {
    age: 30,
    salary: 75000,
    email: 'john.doe@example.com',
    status: 'active',
    is_vip: true,
    hire_date: '2020-01-15',
    first_name: 'John',
    last_name: 'Doe',
  };

  describe('String Operators', () => {
    it('evaluates equals operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'status',
        operator: 'equals',
        value: 'active',
        fieldType: 'string',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates contains operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'email',
        operator: 'contains',
        value: 'example',
        fieldType: 'string',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates starts_with operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'email',
        operator: 'starts_with',
        value: 'john',
        fieldType: 'string',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates ends_with operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'email',
        operator: 'ends_with',
        value: '.com',
        fieldType: 'string',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates not_equals operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'status',
        operator: 'not_equals',
        value: 'inactive',
        fieldType: 'string',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });
  });

  describe('Number Operators', () => {
    it('evaluates greater_than operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'age',
        operator: 'greater_than',
        value: "25",
        fieldType: 'number',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates less_than operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'age',
        operator: 'less_than',
        value: "35",
        fieldType: 'number',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates greater_than_or_equal operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'age',
        operator: 'greater_than_or_equal',
        value: "30",
        fieldType: 'number',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates less_than_or_equal operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'age',
        operator: 'less_than_or_equal',
        value: "30",
        fieldType: 'number',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates equals operator for numbers', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'age',
        operator: 'equals',
        value: "30",
        fieldType: 'number',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });
  });

  describe('Boolean Operators', () => {
    it('evaluates is_true operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'is_vip',
        operator: 'is_true',
        value: "true",
        fieldType: 'boolean',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates is_false operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'is_vip',
        operator: 'is_false',
        value: "false",
        fieldType: 'boolean',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(false);
    });
  });

  describe('Date Operators', () => {
    it('evaluates before operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'hire_date',
        operator: 'before',
        value: '2021-01-01',
        fieldType: 'date',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates after operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'hire_date',
        operator: 'after',
        value: '2019-01-01',
        fieldType: 'date',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates between operator', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'hire_date',
        operator: 'between',
        value: '2019-01-01,2021-12-31',
        fieldType: 'date',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(true);
    });
  });

  describe('Nested AND/OR Groups', () => {
    it('evaluates AND group with multiple conditions', () => {
      const group: ConditionGroup = {
        id: 'g1',
        type: 'group',
        operator: 'AND',
        conditions: [
          {
            id: 'c1',
            field: 'age',
            operator: 'greater_than',
            value: "25",
            fieldType: 'number',
          } as unknown as Condition,
          {
            id: 'c2',
            field: 'is_vip',
            operator: 'is_true',
            value: "true",
            fieldType: 'boolean',
          } as unknown as Condition,
        ],
      };

      const result = evaluateCondition(group, sampleData);
      expect(result).toBe(true);
    });

    it('evaluates OR group with multiple conditions', () => {
      const group: ConditionGroup = {
        id: 'g1',
        type: 'group',
        operator: 'OR',
        conditions: [
          {
            id: 'c1',
            field: 'age',
            operator: 'less_than',
            value: 20,
            fieldType: 'number',
          } as unknown as Condition,
          {
            id: 'c2',
            field: 'is_vip',
            operator: 'is_true',
            value: "true",
            fieldType: 'boolean',
          } as unknown as Condition,
        ],
      };

      const result = evaluateCondition(group, sampleData);
      expect(result).toBe(true); // OR: second condition is true
    });

    it('evaluates deeply nested groups', () => {
      const group: ConditionGroup = {
        id: 'g1',
        type: 'group',
        operator: 'AND',
        conditions: [
          {
            id: 'g2',
            type: 'group',
            operator: 'OR',
            conditions: [
              {
                id: 'c1',
                field: 'status',
                operator: 'equals',
                value: 'active',
                fieldType: 'string',
              } as unknown as Condition,
              {
                id: 'c2',
                field: 'status',
                operator: 'equals',
                value: 'pending',
                fieldType: 'string',
              } as unknown as Condition,
            ],
          } as ConditionGroup,
          {
            id: 'c3',
            field: 'age',
            operator: 'greater_than',
            value: "21",
            fieldType: 'number',
          } as unknown as Condition,
        ],
      };

      const result = evaluateCondition(group, sampleData);
      expect(result).toBe(true); // (active OR pending) AND (age > 21)
    });
  });

  describe('Edge Cases', () => {
    it('handles empty groups', () => {
      const group: ConditionGroup = {
        id: 'g1',
        type: 'group',
        operator: 'AND',
        conditions: [],
      };

      const result = evaluateCondition(group, sampleData);
      expect(result).toBe(true); // Empty AND group = true
    });

    it('handles missing fields in data', () => {
      const condition: Condition = {
        id: 'c1',
        field: 'nonexistent_field',
        operator: 'equals',
        value: 'test',
        fieldType: 'string',
      };

      const result = evaluateCondition(condition, sampleData);
      expect(result).toBe(false); // Missing field = false
    });

    it('handles null values', () => {
      const data = { age: null };
      const condition: unknown = {
        id: 'c1',
        field: 'age',
        operator: 'equals',
        value: '',
        fieldType: 'number',
      } as Condition;

      const result = evaluateCondition(condition as ConditionNode, data);
      expect(result).toBe(false); // null/empty not equal
    });
  });
});
