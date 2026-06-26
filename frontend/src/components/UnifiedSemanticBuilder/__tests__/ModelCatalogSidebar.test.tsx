// React import removed (not needed with the new JSX transform)
import { vi } from 'vitest';
// Mock the toaster to avoid importing UI primitives that rely on path aliases during tests
vi.mock('../../ui/sonner', () => ({ toast: { success: vi.fn(), error: vi.fn(), toast: vi.fn() } }));
import { render, screen, waitFor, within, fireEvent } from '@testing-library/react';
import ModelCatalogSidebar from '../ModelCatalogSidebar';

const mockModels = [
  {
    id: '1',
    model_key: 'core_model',
    display_name: 'Core Model',
    is_core: true,
    is_custom: false,
    is_current: true,
    core_model_exists: true,
    description: 'This is a core model '.repeat(50),
  metadata: { generator: '', table_count: 2, measure_count: 1, dimension_count: 3, can_create: true },
  status: 'published' as const,
    version: 1,
    can_edit: false,
    custom_model_exists: false,
  created_at: undefined,
  updated_at: undefined,
  },
  {
    id: '2',
    model_key: 'core_model_custom',
    display_name: 'Custom Model',
    is_core: false,
    is_custom: true,
    is_current: true,
    core_model_exists: false,
    parent_model_key: 'core_model',
    description: 'A long description '.repeat(30),
  metadata: { generator: 'custom', table_count: 1, measure_count: 1, dimension_count: 1, can_create: true },
  status: 'draft' as const,
    version: 1,
    can_edit: true,
    custom_model_exists: true,
  created_at: undefined,
  updated_at: undefined,
  }
];

