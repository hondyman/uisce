import { useQuery } from '@tanstack/react-query';

export interface RuleBreachAlert {
  rule_code: string;
  metric_value: number;
  threshold_value: number;
  portfolio_id: string;
  severity: string;
}

export interface ScenarioLossAlert {
  scenario_id: string;
  name: string;
  pnl: number;
  portfolio_id: string;
}

export interface ETLFailureAlert {
  etl_run_id: string;
  error_message: string;
  error_time: string;
}

export interface AlertsSummary {
  hard_breaches: RuleBreachAlert[];
  soft_breaches: RuleBreachAlert[];
  scenario_losses: ScenarioLossAlert[];
  etl_failures: ETLFailureAlert[];
  total_alerts: number;
}

export function useAlerts(tenantId: string, valuationDate: string) {
  return useQuery({
    queryKey: ['dashboard-alerts', tenantId, valuationDate],
    queryFn: async () => {
      const res = await fetch(
        `/api/dashboard/alerts?tenant_id=${tenantId}&valuation_date=${valuationDate}`
      );
      if (!res.ok) throw new Error('Failed to load alerts');
      const data = await res.json();
      return data as AlertsSummary;
    },
    enabled: !!tenantId && !!valuationDate,
  });
}
