// React import removed (unused)
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { vi } from 'vitest';
import UnifiedSemanticBuilder from '../../../pages/Fabric/UnifiedSemanticBuilder';

// Mock hooks used by the page so the component can mount without Apollo/Auth providers
const mockSetSemanticModel = vi.fn();
vi.mock('../../../hooks/useUnifiedSemanticBuilder', () => {
  // React require removed (automatic JSX runtime)
  return {
    useUnifiedSemanticBuilder: (_datasourceId: string) => {
      const semanticModel = { name: 'semantic_model', dimensions: [], measures: [], filters: [], joins: [] };
      const showCode = 'json';
      return {
        nodes: [],
        searchTerm: '',
        setSearchTerm: () => {},
        selectedColumn: null,
        setSelectedColumn: () => {},
        modelName: 'semantic_model',
        setModelName: () => {},
        showCode,
        setShowCode: () => {},
        businessTerms: [],
        semanticTerms: [],
        semanticViews: [],
        semanticModel,
        setSemanticModel: mockSetSemanticModel,
        columnMappings: new Map(),
        setColumnMappings: () => {},
        chartLoading: false,
        chartError: null,
        businessLoading: false,
        businessError: null,
        isNumericType: () => false,
        getColumnMapping: () => null,
        getMappingColor: () => 'transparent',
        getBusinessTermForColumn: () => undefined,
        addDimension: () => {},
        addMeasure: () => {},
        addFilter: () => {},
        removeSemanticElement: () => {},
        toggleElementEdit: () => {},
        updateSemanticElement: () => {},
        saveSemanticModel: async () => {},
        generateJSON: () => JSON.stringify(semanticModel, null, 2),
        generateYAML: () => '',
        filteredNodes: [],
        showNotification: () => {}
      };
    }
  };
});

// Mock other hooks used to avoid side-effects
vi.mock('../../../hooks/useModelCatalog', () => ({ useModelCatalog: () => ({ models: [], selectedModel: null, setSelectedModel: () => {}, searchTerm: '', setSearchTerm: () => {}, loading: false, error: null, createCustomModel: async () => ({}), cloneModel: async () => ({}), refreshModels: async () => {}, updateModel: async () => {}, deleteModel: async () => ({ success: true }) }) }));
vi.mock('../../../hooks/useShowCodeSync', () => ({ default: () => ({ formatType: 'json', setFormatType: () => {} }) }));
vi.mock('../../../hooks/usePaletteDrop', () => ({ default: () => ({ isOver: false, drop: () => {} }) }));
vi.mock('../../../hooks/useBuilderGenerators', () => ({ useBuilderGenerators: () => ({ generateCustomModelObject: () => ({}), generateMergedModelObject: () => ({}), generateJSON: () => '{}', generateYAML: () => '', generateCoreJSON: () => '{}', generateCoreYAML: () => '', generateCustomJSON: () => '{}', generateCustomYAML: () => '' }) }));
vi.mock('../../../hooks/useModelSaver', () => ({ default: () => ({ isSaving: false, handleSave: async () => {} }) }));
vi.mock('../../../hooks/useEnsureCustomAndAdd', () => ({ default: () => ({ ensureCustomAndApply: async (cb: any) => cb(), wrapAdd: (fn: any) => fn, enhancedRemove: (fn: any) => fn }) }));
vi.mock('../../../hooks/useCompatibility', () => ({ default: () => ({ compat: null, compatErr: null, compatLoading: false, refreshCompatibility: async () => {}, filteredCompat: [] }) }));
vi.mock('../../../hooks/useCoreModelBuilder', () => ({ useCoreModelBuilder: () => {} }));
vi.mock('../../../hooks/useModelCreator', () => ({ default: () => ({ handleCreateCustomModel: async () => {} }) }));
vi.mock('../../../hooks/useElementCreator', () => ({ default: () => ({ handleCreateElement: async () => {} }) }));
vi.mock('../../../hooks/useClipboard', () => ({ default: () => ({ copyToClipboard: () => {} }) }));
vi.mock('../../../contexts/AuthContext', () => ({ useAuth: () => ({
  user: null,
  token: null,
  refreshTokenValue: null,
  tokenExpiresAt: null,
  isAuthenticated: false,
  isLoading: false,
  isAdmin: () => false,
  login: async () => {},
  register: async () => {},
  forgotPassword: async () => {},
  resetPassword: async () => {},
  logout: async () => {},
  refreshToken: async () => {},
  isTokenExpired: () => false,
  getValidToken: async () => 'test-token'
}) }));

// This test uses a minimal render of the page and simulates editing the code
// in the editor then clicking Apply to ensure the canvas updates.
test('editing code and applying updates tile labels', async () => {
  // Render the builder with minimal required props
  const onClose = () => {};
  render(
    <MemoryRouter>
      <UnifiedSemanticBuilder tenantId="t1" datasourceId="ds1" onClose={onClose} />
    </MemoryRouter>
  );

  // Wait for CodePanel to mount by toggling code view via keyboard event or open code event
  // The page by default may not show code; dispatch openCode to ensure CodePanel appears
  window.dispatchEvent(new CustomEvent('semlayer.openCode'));

  // Find the textarea inside the editor
  const ta = await waitFor(() => screen.getByLabelText(/JSON code editor|YAML code editor|CODE code editor/i));
  expect(ta).toBeTruthy();

  // Replace the content with a simple JSON containing one measure and apply
  const newJson = JSON.stringify({ name: 'test_model', measures: [{ id: 'm1', name: 'm1', title: 'My Measure', is_custom: true }], dimensions: [{ id: 'd1', name: 'd1', title: 'My Dim', is_custom: true }] }, null, 2);
  fireEvent.change(ta, { target: { value: newJson } });

  // Click the Apply button (aria-label="Apply")
  const applyBtn = await waitFor(() => screen.getByLabelText('Apply'));
  fireEvent.click(applyBtn);

  // Wait for the mocked setSemanticModel to be called with parsed model
  await waitFor(() => {
    expect(mockSetSemanticModel).toHaveBeenCalled();
    const calledWith = mockSetSemanticModel.mock.calls[0][0];
    expect(calledWith).toHaveProperty('measures');
    expect(calledWith.measures[0]).toHaveProperty('title', 'My Measure');
  });
});
