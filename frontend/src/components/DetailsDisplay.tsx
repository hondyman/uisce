// React default import removed — using automatic JSX runtime
import { lazy, Suspense, useState, useEffect } from 'react';
import { Box, Typography, IconButton, Card, CardContent, CardHeader, Divider, Chip, Grid, Tooltip } from '@mui/material';
import renderCoreCustomChips from './common/semanticChips';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import AddIcon from '@mui/icons-material/Add';
// Lazy-load grids into separate chunks so DataGrid is not included in this file's chunk
const ProductGrid = lazy(() => import('./ProductGrid')) as unknown as React.ComponentType<any>;
const DatasourceGrid = lazy(() => import('./DatasourceGrid')) as unknown as React.ComponentType<any>;
import { Tenant, TenantInstance, Product, DataSource } from '../types';
import { useTenant } from '../contexts/TenantContext';
import MonacoCodeEditor from './UnifiedSemanticBuilder/MonacoCodeEditor.lazy';

interface DetailsDisplayProps {
  tenant: Tenant | null;
  item: Tenant | TenantInstance | null;
  productViewInstance: TenantInstance | null;
  onEdit: (item: Tenant | TenantInstance | Product | DataSource) => void;
  onDelete: (item: Tenant | TenantInstance | Product | DataSource) => void;
  onAddProduct: (instance: TenantInstance) => void;
  onDeleteProduct: (productId: string) => void;
  onEditProduct: (product: Product) => void;
  onAddDatasource: (product: Product) => void;
  onEditDatasource: (datasource: DataSource) => void;
  onDeleteDatasource: (datasourceId: string) => void;
  onRunScanner: (datasource: DataSource) => void;
    onOpenDataCatalog?: (sourceName: string) => void;
    onTestConnection?: (datasource: DataSource) => void;
  onSelect: (tenant: Tenant, product: Product, datasource: DataSource) => void;
}

const isTenant = (item: any): item is Tenant => item && 'tenant_instances' in item;

const DetailRow: React.FC<{ label: string; value: React.ReactNode }> = (props) => {
        const { label, value } = props;
    return (
        <Box sx={{ display: 'flex', alignItems: 'baseline', mb: 1 }}>
            <Typography variant="body1" sx={{ fontWeight: 'bold', mr: 1, whiteSpace: 'nowrap' }}>{label}:</Typography>
            <Typography component="div" variant="body1" sx={{ overflowWrap: 'break-word', wordBreak: 'break-all', minWidth: 0 }}>
                {value}
            </Typography>
        </Box>
    );
};

