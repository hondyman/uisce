import { useQuery } from '@tanstack/react-query';

export interface ComplianceSummary {
  total_rules: number;
  pass_rate: number;
  hard_breaches: number;
  soft_breaches: number;
  info_alerts: number;
  by_severity: {
    hard: number;
    soft: number;
    info: number;
  };
}

export function useComplianceSummary(
  tenantId: string,
  valuationDate: string
) {
  return useQuery({
    queryKey: ['dashboard-compliance', tenantId, valuationDate],
    queryFn: async () => {
      const res = await fetch(
        `/api/dashboard/compliance?tenant_id=${tenantId}&valuation_date=${valuationDate}`
      );
      if (!res.ok) throw new Error('Failed to load compliance summary');
      const data = await res.json();
      return data as ComplianceSummary;
    },
    enabled: !!tenantId && !!valuationDate,
  });
}
