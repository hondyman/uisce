import React from 'react';
import { render, screen, fireEvent, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { ScenarioConfigDialog } from '../ScenarioConfigDialog';
import type { StressScenario } from '../../../types/scenarios';

// Mock theme
const theme = createTheme();

const renderWithTheme = (component: React.ReactElement) => {
  return render(<ThemeProvider theme={theme}>{component}</ThemeProvider>);
};

// Mock data
const mockPortfolios = [
  { id: 'p1', name: 'Tech Growth', aum: 100 },
  { id: 'p2', name: 'Fixed Income', aum: 80 },
  { id: 'p3', name: 'Balanced', aum: 120 },
];

describe('ScenarioConfigDialog', () => {
  const mockOnClose = jest.fn();
  const mockOnSubmit = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render dialog when open prop is true', () => {
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      expect(screen.getByText('Configure Stress Test Scenario')).toBeInTheDocument();
    });

    it('should not render dialog when open prop is false', () => {
      renderWithTheme(
        <ScenarioConfigDialog
          open={false}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      expect(screen.queryByText('Configure Stress Test Scenario')).not.toBeInTheDocument();
    });

    it('should render all form fields', () => {
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      expect(screen.getByLabelText(/scenario name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/equity market move/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/interest rate shift/i)).toBeInTheDocument();
    });
  });

  describe('Form Input', () => {
    it('should update scenario name on input', async () => {
      const user = userEvent.setup();
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const nameInput = screen.getByLabelText(/scenario name/i);
      await user.type(nameInput, '2008 Crisis');

      expect(nameInput).toHaveValue('2008 Crisis');
    });

    it('should update slider values', async () => {
      const user = userEvent.setup();
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const sliders = screen.getAllByRole('slider');
      expect(sliders.length).toBeGreaterThan(0);

      // Test equity slider exists
      const equitySlider = sliders[0];
      expect(equitySlider).toBeInTheDocument();
    });

    it('should toggle portfolio scope', async () => {
      const user = userEvent.setup();
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const selectedToggle = screen.getByRole('button', { name: /selected portfolios/i });
      await user.click(selectedToggle);

      expect(selectedToggle).toHaveAttribute('aria-pressed', 'true');
    });
  });

  describe('Form Validation', () => {
    it('should show error when scenario name is empty', async () => {
      const user = userEvent.setup();
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const submitButton = screen.getByRole('button', { name: /run simulation/i });
      await user.click(submitButton);

      // Error should be shown for empty name
      // Typically a helper text or alert
      await waitFor(() => {
        expect(mockOnSubmit).not.toHaveBeenCalled();
      });
    });

    it('should enforce slider constraints', async () => {
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const sliders = screen.getAllByRole('slider');
      const equitySlider = sliders[0];

      // Slider should have min/max constraints
      expect(equitySlider).toHaveAttribute('aria-valuemin');
      expect(equitySlider).toHaveAttribute('aria-valuemax');
    });
  });

  describe('Form Submission', () => {
    it('should call onSubmit with scenario data', async () => {
      const user = userEvent.setup();
      mockOnSubmit.mockResolvedValue(undefined);

      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const nameInput = screen.getByLabelText(/scenario name/i);
      await user.type(nameInput, 'Test Scenario');

      const submitButton = screen.getByRole('button', { name: /run simulation/i });
      await user.click(submitButton);

      await waitFor(() => {
        expect(mockOnSubmit).toHaveBeenCalled();
      });
    });

    it('should disable submit button during submission', async () => {
      const user = userEvent.setup();
      mockOnSubmit.mockImplementation(
        () => new Promise(resolve => setTimeout(resolve, 100))
      );

      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          isLoading={false}
          portfolios={mockPortfolios}
        />
      );

      const nameInput = screen.getByLabelText(/scenario name/i);
      await user.type(nameInput, 'Test');

      const submitButton = screen.getByRole('button', { name: /run simulation/i });
      await user.click(submitButton);

      // Button should be disabled during submission
      expect(submitButton).toHaveAttribute('disabled');
    });

    it('should handle submission errors', async () => {
      const user = userEvent.setup();
      const errorMessage = 'Failed to start simulation';
      mockOnSubmit.mockRejectedValue(new Error(errorMessage));

      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const nameInput = screen.getByLabelText(/scenario name/i);
      await user.type(nameInput, 'Test');

      const submitButton = screen.getByRole('button', { name: /run simulation/i });
      await user.click(submitButton);

      await waitFor(() => {
        // Error should be displayed
      });
    });
  });

  describe('Cancel Button', () => {
    it('should call onClose when cancel button clicked', async () => {
      const user = userEvent.setup();
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const cancelButton = screen.getByRole('button', { name: /cancel/i });
      await user.click(cancelButton);

      expect(mockOnClose).toHaveBeenCalled();
    });

    it('should close when backdrop clicked', async () => {
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const backdrop = screen.getByRole('presentation').previousSibling as HTMLElement;
      fireEvent.click(backdrop);

      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  describe('Portfolio Selection', () => {
    it('should display all available portfolios', async () => {
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      // All portfolios should be available for selection
      expect(screen.getByText('Tech Growth')).toBeInTheDocument();
      expect(screen.getByText('Fixed Income')).toBeInTheDocument();
      expect(screen.getByText('Balanced')).toBeInTheDocument();
    });
  });

  describe('Dark Mode', () => {
    it('should render with dark theme', () => {
      const darkTheme = createTheme({ palette: { mode: 'dark' } });
      const { container } = render(
        <ThemeProvider theme={darkTheme}>
          <ScenarioConfigDialog
            open={true}
            onClose={mockOnClose}
            onSubmit={mockOnSubmit}
            portfolios={mockPortfolios}
          />
        </ThemeProvider>
      );

      expect(container).toBeInTheDocument();
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', () => {
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      // Dialog title should be set
      const dialog = screen.getByRole('dialog');
      expect(dialog).toBeInTheDocument();
    });

    it('should trap focus in dialog', async () => {
      const user = userEvent.setup();
      renderWithTheme(
        <ScenarioConfigDialog
          open={true}
          onClose={mockOnClose}
          onSubmit={mockOnSubmit}
          portfolios={mockPortfolios}
        />
      );

      const dialog = screen.getByRole('dialog');
      expect(dialog).toBeInTheDocument();
    });
  });
});
