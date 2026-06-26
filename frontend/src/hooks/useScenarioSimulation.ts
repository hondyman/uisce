import { useCallback, useEffect, useRef, useState } from 'react';
import type {
  SimulationRun,
  StressScenario,
} from '../types/scenarios';

/**
 * API Service for scenario simulation operations
 */
const scenarioSimulationService = {
  async startSimulation(scenario: StressScenario): Promise<SimulationRun> {
    const response = await fetch('/api/v1/simulations', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        scenarioId: scenario.id,
        portfolioIds: scenario.portfoliosIncluded,
      }),
    });
    if (!response.ok) throw new Error('Failed to start simulation');
    return response.json();
  },

  async getSimulationStatus(simulationId: string): Promise<SimulationRun> {
    const response = await fetch(`/api/v1/simulations/${simulationId}`);
    if (!response.ok) throw new Error('Failed to fetch simulation status');
    return response.json();
  },

  async abortSimulation(simulationId: string): Promise<void> {
    const response = await fetch(`/api/v1/simulations/${simulationId}`, {
      method: 'DELETE',
    });
    if (!response.ok) throw new Error('Failed to abort simulation');
  },
};

/**
 * Hook for managing scenario simulation lifecycle
 * Handles starting simulations, polling status, aborting, and cleanup
 *
 * @example
 * const { run, isSimulating, error, start, abort } = useScenarioSimulation();
 * await start(scenario);
 * // ... simulation runs in background with polling
 * if (run?.status === 'completed') { // access results }
 */
export function useScenarioSimulation() {
  const [run, setRun] = useState<SimulationRun | null>(null);
  const [isSimulating, setIsSimulating] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [isAborting, setIsAborting] = useState(false);

  const pollingIntervalRef = useRef<NodeJS.Timeout | null>(null);

  // Poll simulation status every 1 second while running
  useEffect(() => {
    if (!run || run.status === 'completed' || run.status === 'failed' || run.status === 'aborted') {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
        pollingIntervalRef.current = null;
      }
      return;
    }

    pollingIntervalRef.current = setInterval(async () => {
      try {
        const updated = await scenarioSimulationService.getSimulationStatus(run.id);
        setRun(updated);

        if (updated.status === 'completed' || updated.status === 'failed') {
          setIsSimulating(false);
        }
      } catch (err) {
        console.error('Polling failed:', err);
        setError(err instanceof Error ? err : new Error('Unknown error'));
      }
    }, 1000);

    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
        pollingIntervalRef.current = null;
      }
    };
  }, [run]);

  // Start a new simulation
  const start = useCallback(async (scenario: StressScenario): Promise<SimulationRun> => {
    setError(null);
    setIsSimulating(true);

    try {
      const newRun = await scenarioSimulationService.startSimulation(scenario);
      setRun(newRun);
      return newRun;
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to start simulation');
      setError(error);
      setIsSimulating(false);
      throw error;
    }
  }, []);

  // Abort running simulation
  const abort = useCallback(async (): Promise<void> => {
    if (!run) return;

    setIsAborting(true);
    try {
      await scenarioSimulationService.abortSimulation(run.id);
      setRun(prev => (prev ? { ...prev, status: 'aborted' as const } : null));
      setIsSimulating(false);
    } catch (err) {
      const error = err instanceof Error ? err : new Error('Failed to abort simulation');
      setError(error);
    } finally {
      setIsAborting(false);
    }
  }, [run]);

  // Reset state
  const reset = useCallback(() => {
    setRun(null);
    setIsSimulating(false);
    setError(null);
    setIsAborting(false);

    if (pollingIntervalRef.current) {
      clearInterval(pollingIntervalRef.current);
      pollingIntervalRef.current = null;
    }
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (pollingIntervalRef.current) {
        clearInterval(pollingIntervalRef.current);
      }
    };
  }, []);

  return {
    run,
    isSimulating,
    isAborting,
    error,
    start,
    abort,
    reset,
  };
}

export type UseScenarioSimulationReturn = ReturnType<typeof useScenarioSimulation>;
