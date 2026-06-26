import { useState, useEffect } from 'react';
import { useQuery } from '@apollo/client';
import { Dialog, DialogActions, DialogContent, Button, FormControl, InputLabel, Select, MenuItem, CircularProgress, Alert, TextField } from '@mui/material';
import ModalHeader from './ModalHeader';
import { GET_ALL_PRODUCTS } from '../graphql/queries/productQueries';
import { TenantInstance } from '../types';

interface ProductDialogProps {
  open: boolean;
  instance: TenantInstance | null;
  onClose: () => void;
  onSave: (productId: string, version: number) => void;
}

const ProductDialog: React.FC<ProductDialogProps> = ({ open, instance, onClose, onSave }) => {
  const { loading, error, data } = useQuery(GET_ALL_PRODUCTS);
  const [selectedProductId, setSelectedProductId] = useState('');
  const [version, setVersion] = useState<number | string>(1.0);

  useEffect(() => {
    if (open) {
      setSelectedProductId('');
      setVersion(1.0);
    }
  }, [open]);

  const handleSave = () => {
    if (selectedProductId && version !== '') {
      onSave(selectedProductId, Number(version));
    }
  };

  const assignedProductIds = instance?.tenant_products?.map(p => p.alpha_product?.id) || [];
  const availableProducts = data?.alpha_product.filter((p: { id: string }) => !assignedProductIds.includes(p.id)) || [];

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
  <ModalHeader title={`Assign Product to ${String(instance?.display_name || instance?.instance_name || instance?.id || 'Unnamed Instance')}`} onClose={onClose} />
      <DialogContent>
        {loading && <CircularProgress />}
        {error && <Alert severity="error">Could not load products.</Alert>}
        {data && (
          <>
            <FormControl fullWidth margin="dense" sx={{ mt: 2 }}>
              <InputLabel id="product-select-label">Available Products</InputLabel>
              <Select
                labelId="product-select-label"
                value={selectedProductId}
                label="Available Products"
                onChange={(e) => setSelectedProductId(e.target.value)}
              >
                {availableProducts.length === 0 && <MenuItem disabled>No more products to assign</MenuItem>}
                {availableProducts.map((product: { id: string, product_name: string }) => (
                  <MenuItem key={product.id} value={product.id}>
                    {product.product_name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            <TextField
              margin="dense"
              label="Version"
              type="number"
              fullWidth
              variant="standard"
              value={version}
              onChange={(e) => setVersion(e.target.value)}
            />
          </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={!selectedProductId || version === ''}>Assign</Button>
      </DialogActions>
    </Dialog>
  );
};

export default ProductDialog;