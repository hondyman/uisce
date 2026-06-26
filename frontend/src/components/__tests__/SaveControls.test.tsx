import React from 'react';
import { render, fireEvent, screen } from '@testing-library/react';
import { vi } from 'vitest';
import SaveControls from '../SaveControls';
import renderWithProviders from '../../../test/testUtils';
import * as notistack from 'notistack';

describe('SaveControls', () => {
  it('shows error notification when saving without view', async () => {
    const enqueueSpy = vi.fn();
    vi.spyOn(notistack, 'useSnackbar').mockReturnValue({ enqueueSnackbar: enqueueSpy, closeSnackbar: vi.fn() } as any);

    // Render with no view
    const tab = { title: 'My Query', view: null, savedId: null, query: {}, viz: null } as any;
    const mockOnSave = vi.fn();

    renderWithProviders(<SaveControls tab={tab} onSave={mockOnSave} />);

    fireEvent.click(screen.getByText('Save'));

    expect(enqueueSpy).toHaveBeenCalled();
    expect(enqueueSpy.mock.calls[0][0]).toContain('A view must be selected to save a query');
  });
});
