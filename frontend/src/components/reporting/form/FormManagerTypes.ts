import type { DynamicProperty } from '../evaluateDynamicProperty';
import type { ConditionGroup } from '../../ExpressionBuilder/AdvancedConditionBuilder';

export type FormElementType =
  | 'SEMANTIC_FIELD'
  | 'STATIC_LABEL'
  | 'KEY_VALUE_PAIR'
  | 'DIVIDER'
  | 'SIGNATURE_BLOCK'
  | 'CHECKBOX_GROUP';

export type ColSpan = 1 | 2 | 3 | 4 | 6 | 12;

export type BorderSideStyle = 'none' | 'solid' | 'dashed' | 'dotted' | 'double';

export interface BorderSide {
  width: number;
  style: BorderSideStyle;
  color: string;
}

export interface CellStyle {
  fontFamily?: string;
  fontSize?: number;
  fontWeight?: 300 | 400 | 500 | 600 | 700 | 800;
  fontStyle?: 'normal' | 'italic';
  textDecoration?: 'none' | 'underline' | 'line-through' | 'underline line-through';
  textTransform?: 'none' | 'uppercase' | 'lowercase' | 'capitalize';
  textAlign?: 'left' | 'center' | 'right' | 'justify';
  color?: string;
  backgroundColor?: string;
  borderTop?: BorderSide;
  borderRight?: BorderSide;
  borderBottom?: BorderSide;
  borderLeft?: BorderSide;
  paddingTop?: number;
  paddingRight?: number;
  paddingBottom?: number;
  paddingLeft?: number;
}

export interface FormFieldItem {
  id: string;
  type: FormElementType;
  label: string;
  fieldKey?: string;
  valueExpression?: string | DynamicProperty<string>;
  colSpan: ColSpan;
  formatMask?: 'AUTO' | 'CURRENCY' | 'PERCENT' | 'DATE' | 'DECIMAL' | 'INTEGER' | 'TEXT' | 'Custom' | string;
  formatPrefix?: string;
  formatSuffix?: string;
  isRequired?: boolean;
  isReadOnly?: boolean;
  fontSize?: string;
  textColor?: string;
  labelPlacement?: 'TOP' | 'LEFT' | 'INLINE';
  style?: CellStyle;
  visibilityCondition?: ConditionGroup | null;
}

export interface FormSection {
  id: string;
  title: string;
  description?: string;
  columns: 1 | 2 | 3 | 4;
  isCollapsible?: boolean;
  isCollapsed?: boolean;
  items: FormFieldItem[];
}

export interface FormTemplateSpec {
  templateId: string;
  title: string;
  formSpecVersion?: number;
  sections: FormSection[];
}
