import { useCallback, useMemo, useState } from 'react';
import type {
  SimulationResult,
  ScenarioComparison,
  StressScenario,
} from '../types/scenarios';

/**
 * Compares multiple scenario results
 * Calculates variance, winner/loser portfolios, and aggregates
 */
const calculateComparison = (
  scenarios: StressScenario[],
  results: Map<string, SimulationResult[]>
): ScenarioComparison => {
  const scenarioResults = scenarios.map(scenario => ({
    scenarioId: scenario.id,
    scenarioName: scenario.name,
    totalPnL: 0,
    avgPnL: 0,
    avgConfidence: 0,
    portfolioCount: 0,
    minPnL: 0,
    maxPnL: 0,
    variance: 0,
  }));

  // Calculate aggregates for each scenario
  scenarios.forEach(scenario => {
    const scenarioRes = results.get(scenario.id) || [];
    if (scenarioRes.length === 0) return;

    const pnls = scenarioRes.map(r => r.pnl);
    const totalPnL = pnls.reduce((sum, pnl) => sum + pnl, 0);
    const avgPnL = totalPnL / pnls.length;
    const avgConfidence = 
      scenarioRes.reduce((sum, r) => sum + r.confidence, 0) / scenarioRes.length;

    const variance =
      pnls.reduce((sum, pnl) => sum + Math.pow(pnl - avgPnL, 2), 0) / pnls.length;

    const idx = scenarioResults.findIndex(r => r.scenarioId === scenario.id);
    if (idx >= 0) {
      scenarioResults[idx] = {
        ...scenarioResults[idx],
        totalPnL,
        avgPnL,
        avgConfidence,
        portfolioCount: pnls.length,
        minPnL: Math.min(...pnls),
        maxPnL: Math.max(...pnls),
        variance: Math.sqrt(variance),
      };
    }
  });

  return {
    id: `comparison_${Date.now()}`,
    scenarios: scenarioResults,
    timestamp: new Date(),
    metadata: {
      comparedScenarioCount: scenarios.length,
      totalPortfolios: Array.from(results.values()).reduce(
        (sum, res) => Math.max(sum, res.length),
        0
      ),
    },
  };
};

/**
 * Hook for comparing multiple scenario simulation results
 * Calculates variance, aggregates, and comparative metrics
 *
 * @example
 * const { comparison, isCalculating, addScenario, removeScenario } =
 *   useScenarioComparison();
 *
 * // Add scenarios to compare
 * addScenario(scenario1, results1);
 * addScenario(scenario2, results2);
 *
 * // Access comparison metrics
 * console.log(comparison?.scenarios[0].avgPnL);
 */
export function useScenarioComparison() {
  const [scenarios, setScenarios] = useState<StressScenario[]>([]);
  const [results, setResults] = useState<Map<string, SimulationResult[]>>(new Map());
  const [isCalculating, setIsCalculating] = useState(false);

  // Calculate comparison whenever scenarios or results change
  const comparison = useMemo(() => {
    if (scenarios.length === 0) return null;
    setIsCalculating(true);
    try {
      return calculateComparison(scenarios, results);
    } finally {
      setIsCalculating(false);
    }
  }, [scenarios, results]);

  // Add scenario to comparison
  const addScenario = useCallback(
    (scenario: StressScenario, scenarioResults: SimulationResult[]) => {
      setScenarios(prev => {
        const exists = prev.find(s => s.id === scenario.id);
        if (exists) return prev;
        return [...prev, scenario];
      });

      setResults(prev => {
        const map = new Map(prev);
        map.set(scenario.id, scenarioResults);
        return map;
      });
    },
    []
  );

  // Remove scenario from comparison
  const removeScenario = useCallback((scenarioId: string) => {
    setScenarios(prev => prev.filter(s => s.id !== scenarioId));
    setResults(prev => {
      const map = new Map(prev);
      map.delete(scenarioId);
      return map;
    });
  }, []);

  // Clear all scenarios
  const clear = useCallback(() => {
    setScenarios([]);
    setResults(new Map());
  }, []);

  // Get results for specific scenario
  const getScenarioResults = useCallback(
    (scenarioId: string): SimulationResult[] => {
      return results.get(scenarioId) || [];
    },
    [results]
  );

  // Get comparative ranking of scenarios by PnL
  const getRanking = useCallback(
    (metric: 'avgPnL' | 'variance' | 'confidence') => {
      if (!comparison) return [];
      return [...comparison.scenarios].sort((a, b) => {
        if (metric === 'variance') return a.variance - b.variance; // Lower is better
        if (metric === 'confidence') return b.avgConfidence - a.avgConfidence; // Higher is better
        return b.avgPnL - a.avgPnL; // Higher PnL is better
      });
    },
    [comparison]
  );

  // Update scenario results
  const updateResults = useCallback((scenarioId: string, newResults: SimulationResult[]) => {
    setResults(prev => {
      const map = new Map(prev);
      map.set(scenarioId, newResults);
      return map;
    });
  }, []);

  return {
    scenarios,
    comparison,
    isCalculating,
    addScenario,
    removeScenario,
    clear,
    getScenarioResults,
    getRanking,
    updateResults,
  };
}

export type UseScenarioComparisonReturn = ReturnType<typeof useScenarioComparison>;
