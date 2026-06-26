// @vitest-environment jsdom
// React import removed (unused)
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import UnifiedSemanticBuilder from '../../../pages/Fabric/UnifiedSemanticBuilder';
import { test, expect, vi } from 'vitest';
import { TenantProvider } from '../../../contexts/TenantContext';

// Common mocks for page-level tests (reuse patterns from other tests)
vi.mock('react-dnd', () => ({ useDrag: () => [{ isDragging: false }, () => {}], useDrop: () => [{ isOver: false }, () => {}] }));

const mockSetSemanticModel = vi.fn();
const mockSetSelectedModel = vi.fn();
const mockShowNotification = vi.fn();

vi.mock('../../../hooks/useUnifiedSemanticBuilder', () => ({
  useUnifiedSemanticBuilder: () => {
    const semanticModel = { name: 'semantic_model', dimensions: [], measures: [], filters: [], joins: [], is_custom: true };
    return {
      nodes: [],
      searchTerm: '',
      setSearchTerm: () => {},
      selectedColumn: null,
      setSelectedColumn: () => {},
      modelName: 'semantic_model',
      setModelName: () => {},
      // Render canvas by default for this integration test
      showCode: null,
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
      showNotification: mockShowNotification,
    };
  }
}));

vi.mock('../../../hooks/useModelCatalog', () => ({
  useModelCatalog: () => ({
    models: [
      { id: '1', model_key: 'base1', display_name: 'Base 1', is_custom: false, resolved_config: {} },
      { id: '2', model_key: 'custom_1', display_name: 'Custom 1', is_custom: true, resolved_config: {} }
    ],
    selectedModel: { id: 'custom-1', model_key: 'custom_1', is_custom: true, parent_model_key: 'custom_1', title: 'Custom 1', display_name: 'Custom 1' },
    setSelectedModel: mockSetSelectedModel,
    searchTerm: '',
    setSearchTerm: () => {},
    loading: false,
    error: null,
    createCustomModel: async () => ({}),
    cloneModel: async () => ({}),
    refreshModels: async () => {},
    updateModel: async () => {},
    deleteModel: async () => ({ success: true }),
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

test('page-level: editor does not change when attempting to set extends to the same model', async () => {
  render(
    <MemoryRouter>
      <TenantProvider>
        <UnifiedSemanticBuilder tenantId="t1" datasourceId="ds1" onClose={() => {}} />
      </TenantProvider>
    </MemoryRouter>
  );

  // Wait for the extends tile to appear in the canvas
  const extendsTile = await waitFor(() => document.querySelector('[data-element-id^="extends-"]')) as HTMLElement | null;
  expect(extendsTile).toBeTruthy();
  fireEvent.click(extendsTile!);

  // The typeahead input should appear in the form section
  const input = await waitFor(() => screen.getByPlaceholderText(/Search base model(\.|…)/i));
  expect(input).toBeTruthy();
  fireEvent.focus(input);
  // Type a substring to ensure the filtered list opens and activeIndex becomes 0
  fireEvent.change(input, { target: { value: 'Custom' } });
  // Wait for the listbox to open, then pick the option element from inside it (safe against duplicate text nodes)
  // List may render without an explicit role in jsdom; query by class to be robust
  const optionEl = await waitFor(() => {
    const items = Array.from(document.querySelectorAll('.extends-item')) as HTMLElement[];
    return items.find(it => (it.querySelector('.item-label')?.textContent || '').match(/Custom 1/i));
  });
  expect(optionEl).toBeTruthy();
  // Click the inner label element directly (jsdom may treat text nodes as targets for li events)
  const labelSpan = optionEl!.querySelector('.item-label') as HTMLElement | null;
  if (labelSpan) {
    fireEvent.mouseDown(labelSpan);
    fireEvent.mouseUp(labelSpan);
    fireEvent.click(labelSpan);
  } else {
    // fallback to clicking the option itself
    fireEvent.mouseDown(optionEl!);
    fireEvent.mouseUp(optionEl!);
    fireEvent.click(optionEl!);
  }

  // The page-level guard should not update selection (no optimistic selectedModel change)
  await waitFor(() => {
    expect(mockSetSelectedModel).not.toHaveBeenCalledWith(expect.objectContaining({ parent_model_key: 'custom_1' }));
  });
});
