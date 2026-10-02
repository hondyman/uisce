import { describe, it, expect } from 'vitest';
import {
  resolveDrillPath,
  drillDown,
  computeDrillState,
  buildDrillThroughContext,
  type DrillStep,
} from '../../features/query-builder/utils/drilldown';
import type { SavedQueryState, QueryDimension } from '../../features/query-builder/types/queryDef';
import type { SemanticTermView } from '../../studio-core/binding/businessObjectApi';

describe('Drill-Down & Drill-Through Utilities', () => {
  const regionDim: QueryDimension = { termNodeId: 'term-region', alias: 'region' };
  const countryDim: QueryDimension = { termNodeId: 'term-country', alias: 'country' };

  it('priority 1: resolves drill path from term metadata (drillPath / drill_down_dimensions)', () => {
    const catalog: SemanticTermView[] = [
      {
        termNodeId: 't-sector',
        termKey: 'sector',
        termName: 'Sector',
        role: 'DIMENSION',
        metadata: {
          drill_down_dimensions: 'sector, sub_industry, issuer_ticker',
        },
      } as any,
    ];

    const res = resolveDrillPath('t-sector', 'sector', catalog);
    expect(res.status).toBe('ok');
    if (res.status === 'ok') {
      expect(res.path).toEqual(['sector', 'sub_industry', 'issuer_ticker']);
      expect(res.currentLevel).toBe(0);

      const step = drillDown(res, 'Technology', []);
      expect(step.childDimension).toBe('sub_industry');
      expect(step.next).toEqual([{ dimension: 'sector', value: 'Technology' }]);
    }
  });

  it('priority 2: resolves drill path from standard fallbacks when metadata is absent', () => {
    const res = resolveDrillPath('term-region', 'region', undefined);
    expect(res.status).toBe('ok');
    if (res.status === 'ok') {
      expect(res.path).toEqual(['region', 'country', 'state_province', 'city']);
      expect(res.currentLevel).toBe(0);
    }
  });

  it('priority 3: disables drill when no metadata and no fallback exists (never invents synthetic levels)', () => {
    const res = resolveDrillPath('t-random', 'arbitrary_metric_or_dim', undefined);
    expect(res.status).toBe('disabled');
    if (res.status === 'disabled') {
      expect(res.reason).toBe('no-hierarchy-configured');
    }
  });

  it('computes drill state by swapping dimensions and pushing filters', () => {
    const baseState: SavedQueryState = {
      dimensions: [regionDim],
      measures: [{ termNodeId: 'm-rev', alias: 'revenue', agg: 'SUM' }],
      filters: [{ termNodeId: 'status', operator: 'eq', value: 'active' }],
      parameters: [],
    };

    const drillStep: DrillStep = {
      dimension: countryDim,
      filter: { termNodeId: 'region', operator: 'eq', value: 'EMEA' },
      label: 'region',
      value: 'EMEA',
    };

    const drilledState = computeDrillState(baseState, [drillStep]);

    // Primary dimension is swapped to country
    expect(drilledState.dimensions[0].alias).toBe('country');
    // Filter stack contains both base filter and EMEA drill filter
    expect(drilledState.filters).toHaveLength(2);
    expect(drilledState.filters[1]).toEqual({
      termNodeId: 'region',
      operator: 'eq',
      value: 'EMEA',
    });
  });

  it('builds drill-through context preserving drill stack and recordId', () => {
    const drillStep: DrillStep = {
      dimension: countryDim,
      filter: { termNodeId: 'region', operator: 'eq', value: 'APAC' },
      label: 'region',
      value: 'APAC',
    };

    const context = buildDrillThroughContext([drillStep], 'acc-999', { userRole: 'analyst' });

    expect(context.recordId).toBe('acc-999');
    expect(context.userRole).toBe('analyst');
    expect(context.country).toBe('APAC');
    expect(context.region).toBe('APAC');
  });

  it('multi-level drill-down (region -> country) composes full context stack for drill-through action', () => {
    const cityDim: QueryDimension = { termNodeId: 'term-city', alias: 'city' };

    const step1: DrillStep = {
      dimension: countryDim,
      filter: { termNodeId: 'term-region', operator: 'eq', value: 'EMEA' },
      label: 'Region: EMEA',
      value: 'EMEA',
    };

    const step2: DrillStep = {
      dimension: cityDim,
      filter: { termNodeId: 'term-country', operator: 'eq', value: 'Germany' },
      label: 'Country: Germany',
      value: 'Germany',
    };

    const drillSteps = [step1, step2];

    const context = buildDrillThroughContext(drillSteps, 'acc-portfolio-101', {
      sourceDashboard: 'ExecutiveOverview',
    });

    // 1. Selection context preserved
    expect(context.recordId).toBe('acc-portfolio-101');
    expect(context.sourceDashboard).toBe('ExecutiveOverview');

    // 2. Both drill filter values preserved under their aliases and term IDs
    expect(context.country).toBe('EMEA');
    expect(context['term-region']).toBe('EMEA');
    expect(context.city).toBe('Germany');
    expect(context['term-country']).toBe('Germany');
  });
});

