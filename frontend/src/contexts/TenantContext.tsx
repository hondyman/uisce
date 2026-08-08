import React, { createContext, useContext, useState, useEffect, ReactNode, useCallback, useMemo } from 'react';
import { Tenant, Product, DataSource } from '../types';
import { devLog, devError } from '../utils/devLogger';
import { setSelectedRegion, getSelectedRegion } from '../lib/region';
import { resolveGoldCopyTenantId, getCachedGoldCopyId } from '../utils/goldCopy';

interface TenantContextType {
  tenant: Tenant | null;
  product: Product | null;
  datasource: DataSource | null;
  setSelection: (tenant: Tenant, product: Product, datasource: DataSource) => void;
  clearSelection: () => void;
  isSelected: boolean;
}

const TenantContext = createContext<TenantContextType | undefined>(undefined);

// Storage keys
export const TENANT_STORAGE_KEYS = {
  TENANT: 'selected_tenant',
  PRODUCT: 'selected_product', 
  DATASOURCE: 'selected_datasource'
};

interface TenantProviderProps {
  children: ReactNode;
}

export const TenantProvider: React.FC<TenantProviderProps> = ({ children }) => {
  const [tenant, setTenant] = useState<Tenant | null>(null);
  const [product, setProduct] = useState<Product | null>(null);
  const [datasource, setDatasource] = useState<DataSource | null>(null);

  // Load selection from localStorage on mount or resolve gold copy dynamically from API
  useEffect(() => {
    try {
      const storedTenant = localStorage.getItem(TENANT_STORAGE_KEYS.TENANT);
      const storedProduct = localStorage.getItem(TENANT_STORAGE_KEYS.PRODUCT);
      const storedDatasource = localStorage.getItem(TENANT_STORAGE_KEYS.DATASOURCE);

      if (storedTenant && storedProduct && storedDatasource) {
        const parsedTenant = JSON.parse(storedTenant);
        const parsedProduct = JSON.parse(storedProduct);
        const parsedDatasource = JSON.parse(storedDatasource);

        setTenant(parsedTenant);
        setProduct(parsedProduct);
        setDatasource(parsedDatasource);

        if (parsedTenant.region) {
          devLog(`Setting region from tenant: ${parsedTenant.region}`);
          setSelectedRegion(parsedTenant.region);
        } else if (parsedTenant.allowed_regions && parsedTenant.allowed_regions.length > 0) {
          devLog(`Setting region from allowed_regions: ${parsedTenant.allowed_regions[0]}`);
          setSelectedRegion(parsedTenant.allowed_regions[0]);
        }
      } else {
        // Dynamically fetch gold copy tenant details from backend API
        resolveGoldCopyTenantId().then(goldCopyId => {
          if (!goldCopyId) return;
          fetch(`/api/tenants/${goldCopyId}`)
            .then(res => res.ok ? res.json() : null)
            .then(tenantData => {
              if (tenantData) {
                setTenant(tenantData);
                if (tenantData.region) setSelectedRegion(tenantData.region);
                localStorage.setItem(TENANT_STORAGE_KEYS.TENANT, JSON.stringify(tenantData));
              }
            })
            .catch(err => devError('Failed to fetch gold copy tenant:', err));
        });
      }
    } catch (error) {
      devError('Error loading tenant selection:', error);
      setTenant(null);
      setProduct(null);
      setDatasource(null);
    }
  }, []);

  // Save selection to localStorage whenever it changes
  useEffect(() => {
    if (tenant && product && datasource) {
      try {
        localStorage.setItem(TENANT_STORAGE_KEYS.TENANT, JSON.stringify(tenant));
        localStorage.setItem(TENANT_STORAGE_KEYS.PRODUCT, JSON.stringify(product));
        localStorage.setItem(TENANT_STORAGE_KEYS.DATASOURCE, JSON.stringify(datasource));
      } catch (error) {
        devError('Error saving tenant selection to localStorage:', error);
      }
    }
  }, [tenant, product, datasource]);

  const setSelection = useCallback((selectedTenant: Tenant, selectedProduct: Product, selectedDatasource: DataSource) => {
    setTenant(selectedTenant);
    setProduct(selectedProduct);
    setDatasource(selectedDatasource);
    
    devLog('Tenant selection updated:', {
      tenant: String(selectedTenant.display_name || selectedTenant.name || 'Unnamed Tenant'),
      product: selectedProduct.alpha_product?.product_name,
      datasource: selectedDatasource.source_name,
      datasourceId: selectedDatasource.id
    });

    // Auto-select region if available
    if (selectedTenant.allowed_regions && selectedTenant.allowed_regions.length > 0) {
      // Default to first region
      const defaultRegion = selectedTenant.allowed_regions[0];
      devLog(`Setting default region for tenant: ${defaultRegion}`);
      setSelectedRegion(defaultRegion);
    }
  }, []);

  const clearSelection = useCallback(() => {
    setTenant(null);
    setProduct(null);
    setDatasource(null);
    
    localStorage.removeItem(TENANT_STORAGE_KEYS.TENANT);
    localStorage.removeItem(TENANT_STORAGE_KEYS.PRODUCT);
    localStorage.removeItem(TENANT_STORAGE_KEYS.DATASOURCE);
    
    devLog('Tenant selection cleared');
  }, []);

  const isSelected = !!(tenant && product && datasource);

  const contextValue = useMemo(() => ({
    tenant,
    product,
    datasource,
    setSelection,
    clearSelection,
    isSelected
  }), [tenant, product, datasource, setSelection, clearSelection, isSelected]);

  return (
    <TenantContext.Provider value={contextValue}>
      {children}
    </TenantContext.Provider>
  );
};

export const useTenant = (): TenantContextType => {
  const context = useContext(TenantContext);
  if (context === undefined) {
    throw new Error('useTenant must be used within a TenantProvider');
  }
  return context;
};