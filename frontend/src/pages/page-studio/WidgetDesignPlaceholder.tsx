import React from 'react';
import { Box, Typography } from '@mui/material';
import TableChartIcon from '@mui/icons-material/TableChart';
import BarChartIcon from '@mui/icons-material/BarChart';
import ShowChartIcon from '@mui/icons-material/ShowChart';
import PieChartIcon from '@mui/icons-material/PieChart';
import SpeedIcon from '@mui/icons-material/Speed';
import FilterAltIcon from '@mui/icons-material/FilterAlt';

/**
 * Structure-only stand-ins for every widget that normally shows live data
 * (Table, LineChart, KPIGroup, Slicer) - used in Page Studio's Design mode
 * (see PageComponentRenderer.tsx's `mode` prop) so the canvas never fetches
 * or displays a real record, the same principle FormFieldsDesigner.tsx
 * already applies to Form. Preview mode (DraftPreview, the published page,
 * PageBrowser) always renders the real, data-bound widget - these
 * placeholders are Design-mode only.
 */

const SKELETON_BG = 'rgba(0,0,0,0.08)';

export const TableDesignPlaceholder: React.FC<{ title?: string; columns: string[] }> = ({ title, columns }) => {
  const cols = columns.length > 0 ? columns : ['Column A', 'Column B', 'Column C'];
  return (
    <Box sx={{ p: 1.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mb: 1 }}>
        <TableChartIcon fontSize="small" color="action" />
        <Typography variant="subtitle2" fontWeight={700}>{title || 'Table'}</Typography>
      </Box>
      <Box sx={{ border: '1px solid rgba(0,0,0,0.08)', borderRadius: 1, overflow: 'hidden' }}>
        <Box sx={{ display: 'flex', bgcolor: 'rgba(0,0,0,0.03)', borderBottom: '1px solid rgba(0,0,0,0.08)' }}>
          {cols.map((c) => (
            <Typography key={c} variant="caption" fontWeight={700} sx={{ flex: 1, p: 1, borderRight: '1px solid rgba(0,0,0,0.05)' }} noWrap>
              {c}
            </Typography>
          ))}
        </Box>
        {[0, 1, 2].map((row) => (
          <Box key={row} sx={{ display: 'flex', borderBottom: row < 2 ? '1px solid rgba(0,0,0,0.05)' : undefined }}>
            {cols.map((c) => (
              <Box key={c} sx={{ flex: 1, p: 1, borderRight: '1px solid rgba(0,0,0,0.03)' }}>
                <Box sx={{ height: 10, width: `${50 + ((row * 17) % 40)}%`, bgcolor: SKELETON_BG, borderRadius: 0.5 }} />
              </Box>
            ))}
          </Box>
        ))}
      </Box>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
        Design view - switch to Preview to see real rows.
      </Typography>
    </Box>
  );
};

const KIND_ICON: Record<string, React.ElementType> = {
  chart: ShowChartIcon,
  bar: BarChartIcon,
  pie: PieChartIcon,
  gauge: SpeedIcon,
  slicer: FilterAltIcon,
};

export const ChartDesignPlaceholder: React.FC<{
  kind: 'chart' | 'gauge' | 'slicer';
  chartType?: 'bar' | 'line' | 'pie';
  title?: string;
  dimensions: string[];
  measures: string[];
}> = ({ kind, chartType, title, dimensions, measures }) => {
  const Icon = KIND_ICON[chartType && kind === 'chart' ? chartType : kind] || ShowChartIcon;
  const fields = [...dimensions, ...measures];
  return (
    <Box sx={{ p: 1.5 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75, mb: 1 }}>
        <Icon fontSize="small" color="action" />
        <Typography variant="subtitle2" fontWeight={700}>{title || kind}</Typography>
      </Box>
      {kind === 'slicer' ? (
        <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
          {['Value A', 'Value B', 'Value C'].map((v) => (
            <Box key={v} sx={{ px: 1.5, py: 0.5, borderRadius: 4, border: '1px dashed rgba(0,0,0,0.2)', bgcolor: SKELETON_BG }}>
              <Typography variant="caption" color="text.secondary">{v}</Typography>
            </Box>
          ))}
        </Box>
      ) : kind === 'gauge' ? (
        <Box sx={{ textAlign: 'center', py: 1.5 }}>
          <Box sx={{ height: 28, width: 100, bgcolor: SKELETON_BG, borderRadius: 1, mx: 'auto' }} />
        </Box>
      ) : (
        <Box sx={{ display: 'flex', alignItems: 'flex-end', gap: 1, height: 90, px: 1 }}>
          {[40, 70, 50, 85, 60].map((h, i) => (
            <Box key={i} sx={{ flex: 1, height: `${h}%`, bgcolor: SKELETON_BG, borderRadius: '2px 2px 0 0' }} />
          ))}
        </Box>
      )}
      {fields.length > 0 && (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1 }}>
          {fields.join(', ')} - design view, switch to Preview to see real values.
        </Typography>
      )}
    </Box>
  );
};
