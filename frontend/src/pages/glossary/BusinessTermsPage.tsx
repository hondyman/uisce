import React from 'react';
import { useTheme } from '@mui/material/styles';
import { BusinessTermsTab } from '../glossary/BusinessTermsTab';
import { useTenant } from '../../contexts/TenantContext';

const BusinessTermsPage: React.FC = () => {
  const { tenant } = useTenant();
  const theme = useTheme();
  return (
    <div
      className={`business-terms-root${theme.palette.mode === 'dark' ? ' dark' : ''}`}
      style={{ height: '100vh', overflow: 'auto', background: theme.palette.background.default }}
    >
      <BusinessTermsTab scopeTenantId={tenant?.id} />
    </div>
  );
};

export default BusinessTermsPage;
