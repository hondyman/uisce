import React, { createContext, useContext, useState, useEffect, ReactNode, useCallback, useMemo } from 'react';
import { Tenant, Product, DataSource } from '../types';
import { devLog, devError } from '../utils/devLogger';
import { setSelectedRegion, getSelectedRegion } from '../lib/region';

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

  // Load selection from localStorage on mount
  useEffect(() => {
    try {
      const storedTenant = localStorage.getItem(TENANT_STORAGE_KEYS.TENANT);
      const storedProduct = localStorage.getItem(TENANT_STORAGE_KEYS.PRODUCT);
      const storedDatasource = localStorage.getItem(TENANT_STORAGE_KEYS.DATASOURCE);

      if (storedTenant && storedProduct && storedDatasource) {
        const parsedTenant = JSON.parse(storedTenant);
        const parsedProduct = JSON.parse(storedProduct);
        let parsedDatasource = JSON.parse(storedDatasource);

        // Auto-fix: Check for incorrect datasource ID (097...) and replace with correct one (25b...)
        // This fixes the issue where an old auto-fix set the wrong ID
        if (parsedDatasource.id === '097793ae-ceeb-48f4-a7bb-4bcbba1796ae') {
          devLog('Auto-fixing incorrect datasource ID in localStorage (097 -> 25b)');
          parsedDatasource = {
            ...parsedDatasource,
            id: '25b5dce3-27d9-4773-933e-6ee29a42871f',
            source_name: 'Northwinds' 
          };
          // Update localStorage immediately
          localStorage.setItem(TENANT_STORAGE_KEYS.DATASOURCE, JSON.stringify(parsedDatasource));
        }

        setTenant(parsedTenant);
        setProduct(parsedProduct);
        setDatasource(parsedDatasource);

        // Use tenant's region from the API response
        if (parsedTenant.region) {
          devLog(`Setting region from tenant: ${parsedTenant.region}`);
          setSelectedRegion(parsedTenant.region);
        } else if (parsedTenant.allowed_regions && parsedTenant.allowed_regions.length > 0) {
          // Fallback to first allowed region if region field is not set
          devLog(`Setting region from allowed_regions: ${parsedTenant.allowed_regions[0]}`);
          setSelectedRegion(parsedTenant.allowed_regions[0]);
        }
      } else if ((import.meta as any).env && (import.meta as any).env.DEV) {
        // Set default test data for development
        devLog('Setting default tenant scope for development');
        const defaultTenant = {
          id: '99e99e99-99e9-49e9-89e9-99e99e99e999',
          name: 'Uiscé',
          display_name: 'Uiscé',
          gold_copy: true,
          description: 'Default test tenant for development',
          is_active: true,
          region: 'us-west',
          tenant_instances: [],
          allowed_regions: ['us-west']
        };
        const defaultProduct = {
          id: 'test-product-id',
          version: 1,
          tenant_instance_id: 'test-instance-id',
          alpha_product_id: 'test-alpha-product-id',
          alpha_product: {
            id: 'test-alpha-product-id',
            product_name: 'Test Product',
            product_code: 'TEST',
            is_active: true
          },
          tenant_product_datasources: []
        };
        const defaultDatasource = {
          id: '25b5dce3-27d9-4773-933e-6ee29a42871f', // Northwinds Data
          alpha_tenant_instance_id: 'test-alpha-ds-id',
          is_active: true,
          config: {},
          source_name: 'Northwinds',
          alpha_datasource: {
            id: 'test-alpha-ds-id',
            datasource_name: 'Northwinds',
            datasource_type: 'postgres'
          }
        };
        setTenant(defaultTenant);
        setProduct(defaultProduct);
        setDatasource(defaultDatasource);
        
        // Set region from default tenant
        devLog('Setting region from default tenant: us-west');
        setSelectedRegion('us-west');
        // Immediately save to localStorage for setupTenantFetch
        localStorage.setItem(TENANT_STORAGE_KEYS.TENANT, JSON.stringify(defaultTenant));
        localStorage.setItem(TENANT_STORAGE_KEYS.PRODUCT, JSON.stringify(defaultProduct));
        localStorage.setItem(TENANT_STORAGE_KEYS.DATASOURCE, JSON.stringify(defaultDatasource));
      }
    } catch (error) {
      devError('Error loading tenant selection from localStorage:', error);
      // Clear invalid data
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