// Admin UI Types - Complete TypeScript definitions for the Admin Dashboard

export interface Tenant {
  id: string;
  name: string;
  code?: string | null;
  region?: string | null;
  plan: "free" | "pro" | "enterprise";
  max_requests?: number | null;
  window_seconds?: number | null;
  is_suspended: boolean;
  created_at: string;
  updated_at: string;
}

export interface TenantCreateRequest {
  name: string;
  code?: string;
  region?: string;
  plan: "free" | "pro" | "enterprise";
  max_requests?: number;
  window_seconds?: number;
}

export interface TenantUpdateRequest {
  name?: string;
  region?: string;
  plan?: "free" | "pro" | "enterprise";
  max_requests?: number;
  window_seconds?: number;
  is_suspended?: boolean;
}

export interface APIKey {
  id: string;
  name: string;
  user_id: string;
  roles: string[];
  tenant_ids: string[];
  created_at: string;
  last_used_at?: string;
  is_revoked: boolean;
}

export interface APIKeyCreateRequest {
  user_id: string;
  tenant_ids: string[];
  roles: string[];
  name: string;
  description?: string;
}

export interface APIKeyUsage {
  id: string;
  api_key_id: string;
  user_id?: string;
  tenant_id?: string;
  path: string;
  method: string;
  region?: string;
  ip_address?: string;
  user_agent?: string;
  created_at: string;
}

export interface DailyUsageStats {
  day: string;
  count: number;
}

export interface EndpointUsageStats {
  path: string;
  count: number;
}

export interface TenantUsageSummary {
  last_24h: number;
  last_7d: number;
  last_30d: number;
}

export interface ListTenantsResponse {
  tenants: Tenant[];
  total: number;
  limit: number;
  offset: number;
}

export interface ListAPIKeysResponse {
  api_keys: APIKey[];
  total: number;
  limit: number;
  offset: number;
}

export interface APIKeyUsageResponse {
  usage: APIKeyUsage[];
}

export interface TenantDailyUsageResponse {
  tenant_id: string;
  days: number;
  data: DailyUsageStats[];
}

export interface TenantEndpointUsageResponse {
  tenant_id: string;
  top_endpoints: EndpointUsageStats[];
}

// Plan definitions
export const PLANS = {
  free: {
    name: "Free",
    features: ["Basic API access", "1 API key", "100 requests/day"],
    max_requests: 100,
    window_seconds: 86400,
  },
  pro: {
    name: "Pro",
    features: ["Advanced API access", "10 API keys", "10,000 requests/day"],
    max_requests: 10000,
    window_seconds: 86400,
  },
  enterprise: {
    name: "Enterprise",
    features: ["Unlimited API access", "Unlimited API keys", "Custom limits"],
    max_requests: null,
    window_seconds: null,
  },
};

// Role definitions
export const ROLES = {
  GLOBAL_OPS: "Global Operations",
  TENANT_ADMIN: "Tenant Admin",
  USER: "User",
};

// Regions
export const REGIONS = [
  "us-east-1",
  "us-west-2",
  "eu-west-1",
  "eu-central-1",
  "ap-southeast-1",
  "ap-northeast-1",
];
