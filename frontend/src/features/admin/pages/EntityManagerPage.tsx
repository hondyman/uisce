import React, { Suspense, lazy } from 'react';
import { Box, CircularProgress, Container } from '@mui/material';
// Lazy load EntityConfigPageV2 - this is the main entity manager view
const EntityConfigPage = lazy(() => import('../../../pages/EntityConfigPageV2'));

const EntityManagerPage: React.FC = () => {
  return (
    <Suspense fallback={
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <CircularProgress />
      </Box>
    }> 
      <Container maxWidth="lg" sx={{ mt: 2 }}>
        <EntityConfigPage />
      </Container>
    </Suspense>
  );
};

export default EntityManagerPage;
