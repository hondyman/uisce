// React default import removed — component uses JSX only
import AddElementModal, { ElementKind } from './AddElementModal';
import CreateCustomModelModal, { CreateCustomModelFormData } from './CreateCustomModelModal';
// Removed RawCodeModal import - moved to CodeSidebar which was removed

interface Props {
  addModalOpen: boolean;
  pendingKind: ElementKind | null;
  pendingTargetTable: string | { id: string; qualified_path: string } | null;
  setAddModalOpen: (b: boolean) => void;
  onCreateElement: (params: { mode: 'override' | 'custom'; kind: ElementKind; coreName?: string; values: any }) => void;
  coreOptions: any[];
  libraryOptions: any[];
  existingNames: string[];
  nodes: any[];
  semanticModel: any;

  createCustomModelModalOpen: boolean;
  setCreateCustomModelModalOpen: (b: boolean) => void;
  handleCreateCustomModel: (formData: CreateCustomModelFormData) => Promise<void> | void;

  // Removed rawOpen, setRawOpen, rawFormat, setRawFormat - RawCodeModal moved to CodeSidebar which was removed
}

const ModelModals: React.FC<Props> = ({
  addModalOpen, pendingKind, pendingTargetTable, setAddModalOpen, onCreateElement, coreOptions, libraryOptions, existingNames, nodes, semanticModel,
  createCustomModelModalOpen, setCreateCustomModelModalOpen, handleCreateCustomModel,
  // Removed rawOpen, setRawOpen, rawFormat, setRawFormat - RawCodeModal moved to CodeSidebar which was removed
}) => {
  return (
    <>
      <AddElementModal
        open={addModalOpen}
        kind={pendingKind}
        targetTable={pendingTargetTable}
        onClose={() => setAddModalOpen(false)}
        onCreate={onCreateElement}
        coreOptions={coreOptions}
        libraryOptions={libraryOptions}
        existingNames={existingNames}
        nodes={nodes}
        semanticModel={semanticModel}
      />

      <CreateCustomModelModal
        open={createCustomModelModalOpen}
        onClose={() => setCreateCustomModelModalOpen(false)}
        onCreate={handleCreateCustomModel}
        nodes={nodes}
      />

      {/* Removed RawCodeModal - moved to CodeSidebar which was removed */}
    </>
  );
};

export default ModelModals;
