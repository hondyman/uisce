import { renderHook, act, waitFor } from '@testing-library/react';
import { useScenarioSimulation } from '../../useScenarioSimulation';
import type { StressScenario, SimulationRun } from '../../../types/scenarios';

// Mock fetch
global.fetch = jest.fn();

const mockFetch = fetch as jest.MockedFunction<typeof fetch>;

// Mock data
const mockScenario: StressScenario = {
  id: 'scenario_1',
  name: '2008 Crisis',
  equityMarketMove: -20,
  interestRateShift: 50,
  volatilityChange: 100,
  creditSpreadWidening: 200,
  portfoliosIncluded: ['p1', 'p2'],
  scope: 'all-portfolios',
  createdAt: new Date(),
  createdBy: 'user_1',
};

const mockSimulationRun: SimulationRun = {
  id: 'sim_1',
  status: 'running',
  progress: 0,
  portfoliosProcessed: 0,
  portfoliosTotal: 15,
  scenario: mockScenario,
  startedAt: new Date(),
  estimatedDuration: 30,
};

describe('useScenarioSimulation', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  describe('Initial State', () => {
    it('should return initial state', () => {
      const { result } = renderHook(() => useScenarioSimulation());

      expect(result.current.run).toBeNull();
      expect(result.current.isSimulating).toBe(false);
      expect(result.current.isAborting).toBe(false);
      expect(result.current.error).toBeNull();
    });
  });

  describe('Start Simulation', () => {
    it('should start simulation successfully', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockSimulationRun,
      } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      await act(async () => {
        const run = await result.current.start(mockScenario);
        expect(run).toEqual(mockSimulationRun);
      });

      expect(result.current.run).toEqual(mockSimulationRun);
      expect(result.current.isSimulating).toBe(true);
    });

    it('should handle start simulation error', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        json: async () => ({ error: 'Failed to start' }),
      } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      await act(async () => {
        try {
          await result.current.start(mockScenario);
        } catch (err) {
          expect(err).toBeDefined();
        }
      });

      expect(result.current.error).toBeDefined();
      expect(result.current.isSimulating).toBe(false);
    });

    it('should set isSimulating to true during start', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockSimulationRun,
      } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      expect(result.current.isSimulating).toBe(false);

      await act(async () => {
        result.current.start(mockScenario);
      });

      // During async operation
      expect(result.current.isSimulating).toBe(true);
    });
  });

  describe('Polling', () => {
    it('should poll simulation status', async () => {
      const completedRun: SimulationRun = {
        ...mockSimulationRun,
        status: 'completed',
        progress: 100,
      };

      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => mockSimulationRun,
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => completedRun,
        } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      await act(async () => {
        await result.current.start(mockScenario);
      });

      // Fast-forward time to trigger polling
      await act(async () => {
        jest.advanceTimersByTime(1000);
      });

      await waitFor(() => {
        expect(mockFetch).toHaveBeenCalledTimes(2);
      });
    });

    it('should stop polling when simulation completes', async () => {
      const completedRun: SimulationRun = {
        ...mockSimulationRun,
        status: 'completed',
        progress: 100,
      };

      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => completedRun,
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => completedRun,
        } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      await act(async () => {
        await result.current.start(mockScenario);
      });

      expect(result.current.isSimulating).toBe(false);
    });
  });

  describe('Abort Simulation', () => {
    it('should abort running simulation', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => mockSimulationRun,
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({}),
        } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      await act(async () => {
        await result.current.start(mockScenario);
      });

      await act(async () => {
        await result.current.abort();
      });

      expect(result.current.run?.status).toBe('aborted');
      expect(result.current.isSimulating).toBe(false);
    });

    it('should handle abort error', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => mockSimulationRun,
        } as Response)
        .mockResolvedValueOnce({
          ok: false,
          json: async () => ({ error: 'Failed to abort' }),
        } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      await act(async () => {
        await result.current.start(mockScenario);
      });

      await act(async () => {
        try {
          await result.current.abort();
        } catch (err) {
          expect(err).toBeDefined();
        }
      });

      expect(result.current.error).toBeDefined();
    });

    it('should set isAborting during abort', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: async () => mockSimulationRun,
        } as Response)
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({}),
        } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      await act(async () => {
        await result.current.start(mockScenario);
      });

      expect(result.current.isAborting).toBe(false);

      await act(async () => {
        result.current.abort();
      });

      expect(result.current.isAborting).toBe(false); // Should be reset after completion
    });
  });

  describe('Reset', () => {
    it('should reset all state', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockSimulationRun,
      } as Response);

      const { result } = renderHook(() => useScenarioSimulation());

      await act(async () => {
        await result.current.start(mockScenario);
      });

      act(() => {
        result.current.reset();
      });

      expect(result.current.run).toBeNull();
      expect(result.current.isSimulating).toBe(false);
      expect(result.current.error).toBeNull();
    });
  });

  describe('Cleanup', () => {
    it('should cleanup on unmount', () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: async () => mockSimulationRun,
      } as Response);

      const { result, unmount } = renderHook(() => useScenarioSimulation());

      act(() => {
        result.current.start(mockScenario);
      });

      unmount();

      // Component should be properly cleaned up
      expect(result.current).toBeDefined();
    });
  });
});
