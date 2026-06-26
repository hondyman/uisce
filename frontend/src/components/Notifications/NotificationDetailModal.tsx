import React, { useEffect, useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Typography,
  Chip,
  Box,
  Divider,
  Stack,
  IconButton,
  Grid,
  useTheme,
  alpha
} from '@mui/material';
import {
  Close as CloseIcon,
  Email as EmailIcon,
  Sms as SmsIcon,
  Chat as ChatIcon,
  Group as GroupIcon,
  NotificationsActive as NotificationsActiveIcon,
  Notifications as NotificationsIcon,
  Check as CheckIcon,
  Drafts as DraftsIcon,
  Error as ErrorIcon
} from '@mui/icons-material';

interface NotificationDetailModalProps {
  notification: {
    id: string;
    subject: string;
    body: string;
    channel: string;
    priority: string;
    sent_at: string;
    opened_at?: string | null;
    action_taken?: string | null;
    action_taken_at?: string | null;
  } | null;
  isOpen: boolean;
  onClose: () => void;
  onMarkAsRead: () => void;
  onApprove: () => void;
  onReject: () => void;
}

export const NotificationDetailModal: React.FC<NotificationDetailModalProps> = ({
  notification,
  isOpen,
  onClose,
  onMarkAsRead,
  onApprove,
  onReject,
}) => {
  const theme = useTheme();

  if (!notification) return null;

  const getChannelIcon = (channel: string) => {
    switch (channel) {
      case 'email': return <EmailIcon />;
      case 'sms': return <SmsIcon />;
      case 'slack': return <ChatIcon />;
      case 'teams': return <GroupIcon />;
      case 'push': return <NotificationsActiveIcon />;
      default: return <NotificationsIcon />;
    }
  };

  const getPriorityColor = (priority: string) => {
    switch (priority) {
      case 'urgent': return 'error';
      case 'high': return 'warning';
      case 'normal': return 'info';
      case 'low': return 'default';
      default: return 'default';
    }
  };

  const formatDateTime = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    });
  };

  return (
    <Dialog 
      open={isOpen} 
      onClose={onClose} 
      maxWidth="sm" 
      fullWidth
      PaperProps={{
        elevation: 24,
        sx: { borderRadius: 3 }
      }}
    >
      {/* Header */}
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', p: 3, pb: 2 }}>
        <Stack direction="row" spacing={2} alignItems="center" overflow="hidden">
          <Box 
            sx={{ 
              width: 48, 
              height: 48, 
              borderRadius: 2, 
              bgcolor: alpha(theme.palette.primary.main, 0.1),
              color: 'primary.main',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center' 
            }}
          >
            {getChannelIcon(notification.channel)}
          </Box>
          <Box overflow="hidden">
            <Typography variant="h6" noWrap>
              {notification.subject}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              via {notification.channel} • {formatDateTime(notification.sent_at)}
            </Typography>
          </Box>
        </Stack>
        <IconButton onClick={onClose} size="small">
          <CloseIcon />
        </IconButton>
      </DialogTitle>

      <Divider />

      <DialogContent sx={{ p: 3 }}>
        <Stack spacing={2} direction="row" mb={3}>
          {!notification.opened_at && (
            <Chip 
              icon={<DraftsIcon />} 
              label="New" 
              color="primary" 
              size="small" 
              variant="outlined" 
            />
          )}
          <Chip 
            label={`${notification.priority} Priority`} 
            color={getPriorityColor(notification.priority) as any} 
            size="small" 
            sx={{ textTransform: 'capitalize' }}
          />
        </Stack>

        <Typography paragraph variant="body1" color="text.primary" sx={{ lineHeight: 1.7, whiteSpace: 'pre-line' }}>
          {notification.body}
        </Typography>

        <Box sx={{ mt: 4, bgcolor: 'action.hover', p: 3, borderRadius: 2 }}>
           <Typography variant="subtitle2" color="text.secondary" gutterBottom textTransform="uppercase" letterSpacing={1}>
             Notification Details
           </Typography>
           <Grid container spacing={2} mt={0.5}>
             <Grid item xs={6}>
               <Typography variant="caption" color="text.secondary" display="block">Opened</Typography>
               <Typography variant="body2" fontWeight="medium">
                 {notification.opened_at ? formatDateTime(notification.opened_at) : 'Not yet opened'}
               </Typography>
             </Grid>
             <Grid item xs={6}>
               <Typography variant="caption" color="text.secondary" display="block">Action Status</Typography>
               <Stack direction="row" spacing={1} alignItems="center">
                 <Box 
                   sx={{ 
                     width: 8, 
                     height: 8, 
                     borderRadius: '50%', 
                     bgcolor: notification.action_taken ? 'success.main' : 'text.disabled'
                   }} 
                 />
                 <Typography variant="body2" fontWeight="medium" textTransform="capitalize">
                   {notification.action_taken || 'Pending'}
                 </Typography>
               </Stack>
             </Grid>
           </Grid>
        </Box>
      </DialogContent>

      <Divider />

      <DialogActions sx={{ p: 3, bgcolor: 'background.default' }}>
        {!notification.opened_at && (
           <Button 
             startIcon={<DraftsIcon />} 
             onClick={onMarkAsRead}
             color="inherit"
             sx={{ mr: 'auto' }}
           >
             Mark as Read
           </Button>
        )}
        
        {notification.priority === 'urgent' && !notification.action_taken ? (
          <>
            <Button onClick={onReject} variant="outlined" color="error">
              Reject
            </Button>
            <Button onClick={onApprove} variant="contained" color="primary" autoFocus>
              Approve
            </Button>
          </>
        ) : (
          <Button onClick={onClose} variant="outlined" color="inherit">
            Close
          </Button>
        )}
      </DialogActions>
    </Dialog>
  );
};
