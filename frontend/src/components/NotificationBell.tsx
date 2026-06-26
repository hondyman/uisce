import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  IconButton,
  Badge,
  Menu,
  MenuItem,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  Divider,
  Button,
  Chip,
  Stack,
  Avatar,
} from '@mui/material';
import {
  Notifications as NotificationsIcon,
  NotificationsActive as NotificationsActiveIcon,
  Info as InfoIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  CheckCircle as SuccessIcon,
  Close as CloseIcon,
} from '@mui/icons-material';

// ============================================================================
// TYPES
// ============================================================================

export interface Notification {
  id: string;
  category: string;
  title: string;
  message: string;
  link_url?: string;
  link_text?: string;
  priority: 'low' | 'normal' | 'high' | 'urgent';
  created_at: string;
  read_at?: string;
}

interface NotificationBellProps {
  notifications: Notification[];
  onMarkAsRead: (id: string) => void;
  onMarkAllAsRead: () => void;
  onRefresh: () => void;
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

const getCategoryIcon = (category: string) => {
  switch (category) {
    case 'alert':
      return <WarningIcon color="warning" />;
    case 'error':
      return <ErrorIcon color="error" />;
    case 'approval':
      return <InfoIcon color="info" />;
    case 'report':
      return <SuccessIcon color="success" />;
    default:
      return <InfoIcon color="primary" />;
  }
};

const getPriorityColor = (priority: string) => {
  switch (priority) {
    case 'urgent':
      return 'error';
    case 'high':
      return 'warning';
    case 'normal':
      return 'info';
    case 'low':
      return 'default';
    default:
      return 'default';
  }
};

const formatTimeAgo = (dateString: string): string => {
  const date = new Date(dateString);
  const now = new Date();
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000);

  if (seconds < 60) return 'Just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
  return date.toLocaleDateString();
};

// ============================================================================
// NOTIFICATION BELL COMPONENT
// ============================================================================

export const NotificationBell: React.FC<NotificationBellProps> = ({
  notifications,
  onMarkAsRead,
  onMarkAllAsRead,
  onRefresh,
}) => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const open = Boolean(anchorEl);

  const unreadCount = notifications.filter((n) => !n.read_at).length;
  const hasUrgent = notifications.some((n) => n.priority === 'urgent' && !n.read_at);

  const handleClick = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget);
    onRefresh();
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleNotificationClick = (notification: Notification) => {
    onMarkAsRead(notification.id);
    if (notification.link_url) {
      window.location.href = notification.link_url;
    }
  };

  return (
    <>
      <IconButton onClick={handleClick} color="inherit">
        <Badge badgeContent={unreadCount} color={hasUrgent ? 'error' : 'primary'}>
          {hasUrgent ? <NotificationsActiveIcon /> : <NotificationsIcon />}
        </Badge>
      </IconButton>

      <Menu
        anchorEl={anchorEl}
        open={open}
        onClose={handleClose}
        PaperProps={{
          sx: {
            width: 400,
            maxHeight: 500,
          },
        }}
        transformOrigin={{ horizontal: 'right', vertical: 'top' }}
        anchorOrigin={{ horizontal: 'right', vertical: 'bottom' }}
      >
        {/* Header */}
        <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
          <Stack direction="row" justifyContent="space-between" alignItems="center">
            <Typography variant="h6">Notifications</Typography>
            {unreadCount > 0 && (
              <Button size="small" onClick={onMarkAllAsRead}>
                Mark all as read
              </Button>
            )}
          </Stack>
        </Box>

        {/* Notifications List */}
        <Box sx={{ maxHeight: 400, overflow: 'auto' }}>
          {notifications.length === 0 ? (
            <Box sx={{ p: 4, textAlign: 'center' }}>
              <NotificationsIcon sx={{ fontSize: 48, color: 'text.disabled', mb: 1 }} />
              <Typography variant="body2" color="text.secondary">
                No notifications
              </Typography>
            </Box>
          ) : (
            <List sx={{ p: 0 }}>
              {notifications.map((notification, index) => (
                <React.Fragment key={notification.id}>
                  {index > 0 && <Divider />}
                  <ListItem
                    button
                    onClick={() => handleNotificationClick(notification)}
                    sx={{
                      bgcolor: notification.read_at ? 'transparent' : 'action.hover',
                      '&:hover': {
                        bgcolor: 'action.selected',
                      },
                    }}
                  >
                    <Stack direction="row" spacing={2} sx={{ width: '100%' }}>
                      <Avatar sx={{ bgcolor: 'transparent' }}>
                        {getCategoryIcon(notification.category)}
                      </Avatar>
                      <Box sx={{ flex: 1, minWidth: 0 }}>
                        <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                          <Typography variant="subtitle2" noWrap sx={{ fontWeight: notification.read_at ? 'normal' : 'bold' }}>
                            {notification.title}
                          </Typography>
                          <Chip
                            label={notification.priority}
                            size="small"
                            color={getPriorityColor(notification.priority) as any}
                            sx={{ ml: 1, height: 20, fontSize: '0.65rem' }}
                          />
                        </Stack>
                        <Typography
                          variant="body2"
                          color="text.secondary"
                          sx={{
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            display: '-webkit-box',
                            WebkitLineClamp: 2,
                            WebkitBoxOrient: 'vertical',
                          }}
                        >
                          {notification.message}
                        </Typography>
                        <Typography variant="caption" color="text.disabled">
                          {formatTimeAgo(notification.created_at)}
                        </Typography>
                      </Box>
                    </Stack>
                  </ListItem>
                </React.Fragment>
              ))}
            </List>
          )}
        </Box>

        {/* Footer */}
        {notifications.length > 0 && (
          <Box sx={{ p: 1.5, borderTop: 1, borderColor: 'divider', textAlign: 'center' }}>
            <Button size="small" fullWidth>
              View all notifications
            </Button>
          </Box>
        )}
      </Menu>
    </>
  );
};

