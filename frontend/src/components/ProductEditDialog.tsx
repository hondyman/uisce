import { useState, useEffect } from 'react';
import { Dialog, DialogActions, DialogContent, Button, TextField, Typography } from '@mui/material';
import ModalHeader from './ModalHeader';
import { Product } from '../types';

interface ProductEditDialogProps {
  open: boolean;
  product: Product | null;
  onClose: () => void;
  onSave: (productId: string, version: number) => void;
}

const ProductEditDialog: React.FC<ProductEditDialogProps> = ({ open, product, onClose, onSave }) => {
  const [version, setVersion] = useState<number | string>('');

  useEffect(() => {
    if (product) {
      setVersion(product.version || '');
    }
  }, [product]);

  const handleSave = () => {
    if (product && version !== '') {
      onSave(product.id, Number(version));
    }
  };

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
  <ModalHeader title="Edit Product Version" onClose={onClose} />
      <DialogContent>
        <Typography variant="h6" gutterBottom>
          {product?.alpha_product?.product_name}
        </Typography>
        <TextField
          autoFocus
          margin="dense"
          label="Version"
          type="number"
          fullWidth
          variant="standard"
          value={version}
          onChange={(e) => setVersion(e.target.value)}
          sx={{ mt: 2 }}
        />
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained">Save</Button>
      </DialogActions>
    </Dialog>
  );
};

export default ProductEditDialog;