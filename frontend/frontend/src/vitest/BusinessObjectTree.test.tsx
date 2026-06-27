import React from 'react';
import { render, screen } from '@testing-library/react';
import BusinessObjectTree from '@/components/bo/BusinessObjectTree';

test('highlights fields when present in highlightedFields set', async () => {
  const fields = [ { fullPath: 'b1.customer.name', label: 'name' } ];
  render(<BusinessObjectTree fields={fields} highlightedFields={new Set(['b1.customer.name'])} />);

  const item = await screen.findByText('name');
  expect(item).toBeInTheDocument();
  const li = item.closest('li');
  expect(li).toHaveStyle({ borderLeft: '3px solid #f5c518' });
});