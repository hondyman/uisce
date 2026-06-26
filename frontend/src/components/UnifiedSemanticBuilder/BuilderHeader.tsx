// React default import removed — using automatic JSX runtime
import type { ModelCatalogNode } from '../../types/model';
import HeaderBranding from './HeaderBranding';
import HeaderCenter from './HeaderCenter';
import HeaderActions from './HeaderActions';

interface BuilderHeaderProps {
  modelName: string;
  setModelName: (name: string) => void;
  selectedModel: ModelCatalogNode | null;
  handleSave: () => Promise<void> | void;
  isSaving: boolean;
}

const BuilderHeader: React.FC<BuilderHeaderProps> = ({ modelName, setModelName, selectedModel, handleSave, isSaving }) => {
  return (
    <header className="builder-header">
      <div className="header-left">
        <HeaderBranding />
      </div>

      <HeaderCenter modelName={modelName} setModelName={setModelName} selectedModelNameReadonly={Boolean(selectedModel)} />

      <HeaderActions handleSave={handleSave} isSaving={isSaving} />
    </header>
  );
};

export default BuilderHeader;
