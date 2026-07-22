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
  events: (tenantId: string) => [...riskKeys.all, 'events', tenantId] as const,
  portfolios: (tenantId: string) => [...riskKeys.all, 'portfolios', tenantId] as const,
  rebalanceSuggestions: (tenantId: string) => [...riskKeys.all, 'rebalance', tenantId] as const,
  scenarios: (tenantId: string) => [...riskKeys.all, 'scenarios', tenantId] as const,
};

export function useRiskEvents(tenantId: string, refetchInterval = 2000) {
  return useQuery({
    queryKey: riskKeys.events(tenantId),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/risk-events?tenant_id=${encodeURIComponent(tenantId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<RiskEvent[]>;
    },
    enabled: !!tenantId,
    refetchInterval,
  });
}

export function usePortfolios(tenantId: string, refetchInterval = 2000) {
  return useQuery({
    queryKey: riskKeys.portfolios(tenantId),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/portfolios?tenant_id=${encodeURIComponent(tenantId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<Portfolio[]>;
    },
    enabled: !!tenantId,
    refetchInterval,
  });
}

export function useRebalanceSuggestions(tenantId: string, refetchInterval = 2000) {
  return useQuery({
    queryKey: riskKeys.rebalanceSuggestions(tenantId),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/rebalance-suggestions?tenant_id=${encodeURIComponent(tenantId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<RebalanceSuggestion[]>;
    },
    enabled: !!tenantId,
    refetchInterval,
  });
}

export function useScenarios(tenantId: string, refetchInterval = 2000) {
  return useQuery({
    queryKey: riskKeys.scenarios(tenantId),
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/scenarios?tenant_id=${encodeURIComponent(tenantId)}`);
      if (!res.ok) {
        throw new Error(await res.text());
      }
      return res.json() as Promise<Scenario[]>;
    },
    enabled: !!tenantId,
    refetchInterval,
  });
}
