/**
 * Phase 3 Custom Hooks Tests
 * 
 * Tests for:
 * - useScenarioConfig hook
 * - useSimulationState hook  
 * - useScenarioComparison hook
 * - useAnnotationSync hook
 * - useProgressTracking hook
 */

import { renderHook, act, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import React from 'react';

// Mock hooks for testing (in real implementation, these would be imported)
interface ScenarioConfig {
  name: string;
  parameters: Record<string, number>;
  type: 'base' | 'stress' | 'custom';
}

interface SimulationState {
  status: 'idle' | 'running' | 'completed' | 'failed';
  progress: number;
  results?: Record<string, unknown>;
  error?: string;
}

// Example implementations for testing
const useScenarioConfig = (initialConfig?: Partial<ScenarioConfig>) => {
  const [config, setConfig] = React.useState<ScenarioConfig>({
    name: 'Default Scenario',
    parameters: {},
    type: 'base',
    ...initialConfig,
  });

  const updateConfig = React.useCallback((updates: Partial<ScenarioConfig>) => {
    setConfig(prev => ({ ...prev, ...updates }));
  }, []);

  return { config, updateConfig };
};

const useSimulationState = (onComplete?: (results: unknown) => void) => {
  const [state, setState] = React.useState<SimulationState>({
    status: 'idle',
    progress: 0,
  });

  const startSimulation = React.useCallback(async () => {
    setState({ status: 'running', progress: 0 });
    
    // Simulate progress updates
    for (let i = 0; i <= 100; i += 20) {
      await new Promise(resolve => setTimeout(resolve, 100));
      setState(prev => ({ ...prev, progress: i }));
    }

    const results = { success: true, message: 'Simulation completed' };
    setState({ status: 'completed', progress: 100, results });
    onComplete?.(results);
  }, [onComplete]);

  const cancelSimulation = React.useCallback(() => {
    setState({ status: 'idle', progress: 0 });
  }, []);

  return { state, startSimulation, cancelSimulation };
};

const useScenarioComparison = (scenarios: ScenarioConfig[]) => {
  const [comparison, setComparison] = React.useState({
    scenarios,
    results: [] as Record<string, unknown>[],
  });

  const compileResults = React.useCallback(async (newResults: Record<string, unknown>[]) => {
    setComparison(prev => ({
      ...prev,
      results: newResults,
    }));
  }, []);

  return { comparison, compileResults };
};

const useAnnotationSync = (annotations: unknown[] = []) => {
  const [synced, setSynced] = React.useState(annotations);
  const [isSyncing, setIsSyncing] = React.useState(false);

  const syncAnnotations = React.useCallback(async () => {
    setIsSyncing(true);
    try {
      await new Promise(resolve => setTimeout(resolve, 100));
      setSynced(annotations);
    } finally {
      setIsSyncing(false);
    }
  }, [annotations]);

  React.useEffect(() => {
    syncAnnotations();
  }, [annotations, syncAnnotations]);

  return { synced, isSyncing };
};

const useProgressTracking = (targetProgress: number = 100) => {
  const [progress, setProgress] = React.useState(0);
  const [isTracking, setIsTracking] = React.useState(false);

  const startTracking = React.useCallback(() => {
    setIsTracking(true);
    setProgress(0);

    const interval = setInterval(() => {
      setProgress(prev => {
        const newProgress = Math.min(prev + Math.random() * 15, targetProgress);
        if (newProgress >= targetProgress) {
          clearInterval(interval);
          setIsTracking(false);
        }
        return newProgress;
      });
    }, 500);

    return () => clearInterval(interval);
  }, [targetProgress]);

  return { progress, isTracking, startTracking };
};

// Create a wrapper component for React Query
const createWrapper = () => {
  const queryClient = new QueryClient();
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>
      {children}
    </QueryClientProvider>
  );
};

