// src/components/TenantInstancesSidebar.tsx
import React from 'react';
import { List, ListItemButton, ListItemText, ListItemIcon, Divider, Chip } from '@mui/material';
import DnsIcon from '@mui/icons-material/Dns';

interface TenantInstance {
  id: string;
  display_name?: string;
  [key: string]: any; // Allow additional properties
}

interface TenantInstancesSidebarProps {
  instances: TenantInstance[];
  selectedInstanceId: string | null;
  onSelect: (instanceId: string | null) => void;
  activeTab?: number; // 0=Instances, 1=Products, 2=Data Sources, 3=Lookups
  totalLookups?: number; // Total number of lookups for Lookups tab
}

const TenantInstancesSidebar: React.FC<TenantInstancesSidebarProps> = ({ 
  instances, 
  selectedInstanceId, 
  onSelect,
  activeTab = 0,
  totalLookups = 0
}) => {
  
  const getInstanceCount = (instance: TenantInstance) => {
    // Return count based on active tab
    switch (activeTab) {
      case 1: // Products tab
        return (instance as any).tenant_products?.length || 0;
      case 2: // Data Sources tab
        // Count all datasources across all products for this instance
        const products = (instance as any).tenant_products || [];
        return products.reduce((total: number, product: any) => {
          return total + (product.tenant_product_datasources?.length || 0);
        }, 0);
      case 3: // Lookups tab
        return 0; // Lookups are tenant-wide, not instance-specific
      default: // Instances tab or unknown
        return (instance as any).tenant_products?.length || 0;
    }
  };

  const getTotalCount = () => {
    if (activeTab === 3) {
      return totalLookups; // Return total lookups for Lookups tab
    }
    return instances.reduce((sum, instance) => sum + getInstanceCount(instance), 0);
  };

  return (
    <List component="nav" sx={{ width: '100%', maxWidth: 360, bgcolor: 'background.paper' }}>
      <ListItemButton selected={selectedInstanceId === null} onClick={() => onSelect(null)}>
        <ListItemIcon>
          <DnsIcon />
        </ListItemIcon>
        <ListItemText primary="All Instances" />
        <Chip label={getTotalCount()} size="small" />
      </ListItemButton>
      <Divider />
      {instances.map((instance) => (
        <ListItemButton
          key={instance.id}
          selected={selectedInstanceId === instance.id}
          onClick={() => onSelect(instance.id)}
        >
          <ListItemIcon>
            <DnsIcon />
          </ListItemIcon>
          <ListItemText primary={instance.display_name || (instance as any).instance_name || instance.id} />
          {activeTab !== 3 && <Chip label={getInstanceCount(instance)} size="small" />}
        </ListItemButton>
      ))}
    </List>
  );
};

export default TenantInstancesSidebar;
