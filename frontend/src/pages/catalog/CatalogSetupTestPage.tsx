import type React from 'react';
import { Box, Typography } from '@mui/material';

export const CatalogSetupTestPage: React.FC = () => {
  return (
    <Box sx={{ p: 4 }}>
      <Typography variant="h3">Catalog Setup Test Page</Typography>
      <Typography variant="body1" sx={{ mt: 2 }}>
        If you can see this, the route is working!
      </Typography>
      <Box sx={{ mt: 4, p: 3, bgcolor: 'info.light', borderRadius: 2 }}>
        <Typography variant="h6">Debugging Info:</Typography>
        <Typography>- Route: /core/catalog-setup</Typography>
        <Typography>- Component: CatalogSetupPage</Typography>
        <Typography>- Tabs should appear above</Typography>
      </Box>
    </Box>
  );
};
