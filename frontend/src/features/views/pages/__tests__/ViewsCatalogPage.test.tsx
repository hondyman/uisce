// React import removed (unused)
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import renderWithProviders from '../../../../../test/testUtils';
import { vi } from 'vitest';
import ViewsCatalogPage from '../ViewsCatalogPage';
import { AuthProvider } from '../../../../contexts/AuthContext';
import { MemoryRouter } from 'react-router-dom';
import { TenantProvider } from '../../../../contexts/TenantContext';

describe('ViewsCatalogPage', () => {
  beforeEach(() => {
    // @ts-ignore
    global.fetch = vi.fn(async (url: string) => {
      if (String(url).includes('/api/views')) {
        return {
          ok: true,
          status: 200,
          headers: { get: () => null },
          json: async () => ({
            views: [
              { name: 'orders_view', title: 'Orders', description: 'o', cube_count: 1, folder_count: 2 },
              { name: 'customers_view', title: 'Customers', description: 'c', cube_count: 1, folder_count: 1 },
            ],
            total: 2,
            page: 1,
            page_size: 25,
          }),
        } as any;
      }
      return { ok: false, status: 404, statusText: 'Not Found', json: async () => ({}) } as any;
    });
  // provide a selected tenant/datasource in localStorage so TenantProvider isSelected=true
  localStorage.setItem('selected_tenant', JSON.stringify({ id: 't1', display_name: 'Tenant 1', name: 'tenant1' }));
  localStorage.setItem('selected_product', JSON.stringify({ alpha_product: { product_name: 'Alpha' } }));
  localStorage.setItem('selected_datasource', JSON.stringify({ id: 'ds1', source_name: 'ds1' }));
  });

  it('renders list and allows searching', async () => {
    renderWithProviders(
      <MemoryRouter>
        <AuthProvider>
          <TenantProvider>
            <ViewsCatalogPage />
          </TenantProvider>
        </AuthProvider>
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByText('Orders')).toBeInTheDocument());
  // The page's search field placeholder is 'Search views...'
  const search = screen.getByPlaceholderText(/Search views/i);
  await waitFor(() => fireEvent.change(search, { target: { value: 'cust' } }));
  await waitFor(() => fireEvent.keyDown(search, { key: 'Enter' }));
    // still shows list (since fetch is mocked same), but controls exist
    expect(screen.getByLabelText('Page size')).toBeInTheDocument();
  });

  it('shows confirm dialog when deleting a view and shows success notification', async () => {
    renderWithProviders(
      <MemoryRouter>
        <AuthProvider>
          <TenantProvider>
            <ViewsCatalogPage />
          </TenantProvider>
        </AuthProvider>
      </MemoryRouter>
    );
    await waitFor(() => expect(screen.getByText('Orders')).toBeInTheDocument());
    // click delete button on first row
    const deleteButtons = screen.getAllByTitle('Delete');
    fireEvent.click(deleteButtons[0]);
    // expect confirm dialog to appear
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
    // click confirm
    const confirm = screen.getByText('Confirm');
    fireEvent.click(confirm);
    // wait for success snackbar
    await waitFor(() => screen.getByText(/View deleted/i));
  });
});
