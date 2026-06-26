import React, { useState, useMemo } from 'react';
import {
  Box,
  Button,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  IconButton,
  TextField,
  Tooltip,
  Typography,
  Switch,
  FormControlLabel,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import ControlPointIcon from '@mui/icons-material/ControlPoint';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import VisibilityIcon from '@mui/icons-material/Visibility';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import { useMutation } from '@apollo/client';
import { useNotification } from '../../../hooks/useNotification';
import { ADD_TENANT_PRODUCT, DELETE_TENANT_PRODUCT, UPDATE_TENANT_PRODUCT } from '../../../graphql/mutations/tenantMutations';

interface TenantInstance {
  id: string;
  instance_name: string;
  display_name: string;
}

interface Product {
  id: string;
  product_name: string;
}

interface TenantProduct {
  id: string;
  alpha_product: Product;
  tenant_instance: TenantInstance;
}

interface ProductsTabProps {
  tenantId: string;
  instances: TenantInstance[];
  products: TenantProduct[];
  availableProducts: Product[];
  instanceFilter?: string | null;
  onRefetch: () => void;
}

export default function ProductsTab({ tenantId, instances, products, availableProducts, instanceFilter, onRefetch }: ProductsTabProps) {
  const notification = useNotification();
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [detailsDialogOpen, setDetailsDialogOpen] = useState(false);
  const [selectedProduct, setSelectedProduct] = useState<TenantProduct | null>(null);
  const [formData, setFormData] = useState({
    instance_id: '',
    product_id: '',
    version: 1.0,
    is_active: true,
  });
  const [editFormData, setEditFormData] = useState({
    id: '',
    version: 1.0,
    instance_id: '',
    is_active: true,
    current_product_id: '', // Used for filtering
    product_name: '',
  });

  const [addProduct] = useMutation(ADD_TENANT_PRODUCT);
  const [updateProduct] = useMutation(UPDATE_TENANT_PRODUCT);
  const [removeProduct] = useMutation(DELETE_TENANT_PRODUCT);

  // Filter available products for adding new assignment
  const filteredAvailableProducts = useMemo(() => {
    if (!formData.instance_id) return availableProducts;

    const assignedProductIds = new Set(
      products
        .filter(p => p.tenant_instance.id === formData.instance_id)
        .map(p => p.alpha_product.id)
    );

    return availableProducts.filter(p => !assignedProductIds.has(p.id));
  }, [availableProducts, products, formData.instance_id]);

  // Filter available instances for moving an existing assignment
  const availableInstancesForEdit = useMemo(() => {
    if (!editFormData.current_product_id) return instances;

    // Find all instances that ALREADY have this product assigned (excluding the current one we are editing)
    // We want to prevent moving a product to an instance that already has it.
    const instancesWithThisProduct = new Set(
      products
        .filter(p => 
          p.alpha_product.id === editFormData.current_product_id && 
          p.tenant_instance.id !== editFormData.instance_id // Don't block staying on current instance
        )
        .map(p => p.tenant_instance.id)
    );

    return instances.filter(i => !instancesWithThisProduct.has(i.id));
  }, [instances, products, editFormData.current_product_id, editFormData.instance_id]);

  // Filter products based on instance filter
  const filteredProducts = useMemo(() => {
    if (!instanceFilter) return products;
    return products.filter(p => p.tenant_instance.id === instanceFilter);
  }, [products, instanceFilter]);

  const handleOpenDialog = () => {
    setFormData({ instance_id: '', product_id: '', version: 1.0, is_active: true });
    setDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDialogOpen(false);
  };

  const handleSave = async () => {
    try {
      if (!formData.product_id) {
        notification.error('Please select a product');
        return;
      }
      await addProduct({
        variables: {
          tenant_instance_id: formData.instance_id,
          alpha_product_id: formData.product_id,
          version: formData.version,
          is_active: formData.is_active,
        },
      });
      notification.success('Product added to instance successfully');
      handleCloseDialog();
      onRefetch();
    } catch (error: any) {
      notification.error(error.message || 'Failed to add product');
    }
  };

  const handleOpenEditDialog = (product: TenantProduct & { version: number }) => {
    setEditFormData({
      id: product.id,
      version: product.version || 1.0,
      instance_id: product.tenant_instance.id,
      current_product_id: product.alpha_product.id,
      product_name: product.alpha_product.product_name,
      is_active: product.is_active,
    });
    setEditDialogOpen(true);
  };

  const handleEditSave = async () => {
    try {
      await updateProduct({
        variables: {
          id: editFormData.id,
          version: parseFloat(String(editFormData.version)),
          tenant_instance_id: editFormData.instance_id,
          is_active: editFormData.is_active,
        },
      });
      notification.success('Product updated successfully');
      setEditDialogOpen(false);
      onRefetch();
    } catch (error: any) {
      notification.error(error.message || 'Failed to update product');
    }
  };

  const handleViewDetails = (product: TenantProduct) => {
    setSelectedProduct(product);
    setDetailsDialogOpen(true);
  };

  const handleCloseDetailsDialog = () => {
    setDetailsDialogOpen(false);
    setSelectedProduct(null);
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Are you sure you want to remove this product from the instance?')) return;
    
    try {
      await removeProduct({ variables: { id } });
      notification.success('Product removed successfully');
      onRefetch();
    } catch (error: any) {
      notification.error(error.message || 'Failed to remove product');
    }
  };

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 2 }}>
        <Tooltip title="Add Product to Instance">
          <IconButton
            color="primary"
            onClick={handleOpenDialog}
            sx={{ fontSize: '2rem' }}
          >
            <ControlPointIcon sx={{ fontSize: '2rem' }} />
          </IconButton>
        </Tooltip>
      </Box>

      <Table>
        <TableHead>
          <TableRow>
            <TableCell>Product</TableCell>
            <TableCell>Instance</TableCell>
            <TableCell>Version</TableCell>
            <TableCell>Active</TableCell>
            <TableCell align="right">Actions</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {filteredProducts.map((product) => (
            <TableRow key={product.id}>
              <TableCell>{product.alpha_product?.product_name || 'Unknown'}</TableCell>
              <TableCell>{product.tenant_instance?.display_name || product.tenant_instance?.instance_name || 'Unknown'}</TableCell>
              <TableCell>{(product as any).version || 1.0}</TableCell>
              <TableCell>
                {(product as any).is_active ? (
                  <CheckCircleIcon sx={{ color: '#4caf50', fontSize: '20px' }} />
                ) : (
                  <CancelIcon sx={{ color: '#f44336', fontSize: '20px' }} />
                )}
              </TableCell>
              <TableCell align="right">
                <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1 }}>
                  <IconButton size="small" onClick={() => handleViewDetails(product)} color="primary" title="View details">
                    <VisibilityIcon fontSize="small" />
                  </IconButton>
                  <IconButton size="small" onClick={() => handleOpenEditDialog(product as any)} color="primary" title="Edit">
                    <EditIcon fontSize="small" />
                  </IconButton>
                  <IconButton size="small" onClick={() => handleDelete(product.id)} color="error" title="Delete">
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Box>
              </TableCell>
            </TableRow>
          ))}
          {filteredProducts.length === 0 && (
            <TableRow>
              <TableCell colSpan={5} align="center">
                No products assigned. Click "Add Product to Instance" to assign one.
              </TableCell>
            </TableRow>
          )}
        </TableBody>
      </Table>

      <Dialog open={dialogOpen} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>Add Product to Instance</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <FormControl fullWidth required>
              <InputLabel>Instance</InputLabel>
              <Select
                value={formData.instance_id}
                onChange={(e) => setFormData({ ...formData, instance_id: e.target.value, product_id: '' })}
                label="Instance"
              >
                {instances.map((instance) => (
                  <MenuItem key={instance.id} value={instance.id}>
                    {instance.display_name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControl fullWidth required>
              <InputLabel>Product</InputLabel>
              <Select
                value={formData.product_id}
                onChange={(e) => setFormData({ ...formData, product_id: e.target.value })}
                label="Product"
              >
                {filteredAvailableProducts.map((product) => (
                  <MenuItem key={product.id} value={product.id}>
                    {product.product_name}
                  </MenuItem>
                ))}
              </Select>
              {formData.instance_id && filteredAvailableProducts.length === 0 && (
                <Box sx={{ mt: 1, px: 2, color: 'text.secondary', fontSize: '0.75rem' }}>
                  All available products are already assigned to this instance.
                </Box>
              )}
            </FormControl>
            <FormControl fullWidth required>
              <TextField
                label="Version"
                type="number"
                value={formData.version}
                onChange={(e) => setFormData({ ...formData, version: parseFloat(e.target.value) })}
                inputProps={{ step: "0.1" }}
              />
            </FormControl>
            <FormControlLabel
              control={
                <Switch
                  checked={formData.is_active}
                  onChange={(e) =>
                    setFormData({ ...formData, is_active: e.target.checked })
                  }
                />
              }
              label="Active"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button onClick={handleSave} variant="contained" disabled={!formData.instance_id || !formData.product_id}>
            Add
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog open={editDialogOpen} onClose={() => setEditDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Edit {editFormData.product_name}</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <FormControl fullWidth required>
              <InputLabel>Instance</InputLabel>
              <Select
                value={editFormData.instance_id}
                onChange={(e) => setEditFormData({ ...editFormData, instance_id: e.target.value })}
                label="Instance"
              >
                {availableInstancesForEdit.map((instance) => (
                  <MenuItem key={instance.id} value={instance.id}>
                    {instance.display_name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <FormControl fullWidth required>
              <TextField
                label="Version"
                type="number"
                value={editFormData.version}
                onChange={(e) => setEditFormData({ ...editFormData, version: parseFloat(e.target.value) })}
                inputProps={{ step: "0.1" }}
              />
            </FormControl>
            <FormControlLabel
              control={
                <Switch
                  checked={editFormData.is_active}
                  onChange={(e) =>
                    setEditFormData({ ...editFormData, is_active: e.target.checked })
                  }
                />
              }
              label="Active"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleEditSave} variant="contained">
            Save
          </Button>
        </DialogActions>
      </Dialog>

      {/* Details Dialog */}
      <Dialog open={detailsDialogOpen} onClose={handleCloseDetailsDialog} maxWidth="sm" fullWidth>
        <DialogTitle>Product Details</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 2 }}>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Product</Typography>
              <Typography variant="body2">{selectedProduct?.alpha_product?.product_name || '—'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Instance</Typography>
              <Typography variant="body2">{selectedProduct?.tenant_instance?.display_name || selectedProduct?.tenant_instance?.instance_name || '—'}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Version</Typography>
              <Typography variant="body2">{(selectedProduct as any)?.version || 1.0}</Typography>
            </Box>
            <Box>
              <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>Active</Typography>
              <Typography variant="body2">
                {(selectedProduct as any)?.is_active ? (
                  <span style={{ color: '#4caf50' }}>✓ Active</span>
                ) : (
                  <span style={{ color: '#f44336' }}>✗ Inactive</span>
                )}
              </Typography>
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDetailsDialog}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
