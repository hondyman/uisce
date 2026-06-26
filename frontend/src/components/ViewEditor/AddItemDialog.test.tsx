import { render, screen, waitFor } from '@testing-library/react';
import { vi, Mock } from 'vitest';
import userEvent from '@testing-library/user-event';
import '@testing-library/jest-dom';
import { AddItemDialog } from './AddItemDialog';

// We import the named export AddItemDialog via the component file; if default export is ViewEditorComplete,
// tests will import AddItemDialog through the module's exported symbol (we'll reference module directly in render).

describe('AddItemDialog (integration)', () => {
  beforeEach(() => {
    global.fetch = vi.fn() as unknown as typeof fetch;
  });

  afterEach(() => {
    vi.resetAllMocks();
  });

  it('fetches model metadata and allows selecting multiple dimensions', async () => {
    const fakeResponse = {
      models: [
        {
          model_key: '/public/orders',
          display_name: 'Orders',
          is_core: true,
          resolved_config: {
            cubes: [
              {
                dimensions: {
                  order_id: { sql: 'order_id', type: 'number', title: 'Order ID' },
                  customer_id: { sql: 'customer_id', type: 'string', title: 'Customer ID' }
                }
              }
            ]
          }
        }
      ]
    };

    const fetchMock = global.fetch as unknown as Mock;
    fetchMock.mockResolvedValueOnce({ ok: true, json: async () => fakeResponse });

    const onAdd = vi.fn();
    const onClose = vi.fn();

    render(
      <AddItemDialog
        open={true}
        type="dimension"
        onClose={onClose}
        onAdd={onAdd}
        tenantId="tenant-1"
        datasourceId="datasource-1"
        viewData={{ cubes: ['/public/orders'] }}
      />
    );

    const input = screen.getByPlaceholderText(/search dimensions/i);
    await userEvent.click(input);

    await waitFor(() => {
      expect(screen.getByText('Order ID')).toBeInTheDocument();
      expect(screen.getByText('Customer ID')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Order ID'));
    await userEvent.click(screen.getByText('Customer ID'));

    await userEvent.click(screen.getByRole('button', { name: /Add 2 Dimensions/i }));

    await waitFor(() => expect(onAdd).toHaveBeenCalledTimes(1));

    const [, payload] = onAdd.mock.calls[0];
    expect(Array.isArray(payload)).toBe(true);
    expect(payload).toHaveLength(2);
    expect(payload).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: '/public/orders.order_id',
          dimensionData: expect.objectContaining({ name: 'order_id' })
        }),
        expect.objectContaining({
          id: '/public/orders.customer_id',
          dimensionData: expect.objectContaining({ name: 'customer_id' })
        })
      ])
    );
  });

  it('fetches model metadata and allows selecting multiple measures', async () => {
    const fakeResponse = {
      models: [
        {
          model_key: '/public/orders',
          display_name: 'Orders',
          resolved_config: {
            cubes: [
              {
                measures: {
                  count: { sql: '*', type: 'count', title: 'Count' },
                  total_revenue: { sql: 'sum(revenue)', type: 'sum', title: 'Total Revenue' }
                }
              }
            ]
          }
        }
      ]
    };

    const fetchMock = global.fetch as unknown as Mock;
    fetchMock.mockResolvedValueOnce({ ok: true, json: async () => fakeResponse });

    const onAdd = vi.fn();
    const onClose = vi.fn();

    render(
      <AddItemDialog
        open={true}
        type="measure"
        onClose={onClose}
        onAdd={onAdd}
        tenantId="tenant-1"
        datasourceId="datasource-1"
        viewData={{ cubes: ['/public/orders'] }}
      />
    );
    const input = screen.getByPlaceholderText(/search measures/i);
    await userEvent.click(input);

    await waitFor(() => {
      expect(screen.getByText('Count')).toBeInTheDocument();
      expect(screen.getByText('Total Revenue')).toBeInTheDocument();
    });

    await userEvent.click(screen.getByText('Count'));
    await userEvent.click(screen.getByText('Total Revenue'));

    await userEvent.click(screen.getByRole('button', { name: /Add 2 Measures/i }));

    await waitFor(() => expect(onAdd).toHaveBeenCalledTimes(1));

    const [, payload] = onAdd.mock.calls[0];
    expect(Array.isArray(payload)).toBe(true);
    expect(payload).toHaveLength(2);
    expect(payload).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          id: '/public/orders.count',
          measureData: expect.objectContaining({ name: 'count' })
        }),
        expect.objectContaining({
          id: '/public/orders.total_revenue',
          measureData: expect.objectContaining({ name: 'total_revenue' })
        })
      ])
    );
  });
});
