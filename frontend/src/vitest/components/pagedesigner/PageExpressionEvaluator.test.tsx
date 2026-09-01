import { describe, it, expect } from 'vitest';
import { evaluatePageExpression, PageExpressionContext } from '../../../components/pagedesigner/evaluatePageExpression';
import { DynamicProperty } from '../../../components/pagedesigner/DynamicPropertyTypes';

describe('evaluatePageExpression Universal Evaluator', () => {
  const context: PageExpressionContext = {
    rowData: {
      nav_end: -5000,
      currency: 'EUR',
      status: 'ACTIVE',
      account_name: 'Alpha Growth Fund',
    },
    parameters: {
      SelectedRegion: 'EMEA',
      selected_account_id: 'ACC-901',
    },
    globalContext: {
      userName: 'Alice Smith',
      executionTime: new Date('2026-08-26T12:00:00Z'),
      tenantId: 'tenant-101',
    },
  };

  it('returns static fallback value when not an expression', () => {
    const prop: DynamicProperty<string> = { isExpression: false, value: '#F8FAFC' };
    const result = evaluatePageExpression(prop, context);
    expect(result).toBe('#F8FAFC');
  });

  it('evaluates SSRS IIF expressions with BO Field values (Fields!name.Value)', () => {
    const prop: DynamicProperty<string> = {
      isExpression: true,
      value: '#F8FAFC',
      formula: "=IIF(Fields!nav_end.Value < 0, '#EF4444', '#10B981')",
    };
    const result = evaluatePageExpression(prop, context);
    expect(result).toBe('#EF4444');
  });

  it('evaluates SSRS IIF expressions with square bracket field tokens [currency]', () => {
    const prop: DynamicProperty<string> = {
      isExpression: true,
      value: '$#,##0.00',
      formula: "=IIF([currency] == 'EUR', '€#,##0.00', '$#,##0.00')",
    };
    const result = evaluatePageExpression(prop, context);
    expect(result).toBe('€#,##0.00');
  });

  it('evaluates event bus parameters (Parameters!SelectedRegion.Value & @SelectedRegion)', () => {
    const prop1: DynamicProperty<boolean> = {
      isExpression: true,
      value: false,
      formula: "=Parameters!SelectedRegion.Value == 'EMEA'",
    };
    expect(evaluatePageExpression(prop1, context)).toBe(true);

    const prop2: DynamicProperty<boolean> = {
      isExpression: true,
      value: false,
      formula: "=@SelectedRegion == 'APAC'",
    };
    expect(evaluatePageExpression(prop2, context)).toBe(false);
  });

  it('substitutes global tokens {UserName} and {TenantId}', () => {
    const prop: DynamicProperty<string> = {
      isExpression: true,
      value: '',
      formula: "={UserName} + ' - ' + {TenantId}",
    };
    const result = evaluatePageExpression(prop, context);
    expect(result).toBe('Alice Smith - tenant-101');
  });
});
