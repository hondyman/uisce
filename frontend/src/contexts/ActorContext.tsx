import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';

/**
 * Actor types for the Scheduler Intelligence Console
 */
export type ActorRole = 'TENANT_OPS' | 'GLOBAL_OPS';

/**
 * Actor permissions based on UX spec
 */
export interface ActorPermissions {
  // Visibility
  canViewCrossTenant: boolean;
  canViewGlobalCalendars: boolean;
  canViewExceptionClusters: boolean;
  canViewGlobalAIInsights: boolean;
  
  // Actions
  canCreateGlobalJobs: boolean;
  canCreateGlobalDAGs: boolean;
  canApproveChangeSets: boolean;
  canOverrideSchedules: boolean;
  canManageCalendars: boolean;
  
  // Governance
  canProposeChangeSets: boolean;
  canRejectChangeSets: boolean;
}

/**
 * Actor context state
 */
export interface ActorContextState {
  // Current actor role
  role: ActorRole;
  
  // For tenant ops, the specific tenant ID
  tenantId: string | null;
  tenantName: string | null;
  
  // Permissions derived from role
  permissions: ActorPermissions;
  
  // Methods
  setRole: (role: ActorRole) => void;
  setTenant: (tenantId: string, tenantName: string) => void;
  isTenantOps: () => boolean;
  isGlobalOps: () => boolean;
}

// Default permissions for each role
const TENANT_OPS_PERMISSIONS: ActorPermissions = {
  canViewCrossTenant: false,
  canViewGlobalCalendars: false,
  canViewExceptionClusters: false, // Only tenant-scoped
  canViewGlobalAIInsights: false,
  canCreateGlobalJobs: false,
  canCreateGlobalDAGs: false,
  canApproveChangeSets: false,
  canOverrideSchedules: false,
  canManageCalendars: false,
  canProposeChangeSets: true,
  canRejectChangeSets: false,
};

const GLOBAL_OPS_PERMISSIONS: ActorPermissions = {
  canViewCrossTenant: true,
  canViewGlobalCalendars: true,
  canViewExceptionClusters: true,
  canViewGlobalAIInsights: true,
  canCreateGlobalJobs: true,
  canCreateGlobalDAGs: true,
  canApproveChangeSets: true,
  canOverrideSchedules: true,
  canManageCalendars: true,
  canProposeChangeSets: true,
  canRejectChangeSets: true,
};

// Create context
const ActorContext = createContext<ActorContextState | undefined>(undefined);

/**
 * Actor Provider component
 */
export const ActorProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [role, setRoleState] = useState<ActorRole>('TENANT_OPS');
  const [tenantId, setTenantId] = useState<string | null>(null);
  const [tenantName, setTenantName] = useState<string | null>(null);
  const [permissions, setPermissions] = useState<ActorPermissions>(TENANT_OPS_PERMISSIONS);

  // Update permissions when role changes
  useEffect(() => {
    if (role === 'GLOBAL_OPS') {
      setPermissions(GLOBAL_OPS_PERMISSIONS);
    } else {
      setPermissions(TENANT_OPS_PERMISSIONS);
    }
  }, [role]);

  // Detect role from user context (would integrate with auth)
  useEffect(() => {
    // In production, read from auth context
    // For now, check localStorage or default to tenant_ops
    const storedRole = localStorage.getItem('scheduler_actor_role') as ActorRole;
    const storedTenantId = localStorage.getItem('scheduler_tenant_id');
    const storedTenantName = localStorage.getItem('scheduler_tenant_name');
    
    if (storedRole) {
      setRoleState(storedRole as ActorRole);
    }
    if (storedTenantId) {
      setTenantId(storedTenantId);
      setTenantName(storedTenantName || 'Unknown Tenant');
    }
  }, []);

  const setRole = (newRole: ActorRole) => {
    setRoleState(newRole);
    localStorage.setItem('scheduler_actor_role', newRole);
  };

  const setTenant = (newTenantId: string, newTenantName: string) => {
    setTenantId(newTenantId);
    setTenantName(newTenantName);
    localStorage.setItem('scheduler_tenant_id', newTenantId);
    localStorage.setItem('scheduler_tenant_name', newTenantName);
  };

  const isTenantOps = () => role === 'TENANT_OPS';
  const isGlobalOps = () => role === 'GLOBAL_OPS';

  const value: ActorContextState = {
    role,
    tenantId,
    tenantName,
    permissions,
    setRole,
    setTenant,
    isTenantOps,
    isGlobalOps,
  };

  return (
    <ActorContext.Provider value={value}>
      {children}
    </ActorContext.Provider>
  );
};

/**
 * Hook to access actor context
 */
export const useActor = (): ActorContextState => {
  const context = useContext(ActorContext);
  if (!context) {
    throw new Error('useActor must be used within an ActorProvider');
  }
  return context;
};

/**
 * Hook to check specific permission
 */
export const usePermission = (permission: keyof ActorPermissions): boolean => {
  const { permissions } = useActor();
  return permissions[permission];
};

/**
 * Component to conditionally render based on actor role
 */
export const ForActor: React.FC<{
  role: ActorRole | ActorRole[];
  children: ReactNode;
  fallback?: ReactNode;
}> = ({ role, children, fallback = null }) => {
  const { role: currentRole } = useActor();
  const roles = Array.isArray(role) ? role : [role];
  
  if (roles.includes(currentRole)) {
    return <>{children}</>;
  }
  return <>{fallback}</>;
};

/**
 * Component to conditionally render based on permission
 */
export const RequirePermission: React.FC<{
  permission: keyof ActorPermissions;
  children: ReactNode;
  fallback?: ReactNode;
}> = ({ permission, children, fallback = null }) => {
  const hasPermission = usePermission(permission);
  
  if (hasPermission) {
    return <>{children}</>;
  }
  return <>{fallback}</>;
};

/**
 * Actor role switcher for development/testing
 */
export const ActorSwitcher: React.FC = () => {
  const { role, setRole, tenantName } = useActor();
  
  return (
    <div className="actor-switcher">
      <select 
        value={role} 
        onChange={(e) => setRole(e.target.value as ActorRole)}
        className="actor-select"
      >
        <option value="TENANT_OPS">Tenant Ops</option>
        <option value="GLOBAL_OPS">Global Ops</option>
      </select>
      {role === 'TENANT_OPS' && tenantName && (
        <span className="tenant-badge">{tenantName}</span>
      )}
    </div>
  );
};

export default ActorContext;
