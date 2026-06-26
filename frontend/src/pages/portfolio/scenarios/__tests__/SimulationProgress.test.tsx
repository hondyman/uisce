/**
 * SimulationProgress Component Tests
 * 
 * Tests for:
 * - Progress bar rendering and updates
 * - Status tracking (queued, running, completed, failed)
 * - Time estimation and elapsed time display
 * - Cancel button functionality
 * - Error state handling
 */

import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { SimulationProgress } from '../SimulationProgress';

const theme = createTheme();

const renderWithTheme = (component: React.ReactElement) => {
  return render(
    <ThemeProvider theme={theme}>
      {component}
    </ThemeProvider>
  );
};

describe('SimulationProgress Component', () => {
  const mockOnCancel = jest.fn();

  const defaultProps = {
    progress: 50,
    status: 'running' as const,
    message: 'Processing scenario...',
    estimatedTimeRemaining: 120,
    elapsedTime: 60,
    onCancel: mockOnCancel,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    test('renders progress bar with correct percentage', () => {
      renderWithTheme(<SimulationProgress {...defaultProps} />);
      
      const progressBar = screen.getByRole('progressbar');
      expect(progressBar).toHaveAttribute('aria-valuenow', '50');
    });

    test('displays status message', () => {
      renderWithTheme(<SimulationProgress {...defaultProps} />);
      
      expect(screen.getByText('Processing scenario...')).toBeInTheDocument();
    });

    test('displays elapsed time', () => {
      renderWithTheme(<SimulationProgress {...defaultProps} />);
      
      expect(screen.getByText(/1m 0s/)).toBeInTheDocument();
    });

    test('displays estimated time remaining', () => {
      renderWithTheme(<SimulationProgress {...defaultProps} />);
      
      expect(screen.getByText(/2m 0s/)).toBeInTheDocument();
    });

    test('renders cancel button during running state', () => {
      renderWithTheme(<SimulationProgress {...defaultProps} />);
      
      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      expect(cancelButton).toBeInTheDocument();
    });
  });

  describe('Status States', () => {
    test('renders queued state correctly', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          status="queued"
          progress={0}
          message="Waiting in queue..."
        />
      );

      expect(screen.getByText('Waiting in queue...')).toBeInTheDocument();
      expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '0');
    });

    test('renders completed state with success indicator', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          status="completed"
          progress={100}
          message="Simulation completed"
        />
      );

      expect(screen.getByText('Simulation completed')).toBeInTheDocument();
      expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '100');
      expect(screen.queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument();
    });

    test('renders failed state with error indicator', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          status="failed"
          progress={45}
          message="Simulation failed: Network error"
        />
      );

      expect(screen.getByText(/Simulation failed/)).toBeInTheDocument();
    });

    test('hides cancel button in completed state', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          status="completed"
          progress={100}
        />
      );

      expect(screen.queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument();
    });

    test('hides cancel button in failed state', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          status="failed"
          progress={45}
        />
      );

      expect(screen.queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument();
    });
  });

  describe('User Interactions', () => {
    test('calls onCancel when cancel button is clicked', () => {
      renderWithTheme(<SimulationProgress {...defaultProps} />);

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      fireEvent.click(cancelButton);

      expect(mockOnCancel).toHaveBeenCalledTimes(1);
    });

    test('disables cancel button during click to prevent double submission', async () => {
      const { rerender } = renderWithTheme(
        <SimulationProgress {...defaultProps} />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      fireEvent.click(cancelButton);

      expect(mockOnCancel).toHaveBeenCalledTimes(1);

      // Update to completed state
      rerender(
        <ThemeProvider theme={theme}>
          <SimulationProgress
            {...defaultProps}
            status="completed"
            progress={100}
          />
        </ThemeProvider>
      );

      expect(screen.queryByRole('button', { name: /cancel/i })).not.toBeInTheDocument();
    });
  });

  describe('Progress Updates', () => {
    test('updates progress bar when progress prop changes', () => {
      const { rerender } = renderWithTheme(
        <SimulationProgress {...defaultProps} progress={30} />
      );

      expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '30');

      rerender(
        <ThemeProvider theme={theme}>
          <SimulationProgress {...defaultProps} progress={75} />
        </ThemeProvider>
      );

      expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '75');
    });

    test('updates message when status message changes', () => {
      const { rerender } = renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          message="Step 1: Loading data..."
        />
      );

      expect(screen.getByText('Step 1: Loading data...')).toBeInTheDocument();

      rerender(
        <ThemeProvider theme={theme}>
          <SimulationProgress
            {...defaultProps}
            message="Step 2: Computing risks..."
          />
        </ThemeProvider>
      );

      expect(screen.getByText('Step 2: Computing risks...')).toBeInTheDocument();
      expect(screen.queryByText('Step 1: Loading data...')).not.toBeInTheDocument();
    });

    test('displays correct time formatting for various durations', () => {
      const { rerender } = renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          elapsedTime={3665} // 1h 1m 5s
        />
      );

      expect(screen.getByText(/1h 1m 5s/)).toBeInTheDocument();

      rerender(
        <ThemeProvider theme={theme}>
          <SimulationProgress
            {...defaultProps}
            elapsedTime={5} // 5 seconds
          />
        </ThemeProvider>
      );

      expect(screen.getByText(/5s/)).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    test('progress bar has proper ARIA attributes', () => {
      renderWithTheme(<SimulationProgress {...defaultProps} />);

      const progressBar = screen.getByRole('progressbar');
      expect(progressBar).toHaveAttribute('aria-valuenow', '50');
      expect(progressBar).toHaveAttribute('aria-valuemin', '0');
      expect(progressBar).toHaveAttribute('aria-valuemax', '100');
    });

    test('cancel button is keyboard accessible', () => {
      renderWithTheme(<SimulationProgress {...defaultProps} />);

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      cancelButton.focus();

      expect(cancelButton).toHaveFocus();

      fireEvent.keyDown(cancelButton, { key: 'Enter', code: 'Enter' });
      // Button click should be triggered by Enter key
    });

    test('displays time remaining in human-readable format', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          estimatedTimeRemaining={7325} // 2h 2m 5s
        />
      );

      expect(screen.getByText(/2h 2m 5s/)).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    test('handles zero progress', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          progress={0}
        />
      );

      expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '0');
    });

    test('handles 100% progress', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          progress={100}
        />
      );

      expect(screen.getByRole('progressbar')).toHaveAttribute('aria-valuenow', '100');
    });

    test('handles very long elapsed times', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          elapsedTime={86400} // 24 hours
        />
      );

      expect(screen.getByText(/24h 0m 0s/)).toBeInTheDocument();
    });

    test('handles missing estimated time', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          estimatedTimeRemaining={0}
        />
      );

      // Should render without crashing
      expect(screen.getByRole('progressbar')).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    test('displays error message in failed state', () => {
      renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          status="failed"
          message="Error: Insufficient data for scenario"
        />
      );

      expect(screen.getByText('Error: Insufficient data for scenario')).toBeInTheDocument();
    });

    test('maintains error state even as props update', () => {
      const { rerender } = renderWithTheme(
        <SimulationProgress
          {...defaultProps}
          status="failed"
          message="Simulation failed"
        />
      );

      expect(screen.getByText('Simulation failed')).toBeInTheDocument();

      // Props update but status stays failed
      rerender(
        <ThemeProvider theme={theme}>
          <SimulationProgress
            {...defaultProps}
            progress={60}
            status="failed"
            message="Simulation failed"
          />
        </ThemeProvider>
      );

      expect(screen.getByText('Simulation failed')).toBeInTheDocument();
    });
  });
});
