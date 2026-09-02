import React from 'react';
import { Box, Typography } from '@mui/material';
import { useTenant } from '../../../contexts/TenantContext';
import { FieldPermissionEditor } from '../../../components/RBAC/FieldPermissionEditor';

const FieldPermissionEditorPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  if (!tenant || !datasource) {
    return (
      <Box sx={{ minHeight: '60vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Box sx={{ textAlign: 'center' }}>
          <Typography variant="h5" fontWeight={700} gutterBottom>
            No Tenant/Datasource Selected
          </Typography>
          <Typography color="text.secondary">
            Please select a tenant and datasource to configure field permissions.
          </Typography>
        </Box>
      </Box>
    );
  }

  return <FieldPermissionEditor tenant={tenant} datasource={datasource} />;
};

export default FieldPermissionEditorPage;
