import React from 'react';
import { Box, Paper, Typography, IconButton, Tooltip } from '@mui/material';
import {
  Visibility,
  VisibilityOff,
  Code,
  DragIndicator,
  ViewColumn,
  Delete,
} from '@mui/icons-material';
import type { AdvancedReportSection } from './sectionLayoutModel';

interface ReportSectionContainerProps {
  section: AdvancedReportSection;
  isLivePreview: boolean;
  onUpdateSection: (id: string, patch: Partial<AdvancedReportSection>) => void;
  onAddSubSection: (parentId: string) => void;
  children: React.ReactNode;
}

export const ReportSectionContainer: React.FC<ReportSectionContainerProps> = ({
  section,
  isLivePreview,
  onUpdateSection,
  onAddSubSection,
  children,
}) => {
  const { headerConfig, dimensions, flow } = section;

  return (
    <Box
      sx={{
        width: dimensions.flexGrow ? `${dimensions.flexGrow * 100}%` : dimensions.widthPx || '100%',
        minHeight: dimensions.minHeightPx || 60,
        display: 'flex',
        flexDirection: 'column',
        border: '1px solid #1E293B',
        bgcolor: '#050D1A',
        mb: 1.5,
      }}
    >
      {/* Section Header — toggleable, removable */}
      {headerConfig.showHeader && !isLivePreview && (
        <Box
          sx={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            px: 1.5,
            py: 0.5,
            bgcolor: headerConfig.backgroundColor || '#071526',
            borderBottom: '1px solid #1E293B',
          }}
        >
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <DragIndicator sx={{ fontSize: 14, color: '#64748B', cursor: 'grab' }} />
            <Typography
              variant="caption"
              sx={{ fontWeight: 700, color: headerConfig.textColor || '#38BDF8' }}
            >
              {headerConfig.title}
            </Typography>
          </Box>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            <Tooltip title="Split Side-by-Side">
              <IconButton
                size="small"
                onClick={() => onAddSubSection(section.id)}
                sx={{ color: '#64748B', p: 0.4 }}
              >
                <ViewColumn sx={{ fontSize: 14 }} />
              </IconButton>
            </Tooltip>

            <Tooltip title={headerConfig.isCollapsed ? 'Show Section' : 'Hide Section'}>
              <IconButton
                size="small"
                onClick={() =>
                  onUpdateSection(section.id, {
                    headerConfig: { ...headerConfig, isCollapsed: !headerConfig.isCollapsed },
                  })
                }
                sx={{ color: '#64748B', p: 0.4 }}
              >
                {headerConfig.isCollapsed ? (
                  <VisibilityOff sx={{ fontSize: 14 }} />
                ) : (
                  <Visibility sx={{ fontSize: 14 }} />
                )}
              </IconButton>
            </Tooltip>

            <Tooltip title="Remove Header">
              <IconButton
                size="small"
                onClick={() =>
                  onUpdateSection(section.id, {
                    headerConfig: { ...headerConfig, showHeader: false },
                  })
                }
                sx={{ color: '#64748B', p: 0.4 }}
              >
                <Delete sx={{ fontSize: 14 }} />
              </IconButton>
            </Tooltip>
          </Box>
        </Box>
      )}

      {/* Collapsed indicator */}
      {headerConfig.isCollapsed && headerConfig.showHeader && !isLivePreview && (
        <Box sx={{ px: 2, py: 0.5, bgcolor: '#071526' }}>
          <Typography variant="caption" sx={{ color: '#475569', fontStyle: 'italic' }}>
            {headerConfig.title} — collapsed
          </Typography>
        </Box>
      )}

      {/* Section Content */}
      {!headerConfig.isCollapsed && (
        <Box
          sx={{
            display: 'flex',
            flexDirection: flow === 'ROW' ? 'row' : 'column',
            flex: 1,
            position: 'relative',
          }}
        >
          {children}
        </Box>
      )}
    </Box>
  );
};
