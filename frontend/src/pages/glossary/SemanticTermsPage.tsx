import React from 'react';
import { useTheme } from '@mui/material/styles';
import { SemanticTermsTab } from '../glossary/SemanticTermsTab';
import { useTenant } from '../../contexts/TenantContext';

const SemanticTermsPage: React.FC = () => {
  const theme = useTheme();
  const { tenant } = useTenant();
  return (
    <div
      className={`business-terms-root${theme.palette.mode === 'dark' ? ' dark' : ''}`}
      style={{ height: '100vh', overflow: 'auto', background: theme.palette.background.default }}
    >
      <SemanticTermsTab scopeTenantId={tenant?.id} />
    </div>
  );
};

export default SemanticTermsPage;
