import React, { useState } from 'react';
import { useLocation } from 'react-router-dom';
import BlockableLink from './RouteBlocker/BlockableLink';
import {
  AppBar,
  Toolbar,
  Typography,
  Button,
  Menu,
  MenuItem,
  Box,
  Chip,
  IconButton,
  useTheme,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Divider,
  useMediaQuery,
} from '@mui/material';
import {
  Menu as MenuIcon,
  Business as BusinessIcon,
  Storage as StorageIcon,
  Category as CategoryIcon,
  Security as SecurityIcon,
  Assessment as AssessmentIcon,
  SystemUpdateAlt as SystemUpdateAltIcon,
  Settings as SettingsIcon,
  Notifications as NotificationsIcon,
  QueryStats as QueryStatsIcon,
  Schema as SchemaIcon,
  Policy as PolicyIcon,
  Timeline as TimelineIcon,
  Brightness4 as Brightness4Icon,
  Brightness7 as Brightness7Icon,
  KeyboardArrowDown as KeyboardArrowDownIcon,
  Api as ApiIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';

interface NavigationItem {
  label: string;
  path: string;
  icon: React.ReactNode;
  description?: string;
}

interface NavigationGroup {
  label: string;
  icon: React.ReactNode;
  items: NavigationItem[];
}

interface MainNavigationProps {
  onToggleTheme: () => void;
}

export const MainNavigation: React.FC<MainNavigationProps> = ({ onToggleTheme }) => {
  const theme = useTheme();
  const location = useLocation();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  const { tenant } = useTenant();

  const navigationGroups: NavigationGroup[] = [
    {
      label: 'Data Management',
      icon: <StorageIcon />,
      items: [
        { label: 'Connections', path: '/connections', icon: <StorageIcon />, description: 'Manage data connections' },
        { label: 'Schema Browser', path: '/schema', icon: <SchemaIcon />, description: 'Browse database schemas' },
        { label: 'Query Builder', path: '/query-builder', icon: <QueryStatsIcon />, description: 'Build and execute queries' },
      ],
    },
    {
      label: 'Fabric & Policies',
      icon: <PolicyIcon />,
        items: [
        { label: 'Bundles', path: '/bundles', icon: <CategoryIcon />, description: 'Manage data bundles' },
        { label: 'Policies', path: '/policies', icon: <PolicyIcon />, description: 'Data governance policies' },
        { label: 'Drift Reports', path: '/drift', icon: <AssessmentIcon />, description: 'Monitor data drift' },
      ],
    },
    {
      label: 'Analytics & Insights',
      icon: <AssessmentIcon />,
      items: [
        { label: 'Private Markets', path: '/private-markets', icon: <BusinessIcon />, description: 'Private markets analytics' },
        { label: 'API Catalog', path: '/api-catalog', icon: <ApiIcon />, description: 'Available APIs' },
        { label: 'Calculations', path: '/calculations', icon: <TimelineIcon />, description: 'Financial calculations' },
        { label: 'Pre-aggregation', path: '/pre-aggregation', icon: <QueryStatsIcon />, description: 'Aggregation advisor' },
      ],
    },
    {
      label: 'System Management',
      icon: <SettingsIcon />,
      items: [
        { label: 'Tenants', path: '/tenants', icon: <BusinessIcon />, description: 'Multi-tenant management' },
        { label: 'Upgrades', path: '/upgrades', icon: <SystemUpdateAltIcon />, description: 'System upgrades' },
        { label: 'Access Intelligence', path: '/access-intelligence', icon: <SecurityIcon />, description: 'Access analytics' },
        { label: 'Notifications', path: '/notifications', icon: <NotificationsIcon />, description: 'System notifications' },
      ],
    },
  ];

  const handleMenuClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const toggleMobileMenu = () => {
    setMobileMenuOpen(!mobileMenuOpen);
  };

  const closeMobileMenu = () => {
    setMobileMenuOpen(false);
  };

  const renderNavigationItems = () => (
    <>
      {navigationGroups.map((group) => (
        <Box key={group.label} sx={{ mb: 3 }}>
          <Typography
            variant="subtitle2"
            sx={{
              color: 'text.secondary',
              fontWeight: 'bold',
              mb: 1,
              px: 2,
              fontSize: isMobile ? '0.875rem' : '0.75rem',
            }}
          >
            {group.label}
          </Typography>
          <List sx={{ py: 0 }}>
              {group.items.map((item) => (
              <ListItem
                key={item.path}
                component={BlockableLink}
                to={item.path}
                onClick={closeMobileMenu}
                sx={{
                  py: 1,
                  px: 2,
                  borderRadius: 1,
                  mx: 1,
                  mb: 0.5,
                  backgroundColor: location.pathname === item.path ? 'action.selected' : 'transparent',
                  '&:hover': {
                    backgroundColor: 'action.hover',
                  },
                  minHeight: isMobile ? '48px' : '40px',
                }}
              >
                <ListItemIcon sx={{ minWidth: isMobile ? '40px' : '36px', color: 'inherit' }}>
                  {item.icon}
                </ListItemIcon>
                <ListItemText
                  primary={item.label}
                  secondary={isMobile ? null : item.description}
                  primaryTypographyProps={{
                    fontSize: isMobile ? '1rem' : '0.875rem',
                    fontWeight: location.pathname === item.path ? 'bold' : 'normal',
                  }}
                  secondaryTypographyProps={{
                    fontSize: '0.75rem',
                    color: 'text.secondary',
                  }}
                />
              </ListItem>
            ))}
          </List>
        </Box>
      ))}
    </>
  );

  return (
    <>
      <AppBar
        position="static"
        elevation={1}
        sx={{
          backgroundColor: 'background.paper',
          color: 'text.primary',
          borderBottom: 1,
          borderColor: 'divider',
        }}
      >
        <Toolbar sx={{ minHeight: isMobile ? '56px' : '64px', px: isMobile ? 2 : 3 }}>
          {/* Mobile Menu Button */}
          {isMobile && (
            <IconButton
              edge="start"
              color="inherit"
              aria-label="menu"
              onClick={toggleMobileMenu}
              sx={{ mr: 1 }}
            >
              <MenuIcon />
            </IconButton>
          )}

          {/* Logo/Brand */}
            <Typography
            variant={isMobile ? "h6" : "h5"}
            component={BlockableLink}
            to="/"
            sx={{
              flexGrow: 1,
              textDecoration: 'none',
              color: 'inherit',
              fontWeight: 'bold',
              mr: 2,
            }}
          >
            SemLayer
          </Typography>

          {/* Desktop Navigation */}
          {!isMobile && (
            <Box sx={{ display: 'flex', gap: 1, mr: 2 }}>
              {navigationGroups.slice(0, 3).map((group) => (
                <Button
                  key={group.label}
                  color="inherit"
                  startIcon={group.icon}
                  endIcon={<KeyboardArrowDownIcon />}
                  onClick={handleMenuClick}
                  sx={{
                    textTransform: 'none',
                    fontSize: '0.875rem',
                    px: 2,
                    py: 1,
                  }}
                >
                  {group.label}
                </Button>
              ))}
            </Box>
          )}

          {/* Tenant Indicator */}
          {tenant && (
            <Chip
              label={tenant.name}
              size={isMobile ? "small" : "medium"}
              color="primary"
              sx={{
                mr: 2,
                fontSize: isMobile ? '0.75rem' : '0.875rem',
                height: isMobile ? '24px' : '32px',
              }}
            />
          )}

          {/* Theme Toggle */}
          <IconButton
            color="inherit"
            onClick={onToggleTheme}
            sx={{
              mr: isMobile ? 1 : 2,
              p: isMobile ? 1 : 1.5,
            }}
          >
            {theme.palette.mode === 'dark' ? <Brightness7Icon /> : <Brightness4Icon />}
          </IconButton>

          {/* Settings Menu */}
          <IconButton
            color="inherit"
            onClick={handleMenuClick}
            sx={{
              p: isMobile ? 1 : 1.5,
            }}
          >
            <SettingsIcon />
          </IconButton>
        </Toolbar>
      </AppBar>

      {/* Desktop Dropdown Menu */}
      {!isMobile && (
        <Menu
          anchorEl={anchorEl}
          open={Boolean(anchorEl)}
          onClose={handleMenuClose}
          PaperProps={{
            sx: {
              mt: 1,
              minWidth: 200,
            },
          }}
        >
          {navigationGroups.map((group) => [
            <MenuItem key={`${group.label}-header`} disabled>
              <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>
                {group.label}
              </Typography>
            </MenuItem>,
              ...group.items.map((item) => (
              <MenuItem
                key={item.path}
                component={BlockableLink}
                to={item.path}
                onClick={handleMenuClose}
                sx={{
                  pl: 4,
                  backgroundColor: location.pathname === item.path ? 'action.selected' : 'transparent',
                }}
              >
                <ListItemIcon sx={{ minWidth: '36px' }}>
                  {item.icon}
                </ListItemIcon>
                <ListItemText primary={item.label} />
              </MenuItem>
            )),
            <Divider key={`${group.label}-divider`} />,
          ])}
        </Menu>
      )}

      {/* Mobile Drawer */}
      <Drawer
        anchor="left"
        open={mobileMenuOpen}
        onClose={closeMobileMenu}
        sx={{
          '& .MuiDrawer-paper': {
            width: isMobile ? '280px' : '320px',
            backgroundColor: 'background.paper',
          },
        }}
      >
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
            <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
              Navigation
            </Typography>
            <IconButton onClick={closeMobileMenu}>
              <CloseIcon />
            </IconButton>
          </Box>
          {tenant && (
            <Chip
              label={tenant.name}
              size="small"
              color="primary"
              sx={{ mt: 1 }}
            />
          )}
        </Box>

        <Box sx={{ overflow: 'auto', flexGrow: 1 }}>
          {renderNavigationItems()}
        </Box>
      </Drawer>
    </>
  );
};
