/**
 * Phase 3 Utility Functions Tests
 * 
 * Tests for:
 * - formatScenarioName
 * - calculateMetrics
 * - parseScenarioResults
 * - generateProjection
 * - aggregateResults
 * - formatTimeRemaining
 * - getStatusColor
 * - calculatePercentageChange
 */

describe('Phase 3 Utility Functions', () => {
  describe('formatScenarioName', () => {
    const formatScenarioName = (name: string, type: string): string => {
      return `${name} (${type.charAt(0).toUpperCase() + type.slice(1)})`;
    };

    test('formats scenario name with base type', () => {
      expect(formatScenarioName('Conservative', 'base')).toBe('Conservative (Base)');
    });

    test('formats scenario name with stress type', () => {
      expect(formatScenarioName('Market Crash', 'stress')).toBe('Market Crash (Stress)');
    });

    test('formats scenario name with custom type', () => {
      expect(formatScenarioName('User Defined', 'custom')).toBe('User Defined (Custom)');
    });

    test('handles empty strings', () => {
      expect(formatScenarioName('', 'base')).toBe(' (Base)');
    });

    test('handles special characters', () => {
      expect(formatScenarioName('Test & Verify', 'base')).toBe('Test & Verify (Base)');
    });
  });

  describe('calculateMetrics', () => {
    const calculateMetrics = (data: number[]): Record<string, number> => {
      if (data.length === 0) return { mean: 0, min: 0, max: 0, std: 0 };

      const mean = data.reduce((a, b) => a + b, 0) / data.length;
      const min = Math.min(...data);
      const max = Math.max(...data);

      const variance = data.reduce((sum, val) => sum + Math.pow(val - mean, 2), 0) / data.length;
      const std = Math.sqrt(variance);

      return { mean, min, max, std };
    };

    test('calculates metrics for positive data', () => {
      const data = [1, 2, 3, 4, 5];
      const metrics = calculateMetrics(data);

      expect(metrics.mean).toBe(3);
      expect(metrics.min).toBe(1);
      expect(metrics.max).toBe(5);
      expect(metrics.std).toBeCloseTo(1.41, 1);
    });

    test('calculates metrics for negative data', () => {
      const data = [-5, -3, -1];
      const metrics = calculateMetrics(data);

      expect(metrics.mean).toBe(-3);
      expect(metrics.min).toBe(-5);
      expect(metrics.max).toBe(-1);
    });

    test('handles single value', () => {
      const data = [42];
      const metrics = calculateMetrics(data);

      expect(metrics.mean).toBe(42);
      expect(metrics.min).toBe(42);
      expect(metrics.max).toBe(42);
      expect(metrics.std).toBe(0);
    });

    test('handles empty array', () => {
      const metrics = calculateMetrics([]);

      expect(metrics.mean).toBe(0);
      expect(metrics.min).toBe(0);
      expect(metrics.max).toBe(0);
      expect(metrics.std).toBe(0);
    });

    test('handles decimal values', () => {
      const data = [1.5, 2.5, 3.5];
      const metrics = calculateMetrics(data);

      expect(metrics.mean).toBe(2.5);
      expect(metrics.min).toBe(1.5);
      expect(metrics.max).toBe(3.5);
    });
  });

  describe('parseScenarioResults', () => {
    const parseScenarioResults = (
      data: Record<string, unknown>
    ): { success: boolean; values: number[] } => {
      try {
        const values = data.values as number[];
        if (!Array.isArray(values) || values.some(v => typeof v !== 'number')) {
          throw new Error('Invalid values');
        }
        return { success: true, values };
      } catch (error) {
        return { success: false, values: [] };
      }
    };

    test('parses valid scenario results', () => {
      const data = { values: [100, 110, 120] };
      const result = parseScenarioResults(data);

      expect(result.success).toBe(true);
      expect(result.values).toEqual([100, 110, 120]);
    });

    test('handles missing values field', () => {
      const data = {};
      const result = parseScenarioResults(data);

      expect(result.success).toBe(false);
      expect(result.values).toEqual([]);
    });

    test('handles non-array values', () => {
      const data = { values: 'not an array' };
      const result = parseScenarioResults(data);

      expect(result.success).toBe(false);
    });

    test('handles non-numeric values', () => {
      const data = { values: [100, 'string', 120] };
      const result = parseScenarioResults(data);

      expect(result.success).toBe(false);
    });

    test('handles empty values array', () => {
      const data = { values: [] };
      const result = parseScenarioResults(data);

      expect(result.success).toBe(true);
      expect(result.values).toEqual([]);
    });
  });

  describe('generateProjection', () => {
    const generateProjection = (
      baseValue: number,
      growthRate: number,
      periods: number
    ): number[] => {
      const projection = [];
      let value = baseValue;

      for (let i = 0; i < periods; i++) {
        projection.push(value);
        value *= 1 + growthRate;
      }

      return projection;
    };

    test('generates projection with positive growth', () => {
      const projection = generateProjection(100, 0.1, 3);

      expect(projection).toHaveLength(3);
      expect(projection[0]).toBe(100);
      expect(projection[1]).toBeCloseTo(110, 2);
      expect(projection[2]).toBeCloseTo(121, 2);
    });

    test('generates projection with negative growth', () => {
      const projection = generateProjection(100, -0.1, 3);

      expect(projection).toHaveLength(3);
      expect(projection[0]).toBe(100);
      expect(projection[1]).toBe(90);
      expect(projection[2]).toBe(81);
    });

    test('generates projection with zero growth', () => {
      const projection = generateProjection(100, 0, 3);

      expect(projection).toEqual([100, 100, 100]);
    });

    test('generates single period projection', () => {
      const projection = generateProjection(100, 0.05, 1);

      expect(projection).toEqual([100]);
    });

    test('handles zero periods', () => {
      const projection = generateProjection(100, 0.05, 0);

      expect(projection).toEqual([]);
    });
  });

  describe('aggregateResults', () => {
    const aggregateResults = (
      results: Array<{ scenario: string; value: number }>
    ): Record<string, number> => {
      return results.reduce(
        (acc, { scenario, value }) => {
          acc[scenario] = (acc[scenario] || 0) + value;
          return acc;
        },
        {} as Record<string, number>
      );
    };

    test('aggregates results by scenario', () => {
      const results = [
        { scenario: 'A', value: 100 },
        { scenario: 'B', value: 200 },
        { scenario: 'A', value: 50 },
      ];

      const aggregated = aggregateResults(results);

      expect(aggregated.A).toBe(150);
      expect(aggregated.B).toBe(200);
    });

    test('handles empty results', () => {
      const aggregated = aggregateResults([]);

      expect(aggregated).toEqual({});
    });

    test('handles single result', () => {
      const results = [{ scenario: 'A', value: 100 }];

      const aggregated = aggregateResults(results);

      expect(aggregated.A).toBe(100);
    });

    test('handles negative values', () => {
      const results = [
        { scenario: 'A', value: 100 },
        { scenario: 'A', value: -50 },
      ];

      const aggregated = aggregateResults(results);

      expect(aggregated.A).toBe(50);
    });

    test('preserves zero values', () => {
      const results = [{ scenario: 'A', value: 0 }];

      const aggregated = aggregateResults(results);

      expect(aggregated.A).toBe(0);
    });
  });

  describe('formatTimeRemaining', () => {
    const formatTimeRemaining = (seconds: number): string => {
      if (seconds < 60) return `${Math.round(seconds)}s`;
      if (seconds < 3600) return `${Math.round(seconds / 60)}m`;
      return `${Math.round(seconds / 3600)}h`;
    };

    test('formats seconds', () => {
      expect(formatTimeRemaining(30)).toBe('30s');
      expect(formatTimeRemaining(45)).toBe('45s');
    });

    test('formats minutes', () => {
      expect(formatTimeRemaining(120)).toBe('2m');
      expect(formatTimeRemaining(300)).toBe('5m');
    });

    test('formats hours', () => {
      expect(formatTimeRemaining(3600)).toBe('1h');
      expect(formatTimeRemaining(7200)).toBe('2h');
    });

    test('rounds appropriately', () => {
      expect(formatTimeRemaining(29.4)).toBe('29s');
      expect(formatTimeRemaining(29.6)).toBe('30s');
      expect(formatTimeRemaining(90)).toBe('2m');
    });

    test('handles zero time', () => {
      expect(formatTimeRemaining(0)).toBe('0s');
    });

    test('handles large values', () => {
      expect(formatTimeRemaining(86400)).toBe('24h');
    });
  });

  describe('getStatusColor', () => {
    const getStatusColor = (status: string): string => {
      const colorMap: Record<string, string> = {
        completed: '#66bb6a',
        running: '#ffa726',
        failed: '#f44336',
        queued: '#64b5f6',
        idle: '#bdbdbd',
      };
      return colorMap[status] || '#bdbdbd';
    };

    test('returns correct color for completed status', () => {
      expect(getStatusColor('completed')).toBe('#66bb6a');
    });

    test('returns correct color for running status', () => {
      expect(getStatusColor('running')).toBe('#ffa726');
    });

    test('returns correct color for failed status', () => {
      expect(getStatusColor('failed')).toBe('#f44336');
    });

    test('returns correct color for queued status', () => {
      expect(getStatusColor('queued')).toBe('#64b5f6');
    });

    test('returns gray color for idle status', () => {
      expect(getStatusColor('idle')).toBe('#bdbdbd');
    });

    test('returns default color for unknown status', () => {
      expect(getStatusColor('unknown')).toBe('#bdbdbd');
    });

    test('returns default color for empty string', () => {
      expect(getStatusColor('')).toBe('#bdbdbd');
    });
  });

  describe('calculatePercentageChange', () => {
    const calculatePercentageChange = (initial: number, final: number): number => {
      if (initial === 0) return 0;
      return ((final - initial) / Math.abs(initial)) * 100;
    };

    test('calculates positive percentage change', () => {
      expect(calculatePercentageChange(100, 110)).toBe(10);
      expect(calculatePercentageChange(100, 150)).toBe(50);
    });

    test('calculates negative percentage change', () => {
      expect(calculatePercentageChange(100, 90)).toBe(-10);
      expect(calculatePercentageChange(100, 50)).toBe(-50);
    });

    test('calculates percentage change from negative initial', () => {
      expect(calculatePercentageChange(-100, -110)).toBeCloseTo(-10, 1);
      expect(calculatePercentageChange(-100, -50)).toBeCloseTo(50, 1);
    });

    test('handles zero initial value', () => {
      expect(calculatePercentageChange(0, 100)).toBe(0);
      expect(calculatePercentageChange(0, -100)).toBe(0);
    });

    test('handles no change', () => {
      expect(calculatePercentageChange(100, 100)).toBe(0);
    });

    test('handles large percentage changes', () => {
      expect(calculatePercentageChange(1, 100)).toBe(9900);
      expect(calculatePercentageChange(100, 1)).toBe(-99);
    });

    test('rounds to appropriate precision', () => {
      const result = calculatePercentageChange(3, 10);
      expect(result).toBeCloseTo(233.33, 1);
    });
  });
});
