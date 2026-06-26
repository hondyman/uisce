import { useState, MouseEvent } from 'react';
import useBlockableNavigate from '../../components/RouteBlocker/useBlockableNavigate';
import { useAuth } from '../../contexts/AuthContext';
import {
  Box,
  Typography,
  Button,
  Menu,
  MenuItem,
  Avatar,
  Chip,
  Alert,
  CircularProgress,
  Drawer,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  IconButton,
  Tooltip,
  Divider,
  useTheme,
  useMediaQuery,
  AppBar,
  Toolbar,
} from '@mui/material';
import {
  AccountCircle,
  Business,
  Assessment,
  Settings,
  ExitToApp,
  Dashboard,
  People,
  Analytics,
  Compare,
  Wifi,
  WifiOff,
  Refresh,
  Menu as MenuIcon,
  Close as CloseIcon,
} from '@mui/icons-material';
import { Dialog, DialogContent, DialogContentText, DialogActions } from '@mui/material';
import ModalHeader from '../../components/ModalHeader';
// LP/GP dashboards are not imported in this mobile wrapper
import { MobileResponsiveLPPrivateMarketsDashboard } from './MobileResponsiveLPPrivateMarketsDashboard';
import { MobileResponsiveGPPrivateMarketsDashboard } from './MobileResponsiveGPPrivateMarketsDashboard';
import { MobileResponsiveFoFPrivateMarketsDashboard } from './MobileResponsiveFoFPrivateMarketsDashboard';
import { TemplateReviewDashboard } from './TemplateReviewDashboard';
import { useExplorer } from './ExplorerContext';
import { useWebSocket } from '../../hooks/useWebSocket';
import { RealTimeNotification } from '../../components/RealTimeNotification';

interface PrivateMarketsExplorerProps {
  userId?: string;
}

