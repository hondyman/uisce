// React Query hooks for API key management
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { APIKey, CreateAPIKeyRequest, APIKeyUsage } from "../types";

// ============================================================================
// QUERIES
// ============================================================================

export function useAPIKeys() {
  return useQuery({
    queryKey: ["apiKeys"],
    queryFn: () => 
      api<{ api_keys: APIKey[] }>("/admin/api-keys").then(res => res.api_keys ? { api_keys: res.api_keys } : res)
  });
}

export function useAPIKey(id: string | undefined) {
  return useQuery({
    queryKey: ["apiKey", id],
    queryFn: () => 
      api<{ api_key: APIKey }>(`/admin/api-keys/${id}`).then(res => res.api_key ? { api_key: res.api_key } : res),
    enabled: !!id
  });
}

export function useAPIKeyUsage(id: string | undefined, limit = 100) {
  return useQuery({
    queryKey: ["apiKeyUsage", id, limit],
    queryFn: () => 
      api<{ usage: APIKeyUsage[] }>(`/admin/api-keys/${id}/usage?limit=${limit}`).then(res => res.usage ? { usage: res.usage } : res),
    enabled: !!id
  });
}

// ============================================================================
// MUTATIONS
// ============================================================================

export function useCreateAPIKey() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (body: CreateAPIKeyRequest) =>
      api<{ api_key: APIKey & { plaintext?: string } }>("/admin/api-keys", {
        method: "POST",
        body: JSON.stringify(body)
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["apiKeys"] });
    }
  });
}

export function useRevokeAPIKey(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: () =>
      api(`/admin/api-keys/${id}/revoke`, {
        method: "POST"
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["apiKeys"] });
      qc.invalidateQueries({ queryKey: ["apiKey", id] });
    }
  });
}

export function useRotateAPIKey(id: string) {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: () =>
      api<{ api_key: APIKey & { plaintext?: string } }>(`/admin/api-keys/${id}/rotate`, {
        method: "POST"
      }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["apiKeys"] });
      qc.invalidateQueries({ queryKey: ["apiKey", id] });
    }
  });
}
