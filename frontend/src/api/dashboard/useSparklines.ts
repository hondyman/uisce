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

export function useSparklines() {
  return useQuery({
    queryKey: ['dashboard-sparklines'],
    queryFn: async () => {
      const res = await fetch('/api/dashboard/sparklines');
      if (!res.ok) throw new Error('Failed to load sparklines');
      const data = await res.json();
      return data as Sparklines;
    },
  });
}
