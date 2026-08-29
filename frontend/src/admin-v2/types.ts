// Self-Service Studio Types — Phase D frontend
// https://uisce.io/docs/security/MAPPING.md#part-2-–-tenants-self-service-entitlements

export type ProfileKey = string;
export type EntitlementType = "MENU_PAGE" | "WORKFLOW_STEP" | "PUBLIC_API";
export type OverrideState = "INHERIT_BASELINE" | "EXPLICIT_ALLOW" | "FORCE_DENY";
export type PolicyEffect = "allow" | "deny";

// ----------------------------------------------------------------------------
// Profiles (Screen 1 + Screen 2)
// ----------------------------------------------------------------------------

export interface SystemProfile {
  profileKey: ProfileKey;
  profileName: string;
  origin: "system"; // immutable blueprint
  ruleCount: number;
}

export interface TenantProfile {
  profileId: string;
  profileKey: ProfileKey;
  profileName: string;
  origin: "tenant"; // tenant-scoped clone or custom
  ruleCount: number;
  parentProfileKey?: ProfileKey | null;
  updatedAt: string;
}

export interface CloneProfileRequest {
  sourceProfileKey: ProfileKey;
  targetProfileKey: ProfileKey;
  targetProfileName: string;
}

export interface CloneProfileResponse {
  profileId: string;
  profileKey: ProfileKey;
  profileName: string;
  clonedRulesCount: number;
  sourceProfileKey: ProfileKey;
}

// ----------------------------------------------------------------------------
// ABAC Policies (Screen 2)
// ----------------------------------------------------------------------------

export interface AbacPolicy {
  policyId: string;
  origin: "system" | "tenant";
  targetProfileKey: ProfileKey;
  actionAttribute: string;
  effect: PolicyEffect;
  priorityRank: number;
  conditionDsl?: string;
  name: string;
  description?: string;
  updatedAt: string;
}

export interface BackendSecurityProfile {
  profile_id: string;
  tenant_id: string | null;
  profile_key: string;
  profile_name: string;
  parent_profile_id: string | null;
  created_at: string;
  updated_at: string;
}

export interface BackendAbacPolicy {
  policyId: string;
  tenantId: string | null;
  targetProfileKey: string;
  name: string;
  description: string;
  effect: PolicyEffect;
  priority: number;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface BackendComponentEntitlement {
  entitlementId: string;
  tenantId: string;
  targetProfileKey: string;
  entitlementType: EntitlementType;
  nodePath: string;
  overrideState: OverrideState;
  conditionalDsl?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AppendPolicyOverrideRequest {
  targetProfileKey: ProfileKey;
  actionAttribute: string;
  effect: PolicyEffect;
  priorityRank: number;
  conditionDsl?: string;
  name: string;
  description?: string;
}

export interface AppendPolicyOverrideResponse {
  policyId: string;
  tenantId: string;
  targetProfileKey: ProfileKey;
  actionAttribute: string;
  effect: PolicyEffect;
  priorityRank: number;
}

// ----------------------------------------------------------------------------
// Component Entitlements (Screen 3)
// ----------------------------------------------------------------------------

export interface ComponentEntitlement {
  entitlementId: string;
  targetProfileKey: ProfileKey;
  entitlementType: EntitlementType;
  nodePath: string;
  overrideState: OverrideState;
  conditionalDsl?: string;
  updatedAt: string;
}

// Result of walking security.security_profiles.parent_profile_id to resolve
// what a profile actually ends up with (own overrides + inherited baseline).
export interface EffectiveEntitlement {
  node_path: string;
  entitlement_type: EntitlementType;
  override_state: OverrideState;
  condition_dsl?: string;
  resolved_tenant?: string; // absent/empty means resolved from the gold-copy baseline
  inherited: boolean;
}

export interface UpsertEntitlementRequest {
  targetProfileKey: ProfileKey;
  entitlementType: EntitlementType;
  nodePath: string;
  overrideState: OverrideState;
  conditionalDsl?: string;
}

export interface UpsertEntitlementResponse {
  entitlementId: string;
  tenantId: string;
  targetProfileKey: ProfileKey;
  entitlementType: EntitlementType;
  nodePath: string;
  overrideState: OverrideState;
  updatedAt: string;
}

// ----------------------------------------------------------------------------
// IdP Brokers (consumed by Screen 1, written by Phase C backend)
// ----------------------------------------------------------------------------

export interface IdpBroker {
  alias: string;
  providerId: string;
  displayName?: string;
  enabled: boolean;
  internalId?: string;
  linkedRealm: string;
  provisionedAt?: string;
}
