import apiClient from '../../utils/apiClient';

export interface SourceSystem {
  code: string;
  display_name: string;
  default_rank: number;
  description: string;
  is_active: boolean;
}

export interface SurvivorshipRule {
  id: string;
  tenant_id: string;
  entity_type: string;
  semantic_term_id: string;
  strategy: string;
  priority_order: string[];
  max_stale_seconds: number;
  is_active: boolean;
  semantic_term_name?: string;
  semantic_term_path?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateRuleInput {
  entity_type: string;
  semantic_term_id: string;
  strategy?: string;
  priority_order?: string[];
  max_stale_seconds?: number;
}

export interface UpdateRuleInput {
  strategy?: string;
  priority_order?: string[];
  max_stale_seconds?: number;
  is_active?: boolean;
}

const BASE = '/api/v1/mdm/survivorship-rules';

export const survivorshipApi = {
  listSources: () => apiClient<{ sources: SourceSystem[] }>(`${BASE}/sources`),
  list: (entityType: string, includeInactive = false) =>
    apiClient<{ rules: SurvivorshipRule[] }>(
      `${BASE}/?entity_type=${encodeURIComponent(entityType)}${includeInactive ? '&include_inactive=true' : ''}`,
    ),
  create: (body: CreateRuleInput) =>
    apiClient<SurvivorshipRule>(BASE + '/', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  update: (id: string, body: UpdateRuleInput) =>
    apiClient<SurvivorshipRule>(`${BASE}/${id}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  deactivate: (id: string) =>
    apiClient<void>(`${BASE}/${id}`, { method: 'DELETE' }),
};
