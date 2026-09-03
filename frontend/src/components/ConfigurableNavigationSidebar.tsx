import React, { useState, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { stripLocale } from '../i18n/locales';
import { 
  Box, List, ListItem, ListItemButton, ListItemIcon, 
  ListItemText, ListSubheader, Typography, Skeleton 
} from '@mui/material';
import DashboardIcon from '@mui/icons-material/Dashboard';
import ShoppingCartIcon from '@mui/icons-material/ShoppingCart';
import InventoryIcon from '@mui/icons-material/Inventory';
import PeopleIcon from '@mui/icons-material/People';

import { useTenant } from '../contexts/TenantContext';

interface MenuNode {
  id: string;
  node_key: string;
  label: string;
  icon: string;
  target_page_key?: string;
  children?: MenuNode[];
}

const iconMapping: Record<string, React.ReactNode> = {
  'DASHBOARD': <DashboardIcon fontSize="small" />,
  'ORDERS': <ShoppingCartIcon fontSize="small" />,
  'PRODUCTS': <InventoryIcon fontSize="small" />,
  'CUSTOMERS': <PeopleIcon fontSize="small" />
};

export const ConfigurableNavigationSidebar: React.FC = () => {
  const [menuTree, setMenuTree] = useState<MenuNode[]>([]);
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();
  const location = useLocation();
  const { tenant } = useTenant();

  useEffect(() => {
    // Dynamic menu layout loading based on current tenant state boundaries
    const tenantId = tenant?.id || "11111111-1111-1111-1111-111111111111";
    fetch(`/api/v1/layout/navigation-menu?tenant_id=${tenantId}`)
      .then(res => res.json())
      .then(data => {
        setMenuTree(data || []);
        setLoading(false)
      })
      .catch(() => setLoading(false));
  }, [tenant?.id]);

  if (loading) {
    return (
      <Box sx={{ width: 280, p: 3, borderRight: '1px solid #e2e8f0', height: '100vh' }}>
        <Skeleton variant="text" width="60%" height={32} sx={{ mb: 4 }} />
        <Skeleton variant="rounded" height={40} sx={{ mb: 2 }} />
        <Skeleton variant="rounded" height={40} sx={{ mb: 2 }} />
        <Skeleton variant="rounded" height={40} sx={{ mb: 2 }} />
      </Box>
    );
  }

  return (
    <Box sx={{ width: 280, borderRight: '1px solid #e2e8f0', bgcolor: '#0f172a', color: '#94a3b8', height: '100vh', pt: 4 }}>
      <Typography variant="h6" fontWeight="700" color="#f8fafc" px={3} mb={4}>
        Uisce OS <span style={{ fontWeight: 300, fontSize: '14px', color: '#38bdf8' }}>Northwind</span>
      </Typography>

      <List 
        subheader={<ListSubheader sx={{ bgcolor: 'transparent', color: '#475569', fontWeight: 600, fontSize: '11px', textTransform: 'uppercase', letterSpacing: '1px' }}>Data Products Menu</ListSubheader>}
      >
        {menuTree.map((node) => (
          <Box key={node.id}>
            <ListItem disablePadding>
              <ListItemButton
                selected={stripLocale(location.pathname) === `/app/data-product/${node.target_page_key}`}
                onClick={() => node.target_page_key && navigate(`/app/data-product/${node.target_page_key}`)}
                sx={{
                  px: 3,
                  py: 1,
                  color: stripLocale(location.pathname) === `/app/data-product/${node.target_page_key}` ? '#38bdf8' : 'inherit',
                  '&.Mui-selected': { bgcolor: 'rgba(56, 189, 248, 0.08)', color: '#38bdf8' },
                  '&:hover': { bgcolor: 'rgba(255,255,255,0.03)' }
                }}
              >
                <ListItemIcon sx={{ color: 'inherit', minWidth: '40px' }}>
                  {iconMapping[node.icon] || <DashboardIcon />}
                </ListItemIcon>
                <ListItemText primary={node.label} primaryTypographyProps={{ fontSize: '14px', fontWeight: 500 }} />
              </ListItemButton>
            </ListItem>

            {/* Hierarchical rendering wrapper for submenus */}
            {node.children && node.children.map(child => (
              <ListItemButton
                key={child.id}
                selected={stripLocale(location.pathname) === `/app/data-product/${child.target_page_key}`}
                onClick={() => child.target_page_key && navigate(`/app/data-product/${child.target_page_key}`)}
                sx={{ 
                  pl: 6, 
                  py: 0.5, 
                  color: stripLocale(location.pathname) === `/app/data-product/${child.target_page_key}` ? '#38bdf8' : 'inherit',
                  '&.Mui-selected': { bgcolor: 'rgba(56, 189, 248, 0.08)', color: '#38bdf8' },
                  '&:hover': { bgcolor: 'rgba(255,255,255,0.03)' }
                }}
              >
                <ListItemText primary={child.label} primaryTypographyProps={{ fontSize: '13px' }} />
              </ListItemButton>
            ))}
          </Box>
        ))}
      </List>
    </Box>
  );
};
