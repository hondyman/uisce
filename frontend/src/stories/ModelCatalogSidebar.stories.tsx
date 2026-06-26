// React JSX runtime provides automatic React import
import ModelCatalogSidebar from '../components/UnifiedSemanticBuilder/ModelCatalogSidebar';
import { ResizableBox } from 'react-resizable';
import '../components/UnifiedSemanticBuilder/ModelCatalogSidebar.css';
import './ModelCatalogSidebar.stories.css';

export default {
  title: 'UnifiedSemanticBuilder/ModelCatalogSidebar',
  component: ModelCatalogSidebar,
};

const mockModels = new Array(6).fill(0).map((_, i) => ({
  id: String(i + 1),
  model_key: `model_${i + 1}`,
  display_name: `Model ${i + 1}`,
  is_core: i % 2 === 0,
  is_custom: i % 2 === 1,
  description: i % 2 === 1 ? 'Long description '.repeat(30) : 'Short description',
  metadata: { table_count: 1 + i, measure_count: 0, dimension_count: 0 },
  status: 'draft',
  version: 1,
  can_edit: true,
  custom_model_exists: i % 2 === 1,
}));

export const Default = () => (
  <div className="story-container">
    <ResizableBox width={420} height={600} axis="x" resizeHandles={["e"]}>
      <aside className="sidebar-wrapper">
        <ModelCatalogSidebar
          models={mockModels as any}
          searchTerm=""
          setSearchTerm={() => {}}
          selectedModel={null}
          setSelectedModel={() => {}}
        />
      </aside>
    </ResizableBox>
    <div className="workspace">
      <h3>Editor / Workspace</h3>
      <p>This area simulates the workspace side.</p>
    </div>
  </div>
);
