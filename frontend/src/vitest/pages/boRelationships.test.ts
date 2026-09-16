import { describe, expect, it } from 'vitest';
import { fkColumnFromJoinCondition, widgetTypeForCardinality } from '../../pages/page-studio/boRelationships';

describe('widgetTypeForCardinality', () => {
  it('places a related list when the target side is many', () => {
    expect(widgetTypeForCardinality('1:N')).toBe('Table');
    expect(widgetTypeForCardinality('1:M')).toBe('Table');
    expect(widgetTypeForCardinality('N:M')).toBe('Table');
  });

  it('places a form when the target side is one', () => {
    expect(widgetTypeForCardinality('N:1')).toBe('Form');
    expect(widgetTypeForCardinality('1:1')).toBe('Form');
    expect(widgetTypeForCardinality('')).toBe('Form');
  });
});

describe('fkColumnFromJoinCondition', () => {
  it('takes the child-side column from a resolved FK', () => {
    expect(fkColumnFromJoinCondition('execution.order_id = order.id')).toBe('order_id');
  });
});
