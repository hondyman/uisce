import type { FC } from 'react';
import { useDroppable } from '@dnd-kit/core';
import {
  Box,
  Typography,
  Chip,
  IconButton,
  Tooltip,
} from '@mui/material';
import { alpha } from '@mui/material/styles';
import PrintIcon from '@mui/icons-material/Print';
import FunctionsIcon from '@mui/icons-material/Functions';
import VisibilityOffIcon from '@mui/icons-material/VisibilityOff';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ReportElement from './ReportElement';
import { REPORT_SECTIONS } from './reportingUtils';
import { evaluateCondition } from '../ExpressionBuilder/AdvancedConditionBuilder';

const getSectionHeight = (section: string) => {
  switch (section) {
    case REPORT_SECTIONS.REPORT_HEADER:
    case REPORT_SECTIONS.REPORT_FOOTER:
      return 80;
    case REPORT_SECTIONS.PAGE_HEADER:
    case REPORT_SECTIONS.PAGE_FOOTER:
      return 60;
    case REPORT_SECTIONS.GROUP_HEADER:
    case REPORT_SECTIONS.GROUP_FOOTER:
      return 70;
    case REPORT_SECTIONS.BODY:
      return 450;
    default:
      return 100;
  }
};

const getSectionLabel = (section: string) => {
  switch (section) {
    case REPORT_SECTIONS.REPORT_HEADER:
      return 'Report Header';
    case REPORT_SECTIONS.PAGE_HEADER:
      return 'Page Header';
    case REPORT_SECTIONS.GROUP_HEADER:
      return 'Group Header';
    case REPORT_SECTIONS.BODY:
      return 'Body (Detail)';
    case REPORT_SECTIONS.GROUP_FOOTER:
      return 'Group Footer';
    case REPORT_SECTIONS.PAGE_FOOTER:
      return 'Page Footer';
    case REPORT_SECTIONS.REPORT_FOOTER:
      return 'Report Footer';
    default:
      return section;
  }
};

interface ReportSectionProps {
  section: string;
  elements: any[];
  onElementUpdate: (id: string, updates: Partial<any>) => void;
  onElementDelete: (id: string) => void;
  onElementSelect: (id: string) => void;
  onElementAdd?: (element: any) => void;
  selectedElement?: string | null;
  layoutSettings: any;
  sectionConfig?: Record<string, any>;
  onSectionConfigChange?: (section: string, update: Partial<any>) => void;
  selectedSection?: string | null;
  onSectionSelect?: (section: string) => void;
  isLivePreview?: boolean;
  previewData?: any[] | null;
  onFormBlockAdd?: (section: string, payload: { mode: string; templateId: string }) => void;
}

