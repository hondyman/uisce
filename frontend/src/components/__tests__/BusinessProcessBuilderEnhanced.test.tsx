import React from 'react';
import { screen, fireEvent, waitFor } from '@testing-library/react';
import renderWithProviders from '../../../test/testUtils';
import BusinessProcessBuilderEnhanced from '../BusinessProcessBuilderEnhanced';

describe('BusinessProcessBuilderEnhanced', () => {
  it('saves successfully after adding a step', async () => {
    renderWithProviders(<BusinessProcessBuilderEnhanced />);

    // Click the first step type card to add a step
    const stepCard = await screen.findByText('Data Entry');
    fireEvent.click(stepCard);

    // Save button should be enabled now
    const saveButton = screen.getByRole('button', { name: /Save/i });
    fireEvent.click(saveButton);

    await waitFor(() => expect(screen.getByText(/Business Process saved successfully/i)).toBeInTheDocument());
  });

  it('shows simulation toast when simulate clicked with no steps', async () => {
    renderWithProviders(<BusinessProcessBuilderEnhanced />);

    const simulateBtn = screen.getByRole('button', { name: /Simulate BP execution/i }) || screen.getByText(/Simulate BP execution/i);
    fireEvent.click(simulateBtn);

    await waitFor(() => expect(screen.getByText(/Simulating BP execution/)).toBeInTheDocument());
  });
});
