import React, { useState, useEffect, useCallback } from 'react';
import { devError } from '../utils/devLogger';
import { Box, Typography, List, ListItem, ListItemText, ListItemAvatar, Avatar, Chip, IconButton, Menu, MenuItem, Badge, Divider, Paper, Button, Dialog as _Dialog, DialogContent as _DialogContent, DialogActions as _DialogActions } from '@mui/material';
// ModalHeader import removed — not used in this component
import { Notifications as NotificationsIcon, MoreVert as MoreVertIcon, CheckCircle as CheckCircleIcon, Error as ErrorIcon, Info as InfoIcon, Warning as WarningIcon, Close as CloseIcon } from '@mui/icons-material';
import { useWebSocket } from '../hooks/useWebSocket';
import { useNotificationAPI } from '../hooks/useNotificationAPI';

interface EngagementNotification {
  id: string;
  user_id: string;
  type: string;
  title: string;
  message: string;
  rich_content?: any;
  priority: number;
  channels: string[];
  status: string;
  scheduled_at?: string;
  sent_at?: string;
  read_at?: string;
  clicked_at?: string;
  dismissed_at?: string;
  expires_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  engagement_score?: number;
  user_segment?: string;
  ab_test_variant?: string;
  template_id?: string;
  personalization?: any;
  actions?: NotificationAction[];
  cta?: CallToAction;
}

interface NotificationAction {
  id: string;
  label: string;
  type: string;
  url?: string;
  payload?: any;
  primary: boolean;
}

interface CallToAction {
  text: string;
  url: string;
  type: string;
  tracking?: string;
}

interface EngagementNotificationCenterProps {
  userId: string;
  onNotificationClick?: (notification: EngagementNotification) => void;
  onEngagementEvent?: (event: any) => void;
}

