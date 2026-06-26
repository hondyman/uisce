import React, { useState, useEffect } from 'react';
import { devError } from '../utils/devLogger';
import {
  IconButton,
  Badge,
  Popover,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  Typography,
  Divider,
  Box,
  Chip,
  Tooltip,
} from '@mui/material';
import NotificationsIcon from '@mui/icons-material/Notifications';
import HelpOutlineIcon from '@mui/icons-material/HelpOutline';
import { listNotifications, markNotificationAsRead } from '../api';
import { SemanticNotification } from '../types';

function timeAgo(dateString: string) {
  const date = new Date(dateString);
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000);
  let interval = seconds / 31536000;
  if (interval > 1) return `${Math.floor(interval)}y ago`;
  interval = seconds / 2592000;
  if (interval > 1) return `${Math.floor(interval)}mo ago`;
  interval = seconds / 86400;
  if (interval > 1) return `${Math.floor(interval)}d ago`;
  interval = seconds / 3600;
  if (interval > 1) return `${Math.floor(interval)}h ago`;
  interval = seconds / 60;
  if (interval > 1) return `${Math.floor(interval)}m ago`;
  return `${Math.floor(seconds)}s ago`;
}

const eventTypeColors: { [key: string]: 'success' | 'info' | 'warning' | 'error' | 'default' } = {
  certification_updated: 'success',
  claim_granted: 'info',
  lineage_changed: 'warning',
  claim_revoked: 'error',
};

export default function NotificationCenter() {
  const [anchorEl, setAnchorEl] = useState<HTMLButtonElement | null>(null);
  const [notifications, setNotifications] = useState<SemanticNotification[]>([]);

  useEffect(() => {
    listNotifications().then(setNotifications).catch((e) => { devError(e); });
  }, []);

  const handleClick = (event: React.MouseEvent<HTMLButtonElement>) => {
    setAnchorEl(event.currentTarget);
  };

  const handleClose = () => {
    setAnchorEl(null);
  };

  const handleMarkAsRead = (notificationId: string) => {
    markNotificationAsRead(notificationId).catch((e) => { devError(e); });
    setNotifications(
      notifications.map((n) => (n.id === notificationId ? { ...n, is_read: true } : n))
    );
  };

  const open = Boolean(anchorEl);
  const unreadCount = notifications.filter((n) => !n.is_read).length;

  return (
    <>
      <IconButton color="inherit" onClick={handleClick}>
        <Badge badgeContent={unreadCount} color="error">
          <NotificationsIcon />
        </Badge>
      </IconButton>
      <Popover
        open={open}
        anchorEl={anchorEl}
        onClose={handleClose}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Box sx={{ width: 360, maxWidth: '100%' }}>
          <Typography sx={{ p: 2 }} variant="h6">Notifications</Typography>
          <Divider />
          <List sx={{ p: 0 }}>
            {notifications.length === 0 ? (
              <ListItem>
                <ListItemText primary="No new notifications." />
              </ListItem>
            ) : (
              notifications.map((n) => (
                <ListItem
                  key={n.id}
                  button
                  onClick={() => handleMarkAsRead(n.id)}
                  sx={{
                    backgroundColor: n.is_read ? 'transparent' : 'action.hover',
                    alignItems: 'flex-start',
                  }}
                >
                  <ListItemText
                    primary={n.message}
                    secondary={
                      <Box component="span" sx={{ display: 'flex', justifyContent: 'space-between', mt: 1 }}>
                        <Box sx={{ display: 'flex', gap: 0.5 }}>
                          <Chip label={n.event_type.replace(/_/g, ' ')} size="small" color={eventTypeColors[n.event_type] || 'default'} />
                          <Chip label={n.status} size="small" variant="outlined" />
                        </Box>
                        <Typography variant="caption" color="text.secondary">{timeAgo(n.timestamp)}</Typography>
                      </Box>
                    }
                  />
                  {n.routing_rule_id && (
                    <ListItemSecondaryAction>
                      <Tooltip title={`Routed via rule: ${n.routing_rule_id}`}>
                        <HelpOutlineIcon sx={{ color: 'text.secondary', fontSize: 16 }} />
                      </Tooltip>
                    </ListItemSecondaryAction>
                  )}
                </ListItem>
              ))
            )}
          </List>
        </Box>
      </Popover>
    </>
  );
}