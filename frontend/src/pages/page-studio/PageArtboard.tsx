import React, { useLayoutEffect, useRef, useState } from 'react';
import { Box } from '@mui/material';

interface PageArtboardProps {
  width: number;
  /** Scale the page down to fit the editor workspace without changing layout. */
  fit: boolean;
  children: React.ReactNode;
}

/**
 * A page-sized surface shared by Design and Preview so widget width/wrap
 * match the published page (or a graded breakpoint), instead of stretching
 * to whatever is left between the palette and properties panel.
 */
const PageArtboard: React.FC<PageArtboardProps> = ({ width, fit, children }) => {
  const workspaceRef = useRef<HTMLDivElement>(null);
  const pageRef = useRef<HTMLDivElement>(null);
  const [scale, setScale] = useState(1);
  const [pageHeight, setPageHeight] = useState(640);

  useLayoutEffect(() => {
    const update = () => {
      const avail = workspaceRef.current?.clientWidth ?? width;
      const h = pageRef.current?.scrollHeight ?? 640;
      setPageHeight(h);
      setScale(fit ? Math.min(1, Math.max(0.25, (avail - 32) / width)) : 1);
    };
    update();
    const ro = new ResizeObserver(update);
    if (workspaceRef.current) ro.observe(workspaceRef.current);
    if (pageRef.current) ro.observe(pageRef.current);
    window.addEventListener('resize', update);
    return () => {
      ro.disconnect();
      window.removeEventListener('resize', update);
    };
  }, [width, fit]);

  return (
    <Box
      ref={workspaceRef}
      sx={{ flex: 1, overflow: 'auto', bgcolor: '#e8edf3', p: 2 }}
    >
      <Box sx={{ width: width * scale, height: pageHeight * scale, mx: 'auto' }}>
        <Box
          ref={pageRef}
          sx={{
            width,
            minHeight: 640,
            bgcolor: 'background.paper',
            boxShadow: '0 8px 24px rgba(15, 23, 42, 0.08)',
            // Same padding as PageBrowser PageContent / DraftPreview (theme spacing 3).
            p: 3,
            transform: `scale(${scale})`,
            transformOrigin: 'top left',
          }}
        >
          {children}
        </Box>
      </Box>
    </Box>
  );
};

export default PageArtboard;
