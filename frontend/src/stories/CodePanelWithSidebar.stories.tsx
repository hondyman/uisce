// React import removed (automatic JSX runtime)
import ModelCatalogSidebar from '../components/UnifiedSemanticBuilder/ModelCatalogSidebar';
import { CodePanel } from '../components/UnifiedSemanticBuilder/CodePanel';
import '../components/UnifiedSemanticBuilder/ModelCatalogSidebar.css';
import '../components/UnifiedSemanticBuilder/CodePanel.css';
import './ModelCatalogSidebar.stories.css';
import './CodePanelWithSidebar.stories.css';
import { devLog } from '../utils/devLogger';

export default {
  title: 'UnifiedSemanticBuilder/CodePanelWithSidebar',
  component: CodePanel,
};

const mockModels = [
  {
    id: '1',
    model_key: 'core_model',
    display_name: 'Core Model',
    is_core: true,
    is_custom: false,
    description: 'This is a core model',
    metadata: { table_count: 2, measure_count: 1, dimension_count: 3 },
    status: 'published',
    version: 1,
    can_edit: false,
    custom_model_exists: false,
    created_at: null,
    updated_at: null,
  },
  {
    id: '2',
    model_key: 'core_model_custom',
    display_name: 'Custom Model',
    is_core: false,
    is_custom: true,
    parent_model_key: 'core_model',
    description: 'A custom model',
    metadata: { table_count: 1, measure_count: 1, dimension_count: 1 },
    status: 'draft',
    version: 1,
    can_edit: true,
    custom_model_exists: true,
    created_at: null,
    updated_at: null,
  }
];

export const Default = () => {
  const generateJSON = () => JSON.stringify({ dimensions: [{ id: 'd1' }], measures: [{ id: 'm1' }] }, null, 2);
  const generateYAML = () => 'dimensions:\n  - id: d1\nmeasures:\n  - id: m1\njoins:\n  - name: j1\nfilters:\n  - name: f1';

  const createCustomModel = async (baseModelKey: string) => {
    // simple mock: push to console
    devLog('create custom model', baseModelKey);
  };

  return (
    <div className="story-row">
      <div className="story-left">
        <ModelCatalogSidebar
          models={mockModels as any}
          searchTerm={''}
          setSearchTerm={() => {}}
          selectedModel={mockModels[0] as any}
          setSelectedModel={() => {}}
          onCreateCustomModel={createCustomModel}
        />
      </div>

      <div className="story-right">
        <CodePanel
          showCode={'yaml'}
          modelName={'core_model'}
          searchTerm={''}
          setMatchIndex={() => {}}
          setMatchCount={() => {}}
          generateJSON={generateJSON}
          generateYAML={generateYAML}
          codeEditable={true}
          onImportCode={async () => {}}
          extendsModel={'core_model'}
        />
      </div>
    </div>
  );
};