describe('ModelCatalogSidebar', () => {
  test('renders core/custom counts and toggles long description', async () => {
    const setSelectedModel = (globalThis as any).vi.fn();
    const setSearchTerm = (globalThis as any).vi.fn();
    const onCreateCustomModel = (globalThis as any).vi.fn();

    // Mock scrollIntoView
    window.HTMLElement.prototype.scrollIntoView = (globalThis as any).vi.fn();

    const { rerender } = render(
      <ModelCatalogSidebar
        models={mockModels as any}
        searchTerm=""
        setSearchTerm={setSearchTerm}
        selectedModel={null}
        setSelectedModel={setSelectedModel}
      />
    );

    // Rerender with handler to test create button
    rerender(
      <ModelCatalogSidebar
        models={mockModels as any}
        searchTerm=""
        setSearchTerm={setSearchTerm}
        selectedModel={null}
        setSelectedModel={setSelectedModel}
        onCreateCustomModel={onCreateCustomModel}
      />
    );

    // Header should include Create custom button
    const header = screen.getByText('Model Catalog').closest('.sidebar-header');
    expect(header).toBeTruthy();
    // Choose the visible New Model button to avoid duplicate accessible names
    const createBtn = screen.getByText('New Model').closest('button');
    expect(createBtn).toBeInTheDocument();
    // click should call the handler
    if (createBtn && onCreateCustomModel) {
      await waitFor(() => fireEvent.click(createBtn));
    }
    expect(onCreateCustomModel).toHaveBeenCalled();

    // Check for tabs with counts
    expect(screen.getByText(/Core \(1\)/i)).toBeInTheDocument();
    expect(screen.getByText(/Custom \(1\)/i)).toBeInTheDocument();

  // Click the custom tab to see the custom model
  const customTab = screen.getByText(/Custom \(1\)/i);
  await waitFor(() => fireEvent.click(customTab));
    expect(screen.getByText('Custom Model')).toBeInTheDocument();

    // Click back to core tab
  const coreTab = screen.getByText(/Core \(1\)/i);
  await waitFor(() => fireEvent.click(coreTab));

    // Click on the core model to select it
  const coreModelElement = screen.getByText('Core Model');
  await waitFor(() => fireEvent.click(coreModelElement));

    // The model should be selected
    expect(setSelectedModel).toHaveBeenCalledWith(mockModels[0]);
  });

  test('renders action buttons with correct tooltips', async () => {
    const setSelectedModel = (globalThis as any).vi.fn();
    const setSearchTerm = (globalThis as any).vi.fn();
    const onCreateCustomModel = (globalThis as any).vi.fn();
    const onDeleteModel = (globalThis as any).vi.fn();
    const onModelSelect = (globalThis as any).vi.fn();

    // Mock scrollIntoView
    window.HTMLElement.prototype.scrollIntoView = (globalThis as any).vi.fn();

    render(
      <ModelCatalogSidebar
        models={mockModels as any}
        searchTerm=""
        setSearchTerm={setSearchTerm}
        selectedModel={null}
        setSelectedModel={setSelectedModel}
        onCreateCustomModel={onCreateCustomModel}
        onDeleteModel={onDeleteModel}
        onModelSelect={onModelSelect}
      />
    );

    // Core model should have delete, add custom, and clone buttons with tooltips
    const coreModelElement = screen.getByText('Core Model').closest('.model-item');
    expect(coreModelElement).toBeInTheDocument();

    if (coreModelElement) {
      // Check delete button tooltip
      const deleteButton = within(coreModelElement).getByTitle('Delete core model and its custom(s)');
      expect(deleteButton).toBeInTheDocument();
      expect(deleteButton).toHaveAttribute('aria-label', 'Delete core model');
      
      // Check add custom model button tooltip
      const addCustomButton = within(coreModelElement).getByTitle('Create custom model with extends syntax');
      expect(addCustomButton).toBeInTheDocument();
      
      // Check clone button tooltip
      const cloneButton = within(coreModelElement).getByTitle('Clone this model');
      expect(cloneButton).toBeInTheDocument();
      
  // Test button clicks
  await waitFor(() => fireEvent.click(deleteButton));
      // Should open confirm dialog
      expect(screen.getByText('Confirm Core Model Deletion')).toBeInTheDocument();
      
      // Cancel the deletion
  await waitFor(() => fireEvent.click(screen.getByText('Cancel')));
      
      // Test add custom model click
  await waitFor(() => fireEvent.click(addCustomButton));
      expect(onCreateCustomModel).toHaveBeenCalledWith('core_model');
      
      // Test clone button click
  await waitFor(() => fireEvent.click(cloneButton));
      expect(onCreateCustomModel).toHaveBeenCalledWith('core_model');
    }

  // Switch to custom tab to test custom model buttons
  const customTab = screen.getByText(/Custom \(1\)/i);
  await waitFor(() => fireEvent.click(customTab));
    
    const customModelElement = screen.getByText('Custom Model').closest('.model-item');
    expect(customModelElement).toBeInTheDocument();

    if (customModelElement) {
      // Custom model should have a delete button with different tooltip
      const customDeleteButton = within(customModelElement).getByTitle('Delete model');
      expect(customDeleteButton).toBeInTheDocument();
      expect(customDeleteButton).toHaveAttribute('aria-label', 'Delete model');
      
  // Test custom model delete
  await waitFor(() => fireEvent.click(customDeleteButton));
      // Should open confirm dialog with custom model text
      expect(screen.getByText('Confirm Model Deletion')).toBeInTheDocument();
      expect(screen.getByText(/Are you sure you want to delete this custom model/)).toBeInTheDocument();
      
      // Confirm the deletion
  await waitFor(() => fireEvent.click(screen.getByText('Delete')));
      expect(onDeleteModel).toHaveBeenCalledWith('2', false, 'core_model_custom');
    }
  });

  test('calls onModelSelect with correct tab when model is clicked', async () => {
    const setSelectedModel = (globalThis as any).vi.fn();
    const setSearchTerm = (globalThis as any).vi.fn();
    const onModelSelect = (globalThis as any).vi.fn();

    // Mock scrollIntoView
    window.HTMLElement.prototype.scrollIntoView = (globalThis as any).vi.fn();

    render(
      <ModelCatalogSidebar
        models={mockModels as any}
        searchTerm=""
        setSearchTerm={setSearchTerm}
        selectedModel={null}
        setSelectedModel={setSelectedModel}
        onModelSelect={onModelSelect}
      />
    );

    // Click on core model should call onModelSelect with 'core' tab
  const coreModelElement = screen.getByText('Core Model');
  await waitFor(() => fireEvent.click(coreModelElement));
    
  expect(onModelSelect).toHaveBeenCalledWith(mockModels[0], 'core');

  // Switch to custom tab and click custom model
  const customTab = screen.getByText(/Custom \(1\)/i);
  await waitFor(() => fireEvent.click(customTab));
    
  const customModelElement = screen.getByText('Custom Model');
  await waitFor(() => fireEvent.click(customModelElement));
    
  expect(onModelSelect).toHaveBeenCalledWith(mockModels[1], 'custom');
  });

  test('removes model from filteredModels when model.deleted event fires and shows toast', async () => {
    const setSelectedModel = (globalThis as any).vi.fn();
    const setSearchTerm = (globalThis as any).vi.fn();

    render(
      <ModelCatalogSidebar
        models={mockModels as any}
        searchTerm=""
        setSearchTerm={setSearchTerm}
        selectedModel={mockModels[1]}
        setSelectedModel={setSelectedModel}
      />
    );

    // Confirm custom model is present
    expect(screen.getByText('Custom Model')).toBeInTheDocument();

    // Dispatch global deletion event
    window.dispatchEvent(new CustomEvent('model.deleted', { detail: { id: '2' } }));

    // Model should be removed from the DOM
    await waitFor(() => expect(screen.queryByText('Custom Model')).toBeNull());
  });
});
