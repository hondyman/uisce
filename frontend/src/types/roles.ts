export type RoleStatus = 'Draft' | 'Active' | 'Suspended' | 'Retired';
export type RoleType = 'Business' | 'System' | 'Technical';
export type RoleScope = 'Global' | 'Tenant' | 'Environment';

export interface RoleSummary {
  id: string;
  name: string;
  displayName: string;
  description?: string;
  status: RoleStatus;
  type: RoleType;
  owner: string;
  scope?: RoleScope;
  tags?: string[];
  bundleIds?: string[];
  updatedAt: string;
}

export interface RoleDetail extends RoleSummary {
  tenantId?: string;
  attributes?: Record<string, string>;
  permissions?: Array<{
    resource: string;
    actions: string[];
    effect: string;
    description?: string;
  }>;
}
