import type { DynamicProperty } from '../evaluateDynamicProperty';
import type { CellStyle, BorderSide } from './tableColumnModel';
import type { ConditionGroup } from '../../ExpressionBuilder/AdvancedConditionBuilder';

export type SectionFlow = 'ROW' | 'COLUMN';

export interface SectionDimension {
  widthPx?: number;
  flexGrow?: number;
  minWidthPx?: number;
  heightPx?: number;
  minHeightPx?: number;
}

export interface SectionHeaderConfig {
  showHeader: boolean;
  title: string;
  backgroundColor?: string;
  textColor?: string;
  isCollapsible?: boolean;
  isCollapsed?: boolean;
  showToggleEye?: boolean;
  showRuleIcon?: boolean;
}

export interface AdvancedReportSection {
  id: string;
  parentId?: string | null;
  sectionType: 'REPORT_HEADER' | 'PAGE_HEADER' | 'GROUP_HEADER' | 'BODY' | 'GROUP_FOOTER' | 'PAGE_FOOTER' | 'REPORT_FOOTER' | 'CUSTOM_CONTAINER';
  flow: SectionFlow;
  dimensions: SectionDimension;
  headerConfig: SectionHeaderConfig;
  elements: string[];
  subSections?: AdvancedReportSection[];
  visibilityCondition?: ConditionGroup | null;
}

export function defaultSectionLayout(
  sectionType: AdvancedReportSection['sectionType']
): AdvancedReportSection {
  const heights: Record<string, number> = {
    REPORT_HEADER: 80,
    REPORT_FOOTER: 80,
    PAGE_HEADER: 60,
    PAGE_FOOTER: 60,
    GROUP_HEADER: 70,
    GROUP_FOOTER: 70,
    BODY: 450,
    CUSTOM_CONTAINER: 120,
  };
  return {
    id: `layout_${sectionType.toLowerCase()}_${Date.now()}`,
    sectionType,
    flow: 'COLUMN',
    dimensions: {
      heightPx: heights[sectionType] ?? 120,
      minHeightPx: 60,
    },
    headerConfig: {
      showHeader: true,
      title: sectionType.replace(/_/g, ' '),
      isCollapsible: false,
      isCollapsed: false,
      showToggleEye: true,
      showRuleIcon: false,
    },
    elements: [],
  };
}
