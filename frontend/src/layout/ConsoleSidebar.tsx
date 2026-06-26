import { List, ListItem, ListItemButton, ListItemIcon, ListItemText, Drawer } from '@mui/material';
import DashboardIcon from '@mui/icons-material/Dashboard';
import VerifiedUserIcon from '@mui/icons-material/VerifiedUser';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import StorageIcon from '@mui/icons-material/Storage';
import AdminPanelSettingsIcon from '@mui/icons-material/AdminPanelSettings';

const NAV_ITEMS = [
  {
    label: 'Dashboard',
    icon: DashboardIcon,
    href: '/console/dashboard',
  },
  {
    label: 'Compliance',
    items: [
      { label: 'Rules', href: '/console/compliance/rules' },
      { label: 'Breaches', href: '/console/compliance/breaches' },
      { label: 'Lineage', href: '/console/compliance/lineage' },
      { label: 'Evaluations', href: '/console/compliance/evaluations' },
    ],
  },
  {
    label: 'Risk',
    items: [
      { label: 'Portfolio Risk', href: '/console/risk/portfolio' },
      { label: 'Factor Exposures', href: '/console/risk/factors' },
      { label: 'VaR', href: '/console/risk/var' },
      { label: 'Scenarios', href: '/console/risk/scenarios' },
      { label: 'Lineage', href: '/console/risk/lineage' },
    ],
  },
  {
    label: 'ETL & Execution',
    icon: StorageIcon,
    items: [
      { label: 'ETL Runs', href: '/console/etl/runs' },
      { label: 'WASM Versions', href: '/console/etl/wasm' },
      { label: 'Execution Logs', href: '/console/etl/logs' },
    ],
  },
  {
    label: 'Admin',
    icon: AdminPanelSettingsIcon,
    items: [
      { label: 'Tenants', href: '/console/admin/tenants' },
      { label: 'Users', href: '/console/admin/users' },
      { label: 'Settings', href: '/console/admin/settings' },
    ],
  },
];

export function ConsoleSidebar() {
  return (
    <Drawer
      variant="permanent"
      sx={{
        width: 280,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: 280,
          boxSizing: 'border-box',
          backgroundColor: '#f5f5f5',
          borderRight: '1px solid #e0e0e0',
        },
      }}
    >
      <List sx={{ pt: 2 }}>
        {NAV_ITEMS.map((item, i) => (
          item.href ?
            (
              <ListItem key={i} disablePadding>
                <ListItemButton href={item.href}>
                  {item.icon && <ListItemIcon>{React.createElement(item.icon)}</ListItemIcon>}
                  <ListItemText primary={item.label} />
                </ListItemButton>
              </ListItem>
            )
            : (
              <div key={i}>
                <ListItem sx={{ py: 1 }}>
                  <ListItemText
                    primary={item.label}
                    primaryTypographyProps={{ variant: 'caption', fontWeight: 600 }}
                    sx={{ textTransform: 'uppercase', color: '#666' }}
                  />
                </ListItem>
                {item.items?.map((subitem, j) => (
                  <ListItem key={j} disablePadding sx={{ pl: 2 }}>
                    <ListItemButton href={subitem.href}>
                      <ListItemText
                        primary={subitem.label}
                        primaryTypographyProps={{ variant: 'body2' }}
                      />
                    </ListItemButton>
                  </ListItem>
                ))}
              </div>
            )
        ))}
      </List>
    </Drawer>
  );
}

import React from 'react';
