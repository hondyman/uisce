import type { FC } from 'react';
import { useMemo } from 'react';
import { Paper, Box } from '@mui/material';
import ReportSection from './ReportSection';
import { ReportSectionContainer } from './ReportSectionContainer';
import PrintCanvasRuler from './PrintCanvasRuler';
import { REPORT_SECTIONS } from './reportingUtils';
import type { AdvancedReportSection } from './sectionLayoutModel';
import { defaultSectionLayout } from './sectionLayoutModel';

interface ReportCanvasProps {
  elements: any[];
  layoutSettings: any;
  selectedElement: string | null;
  onElementUpdate: (id: string, updates: Partial<any>) => void;
  onElementDelete: (id: string) => void;
  onElementSelect: (id: string) => void;
  onElementAdd?: (element: any) => void;
  onElementDuplicate?: (id: string) => void;
  onLayoutSettingsChange?: (key: string, value: any) => void;
  sectionConfig?: Record<string, any>;
  onSectionConfigChange?: (section: string, update: Partial<any>) => void;
  selectedSection?: string | null;
  onSectionSelect?: (section: string) => void;
  orientation?: 'Portrait' | 'Landscape';
  isLivePreview?: boolean;
  previewData?: any[] | null;
  availableFieldDefs?: any[];
  formRegistry?: Record<string, any>;
  onFormBlockAdd?: (section: string, payload: { mode: string; templateId: string }) => void;
  layoutSections?: AdvancedReportSection[];
  onUpdateSectionLayout?: (id: string, patch: Partial<AdvancedReportSection>) => void;
  onAddSubSection?: (parentId: string) => void;
}

const ReportCanvas: FC<ReportCanvasProps> = ({
  elements,
  layoutSettings,
  selectedElement,
  onElementUpdate,
  onElementDelete,
  onElementSelect,
  onElementAdd,
  sectionConfig,
  onSectionConfigChange,
  selectedSection,
  onSectionSelect,
  orientation = 'Portrait',
  isLivePreview = false,
  previewData = null,
  availableFieldDefs = [],
  formRegistry = {},
  onFormBlockAdd,
  layoutSections = [],
  onUpdateSectionLayout,
  onAddSubSection,
}) => {
  const totalRenderedHeightMm = Math.max(120, elements.length * 35 + 40);

  const layoutMap = useMemo(() => {
    const m = new Map<string, AdvancedReportSection>();
    layoutSections.forEach((sec) => m.set(sec.sectionType, sec));
    return m;
  }, [layoutSections]);

  return (
    <Box sx={{ width: orientation === 'Portrait' ? 794 : 1123, mx: 'auto' }}>
      <PrintCanvasRuler
        format={orientation === 'Portrait' ? 'A4_PORTRAIT' : 'A4_LANDSCAPE'}
        targetMaxPages={1}
        totalRenderedHeightMm={totalRenderedHeightMm}
        marginsMm={{ top: 15, bottom: 15, left: 15, right: 15 }}
      />
      <Paper sx={{ width: '100%', border: '1px solid #ddd', bgcolor: '#FFFFFF', borderRadius: '0 0 4px 4px' }}>
        {Object.values(REPORT_SECTIONS).map((secType) => {
          const layout = layoutMap.get(secType) || defaultSectionLayout(secType as any);
          return (
            <ReportSectionContainer
              key={secType}
              section={{ ...layout, headerConfig: { ...layout.headerConfig, showHeader: false } }}
              isLivePreview={isLivePreview}
              onUpdateSection={onUpdateSectionLayout!}
              onAddSubSection={onAddSubSection!}
            >
              <ReportSection
                section={secType}
                elements={elements}
                onElementUpdate={onElementUpdate}
                onElementDelete={onElementDelete}
                onElementSelect={onElementSelect}
                onElementAdd={onElementAdd}
                selectedElement={selectedElement}
                layoutSettings={layoutSettings}
                sectionConfig={sectionConfig}
                onSectionConfigChange={onSectionConfigChange}
                selectedSection={selectedSection}
                onSectionSelect={onSectionSelect}
                isLivePreview={isLivePreview}
                previewData={previewData}
                availableFieldDefs={availableFieldDefs}
                formRegistry={formRegistry}
                onFormBlockAdd={onFormBlockAdd}
              />
            </ReportSectionContainer>
          );
        })}
      </Paper>
    </Box>
  );
};

export default ReportCanvas;
