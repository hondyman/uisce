import React from 'react';
import { Box, Typography } from '@mui/material';
import { useTenant } from '../../../contexts/TenantContext';
import { RoleManagerMasterDetail } from '../../../components/RBAC/RoleManager_MasterDetail';

const RoleManagerPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  if (!tenant || !datasource) {
    return (
      <Box sx={{ minHeight: '60vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Box sx={{ textAlign: 'center' }}>
          <Typography variant="h5" fontWeight={700} gutterBottom>
            No Tenant/Datasource Selected
          </Typography>
          <Typography color="text.secondary">
            Please select a tenant and datasource to manage roles and permissions.
          </Typography>
        </Box>
      </Box>
    );
  }

  return <RoleManagerMasterDetail tenant={tenant} datasource={datasource} />;
};

export default RoleManagerPage;