const DetailsDisplay: React.FC<DetailsDisplayProps> = ({ 
  tenant, 
  item, 
  productViewInstance, 
  onEdit, 
  onDelete, 
  onAddProduct, 
  onDeleteProduct, 
  onEditProduct, 
  onAddDatasource, 
  onEditDatasource, 
  onDeleteDatasource, 
  onRunScanner, 
  onSelect 
}) => {
    const [selectedProduct, setSelectedProduct] = useState<Product | null>(null);
    const { setSelection } = useTenant();
    
    // --- THIS IS THE KEY FIX (Part 2) ---
    // This hook syncs the local state of this component with the fresh props
    // that it receives from the parent TenantsPage, ensuring the datasource grid updates.
    useEffect(() => {
        if (productViewInstance && selectedProduct) {
            const updatedSelectedProduct = productViewInstance.tenant_products?.find( p => p.id === selectedProduct.id );
            setSelectedProduct(updatedSelectedProduct || null);
        } else {
            // If the main item changes, clear the product selection
            setSelectedProduct(null);
        }
    }, [productViewInstance, selectedProduct]);

    if (!item || typeof item !== 'object') {
        return <Card sx={{ height: '100%', display: 'flex', justifyContent: 'center', alignItems: 'center' }}><CardContent><Typography color="text.secondary">Select an item to see details.</Typography></CardContent></Card>;
    }

        const renderTenantDetails = (tenant: Tenant) => (
            tenant && typeof tenant === 'object' ? <>
                <DetailRow label="ID" value={String(tenant.id)} />
                <DetailRow label="Display Name" value={String(tenant.display_name || tenant.name || 'Unnamed Tenant')} />
                <DetailRow label="Status" value={<Chip label={tenant.is_active ? 'Active' : 'Inactive'} color={tenant.is_active ? 'success' : 'default'} size="small" />} />
                <DetailRow label="Description" value={String(tenant.description || 'N/A')} />
            </> : null
        );

        const renderInstanceDetails = (instance: TenantInstance) => (
            instance && typeof instance === 'object' ? <>
                <DetailRow label="ID" value={String(instance.id)} />
                <DetailRow label="Display Name" value={String(instance.display_name || instance.instance_name || instance.id || 'Unnamed Instance')} />
                <DetailRow label="Status" value={<Chip label={instance.is_active ? 'Active' : 'Inactive'} color={instance.is_active ? 'success' : 'default'} size="small" />} />
                <DetailRow label="URL" value={String(instance.url || 'N/A')} />
                <DetailRow label="Description" value={String(instance.description || 'N/A')} />
                <Divider sx={{ my: 2 }} />
                <Typography variant="h6" gutterBottom>Configuration</Typography>
                <Box border={1} borderColor="grey.400" borderRadius={1} sx={{ flexGrow: 1, minHeight: '200px' }}>
                                                <Suspense fallback={<div>Loading editor...</div>}>
                                                                                                                <div className="editor-wrapper-full editor-h-400">
                                                                                                                    <MonacoCodeEditor
                                                                                                                        value={typeof instance.config === 'string' ? instance.config : JSON.stringify(instance.config || {}, null, 2)}
                                                                                                                        language="json"
                                                                                                                        readOnly
                                                                                                                    />
                                                                                                                </div>
                                                </Suspense>
                </Box>
            </> : null
        );
    
    const handleSelectDatasource = (tenant: Tenant, product: Product, datasource: DataSource) => {
        // Update the tenant context
        setSelection(tenant, product, datasource);
        
        // Call the original onSelect callback
        onSelect(tenant, product, datasource);
    };

    const renderProductDetails = (instance: TenantInstance) => {
        if (!instance || typeof instance !== 'object') return null;
        const products = Array.isArray(instance.tenant_products) ? instance.tenant_products : [];
        return (
            <Grid container spacing={3} sx={{ height: '100%' }}>
                <Grid item xs={12} md={6} sx={{ display: 'flex', flexDirection: 'column' }}>
                    <Card variant="outlined" sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
                        <CardHeader title="Assigned Products" action={<IconButton onClick={() => onAddProduct(instance)}><AddIcon /></IconButton>} />
                        <CardContent sx={{ flexGrow: 1, position: 'relative' }}>
                            <Box sx={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 }}>
                                <Suspense fallback={<div>Loading products...</div>}>
                                    <ProductGrid instance={instance} products={products} selectedProductId={selectedProduct ? selectedProduct.id : null} onSelectProduct={(id: string | null) => setSelectedProduct(products.find((p: any) => p.id === id) || null)} onAddProduct={onAddProduct} onEditProduct={onEditProduct} onDeleteProduct={onDeleteProduct} />
                                </Suspense>
                            </Box>
                        </CardContent>
                    </Card>
                </Grid>
                <Grid item xs={12} md={6} sx={{ display: 'flex', flexDirection: 'column' }}>
                    <Card variant="outlined" sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
                        <CardHeader title="Data Sources" action={ <IconButton onClick={() => onAddDatasource(selectedProduct!)} disabled={!selectedProduct}> <AddIcon /> </IconButton> } />
                        <CardContent sx={{ flexGrow: 1, position: 'relative' }}>
                            {selectedProduct ? (
                                <Box sx={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 }}>
                                    <Suspense fallback={<div>Loading data sources...</div>}>
                                        <DatasourceGrid tenant={tenant && typeof tenant === 'object' ? tenant : {}} product={selectedProduct} datasources={Array.isArray(selectedProduct.tenant_product_datasources) ? selectedProduct.tenant_product_datasources : []} onSelect={handleSelectDatasource} onRunScanner={onRunScanner} onEditDatasource={onEditDatasource} onDeleteDatasource={onDeleteDatasource} />
                                    </Suspense>
                                </Box>
                            ) : ( <Typography color="text.secondary">Select a product to see its data sources.</Typography> )}
                        </CardContent>
                    </Card>
                </Grid>
            </Grid>
        );
    };

    return (
        <Card elevation={3} sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
            <CardHeader
                title={
                    <Box sx={{ display: 'flex', alignItems: 'center' }}>
                        <Typography variant="h5">
                            {productViewInstance ? (
                                // When viewing products for an instance, show instance + tenant
                                `Products for ${String(productViewInstance.instance_name || productViewInstance.display_name || 'Instance')}` + (tenant ? ` — Tenant: ${String(tenant.display_name || tenant.name || tenant.id)}` : '')
                            ) : isTenant(item) ? (
                                // Tenant selected: show tenant display
                                String(item.display_name || item.name || 'Unnamed Tenant')
                            ) : (
                                // Instance selected: show instance and tenant (if available)
                                `${String((item as TenantInstance).display_name || (item as TenantInstance).instance_name || (item as TenantInstance).id || 'Unnamed Instance')}` + (tenant ? ` — Tenant: ${String(tenant.display_name || tenant.name || tenant.id)}` : '')
                            )}
                        </Typography>
                        {isTenant(item) && item.gold_copy && (
                            <Tooltip title="core — read-only"><Box component="span">{renderCoreCustomChips({ is_core: true })}</Box></Tooltip>
                        )}
                    </Box>
                }
                action={
                    !productViewInstance && (
                        <Box>
                            <IconButton onClick={() => onEdit(item!)}><EditIcon /></IconButton>
                            <IconButton onClick={() => onDelete(item!)}><DeleteIcon /></IconButton>
                        </Box>
                    )
                }
            />
            <Divider />
            <CardContent sx={{ flexGrow: 1, overflow: 'auto' }}>
                {productViewInstance
                    ? renderProductDetails(productViewInstance)
                    : (isTenant(item) ? renderTenantDetails(item as Tenant) : renderInstanceDetails(item as TenantInstance))}
            </CardContent>
        </Card>
    );
};

export default DetailsDisplay;