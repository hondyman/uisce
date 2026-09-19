import React, { useState } from 'react';
import { Box, IconButton, Tooltip } from '@mui/material';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import type { PanelNodeProps } from '../../types/pageStudio';

interface PanelRegionProps extends PanelNodeProps {
  children: React.ReactNode;
}

/**
 * Chrome for a Panel layout node: a side region that slides open/closed by
 * animating its width, collapsing down to a thin strip with a single toggle
 * button when closed. Shared by LayoutCanvas.tsx (editing), DraftPreview.tsx
 * and PageBrowser.tsx (read-only) so the open/collapse behavior a page
 * author sees in the designer is exactly what a real viewer gets - built
 * once here instead of three times to avoid those diverging.
 */
const PanelRegion: React.FC<PanelRegionProps> = ({ side, collapsible, defaultOpen, widthPx, label, children }) => {
  const [open, setOpen] = useState(defaultOpen);
  const isLeft = side === 'left';
  const ToggleIcon = open ? (isLeft ? ChevronLeftIcon : ChevronRightIcon) : (isLeft ? ChevronRightIcon : ChevronLeftIcon);

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: isLeft ? 'row' : 'row-reverse',
        alignItems: 'stretch',
        flex: '0 0 auto',
      }}
    >
      {collapsible && (
        <Box sx={{ display: 'flex', alignItems: 'flex-start', pt: 1 }}>
          <Tooltip title={open ? `Collapse ${label || 'panel'}` : `Expand ${label || 'panel'}`}>
            <IconButton size="small" onClick={() => setOpen((o) => !o)}>
              <ToggleIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      )}
      <Box
        sx={{
          width: open ? widthPx : 0,
          minWidth: open ? widthPx : 0,
          overflow: 'hidden',
          transition: 'width 0.22s ease, min-width 0.22s ease',
        }}
      >
        <Box sx={{ width: widthPx }}>{children}</Box>
      </Box>
    </Box>
  );
};

export default PanelRegion;
