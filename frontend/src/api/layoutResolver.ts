import apiClient from '@/utils/apiClient';
import type {
  MutationRequest,
  MutationResponse,
  PageResolution,
} from '@/types/mutability';

/**
 * Resolve a pageKey into a hydration-aware payload.
 *
 * Cardinal Rule 1.3: tenantId is sent via the standard X-Tenant-ID header
 * that apiClient already injects — no UUID duplication in the body.
 */
export async function resolveLayoutApi(
  pageKey: string,
  options?: { bindingId?: string; tenantId?: string },
): Promise<PageResolution> {
  const params = new URLSearchParams({ pageKey });
  if (options?.bindingId) params.set('bindingId', options.bindingId);
  if (options?.tenantId) params.set('tenantId', options.tenantId);

  const res = await apiClient<Response>(`/api/v1/layout/resolve?${params.toString()}`);
  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new Error(`layout resolve failed (${res.status}): ${body}`);
  }
  return res.json();
}

/**
 * Dispatch a mutation to either the direct-PG path or the CQRS topic.
 *
 * The backend routes based on `bindingId.is_directly_writeable`. The
 * caller passes whatever the page resolution returned for `bindingId`.
 */
export async function dispatchMutationApi(
  req: MutationRequest,
): Promise<MutationResponse> {
  const res = await apiClient<Response>('/api/v1/mutations/dispatch', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(req),
  });
  if (!res.ok && res.status !== 409 && res.status !== 202) {
    const body = await res.text().catch(() => '');
    throw new Error(`mutation dispatch failed (${res.status}): ${body}`);
  }
  return res.json();
}