const ReportSection: FC<ReportSectionProps> = ({
  section,
  elements,
  onElementUpdate,
  onElementDelete,
  onElementSelect,
  onElementAdd,
  selectedElement,
  layoutSettings,
  sectionConfig = {},
  onSectionConfigChange,
  selectedSection,
  onSectionSelect,
  isLivePreview = false,
  previewData = null,
  onFormBlockAdd,
  formRegistry = {},
}) => {
  const currentSectionConfig = sectionConfig[section] || {};

  const { setNodeRef, isOver } = useDroppable({
    id: section,
  });

  const isHiddenByCondition = isLivePreview && currentSectionConfig.visibilityCondition && previewData && previewData.length > 0
    ? evaluateCondition(currentSectionConfig.visibilityCondition, previewData[0])
    : false;

  const hasExpression = !!currentSectionConfig.visibilityCondition;
  const isManuallyHidden = !hasExpression && currentSectionConfig.visible === false;

  if (isHiddenByCondition) {
    return null;
  }

  const isBody = section === REPORT_SECTIONS.BODY;
  const isHeader = section === REPORT_SECTIONS.PAGE_HEADER || section === REPORT_SECTIONS.REPORT_HEADER;
  const isFooter = section === REPORT_SECTIONS.PAGE_FOOTER || section === REPORT_SECTIONS.REPORT_FOOTER;
  const isGroup = section === REPORT_SECTIONS.GROUP_HEADER || section === REPORT_SECTIONS.GROUP_FOOTER;
  const columnStyles = isBody && layoutSettings.columns > 1
    ? { columnCount: layoutSettings.columns, columnGap: `${layoutSettings.columnSpacing}px` }
    : {};
  const shouldIndicatePageBreak = layoutSettings.pageBreakBetweenRegions && section !== REPORT_SECTIONS.REPORT_HEADER;

  const sectionElements = elements.filter((el: any) => el.section === section);
  const isSelected = selectedSection === section;

  return (
    <Box sx={{ mb: 0.5, opacity: isManuallyHidden ? 0.55 : 1, transition: 'opacity 150ms ease' }}>
      {/* Section Header Bar — clickable to select */}
      <Box
        onClick={() => onSectionSelect?.(section)}
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          px: 1.5,
          py: 0.5,
          cursor: 'pointer',
          borderLeft: isSelected ? '3px solid' : '3px solid transparent',
          borderLeftColor: isSelected ? 'primary.main' : 'transparent',
          bgcolor: isManuallyHidden
            ? 'rgba(0, 0, 0, 0.02)'
            : isSelected
              ? 'action.selected'
              : isGroup
                ? 'rgba(99, 102, 241, 0.08)'
                : 'rgba(0, 0, 0, 0.04)',
          borderBottom: '1px solid',
          borderColor: 'divider',
          transition: 'border-left-color 100ms ease, background-color 100ms ease',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          <Typography variant="caption" sx={{ fontWeight: 700, letterSpacing: '0.02em', color: isManuallyHidden ? 'text.disabled' : isGroup ? 'primary.main' : 'text.primary' }}>
            {getSectionLabel(section)}
          </Typography>
          {currentSectionConfig.visibilityCondition && (
            <Tooltip title="Has conditional visibility rule">
              <Chip size="small" label="Dynamic Rule" color="warning" icon={<VisibilityOffIcon sx={{ fontSize: 12 }} />} sx={{ height: 18, fontSize: '0.65rem' }} />
            </Tooltip>
          )}
          {isManuallyHidden && (
            <Chip size="small" label="Hidden" icon={<VisibilityOffIcon sx={{ fontSize: 12 }} />} sx={{ height: 18, fontSize: '0.65rem' }} />
          )}
        </Box>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          {!isLivePreview && onSectionConfigChange && (
            <Tooltip title={
              hasExpression
                ? 'Visibility controlled by expression'
                : (currentSectionConfig.visible === false ? 'Show section' : 'Hide section')
            }>
              <IconButton
                size="small"
                onClick={(e) => {
                  e.stopPropagation();
                  onSectionSelect?.(section);
                  onSectionConfigChange(section, {
                    visible: currentSectionConfig.visible === false ? true : false,
                  });
                }}
                sx={{ p: 0.25 }}
              >
                {isHiddenByCondition
                  ? <VisibilityOffIcon sx={{ fontSize: 15, color: 'warning.main' }} />
                  : currentSectionConfig.visible === false
                    ? <VisibilityOffIcon sx={{ fontSize: 15, color: 'text.disabled' }} />
                    : <VisibilityIcon sx={{ fontSize: 15, color: 'text.secondary' }} />}
              </IconButton>
            </Tooltip>
          )}
          {shouldIndicatePageBreak && (
            <Chip
              size="small"
              label="Page Break"
              color={layoutSettings.pageBreakAfterGroup ? 'primary' : 'default'}
              icon={<PrintIcon sx={{ fontSize: 12 }} />}
              sx={{ height: 20, fontSize: '0.68rem' }}
            />
          )}
        </Box>
      </Box>

      {/* Header/Footer System Tokens */}
      {(isHeader || isFooter) && (
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, px: 1, py: 0.5, bgcolor: 'rgba(0,0,0,0.02)' }}>
          {(isHeader ? layoutSettings.headerTokens : layoutSettings.footerTokens).map((token: string) => (
            <Chip
              key={`${section}_${token}`}
              size="small"
              label={token}
              icon={<FunctionsIcon sx={{ fontSize: 12 }} />}
              sx={{ height: 20, fontSize: '0.68rem' }}
            />
          ))}
        </Box>
      )}

      {/* Section Canvas Drop Zone */}
      <Box
        ref={setNodeRef}
        onDragOver={(e) => {
          e.preventDefault();
          e.dataTransfer.dropEffect = 'copy';
        }}
        onDrop={(e) => {
          e.preventDefault();
          e.stopPropagation();
          try {
            const raw = e.dataTransfer.getData('application/json') || e.dataTransfer.getData('text/plain');
            if (!raw) return;
            const data = JSON.parse(raw);
            if (
              data?.type === 'BO_FIELD' ||
              data?.type === 'bofield' ||
              data?.type === 'bofield_batch' ||
              data?.type === 'form-block' ||
              data?.type === 'form-block-multi'
            ) {
              return;
            }
            let fields: any[] = [];
            if (data?.type === 'bo-field-bundle' && Array.isArray(data.fields)) {
              fields = data.fields;
            } else if (data?.type === 'bo-field' && data.field) {
              fields = [data.field];
            } else if (Array.isArray(data)) {
              fields = data;
            } else if (data && (data.name || data.technicalName)) {
              fields = [data];
            }

            if (fields.length > 1) {
              const tableColumns = fields.map((f: any) => f.name || f.technicalName);
              const newTable = {
                id: `table_bo_${Date.now()}`,
                type: 'table',
                section,
                position: { x: 30, y: 30 },
                size: { width: Math.min(700, Math.max(400, tableColumns.length * 110)), height: 220 },
                properties: {
                  name: `Data Table`,
                  columns: tableColumns,
                  fontSize: 11,
                  showGridLines: true,
                  alternatingRowColors: true,
                },
              };
              if (onElementAdd) onElementAdd(newTable);
              else onElementUpdate(newTable.id, newTable);
            } else if (fields.length === 1) {
              const field = fields[0];
              const newElement = {
                id: `txt_${field.name || 'field'}_${Date.now()}`,
                type: 'textbox',
                section,
                position: { x: 30 + (elements.length % 6) * 20, y: 30 + (elements.length % 6) * 20 },
                size: { width: 160, height: 40 },
                properties: {
                  text: `[${field.label || field.name}]`,
                  valueExpression: `[${field.name}]`,
                  fieldName: field.name,
                  name: field.label || field.name,
                  fontSize: 12,
                  fontWeight: 500,
                },
              };
              if (onElementAdd) onElementAdd(newElement);
              else onElementUpdate(newElement.id, newElement);
            } else if (data?.type === 'form-block' && onFormBlockAdd) {
              // Static single-form block — create a formReference element pointing at the templateId
              const newElement = {
                id: `formref_${Date.now()}`,
                type: 'formReference',
                section,
                position: { x: 30, y: 30 },
                size: { width: 600, height: 300 },
                properties: {
                  templateId: { isExpression: false, value: data.templateId },
                  containerStyle: {},
                },
              };
              if (onElementAdd) onElementAdd(newElement);
              else onElementUpdate(newElement.id, newElement);
            } else if (data?.type === 'form-block-multi' && onFormBlockAdd) {
              // Multi-form block — delegate to parent for disambiguation modal
              onFormBlockAdd(section, { mode: 'reference', templateId: data.availableTemplateIds?.[0] || '' });
            }
          } catch (err) {
            console.error('Section native onDrop error:', err);
          }
        }}
        sx={{
          position: 'relative',
          height: getSectionHeight(section),
          border: '1px solid',
          borderColor: isOver ? 'primary.main' : 'divider',
          borderStyle: isManuallyHidden ? 'dashed' : 'solid',
          bgcolor: isOver ? alpha('#6366f1', 0.1) : '#ffffff',
          backgroundImage: isManuallyHidden ? 'none' : `linear-gradient(rgba(0,0,0,.06) 1px, transparent 1px), linear-gradient(90deg, rgba(0,0,0,.06) 1px, transparent 1px)`,
          backgroundSize: '20px 20px',
          overflow: 'hidden',
          opacity: isManuallyHidden ? 0.4 : 1,
          pointerEvents: isManuallyHidden ? 'none' : 'auto',
          transition: 'opacity 150ms ease, border-style 150ms ease',
          ...columnStyles,
        }}
        aria-label={`Drop zone for ${getSectionLabel(section)}`}
      >
        {sectionElements.map((element: any) => (
          <ReportElement
            key={element.id}
            {...element}
            onUpdate={onElementUpdate}
            onDelete={onElementDelete}
            onSelect={onElementSelect}
            isSelected={selectedElement === element.id}
            isLivePreview={isLivePreview}
            previewData={previewData}
            formRegistry={formRegistry}
          />
        ))}
        {sectionElements.length === 0 && !isLivePreview && (
          <Typography
            sx={{
              position: 'absolute',
              top: '50%',
              left: '50%',
              transform: 'translate(-50%, -50%)',
              color: 'text.secondary',
              fontSize: '0.8rem',
              pointerEvents: 'none',
              opacity: 0.6,
            }}
          >
            Drop items here
          </Typography>
        )}
      </Box>
    </Box>
  );
};

export default ReportSection;
