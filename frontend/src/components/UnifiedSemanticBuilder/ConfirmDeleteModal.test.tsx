// React import removed (unused)
import { render, screen, fireEvent } from '@testing-library/react';
import { vi } from 'vitest';
import ConfirmDeleteModal from './ConfirmDeleteModal';
import userEvent from '@testing-library/user-event';

describe('ConfirmDeleteModal', () => {
  it('renders when open and calls onCancel/onConfirm appropriately', async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    const onConfirm = vi.fn();

    render(
      <ConfirmDeleteModal
        open={true}
        title="Test Delete"
        message="Are you sure?"
        associated={[{ id: 'a1', display_name: 'Assoc A' }]}
        onCancel={onCancel}
        onConfirm={onConfirm}
      />
    );

    // Title and message present
    expect(screen.getByRole('heading', { name: /Test Delete/i })).toBeInTheDocument();
    expect(screen.getByText(/Are you sure\?/i)).toBeInTheDocument();

  // Click cancel
  await user.click(screen.getByRole('button', { name: /Cancel/i }));
  expect(onCancel).toHaveBeenCalled();

  // Click confirm (button)
  await user.click(screen.getByRole('button', { name: /Delete/i }));
  expect(onConfirm).toHaveBeenCalled();
  });

  it('closes on Escape key and backdrop click', async () => {
    const user = userEvent.setup();
    const onCancel = vi.fn();
    const onConfirm = vi.fn();

    const { container } = render(
      <ConfirmDeleteModal
        open={true}
        onCancel={onCancel}
        onConfirm={onConfirm}
      />
    );

    // Escape key
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onCancel).toHaveBeenCalled();

    // Backdrop click (click on the backdrop element)
    const backdrop = container.querySelector('.confirm-modal-backdrop');
    expect(backdrop).toBeTruthy();
    if (backdrop) {
      await user.click(backdrop);
      expect(onCancel).toHaveBeenCalledTimes(2);
    }
  });
});
