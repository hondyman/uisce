import React from 'react';
import { useTheme } from '@mui/material/styles';
import { useTenant } from '../../../contexts/TenantContext';

const RoleManagerPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const { mode } = useTheme();

  if (!tenant || !datasource) {
    return (
      <Box sx={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: mode === 'dark' ? '#111827' : '#f9fafb', color: mode === 'dark' ? '#f9fafb' : '#111827' }}>
        <div sx={{ textAlign: 'center' }}>
          <h2 sx={{ fontSize: '2xl', fontWeight: 700, mb: 4, color: mode === 'dark' ? '#f9fafb' : '#111827' }}>
            No Tenant/Datasource Selected
          </h2>
          <p sx={{ color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>
            Please select a tenant and datasource to manage roles and permissions.
          </p>
        </div>
      </Box>
    );
  }

  return <></>;
};

export default RoleManagerPage;