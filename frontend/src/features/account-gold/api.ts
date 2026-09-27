import apiClient from '../../utils/apiClient';

export interface AccountFieldOverride {
  id: string;
  tenant_id: string;
  account_cd: string;
  semantic_term_id?: string | null;
  field_cd: string;
  override_value: string;
  reason: string;
  approval_status: 'pending' | 'approved' | 'rejected' | 'withdrawn';
  expires_at?: string | null;
}

export interface AccountGoldRecord {
  id: string;
  tenant_id: string;
  account_cd: string;
  account_name: string;
  account_type_cd: string;
  status_cd: string;
  base_currency?: string;
  domicile?: string;
  custom_attributes?: Record<string, unknown>;
  source_systems?: Record<string, string>;
  confidence_score: number;
  gold_version: number;
  valid_from: string;
}

const BASE = '/api/v1/mdm/account-gold';

export const accountGoldApi = {
  listOverrides: (accountCd?: string) =>
    apiClient<{ overrides: AccountFieldOverride[] }>(
      `${BASE}/overrides${accountCd ? `?account_cd=${encodeURIComponent(accountCd)}` : ''}`,
    ),
  createOverride: (body: {
    account_cd: string;
    field_cd: string;
    override_value: string;
    reason: string;
    semantic_term_id?: string;
  }) =>
    apiClient<{ id: string; approval_status: string }>(`${BASE}/overrides`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
  approve: (id: string) =>
    apiClient<{ id: string; approval_status: string }>(`${BASE}/overrides/${id}/approve`, {
      method: 'POST',
    }),
  reject: (id: string) =>
    apiClient<{ id: string; approval_status: string }>(`${BASE}/overrides/${id}/reject`, {
      method: 'POST',
    }),
  build: (body: { account_cd: string; publish?: boolean; change_reason?: string }) =>
    apiClient<AccountGoldRecord>(`${BASE}/build`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    }),
};
