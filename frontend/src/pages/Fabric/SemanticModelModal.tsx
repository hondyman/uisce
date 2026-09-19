// React JSX runtime provides automatic React import
import { DndContext } from '@dnd-kit/core';
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
    <DndContext>
      <UnifiedSemanticBuilder
        tenantId={tenantId}
        datasourceId={datasourceId}
        alphaDatasourceId={alphaDatasourceId}
        onClose={onClose}
      />
    </DndContext>
  );
};

export default SemanticModelModal;