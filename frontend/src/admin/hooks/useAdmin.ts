// Admin API Hooks - React hooks for interacting with admin endpoints

import { useState, useCallback, useEffect } from "react";
import {
  Tenant,
  TenantCreateRequest,
  TenantUpdateRequest,
  APIKey,
  APIKeyUsage,
  DailyUsageStats,
  EndpointUsageStats,
  ListTenantsResponse,
  ListAPIKeysResponse,
} from "../types";

const API_BASE = process.env.REACT_APP_API_URL || "http://localhost:8082/api";

// ============================================================================
// Tenant Hooks
// ============================================================================

export function useTenants(limit = 50, offset = 0) {
  const [tenants, setTenants] = useState<Tenant[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTenants = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${API_BASE}/admin/tenants?limit=${limit}&offset=${offset}`
      );
      if (!response.ok) throw new Error("Failed to fetch tenants");
      const data: ListTenantsResponse = await response.json();
      setTenants(data.tenants || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [limit, offset]);

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  return { tenants, total, loading, error, refetch: fetchTenants };
}

export function useTenant(tenantId: string) {
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchTenant = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE}/admin/tenants/${tenantId}`);
      if (!response.ok) throw new Error("Failed to fetch tenant");
      const data = await response.json();
      setTenant(data.tenant);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  useEffect(() => {
    if (tenantId) {
      fetchTenant();
    }
  }, [tenantId, fetchTenant]);

  return { tenant, loading, error, refetch: fetchTenant };
}

export function useCreateTenant() {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const create = useCallback(async (req: TenantCreateRequest) => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(`${API_BASE}/admin/tenants`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(req),
      });
      if (!response.ok) throw new Error("Failed to create tenant");
      const data = await response.json();
      return data.tenant as Tenant;
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, []);

  return { create, loading, error };
}

export function useUpdateTenant(tenantId: string) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const update = useCallback(
    async (req: TenantUpdateRequest) => {
      setLoading(true);
      setError(null);
      try {
        const response = await fetch(`${API_BASE}/admin/tenants/${tenantId}`, {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(req),
        });
        if (!response.ok) throw new Error("Failed to update tenant");
        const data = await response.json();
        return data.tenant as Tenant;
      } catch (err) {
        const message = err instanceof Error ? err.message : "Unknown error";
        setError(message);
        throw err;
      } finally {
        setLoading(false);
      }
    },
    [tenantId]
  );

  return { update, loading, error };
}

export function useSuspendTenant(tenantId: string) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const suspend = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${API_BASE}/admin/tenants/${tenantId}/suspend`,
        { method: "POST" }
      );
      if (!response.ok) throw new Error("Failed to suspend tenant");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  const unsuspend = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${API_BASE}/admin/tenants/${tenantId}/unsuspend`,
        { method: "POST" }
      );
      if (!response.ok) throw new Error("Failed to unsuspend tenant");
    } catch (err) {
      const message = err instanceof Error ? err.message : "Unknown error";
      setError(message);
      throw err;
    } finally {
      setLoading(false);
    }
  }, [tenantId]);

  return { suspend, unsuspend, loading, error };
}

// ============================================================================
// API Key Hooks
// ============================================================================

export function useAPIKeys(limit = 50, offset = 0) {
  const [keys, setKeys] = useState<APIKey[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchKeys = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${API_BASE}/admin/api-keys?limit=${limit}&offset=${offset}`
      );
      if (!response.ok) throw new Error("Failed to fetch API keys");
      const data: ListAPIKeysResponse = await response.json();
      setKeys(data.api_keys || []);
      setTotal(data.total || 0);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [limit, offset]);

  useEffect(() => {
    fetchKeys();
  }, [fetchKeys]);

  return { keys, total, loading, error, refetch: fetchKeys };
}

export function useAPIKeyUsage(apiKeyId: string, limit = 100) {
  const [usage, setUsage] = useState<APIKeyUsage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchUsage = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${API_BASE}/admin/api-keys/${apiKeyId}/usage?limit=${limit}`
      );
      if (!response.ok) throw new Error("Failed to fetch API key usage");
      const data = await response.json();
      setUsage(data.usage || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [apiKeyId, limit]);

  useEffect(() => {
    if (apiKeyId) {
      fetchUsage();
    }
  }, [apiKeyId, fetchUsage]);

  return { usage, loading, error, refetch: fetchUsage };
}

// ============================================================================
// Tenant Usage Hooks
// ============================================================================

export function useTenantDailyUsage(tenantId: string, days = 30) {
  const [stats, setStats] = useState<DailyUsageStats[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${API_BASE}/admin/tenants/${tenantId}/usage/daily?days=${days}`
      );
      if (!response.ok) throw new Error("Failed to fetch daily usage");
      const data = await response.json();
      setStats(data.data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [tenantId, days]);

  useEffect(() => {
    if (tenantId) {
      fetchStats();
    }
  }, [tenantId, fetchStats]);

  return { stats, loading, error, refetch: fetchStats };
}

export function useTenantEndpointUsage(tenantId: string, limit = 20) {
  const [stats, setStats] = useState<EndpointUsageStats[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchStats = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await fetch(
        `${API_BASE}/admin/tenants/${tenantId}/usage/endpoints?limit=${limit}`
      );
      if (!response.ok) throw new Error("Failed to fetch endpoint usage");
      const data = await response.json();
      setStats(data.top_endpoints || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unknown error");
    } finally {
      setLoading(false);
    }
  }, [tenantId, limit]);

  useEffect(() => {
    if (tenantId) {
      fetchStats();
    }
  }, [tenantId, fetchStats]);

  return { stats, loading, error, refetch: fetchStats };
}
