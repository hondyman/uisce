import { FC } from 'react';
import { Box, Container } from '@mui/material';
import WealthManagementDashboard from '../features/wealth-management/WealthManagementDashboard';

interface WealthManagementPageProps {
  tenantId: string;
  clientId?: string;
}

const WealthManagementPage: FC<WealthManagementPageProps> = ({
  tenantId,
  clientId
}) => {
  return (
    <Container maxWidth="xl">
      <Box sx={{ py: 2 }}>
        <WealthManagementDashboard
          tenantId={tenantId}
          clientId={clientId}
        />
      </Box>
    </Container>
  );
};

export default WealthManagementPage;
