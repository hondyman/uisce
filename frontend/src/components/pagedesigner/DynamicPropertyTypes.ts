import { ConditionGroup } from '../reporting/AdvancedConditionBuilder';

export type DynamicProperty<T> = {
  isExpression: boolean;
  value: T;        // Static constant fallback
  formula?: string; // e.g., "=IIF(Fields!nav_end.Value < 0, '#EF4444', '#10B981')"
};

export interface DynamicFieldUIConfig {
  // 1. Dynamic Formatting & Styling
  textColor?: DynamicProperty<string>;
  backgroundColor?: DynamicProperty<string>;
  fontWeight?: DynamicProperty<string | number>;
  formatMask?: DynamicProperty<string>; // e.g. "=IIF(Fields!currency.Value == 'EUR', '€#,##0.00', '$#,##0.00')"

  // 2. Dynamic Rendering & Interactivity
  isVisible?: DynamicProperty<boolean>;
  isReadOnly?: DynamicProperty<boolean>;
  isRequired?: DynamicProperty<boolean>;

  // 3. Conditional Rule Overrides
  visibilityCondition?: ConditionGroup | null;
}

export type PagePresentationConfig = DynamicFieldUIConfig;
