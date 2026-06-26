// frontend/src/components/ScopeSelectorDialog.tsx
import React, { useState, useMemo } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  Typography,
  TextField,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Divider,
  Paper,
  IconButton,
  Breadcrumbs,
  Link,
  Chip,
  InputAdornment,
  Grid,
} from '@mui/material';
import {
  Search as SearchIcon,
  Business as BusinessIcon,
  Dns as InstanceIcon,
  Inventory as ProductIcon,
  Storage as DatasourceIcon,
  ChevronRight as ChevronRightIcon,
  Check as CheckIcon,
  ArrowBack as ArrowBackIcon,
  Public as PublicIcon,
} from '@mui/icons-material';
import { useAccess } from '../contexts/AccessContext';
import { Tenant, TenantInstance, Product, DataSource } from '../types';

interface ScopeSelectorDialogProps {
  open: boolean;
  onClose: () => void;
}

export const ScopeSelectorDialog: React.FC<ScopeSelectorDialogProps> = ({ open, onClose }) => {
  const {
    accessibleTenants,
    isPlatformOperator,
    scope,
    setGlobalScope,
    setTenantScope,
    setInstanceScope,
    setProductScope,
    setDatasourceScope,
    currentTenant,
    currentInstance,
    currentProduct,
    currentDatasource,
  } = useAccess();

  const [searchQuery, setSearchQuery] = useState('');
  const [selectedTenant, setSelectedTenant] = useState<Tenant | null>(currentTenant);
  const [selectedInstance, setSelectedInstance] = useState<TenantInstance | null>(currentInstance);
  const [selectedProduct, setSelectedProduct] = useState<Product | null>(currentProduct);

  // Filter tenants based on search
  const filteredTenants = useMemo(() => {
    if (!searchQuery.trim()) return accessibleTenants;
    const query = searchQuery.toLowerCase();
    return accessibleTenants.filter(t => 
      (t.display_name || t.name || '').toLowerCase().includes(query)
    );
  }, [accessibleTenants, searchQuery]);

  const handleSelectTenant = (tenant: Tenant) => {
    setSelectedTenant(tenant);
    setSelectedInstance(null);
    setSelectedProduct(null);
  };

  const handleSelectInstance = (instance: TenantInstance) => {
    setSelectedInstance(instance);
    setSelectedProduct(null);
  };

  const handleSelectProduct = (product: Product) => {
    setSelectedProduct(product);
  };

  const handleApplyScope = (datasource: DataSource) => {
    if (selectedTenant && selectedInstance && selectedProduct) {
      setDatasourceScope(selectedTenant, selectedInstance, selectedProduct, datasource);
      onClose();
    }
  };

  const handleClearSelection = () => {
    setSelectedTenant(null);
    setSelectedInstance(null);
    setSelectedProduct(null);
  };

  const handleBack = () => {
    if (selectedProduct) {
      setSelectedProduct(null);
    } else if (selectedInstance) {
      setSelectedInstance(null);
    } else if (selectedTenant) {
      setSelectedTenant(null);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth PaperProps={{ sx: { height: '80vh' } }}>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
          {(selectedTenant || selectedInstance || selectedProduct) && (
            <IconButton size="small" onClick={handleBack} sx={{ mr: 1 }}>
              <ArrowBackIcon />
            </IconButton>
          )}
          <Typography variant="h6">Select Operating Scope</Typography>
        </Box>
        <IconButton onClick={onClose} size="small">
          <ChevronRightIcon sx={{ transform: 'rotate(90deg)' }} />
        </IconButton>
      </DialogTitle>
      
      <DialogContent dividers sx={{ p: 0, display: 'flex', flexDirection: 'column' }}>
        {/* Navigation Breadcrumbs */}
        <Box sx={{ px: 3, py: 1.5, bgcolor: 'action.hover', borderBottom: 1, borderColor: 'divider' }}>
          <Breadcrumbs separator={<ChevronRightIcon fontSize="small" />}>
            <Link
              component="button"
              variant="body2"
              onClick={handleClearSelection}
              sx={{ color: !selectedTenant ? 'primary.main' : 'text.secondary', fontWeight: !selectedTenant ? 'bold' : 'normal', textDecoration: 'none' }}
            >
              Tenants
            </Link>
            {selectedTenant && (
              <Link
                component="button"
                variant="body2"
                onClick={() => { setSelectedInstance(null); setSelectedProduct(null); }}
                sx={{ color: selectedTenant && !selectedInstance ? 'primary.main' : 'text.secondary', fontWeight: selectedTenant && !selectedInstance ? 'bold' : 'normal', textDecoration: 'none' }}
              >
                {selectedTenant.display_name || selectedTenant.name}
              </Link>
            )}
            {selectedInstance && (
              <Link
                component="button"
                variant="body2"
                onClick={() => { setSelectedProduct(null); }}
                sx={{ color: selectedInstance && !selectedProduct ? 'primary.main' : 'text.secondary', fontWeight: selectedInstance && !selectedProduct ? 'bold' : 'normal', textDecoration: 'none' }}
              >
                {selectedInstance.display_name || selectedInstance.instance_name}
              </Link>
            )}
            {selectedProduct && (
              <Typography variant="body2" color="primary.main" sx={{ fontWeight: 'bold' }}>
                {selectedProduct.alpha_product?.product_name || 'Product'}
              </Typography>
            )}
          </Breadcrumbs>
        </Box>

        <Box sx={{ flex: 1, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
          {/* Search Bar - only show when selecting tenant */}
          {!selectedTenant && (
            <Box sx={{ p: 2 }}>
              <TextField
                fullWidth
                size="small"
                placeholder="Search tenants..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon color="action" />
                    </InputAdornment>
                  ),
                }}
              />
            </Box>
          )}

          <Box sx={{ flex: 1, overflow: 'auto' }}>
            {!selectedTenant ? (
              <List sx={{ pt: 0 }}>
                {isPlatformOperator && (
                  <ListItemButton 
                    onClick={() => { setGlobalScope(); onClose(); }}
                    sx={{ py: 2, borderBottom: 1, borderColor: 'divider' }}
                  >
                    <ListItemIcon>
                      <PublicIcon color="info" />
                    </ListItemIcon>
                    <ListItemText 
                      primary="All Tenants (Global View)" 
                      secondary="Platform-wide administration and monitoring"
                    />
                  </ListItemButton>
                )}
                {filteredTenants.map((tenant) => (
                  <ListItemButton 
                    key={tenant.id} 
                    onClick={() => handleSelectTenant(tenant)}
                    sx={{ py: 1.5, borderBottom: 1, borderColor: 'divider' }}
                  >
                    <ListItemIcon>
                      <BusinessIcon color={currentTenant?.id === tenant.id ? 'primary' : 'inherit'} />
                    </ListItemIcon>
                    <ListItemText 
                      primary={tenant.display_name || tenant.name} 
                      secondary={`${tenant.tenant_instances?.length || 0} instances available`}
                    />
                    <ChevronRightIcon color="action" />
                  </ListItemButton>
                ))}
              </List>
            ) : !selectedInstance ? (
              <List sx={{ pt: 0 }}>
                {selectedTenant.tenant_instances?.map((instance) => (
                  <ListItemButton 
                    key={instance.id} 
                    onClick={() => handleSelectInstance(instance)}
                    sx={{ py: 1.5, borderBottom: 1, borderColor: 'divider' }}
                  >
                    <ListItemIcon>
                      <InstanceIcon color={currentInstance?.id === instance.id ? 'primary' : 'inherit'} />
                    </ListItemIcon>
                    <ListItemText 
                      primary={instance.display_name || instance.instance_name} 
                      secondary={`${instance.tenant_products?.length || 0} products registered`}
                    />
                    <ChevronRightIcon color="action" />
                  </ListItemButton>
                ))}
              </List>
            ) : !selectedProduct ? (
              <List sx={{ pt: 0 }}>
                {selectedInstance.tenant_products?.map((product) => (
                  <ListItemButton 
                    key={product.id} 
                    onClick={() => handleSelectProduct(product)}
                    sx={{ py: 1.5, borderBottom: 1, borderColor: 'divider' }}
                  >
                    <ListItemIcon>
                      <ProductIcon color={currentProduct?.id === product.id ? 'primary' : 'inherit'} />
                    </ListItemIcon>
                    <ListItemText 
                      primary={product.alpha_product?.product_name || 'Unknown Product'} 
                      secondary={`${product.tenant_product_datasources?.length || 0} datasources available`}
                    />
                    <ChevronRightIcon color="action" />
                  </ListItemButton>
                ))}
              </List>
            ) : (
              <List sx={{ pt: 0 }}>
                {selectedProduct.tenant_product_datasources?.map((datasource) => (
                  <ListItemButton 
                    key={datasource.id} 
                    onClick={() => handleApplyScope(datasource)}
                    sx={{ py: 2, borderBottom: 1, borderColor: 'divider', bgcolor: currentDatasource?.id === datasource.id ? 'action.selected' : 'transparent' }}
                  >
                    <ListItemIcon>
                      <DatasourceIcon color={currentDatasource?.id === datasource.id ? 'success' : 'inherit'} />
                    </ListItemIcon>
                    <ListItemText 
                      primary={datasource.source_name} 
                      secondary={datasource.alpha_datasource?.datasource_type || 'Datasource'}
                      primaryTypographyProps={{ fontWeight: currentDatasource?.id === datasource.id ? 'bold' : 'normal' }}
                    />
                    {currentDatasource?.id === datasource.id && (
                      <CheckIcon color="success" />
                    )}
                  </ListItemButton>
                ))}
              </List>
            )}
          </Box>
        </Box>
      </DialogContent>
      
      <DialogActions sx={{ p: 2, justifyContent: 'space-between' }}>
        <Box>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
            Current Scope:
          </Typography>
          <Typography variant="body2" sx={{ fontWeight: 500 }}>
            {scope.isGlobal ? 'Global' : [scope.tenantName, scope.instanceName, scope.productName, scope.datasourceName].filter(Boolean).join(' » ') || 'None selected'}
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button onClick={onClose}>Cancel</Button>
          {!selectedTenant && isPlatformOperator && (
             <Button variant="contained" onClick={() => { setGlobalScope(); onClose(); }}>Set Global</Button>
          )}
        </Box>
      </DialogActions>
    </Dialog>
  );
};
