export interface FilterCondition {
  id: string;
  field: string;
  operator: string;
  value: string | number | boolean;
  values?: string[];
  logic?: 'and' | 'or';
}

export interface FilterModel {
  id?: string;
  name: string;
  conditions: FilterCondition[];
  logic: 'and' | 'or';
}

export interface FilterCategory {
  id: string;
  name: string;
  filters: FilterModel[];
}

export type FilterOperator = 
  | 'equals'
  | 'notEquals'
  | 'contains'
  | 'notContains'
  | 'startsWith'
  | 'endsWith'
  | 'greaterThan'
  | 'lessThan'
  | 'between'
  | 'in'
  | 'isEmpty'
  | 'isNotEmpty';
