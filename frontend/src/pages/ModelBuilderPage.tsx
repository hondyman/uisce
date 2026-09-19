import { Box } from '@mui/material';
import { DndContext } from '@dnd-kit/core';
import SelectionStatus from '../components/SelectionStatus';
import { useTenant } from '../contexts/TenantContext';
import UnifiedSemanticBuilder from './Fabric/UnifiedSemanticBuilder';

const ModelBuilderPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  if (!tenant || !datasource) {
    return (
      <Box sx={{ p: 3 }}>
        <SelectionStatus variant="full" />
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <DndContext>
        <UnifiedSemanticBuilder
          tenantId={tenant.id}
          datasourceId={datasource.id}
          alphaDatasourceId={datasource.alpha_datasource?.id || ''}
          onClose={() => undefined}
        />
      </DndContext>
    </Box>
  );
};

export default ModelBuilderPage;
