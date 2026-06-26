// React import removed (unused)
import { render, screen, fireEvent } from '@testing-library/react';
import EnhancedTileForm from '../EnhancedTileForm';

const baseProps = {
  element: { name: 'test', title: 'Test', sql: '', mappings: {}, joins: [] },
  type: 'dimension' as const,
  isCore: false,
  isNew: false,
  coreOptions: [],
  onUpdate: () => {},
  onSave: () => {},
  onCancel: () => {},
};

describe('EnhancedTileForm accessibility', () => {
  test('section headers are keyboard toggleable (Enter and Space)', () => {
    render(<EnhancedTileForm {...baseProps} />);

    // Basic section header
    const basicHeader = screen.getByText(/Basic Properties/i).closest('.section-header') as HTMLElement;
    expect(basicHeader).toBeTruthy();

  // Initially expanded: content should be visible (check for the name input by placeholder)
  expect(screen.queryByPlaceholderText('Enter name')).toBeTruthy();

    // Press Space to collapse
    basicHeader.focus();
    fireEvent.keyDown(basicHeader, { key: ' ' });
    fireEvent.keyUp(basicHeader, { key: ' ' });

  // After collapse, name input should not be in the document
  expect(screen.queryByPlaceholderText('Enter name')).toBeNull();

    // Press Enter to expand
    fireEvent.keyDown(basicHeader, { key: 'Enter' });
    fireEvent.keyUp(basicHeader, { key: 'Enter' });

  expect(screen.queryByPlaceholderText('Enter name')).toBeTruthy();
  });
});
