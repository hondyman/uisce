import React from 'react';
import { Box, Typography } from '@mui/material';

/**
 * Title + slug chrome shared by Design (inside PageArtboard) and Preview /
 * PageBrowser so the first widget starts at the same vertical offset.
 * `hidden`: the page draws its own header (app.chrome = 'none').
 */
const PageBody: React.FC<{ name: string; slug: string; hidden?: boolean; children: React.ReactNode }> = ({ name, slug, hidden, children }) => (
  <Box>
    {!hidden && (
      <>
        <Typography variant="h5" sx={{ fontWeight: 800, mb: 0.5 }}>{name}</Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>/{slug}</Typography>
      </>
    )}
    {children}
  </Box>
);

export default PageBody;
