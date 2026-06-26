// @vitest-environment jsdom
// React import removed (unused)
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import WorkspaceMain from '../../../components/UnifiedSemanticBuilder/WorkspaceMain';
import { TenantProvider } from '../../../contexts/TenantContext';
import { test, expect, vi } from 'vitest';

// Mock react-dnd hooks used by PaletteItem to avoid setting up a DnD provider in tests.
vi.mock('react-dnd', () => ({
  useDrag: () => [{ isDragging: false }, (_node: any) => {}],
  useDrop: () => [{ isOver: false }, (_node: any) => {}]
}));

// Reuse many of the same mocks as other page-level tests to allow mounting
const mockSetSemanticModel = vi.fn();
const mockSetSelectedModel = vi.fn();

vi.mock('../../../hooks/useUnifiedSemanticBuilder', () => {
  // React require removed (automatic JSX runtime)
  return {
    useUnifiedSemanticBuilder: (_datasourceId: string) => {
      const semanticModel = { name: 'semantic_model', dimensions: [], measures: [], filters: [], joins: [], is_custom: true };
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

vi.mock('../../../hooks/useModelCatalog', () => ({
  useModelCatalog: () => ({
    models: [
      { id: '1', model_key: 'base1', display_name: 'Base 1', is_custom: false, resolved_config: {} },
      { id: '2', model_key: 'base2', display_name: 'Base 2', is_custom: false, resolved_config: {} },
    ],
    selectedModel: { id: 'custom-1', model_key: 'custom_1', is_custom: true, parent_model_key: 'base1', title: 'Custom 1', display_name: 'Custom 1' },
    setSelectedModel: mockSetSelectedModel,
    searchTerm: '',
    setSearchTerm: () => {},
    loading: false,
    error: null,
    createCustomModel: async () => ({}),
    cloneModel: async () => ({}),
    refreshModels: async () => {},
    updateModel: async () => {},
    deleteModel: async () => ({ success: true })
  })
}));

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
vi.mock('../../../contexts/AuthContext', () => ({ useAuth: () => ({ user: null, token: null, refreshTokenValue: null, tokenExpiresAt: null, isAuthenticated: false, isLoading: false, isAdmin: () => false, login: async () => {}, register: async () => {}, forgotPassword: async () => {}, resetPassword: async () => {}, logout: async () => {}, refreshToken: async () => {}, isTokenExpired: () => false, getValidToken: async () => 'test-token' }) }));

test('selecting a new extends value calls onChangeExtends with the chosen base', async () => {
  const onChangeExtends = vi.fn();

  const semanticModel = { name: 'semantic_model', dimensions: [], measures: [], filters: [], joins: [], is_custom: true };
  const selectedModel = { id: 'custom-1', model_key: 'custom_1', is_custom: true, parent_model_key: 'base1', title: 'Custom 1', display_name: 'Custom 1' };

  const props: any = {
    isOver: false,
    drop: () => {},
    activeWorkspaceTab: 'canvas',
    setActiveWorkspaceTab: () => {},
    selectedColumn: null,
    addDimension: () => {},
    addMeasure: () => {},
    addFilter: () => {},
    getBusinessTermForColumn: () => null,
    semanticModel,
    setSemanticModel: () => {},
    modelName: 'semantic_model',
    showCode: null,
    setShowCode: () => {},
    rawGenerateJSON: () => JSON.stringify(semanticModel),
    rawGenerateYAML: () => '',
    generateCustomJSON: () => JSON.stringify(semanticModel),
    generateCustomYAML: () => '',
    generateCoreJSON: () => JSON.stringify(semanticModel),
    generateCoreYAML: () => '',
    generateMergedModelObject: () => semanticModel,
    selectedModel,
    openAddModal: () => {},
    enhancedRemoveSemanticElement: () => {},
    toggleElementEdit: () => {},
    updateSemanticElement: () => {},
    coreOptions: [],
    refreshCompatibility: async () => {},
    compatLoading: false,
    issueLevelFilter: 'all',
    setIssueLevelFilter: () => {},
    issueCodeFilter: '',
    setIssueCodeFilter: () => {},
    compatErr: null,
    filteredCompat: [],
    filteredNodes: [],
    setSearchTerm: () => {},
    expandIssues: {},
    setExpandIssues: () => {},
    expandChanges: {},
    setExpandChanges: () => {},
    isCodeDirty: false,
    setIsCodeDirty: () => {},
    editMode: true,
    setEditMode: () => {},
    availableBaseModels: [
      { key: 'base1', label: 'Base 1', kind: 'core' },
      { key: 'base2', label: 'Base 2', kind: 'core' }
    ],
    onChangeExtends,
    onImportCode: async () => {}
  };

  render(
    <TenantProvider>
      <WorkspaceMain {...props} />
    </TenantProvider>
  );

  // Click the extends tile if present in canvas
  const extendsTile = await waitFor(() => document.querySelector('[data-element-id^="extends-"]')) as HTMLElement | null;
  expect(extendsTile).toBeTruthy();
  fireEvent.click(extendsTile!);

  // The typeahead input should appear in the form section
  // Accept plain dots or Unicode ellipsis used in the placeholder
  const input = await waitFor(() => screen.getByPlaceholderText(/Search base model(\.|…)/i));
  expect(input).toBeTruthy();
  fireEvent.change(input, { target: { value: 'Base 2' } });

  // The list should show an option for Base 2
  const option = await waitFor(() => screen.getByText(/Base 2/i));
  expect(option).toBeTruthy();
  fireEvent.mouseDown(option);

  // Ensure onChangeExtends was invoked with the expected key
  await waitFor(() => {
    expect(onChangeExtends).toHaveBeenCalled();
    expect(onChangeExtends).toHaveBeenCalledWith(expect.stringMatching(/base2|Base 2/i));
  });
});
