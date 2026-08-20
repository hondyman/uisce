import { useQuery } from '@tanstack/react-query';
import { apiFetch } from '../lib/apiClient';

export interface RiskEvent {
  id: string;
  tenant_id: string;
  portfolio_id: string;
  event_type: string;
  severity: string;
  description: string;
  detected_at: string;
  status: string;
  metadata?: any;
}

export interface Portfolio {
  id: string;
  portfolio_name: string;
  aum: number;
  risk_score: number;
  status: string;
  sharpe?: number;
  var_95?: number;
  cvar_95?: number;
  liquidity_ratio?: number;
}

export interface RebalanceSuggestion {
  id: string;
  portfolio_id: string;
  current_weights: any;
  target_weights: any;
  rationale: string;
  confidence: number;
  created_at: string;
}

export interface Scenario {
  id: string;
  name: string;
  description: string;
  stress_type: string;
  parameters: any;
  results?: any;
  created_at: string;
}

export const riskKeys = {
  all: ['risk'] as const,
  events: () => [...riskKeys.all, 'events'] as const,
  portfolios: () => [...riskKeys.all, 'portfolios'] as const,
  rebalanceSuggestions: () => [...riskKeys.all, 'rebalance'] as const,
  scenarios: () => [...riskKeys.all, 'scenarios'] as const,
};

export function useRiskEvents(refetchInterval = 2000) {
  return useQuery({
    queryKey: riskKeys.events(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/risk-events');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<RiskEvent[]>;
    },
    refetchInterval,
  });
}

export function usePortfolios(refetchInterval = 2000) {
  return useQuery({
    queryKey: riskKeys.portfolios(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/portfolios');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<Portfolio[]>;
    },
    refetchInterval,
  });
}

export function useRebalanceSuggestions(refetchInterval = 2000) {
  return useQuery({
    queryKey: riskKeys.rebalanceSuggestions(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/rebalance-suggestions');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<RebalanceSuggestion[]>;
    },
    refetchInterval,
  });
}

export function useScenarios(refetchInterval = 2000) {
  return useQuery({
    queryKey: riskKeys.scenarios(),
    queryFn: async () => {
      const res = await apiFetch('/api/rest/scenarios');
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<Scenario[]>;
    },
    refetchInterval,
  });
}
