import React, { useMemo, useState, Suspense, lazy } from 'react';
import { Drawer, Box, IconButton, Typography, Divider } from '@mui/material';
import CloseIcon from '@mui/icons-material/Close';
import AddIcon from '@mui/icons-material/Add';
import type { Tenant, TenantInstance, Product, DataSource } from '../../../types';

const ProductGrid = lazy(() => import('../../../components/ProductGrid')) as unknown as React.ComponentType<any>;
const DatasourceGrid = lazy(() => import('../../../components/DatasourceGrid')) as unknown as React.ComponentType<any>;

interface InstanceProductsDrawerProps {
  open: boolean;
  tenant: Tenant | null;
  instance: TenantInstance | null;
  onClose: () => void;
  onAddProduct: (instance: TenantInstance) => void;
  onDeleteProduct: (productId: string) => void;
  onEditProduct: (product: Product) => void;
  onAddDatasource: (product: Product) => void;
  onEditDatasource: (datasource: DataSource) => void;
  onDeleteDatasource: (datasourceId: string) => void;
  onSelect: (tenant: Tenant, product: Product, datasource: DataSource) => void;
  onRunScanner: (datasource: DataSource) => void;
  telemetryHook?: (event: string, payload?: any) => void;
  // optional controlled selection from parent
  selectedProductId?: string | null;
  onSelectProduct?: (id: string | null) => void;
}

export const InstanceProductsDrawer: React.FC<InstanceProductsDrawerProps> = ({
  open,
  tenant,
  instance,
  onClose,
  onAddProduct,
  onDeleteProduct,
  onEditProduct,
  onAddDatasource,
  onEditDatasource,
  onDeleteDatasource,
  onSelect,
  onRunScanner,
  telemetryHook,
  selectedProductId,
  onSelectProduct,
}) => {
  // prefer controlled selection from parent when provided
  const [internalSelectedProductId, setInternalSelectedProductId] = useState<string | null>(null);
  const selectedProductIdEffective = selectedProductId !== undefined ? selectedProductId : internalSelectedProductId;
  const setSelectedProductIdEffective = (id: string | null) => {
    if (onSelectProduct) onSelectProduct(id);
    setInternalSelectedProductId(id);
  };

  const products = useMemo(() => (
    instance?.tenant_products ?? []
  ), [instance]);

  const selectedProduct = useMemo(() => (
    products.find((p: any) => p.id === selectedProductIdEffective) || null
  ), [products, selectedProductIdEffective]);
  

  return (
    <Drawer anchor="right" open={open} onClose={onClose} PaperProps={{ sx: { width: { xs: '100%', md: '60vw' } } }}>
      <Box sx={{ p: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography variant="h6" sx={{ flexGrow: 1 }}>
          {`Products for ${String(instance?.display_name || instance?.instance_name || instance?.id || '')}`}
          {tenant ? ` — Tenant: ${String(tenant.display_name || tenant.name || tenant.id)}` : ''}
        </Typography>
        <IconButton onClick={onClose}><CloseIcon /></IconButton>
      </Box>
      <Divider />
      <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: '1fr 1fr' }, gap: 2, p: 2, height: '100%', boxSizing: 'border-box' }}>
        <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
            <Typography variant="subtitle1">Assigned Products</Typography>
            {instance && <IconButton size="small" onClick={() => onAddProduct(instance)}><AddIcon fontSize="small" /></IconButton>}
          </Box>
          <Box sx={{ flex: 1, minHeight: 0 }}>
            <Suspense fallback={<div>Loading products...</div>}>
              <ProductGrid
                instance={instance as any}
                products={products}
                selectedProductId={selectedProductIdEffective}
                onSelectProduct={setSelectedProductIdEffective}
                onAddProduct={onAddProduct}
                onEditProduct={onEditProduct}
                onDeleteProduct={onDeleteProduct}
              />
            </Suspense>
          </Box>
        </Box>
        <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: 0 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
            <Typography variant="subtitle1">Data Sources</Typography>
            <IconButton size="small" onClick={() => selectedProduct && onAddDatasource(selectedProduct)} disabled={!selectedProduct}><AddIcon fontSize="small" /></IconButton>
          </Box>
          <Box sx={{ flex: 1, minHeight: 0 }}>
            {selectedProduct ? (
              <Suspense fallback={<div>Loading data sources...</div>}>
                <DatasourceGrid
                  tenant={tenant as any}
                  product={selectedProduct as any}
                  datasources={Array.isArray((selectedProduct as any).tenant_product_datasources) ? (selectedProduct as any).tenant_product_datasources : []}
                  onSelect={onSelect as any}
                  onRunScanner={onRunScanner}
                  onEditDatasource={onEditDatasource}
                  onDeleteDatasource={onDeleteDatasource}
                  telemetryHook={telemetryHook}
                />
              </Suspense>
            ) : (
              <Typography color="text.secondary">Select a product to see its data sources.</Typography>
            )}
          </Box>
        </Box>
      </Box>
    </Drawer>
  );
};

export default InstanceProductsDrawer;
