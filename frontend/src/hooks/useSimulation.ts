import { useState, useCallback } from 'react';
import * as ruleService from '../services/ruleService';

export interface SimulationInput {
  testData: any;
  rule: any;
}

export interface ExecutionTrace {
  date: string;
  region: string;
  winningRule: string;
  confidence: number;
  evaluatedRules: Array<{
    priority: number;
    condition: string;
    matched: boolean;
  }>;
}

export interface SimulationResults {
  executionTrace: ExecutionTrace[];
  impactedDates: number;
  changedDates: number;
  avgConfidence: number;
  samples?: Array<{
    date: string;
    before: string;
    after: string;
  }>;
}

export interface UseSimulationReturn {
  results: SimulationResults | null;
  loading: boolean;
  error: string | null;
  runSimulation: (ruleId: string, testData: any) => Promise<void>;
}

/**
 * useSimulation Hook
 *
 * Executes rules against test data in real-time by calling the backend
 * rule simulation engine.
 *
 * Features:
 * - Backend execution against actual calendar MDM data
 * - Execution trace showing which rules matched
 * - Impact analysis for rule changes
 * - Confidence metrics and before/after comparison
 * - Full audit trail via backend
 *
 * @returns Hook state with simulation results and execution method
 */
export const useSimulation = (): UseSimulationReturn => {
  const [results, setResults] = useState<SimulationResults | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const runSimulation = useCallback(async (ruleId: string, testData: any) => {
    if (!ruleId || !testData) return;

    try {
      setLoading(true);
      setError(null);

      // Call backend simulation endpoint
      const response = await ruleService.simulateRule(ruleId, testData);
      setResults(response);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Simulation failed';
      setError(message);
      console.error('Simulation error:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  return {
    results,
    loading,
    error,
    runSimulation,
  };
};

export default useSimulation;
