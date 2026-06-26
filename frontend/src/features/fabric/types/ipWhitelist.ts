export interface IPWhitelistEntry {
  id: string;
  ipAddress: string;
  label: string;
  description?: string;
  tenantIds: string[];
  createdAt?: string;
  updatedAt?: string;
  isActive?: boolean;
}

export interface Tenant {
  id: string;
  displayName: string;
  description?: string;
  name?: string;
  tenant_code?: string;
}

// Special tenant ID for "All Tenants" capability
export const ALL_TENANTS_ID = '__ALL_TENANTS__';
export const ALL_TENANTS_DISPLAY_NAME = 'All Tenants';

export interface IPAssignment {
  ipAddress: string;
  label?: string | null;
  description?: string | null;
  tenantId: string;
  tenantDisplayName: string;
  assignedAt?: string;
}

export interface TenantIPSummary {
  tenant: Tenant;
  ipAddresses: IPWhitelistEntry[];
  totalIPs: number;
  activeIPs: number;
}

export interface IPWhitelistFilters {
  search: string;
  tenantFilter: string; // '', 'all', 'assigned', 'unassigned', or tenantId
  statusFilter: 'all' | 'active' | 'inactive';
  sortBy: 'ipAddress' | 'label' | 'tenantCount' | 'createdAt';
  sortOrder: 'asc' | 'desc';
}

export interface TenantFilters {
  search: string;
  hasIPs: 'all' | 'with-ips' | 'without-ips';
  sortBy: 'displayName' | 'ipCount' | 'lastAssigned';
  sortOrder: 'asc' | 'desc';
  selectedTenantIds: string[];
}

export interface ConflictInfo {
  ipAddress: string;
  conflictingTenantIds: string[];
  conflictingTenantNames: string[];
}
