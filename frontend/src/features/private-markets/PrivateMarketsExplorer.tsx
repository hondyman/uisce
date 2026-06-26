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
  Divider,
  IconButton,
  Tooltip,
  useTheme,
  useMediaQuery,
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
  Refresh
} from '@mui/icons-material';
import { Dialog, DialogContent, DialogContentText, DialogActions } from '@mui/material';
import ModalHeader from '../../components/ModalHeader';
import { LPPrivateMarketsDashboard } from './LPPrivateMarketsDashboard';
import { GPPrivateMarketsDashboard } from './GPPrivateMarketsDashboard';
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

export const PrivateMarketsExplorer: React.FC<PrivateMarketsExplorerProps> = ({ userId }) => {
  const { user, bundle, isLoading, error } = useExplorer();
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const navigate = useBlockableNavigate();
  const auth = useAuth();
  const [logoutDialogOpen, setLogoutDialogOpen] = useState(false);
  const [currentView, setCurrentView] = useState<'dashboard' | 'review' | 'analytics'>('dashboard');
  // drawer state not used in this variant
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
    // Open confirmation dialog
    handleMenuClose();
    setLogoutDialogOpen(true);
  };

  const confirmLogout = async () => {
    setLogoutDialogOpen(false);
    try {
      await auth.logout();
    } finally {
      void navigate('/login');
    }
  };

  const cancelLogout = () => {
    setLogoutDialogOpen(false);
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
        <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
          <CircularProgress />
        </Box>
      );
    }

    switch (currentView) {
      case 'dashboard':
        if (user.role === 'lp') {
          return isMobile ? (
            <MobileResponsiveLPPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />
          ) : (
            <LPPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />
          );
        } else if (user.role === 'gp') {
          return isMobile ? (
            <MobileResponsiveGPPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />
          ) : (
            <GPPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />
          );
        } else if (user.role === 'fof') {
          return isMobile ? (
            <MobileResponsiveFoFPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />
          ) : (
            <LPPrivateMarketsDashboard userId={user.id} realTimeData={realTimeData} />
          );
        }
        break;

      case 'review':
        if (user.role === 'steward') {
          return <TemplateReviewDashboard domain="private_markets" />;
        }
        break;

      case 'analytics':
        return (
          <Box sx={{ p: 3 }}>
            <Typography variant="h4">Advanced Analytics</Typography>
            <Typography variant="body1" sx={{ mt: 2 }}>
              Advanced analytics features would be implemented here based on the current bundle.
            </Typography>
          </Box>
        );
    }

    return (
      <Box sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          No dashboard available for your role
        </Typography>
      </Box>
    );
  };

  if (isLoading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="100vh">
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {error}
      </Alert>
    );
  }

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      {/* Navigation Drawer */}
      <Drawer
        variant="permanent"
        sx={{
          width: 280,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: 280,
            boxSizing: 'border-box',
          },
        }}
      >
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Private Markets Explorer
          </Typography>
          {user && (
            <Box display="flex" alignItems="center" gap={1}>
              <Avatar sx={{ width: 32, height: 32 }}>
                {getRoleIcon(user.role)}
              </Avatar>
              <Box>
                <Typography variant="body2" fontWeight="bold">
                  {user.name}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {getRoleDisplayName(user.role)}
                </Typography>
              </Box>
            </Box>
          )}

          {/* WebSocket Connection Status */}
          <Box sx={{ mt: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
            <Tooltip title={`Connection: ${connectionStatus}`}>
              <IconButton
                size="small"
                onClick={reconnect}
                color={isConnected ? 'success' : 'error'}
              >
                {isConnected ? <Wifi /> : <WifiOff />}
              </IconButton>
            </Tooltip>
            <Box>
              <Typography variant="caption" color="text.secondary">
                Live Updates
              </Typography>
              <Typography
                variant="caption"
                color={isConnected ? 'success.main' : 'error.main'}
                sx={{ display: 'block', fontWeight: 'bold' }}
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

          {/* Real-time Data Summary */}
          {Object.keys(realTimeData).length > 0 && (
            <Box sx={{ mt: 1 }}>
              <Typography variant="caption" color="text.secondary">
                Live updates: {Object.keys(realTimeData).length} funds
              </Typography>
            </Box>
          )}
        </Box>

        <List>
          <ListItem
            button
            selected={currentView === 'dashboard'}
            onClick={() => setCurrentView('dashboard')}
          >
            <ListItemIcon>
              <Dashboard />
            </ListItemIcon>
            <ListItemText primary="Dashboard" />
          </ListItem>

          {user?.role === 'steward' && (
            <ListItem
              button
              selected={currentView === 'review'}
              onClick={() => setCurrentView('review')}
            >
              <ListItemIcon>
                <Assessment />
              </ListItemIcon>
              <ListItemText primary="Template Review" />
            </ListItem>
          )}

          <ListItem
            button
            selected={currentView === 'analytics'}
            onClick={() => setCurrentView('analytics')}
          >
            <ListItemIcon>
              <Compare />
            </ListItemIcon>
            <ListItemText primary="Analytics" />
          </ListItem>
        </List>

        <Divider />

        {/* Bundle Info */}
        <Box sx={{ p: 2 }}>
          <Typography variant="subtitle2" gutterBottom>
            Active Bundle
          </Typography>
          {bundle && (
            <Chip
              label={bundle.name}
              size="small"
              color="primary"
              sx={{ mb: 1 }}
            />
          )}

          <Typography variant="subtitle2" gutterBottom sx={{ mt: 2 }}>
            Bundle Version
          </Typography>
          {bundle && (
            <Typography variant="caption" color="text.secondary">
              v{bundle.version}
            </Typography>
          )}
        </Box>
      </Drawer>

      {/* Main Content */}
      <Box sx={{ flexGrow: 1, bgcolor: 'grey.50' }}>
        {/* Top Bar */}
        <Box
          sx={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            p: 2,
            bgcolor: 'white',
            borderBottom: 1,
            borderColor: 'divider'
          }}
        >
          <Typography variant="h5">
            {bundle ? bundle.name : 'Private Markets'}
          </Typography>

          <Box display="flex" alignItems="center" gap={2}>
            <Button
              variant="outlined"
              startIcon={<Settings />}
              onClick={handleMenuClick}
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

        {/* Main Content Area */}
        {renderMainContent()}
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