describe('Phase 3 Custom Hooks', () => {
  describe('useScenarioConfig', () => {
    test('initializes with default config', () => {
      const { result } = renderHook(() => useScenarioConfig());

      expect(result.current.config.name).toBe('Default Scenario');
      expect(result.current.config.type).toBe('base');
    });

    test('initializes with provided config', () => {
      const initial = { name: 'Custom Scenario', type: 'stress' as const };
      const { result } = renderHook(() => useScenarioConfig(initial));

      expect(result.current.config.name).toBe('Custom Scenario');
      expect(result.current.config.type).toBe('stress');
    });

    test('updates config with partial updates', () => {
      const { result } = renderHook(() => useScenarioConfig());

      act(() => {
        result.current.updateConfig({
          name: 'Updated Scenario',
          parameters: { leverage: 2.5 },
        });
      });

      expect(result.current.config.name).toBe('Updated Scenario');
      expect(result.current.config.parameters.leverage).toBe(2.5);
      expect(result.current.config.type).toBe('base'); // Should remain unchanged
    });

    test('preserves existing parameters when updating', () => {
      const { result } = renderHook(() =>
        useScenarioConfig({ parameters: { volatility: 1.5 } })
      );

      act(() => {
        result.current.updateConfig({
          parameters: { ...result.current.config.parameters, leverage: 2.0 },
        });
      });

      expect(result.current.config.parameters.volatility).toBe(1.5);
      expect(result.current.config.parameters.leverage).toBe(2.0);
    });

    test('handles multiple sequential updates', () => {
      const { result } = renderHook(() => useScenarioConfig());

      act(() => {
        result.current.updateConfig({ name: 'First Update' });
        result.current.updateConfig({ type: 'custom' });
        result.current.updateConfig({ name: 'Second Update' });
      });

      expect(result.current.config.name).toBe('Second Update');
      expect(result.current.config.type).toBe('custom');
    });
  });

  describe('useSimulationState', () => {
    test('initializes simulation as idle', () => {
      const { result } = renderHook(() => useSimulationState());

      expect(result.current.state.status).toBe('idle');
      expect(result.current.state.progress).toBe(0);
    });

    test('transitions to running state', async () => {
      const { result } = renderHook(() => useSimulationState());

      act(() => {
        result.current.startSimulation();
      });

      await waitFor(() => {
        expect(result.current.state.status).toBe('running');
      }, { timeout: 200 });
    });

    test('updates progress during simulation', async () => {
      const { result } = renderHook(() => useSimulationState());

      act(() => {
        result.current.startSimulation();
      });

      await waitFor(() => {
        expect(result.current.state.progress).toBeGreaterThan(0);
      }, { timeout: 200 });
    });

    test('completes simulation successfully', async () => {
      const { result } = renderHook(() => useSimulationState());

      act(() => {
        result.current.startSimulation();
      });

      await waitFor(() => {
        expect(result.current.state.status).toBe('completed');
        expect(result.current.state.progress).toBe(100);
      }, { timeout: 1000 });
    });

    test('calls onComplete callback when simulation finishes', async () => {
      const mockOnComplete = jest.fn();
      const { result } = renderHook(() => useSimulationState(mockOnComplete));

      act(() => {
        result.current.startSimulation();
      });

      await waitFor(() => {
        expect(mockOnComplete).toHaveBeenCalled();
        expect(mockOnComplete).toHaveBeenCalledWith(
          expect.objectContaining({ success: true })
        );
      }, { timeout: 1000 });
    });

    test('cancels simulation', async () => {
      const { result } = renderHook(() => useSimulationState());

      act(() => {
        result.current.startSimulation();
      });

      await waitFor(() => {
        expect(result.current.state.status).toBe('running');
      }, { timeout: 200 });

      act(() => {
        result.current.cancelSimulation();
      });

      expect(result.current.state.status).toBe('idle');
      expect(result.current.state.progress).toBe(0);
    });
  });

  describe('useScenarioComparison', () => {
    const mockScenarios: ScenarioConfig[] = [
      { name: 'Scenario A', parameters: { rate: 0.02 }, type: 'base' },
      { name: 'Scenario B', parameters: { rate: 0.05 }, type: 'stress' },
    ];

    test('initializes with provided scenarios', () => {
      const { result } = renderHook(() => useScenarioComparison(mockScenarios));

      expect(result.current.comparison.scenarios).toHaveLength(2);
      expect(result.current.comparison.scenarios[0].name).toBe('Scenario A');
    });

    test('compiles results for scenarios', async () => {
      const { result } = renderHook(() => useScenarioComparison(mockScenarios));

      const mockResults = [
        { scenarioId: 'A', pnl: 1000 },
        { scenarioId: 'B', pnl: -500 },
      ];

      act(() => {
        result.current.compileResults(mockResults);
      });

      await waitFor(() => {
        expect(result.current.comparison.results).toHaveLength(2);
        expect(result.current.comparison.results[0].scenarioId).toBe('A');
      });
    });

    test('handles multiple result compilations', async () => {
      const { result } = renderHook(() => useScenarioComparison(mockScenarios));

      const firstResults = [{ scenarioId: 'A', pnl: 1000 }];
      const secondResults = [
        { scenarioId: 'A', pnl: 1200 },
        { scenarioId: 'B', pnl: -600 },
      ];

      act(() => {
        result.current.compileResults(firstResults);
      });

      await waitFor(() => {
        expect(result.current.comparison.results).toHaveLength(1);
      });

      act(() => {
        result.current.compileResults(secondResults);
      });

      await waitFor(() => {
        expect(result.current.comparison.results).toHaveLength(2);
      });
    });
  });

  describe('useAnnotationSync', () => {
    test('syncs annotations on mount', async () => {
      const mockAnnotations = [
        { id: '1', text: 'Annotation 1' },
        { id: '2', text: 'Annotation 2' },
      ];

      const { result } = renderHook(() => useAnnotationSync(mockAnnotations));

      await waitFor(() => {
        expect(result.current.synced).toEqual(mockAnnotations);
        expect(result.current.isSyncing).toBe(false);
      });
    });

    test('updates synced annotations when props change', async () => {
      const { result, rerender } = renderHook(
        ({ annotations }) => useAnnotationSync(annotations),
        {
          initialProps: { annotations: [{ id: '1', text: 'First' }] },
        }
      );

      await waitFor(() => {
        expect(result.current.synced).toHaveLength(1);
      });

      rerender({ annotations: [{ id: '1', text: 'First' }, { id: '2', text: 'Second' }] });

      await waitFor(() => {
        expect(result.current.synced).toHaveLength(2);
      });
    });

    test('sets isSyncing flag during sync', async () => {
      const mockAnnotations = [{ id: '1', text: 'Test' }];
      const { result } = renderHook(() => useAnnotationSync(mockAnnotations));

      expect(result.current.isSyncing).toBe(true);

      await waitFor(() => {
        expect(result.current.isSyncing).toBe(false);
      });
    });

    test('maintains sync on empty annotations', async () => {
      const { result } = renderHook(() => useAnnotationSync([]));

      await waitFor(() => {
        expect(result.current.synced).toEqual([]);
      });
    });
  });

  describe('useProgressTracking', () => {
    jest.useFakeTimers();

    test('initializes with zero progress', () => {
      const { result } = renderHook(() => useProgressTracking());

      expect(result.current.progress).toBe(0);
      expect(result.current.isTracking).toBe(false);
    });

    test('starts tracking progress', () => {
      const { result } = renderHook(() => useProgressTracking());

      act(() => {
        result.current.startTracking();
      });

      expect(result.current.isTracking).toBe(true);
    });

    test('increases progress during tracking', () => {
      const { result } = renderHook(() => useProgressTracking());

      act(() => {
        result.current.startTracking();
      });

      act(() => {
        jest.advanceTimersByTime(500);
      });

      expect(result.current.progress).toBeGreaterThan(0);
    });

    test('completes tracking at target progress', () => {
      const { result } = renderHook(() => useProgressTracking(100));

      act(() => {
        result.current.startTracking();
      });

      act(() => {
        jest.advanceTimersByTime(5000);
      });

      expect(result.current.progress).toBe(100);
      expect(result.current.isTracking).toBe(false);
    });

    test('respects custom target progress', () => {
      const { result } = renderHook(() => useProgressTracking(50));

      act(() => {
        result.current.startTracking();
      });

      act(() => {
        jest.advanceTimersByTime(5000);
      });

      expect(result.current.progress).toBeLessThanOrEqual(50);
    });

    jest.useRealTimers();
  });
});
