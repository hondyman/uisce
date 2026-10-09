import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import MenuDesignerPage from '../../pages/menu-designer/MenuDesignerPage';
import type { NavigationMenuNode } from '../../api/navigationMenu';

// The Menu Designer edits the tenant's own navigation tree and can switch
// gold-copy entries off for this tenant. These pin the two behaviours a
// tenant depends on: hiding a core entry, and moving an entry under another
// folder without offering a folder's own subtree as its new parent.

const { mockListTree, mockSetHidden, mockUpdate, mockListPages } = vi.hoisted(() => ({
  mockListTree: vi.fn(),
  mockSetHidden: vi.fn(),
  mockUpdate: vi.fn(),
  mockListPages: vi.fn(),
}));

vi.mock('../../api/navigationMenu', () => ({
  NavigationMenuApi: {
    listTree: mockListTree,
    setHidden: mockSetHidden,
    update: mockUpdate,
    create: vi.fn(),
    remove: vi.fn(),
  },
}));
vi.mock('../../api/pageStudio', () => ({
  PageStudioApi: { listPages: mockListPages },
}));

const node = (over: Partial<NavigationMenuNode> & { id: string; label: string }): NavigationMenuNode => ({
  nodeKey: over.id,
  requiredEntitlement: 'BASE_USER',
  displayOrder: 0,
  children: [],
  ...over,
});

const tree: NavigationMenuNode[] = [
  node({
    id: 'core-build',
    label: 'Build',
    inherited: true,
    children: [node({ id: 'core-cubes', label: 'Cubes', parentId: 'core-build', targetPageKey: 'cubes-catalog', inherited: true })],
  }),
  node({
    id: 'f-models',
    label: 'Models',
    children: [node({ id: 'f-my-cubes', label: 'My cubes', parentId: 'f-models', targetPageKey: 'cubes-catalog' })],
  }),
  node({ id: 'f-reports', label: 'Reports' }),
];

const renderPage = () =>
  render(
    <MemoryRouter>
      <MenuDesignerPage />
    </MemoryRouter>,
  );

describe('MenuDesignerPage', () => {
  beforeEach(() => {
    mockListTree.mockReset().mockResolvedValue(tree);
    mockSetHidden.mockReset().mockResolvedValue(undefined);
    mockUpdate.mockReset().mockResolvedValue({});
    // The edited entry targets cubes-catalog, so the page list must offer it (MUI warns otherwise).
    mockListPages.mockReset().mockResolvedValue([{ id: 'p-cubes', slug: 'cubes-catalog', name: 'Cubes' }]);
  });

  it('hides a gold-copy entry for this tenant only', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText('Build')).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: 'Hide for this tenant' }));

    await waitFor(() => expect(mockSetHidden).toHaveBeenCalledWith('core-build', true));
    expect(mockUpdate).not.toHaveBeenCalled();
  });

  it('moves a tenant entry under another folder', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText('Models')).toBeInTheDocument());

    // Expand Models so its child is in the tree, then edit "My cubes".
    fireEvent.click(screen.getByText('Models'));
    await waitFor(() => expect(screen.getByText('My cubes')).toBeInTheDocument());
    // Edit buttons in DOM order: Models, My cubes, Reports (Build is core, so none).
    fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[1]);

    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Under' }));
    fireEvent.click(await screen.findByRole('option', { name: 'Reports' }));
    fireEvent.click(screen.getByRole('button', { name: 'Save' }));

    await waitFor(() =>
      expect(mockUpdate).toHaveBeenCalledWith(
        'f-my-cubes',
        expect.objectContaining({ parentId: 'f-reports', label: 'My cubes', nodeKey: 'f-my-cubes' }),
      ),
    );
  });

  it("does not offer a folder's own subtree as its new parent", async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText('Models')).toBeInTheDocument());

    // Edit the Models folder itself (first Edit button: Build is core, Models is the first tenant row).
    fireEvent.click(screen.getAllByRole('button', { name: 'Edit' })[0]);
    fireEvent.mouseDown(screen.getByRole('combobox', { name: 'Under' }));

    expect(await screen.findByRole('option', { name: 'Reports' })).toBeInTheDocument();
    expect(screen.queryByRole('option', { name: 'Models' })).not.toBeInTheDocument();
  });
});
