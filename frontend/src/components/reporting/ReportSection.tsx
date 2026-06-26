import type { FC } from 'react';
import { useDroppable } from '@dnd-kit/core';
import { Box, Typography, Chip } from '@mui/material';
import { alpha } from '@mui/material/styles';
import { Printer, Sigma } from 'lucide-react';
import ReportElement from './ReportElement';
import { REPORT_SECTIONS } from './reportingUtils';

const getSectionHeight = (section: string) => {
  switch (section) {
    case REPORT_SECTIONS.REPORT_HEADER:
    case REPORT_SECTIONS.REPORT_FOOTER:
      return 80;
    case REPORT_SECTIONS.PAGE_HEADER:
    case REPORT_SECTIONS.PAGE_FOOTER:
      return 60;
    case REPORT_SECTIONS.BODY:
      return 400;
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
    case REPORT_SECTIONS.BODY:
      return 'Body';
    case REPORT_SECTIONS.PAGE_FOOTER:
      return 'Page Footer';
    case REPORT_SECTIONS.REPORT_FOOTER:
      return 'Report Footer';
    default:
      return section;
  }
};

const ReportSection: FC<any> = ({ section, elements, onElementUpdate, onElementDelete, onElementSelect, selectedElement, layoutSettings }) => {
  const { setNodeRef, isOver } = useDroppable({
    id: section,
  });

  const isBody = section === REPORT_SECTIONS.BODY;
  const isHeader = section === REPORT_SECTIONS.PAGE_HEADER || section === REPORT_SECTIONS.REPORT_HEADER;
  const isFooter = section === REPORT_SECTIONS.PAGE_FOOTER || section === REPORT_SECTIONS.REPORT_FOOTER;
  const columnStyles = isBody && layoutSettings.columns > 1 ? { columnCount: layoutSettings.columns, columnGap: `${layoutSettings.columnSpacing}px` } : {};
  const shouldIndicatePageBreak = layoutSettings.pageBreakBetweenRegions && section !== REPORT_SECTIONS.REPORT_HEADER;

  return (
    <Box>
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', p: 0.5, bgcolor: '#f0f0f0', borderBottom: '1px solid #ddd' }}>
        <Typography variant="caption" sx={{ fontWeight: 600 }}>{getSectionLabel(section)}</Typography>
        {shouldIndicatePageBreak && (
          <Chip size="small" label="Page Break" color={layoutSettings.pageBreakAfterGroup ? 'primary' : 'default'} icon={<Printer size={12} />} />
        )}
      </Box>
      {(isHeader || isFooter) && (
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, p: 0.5 }}>
          {(isHeader ? layoutSettings.headerTokens : layoutSettings.footerTokens).map((token: string) => (
            <Box key={`${section}_${token}`}>
              <Chip size="small" label={token} icon={<Sigma size={12} />} />
            </Box>
          ))}
        </Box>
      )}
      <Box ref={setNodeRef} sx={{ position: 'relative', height: getSectionHeight(section), border: '1px solid #ddd', bgcolor: isOver ? alpha('#6366f1', 0.1) : '#ffffff', backgroundImage: `linear-gradient(rgba(0,0,0,.1) 1px, transparent 1px), linear-gradient(90deg, rgba(0,0,0,.1) 1px, transparent 1px)`, backgroundSize: '20px 20px', overflow: 'hidden', ...columnStyles }} aria-label={`Drop zone for ${getSectionLabel(section)}`}>
        {elements.filter((el: any) => el.section === section).map((element: any) => (
          <ReportElement key={element.id} {...element} onUpdate={onElementUpdate} onDelete={onElementDelete} onSelect={onElementSelect} isSelected={selectedElement === element.id} />
        ))}
        {elements.filter((el: any) => el.section === section).length === 0 && (
          <Typography sx={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', color: 'text.secondary', pointerEvents: 'none' }}>
            Drop items here
          </Typography>
        )}
      </Box>
    </Box>
  );
};

export default ReportSection;
