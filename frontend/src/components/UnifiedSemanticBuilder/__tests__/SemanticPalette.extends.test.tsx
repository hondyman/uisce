// React import removed (unused)
import { render, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';

// Mock react-dnd hooks used by PaletteItem to avoid setting up a DnD provider in tests.
vi.mock('react-dnd', () => ({
  useDrag: () => [{ isDragging: false }, (_node: any) => {}]
}));

import SemanticPalette from '../SemanticPalette';

describe('SemanticPalette Extends button', () => {
  it('disables Extends when canAddExtends is false and shows reason tooltip', async () => {
  const onAdd = vi.fn();
    const reason = 'Extends already present on canvas';
  const { container } = render(<SemanticPalette onAdd={onAdd} canAddExtends={false} extendsDisabledReason={reason} horizontal={true} />);
  // main extends button has classes 'palette-icon-btn' and 'extends'
  const btn = container.querySelector('button.palette-icon-btn.extends') as HTMLButtonElement | null;
  expect(btn).toBeTruthy();
  expect(btn).toBeDisabled();

  // The button should expose the reason via its title attribute when disabled
  expect(btn!.getAttribute('title')).toBe(reason);

  // The small visual indicator should be present (select by class)
  const indicator = container.querySelector('button.extends-disabled-indicator') as HTMLButtonElement | null;
  expect(indicator).toBeTruthy();
  expect(indicator!.getAttribute('title')).toBe(reason);

    // clicking should not call onAdd
    fireEvent.click(btn);
    expect(onAdd).not.toHaveBeenCalled();
  });
});
