import React, { useState, useEffect } from 'react';
import {
  Snackbar,
  Alert,
  AlertTitle,
  IconButton,
  Box,
  Typography
} from '@mui/material';
import { Close as CloseIcon } from '@mui/icons-material';
import { RealTimeData } from '../hooks/useWebSocket';

interface RealTimeNotificationProps {
  realTimeData: RealTimeData;
  isConnected: boolean;
}

interface Notification {
  id: string;
  type: 'fund_update' | 'connection' | 'error';
  title: string;
  message: string;
  severity: 'success' | 'info' | 'warning' | 'error';
  timestamp: Date;
}

export const RealTimeNotification: React.FC<RealTimeNotificationProps> = ({
  realTimeData,
  isConnected
}) => {
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [currentNotification, setCurrentNotification] = useState<Notification | null>(null);

  // Track previous realTimeData to detect changes
  const [prevRealTimeData, setPrevRealTimeData] = useState<RealTimeData>({});

  useEffect(() => {
    // Detect new or updated fund data
    Object.keys(realTimeData).forEach(fundId => {
      const currentData = realTimeData[fundId];
      const prevData = (prevRealTimeData as any)[fundId];

      if (!prevData || currentData.lastUpdate !== prevData.lastUpdate) {
        // New or updated fund data
        const notification: Notification = {
          id: `fund-${fundId}-${Date.now()}`,
          type: 'fund_update',
          title: 'Fund Data Updated',
          message: `Real-time metrics updated for fund ${fundId}`,
          severity: 'info',
          timestamp: new Date()
        };

        setNotifications(prev => [notification, ...prev.slice(0, 9)]); // Keep last 10
        setCurrentNotification(notification);
      }
    });

    // intentionally not listing setPrevRealTimeData in deps
    // to avoid unnecessary re-renders; we still update local ref below
    setPrevRealTimeData(realTimeData);
  }, [realTimeData]);

  useEffect(() => {
    // Connection status notifications
    if (!isConnected) {
      const notification: Notification = {
        id: `connection-${Date.now()}`,
        type: 'connection',
        title: 'Connection Lost',
        message: 'Real-time updates are currently unavailable',
        severity: 'warning',
        timestamp: new Date()
      };

      setNotifications(prev => [notification, ...prev.slice(0, 9)]);
      setCurrentNotification(notification);
    }
  }, [isConnected]);

  const handleClose = () => {
    setCurrentNotification(null);
  };

  // unused click handler removed

  return (
    <>
      {/* Current notification snackbar */}
      <Snackbar
        open={!!currentNotification}
        autoHideDuration={6000}
        onClose={handleClose}
        anchorOrigin={{ vertical: 'top', horizontal: 'right' }}
      >
        <Alert
          onClose={handleClose}
          severity={currentNotification?.severity || 'info'}
          sx={{ width: '100%' }}
          action={
            <IconButton
              size="small"
              aria-label="close"
              color="inherit"
              onClick={handleClose}
            >
              <CloseIcon fontSize="small" />
            </IconButton>
          }
        >
          <AlertTitle>{currentNotification?.title}</AlertTitle>
          {currentNotification?.message}
        </Alert>
      </Snackbar>

      {/* Notification history panel (can be expanded later) */}
      {notifications.length > 0 && (
        <Box sx={{ position: 'fixed', bottom: 16, right: 16, zIndex: 1000 }}>
          <Typography variant="caption" color="text.secondary">
            {notifications.length} recent updates
          </Typography>
        </Box>
      )}
    </>
  );
};
