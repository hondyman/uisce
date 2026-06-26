// frontend/src/types/access.ts
// Multi-tier tenant access model for Workday-like platform

/**
 * Access levels for the platform
 * 
 * PLATFORM_OPERATOR: Managed service provider (your team) - sees all managed tenants
 * TENANT_ADMIN: Client's IT team - manages their own tenant
 * TENANT_USER: End users within a tenant - limited to assigned datasources
 */
export type UserAccessLevel = 'platform_operator' | 'tenant_admin' | 'tenant_user';

/**
 * Tenant assignment - links users to tenants they can access
 */
export interface TenantAssignment {
  tenantId: string;
  tenantName: string;
  accessLevel: UserAccessLevel;
  /** If true, user can view/support but not modify */
  isReadOnly?: boolean;
  /** Specific instances this user can access (empty = all) */
  instanceIds?: string[];
}

/**
 * Extended user with access control information
 */
export interface AccessControlUser {
  id: string;
  email: string;
  name: string;
  organization: string;

  /** Global access level - highest privilege */
  globalAccessLevel: UserAccessLevel;

  /** Whether user is a platform operator (can see all tenants) */
  isPlatformOperator: boolean;

  /** Specific tenant assignments */
  tenantAssignments: TenantAssignment[];

  /** Legacy fields for backward compatibility */
  is_core_admin?: boolean;
  isCoreAdmin?: boolean;
  is_admin?: boolean;
  role?: string;
  permissions?: string[];
}

/**
 * Scope levels - from broadest to most granular
 */
export type ScopeLevel = 'global' | 'tenant' | 'instance' | 'product' | 'datasource';

/**
 * Current operating scope
 * 
 * Platform operators can work at any level.
 * Tenant admins start at their tenant level.
 * Tenant users are typically scoped to specific datasources.
 */
export interface OperatingScope {
  level: ScopeLevel;

  /** If viewing all tenants (platform operators only) */
  isGlobal?: boolean;

  /** Current tenant context */
  tenantId?: string;
  tenantName?: string;

  /** Current instance context */
  instanceId?: string;
  instanceName?: string;

  /** Current product context */
  productId?: string;
  productName?: string;

  /** Current datasource context */
  datasourceId?: string;
  datasourceName?: string;
}

/**
 * Feature requirements - what scope level a feature needs
 */
export interface FeatureRequirement {
  /** Minimum scope level required */
  minScope: ScopeLevel;
  /** Minimum access level required */
  minAccess: UserAccessLevel;
  /** Whether feature is read-only at this scope */
  readOnly?: boolean;
}

/**
 * Navigation configuration with access control
 */
export interface NavigationItemConfig {
  path: string;
  label: string;
  icon?: React.ReactNode;
  description?: string;
  /** Required scope and access for this item */
  requirement?: FeatureRequirement;
  /** Children for nested navigation */
  children?: NavigationItemConfig[];
}

/**
 * Define which features are available at each scope level
 */