// ============================================================================
// NOTIFICATION CENTER (Full Page)
// ============================================================================

interface NotificationCenterProps {
  notifications: Notification[];
  onMarkAsRead: (id: string) => void;
  onDelete: (id: string) => void;
  onRefresh: () => void;
}

export const NotificationCenter: React.FC<NotificationCenterProps> = ({
  notifications,
  onMarkAsRead,
  onDelete,
  onRefresh,
}) => {
  const [filter, setFilter] = useState<string>('all');

  const filteredNotifications = notifications.filter((n) => {
    if (filter === 'unread') return !n.read_at;
    if (filter === 'urgent') return n.priority === 'urgent';
    return true;
  });

  return (
    <Box>
      {/* Header */}
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Typography variant="h5" fontWeight="bold">
          Notifications
        </Typography>
        <Button variant="outlined" startIcon={<NotificationsIcon />} onClick={onRefresh}>
          Refresh
        </Button>
      </Stack>

      {/* Filters */}
      <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
        <Chip
          label="All"
          onClick={() => setFilter('all')}
          color={filter === 'all' ? 'primary' : 'default'}
          variant={filter === 'all' ? 'filled' : 'outlined'}
        />
        <Chip
          label="Unread"
          onClick={() => setFilter('unread')}
          color={filter === 'unread' ? 'primary' : 'default'}
          variant={filter === 'unread' ? 'filled' : 'outlined'}
        />
        <Chip
          label="Urgent"
          onClick={() => setFilter('urgent')}
          color={filter === 'urgent' ? 'error' : 'default'}
          variant={filter === 'urgent' ? 'filled' : 'outlined'}
        />
      </Stack>

      {/* Notifications List */}
      <Stack spacing={1}>
        {filteredNotifications.map((notification) => (
          <Paper
            key={notification.id}
            sx={{
              p: 2,
              bgcolor: notification.read_at ? 'transparent' : 'action.hover',
              cursor: 'pointer',
              '&:hover': {
                boxShadow: 2,
              },
            }}
            onClick={() => onMarkAsRead(notification.id)}
          >
            <Stack direction="row" spacing={2} alignItems="flex-start">
              {getCategoryIcon(notification.category)}
              <Box sx={{ flex: 1 }}>
                <Stack direction="row" justifyContent="space-between" alignItems="flex-start">
                  <Typography variant="subtitle1" fontWeight={notification.read_at ? 'normal' : 'bold'}>
                    {notification.title}
                  </Typography>
                  <Stack direction="row" spacing={1} alignItems="center">
                    <Chip
                      label={notification.priority}
                      size="small"
                      color={getPriorityColor(notification.priority) as any}
                    />
                    <IconButton size="small" onClick={(e) => { e.stopPropagation(); onDelete(notification.id); }}>
                      <CloseIcon fontSize="small" />
                    </IconButton>
                  </Stack>
                </Stack>
                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                  {notification.message}
                </Typography>
                <Stack direction="row" spacing={2} alignItems="center" sx={{ mt: 1 }}>
                  <Typography variant="caption" color="text.disabled">
                    {formatTimeAgo(notification.created_at)}
                  </Typography>
                  {notification.link_url && (
                    <Button size="small" href={notification.link_url}>
                      {notification.link_text || 'View'}
                    </Button>
                  )}
                </Stack>
              </Box>
            </Stack>
          </Paper>
        ))}
      </Stack>

      {filteredNotifications.length === 0 && (
        <Paper sx={{ p: 6, textAlign: 'center' }}>
          <NotificationsIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
          <Typography variant="h6" color="text.secondary">
            No notifications
          </Typography>
        </Paper>
      )}
    </Box>
  );
};

export default NotificationBell;
