import { Box } from '@mui/material';
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
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
      <DndProvider backend={HTML5Backend}>
        <UnifiedSemanticBuilder
          tenantId={tenant.id}
          datasourceId={datasource.id}
          alphaDatasourceId={datasource.alpha_datasource?.id || ''}
          onClose={() => undefined}
        />
      </DndProvider>
    </Box>
  );
};

export default ModelBuilderPage;
