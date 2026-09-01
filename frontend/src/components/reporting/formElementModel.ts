import type { DynamicProperty } from './evaluateDynamicProperty';
import type { CellStyle } from './tableColumnModel';
import type { ConditionGroup } from '../../ExpressionBuilder/AdvancedConditionBuilder';

export interface FormReferenceElement {
  id: string;
  type: 'formReference';
  sectionId: string;
  templateId: DynamicProperty<string>;
  containerStyle: CellStyle;
  visibilityCondition?: ConditionGroup | null;
}
