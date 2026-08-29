export type FieldType = 'string' | 'number' | 'date' | 'boolean' | 'currency';

export interface Operator {
  value: string;
  label: string;
  requiresValue: boolean;
  requiresListValues?: boolean;
}

const STRING_OPERATORS: Operator[] = [
  { value: 'equals', label: 'Equals', requiresValue: true },
  { value: 'notEquals', label: 'Not Equals', requiresValue: true },
  { value: 'contains', label: 'Contains', requiresValue: true },
  { value: 'notContains', label: 'Does Not Contain', requiresValue: true },
  { value: 'startsWith', label: 'Starts With', requiresValue: true },
  { value: 'endsWith', label: 'Ends With', requiresValue: true },
  { value: 'isEmpty', label: 'Is Empty', requiresValue: false },
  { value: 'isNotEmpty', label: 'Is Not Empty', requiresValue: false },
  { value: 'in', label: 'In', requiresValue: true, requiresListValues: true },
  { value: 'notIn', label: 'Not In', requiresValue: true, requiresListValues: true },
];

const NUMBER_OPERATORS: Operator[] = [
  { value: 'equals', label: 'Equals', requiresValue: true },
  { value: 'notEquals', label: 'Not Equals', requiresValue: true },
  { value: 'greaterThan', label: 'Greater Than', requiresValue: true },
  { value: 'greaterThanOrEqual', label: 'Greater Than or Equal', requiresValue: true },
  { value: 'lessThan', label: 'Less Than', requiresValue: true },
  { value: 'lessThanOrEqual', label: 'Less Than or Equal', requiresValue: true },
  { value: 'between', label: 'Between', requiresValue: true },
  { value: 'isEmpty', label: 'Is Empty', requiresValue: false },
  { value: 'isNotEmpty', label: 'Is Not Empty', requiresValue: false },
];

const DATE_OPERATORS: Operator[] = [
  { value: 'equals', label: 'Equals', requiresValue: true },
  { value: 'notEquals', label: 'Not Equals', requiresValue: true },
  { value: 'greaterThan', label: 'After', requiresValue: true },
  { value: 'greaterThanOrEqual', label: 'On or After', requiresValue: true },
  { value: 'lessThan', label: 'Before', requiresValue: true },
  { value: 'lessThanOrEqual', label: 'On or Before', requiresValue: true },
  { value: 'between', label: 'Between', requiresValue: true },
  { value: 'isEmpty', label: 'Is Empty', requiresValue: false },
  { value: 'isNotEmpty', label: 'Is Not Empty', requiresValue: false },
];

const BOOLEAN_OPERATORS: Operator[] = [
  { value: 'equals', label: 'Equals', requiresValue: true },
  { value: 'notEquals', label: 'Not Equals', requiresValue: true },
];

export function getOperatorsForFieldType(fieldType: FieldType): Operator[] {
  switch (fieldType) {
    case 'string':
      return STRING_OPERATORS;
    case 'number':
    case 'currency':
      return NUMBER_OPERATORS;
    case 'date':
      return DATE_OPERATORS;
    case 'boolean':
      return BOOLEAN_OPERATORS;
    default:
      return STRING_OPERATORS;
  }
}

export function needsValue(operator: string): boolean {
  return !['isEmpty', 'isNotEmpty'].includes(operator);
}

export function needsListValues(operator: string): boolean {
  return ['in', 'notIn'].includes(operator);
}
