// React import removed (automatic JSX runtime)
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import AccessibleActionsMenu from '../AccessibleActionsMenu';

const mockModel = {
  id: 'm1',
  model_key: 'test_model',
  display_name: 'Test Model',
  is_custom: true,
  custom_model_exists: true,
  is_core: false,
  description: 'desc',
  status: 'published',
  version: '1',
  metadata: { can_create: true }
} as any;

describe('AccessibleActionsMenu', () => {
  it('renders nothing when closed', () => {
    const { container } = render(
      <AccessibleActionsMenu model={mockModel} isOpen={false} onClose={() => {}} />
    );
    expect(container.querySelector('.actions-dropdown')).toBeNull();
  });

  it('focuses first item when opened and closes on Escape', async () => {
    const onClose = vi.fn();
    render(<div>
      <button data-testid="before">before</button>
      <AccessibleActionsMenu model={mockModel} isOpen={true} onClose={onClose} onCreateCustom={() => {}} onClone={() => {}} onInfo={() => {}} />
      <button data-testid="after">after</button>
    </div>);

    const first = screen.getByText(/Create custom/i).closest('button');
    expect(first).toBeTruthy();
    // first should be focused (wait for the component's setTimeout focus)
    await waitFor(() => {
      expect(document.activeElement).toBe(first);
    });

    // Press Escape -> onClose called
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(onClose).toHaveBeenCalled();
  });

  it('traps Tab within menu', async () => {
    const onClose = vi.fn();
    render(<div>
      <button data-testid="before">before</button>
      <AccessibleActionsMenu model={mockModel} isOpen={true} onClose={onClose} onCreateCustom={() => {}} onClone={() => {}} onInfo={() => {}} />
      <button data-testid="after">after</button>
    </div>);

    const items = Array.from(document.querySelectorAll('.dropdown-item')) as HTMLButtonElement[];
    expect(items.length).toBeGreaterThanOrEqual(2);

    const first = items[0];
    const last = items[items.length - 1];

    // wait for initial focus
    await waitFor(() => {
      expect(document.activeElement).toBe(first);
    });

    // Tab forward from last wraps to first
    last.focus();
    fireEvent.keyDown(document, { key: 'Tab' });
    await waitFor(() => {
      expect(document.activeElement).toBe(first);
    });

    // Shift+Tab from first wraps to last
    first.focus();
    fireEvent.keyDown(document, { key: 'Tab', shiftKey: true });
    await waitFor(() => {
      expect(document.activeElement).toBe(last);
    });
  });

  it('closes on outside click', () => {
    const onClose = vi.fn();
    render(<div>
      <button data-testid="outside">outside</button>
      <AccessibleActionsMenu model={mockModel} isOpen={true} onClose={onClose} onCreateCustom={() => {}} onClone={() => {}} onInfo={() => {}} />
    </div>);

    fireEvent.click(screen.getByTestId('outside'));
    expect(onClose).toHaveBeenCalled();
  });
});
