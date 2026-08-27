import React from 'react';
import { Box, Typography, Chip, Tooltip } from '@mui/material';
import StraightenIcon from '@mui/icons-material/Straighten';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import WarningAmberIcon from '@mui/icons-material/WarningAmber';

export interface PageBudgetProps {
  format?: 'A4_PORTRAIT' | 'A4_LANDSCAPE' | 'LETTER_PORTRAIT' | 'LETTER_LANDSCAPE';
  targetMaxPages?: number; // 1 for Fact Sheet, 2 for Tear Sheet, 0 for Unbounded
  totalRenderedHeightMm?: number;
  marginsMm?: { top: number; bottom: number; left: number; right: number };
}

export const PrintCanvasRuler: React.FC<PageBudgetProps> = ({
  format = 'A4_PORTRAIT',
  targetMaxPages = 1,
  totalRenderedHeightMm = 180,
  marginsMm = { top: 15, bottom: 15, left: 15, right: 15 },
}) => {
  const pageHeightMm = format.includes('A4')
    ? format === 'A4_PORTRAIT' ? 297 : 210
    : format === 'LETTER_PORTRAIT' ? 279.4 : 215.9;

  const usableHeightMm = pageHeightMm - (marginsMm.top + marginsMm.bottom);
  const calculatedPages = Math.ceil(totalRenderedHeightMm / usableHeightMm) || 1;
  const isBudgetExceeded = targetMaxPages > 0 && calculatedPages > targetMaxPages;
  const budgetUsagePct = Math.min(100, Math.round((totalRenderedHeightMm / (usableHeightMm * (targetMaxPages || 1))) * 100));

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'space-between',
        px: 2,
        py: 0.75,
        bgcolor: '#071526',
        borderBottom: '1px solid #1E293B',
        fontFamily: 'monospace',
      }}
    >
      {/* Millimeter Dimensions & Format */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        <StraightenIcon sx={{ color: '#00D4FF', fontSize: 18 }} />
        <Typography variant="caption" sx={{ color: '#E2E8F0', fontWeight: 600, fontSize: '0.72rem' }}>
          {format.replace('_', ' ')}: {usableHeightMm.toFixed(1)}mm Usable Canvas
        </Typography>
        <Typography variant="caption" sx={{ color: '#64748B', fontSize: '0.68rem' }}>
          (Margins: {marginsMm.top}mm T/B, {marginsMm.left}mm L/R)
        </Typography>
      </Box>

      {/* Page Budget Gauge */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
        <Tooltip title={`Rendered ${totalRenderedHeightMm.toFixed(1)}mm / ${usableHeightMm.toFixed(1)}mm usable height per page`}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: '0.7rem' }}>
              Height Budget:
            </Typography>
            <Box
              sx={{
                width: 70,
                height: 6,
                bgcolor: '#0B1E36',
                borderRadius: 1,
                overflow: 'hidden',
                border: '1px solid #1E293B',
              }}
            >
              <Box
                sx={{
                  width: `${budgetUsagePct}%`,
                  height: '100%',
                  bgcolor: isBudgetExceeded ? '#EF4444' : budgetUsagePct > 85 ? '#F59E0B' : '#10B981',
                  transition: 'width 0.3s ease',
                }}
              />
            </Box>
            <Typography
              variant="caption"
              sx={{
                color: isBudgetExceeded ? '#EF4444' : '#E2E8F0',
                fontWeight: 700,
                fontSize: '0.7rem',
              }}
            >
              {totalRenderedHeightMm.toFixed(1)} mm
            </Typography>
          </Box>
        </Tooltip>

        {/* Status Chip */}
        {targetMaxPages > 0 && (
          <Chip
            size="small"
            icon={isBudgetExceeded ? <WarningAmberIcon sx={{ fontSize: 13 }} /> : <CheckCircleIcon sx={{ fontSize: 13 }} />}
            label={`${calculatedPages} / ${targetMaxPages} Page Budget ${isBudgetExceeded ? 'EXCEEDED' : 'VALID'}`}
            color={isBudgetExceeded ? 'error' : 'success'}
            sx={{ fontWeight: 700, fontSize: '0.65rem', height: 20 }}
          />
        )}
      </Box>
    </Box>
  );
};

export default PrintCanvasRuler;
