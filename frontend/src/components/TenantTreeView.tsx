import { FC, SyntheticEvent, MouseEvent } from 'react';
import { SimpleTreeView, TreeItem } from '@mui/x-tree-view';
import { Box, Typography, IconButton, Tooltip } from '@mui/material';
import renderCoreCustomChips from './common/semanticChips';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import ViewListIcon from '@mui/icons-material/ViewList';
import { Tenant, TenantInstance } from '../types';

interface TenantTreeViewProps {
  tenants: Tenant[];
  onSelect: (item: Tenant | TenantInstance) => void;
  onAddInstance: (tenantId: string) => void;
  onShowProducts: (instance: TenantInstance) => void;
}

const TenantTreeView: FC<TenantTreeViewProps> = ({ 
  tenants, 
  onSelect, 
  onAddInstance, 
  onShowProducts 
} : TenantTreeViewProps) => {
  // Our app expects a simple handler that receives a single selected id (string|null).
  // MUI's SimpleTreeView uses a different signature: (event, itemIds: string[]).
  // Provide a small wrapper that adapts the library signature to our app logic.
  const handleSelect = (_event: SyntheticEvent, itemIds: string[] | string | null) => {
    // Normalize to a single id if possible
    let itemId: string | null = null;
    if (!itemIds) itemId = null;
    else if (Array.isArray(itemIds)) itemId = itemIds[0] ?? null;
    else itemId = itemIds as string;

    if (!itemId) return;

    const tenant = tenants.find(t => t.id === itemId);
    if (tenant) {
      onSelect(tenant);
      return;
    }

    for (const t of tenants) {
      const instance = t.tenant_instances.find(i => i.id === itemId);
      if (instance) {
        onSelect(instance);
        return;
      }
    }
  };

  const handleIconClick = (event: MouseEvent, action: () => void) => {
    event.stopPropagation(); // Prevent the tree item from being selected
    action();
  };
  return (
    <SimpleTreeView onSelectedItemsChange={handleSelect}>
      {tenants.map((tenant) => {
        // Runtime check for itemId and label for tenant
        const tenantId = tenant.id != null && typeof tenant.id !== 'object' ? String(tenant.id) : '';
        const tenantLabel = (
          <Box sx={{ display: 'flex', alignItems: 'center', py: 0.5 }}>
            <Typography sx={{ flexGrow: 1 }}>
              {String(tenant.display_name || tenant.name || 'Unnamed Tenant')}
              {tenant.gold_copy && <Tooltip title="core — read-only"><Box component="span">{renderCoreCustomChips({ is_core: true })}</Box></Tooltip>}
            </Typography>
            <IconButton 
              size="small" 
              onClick={(e) => handleIconClick(e, () => onAddInstance(tenant.id))}
              title="Add Instance"
            >
              <AddCircleOutlineIcon fontSize="inherit" />
            </IconButton>
          </Box>
        );
        return (
          <TreeItem
            key={tenantId}
            itemId={tenantId}
            label={tenantLabel}
          >
            {tenant.tenant_instances.map((instance) => {
              // Runtime check for itemId and label for instance
              const instanceId = instance.id != null && typeof instance.id !== 'object' ? String(instance.id) : '';
              const instanceLabel = (
                <Box sx={{ display: 'flex', alignItems: 'center', py: 0.5 }}>
                  <Typography sx={{ flexGrow: 1 }}>
                    {String(instance.display_name || instance.instance_name || instance.id || 'Unnamed Instance')}
                  </Typography>
                  <IconButton 
                    size="small" 
                    onClick={(e) => handleIconClick(e, () => onShowProducts(instance))}
                    title="Show Products"
                  >
                    <ViewListIcon fontSize="inherit" />
                  </IconButton>
                </Box>
              );
              return (
                <TreeItem
                  key={instanceId}
                  itemId={instanceId}
                  label={instanceLabel}
                />
              );
            })}
          </TreeItem>
        );
      })}
    </SimpleTreeView>
  );
};

export default TenantTreeView;