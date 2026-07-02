// frontend/src/contexts/AccessContext.tsx
// Unified access control context - replaces fragmented TenantContext

import React, { createContext, useContext, useState, useEffect, useCallback, ReactNode } from 'react';
import { 
  UserAccessLevel, 
  TenantAssignment, 
  OperatingScope, 
  ScopeLevel,
  canAccessFeature,
  getScopeDescription 
} from '../types/access';
import { Tenant, TenantInstance, Product, DataSource } from '../types';
import { useAuth } from './AuthContext';
import { devLog, devWarn, devError } from '../utils/devLogger';
import { setSelectedRegion } from '../lib/region';
import { apiFetch } from '../lib/apiClient';

// Storage keys
export const ACCESS_STORAGE_KEYS = {
  SCOPE: 'operating_scope',
  TENANT_ASSIGNMENTS: 'tenant_assignments',
};

interface AccessContextType {
  // Current user's access level
  accessLevel: UserAccessLevel;
  isPlatformOperator: boolean;
  
  // Tenant assignments for this user
  tenantAssignments: TenantAssignment[];
  accessibleTenants: Tenant[];
  
  // Current operating scope
  scope: OperatingScope;
  scopeDescription: string;
  
  // Scope navigation
  setGlobalScope: () => void;
  setTenantScope: (tenant: Tenant) => void;
  setInstanceScope: (tenant: Tenant, instance: TenantInstance) => void;
  setProductScope: (tenant: Tenant, instance: TenantInstance, product: Product) => void;
  setDatasourceScope: (tenant: Tenant, instance: TenantInstance, product: Product, datasource: DataSource) => void;
  clearScope: () => void;
  
  // Convenience - scope at different levels
  currentTenant: Tenant | null;
  currentInstance: TenantInstance | null;
  currentProduct: Product | null;
  currentDatasource: DataSource | null;
  
  // Feature access checks
  canAccess: (path: string) => { allowed: boolean; reason?: string };
  requiresScope: (path: string) => ScopeLevel;
  
  // Legacy compatibility (for existing useTenant consumers)
  tenant: Tenant | null;
  product: Product | null;
  datasource: DataSource | null;
  isSelected: boolean;
  setSelection: (tenant: Tenant, product: Product, datasource: DataSource) => void;
  
  // Loading state
  isLoading: boolean;
}

const AccessContext = createContext<AccessContextType | undefined>(undefined);

interface AccessProviderProps {
  children: ReactNode;
}

