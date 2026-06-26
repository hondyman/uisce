// React JSX runtime provides automatic React import
import { DndProvider } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import UnifiedSemanticBuilder from './UnifiedSemanticBuilder';

interface SemanticModelModalProps {
  tenantId: string;
  datasourceId: string;
  alphaDatasourceId: string;
  onClose: () => void;
}

const SemanticModelModal: React.FC<SemanticModelModalProps> = ({ 
  tenantId,
  datasourceId, 
  alphaDatasourceId,
  onClose 
}) => {
  return (
    <DndProvider backend={HTML5Backend}>
      <UnifiedSemanticBuilder 
        tenantId={tenantId}
        datasourceId={datasourceId} 
        alphaDatasourceId={alphaDatasourceId}
        onClose={onClose} 
      />
    </DndProvider>
  );
};

export default SemanticModelModal;