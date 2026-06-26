import React from 'react';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import renderWithProviders from '../../../../test/testUtils';
import Marketplace from '../Marketplace';

describe('Marketplace', () => {
  beforeEach(() => {
    localStorage.setItem('selected_tenant', JSON.stringify({ id: 'tenant123' }));
    localStorage.setItem('selected_product', JSON.stringify({ alpha_product: { product_name: 'Alpha' } }));
    localStorage.setItem('selected_datasource', JSON.stringify({ id: 'ds1' }));

    // default fetch mock: marketplace items and tenant items
    // @ts-ignore
    global.fetch = vi.fn(async (url: string) => {
      if (String(url).startsWith('/api/marketplace/items')) {
        return {
          ok: true,
          json: async () => ({ items: [ { id: 'm1', name: 'My Rule', summary: 'S', item_type: 'rule', category: 'cat', icon_emoji: '🔧', color_hex: '#fff', is_official: true, is_core: false, version: '1.0', implementation_json: {}, scope:'', rule_type:'', frequency:'', evaluation_order: 0, is_public:true, status:'active', external_api_providers: [], requires_credentials: false, usage_count: 0 } ]}),
        } as any;
      }
      if (String(url).startsWith('/api/marketplace/tenant-items')) {
        return {
          ok: true,
          json: async () => ([{ id: 'ti1', tenant_id: 'tenant123', marketplace_item_id: 'm1', custom_name: 'My Rule Local', enabled_for_tenant: true, added_at: new Date().toISOString(), usage_count: 2, marketplace_version_at_time_of_add: '1.0', local_version:'1.0', has_local_modifications:false }]),
        } as any;
      }

      // remove API
      if (String(url).includes('/api/marketplace/tenant-items')) {
        return { ok: true, json: async () => ({}) } as any;
      }

      return { ok: false } as any;
    });
  });

  it('removes an item with confirmation and shows a success notification', async () => {
    renderWithProviders(<Marketplace />);

    // Switch to 'My Items' tab
    const tab = await screen.findByText(/My Items/i);
    fireEvent.click(tab);

    // wait for our tenant item to render
    await waitFor(() => expect(screen.getByText(/My Rule Local/i)).toBeInTheDocument());

    const removeBtn = screen.getByText(/Remove/i);
    fireEvent.click(removeBtn);

    // confirm dialog
    await waitFor(() => expect(screen.getByRole('dialog')).toBeInTheDocument());
    const confirm = screen.getByText('Confirm');
    fireEvent.click(confirm);

    // Expect notification text
    await waitFor(() => expect(screen.getByText(/Item removed successfully/i)).toBeInTheDocument());
  });
});
