import { useQuery } from '@tanstack/react-query';

export interface RiskSummary {
  avg_volatility: number;
  avg_var_95: number;
  avg_var_99: number;
  total_scenarios: number;
  worst_scenario?: {
    name: string;
    pnl: number;
  };
  exposure_breakdown: {
    equity: number;
    rates: number;
    credit: number;
    fx: number;
  };
}

export function useRiskSummary(
  tenantId: string,
  valuationDate: string
) {
  return useQuery({
    queryKey: ['dashboard-risk', tenantId, valuationDate],
    queryFn: async () => {
      const res = await fetch(
        `/api/dashboard/risk?tenant_id=${tenantId}&valuation_date=${valuationDate}`
      );
      if (!res.ok) throw new Error('Failed to load risk summary');
      const data = await res.json();
      return data as RiskSummary;
    },
    enabled: !!tenantId && !!valuationDate,
  });
}