export const EngagementNotificationCenter: React.FC<EngagementNotificationCenterProps> = ({
  userId,
  onNotificationClick
}) => {
  const [notifications, setNotifications] = useState<EngagementNotification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null);
  const [_selectedNotification, _setSelectedNotification] = useState<EngagementNotification | null>(null);
  const [engagementAnalytics] = useState<any>(null);

  const { getUserNotifications, markAsRead, trackEngagement } = useNotificationAPI();

  // WebSocket messages are handled by the useWebSocket hook internals and
  // application-level events are processed via API polling and hooks.

  const { isConnected, sendMessage } = useWebSocket('notifications', userId);

  const loadNotifications = useCallback(async () => {
    try {
      const data = await getUserNotifications(userId, 20, 0);
      setNotifications(data);
      setUnreadCount(data.filter(n => !n.read_at).length);
    } catch (error) {
      try { devError('Failed to load notifications:', error); } catch {}
    }
  }, [userId, getUserNotifications]);

  useEffect(() => {
    loadNotifications();
  }, [loadNotifications]);

  const handleNotificationClick = async (notification: EngagementNotification) => {
    if (!notification.read_at) {
      try {
        await markAsRead(notification.id);
        setNotifications(prev =>
          prev.map(n =>
            n.id === notification.id
              ? { ...n, read_at: new Date().toISOString(), status: 'read' }
              : n
          )
        );
  setUnreadCount(prev => Math.max(0, prev - 1));
        // Track engagement event
        await trackEngagement({
          notification_id: notification.id,
          user_id: userId,
          event_type: 'read',
          event_timestamp: new Date().toISOString()
        });

        // Send WebSocket message
        sendMessage({
          type: 'mark_read',
          notification_id: notification.id
        });
      } catch (error) {
        try { devError('Failed to mark notification as read:', error); } catch {}
      }
    }

    onNotificationClick?.(notification);
  };

  const handleActionClick = async (notification: EngagementNotification, action: NotificationAction) => {
    // Track engagement event
    await trackEngagement({
      notification_id: notification.id,
      user_id: userId,
      event_type: 'clicked',
      event_timestamp: new Date().toISOString(),
      action_id: action.id,
      action_type: action.type
    });

    // Send WebSocket message
    sendMessage({
      type: 'engagement_event',
      notification_id: notification.id,
      event_type: 'clicked',
      action: action
    });

    // Handle action
    if (action.url) {
      window.open(action.url, action.type === 'external' ? '_blank' : '_self');
    }
  };

  const handleDismiss = async (notification: EngagementNotification) => {
    try {
      await trackEngagement({
        notification_id: notification.id,
        user_id: userId,
        event_type: 'dismissed',
        event_timestamp: new Date().toISOString()
      });

      sendMessage({
        type: 'engagement_event',
        notification_id: notification.id,
        event_type: 'dismissed'
      });

      setNotifications(prev => prev.filter(n => n.id !== notification.id));
      setUnreadCount(prev => Math.max(0, prev - 1));
    } catch (error) {
      try { devError('Failed to dismiss notification:', error); } catch {}
    }
  };

  const getNotificationIcon = (type: string) => {
    switch (type) {
      case 'alert':
        return <ErrorIcon color="error" />;
      case 'warning':
        return <WarningIcon color="warning" />;
      case 'info':
        return <InfoIcon color="info" />;
      case 'success':
        return <CheckCircleIcon color="success" />;
      default:
        return <NotificationsIcon />;
    }
  };

  const getPriorityColor = (priority: number) => {
    switch (priority) {
      case 4:
        return 'error';
      case 3:
        return 'warning';
      case 2:
        return 'info';
      default:
        return 'default';
    }
  };

  return (
    <Paper elevation={2} sx={{ maxWidth: 400, maxHeight: 600, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ p: 2, borderBottom: 1, borderColor: 'divider' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <Typography variant="h6">
            Notifications
            {unreadCount > 0 && (
              <Badge badgeContent={unreadCount} color="error" sx={{ ml: 1 }} />
            )}
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Chip
              size="small"
              label={isConnected ? 'Live' : 'Offline'}
              color={isConnected ? 'success' : 'default'}
              variant="outlined"
            />
            <IconButton size="small" onClick={(e) => setAnchorEl(e.currentTarget)}>
              <MoreVertIcon />
            </IconButton>
          </Box>
        </Box>
      </Box>

      <List sx={{ flex: 1, overflow: 'auto' }}>
        {notifications.length === 0 ? (
          <ListItem>
            <ListItemText
              primary="No notifications"
              secondary="You're all caught up!"
              sx={{ textAlign: 'center', py: 4 }}
            />
          </ListItem>
        ) : (
          notifications.map((notification) => (
            <>
              <ListItem
                sx={{
                  cursor: 'pointer',
                  bgcolor: !notification.read_at ? 'action.hover' : 'inherit',
                  '&:hover': { bgcolor: 'action.selected' }
                }}
                onClick={() => handleNotificationClick(notification)}
                secondaryAction={
                  <IconButton
                    edge="end"
                    onClick={(e) => {
                      e.stopPropagation();
                      handleDismiss(notification);
                    }}
                  >
                    <CloseIcon />
                  </IconButton>
                }
              >
                <ListItemAvatar>
                  <Avatar sx={{ bgcolor: 'primary.main' }}>
                    {getNotificationIcon(notification.type)}
                  </Avatar>
                </ListItemAvatar>
                <ListItemText
                  primary={
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <Typography variant="subtitle2" component="span" sx={{ flex: 1 }}>
                        {notification.title}
                      </Typography>
                      <Chip
                        size="small"
                        label={`P${notification.priority}`}
                        color={getPriorityColor(notification.priority)}
                        variant="outlined"
                      />
                    </Box>
                  }
                  secondary={
                    <Box>
                      <Typography variant="body2" color="text.secondary" component="div">
                        {notification.message}
                      </Typography>
                      <Typography variant="caption" color="text.disabled" component="span">
                        {new Date(notification.created_at).toLocaleString()}
                      </Typography>
                    </Box>
                  }
                  primaryTypographyProps={{ component: 'div' }}
                  secondaryTypographyProps={{ component: 'div' }}
                />
              </ListItem>

              {notification.actions && notification.actions.length > 0 && (
                <Box sx={{ px: 2, pb: 1 }}>
                  <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                    {notification.actions.map((action) => (
                      <Button
                        key={action.id}
                        size="small"
                        variant={action.primary ? 'contained' : 'outlined'}
                        onClick={(e) => {
                          e.stopPropagation();
                          handleActionClick(notification, action);
                        }}
                      >
                        {action.label}
                      </Button>
                    ))}
                  </Box>
                </Box>
              )}

              <Divider />
            </>
          ))
        )}
      </List>

      {engagementAnalytics && (
        <Box sx={{ p: 2, borderTop: 1, borderColor: 'divider' }}>
          <Typography variant="subtitle2" gutterBottom>
            Engagement Analytics
          </Typography>
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
            <Chip size="small" label={`Open Rate: ${(engagementAnalytics.avg_open_rate * 100).toFixed(1)}%`} />
            <Chip size="small" label={`Click Rate: ${(engagementAnalytics.avg_click_rate * 100).toFixed(1)}%`} />
            <Chip size="small" label={`Total Sent: ${engagementAnalytics.total_sent}`} />
          </Box>
        </Box>
      )}

      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={() => setAnchorEl(null)}
      >
        <MenuItem onClick={() => { setAnchorEl(null); loadNotifications(); }}>
          Refresh
        </MenuItem>
        <MenuItem onClick={() => { setAnchorEl(null); /* Mark all as read */ }}>
          Mark All as Read
        </MenuItem>
        <MenuItem onClick={() => { setAnchorEl(null); /* Clear all */ }}>
          Clear All
        </MenuItem>
      </Menu>
    </Paper>
  );
};
