import { ConditionGroup } from '../reporting/AdvancedConditionBuilder';
import type { DynamicProperty } from '../reporting/evaluateDynamicProperty';

export type { DynamicProperty };

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
