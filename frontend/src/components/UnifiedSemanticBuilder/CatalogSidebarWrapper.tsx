// React default import removed — using automatic JSX runtime
import ModelCatalogSidebar from './ModelCatalogSidebar';

interface Props {
  processedCatalogModels: any[];
  modelSearchTerm: string;
  setModelSearchTerm: (s: string) => void;
  selectedModel: any;
  setSelectedModel: (m: any) => void;
  setCreateCustomModelModalOpen: (v: boolean) => void;
  onCloneModel?: (baseModelKey: string) => void;
  onDeleteModel?: (modelId: string, isCore?: boolean, modelKey?: string) => void;
  onArchiveModel?: (modelId: string, isCore?: boolean, modelKey?: string) => void;
  onPublishModel?: (modelId: string) => void;
  onDraftModel?: (modelId: string) => void;
  onRenameModel?: (modelId: string, newName: string) => void;
  modelsLoading?: boolean;
  modelsError?: string | null;
  onEnterEditMode?: () => void;
  onModelSelect?: (model: any, targetTab: 'core' | 'custom') => void;
  activeTab?: 'core' | 'custom';
  onTabChange?: (tab: 'core' | 'custom') => void;
}

const CatalogSidebarWrapper: React.FC<Props> = ({
  processedCatalogModels,
  modelSearchTerm: _modelSearchTerm,
  setModelSearchTerm: _setModelSearchTerm,
  selectedModel,
  setSelectedModel,
  setCreateCustomModelModalOpen,
  onCloneModel,
  modelsLoading,
  modelsError,
  onDeleteModel,
  onArchiveModel,
  onPublishModel,
  onDraftModel,
  onRenameModel,
  // onEnterEditMode removed (unused)
  onModelSelect,
  activeTab,
  onTabChange,
}) => {
  return (
    <div className="sidebar-container fixed">
      <aside className="catalog-sidebar">
        <ModelCatalogSidebar
          models={processedCatalogModels as any}
          selectedModel={selectedModel}
          setSelectedModel={setSelectedModel}
          onCreateCustomModel={() => setCreateCustomModelModalOpen(true)}
          onCloneModel={onCloneModel}
          onDeleteModel={onDeleteModel}
          onArchiveModel={onArchiveModel}
          onPublishModel={onPublishModel}
          onDraftModel={onDraftModel}
          onRenameModel={onRenameModel}
          onModelSelect={onModelSelect}
          loading={modelsLoading}
          error={modelsError || undefined}
          /* onEnterEditMode intentionally not passed through */
          activeTab={activeTab}
          onTabChange={onTabChange}
        />
      </aside>
    </div>
  );
};

export default CatalogSidebarWrapper;
