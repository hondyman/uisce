import React, { useState } from 'react';
import { Box } from '@mui/material';
import SemanticMapper from '../../../components/SemanticMapper';
import { SemanticMappingWizard } from '../../../components/SemanticMappingWizard';
import { useTenant } from '../../../contexts/TenantContext';

export default function SemanticMapperPage() {
  const [showWizard, setShowWizard] = useState(false);
  
  // Get tenant and datasource from context
  const { tenant, datasource } = useTenant();
  
  const tenantId = tenant?.id || '';
  const datasourceId = datasource?.id || '';

  if (showWizard) {
    return (
      <Box sx={{ p: 3 }}>
        <SemanticMappingWizard
          tenantId={tenantId}
          datasourceId={datasourceId}
          onClose={() => setShowWizard(false)}
        />
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column', p: 3 }}>
      <SemanticMapper onOpenWizard={() => setShowWizard(true)} />
    </Box>
  );
}
