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
import { useQuery, useMutation, gql } from '@apollo/client';
import { GET_TENANT_REGISTERED_PRODUCTS } from '../../../graphql/queries/tenantQueries';
import { ADD_TENANT_PRODUCT, DELETE_TENANT_PRODUCT } from '../../../graphql/mutations/tenantMutations';
import { useNotification } from '../../../hooks/useNotification';

// Query to get all available products (from alpha_product table)
const GET_AVAILABLE_PRODUCTS = gql`
  query GetAvailableProducts {
    alpha_product(where: { is_active: { _eq: true } }) {
      id
      product_name
      product_code
      is_active
    }
  }
`;

// Query to get tenant details including gold_copy status
const GET_TENANT_INFO = gql`
  query GetTenantInfo($tenantId: uuid!) {
    tenants(where: { id: { _eq: $tenantId } }) {
      id
      display_name
      gold_copy
    }
  }
`;

interface AlphaProduct {
  id: string;
  product_name: string;
  product_code?: string;
  is_active: boolean;
}

interface ProductsTabContentProps {
  tenantId: string;
  datasourceId?: string;
  productCounts?: Map<string, { instances: Set<string>, connections: number }>;
  onProductInstancesClick?: (productId: string) => void;
  onProductConnectionsClick?: (productId: string) => void;
}

export const ProductsTabContent: React.FC<ProductsTabContentProps> = ({ 
  tenantId,
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

  // Fetch tenant info to check if it's gold copy
  const { data: tenantData } = useQuery(GET_TENANT_INFO, {
    variables: { tenantId },
    skip: !tenantId,
  });

  // Fetch available products from alpha_product table
  const { data: productsData, loading: productsLoading, error: productsError } = useQuery(GET_AVAILABLE_PRODUCTS);
  
  // Fetch tenant's registered products
  const { data: registeredData, loading: registeredLoading, refetch: refetchRegistered } = useQuery(
    GET_TENANT_REGISTERED_PRODUCTS,
    { variables: { tenantId }, skip: !tenantId }
  );

  const [registerProduct, { loading: registerLoading }] = useMutation(ADD_TENANT_PRODUCT, {
    refetchQueries: [{ query: GET_TENANT_REGISTERED_PRODUCTS, variables: { tenantId } }],
  });
  const [unregisterProduct, { loading: unregisterLoading }] = useMutation(DELETE_TENANT_PRODUCT, {
    refetchQueries: [{ query: GET_TENANT_REGISTERED_PRODUCTS, variables: { tenantId } }],
  });

  // Check if this is the gold copy tenant
  const isGoldCopy = tenantData?.tenants?.[0]?.gold_copy ?? false;

  // Get available products
  const availableProducts: AlphaProduct[] = useMemo(() => {
    return productsData?.alpha_product || [];
  }, [productsData]);

  // Get registered product IDs for this tenant
  const registeredProductIds = useMemo(() => {
    // For gold copy, all products are considered "registered"
    if (isGoldCopy) {
      return new Set(availableProducts.map(p => p.id));
    }
    const products = registeredData?.tenant_product || [];
    return new Set(products.map((p: any) => p.alpha_product_id));
  }, [registeredData, isGoldCopy, availableProducts]);

  // Get registered products map for looking up tenant_product.id
  const registeredProductsMap = useMemo(() => {
    const products = registeredData?.tenant_product || [];
    const map = new Map<string, any>();
    products.forEach((p: any) => {
      if (p.alpha_product_id) map.set(p.alpha_product_id, p);
    });
    return map;
  }, [registeredData]);

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
      await registerProduct({
        variables: {
          tenant_id: tenantId,
          alpha_product_id: product.id,
          version: 1.0,
          is_active: true,
        },
      });
      notification.success(`Registered "${product.product_name}" for this tenant`);
      // refetchRegistered is now handled by refetchQueries
    } catch (error: any) {
      notification.error(error.message || 'Failed to register product');
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
      await unregisterProduct({
        variables: { id: tenantProduct.id },
      });
      notification.success(`Unregistered "${product.product_name}" from this tenant`);
      // refetchRegistered is now handled by refetchQueries
    } catch (error: any) {
      notification.error(error.message || 'Failed to unregister product');
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

  const isRegistered = (productId: string) => registeredProductIds.has(productId);

  const handleSort = (property: string) => {
    const isAsc = sortBy === property && sortOrder === 'asc';
    setSortOrder(isAsc ? 'desc' : 'asc');
    setSortBy(property);
  };

  if (productsLoading || registeredLoading) {
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
            {isGoldCopy ? 'Product Catalog (Gold Copy)' : 'Product Registration'}
          </Typography>
          <Typography variant="body2" color="text.secondary">
            {isGoldCopy 
              ? 'This is the gold copy tenant. All products are available here. Toggle active/inactive to control availability for downstream tenants.'
              : 'Register products from the available catalog that this tenant will use. Registered products can then be linked to datasources.'
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
        <Card sx={{ flex: 1, bgcolor: 'grey.200' }}>
          <CardContent sx={{ py: 1.5 }}>
            <Typography variant="h4" sx={{ fontWeight: 700, color: 'text.primary' }}>
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
                    <Typography variant="body2" sx={{ fontWeight: 500 }}>
                      {product.product_name}
                    </Typography>
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
                  {!isGoldCopy && (
                    <TableCell align="right">
                      {registered ? (
                        <Tooltip title="Unregister this product from the tenant">
                          <IconButton
                            size="small"
                            color="error"
                            onClick={() => openConfirmDialog(product, 'unregister')}
                            disabled={unregisterLoading}
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
                    </TableCell>
                  )}
                </TableRow>
              );
            })}
            {filteredProducts.length === 0 && (
              <TableRow>
                <TableCell colSpan={isGoldCopy ? 3 : 4} align="center">
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
    </Box>
  );
};

export default ProductsTabContent;
