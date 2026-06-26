// React import removed (automatic JSX runtime in use)
import { Box, Typography, Chip, Paper, Button, Tooltip } from '@mui/material';
import renderCoreCustomChips from './common/semanticChips';
import BusinessIcon from '@mui/icons-material/Business';
import CategoryIcon from '@mui/icons-material/Category';
import StorageIcon from '@mui/icons-material/Storage';
import ClearIcon from '@mui/icons-material/Clear';
import { useTenant } from '../contexts/TenantContext';

interface SelectionStatusProps {
  showClearButton?: boolean;
  variant?: 'compact' | 'full';
}

const SelectionStatus: React.FC<SelectionStatusProps> = ({ 
  showClearButton = false, 
  variant = 'compact' 
}) => {
  const { tenant, product, datasource, clearSelection, isSelected } = useTenant();

  if (!isSelected) {
    return (
      <Paper sx={{ p: 2, textAlign: 'center' }}>
        <Typography color="text.secondary" variant="body2">
          No datasource selected. Please go to the Tenants page to make a selection.
        </Typography>
      </Paper>
    );
  }

  if (variant === 'compact') {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
        {tenant && (
          <>
            <Chip 
              icon={<BusinessIcon />}
              label={String(tenant.display_name || tenant.name || 'Unnamed Tenant')}
              size="small"
              color="primary"
              variant="outlined"
            />
            {tenant.gold_copy && (
              <Tooltip title="core — read-only">
                <Box component="span">{renderCoreCustomChips({ is_core: true })}</Box>
              </Tooltip>
            )}
          </>
        )}
        {product && (
          <Chip 
            icon={<CategoryIcon />}
            label={product.alpha_product?.product_name}
            size="small"
            color="secondary"
            variant="outlined"
          />
        )}
        {datasource && (
          <Chip 
            icon={<StorageIcon />}
            label={`${datasource.source_name} (${datasource.alpha_datasource?.datasource_type})`}
            size="small"
            color="success"
            variant="outlined"
          />
        )}
        {showClearButton && (
          <Button
            size="small"
            startIcon={<ClearIcon />}
            onClick={clearSelection}
            variant="outlined"
            color="warning"
          >
            Clear Selection
          </Button>
        )}
      </Box>
    );
  }

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        Current Selection
      </Typography>
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {tenant && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <BusinessIcon color="primary" />
            <Typography variant="body1">
              <strong>Tenant:</strong> {String(tenant.display_name || tenant.name || 'Unnamed Tenant')}
              {tenant.gold_copy && <Tooltip title="core — read-only"><Box component="span">{renderCoreCustomChips({ is_core: true })}</Box></Tooltip>}
            </Typography>
          </Box>
        )}
        {product && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <CategoryIcon color="secondary" />
            <Typography variant="body1">
              <strong>Product:</strong> {product.alpha_product?.product_name} (v{product.version})
            </Typography>
          </Box>
        )}
        {datasource && (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <StorageIcon color="success" />
            <Typography variant="body1">
              <strong>Datasource:</strong> {datasource.source_name} ({datasource.alpha_datasource?.datasource_type})
            </Typography>
          </Box>
        )}
        {showClearButton && (
          <Box sx={{ mt: 2 }}>
            <Button
              startIcon={<ClearIcon />}
              onClick={clearSelection}
              variant="outlined"
              color="warning"
            >
              Clear Selection
            </Button>
          </Box>
        )}
      </Box>
    </Paper>
  );
};

export default SelectionStatus;