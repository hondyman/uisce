// frontend/src/components/AccessAwareNavigation.tsx
// Example of access-aware navigation that shows/hides menus based on user scope

import React from 'react';
import { useLocation } from 'react-router-dom';
import {
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Collapse,
  Alert,
  Box,
  Typography,
  Chip,
  Tooltip,
} from '@mui/material';
import {
  ExpandLess,
  ExpandMore,
  Lock as LockIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';
import { useAccess } from '../contexts/AccessContext';
import { ScopeLevel } from '../types/access';
import BlockableLink from './RouteBlocker/BlockableLink';

interface NavItem {
  label: string;
  path: string;
  icon: React.ReactNode;
  description?: string;
  children?: NavItem[];
}

interface AccessAwareNavigationProps {
  items: NavItem[];
}

/**
 * Navigation component that is aware of user access levels and current scope.
 * Shows which items are accessible, which need scope selection, and which are locked.
 */
export const AccessAwareNavigation: React.FC<AccessAwareNavigationProps> = ({ items }) => {
  const location = useLocation();
  const { 
    canAccess, 
    requiresScope, 
    scope,
    isPlatformOperator,
    currentTenant,
    currentDatasource,
  } = useAccess();
  
  const [openGroups, setOpenGroups] = React.useState<Record<string, boolean>>({});

  const toggleGroup = (label: string) => {
    setOpenGroups(prev => ({ ...prev, [label]: !prev[label] }));
  };

  const getScopeStatusIcon = (minScope: ScopeLevel) => {
    // Check if current scope meets requirement
    const scopeLevels: ScopeLevel[] = ['global', 'tenant', 'instance', 'product', 'datasource'];
    const currentLevel = scopeLevels.indexOf(scope.level);
    const requiredLevel = scopeLevels.indexOf(minScope);

    if (minScope === 'global') {
      return <CheckCircleIcon fontSize="small" color="success" />;
    }

    // For non-global, we need to have drilled down to the right level
    if (currentLevel >= requiredLevel) {
      return <CheckCircleIcon fontSize="small" color="success" />;
    }

    return (
      <Tooltip title={`Select a ${minScope} first`}>
        <WarningIcon fontSize="small" color="warning" />
      </Tooltip>
    );
  };

  const renderNavItem = (item: NavItem, depth: number = 0) => {
    const { allowed, reason } = canAccess(item.path);
    const minScope = requiresScope(item.path);
    const isCurrentPath = location.pathname === item.path;
    const hasChildren = item.children && item.children.length > 0;

    if (hasChildren) {
      // Render as expandable group
      const isOpen = openGroups[item.label] ?? false;
      
      return (
        <React.Fragment key={item.label}>
          <ListItem disablePadding>
            <ListItemButton onClick={() => toggleGroup(item.label)} sx={{ pl: depth * 2 + 2 }}>
              <ListItemIcon>{item.icon}</ListItemIcon>
              <ListItemText primary={item.label} />
              {isOpen ? <ExpandLess /> : <ExpandMore />}
            </ListItemButton>
          </ListItem>
          <Collapse in={isOpen} timeout="auto" unmountOnExit>
            <List component="div" disablePadding>
              {item.children?.map(child => renderNavItem(child, depth + 1))}
            </List>
          </Collapse>
        </React.Fragment>
      );
    }

    // Regular nav item
    return (
      <ListItem key={item.path} disablePadding>
        <ListItemButton
          component={allowed ? BlockableLink : 'div'}
          to={allowed ? item.path : undefined}
          disabled={!allowed}
          selected={isCurrentPath}
          sx={{ 
            pl: depth * 2 + 2,
            opacity: allowed ? 1 : 0.5,
          }}
        >
          <ListItemIcon>
            {allowed ? item.icon : <LockIcon color="disabled" />}
          </ListItemIcon>
          <ListItemText 
            primary={item.label} 
            secondary={!allowed ? reason : item.description}
            primaryTypographyProps={{
              color: allowed ? 'text.primary' : 'text.disabled'
            }}
          />
          {minScope !== 'global' && getScopeStatusIcon(minScope)}
        </ListItemButton>
      </ListItem>
    );
  };

  return (
    <Box>
      {/* Scope status banner */}
      {!currentDatasource && (
        <Alert 
          severity="info" 
          sx={{ m: 1, mb: 2 }}
          icon={<WarningIcon />}
        >
          <Typography variant="body2">
            {!currentTenant 
              ? 'Select a tenant to access most features'
              : 'Drill down to a datasource to access all features'
            }
          </Typography>
        </Alert>
      )}

      {/* Access level indicator */}
      <Box sx={{ px: 2, py: 1, display: 'flex', alignItems: 'center', gap: 1 }}>
        <Typography variant="caption" color="text.secondary">
          Access Level:
        </Typography>
        <Chip 
          size="small" 
          label={isPlatformOperator ? 'Platform Operator' : 'Tenant User'}
          color={isPlatformOperator ? 'primary' : 'default'}
        />
      </Box>

      <List>
        {items.map(item => renderNavItem(item))}
      </List>
    </Box>
  );
};

export default AccessAwareNavigation;
