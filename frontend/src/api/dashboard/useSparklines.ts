import { useQuery } from '@tanstack/react-query';

export interface SparklineData {
  timestamp: string;
  value: number;
}

export interface Sparklines {
  pass_rate: SparklineData[];
  hard_breaches: SparklineData[];
  soft_breaches: SparklineData[];
  volatility: SparklineData[];
  etl_duration: SparklineData[];
}

export function useSparklines(tenantId: string) {
  return useQuery({
    queryKey: ['dashboard-sparklines', tenantId],
    queryFn: async () => {
      const res = await fetch(`/api/dashboard/sparklines?tenant_id=${tenantId}`);
      if (!res.ok) throw new Error('Failed to load sparklines');
      const data = await res.json();
      return data as Sparklines;
    },
    enabled: !!tenantId,
  });
}
