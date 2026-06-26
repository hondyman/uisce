import React, { useMemo } from 'react';
import { Box, Typography, List, ListItemButton, ListItemText, Chip, Paper, Collapse } from '@mui/material';
import BusinessIcon from '@mui/icons-material/Business';
import DnsIcon from '@mui/icons-material/Dns';
import FilterAltOffIcon from '@mui/icons-material/FilterAltOff';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';

interface ConnectionsFacetProps {
  connections: any[];
  selectedTenantId: string | null;
  selectedInstanceId: string | null;
  onFilterChange: (tenantId: string | null, instanceId: string | null) => void;
}

interface InstanceNode {
  id: string;
  name: string;
  count: number;
}

interface TenantNode {
  id: string;
  name: string;
  count: number;
  instances: InstanceNode[];
}

const ConnectionsFacet: React.FC<ConnectionsFacetProps> = ({
  connections,
  selectedTenantId,
  selectedInstanceId,
  onFilterChange,
}) => {
  
  // Compute hierarchy and counts
  const hierarchy = useMemo(() => {
    const tenantMap = new Map<string, TenantNode>();

    connections.forEach(conn => {
      const tenantId = conn.tenant?.id;
      const tenantName = conn.tenantName;
      const instanceId = conn.instance?.id;
      const instanceName = conn.instanceName;

      if (!tenantId || !instanceId) return;

      if (!tenantMap.has(tenantId)) {
        tenantMap.set(tenantId, {
          id: tenantId,
          name: tenantName,
          count: 0,
          instances: []
        });
      }

      const tenantNode = tenantMap.get(tenantId)!;
      tenantNode.count++;

      let instanceNode = tenantNode.instances.find(i => i.id === instanceId);
      if (!instanceNode) {
        instanceNode = { id: instanceId, name: instanceName, count: 0 };
        tenantNode.instances.push(instanceNode);
      }
      instanceNode.count++;
    });

    return Array.from(tenantMap.values()).sort((a, b) => a.name.localeCompare(b.name));
  }, [connections]);

  const [openTenants, setOpenTenants] = React.useState<Record<string, boolean>>({});

  const handleToggleTenant = (tenantId: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setOpenTenants(prev => ({ ...prev, [tenantId]: !prev[tenantId] }));
  };

  const handleSelectTenant = (tenantId: string) => {
    // If clicking the already selected tenant, keep it selected (or could deselect? standard is keep)
    // If we want to allow deselecting by clicking again, we could check if selectedTenantId === tenantId
    // But usually "All" button is for clearing.
    // Let's toggle expand when selecting
    if (!openTenants[tenantId]) {
        setOpenTenants(prev => ({ ...prev, [tenantId]: true }));
    }
    onFilterChange(tenantId, null);
  };

  const handleSelectInstance = (tenantId: string, instanceId: string) => {
    onFilterChange(tenantId, instanceId);
  };

  const clearFilter = () => {
    onFilterChange(null, null);
  };

  // Auto-expand selected tenant on mount/update
  React.useEffect(() => {
    if (selectedTenantId && !openTenants[selectedTenantId]) {
      setOpenTenants(prev => ({ ...prev, [selectedTenantId]: true }));
    }
  }, [selectedTenantId]);

  return (
    <Paper 
      elevation={0} 
      className="connections-facet-sidebar"
      sx={{ 
        width: 280, 
        minWidth: 280,
        mr: 3, 
        p: 2, 
        border: '1px solid', 
        borderColor: 'divider', 
        height: 'fit-content',
        maxHeight: 'calc(100vh - 100px)',
        overflowY: 'auto'
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="subtitle1" fontWeight="bold">Filters</Typography>
        {(selectedTenantId || selectedInstanceId) && (
          <Chip 
            size="small" 
            label="Reset" 
            icon={<FilterAltOffIcon sx={{ fontSize: 14 }} />} 
            onClick={clearFilter} 
            color="primary" 
            variant="outlined"
            clickable
          />
        )}
      </Box>

      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1, textTransform: 'uppercase', fontWeight: 600 }}>
        By Tenant & Instance
      </Typography>

      <List component="nav" dense disablePadding>
        {hierarchy.map(tenant => {
            const isTenantSelected = selectedTenantId === tenant.id;
            const isTenantActive = selectedTenantId === tenant.id && !selectedInstanceId; // strictly expanding this tenant logic visually?
            const isOpen = openTenants[tenant.id];

            return (
            <React.Fragment key={tenant.id}>
                <ListItemButton 
                    selected={isTenantSelected && !selectedInstanceId}
                    onClick={() => handleSelectTenant(tenant.id)}
                    sx={{ 
                        borderRadius: 1, 
                        mb: 0.5,
                        '&.Mui-selected': { bgcolor: 'primary.lighter', color: 'primary.main' }
                    }}
                >
                    <BusinessIcon sx={{ mr: 1, opacity: 0.7, fontSize: 16 }} />
                    <ListItemText 
                        primary={tenant.name} 
                        secondary={`${tenant.count} datasources`}
                        primaryTypographyProps={{ variant: 'body2', fontWeight: isTenantSelected ? 600 : 400 }}
                        secondaryTypographyProps={{ variant: 'caption', sx: { fontSize: '0.7rem' } }}
                    />
                    <Box 
                        onClick={(e) => handleToggleTenant(tenant.id, e)} 
                        sx={{ p: 0.5, borderRadius: '50%', '&:hover': { bgcolor: 'action.hover' } }}
                    >
                        {isOpen ? <ExpandLess fontSize="small" /> : <ExpandMore fontSize="small" />}
                    </Box>
                </ListItemButton>
                
                <Collapse in={isOpen} timeout="auto" unmountOnExit>
                    <List component="div" disablePadding>
                        {tenant.instances.map(instance => (
                            <ListItemButton 
                                key={instance.id}
                                selected={selectedInstanceId === instance.id}
                                onClick={() => handleSelectInstance(tenant.id, instance.id)}
                                sx={{ 
                                    pl: 4, 
                                    borderRadius: 1, 
                                    mb: 0.5,
                                    py: 0.5
                                }}
                            >
                                <DnsIcon sx={{ mr: 1, opacity: 0.7, fontSize: 14 }} />
                                <ListItemText 
                                    primary={instance.name} 
                                    secondary={instance.count}
                                    primaryTypographyProps={{ variant: 'body2', fontSize: '0.8125rem' }}
                                    secondaryTypographyProps={{ variant: 'caption' }}
                                    sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                                />
                            </ListItemButton>
                        ))}
                    </List>
                </Collapse>
            </React.Fragment>
            );
        })}
      </List>
      
      {hierarchy.length === 0 && (
          <Typography variant="body2" color="text.disabled" sx={{ fontStyle: 'italic', mt: 2, textAlign: 'center' }}>
              No data available
          </Typography>
      )}
    </Paper>
  );
};

export default ConnectionsFacet;