export const FEATURE_REQUIREMENTS: Record<string, FeatureRequirement> = {
  // Global features (platform operators only)
  '/admin/tenants': { minScope: 'global', minAccess: 'platform_operator' },
  '/admin/monitoring': { minScope: 'global', minAccess: 'platform_operator' },
  '/admin/support': { minScope: 'global', minAccess: 'platform_operator' },
  '/admin/entity-manager': { minScope: 'global', minAccess: 'platform_operator' },
  '/admin/temporal-ops': { minScope: 'global', minAccess: 'platform_operator' },
  '/admin/seeding': { minScope: 'global', minAccess: 'platform_operator' },
  '/admin/related-objects': { minScope: 'global', minAccess: 'platform_operator' },

  // Tenant-level features (no datasource needed, just tenant selected)
  '/tenants': { minScope: 'global', minAccess: 'tenant_user' },  // Anyone can view tenants
  '/tenants/connections': { minScope: 'tenant', minAccess: 'tenant_admin' },

  // Instance-level features
  '/instances/products': { minScope: 'instance', minAccess: 'tenant_admin' },

  // Datasource-level features (most features)
  '/fabric/bundles': { minScope: 'instance', minAccess: 'tenant_user' },
  '/fabric/roles': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/fabric/calculations': { minScope: 'instance', minAccess: 'tenant_user' },
  '/fabric/custom-components': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/policy-management': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/api-catalog': { minScope: 'instance', minAccess: 'tenant_user' },
  '/views': { minScope: 'instance', minAccess: 'tenant_user' },
  '/core/glossary': { minScope: 'tenant', minAccess: 'tenant_user' },
  '/core/domains': { minScope: 'tenant', minAccess: 'tenant_user' },
  '/core/validation-rules': { minScope: 'instance', minAccess: 'tenant_user' },
  '/core/catalog-setup': { minScope: 'tenant', minAccess: 'tenant_admin' },
  '/core/semantic-mapper': { minScope: 'instance', minAccess: 'tenant_user' },
  '/core/bp-builder': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/core/approval-workflows': { minScope: 'instance', minAccess: 'tenant_user' },
  '/query-builder': { minScope: 'instance', minAccess: 'tenant_user' },
  '/model-generator': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/model-builder': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/claim-aware-lineage': { minScope: 'instance', minAccess: 'tenant_user' },
  '/drift-report': { minScope: 'instance', minAccess: 'tenant_user', readOnly: true },
  '/access-intelligence': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/access-debugger': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/pre-aggregation-advisor': { minScope: 'instance', minAccess: 'tenant_user' },
  '/frontier-explorer': { minScope: 'instance', minAccess: 'tenant_user' },
  '/marketplace': { minScope: 'instance', minAccess: 'tenant_user' },
  '/dynamic-ui': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/semantic-layout-builder': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/reporting': { minScope: 'instance', minAccess: 'tenant_user' },
  '/notification-dashboard': { minScope: 'instance', minAccess: 'tenant_user' },
  '/notification-rules': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/notification-campaigns': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/upgrade': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/upgrade-compare': { minScope: 'instance', minAccess: 'tenant_admin' },
  '/fabric/ip-whitelist': { minScope: 'tenant', minAccess: 'tenant_admin' },
  '/metrics/calc-console': { minScope: 'instance', minAccess: 'tenant_user' },
};

/**
 * Helper to check if user can access a feature
 */
export function canAccessFeature(
  user: AccessControlUser | null,
  scope: OperatingScope,
  path: string
): { allowed: boolean; reason?: string } {
  if (!user) {
    return { allowed: false, reason: 'Not authenticated' };
  }

  const requirement = FEATURE_REQUIREMENTS[path];
  if (!requirement) {
    // Default: allow if authenticated
    return { allowed: true };
  }

  // Check access level
  const accessLevels: UserAccessLevel[] = ['tenant_user', 'tenant_admin', 'platform_operator'];
  const userLevel = accessLevels.indexOf(user.globalAccessLevel);
  const requiredLevel = accessLevels.indexOf(requirement.minAccess);

  if (userLevel < requiredLevel) {
    return { allowed: false, reason: `Requires ${requirement.minAccess} access` };
  }

  // Check scope level
  const scopeLevels: ScopeLevel[] = ['global', 'tenant', 'instance', 'product', 'instance'];
  const requiredScope = scopeLevels.indexOf(requirement.minScope);
  const currentScope = scopeLevels.indexOf(scope.level);

  // For non-global features, user must have selected down to required level
  if (requirement.minScope !== 'global' && currentScope < requiredScope) {
    return {
      allowed: false,
      reason: `Please select a ${requirement.minScope} first`
    };
  }

  return { allowed: true };
}

/**
 * Get human-readable scope description
 */
export function getScopeDescription(scope: OperatingScope): string {
  if (scope.isGlobal) return 'All Tenants';

  const parts: string[] = [];
  if (scope.tenantName) parts.push(scope.tenantName);
  if (scope.instanceName) parts.push(scope.instanceName);
  if (scope.productName) parts.push(scope.productName);
  if (scope.datasourceName) parts.push(scope.datasourceName);

  return parts.join(' → ') || 'No scope selected';
}

/**
 * Get the minimum required scope for the current path
 */
export function getRequiredScope(path: string): ScopeLevel {
  const requirement = FEATURE_REQUIREMENTS[path];
  return requirement?.minScope || 'global';
}
