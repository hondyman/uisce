import type { FC } from 'react';
import { Paper } from '@mui/material';
import ReportSection from './ReportSection';
import { REPORT_SECTIONS } from './reportingUtils';

const ReportCanvas: FC<any> = ({ elements, layoutSettings, selectedElement, onElementUpdate, onElementDelete, onElementSelect, orientation }) => (
  <Paper sx={{ width: orientation === 'Portrait' ? 794 : 1123, mx: 'auto', border: '1px solid #ddd' }}>
    {Object.values(REPORT_SECTIONS).map(section => (
      <ReportSection key={section} section={section} elements={elements} onElementUpdate={onElementUpdate} onElementDelete={onElementDelete} onElementSelect={onElementSelect} selectedElement={selectedElement} layoutSettings={layoutSettings} />
    ))}
  </Paper>
);

export default ReportCanvas;
