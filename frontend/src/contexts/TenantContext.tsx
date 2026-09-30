import React, { createContext, useContext, ReactNode, useMemo } from 'react';
import { Tenant, Product, DataSource } from '../types';
import { useAccess, useAccessOptional } from './AccessContext';

/**
 * Public shape of the tenant context. Exported so test fixtures (e.g.
 * `vitest/fixtures/tenantContextStub.ts`) can compile-time-check their stub
 * shape against the real module's return type with a bidirectional exactness
 * comparison. Without this export, fixtures would have to fall back on `as any`,
 * which is assignable to every type and therefore detects nothing.
 */
export interface TenantContextType {
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
  // Both contexts are read unconditionally, at the top level, so the hook count no
  // longer depends on whether a provider is mounted. `useAccess` was previously
  // wrapped in `try { useAccess() } catch`, and the tenant early-return sat above
  // it, so provider presence changing between renders was a live hook-count crash
  // ("Rendered fewer hooks than expected") and could cross state between mismatched
  // hook instances.
  const context = useContext(TenantContext);
  const access = useAccessOptional();

  if (context !== undefined) {
    return context;
  }

  // Outside TenantProvider but inside AccessProvider, fall back directly to the
  // access context.
  //
  // The failure is deliberately still loud, and still carries this message:
  // `useAccessOptional` returns `undefined` (the context default) when no
  // AccessProvider is mounted. This is NOT a silent-default fallback — a caller
  // cannot observe a missing provider here.
  if (access === undefined) {
    throw new Error('useTenant must be used within an AccessProvider or TenantProvider');
  }
  return {
    tenant: access.currentTenant,
    product: access.currentProduct,
    datasource: access.currentDatasource,
    setSelection: access.setSelection,
    clearSelection: access.clearScope,
    isSelected: access.isSelected,
  };
};