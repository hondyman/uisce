import React from 'react';
import {
  Box,
  Typography,
  FormControl,
  FormLabel,
  FormGroup,
  FormControlLabel,
  Checkbox,
  Divider,
  Paper,
  Button,
} from '@mui/material';
import { Clear as ClearIcon } from '@mui/icons-material';

interface ConnectionsFacetsProps {
  instances: Array<{ id: string; display_name: string; instance_name: string }>;
  products: Array<{ id: string; product_name: string }>;
  selectedInstances: string[];
  selectedProducts: string[];
  onInstanceChange: (instanceIds: string[]) => void;
  onProductChange: (productIds: string[]) => void;
}

export const ConnectionsFacets: React.FC<ConnectionsFacetsProps> = ({
  instances,
  products,
  selectedInstances,
  selectedProducts,
  onInstanceChange,
  onProductChange,
}) => {
  const handleInstanceToggle = (instanceId: string) => {
    const newSelection = selectedInstances.includes(instanceId)
      ? selectedInstances.filter(id => id !== instanceId)
      : [...selectedInstances, instanceId];
    onInstanceChange(newSelection);
  };

  const handleProductToggle = (productId: string) => {
    const newSelection = selectedProducts.includes(productId)
      ? selectedProducts.filter(id => id !== productId)
      : [...selectedProducts, productId];
    onProductChange(newSelection);
  };

  const handleClearAll = () => {
    onInstanceChange([]);
    onProductChange([]);
  };

  const hasActiveFilters = selectedInstances.length > 0 || selectedProducts.length > 0;

  return (
    <Paper variant="outlined" sx={{ p: 2, height: 'fit-content', minWidth: 250 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
        <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
          Filters
        </Typography>
        {hasActiveFilters && (
          <Button
            size="small"
            startIcon={<ClearIcon />}
            onClick={handleClearAll}
            sx={{ textTransform: 'none' }}
          >
            Clear
          </Button>
        )}
      </Box>

      {/* Instance Facets */}
      <FormControl component="fieldset" fullWidth sx={{ mb: 3 }}>
        <FormLabel component="legend" sx={{ mb: 1, fontWeight: 600, fontSize: '0.875rem' }}>
          Instance
        </FormLabel>
        <FormGroup>
          {instances.length === 0 ? (
            <Typography variant="caption" color="text.secondary">
              No instances available
            </Typography>
          ) : (
            instances.map((instance) => (
              <FormControlLabel
                key={instance.id}
                control={
                  <Checkbox
                    checked={selectedInstances.includes(instance.id)}
                    onChange={() => handleInstanceToggle(instance.id)}
                    size="small"
                  />
                }
                label={
                  <Typography variant="body2">
                    {instance.display_name || instance.instance_name}
                  </Typography>
                }
              />
            ))
          )}
        </FormGroup>
      </FormControl>

      <Divider sx={{ my: 2 }} />

      {/* Product Facets */}
      <FormControl component="fieldset" fullWidth>
        <FormLabel component="legend" sx={{ mb: 1, fontWeight: 600, fontSize: '0.875rem' }}>
          Product
        </FormLabel>
        <FormGroup>
          {products.length === 0 ? (
            <Typography variant="caption" color="text.secondary">
              No products available
            </Typography>
          ) : (
            products.map((product) => (
              <FormControlLabel
                key={product.id}
                control={
                  <Checkbox
                    checked={selectedProducts.includes(product.id)}
                    onChange={() => handleProductToggle(product.id)}
                    size="small"
                  />
                }
                label={
                  <Typography variant="body2">
                    {product.product_name}
                  </Typography>
                }
              />
            ))
          )}
        </FormGroup>
      </FormControl>
    </Paper>
  );
};
