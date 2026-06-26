import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import AdvancedFieldSelector from '@/components/validation/AdvancedFieldSelector';

test('highlights fields passed via highlightedFields prop', async () => {
  const onFieldSelected = vi.fn();

  render(
    <AdvancedFieldSelector
      onFieldSelected={onFieldSelected}
      entities={[
        {
          name: 'b1',
          displayName: 'b1',
          fields: [ { name: 'name', dataType: 'string', nullable: false } ],
          relationships: [],
        }
      ]}
      currentEntity={'b1'}
      highlightedFields={new Set(['b1.name'])}
    />
  );

  // Open the dialog
  const trigger = screen.getByRole('button', { name: /Select Field.../i });
  fireEvent.click(trigger);

  // The field should be rendered and highlighted
  const fieldText = await screen.findByText('name');
  expect(fieldText).toBeInTheDocument();

  // Find the list item button that contains this text
  const buttons = screen.getAllByRole('button');
  const button = buttons.find(b => b.textContent && b.textContent.includes('name'));
  expect(button).toBeTruthy();

  // Expect highlight style (border-left) applied
  // Note: sx translates to style attributes in DOM; check partial style
  expect(button).toHaveStyle({ borderLeft: '3px solid #f5c518' });
});