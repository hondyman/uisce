// frontend/src/components/TenantSwitcher.tsx
// Unified tenant/scope switcher component

import React, { useState } from 'react';
import {
  Box,
  Button,
  Typography,
  IconButton,
  Tooltip,
  Breadcrumbs,
  Link,
} from '@mui/material';
import {
  Business as BusinessIcon,
  ExpandMore as ExpandMoreIcon,
  ChevronRight as ChevronRightIcon,
  Public as PublicIcon,
  Dns as InstanceIcon,
  Inventory as ProductIcon,
  Storage as DatasourceIcon,
  ArrowDropDown as ArrowDropDownIcon,
} from '@mui/icons-material';
import { useAccess } from '../contexts/AccessContext';
import { ScopeSelectorDialog } from './ScopeSelectorDialog';

interface TenantSwitcherProps {
  /** Compact mode for narrow spaces */
  compact?: boolean;
  /** Show as inline breadcrumb style */
  breadcrumbMode?: boolean;
}

export const TenantSwitcher: React.FC<TenantSwitcherProps> = ({ 
  compact = false,
  breadcrumbMode = false 
}) => {
  const {
    isPlatformOperator,
    scope,
    scopeDescription,
    currentTenant,
    currentInstance,
    currentProduct,
    currentDatasource,
    setGlobalScope,
    setTenantScope,
    setInstanceScope,
    setProductScope,
  } = useAccess();

  const [dialogOpen, setDialogOpen] = useState(false);

  const handleClick = () => {
    setDialogOpen(true);
  };

  const handleClose = () => {
    setDialogOpen(false);
  };

  // Get scope icon based on level
  const getScopeIcon = () => {
    if (scope.isGlobal) return <PublicIcon fontSize="small" />;
    if (scope.level === 'datasource') return <DatasourceIcon fontSize="small" />;
    if (scope.level === 'product') return <ProductIcon fontSize="small" />;
    if (scope.level === 'instance') return <InstanceIcon fontSize="small" />;
    if (scope.level === 'tenant') return <BusinessIcon fontSize="small" />;
    return <BusinessIcon fontSize="small" />;
  };

  // Breadcrumb mode - shows current scope as clickable breadcrumbs
  if (breadcrumbMode) {
    return (
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <Breadcrumbs separator={<ChevronRightIcon fontSize="small" />} sx={{ color: 'inherit' }}>
          {isPlatformOperator && (
            <Link
              component="button"
              variant="body2"
              onClick={() => setGlobalScope()}
              sx={{ 
                color: scope.isGlobal ? 'primary.main' : 'inherit',
                textDecoration: 'none',
                cursor: 'pointer',
                '&:hover': { textDecoration: 'underline' }
              }}
            >
              All Tenants
            </Link>
          )}
          {currentTenant && (
            <Link
              component="button"
              variant="body2"
              onClick={() => setTenantScope(currentTenant)}
              sx={{ 
                color: scope.level === 'tenant' ? 'primary.main' : 'inherit',
                textDecoration: 'none',
                cursor: 'pointer',
                '&:hover': { textDecoration: 'underline' }
              }}
            >
              {currentTenant.display_name || currentTenant.name}
            </Link>
          )}
          {currentInstance && (
            <Link
              component="button"
              variant="body2"
              onClick={() => currentTenant && setInstanceScope(currentTenant, currentInstance)}
              sx={{ 
                color: scope.level === 'instance' ? 'primary.main' : 'inherit',
                textDecoration: 'none',
                cursor: 'pointer',
                '&:hover': { textDecoration: 'underline' }
              }}
            >
              {currentInstance.display_name || currentInstance.instance_name}
            </Link>
          )}
          {currentProduct && (
            <Link
              component="button"
              variant="body2"
              onClick={() => currentTenant && currentInstance && setProductScope(currentTenant, currentInstance, currentProduct)}
              sx={{ 
                color: scope.level === 'product' ? 'primary.main' : 'inherit',
                textDecoration: 'none',
                cursor: 'pointer',
                '&:hover': { textDecoration: 'underline' }
              }}
            >
              {currentProduct.alpha_product?.product_name}
            </Link>
          )}
          {currentDatasource && (
            <Typography variant="body2" color="primary">
              {currentDatasource.source_name}
            </Typography>
          )}
        </Breadcrumbs>
        <IconButton size="small" onClick={handleClick}>
          <ArrowDropDownIcon />
        </IconButton>
        <ScopeSelectorDialog open={dialogOpen} onClose={handleClose} />
      </Box>
    );
  }

  // Standard mode - button with dialog
  return (
    <>
      <Tooltip title={`Current Operating Scope: ${scopeDescription}`}>
        <Button
          onClick={handleClick}
          variant="outlined"
          size={compact ? 'small' : 'medium'}
          startIcon={getScopeIcon()}
          endIcon={<ExpandMoreIcon />}
          sx={{
            textTransform: 'none',
            minWidth: compact ? 'auto' : 220,
            maxWidth: 400,
            justifyContent: 'space-between',
            borderColor: 'rgba(255,255,255,0.3)',
            color: 'inherit',
            transition: 'all 0.2s',
            '&:hover': {
              borderColor: 'rgba(255,255,255,0.7)',
              bgcolor: 'rgba(255,255,255,0.05)',
            }
          }}
        >
          <Box sx={{ flex: 1, textAlign: 'left', overflow: 'hidden', mr: 1 }}>
            <Typography 
              variant="caption" 
              sx={{ 
                display: 'block', 
                opacity: 0.7, 
                lineHeight: 1,
                mb: 0.2
              }}
            >
              Operating Scope
            </Typography>
            <Typography 
              variant="body2" 
              noWrap 
              sx={{ 
                fontWeight: 'bold',
                maxWidth: compact ? 100 : 250 
              }}
            >
              {scope.isGlobal 
                ? 'All Tenants' 
                : scope.datasourceName 
                  || scope.productName 
                  || scope.instanceName 
                  || scope.tenantName 
                  || 'Not set'}
            </Typography>
          </Box>
        </Button>
      </Tooltip>

      <ScopeSelectorDialog open={dialogOpen} onClose={handleClose} />
    </>
  );
};

export default TenantSwitcher;
