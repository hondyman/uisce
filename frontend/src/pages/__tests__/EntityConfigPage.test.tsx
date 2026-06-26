// @vitest-environment jsdom
import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { vi, describe, it, expect, beforeEach } from 'vitest';

// Mock tenant scope before importing the component so import-time checks use the mock
vi.mock('../../utils/tenantScope', () => ({ hasTenantScope: () => true }));
// Mock a minimal subset of antd used by the component to make tests deterministic
vi.mock('antd', () => {
  const React = require('react');
  return {
    Card: ({ children, title, extra }: any) => React.createElement('div', null, React.createElement('div', null, title, extra), React.createElement('div', null, children)),
    Button: ({ children, onClick, type, danger, icon, ...rest }: any) => React.createElement('button', { onClick, ...(rest || {}) }, children || (icon ? React.createElement('span', null) : null)),
    Form: ({ children }: any) => React.createElement('div', null, children),
    Input: ({ value, onChange, placeholder, disabled }: any) => React.createElement('input', { value, onChange, placeholder, disabled }),
    Select: ({ children, value, onChange }: any) => React.createElement('select', { value, onChange }, children),
    Option: ({ children, value }: any) => React.createElement('option', { value }, children),
    message: { info: () => {}, success: () => {}, error: () => {} },
    Modal: ({ children, open }: any) => open ? React.createElement('div', null, children) : null,
    Tree: ({ treeData, onSelect }: any) => React.createElement('div', null, (treeData || []).map((node: any) => React.createElement('div', { key: node.key, onClick: () => onSelect && onSelect([node.key]) }, typeof node.title === 'string' ? node.title : (node.title && node.title.props ? node.title.props.children : node.title)))),
    Table: ({ dataSource = [], columns = [] }: any) => React.createElement('table', null, React.createElement('tbody', null, (dataSource || []).map((row: any) => React.createElement('tr', { key: row.key }, (columns || []).map((col: any) => React.createElement('td', { key: col.key || col.title }, col.render ? col.render(null, row) : row[col.dataIndex])))))),
    Popconfirm: ({ children, onConfirm }: any) => React.createElement('div', null, children, React.createElement('button', { onClick: onConfirm }, 'Yes')),
    Space: ({ children }: any) => React.createElement('div', null, children),
    Typography: { Title: ({ children }: any) => React.createElement('h4', null, children) },
    Dropdown: ({ children }: any) => React.createElement('div', null, children),
  };
});
import EntityConfigPage from '../EntityConfigPage';

const mockEntities = {
  entity_registry: [
    {
      entity_name: 'trade',
      display_name: 'Trade',
      subtypes: [ { key: 'Trade', label: 'Trade' }, { key: 'BlockTrade', label: 'Block Trade' } ],
      default_schema: {
        trade: {
          fields: [
            { key: 'trade_id', label: 'Trade ID', type: 'string' },
            { key: 'amount', label: 'Amount', type: 'number', condition: "values.type === 'Trade'" }
          ]
        }
      }
    }
  ]
};

beforeEach(() => {
  global.fetch = vi.fn((url, opts) => {
    if (url?.toString().includes('/api/entity_registry/trade') && opts?.method === 'PUT') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) } as any);
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve(mockEntities) } as any);
  }) as any;
  // jsdom doesn't implement getComputedStyle fully; stub what's needed by rc-table
  // some libraries call getComputedStyle for measuring scrollbars
  (window as any).getComputedStyle = (_elt: any) => ({
    getPropertyValue: () => ''
  });
});

describe('EntityConfigPage', () => {
  it('shows fields in table for selected entity and allows delete', async () => {
    render(<EntityConfigPage />);

    // wait for tree to populate and click the first matching 'Trade' title
    await waitFor(() => {
      const elems = screen.getAllByText('Trade');
      if (!elems || elems.length === 0) throw new Error('Trade not rendered');
      // click the first one (entity title)
      fireEvent.click(elems[0]);
    });

    // table should show Trade ID
    await waitFor(() => screen.getByText('Trade ID'));
    expect(screen.getByText('Trade ID')).toBeTruthy();

    // find the table row for Trade ID and click the delete button inside it
    const row = screen.getByText('Trade ID').closest('tr') as HTMLElement;
    expect(row).toBeTruthy();
    const { within } = await import('@testing-library/react');
    const rowButtons = within(row).getAllByRole('button');
    // assume Edit is first, Delete is last
    const deleteBtn = rowButtons[rowButtons.length - 1];
    fireEvent.click(deleteBtn);

  // confirm Popconfirm within the same table row
  await waitFor(() => within(row).getByText('Yes'));
  fireEvent.click(within(row).getByText('Yes'));

    // expect fetch called with PUT to trade endpoint
    await waitFor(() => {
      expect((global.fetch as any).mock.calls.some((c: any) => c[0].toString().includes('/api/entity_registry/trade') && c[1]?.method === 'PUT')).toBeTruthy();
    });
  });
});