export const AccessProvider: React.FC<AccessProviderProps> = ({ children }) => {
  const { user, isAuthenticated } = useAuth();
  
  const [accessLevel, setAccessLevel] = useState<UserAccessLevel>('tenant_user');
  const [tenantAssignments, _setTenantAssignments] = useState<TenantAssignment[]>([]);
  const [accessibleTenants, setAccessibleTenants] = useState<Tenant[]>([]);
  const [scope, setScope] = useState<OperatingScope>({ level: 'global', isGlobal: true });
  const [isLoading, setIsLoading] = useState(true);
  
  // Expanded scope state
  const [currentTenant, setCurrentTenant] = useState<Tenant | null>(null);
  const [currentInstance, setCurrentInstance] = useState<TenantInstance | null>(null);
  const [currentProduct, setCurrentProduct] = useState<Product | null>(null);
  const [currentDatasource, setCurrentDatasource] = useState<DataSource | null>(null);

  // Derive access level from user
  useEffect(() => {
    if (!user) {
      setAccessLevel('tenant_user');
      return;
    }

    // Check for platform operator status
    // Platform operators are identified by:
    // 1. is_core_admin or isCoreAdmin flag
    // 2. role === 'platform_operator' or 'admin' or 'global_ops'
    // 3. Having 'platform:operator' permission
    const isPlatformOp = 
      user.is_core_admin === true ||
      user.isCoreAdmin === true ||
      user.role === 'platform_operator' ||
      user.role === 'global_ops' ||
      user.role === 'admin' ||
      user.role === 'global_admin' ||
      user.is_global_admin === true ||
      user.permissions?.includes('platform:operator') ||
      user.permissions?.includes('*');

    if (isPlatformOp) {
      setAccessLevel('platform_operator');
      devLog('User is platform operator');
    } else if (user.role === 'tenant_admin' || user.permissions?.includes('tenant:admin')) {
      setAccessLevel('tenant_admin');
      devLog('User is tenant admin');
    } else {
      setAccessLevel('tenant_user');
      devLog('User is tenant user');
    }
  }, [user]);

  // Load scope from storage on mount
  useEffect(() => {
    const loadPersistedScope = () => {
      try {
        const storedScope = localStorage.getItem(ACCESS_STORAGE_KEYS.SCOPE);
        if (storedScope) {
          const parsed = JSON.parse(storedScope);
          setScope(parsed);
          
          // Also restore the full objects if we have IDs
          // This would need to be enhanced to fetch the full objects
          devLog('Restored scope from storage:', parsed);
        }
      } catch (error) {
        devError('Error loading scope from storage:', error);
      } finally {
        setIsLoading(false);
      }
    };

    loadPersistedScope();
  }, []);

  // Fetch accessible tenants based on user access level
  useEffect(() => {
    if (!isAuthenticated) {
      setAccessibleTenants([]);
      return;
    }

    const fetchTenants = async () => {
      try {
        // Platform operators see all tenants with full hierarchy (instances, products,
        // datasources) via /api/tenants/all. Non-operators get a user-scoped projection
        // via /api/tenants/accessible.
        const endpoint = accessLevel === 'platform_operator'
          ? '/api/tenants/all'
          : '/api/tenants/accessible';

        devLog(`Fetching tenants from ${endpoint}`);
        // apiFetch injects Authorization: Bearer <jwt> + X-Tenant-ID + region automatically
        const response = await apiFetch(endpoint, {
          credentials: import.meta.env.DEV ? 'include' : 'same-origin'
        });

        if (response.ok) {
          const json = await response.json();
          const tenantsList = json.success ? json.data : json;
          const list = Array.isArray(tenantsList) ? tenantsList : [];
          setAccessibleTenants(list);
          devLog(`Loaded ${list.length} accessible tenants`, list);
        } else {
          devWarn(`Failed to fetch tenants: ${response.status} ${response.statusText}`);
        }
      } catch (error) {
        devWarn('Failed to fetch tenants:', error);
        setAccessibleTenants([]);
      }
    };

    fetchTenants();
  }, [isAuthenticated, accessLevel]);

  // Persist scope changes
  useEffect(() => {
    if (!isLoading) {
      try {
        localStorage.setItem(ACCESS_STORAGE_KEYS.SCOPE, JSON.stringify(scope));
      } catch (error) {
        devError('Error persisting scope:', error);
      }
    }
  }, [scope, isLoading]);

  // Scope setters
  const setGlobalScope = useCallback(() => {
    if (accessLevel !== 'platform_operator') {
      devWarn('Only platform operators can set global scope');
      return;
    }
    setScope({ level: 'global', isGlobal: true });
    setCurrentTenant(null);
    setCurrentInstance(null);
    setCurrentProduct(null);
    setCurrentDatasource(null);
    devLog('Scope set to global');
  }, [accessLevel]);

  const setTenantScope = useCallback((tenant: Tenant) => {
    setScope({
      level: 'tenant',
      isGlobal: false,
      tenantId: tenant.id,
      tenantName: tenant.display_name || tenant.name,
    });
    setCurrentTenant(tenant);
    setCurrentInstance(null);
    setCurrentProduct(null);
    setCurrentDatasource(null);
    devLog('Scope set to tenant:', tenant.display_name);
  }, []);

  const setInstanceScope = useCallback((tenant: Tenant, instance: TenantInstance) => {
    setScope({
      level: 'instance',
      isGlobal: false,
      tenantId: tenant.id,
      tenantName: tenant.display_name || tenant.name,
      instanceId: instance.id,
      instanceName: instance.display_name || instance.instance_name,
    });
    setCurrentTenant(tenant);
    setCurrentInstance(instance);
    setCurrentProduct(null);
    setCurrentDatasource(null);
    devLog('Scope set to instance:', instance.display_name);
  }, []);

  const setProductScope = useCallback((tenant: Tenant, instance: TenantInstance, product: Product) => {
    setScope({
      level: 'product',
      isGlobal: false,
      tenantId: tenant.id,
      tenantName: tenant.display_name || tenant.name,
      instanceId: instance.id,
      instanceName: instance.display_name || instance.instance_name,
      productId: product.id,
      productName: product.alpha_product?.product_name,
    });
    setCurrentTenant(tenant);
    setCurrentInstance(instance);
    setCurrentProduct(product);
    setCurrentDatasource(null);
    devLog('Scope set to product:', product.alpha_product?.product_name);
  }, []);

  const setDatasourceScope = useCallback((
    tenant: Tenant, 
    instance: TenantInstance, 
    product: Product, 
    datasource: DataSource
  ) => {
    setScope({
      level: 'datasource',
      isGlobal: false,
      tenantId: tenant.id,
      tenantName: tenant.display_name || tenant.name,
      instanceId: instance.id,
      instanceName: instance.display_name || instance.instance_name,
      productId: product.id,
      productName: product.alpha_product?.product_name,
      datasourceId: datasource.id,
      datasourceName: datasource.source_name,
    });
    setCurrentTenant(tenant);
    setCurrentInstance(instance);
    setCurrentProduct(product);
    setCurrentDatasource(datasource);
    
    // Set region from tenant
    if (tenant.region) {
      try {
        setSelectedRegion(tenant.region);
        devLog(`Set region from tenant: ${tenant.region}`);
      } catch (error) {
        devError('Error setting region:', error);
      }
    }
    
    devLog('Scope set to datasource:', datasource.source_name);
  }, []);

  const clearScope = useCallback(() => {
    if (accessLevel === 'platform_operator') {
      setGlobalScope();
    } else {
      // Non-operators clear to no selection
      setScope({ level: 'global', isGlobal: false });
      setCurrentTenant(null);
      setCurrentInstance(null);
      setCurrentProduct(null);
      setCurrentDatasource(null);
    }
    devLog('Scope cleared');
  }, [accessLevel, setGlobalScope]);

  // Feature access check
  const canAccess = useCallback((path: string) => {
    return canAccessFeature(
      user ? { 
        ...user, 
        globalAccessLevel: accessLevel, 
        isPlatformOperator: accessLevel === 'platform_operator',
        tenantAssignments: []
      } : null, 
      scope, 
      path
    );
  }, [user, scope, accessLevel]);

  const requiresScope = useCallback((path: string): ScopeLevel => {
    const { FEATURE_REQUIREMENTS } = require('../types/access');
    return FEATURE_REQUIREMENTS[path]?.minScope || 'global';
  }, []);

  // Legacy compatibility - maps to old TenantContext interface
  const setSelection = useCallback((tenant: Tenant, product: Product, datasource: DataSource) => {
    // Find the instance from the product
    const instance = tenant.tenant_instances?.find(
      i => i.tenant_products?.some(p => p.id === product.id)
    );
    
    if (instance) {
      setDatasourceScope(tenant, instance, product, datasource);
    } else {
      // Fallback - create minimal instance
      devWarn('Could not find instance for product, creating placeholder');
      const placeholderInstance: TenantInstance = {
        id: product.tenant_instance_id || 'temp-instance-id',
        display_name: 'Instance',
        instance_name: 'instance',
        description: null,
        is_active: true,
        url: null,
        config: {},
        tenant_id: tenant.id,
        tenant_products: [product],
      };
      setDatasourceScope(tenant, placeholderInstance, product, datasource);
    }
  }, [setDatasourceScope]);

  const isPlatformOperator = accessLevel === 'platform_operator';
  const scopeDescription = getScopeDescription(scope);
  const isSelected = Boolean(currentTenant && currentProduct && currentDatasource);

  const contextValue: AccessContextType = {
    accessLevel,
    isPlatformOperator,
    tenantAssignments,
    accessibleTenants,
    scope,
    scopeDescription,
    setGlobalScope,
    setTenantScope,
    setInstanceScope,
    setProductScope,
    setDatasourceScope,
    clearScope,
    currentTenant,
    currentInstance,
    currentProduct,
    currentDatasource,
    canAccess,
    requiresScope,
    // Legacy compatibility
    tenant: currentTenant,
    product: currentProduct,
    datasource: currentDatasource,
    isSelected,
    setSelection,
    isLoading,
  };

  return (
    <AccessContext.Provider value={contextValue}>
      {children}
    </AccessContext.Provider>
  );
};

export const useAccess = (): AccessContextType => {
  const context = useContext(AccessContext);
  if (context === undefined) {
    throw new Error('useAccess must be used within an AccessProvider');
  }
  return context;
};

// Legacy hook - wraps useAccess for backward compatibility
export const useTenantCompat = () => {
  const access = useAccess();
  return {
    tenant: access.tenant,
    product: access.product,
    datasource: access.datasource,
    setSelection: access.setSelection,
    clearSelection: access.clearScope,
    isSelected: access.isSelected,
  };
};
