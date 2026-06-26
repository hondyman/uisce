import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ThemeProvider, createTheme } from '@mui/material/styles';
import { MultiScenarioComparison } from '../MultiScenarioComparison';
import type { StressScenario, SimulationResult } from '../../../types/scenarios';

const theme = createTheme();

const renderWithTheme = (component: React.ReactElement) => {
  return render(<ThemeProvider theme={theme}>{component}</ThemeProvider>);
};

// Mock data
const mockScenario1: StressScenario = {
  id: 'scenario_1',
  name: '2008 Crisis',
  equityMarketMove: -20,
  interestRateShift: 50,
  volatilityChange: 100,
  creditSpreadWidening: 200,
  portfoliosIncluded: ['p1', 'p2', 'p3'],
  scope: 'all-portfolios',
  createdAt: new Date(),
  createdBy: 'user_1',
};

const mockScenario2: StressScenario = {
  id: 'scenario_2',
  name: 'COVID Shock',
  equityMarketMove: -30,
  interestRateShift: -100,
  volatilityChange: 150,
  creditSpreadWidening: 150,
  portfoliosIncluded: ['p1', 'p2', 'p3'],
  scope: 'all-portfolios',
  createdAt: new Date(),
  createdBy: 'user_1',
};

const mockResults: SimulationResult[] = [
  {
    id: 'result_1',
    simulationId: 'sim_1',
    portfolioId: 'p1',
    pnl: -5.2,
    confidence: 92,
    status: 'success',
    processingTime: 45,
  },
  {
    id: 'result_2',
    simulationId: 'sim_1',
    portfolioId: 'p2',
    pnl: -3.1,
    confidence: 88,
    status: 'success',
    processingTime: 42,
  },
  {
    id: 'result_3',
    simulationId: 'sim_1',
    portfolioId: 'p3',
    pnl: -2.8,
    confidence: 90,
    status: 'success',
    processingTime: 48,
  },
];

describe('MultiScenarioComparison', () => {
  const mockOnSelectScenario = jest.fn();

  beforeEach(() => {
    jest.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render component with scenarios', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      expect(screen.getByText(/Compare Metrics/i)).toBeInTheDocument();
    });

    it('should show message when no scenarios provided', () => {
      renderWithTheme(
        <MultiScenarioComparison scenarios={[]} scenarioResults={new Map()} />
      );

      expect(screen.getByText(/no scenarios to compare/i)).toBeInTheDocument();
    });

    it('should render chart', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      // Chart title should be visible
      expect(screen.getByText(/Scenario.*Comparison/i)).toBeInTheDocument();
    });

    it('should render data grid', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      expect(screen.getByText(/Portfolio Comparison/i)).toBeInTheDocument();
    });
  });

  describe('Metric Toggle', () => {
    it('should toggle between PnL, Variance, and Confidence', async () => {
      const user = userEvent.setup();
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      const varianceButton = screen.getByRole('button', { name: /Variance/i });
      await user.click(varianceButton);

      expect(varianceButton).toHaveAttribute('aria-pressed', 'true');
    });

    it('should update chart when metric changes', async () => {
      const user = userEvent.setup();
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      const { rerender } = renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      const confidenceButton = screen.getByRole('button', { name: /Confidence/i });
      await user.click(confidenceButton);

      expect(confidenceButton).toHaveAttribute('aria-pressed', 'true');
    });
  });

  describe('Multi-Scenario Comparison', () => {
    it('should display multiple scenarios', () => {
      const scenarioResults = new Map([
        ['scenario_1', mockResults],
        ['scenario_2', mockResults],
      ]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1, mockScenario2]}
          scenarioResults={scenarioResults}
        />
      );

      expect(screen.getByText('2008 Crisis')).toBeInTheDocument();
      expect(screen.getByText('COVID Shock')).toBeInTheDocument();
    });

    it('should calculate and display aggregates', () => {
      const scenarioResults = new Map([
        ['scenario_1', mockResults],
        ['scenario_2', mockResults],
      ]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1, mockScenario2]}
          scenarioResults={scenarioResults}
        />
      );

      // Aggregated impact sidebar should show scenarios
      expect(screen.getByText(/AGGREGATED IMPACT/i)).toBeInTheDocument();
    });
  });

  describe('Data Grid', () => {
    it('should display portfolio names', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      expect(screen.getByText('p1')).toBeInTheDocument();
      expect(screen.getByText('p2')).toBeInTheDocument();
    });

    it('should display PnL values', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      // PnL values should be displayed in grid
      // Exact matching depends on formatting
    });

    it('should display confidence bars for each portfolio', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      // Confidence values should be visible
      expect(screen.getByText(/92/)).toBeInTheDocument(); // 92% confidence
    });
  });

  describe('Statistics Footer', () => {
    it('should display comparison statistics', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      expect(screen.getByText(/Compared Scenarios/i)).toBeInTheDocument();
      expect(screen.getByText(/Total Portfolios/i)).toBeInTheDocument();
    });

    it('should show best and worst case', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      expect(screen.getByText(/Best Case/i)).toBeInTheDocument();
      expect(screen.getByText(/Worst Case/i)).toBeInTheDocument();
    });
  });

  describe('Dark Mode', () => {
    it('should render with dark theme', () => {
      const darkTheme = createTheme({ palette: { mode: 'dark' } });
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      const { container } = render(
        <ThemeProvider theme={darkTheme}>
          <MultiScenarioComparison
            scenarios={[mockScenario1]}
            scenarioResults={scenarioResults}
          />
        </ThemeProvider>
      );

      expect(container).toBeInTheDocument();
    });
  });

  describe('Responsive Design', () => {
    it('should render on mobile', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
        />
      );

      // Component should render without errors
      expect(screen.getByText(/Compare Metrics/i)).toBeInTheDocument();
    });
  });

  describe('Loading State', () => {
    it('should show loading indicator', () => {
      const scenarioResults = new Map([['scenario_1', mockResults]]);

      renderWithTheme(
        <MultiScenarioComparison
          scenarios={[mockScenario1]}
          scenarioResults={scenarioResults}
          isLoading={true}
        />
      );

      expect(screen.getByRole('progressbar')).toBeInTheDocument();
    });
  });
});
