import { useCallback } from 'react';

export interface Tenant {
  id: string;
  display_name: string;
}

export interface Product {
  id: string;
  alpha_product: {
    product_name: string;
  };
}

export interface Datasource {
  id: string;
  source_name: string;
}

export interface TenantContextType {
  selectedTenant: Tenant | null;
  selectedProduct: Product | null;
  selectedDatasource: Datasource | null;
  setSelectedTenant: (tenant: Tenant) => void;
  setSelectedProduct: (product: Product) => void;
  setSelectedDatasource: (datasource: Datasource) => void;
  clearSelection: () => void;
  hasValidScope: boolean;
}

const STORAGE_KEYS = {
  TENANT: 'selected_tenant',
  PRODUCT: 'selected_product',
  DATASOURCE: 'selected_datasource',
};

export const useTenantContext = (): TenantContextType => {
  const getSelectedTenant = useCallback((): Tenant | null => {
    try {
      const stored = localStorage.getItem(STORAGE_KEYS.TENANT);
      return stored ? JSON.parse(stored) : null;
    } catch {
      return null;
    }
  }, []);

  const getSelectedProduct = useCallback((): Product | null => {
    try {
      const stored = localStorage.getItem(STORAGE_KEYS.PRODUCT);
      return stored ? JSON.parse(stored) : null;
    } catch {
      return null;
    }
  }, []);

  const getSelectedDatasource = useCallback((): Datasource | null => {
    try {
      const stored = localStorage.getItem(STORAGE_KEYS.DATASOURCE);
      return stored ? JSON.parse(stored) : null;
    } catch {
      return null;
    }
  }, []);

  const setSelectedTenant = useCallback((tenant: Tenant) => {
    localStorage.setItem(STORAGE_KEYS.TENANT, JSON.stringify(tenant));
  }, []);

  const setSelectedProduct = useCallback((product: Product) => {
    localStorage.setItem(STORAGE_KEYS.PRODUCT, JSON.stringify(product));
  }, []);

  const setSelectedDatasource = useCallback((datasource: Datasource) => {
    localStorage.setItem(STORAGE_KEYS.DATASOURCE, JSON.stringify(datasource));
  }, []);

  const clearSelection = useCallback(() => {
    localStorage.removeItem(STORAGE_KEYS.TENANT);
    localStorage.removeItem(STORAGE_KEYS.PRODUCT);
    localStorage.removeItem(STORAGE_KEYS.DATASOURCE);
  }, []);

  const selectedTenant = getSelectedTenant();
  const selectedProduct = getSelectedProduct();
  const selectedDatasource = getSelectedDatasource();

  const hasValidScope = !!(selectedTenant && selectedDatasource);

  return {
    selectedTenant,
    selectedProduct,
    selectedDatasource,
    setSelectedTenant,
    setSelectedProduct,
    setSelectedDatasource,
    clearSelection,
    hasValidScope,
  };
};
