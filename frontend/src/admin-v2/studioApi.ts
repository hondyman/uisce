// Self-Service Studio API client — Phase D
// All endpoints live under /api/v1/tenant/. Cardinal Rule 7.4: the
// tenant_id is sourced from the cryptographic session — never sent in
// the request body. The backend enforces this via tenantIDFromSession.

import type {
  AppendPolicyOverrideRequest,
  AppendPolicyOverrideResponse,
  CloneProfileRequest,
  CloneProfileResponse,
  IdpBroker,
  UpsertEntitlementRequest,
  UpsertEntitlementResponse,
} from "./types";

const API_BASE = import.meta.env.VITE_API_URL || "http://localhost:8080/api";

class StudioApiError extends Error {
  status: number;
  detail?: string;
  constructor(status: number, msg: string, detail?: string) {
    super(msg);
    this.status = status;
    this.detail = detail;
  }
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    credentials: "include",
  });
  if (!res.ok) {
    const text = await res.text();
    let detail: string | undefined;
    try {
      const j = JSON.parse(text);
      detail = j.detail || j.error;
    } catch {
      detail = text;
    }
    throw new StudioApiError(res.status, `POST ${path} failed`, detail);
  }
  return res.json();
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "GET",
    credentials: "include",
  });
  if (!res.ok) {
    throw new StudioApiError(res.status, `GET ${path} failed`, await res.text());
  }
  return res.json();
}

async function del(path: string): Promise<void> {
  const res = await fetch(`${API_BASE}${path}`, {
    method: "DELETE",
    credentials: "include",
  });
  if (!res.ok && res.status !== 404) {
    throw new StudioApiError(res.status, `DELETE ${path} failed`, await res.text());
  }
}

export const studioApi = {
  // -------------------------------------------------------------------------
  // Screen 1: Profile cloning
  // -------------------------------------------------------------------------
  cloneProfile: (req: CloneProfileRequest) =>
    post<CloneProfileResponse>("/v1/tenant/profiles/clone", req),

  listProfiles: () =>
    get<{ profiles: import("./types").BackendSecurityProfile[]; count: number }>("/v1/tenant/profiles"),

  // -------------------------------------------------------------------------
  // Screen 2: ABAC policy overrides
  // -------------------------------------------------------------------------
  appendPolicyOverride: (req: AppendPolicyOverrideRequest) =>
    post<AppendPolicyOverrideResponse>("/v1/tenant/policies/override", req),

  listPolicies: (targetProfileKey: string) =>
    get<{ policies: import("./types").BackendAbacPolicy[]; count: number }>(
      `/v1/tenant/policies?target_profile_key=${encodeURIComponent(targetProfileKey)}`
    ),

  // -------------------------------------------------------------------------
  // Screen 3: Component entitlements
  // -------------------------------------------------------------------------
  upsertEntitlement: (req: UpsertEntitlementRequest) =>
    post<UpsertEntitlementResponse>("/v1/tenant/entitlements/map", req),

  listEntitlements: (targetProfileKey: string, entitlementType?: string) => {
    const qs = new URLSearchParams({ target_profile_key: targetProfileKey });
    if (entitlementType) qs.set("entitlement_type", entitlementType);
    return get<{ entitlements: import("./types").BackendComponentEntitlement[]; count: number }>(
      `/v1/tenant/entitlements?${qs.toString()}`
    );
  },

  deleteEntitlement: (entitlementId: string) =>
    del(`/v1/tenant/entitlements/${entitlementId}`),

  // -------------------------------------------------------------------------
  // Phase C — IdP broker lifecycle (consumed by Screen 1)
  // -------------------------------------------------------------------------
  listIdpBrokers: (targetRealm: string) =>
    get<{ brokers: IdpBroker[]; target_realm: string; count: number }>(
      `/v1/admin/idp-brokers?target_realm=${encodeURIComponent(targetRealm)}`
    ),
  registerIdpBroker: (targetRealm: string, alias: string, providerId: string) =>
    post<IdpBroker>("/v1/admin/idp-brokers", {
      target_realm: targetRealm,
      spec: { alias, providerId },
    }),
  deregisterIdpBroker: (targetRealm: string, alias: string) =>
    del(
      `/v1/admin/idp-brokers/${encodeURIComponent(alias)}?target_realm=${encodeURIComponent(targetRealm)}`
    ),
};

export { StudioApiError };
