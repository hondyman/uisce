import React, { useState, useMemo } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Checkbox,
  Button,
  Chip,
  CircularProgress,
  Alert,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  InputAdornment,
  FormControlLabel,
  Switch,
  TableSortLabel,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import AddIcon from '@mui/icons-material/Add';
import RemoveIcon from '@mui/icons-material/Remove';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import InfoIcon from '@mui/icons-material/Info';
import EditIcon from '@mui/icons-material/Edit';
import { useNotification } from '../../../hooks/useNotification';
import { apiClient } from '../../../utils/apiClient';
import { useApiQuery } from '../../../hooks/useApiQuery';

interface AlphaProduct {
  id: string;
  product_name: string;
  product_code?: string;
  is_active: boolean;
  is_registered?: boolean;
  registered_at?: string | null;
  registered_is_active?: boolean;
}

interface ProductsTabContentProps {
  tenantId: string;
  activeInstanceId?: string;
  datasourceId?: string;
  productCounts?: Map<string, { instances: Set<string>, connections: number }>;
  onProductInstancesClick?: (productId: string) => void;
  onProductConnectionsClick?: (productId: string) => void;
}

export const ProductsTabContent: React.FC<ProductsTabContentProps> = ({ 
  tenantId,
  activeInstanceId,
  productCounts = new Map(),
  onProductInstancesClick,
  onProductConnectionsClick
}) => {
  const notification = useNotification();
  const [searchQuery, setSearchQuery] = useState('');
  const [sortBy, setSortBy] = useState('name');
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('asc');
  const [confirmDialogOpen, setConfirmDialogOpen] = useState(false);
  const [selectedForAction, setSelectedForAction] = useState<{ product: any; action: 'register' | 'unregister' } | null>(null);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [editProduct, setEditProduct] = useState<{ id: string; product_name: string; is_active: boolean } | null>(null);
  const [editLoading, setEditLoading] = useState(false);

  // Fetch tenant data for gold_copy status and registered products
  const { data: tenantData, refetch: refetchTenant } = useApiQuery<{ tenant: any }>(
    tenantId ? `/api/tenants/${tenantId}` : ''
  );

  // Fetch available products from alpha_product table
  const { data: productsData, loading: productsLoading, error: productsError, refetch: refetchProducts } = useApiQuery<{ alpha_product: AlphaProduct[] }>(
    tenantId ? `/api/rest/products?tenant_id=${tenantId}` : '/api/rest/products'
  );

  const [registerLoading, setRegisterLoading] = useState(false);
  const [unregisterLoading, setUnregisterLoading] = useState(false);

  // Check if this is the gold copy tenant
  const isGoldCopy = tenantData?.tenant?.gold_copy ?? false;

  // Get available products
  const availableProducts: AlphaProduct[] = useMemo(() => {
    return productsData?.alpha_product || [];
  }, [productsData]);

  // Extract registered products from nested tenant instances/products structure
  const registeredProducts = useMemo(() => {
    if (!tenantData?.tenant) return [];
    const products: any[] = [];
    tenantData.tenant.tenant_instances?.forEach((instance: any) => {
      if (activeInstanceId && instance.id !== activeInstanceId) return;
      (instance.products || instance.tenant_products)?.forEach((p: any) => {
        products.push(p);
      });
    });
    return products;
  }, [tenantData, activeInstanceId]);

  const instanceName = useMemo(() => {
    if (!activeInstanceId || !tenantData?.tenant?.tenant_instances) return null;
    const inst = tenantData.tenant.tenant_instances.find((i: any) => i.id === activeInstanceId);
    return inst ? (inst.display_name || inst.instance_name) : null;
  }, [activeInstanceId, tenantData]);

  // Get registered product IDs for this tenant
  const registeredProductIds = useMemo(() => {
    // For gold copy, all products are considered "registered"
    if (isGoldCopy) {
      return new Set(availableProducts.map(p => p.id));
    }
    return new Set(registeredProducts.map((p: any) => p.alpha_product_id));
  }, [registeredProducts, isGoldCopy, availableProducts]);

  // Get registered products map for looking up tenant_product.id
  const registeredProductsMap = useMemo(() => {
    const map = new Map<string, any>();
    registeredProducts.forEach((p: any) => {
      if (p.alpha_product_id) map.set(p.alpha_product_id, p);
    });
    return map;
  }, [registeredProducts]);

  // Filter products by search query
  const filteredProducts = useMemo(() => {
    const query = searchQuery.toLowerCase();
    const filtered = availableProducts.filter(p => 
      p.product_name?.toLowerCase().includes(query)
    );

    return filtered.sort((a, b) => {
      let aValue: any = '';
      let bValue: any = '';

      switch (sortBy) {
        case 'name':
          aValue = a.product_name?.toLowerCase() || '';
          bValue = b.product_name?.toLowerCase() || '';
          break;
        case 'status':
          aValue = a.is_active ? 1 : 0;
          bValue = b.is_active ? 1 : 0;
          break;
        case 'registered':
          aValue = isRegistered(a.id) ? 1 : 0;
          bValue = isRegistered(b.id) ? 1 : 0;
          break;
        default:
          return 0;
      }

      if (aValue < bValue) return sortOrder === 'asc' ? -1 : 1;
      if (aValue > bValue) return sortOrder === 'asc' ? 1 : -1;
      return 0;
    });
  }, [availableProducts, searchQuery, sortBy, sortOrder, registeredProductIds]); // Added dependencies

  const handleRegisterProduct = async (product: AlphaProduct) => {
    try {
      setRegisterLoading(true);
      await apiClient(`/api/tenant-ops/products`, {
        method: 'POST',
        body: JSON.stringify({
          alpha_product_id: product.id,
          tenant_instance_id: activeInstanceId,
          version: 1.0,
          is_active: true,
        }),
      });
      notification.success(`Registered "${product.product_name}" for this tenant instance`);
      refetchTenant();
    } catch (error: any) {
      notification.error(error.message || 'Failed to register product');
    } finally {
      setRegisterLoading(false);
    }
    setConfirmDialogOpen(false);
    setSelectedForAction(null);
  };

  const handleUnregisterProduct = async (product: AlphaProduct) => {
    const tenantProduct = registeredProductsMap.get(product.id);
    if (!tenantProduct) {
      notification.error('Product registration not found');
      return;
    }

    try {
      setUnregisterLoading(true);
      await apiClient(`/api/tenant-ops/products/${tenantProduct.id}`, {
        method: 'DELETE',
      });
      notification.success(`Unregistered "${product.product_name}" from this tenant`);
      refetchTenant();
    } catch (error: any) {
      notification.error(error.message || 'Failed to unregister product');
    } finally {
      setUnregisterLoading(false);
    }
    setConfirmDialogOpen(false);
    setSelectedForAction(null);
  };

  const openConfirmDialog = (product: AlphaProduct, action: 'register' | 'unregister') => {
    setSelectedForAction({ product, action });
    setConfirmDialogOpen(true);
  };

  const handleConfirm = () => {
    if (!selectedForAction) return;
    if (selectedForAction.action === 'register') {
      handleRegisterProduct(selectedForAction.product);
    } else {
      handleUnregisterProduct(selectedForAction.product);
    }
  };

  const handleOpenEditDialog = (product: AlphaProduct) => {
    setEditProduct({ id: product.id, product_name: product.product_name, is_active: product.is_active });
    setEditDialogOpen(true);
  };

  const handleEditSave = async () => {
    if (!editProduct) return;
    try {
      setEditLoading(true);
      await apiClient(`/api/rest/products/${editProduct.id}`, {
        method: 'PATCH',
        body: JSON.stringify({
          product_name: editProduct.product_name,
          is_active: editProduct.is_active,
        }),
      });
      notification.success('Product updated successfully');
      setEditDialogOpen(false);
      setEditProduct(null);
      refetchTenant();
      refetchProducts();
    } catch (error: any) {
      notification.error(error.message || 'Failed to update product');
    } finally {
      setEditLoading(false);
    }
  };

  const isRegistered = (productId: string) => registeredProductIds.has(productId);

  const handleSort = (property: string) => {
    const isAsc = sortBy === property && sortOrder === 'asc';
    setSortOrder(isAsc ? 'desc' : 'asc');
    setSortBy(property);
  };

  if (productsLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: 200 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (productsError) {
    return <Alert severity="error">Failed to load products: {productsError.message}</Alert>;
  }

  if (availableProducts.length === 0) {
    return (
      <Alert severity="info">
        No products available. Please ensure products are configured in the alpha_product table.
      </Alert>
    );
  }

  return (
    <Box>
      {/* Header with search */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h6" sx={{ fontWeight: 600 }}>
            {isGoldCopy ? 'Product Catalog (Gold Copy)' : (instanceName ? `Product Registration for: ${instanceName}` : 'Product Registration')}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {isGoldCopy 
              ? 'This is the gold copy tenant. All products are available here. Toggle active/inactive to control availability for downstream tenants.'
              : (instanceName 
                ? `Register products from the available catalog that the scoped instance "${instanceName}" will use. Registered products can then be linked to datasources.` 
                : 'Register products from the available catalog that this tenant will use. Registered products can then be linked to datasources.')
            }
          </Typography>
        </Box>
        <TextField
          size="small"
          placeholder="Search products..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
          sx={{ minWidth: 250 }}
        />
      </Box>

      {/* Stats cards */}
      <Box sx={{ display: 'flex', gap: 2, mb: 3 }}>
        <Card sx={{ flex: 1, bgcolor: 'primary.light', color: 'primary.contrastText' }}>
          <CardContent sx={{ py: 1.5 }}>
            <Typography variant="h4" sx={{ fontWeight: 700 }}>
              {availableProducts.length}
            </Typography>
            <Typography variant="body2">Available Products</Typography>
          </CardContent>
        </Card>
        <Card sx={{ flex: 1, bgcolor: 'success.light', color: 'success.contrastText' }}>
          <CardContent sx={{ py: 1.5 }}>
            <Typography variant="h4" sx={{ fontWeight: 700 }}>
              {registeredProductIds.size}
            </Typography>
            <Typography variant="body2">Registered</Typography>
          </CardContent>
        </Card>
        <Card
          sx={{
            flex: 1,
            background: (theme) =>
              theme.palette.mode === 'dark'
                ? `rgba(7, 21, 38, 0.6)`
                : 'rgba(255, 255, 255, 0.8)',
            backdropFilter: 'blur(12px)',
            border: (theme) =>
              theme.palette.mode === 'dark'
                ? '1px solid rgba(0, 201, 200, 0.3)'
                : '1px solid rgba(0, 201, 200, 0.2)',
          }}
        >
          <CardContent sx={{ py: 1.5 }}>
            <Typography
              variant="h4"
              sx={{ fontWeight: 700, color: 'text.primary' }}
            >
              {availableProducts.length - registeredProductIds.size}
            </Typography>
            <Typography variant="body2" color="text.secondary">Available to Register</Typography>
          </CardContent>
        </Card>
      </Box>

      {/* Products table */}
      <Card>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>
                <TableSortLabel
                  active={sortBy === 'name'}
                  direction={sortBy === 'name' ? sortOrder : 'asc'}
                  onClick={() => handleSort('name')}
                >
                  Product Name
                </TableSortLabel>
              </TableCell>
              <TableCell>
                <TableSortLabel
                  active={sortBy === 'status'}
                  direction={sortBy === 'status' ? sortOrder : 'asc'}
                  onClick={() => handleSort('status')}
                >
                  Status
                </TableSortLabel>
              </TableCell>
              <TableCell>
                <TableSortLabel
                  active={sortBy === 'registered'}
                  direction={sortBy === 'registered' ? sortOrder : 'asc'}
                  onClick={() => handleSort('registered')}
                >
                  Registered
                </TableSortLabel>
              </TableCell>
              <TableCell align="center">Usage</TableCell>
              {!isGoldCopy && <TableCell align="right">Actions</TableCell>}
            </TableRow>
          </TableHead>
          <TableBody>
            {filteredProducts.map((product) => {
              const registered = isRegistered(product.id);
              const counts = productCounts.get(product.id);
              const instanceCount = counts?.instances.size || 0;
              const connectionCount = counts?.connections || 0;
              return (
                <TableRow key={product.id} hover>
                  <TableCell>
                    <Box
                      onClick={() => handleOpenEditDialog(product)}
                      sx={{ cursor: 'pointer', '&:hover': { color: 'primary.main' } }}
                    >
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {product.product_name}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={product.is_active ? 'Active' : 'Inactive'}
                      size="small"
                      color={product.is_active ? 'success' : 'default'}
                    />
                  </TableCell>
                  <TableCell>
                    {registered ? (
                      <Box sx={{ display: 'flex', gap: 1 }}>
                        <Chip
                          icon={<CheckCircleIcon />}
                          label={isGoldCopy ? "Available" : "Registered"}
                          size="small"
                          color="primary"
                          variant="filled"
                        />
                        {/* Show CORE badge if product has a core_id */}
                        {registeredProductsMap.get(product.id)?.core_id && (
                          <Chip
                            label="CORE"
                            size="small"
                            color="info"
                            title="Cloned from Gold Copy"
                            sx={{ fontWeight: 'bold' }}
                          />
                        )}
                      </Box>
                    ) : (
                      <Chip
                        label="Not Registered"
                        size="small"
                        variant="outlined"
                      />
                    )}
                  </TableCell>
                  <TableCell align="center">
                    <Box sx={{ display: 'flex', gap: 1, justifyContent: 'center', alignItems: 'center' }}>
                      {instanceCount > 0 && (
                        <Tooltip title={`Click to view ${instanceCount} instance${instanceCount !== 1 ? 's' : ''}`}>
                          <Chip
                            label={`${instanceCount} Instance${instanceCount !== 1 ? 's' : ''}`}
                            size="small"
                            variant="outlined"
                            onClick={() => onProductInstancesClick?.(product.id)}
                            sx={{ cursor: 'pointer', '&:hover': { bgcolor: 'action.hover' } }}
                          />
                        </Tooltip>
                      )}
                      {connectionCount > 0 && (
                        <Tooltip title={`Click to view ${connectionCount} connection${connectionCount !== 1 ? 's' : ''}`}>
                          <Chip
                            label={`${connectionCount} Connection${connectionCount !== 1 ? 's' : ''}`}
                            size="small"
                            variant="outlined"
                            color="info"
                            onClick={() => onProductConnectionsClick?.(product.id)}
                            sx={{ cursor: 'pointer', '&:hover': { bgcolor: 'info.light' } }}
                          />
                        </Tooltip>
                      )}
                      {instanceCount === 0 && connectionCount === 0 && (
                        <Typography variant="caption" color="text.secondary">—</Typography>
                      )}
                    </Box>
                  </TableCell>
                  <TableCell align="right">
                    <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 0.5 }}>
                      <Tooltip title="Edit product">
                        <IconButton
                          size="small"
                          color="primary"
                          onClick={() => handleOpenEditDialog(product)}
                        >
                          <EditIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                      {isGoldCopy ? (
                        <Tooltip title="Products cannot be registered at Gold Copy level - they are inherited from master">
                          <span>
                            <IconButton size="small" disabled>
                              <AddIcon />
                            </IconButton>
                          </span>
                        </Tooltip>
                      ) : registered ? (
                        <Tooltip title="Products cannot be unregistered - contact administrator">
                          <IconButton
                            size="small"
                            color="error"
                            disabled
                          >
                            <RemoveIcon />
                          </IconButton>
                        </Tooltip>
                      ) : (
                        <Tooltip title="Register this product for the tenant">
                          <IconButton
                            size="small"
                            color="primary"
                            onClick={() => openConfirmDialog(product, 'register')}
                            disabled={registerLoading}
                          >
                            <AddIcon />
                          </IconButton>
                        </Tooltip>
                      )}
                    </Box>
                  </TableCell>
                </TableRow>
              );
            })}
            {filteredProducts.length === 0 && (
              <TableRow>
                <TableCell colSpan={4} align="center">
                  {searchQuery ? 'No products match your search.' : 'No products available.'}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </Card>

      {/* Confirmation Dialog */}
      <Dialog open={confirmDialogOpen} onClose={() => setConfirmDialogOpen(false)}>
        <DialogTitle>
          {selectedForAction?.action === 'register' ? 'Register Product' : 'Unregister Product'}
        </DialogTitle>
        <DialogContent>
          <Typography>
            {selectedForAction?.action === 'register' ? (
              <>
                Are you sure you want to register <strong>{selectedForAction?.product?.product_name}</strong> for this tenant?
                Once registered, you can create datasources linked to this product.
              </>
            ) : (
              <>
                Are you sure you want to unregister <strong>{selectedForAction?.product?.product_name}</strong>?
                This may affect existing datasources linked to this product.
              </>
            )}
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirmDialogOpen(false)}>Cancel</Button>
          <Button
            variant="contained"
            color={selectedForAction?.action === 'register' ? 'primary' : 'error'}
            onClick={handleConfirm}
            disabled={registerLoading || unregisterLoading}
          >
            {selectedForAction?.action === 'register' ? 'Register' : 'Unregister'}
          </Button>
        </DialogActions>
      </Dialog>

      {/* Edit Product Dialog */}
      <Dialog open={editDialogOpen} onClose={() => setEditDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit Product</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              label="Product Name"
              value={editProduct?.product_name || ''}
              onChange={(e) => setEditProduct((prev) => prev ? { ...prev, product_name: e.target.value } : null)}
              fullWidth
            />
            <FormControlLabel
              control={
                <Switch
                  checked={editProduct?.is_active ?? false}
                  onChange={(e) => setEditProduct((prev) => prev ? { ...prev, is_active: e.target.checked } : null)}
                />
              }
              label="Active"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditDialogOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleEditSave} disabled={editLoading}>
            Save
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default ProductsTabContent;
