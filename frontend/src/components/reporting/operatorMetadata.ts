import { FilterOperator } from './filterTypes';

export interface OperatorDef {
  id: FilterOperator;
  label: string;
  valueType: 'string' | 'number' | 'date' | 'boolean' | 'list';
  groupLabel: string;
}

export const OPERATOR_GROUPS: { label: string; operators: OperatorDef[] }[] = [
  {
    label: 'Comparison',
    operators: [
      { id: 'equals', label: 'equals', valueType: 'string', groupLabel: 'Comparison' },
      { id: 'not_equals', label: 'does not equal', valueType: 'string', groupLabel: 'Comparison' },
      { id: 'greater_than', label: 'is greater than', valueType: 'number', groupLabel: 'Comparison' },
      { id: 'less_than', label: 'is less than', valueType: 'number', groupLabel: 'Comparison' },
      { id: 'greater_equal', label: 'is greater than or equal to', valueType: 'number', groupLabel: 'Comparison' },
      { id: 'less_equal', label: 'is less than or equal to', valueType: 'number', groupLabel: 'Comparison' },
    ],
  },
  {
    label: 'Range',
    operators: [
      { id: 'between', label: 'is between', valueType: 'list', groupLabel: 'Range' },
      { id: 'not_between', label: 'is not between', valueType: 'list', groupLabel: 'Range' },
      { id: 'in', label: 'is in', valueType: 'list', groupLabel: 'Range' },
      { id: 'not_in', label: 'is not in', valueType: 'list', groupLabel: 'Range' },
    ],
  },
  {
    label: 'Null',
    operators: [
      { id: 'is_null', label: 'is null', valueType: 'boolean', groupLabel: 'Null' },
      { id: 'is_not_null', label: 'is not null', valueType: 'boolean', groupLabel: 'Null' },
    ],
  },
  {
    label: 'Text',
    operators: [
      { id: 'contains', label: 'contains', valueType: 'string', groupLabel: 'Text' },
      { id: 'starts_with', label: 'starts with', valueType: 'string', groupLabel: 'Text' },
      { id: 'ends_with', label: 'ends with', valueType: 'string', groupLabel: 'Text' },
    ],
  },
  {
    label: 'Calendar',
    operators: [
      { id: 'is_business_day', label: 'is a business day', valueType: 'boolean', groupLabel: 'Calendar' },
      { id: 'is_holiday', label: 'is a holiday', valueType: 'boolean', groupLabel: 'Calendar' },
      { id: 'next_business_day', label: 'is the next business day', valueType: 'boolean', groupLabel: 'Calendar' },
      { id: 'previous_business_day', label: 'is the previous business day', valueType: 'boolean', groupLabel: 'Calendar' },
      { id: 'add_business_days', label: 'plus N business days', valueType: 'list', groupLabel: 'Calendar' },
      { id: 'last_n_business_days', label: 'is in the last N business days', valueType: 'list', groupLabel: 'Calendar' },
      { id: 'next_n_business_days', label: 'is in the next N business days', valueType: 'list', groupLabel: 'Calendar' },
    ],
  },
  {
    label: 'Relative Date',
    operators: [
      { id: 'today', label: 'is today', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'yesterday', label: 'is yesterday', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'tomorrow', label: 'is tomorrow', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'start_of_week', label: 'is at start of this week', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'end_of_week', label: 'is at end of this week', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'start_of_month', label: 'is at start of this month', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'end_of_month', label: 'is at end of this month', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'start_of_quarter', label: 'is at start of this quarter', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'end_of_quarter', label: 'is at end of this quarter', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'start_of_year', label: 'is at start of this year', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'end_of_year', label: 'is at end of this year', valueType: 'boolean', groupLabel: 'Relative Date' },
      { id: 'last_n_days', label: 'is in the last N days', valueType: 'list', groupLabel: 'Relative Date' },
    ],
  },
  {
    label: 'Offset',
    operators: [
      { id: 'previous', label: 'is the previous [period]', valueType: 'list', groupLabel: 'Offset' },
      { id: 'next', label: 'is the next [period]', valueType: 'list', groupLabel: 'Offset' },
    ],
  },
];

export function getOperatorsForFieldType(dataType: string): OperatorDef[] {
  const t = (dataType || '').toLowerCase();
  const isDate = ['date', 'time', 'timestamp', 'datetime'].some(k => t.includes(k));
  const isNumeric = ['number', 'int', 'float', 'double', 'decimal', 'numeric', 'currency', 'money'].some(k => t.includes(k));
  const isBool = ['bool', 'boolean'].some(k => t.includes(k));

  if (isDate) {
    return [
      ...OPERATOR_GROUPS.find(g => g.label === 'Comparison')!.operators.filter(o => o.valueType === 'string'),
      ...OPERATOR_GROUPS.find(g => g.label === 'Range')!.operators,
      ...OPERATOR_GROUPS.find(g => g.label === 'Null')!.operators,
      ...OPERATOR_GROUPS.find(g => g.label === 'Calendar')!.operators,
      ...OPERATOR_GROUPS.find(g => g.label === 'Relative Date')!.operators,
      ...OPERATOR_GROUPS.find(g => g.label === 'Offset')!.operators,
    ];
  }
  if (isNumeric) {
    return [
      ...OPERATOR_GROUPS.find(g => g.label === 'Comparison')!.operators.filter(o => ['number', 'string'].includes(o.valueType)),
      ...OPERATOR_GROUPS.find(g => g.label === 'Range')!.operators,
      ...OPERATOR_GROUPS.find(g => g.label === 'Null')!.operators,
    ];
  }
  if (isBool) {
    return OPERATOR_GROUPS.find(g => g.label === 'Null')!.operators;
  }
  return [
    ...OPERATOR_GROUPS.find(g => g.label === 'Comparison')!.operators,
    ...OPERATOR_GROUPS.find(g => g.label === 'Range')!.operators,
    ...OPERATOR_GROUPS.find(g => g.label === 'Null')!.operators,
    ...OPERATOR_GROUPS.find(g => g.label === 'Text')!.operators,
  ];
}

export function getAllOperators(): OperatorDef[] {
  return OPERATOR_GROUPS.flatMap(g => g.operators);
}

export function getOperatorById(id: FilterOperator): OperatorDef | undefined {
  return getAllOperators().find(o => o.id === id);
}

export function needsValue(operator: FilterOperator): boolean {
  return !['is_null', 'is_not_null', 'today', 'yesterday', 'tomorrow', 'start_of_week', 'end_of_week',
    'start_of_month', 'end_of_month', 'start_of_quarter', 'end_of_quarter', 'start_of_year', 'end_of_year',
    'is_business_day', 'is_holiday', 'next_business_day', 'previous_business_day'].includes(operator);
}

export function needsListValues(operator: FilterOperator): boolean {
  return ['between', 'not_between', 'in', 'not_in', 'add_business_days', 'last_n_days', 'last_n_business_days', 'next_n_business_days', 'previous', 'next'].includes(operator);
}
