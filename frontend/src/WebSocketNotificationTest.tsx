import { useState, useCallback } from 'react';
import { Box, Typography, Paper, Button, Alert, Chip, List, ListItem, ListItemText, Divider } from '@mui/material';
import { getWebSocketService } from '../services/WebSocketService';
import { devLog, devError } from '../utils/devLogger';
import { useAuthFetch } from './utils/authFetch';

interface NotificationMessage {
  id: string;
  user_id: string;
  type: string;
  title: string;
  message: string;
  priority: number;
  channels: string[];
  status: string;
  created_at: string;
  updated_at: string;
}

export default function WebSocketNotificationTest() {
  const { authFetch } = useAuthFetch();
  const [isConnected, setIsConnected] = useState(false);
  const [notifications, setNotifications] = useState<NotificationMessage[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<'disconnected' | 'connecting' | 'connected' | 'error'>('disconnected');
  const [errorMessage, setErrorMessage] = useState<string>('');

  const wsUrl = (import.meta.env.VITE_API_BASE_URL || 'http://localhost:8000').replace(/^http/, 'ws') + '/ws';
  const wsService = getWebSocketService(wsUrl);

  const handleNotification = useCallback((data: any) => {
    devLog('Received notification:', data);
    if (data && data.id) {
      setNotifications(prev => [data as NotificationMessage, ...prev.slice(0, 9)]); // Keep last 10
    }
  }, []);

  const connectWebSocket = async () => {
    try {
      setConnectionStatus('connecting');
      setErrorMessage('');

      await wsService.connect();
  setIsConnected(true);
  setConnectionStatus('connected');

      // Create notification subscription using the utility method
      const notificationSubscription = wsService.createNotificationSubscription(
        handleNotification, // callback first
        undefined, // userId optional
        {} // filters
      );

      wsService.subscribe(notificationSubscription);

    } catch (error) {
  devError('WebSocket connection failed:', error);
      setConnectionStatus('error');
      setErrorMessage(error instanceof Error ? error.message : 'Connection failed');
    }
  };

  const disconnectWebSocket = () => {
    wsService.disconnect();
    setIsConnected(false);
    setConnectionStatus('disconnected');
    setNotifications([]);
  };

  const testNotificationAPI = async () => {
    try {
      // Create a test notification via API
  const response = await authFetch('/api/notifications/', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': 'Bearer test-token' // In production, use real auth
        },
        body: JSON.stringify({
          user_id: 'test-user-123',
          type: 'test',
          title: 'WebSocket Test Notification',
          message: 'This is a test notification sent via WebSocket',
          priority: 2,
          channels: ['in_app'],
          engagement_score: 0.9,
          user_segment: 'test_users'
        })
      });

      if (response.ok) {
    devLog('Test notification created successfully');
      } else {
    devError('Failed to create test notification:', response.status);
      }
    } catch (error) {
  devError('Error creating test notification:', error);
    }
  };

  const sendTestNotification = async () => {
    try {
      // Send the notification (this should trigger WebSocket broadcast)
  const response = await authFetch('/api/notifications/test-notification-id/send', {
        method: 'POST',
        headers: {
          'Authorization': 'Bearer test-token'
        }
      });

      if (response.ok) {
        devLog('Test notification sent successfully');
      } else {
        devError('Failed to send test notification:', response.status);
      }
    } catch (error) {
      devError('Error sending test notification:', error);
    }
  };

  const getPriorityColor = (priority: number) => {
    switch (priority) {
      case 1: return 'success';
      case 2: return 'info';
      case 3: return 'warning';
      case 4: return 'error';
      default: return 'default';
    }
  };

  const getPriorityLabel = (priority: number) => {
    switch (priority) {
      case 1: return 'Low';
      case 2: return 'Normal';
      case 3: return 'High';
      case 4: return 'Critical';
      default: return 'Unknown';
    }
  };

  return (
    <Box sx={{ p: 3, maxWidth: 800, mx: 'auto' }}>
      <Typography variant="h4" gutterBottom>
        WebSocket Notification Test
      </Typography>

      <Paper sx={{ p: 2, mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          Connection Status
        </Typography>

        <Box sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
          <Chip
            label={connectionStatus.toUpperCase()}
            color={
              connectionStatus === 'connected' ? 'success' :
              connectionStatus === 'connecting' ? 'warning' :
              connectionStatus === 'error' ? 'error' : 'default'
            }
            variant="outlined"
          />

          {!isConnected ? (
            <Button variant="contained" onClick={connectWebSocket}>
              Connect WebSocket
            </Button>
          ) : (
            <Button variant="outlined" color="error" onClick={disconnectWebSocket}>
              Disconnect
            </Button>
          )}
        </Box>

        {errorMessage && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {errorMessage}
          </Alert>
        )}

        <Typography variant="body2" color="text.secondary">
          WebSocket URL: {wsUrl}
        </Typography>
      </Paper>

      <Paper sx={{ p: 2, mb: 3 }}>
        <Typography variant="h6" gutterBottom>
          Test Actions
        </Typography>

        <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
          <Button
            variant="contained"
            onClick={testNotificationAPI}
            disabled={!isConnected}
          >
            Create Test Notification
          </Button>

          <Button
            variant="outlined"
            onClick={sendTestNotification}
            disabled={!isConnected}
          >
            Send Test Notification
          </Button>
        </Box>
      </Paper>

      <Paper sx={{ p: 2 }}>
        <Typography variant="h6" gutterBottom>
          Received Notifications ({notifications.length})
        </Typography>

        {notifications.length === 0 ? (
          <Typography variant="body2" color="text.secondary">
            No notifications received yet. Create and send a test notification to see real-time updates.
          </Typography>
        ) : (
          <List>
            {notifications.map((notification, index) => (
              <div key={notification.id || index}>
                <ListItem alignItems="flex-start">
                  <ListItemText
                    primary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                        <Typography variant="subtitle1" component="span">
                          {notification.title}
                        </Typography>
                        <Chip
                          label={getPriorityLabel(notification.priority)}
                          size="small"
                          color={getPriorityColor(notification.priority) as any}
                        />
                      </Box>
                    }
                    secondary={
                      <Box>
                        <Typography variant="body2" color="text.primary" sx={{ mb: 1 }} component="div">
                          {notification.message}
                        </Typography>
                        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center' }}>
                          <Typography variant="caption" color="text.secondary" component="span">
                            Type: {notification.type}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" component="span">
                            Status: {notification.status}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" component="span">
                            Channels: {notification.channels?.join(', ')}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" component="span">
                            {new Date(notification.created_at).toLocaleString()}
                          </Typography>
                        </Box>
                      </Box>
                    }
                    primaryTypographyProps={{ component: 'div' }}
                    secondaryTypographyProps={{ component: 'div' }}
                  />
                </ListItem>
                {index < notifications.length - 1 && <Divider />}
              </div>
            ))}
          </List>
        )}
      </Paper>
    </Box>
  );
}
