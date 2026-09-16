import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../../lib/apiClient';

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

export function useSparklines() {
  return useQuery({
    queryKey: ['dashboard-sparklines'],
    queryFn: async () => {
      const res = await apiFetch('/api/dashboard/sparklines');
      const data = await res.json();
      return data as Sparklines;
    },
  });
}
