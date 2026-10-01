import { describe, it, expect, vi } from 'vitest';
import {
  CHART_REGISTRY,
  SUPPORTED_CHART_TYPES,
  getChartSpec,
  buildOptionFromRegistry,
  CHART_ROW_SOFT_CAP,
} from '../../features/query-builder/charts';
import type { QueryResultsData } from '../../features/query-builder/charts/types';

describe('ECharts Registry & Adapters', () => {
  const sampleResults1Dim1Meas: QueryResultsData = {
    columns: [
      { name: 'region', termNodeId: 'dim-region', alias: 'Region', type: 'dimension' },
      { name: 'revenue', termNodeId: 'meas-rev', alias: 'Revenue', type: 'measure' },
    ],
    rows: [
      { region: 'North America', revenue: 150000 },
      { region: 'EMEA', revenue: 120000 },
      { region: 'APAC', revenue: 90000 },
    ],
  };

  const sampleResults2Dim1Meas: QueryResultsData = {
    columns: [
      { name: 'region', termNodeId: 'dim-region', alias: 'Region', type: 'dimension' },
      { name: 'tier', termNodeId: 'dim-tier', alias: 'Tier', type: 'dimension' },
      { name: 'revenue', termNodeId: 'meas-rev', alias: 'Revenue', type: 'measure' },
    ],
    rows: [
      { region: 'North America', tier: 'Enterprise', revenue: 100000 },
      { region: 'North America', tier: 'Mid-Market', revenue: 50000 },
      { region: 'EMEA', tier: 'Enterprise', revenue: 80000 },
      { region: 'EMEA', tier: 'Mid-Market', revenue: 40000 },
    ],
  };

  const sampleResults1Dim2Meas: QueryResultsData = {
    columns: [
      { name: 'quarter', termNodeId: 'dim-qtr', alias: 'Quarter', type: 'dimension' },
      { name: 'sales', termNodeId: 'meas-sales', alias: 'Sales', type: 'measure' },
      { name: 'target', termNodeId: 'meas-target', alias: 'Target', type: 'measure' },
    ],
    rows: [
      { quarter: 'Q1', sales: 100, target: 90 },
      { quarter: 'Q2', sales: 130, target: 110 },
      { quarter: 'Q3', sales: 125, target: 120 },
      { quarter: 'Q4', sales: 160, target: 140 },
    ],
  };

  const sampleResultsScatter: QueryResultsData = {
    columns: [
      { name: 'company', termNodeId: 'dim-comp', alias: 'Company', type: 'dimension' },
      { name: 'spend', termNodeId: 'meas-spend', alias: 'Spend', type: 'measure' },
      { name: 'roi', termNodeId: 'meas-roi', alias: 'ROI', type: 'measure' },
      { name: 'headcount', termNodeId: 'meas-hc', alias: 'Headcount', type: 'measure' },
    ],
    rows: [
      { company: 'Alpha Inc', spend: 50000, roi: 3.2, headcount: 120 },
      { company: 'Beta Corp', spend: 80000, roi: 4.5, headcount: 250 },
    ],
  };

  describe('Registry Integrity & Shelf Validation Matrix', () => {
    it('defines specifications for all 8 supported subset chart types', () => {
      expect(SUPPORTED_CHART_TYPES).toEqual([
        'bar',
        'stackedBar',
        'line',
        'area',
        'pie',
        'donut',
        'scatter',
        'combo',
      ]);

      for (const type of SUPPORTED_CHART_TYPES) {
        const spec = getChartSpec(type);
        expect(spec).toBeDefined();
        expect(spec?.id).toBe(type);
        expect(spec?.label).toBeTruthy();
        expect(spec?.interactions.crossFilter).toBe(true);
        expect(typeof spec?.buildOption).toBe('function');
      }
    });

    it('validates shelf requirements across all chart types (Shelf Compat Matrix)', () => {
      // 1. Bar / StackedBar / Line / Area: 1-2 dims, 1-3 measures
      const barSpec = getChartSpec('bar')!;
      expect(barSpec.validateShelves(0, 1).valid).toBe(false);
      expect(barSpec.validateShelves(1, 0).valid).toBe(false);
      expect(barSpec.validateShelves(1, 1).valid).toBe(true);
      expect(barSpec.validateShelves(2, 2).valid).toBe(true);
      expect(barSpec.validateShelves(3, 1).valid).toBe(false);
      expect(barSpec.validateShelves(1, 4).valid).toBe(false);

      // 2. Pie / Donut: exactly 1 dim, 1 measure
      const pieSpec = getChartSpec('pie')!;
      expect(pieSpec.validateShelves(1, 1).valid).toBe(true);
      expect(pieSpec.validateShelves(2, 1).valid).toBe(false);
      expect(pieSpec.validateShelves(1, 2).valid).toBe(false);

      const donutSpec = getChartSpec('donut')!;
      expect(donutSpec.validateShelves(1, 1).valid).toBe(true);
      expect(donutSpec.validateShelves(0, 1).valid).toBe(false);

      // 3. Scatter / Combo: exactly 1 dim, 2-3 measures
      const scatterSpec = getChartSpec('scatter')!;
      expect(scatterSpec.validateShelves(1, 1).valid).toBe(false);
      expect(scatterSpec.validateShelves(1, 2).valid).toBe(true);
      expect(scatterSpec.validateShelves(1, 3).valid).toBe(true);
      expect(scatterSpec.validateShelves(2, 2).valid).toBe(false);

      const comboSpec = getChartSpec('combo')!;
      expect(comboSpec.validateShelves(1, 2).valid).toBe(true);
      expect(comboSpec.validateShelves(1, 1).valid).toBe(false);
    });
  });

  describe('Adapter Option Building & Term Identity Preservation', () => {
    it('builds Bar chart options with term identity on data points', () => {
      const option = buildOptionFromRegistry('bar', sampleResults1Dim1Meas) as any;
      expect(option.xAxis.data).toEqual(['North America', 'EMEA', 'APAC']);
      expect(option.series).toHaveLength(1);
      expect(option.series[0].type).toBe('bar');
      expect(option.series[0].data[0]).toMatchObject({
        value: 150000,
        termNodeId: 'dim-region',
        memberValue: 'North America',
      });
    });

    it('builds Bar & StackedBar chart with 2-dimension series grouping', () => {
      const barOpt = buildOptionFromRegistry('bar', sampleResults2Dim1Meas) as any;
      expect(barOpt.xAxis.data).toEqual(['North America', 'EMEA']);
      expect(barOpt.legend.data).toEqual(['Enterprise', 'Mid-Market']);
      expect(barOpt.series).toHaveLength(2);
      expect(barOpt.series[0].name).toBe('Enterprise');
      expect(barOpt.series[0].data[0]).toMatchObject({
        value: 100000,
        termNodeId: 'dim-region',
        memberValue: 'North America',
        secTermNodeId: 'dim-tier',
        secMemberValue: 'Enterprise',
      });

      const stackedOpt = buildOptionFromRegistry('stackedBar', sampleResults2Dim1Meas) as any;
      expect(stackedOpt.series[0].stack).toBe('total');
    });

    it('builds Line and Area chart options with smooth curve and area styling', () => {
      const lineOpt = buildOptionFromRegistry('line', sampleResults1Dim2Meas) as any;
      expect(lineOpt.xAxis.data).toEqual(['Q1', 'Q2', 'Q3', 'Q4']);
      expect(lineOpt.series).toHaveLength(2);
      expect(lineOpt.series[0].type).toBe('line');
      expect(lineOpt.series[0].smooth).toBe(true);

      const areaOpt = buildOptionFromRegistry('area', sampleResults1Dim2Meas) as any;
      expect(areaOpt.series[0].areaStyle).toBeDefined();
      expect(areaOpt.series[0].stack).toBe('total');
    });

    it('builds Pie and Donut chart options with category slices', () => {
      const pieOpt = buildOptionFromRegistry('pie', sampleResults1Dim1Meas) as any;
      expect(pieOpt.series[0].type).toBe('pie');
      expect(pieOpt.series[0].radius).toBe('65%');
      expect(pieOpt.series[0].data).toHaveLength(3);
      expect(pieOpt.series[0].data[0]).toMatchObject({
        name: 'North America',
        value: 150000,
        termNodeId: 'dim-region',
      });

      const donutOpt = buildOptionFromRegistry('donut', sampleResults1Dim1Meas) as any;
      expect(donutOpt.series[0].radius).toEqual(['40%', '70%']);
    });

    it('builds Scatter Plot options with X/Y and point size mapping', () => {
      const scatterOpt = buildOptionFromRegistry('scatter', sampleResultsScatter) as any;
      expect(scatterOpt.series[0].type).toBe('scatter');
      expect(scatterOpt.series[0].data[0]).toMatchObject({
        name: 'Alpha Inc',
        value: [50000, 3.2, 120],
        termNodeId: 'dim-comp',
      });
    });

    it('builds Combo (mixed bar + line) options with dual Y axes', () => {
      const comboOpt = buildOptionFromRegistry('combo', sampleResults1Dim2Meas) as any;
      expect(comboOpt.yAxis).toHaveLength(2);
      expect(comboOpt.series).toHaveLength(2);
      expect(comboOpt.series[0].type).toBe('bar');
      expect(comboOpt.series[0].yAxisIndex).toBe(0);
      expect(comboOpt.series[1].type).toBe('line');
      expect(comboOpt.series[1].yAxisIndex).toBe(1);
    });

    it('handles empty results gracefully without throwing', () => {
      const emptyResults: QueryResultsData = { rows: [], columns: [] };
      for (const type of SUPPORTED_CHART_TYPES) {
        const opt = buildOptionFromRegistry(type, emptyResults);
        expect(opt).toBeDefined();
      }
    });

    it('falls back to default bar chart when unknown chartType is requested', () => {
      const opt = buildOptionFromRegistry('future_chart_type_3d', sampleResults1Dim1Meas) as any;
      expect(opt).toBeDefined();
      expect(opt.series[0].type).toBe('bar');
    });
  });

  describe('Result Soft-Cap Threshold', () => {
    it('defines 10,000 as the browser performance soft cap', () => {
      expect(CHART_ROW_SOFT_CAP).toBe(10000);
    });
  });
});
