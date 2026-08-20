import React, { createContext, useContext, ReactNode, useMemo } from 'react';
import { Tenant, Product, DataSource } from '../types';
import { useAccess } from './AccessContext';

interface TenantContextType {
  tenant: Tenant | null;
  product: Product | null;
  datasource: DataSource | null;
  setSelection: (tenant: Tenant, product: Product, datasource: DataSource) => void;
  clearSelection: () => void;
  isSelected: boolean;
}

const TenantContext = createContext<TenantContextType | undefined>(undefined);

export const TENANT_STORAGE_KEYS = {
  TENANT: 'selected_tenant',
  PRODUCT: 'selected_product', 
  DATASOURCE: 'selected_datasource'
};

interface TenantProviderProps {
  children: ReactNode;
}

/**
 * TenantProvider acts as a transparent bridge over AccessContext.
 * AccessContext is the single source of truth for all tenant, instance,
 * product, datasource and JWT-scoped access.
 */
export const TenantProvider: React.FC<TenantProviderProps> = ({ children }) => {
  const access = useAccess();

  const contextValue = useMemo(() => ({
    tenant: access.currentTenant,
    product: access.currentProduct,
    datasource: access.currentDatasource,
    setSelection: access.setSelection,
    clearSelection: access.clearScope,
    isSelected: access.isSelected,
  }), [
    access.currentTenant,
    access.currentProduct,
    access.currentDatasource,
    access.setSelection,
    access.clearScope,
    access.isSelected,
  ]);

  return (
    <TenantContext.Provider value={contextValue}>
      {children}
    </TenantContext.Provider>
  );
};

export const useTenant = (): TenantContextType => {
  const context = useContext(TenantContext);
  if (context !== undefined) {
    return context;
  }
  // If outside TenantProvider but inside AccessProvider, fall back directly to useAccess
  try {
    const access = useAccess();
    return {
      tenant: access.currentTenant,
      product: access.currentProduct,
      datasource: access.currentDatasource,
      setSelection: access.setSelection,
      clearSelection: access.clearScope,
      isSelected: access.isSelected,
    };
  } catch (_) {
    throw new Error('useTenant must be used within an AccessProvider or TenantProvider');
  }
};