export const MobileResponsivePrivateMarketsExplorer: React.FC<PrivateMarketsExplorerProps> = ({ userId }) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const isTablet = useMediaQuery(theme.breakpoints.down('lg'));

  const { user, bundle, isLoading, error } = useExplorer();
  const navigate = useBlockableNavigate();
  const auth = useAuth();
  const [logoutDialogOpen, setLogoutDialogOpen] = useState(false);
  const [currentView, setCurrentView] = useState<'dashboard' | 'review' | 'analytics'>('dashboard');
  const [drawerOpen, setDrawerOpen] = useState(!isMobile); // Auto-open on desktop, closed on mobile
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);

  // WebSocket integration for real-time updates
  const { isConnected, connectionStatus, realTimeData, sendMessage: _sendMessage, reconnect } = useWebSocket(
    user?.role || 'lp',
    user?.id || userId
  );

  const handleMenuClick = (event: MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleMenuClose = () => {
    setAnchorEl(null);
  };

  const handleLogout = () => {
    handleMenuClose();
    setLogoutDialogOpen(true);
  };

  const confirmLogout = async () => {
    setLogoutDialogOpen(false);
    try {
      await auth.logout();
    } finally {
      navigate('/login');
    }
  };

  const cancelLogout = () => {
    setLogoutDialogOpen(false);
  };

  const toggleDrawer = () => {
    setDrawerOpen(!drawerOpen);
  };

  const closeDrawer = () => {
    if (isMobile) {
      setDrawerOpen(false);
    }
  };

  const getRoleDisplayName = (role: string) => {
    switch (role) {
      case 'lp': return 'Limited Partner';
      case 'gp': return 'General Partner';
      case 'fof': return 'Fund of Funds';
      case 'steward': return 'Steward';
      default: return role;
    }
  };

  const getRoleIcon = (role: string) => {
    switch (role) {
      case 'lp': return <People />;
      case 'gp': return <Business />;
      case 'fof': return <Analytics />;
      case 'steward': return <Assessment />;
      default: return <AccountCircle />;
    }
  };

  const renderMainContent = () => {
    if (!user || !bundle) {
      return (
        <Box
          display="flex"
          justifyContent="center"
          alignItems="center"
          minHeight={isMobile ? "60vh" : "400px"}
          sx={{ p: isMobile ? 2 : 3 }}
        >
          <CircularProgress size={isMobile ? 40 : 60} />
        </Box>
      );
    }

    switch (currentView) {
      case 'dashboard':
        if (user.role === 'lp') {
          return <MobileResponsiveLPPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />;
        } else if (user.role === 'gp') {
          return <MobileResponsiveGPPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />;
        } else if (user.role === 'fof') {
          return <MobileResponsiveFoFPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />;
        }
        break;

      case 'review':
        if (user.role === 'steward') {
          return <TemplateReviewDashboard domain="private_markets" />;
        }
        break;

      case 'analytics':
        return (
          <Box sx={{ p: isMobile ? 2 : 3 }}>
            <Typography
              variant={isMobile ? "h5" : "h4"}
              sx={{ mb: isMobile ? 2 : 3 }}
            >
              Advanced Analytics
            </Typography>
            <Typography
              variant={isMobile ? "body1" : "h6"}
              sx={{ mb: 2 }}
            >
              Advanced analytics features would be implemented here based on the current bundle.
            </Typography>
          </Box>
        );
    }

    return (
      <Box sx={{
        p: isMobile ? 2 : 3,
        textAlign: 'center',
        minHeight: isMobile ? "60vh" : "400px",
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center'
      }}>
        <Typography
          variant={isMobile ? "body1" : "h6"}
          color="text.secondary"
        >
          No dashboard available for your role
        </Typography>
      </Box>
    );
  };

  if (isLoading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight="100vh"
        sx={{ p: isMobile ? 2 : 3 }}
      >
        <CircularProgress size={isMobile ? 40 : 60} />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert
        severity="error"
        sx={{
          m: isMobile ? 2 : 3,
          fontSize: isMobile ? '0.875rem' : '1rem'
        }}
      >
        {error}
      </Alert>
    );
  }

  const drawerWidth = isMobile ? 280 : isTablet ? 320 : 360;

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', flexDirection: 'column' }}>
      {/* Mobile App Bar */}
      {isMobile && (
        <AppBar
          position="fixed"
          sx={{
            zIndex: theme.zIndex.drawer + 1,
            backgroundColor: 'background.paper',
            color: 'text.primary',
            borderBottom: 1,
            borderColor: 'divider',
          }}
        >
          <Toolbar sx={{ minHeight: '56px', px: 2 }}>
            <IconButton
              edge="start"
              color="inherit"
              aria-label="menu"
              onClick={toggleDrawer}
              sx={{ mr: 1 }}
            >
              <MenuIcon />
            </IconButton>

            <Typography
              variant="h6"
              sx={{
                flexGrow: 1,
                fontWeight: 'bold',
                fontSize: '1.1rem'
              }}
            >
              {bundle ? bundle.name : 'Private Markets'}
            </Typography>

            {/* Connection Status - Mobile */}
            <Tooltip title={`Connection: ${connectionStatus}`}>
              <IconButton
                size="small"
                onClick={reconnect}
                color={isConnected ? 'success' : 'error'}
              >
                {isConnected ? <Wifi /> : <WifiOff />}
              </IconButton>
            </Tooltip>
          </Toolbar>
        </AppBar>
      )}

      {/* Navigation Drawer */}
      <Drawer
        variant={isMobile ? "temporary" : "permanent"}
        open={drawerOpen}
        onClose={closeDrawer}
        sx={{
          width: drawerWidth,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: drawerWidth,
            boxSizing: 'border-box',
            backgroundColor: 'background.paper',
            borderRight: 1,
            borderColor: 'divider',
          },
        }}
        ModalProps={{
          keepMounted: true, // Better mobile performance
        }}
      >
        <Box sx={{
          p: isMobile ? 2 : 3,
          borderBottom: 1,
          borderColor: 'divider',
          backgroundColor: 'background.default'
        }}>
          {/* Close button for mobile */}
          {isMobile && (
            <Box sx={{ display: 'flex', justifyContent: 'flex-end', mb: 1 }}>
              <IconButton onClick={closeDrawer} size="small">
                <CloseIcon />
              </IconButton>
            </Box>
          )}

          <Typography
            variant={isMobile ? "h6" : "h5"}
            sx={{
              mb: 2,
              fontWeight: 'bold',
              fontSize: isMobile ? '1.1rem' : '1.25rem'
            }}
          >
            Private Markets Explorer
          </Typography>

          {user && (
            <Box display="flex" alignItems="center" gap={1} sx={{ mb: 2 }}>
              <Avatar sx={{
                width: isMobile ? 32 : 40,
                height: isMobile ? 32 : 40
              }}>
                {getRoleIcon(user.role)}
              </Avatar>
              <Box sx={{ minWidth: 0, flex: 1 }}>
                <Typography
                  variant="body2"
                  fontWeight="bold"
                  sx={{
                    fontSize: isMobile ? '0.875rem' : '1rem',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap'
                  }}
                >
                  {user.name}
                </Typography>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{
                    fontSize: isMobile ? '0.75rem' : '0.875rem',
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap'
                  }}
                >
                  {getRoleDisplayName(user.role)}
                </Typography>
              </Box>
            </Box>
          )}

          {/* WebSocket Connection Status */}
          {!isMobile && (
            <Box sx={{
              mt: 2,
              display: 'flex',
              alignItems: 'center',
              gap: 1,
              p: 1,
              borderRadius: 1,
              backgroundColor: isConnected ? 'success.light' : 'error.light',
              opacity: 0.8
            }}>
              <Tooltip title={`Connection: ${connectionStatus}`}>
                <IconButton
                  size="small"
                  onClick={reconnect}
                  color={isConnected ? 'success' : 'error'}
                >
                  {isConnected ? <Wifi /> : <WifiOff />}
                </IconButton>
              </Tooltip>
              <Box sx={{ minWidth: 0, flex: 1 }}>
                <Typography
                  variant="caption"
                  color="text.secondary"
                  sx={{
                    display: 'block',
                    fontWeight: 'bold',
                    fontSize: '0.75rem'
                  }}
                >
                  Live Updates
                </Typography>
                <Typography
                  variant="caption"
                  color={isConnected ? 'success.main' : 'error.main'}
                  sx={{
                    display: 'block',
                    fontWeight: 'bold',
                    fontSize: '0.75rem'
                  }}
                >
                  {isConnected ? 'Connected' : connectionStatus === 'connecting' ? 'Connecting...' : 'Disconnected'}
                </Typography>
              </Box>
              {!isConnected && (
                <IconButton size="small" onClick={reconnect} color="primary">
                  <Refresh />
                </IconButton>
              )}
            </Box>
          )}

          {/* Real-time Data Summary */}
          {Object.keys(realTimeData).length > 0 && (
            <Box sx={{ mt: 1 }}>
              <Typography
                variant="caption"
                color="text.secondary"
                sx={{ fontSize: '0.75rem' }}
              >
                Live updates: {Object.keys(realTimeData).length} funds
              </Typography>
            </Box>
          )}
        </Box>

        <List sx={{ py: 0 }}>
          <ListItem
            button
            selected={currentView === 'dashboard'}
            onClick={() => {
              setCurrentView('dashboard');
              closeDrawer();
            }}
            sx={{
              py: isMobile ? 2 : 1.5,
              px: isMobile ? 3 : 2,
              minHeight: isMobile ? '56px' : '48px',
            }}
          >
            <ListItemIcon sx={{
              minWidth: isMobile ? '40px' : '36px',
              color: currentView === 'dashboard' ? 'primary.main' : 'inherit'
            }}>
              <Dashboard />
            </ListItemIcon>
            <ListItemText
              primary="Dashboard"
              primaryTypographyProps={{
                fontSize: isMobile ? '1rem' : '0.875rem',
                fontWeight: currentView === 'dashboard' ? 'bold' : 'normal',
              }}
            />
          </ListItem>

          {user?.role === 'steward' && (
            <ListItem
              button
              selected={currentView === 'review'}
              onClick={() => {
                setCurrentView('review');
                closeDrawer();
              }}
              sx={{
                py: isMobile ? 2 : 1.5,
                px: isMobile ? 3 : 2,
                minHeight: isMobile ? '56px' : '48px',
              }}
            >
              <ListItemIcon sx={{
                minWidth: isMobile ? '40px' : '36px',
                color: currentView === 'review' ? 'primary.main' : 'inherit'
              }}>
                <Assessment />
              </ListItemIcon>
              <ListItemText
                primary="Template Review"
                primaryTypographyProps={{
                  fontSize: isMobile ? '1rem' : '0.875rem',
                  fontWeight: currentView === 'review' ? 'bold' : 'normal',
                }}
              />
            </ListItem>
          )}

          <ListItem
            button
            selected={currentView === 'analytics'}
            onClick={() => {
              setCurrentView('analytics');
              closeDrawer();
            }}
            sx={{
              py: isMobile ? 2 : 1.5,
              px: isMobile ? 3 : 2,
              minHeight: isMobile ? '56px' : '48px',
            }}
          >
            <ListItemIcon sx={{
              minWidth: isMobile ? '40px' : '36px',
              color: currentView === 'analytics' ? 'primary.main' : 'inherit'
            }}>
              <Compare />
            </ListItemIcon>
            <ListItemText
              primary="Analytics"
              primaryTypographyProps={{
                fontSize: isMobile ? '1rem' : '0.875rem',
                fontWeight: currentView === 'analytics' ? 'bold' : 'normal',
              }}
            />
          </ListItem>
        </List>

        <Divider />

        {/* Bundle Info */}
        <Box sx={{ p: isMobile ? 2 : 3, mt: 'auto' }}>
          <Typography
            variant="subtitle2"
            gutterBottom
            sx={{ fontSize: isMobile ? '0.875rem' : '0.75rem' }}
          >
            Active Bundle
          </Typography>
          {bundle && (
            <Chip
              label={bundle.name}
              size={isMobile ? "small" : "medium"}
              color="primary"
              sx={{
                mb: 1,
                fontSize: isMobile ? '0.75rem' : '0.875rem',
                height: isMobile ? '24px' : '32px'
              }}
            />
          )}

          <Typography
            variant="subtitle2"
            gutterBottom
            sx={{
              mt: 2,
              fontSize: isMobile ? '0.875rem' : '0.75rem'
            }}
          >
            Bundle Version
          </Typography>
          {bundle && (
            <Typography
              variant="caption"
              color="text.secondary"
              sx={{ fontSize: isMobile ? '0.75rem' : '0.875rem' }}
            >
              v{bundle.version}
            </Typography>
          )}
        </Box>
      </Drawer>

      {/* Main Content */}
      <Box
        sx={{
          flexGrow: 1,
          bgcolor: 'grey.50',
          marginTop: isMobile ? '56px' : 0, // Account for fixed app bar on mobile
          minHeight: isMobile ? 'calc(100vh - 56px)' : '100vh',
        }}
      >
        {/* Desktop Top Bar */}
        {!isMobile && (
          <Box
            sx={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              p: 3,
              bgcolor: 'white',
              borderBottom: 1,
              borderColor: 'divider'
            }}
          >
            <Typography variant="h4" sx={{ fontWeight: 'bold' }}>
              {bundle ? bundle.name : 'Private Markets'}
            </Typography>

            <Box display="flex" alignItems="center" gap={2}>
              <Button
                variant="outlined"
                startIcon={<Settings />}
                onClick={handleMenuClick}
                size={isTablet ? "small" : "medium"}
              >
                Settings
              </Button>

              <Menu
                anchorEl={anchorEl}
                open={Boolean(anchorEl)}
                onClose={handleMenuClose}
              >
                <MenuItem onClick={handleMenuClose}>
                  <ListItemIcon>
                    <AccountCircle fontSize="small" />
                  </ListItemIcon>
                  Profile
                </MenuItem>
                <MenuItem onClick={handleMenuClose}>
                  <ListItemIcon>
                    <Settings fontSize="small" />
                  </ListItemIcon>
                  Preferences
                </MenuItem>
                <Divider />
                <MenuItem onClick={handleLogout}>
                  <ListItemIcon>
                    <ExitToApp fontSize="small" />
                  </ListItemIcon>
                    Logout
                </MenuItem>
              </Menu>
            </Box>
          </Box>
        )}

        {/* Main Content Area */}
        <Box sx={{
          p: isMobile ? 1 : 0,
          overflow: 'auto',
          height: '100%'
        }}>
          {renderMainContent()}
        </Box>
      </Box>

      {/* Real-time Notifications */}
      <RealTimeNotification
        realTimeData={realTimeData}
        isConnected={isConnected}
      />
      <Dialog open={logoutDialogOpen} onClose={cancelLogout}>
    <ModalHeader title="Confirm Logout" onClose={cancelLogout} />
        <DialogContent>
          <DialogContentText>
            Are you sure you want to log out? This will clear your local session.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={cancelLogout}>Cancel</Button>
          <Button onClick={confirmLogout} color="primary" autoFocus>Logout</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
