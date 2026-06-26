import React, { useState, useEffect, useMemo, useCallback } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { NotificationDetailModal } from './NotificationDetailModal';
import {
  Box,
  Typography,
  Paper,
  Tabs,
  Tab,
  IconButton,
  Badge,
  Avatar,
  Button,
  Stack,
  Chip,
  Card,
  CardActionArea,
  CardContent,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  ToggleButton,
  ToggleButtonGroup,
  CircularProgress,
  Container,
  Divider,
  useTheme,
  alpha
} from '@mui/material';
import {
  Notifications as NotificationsIcon,
  ExpandMore as ExpandMoreIcon,
  Tune as TuneIcon,
  Email as EmailIcon,
  Sms as SmsIcon,
  Chat as ChatIcon,
  Group as GroupIcon,
  NotificationsActive as NotificationsActiveIcon,
  Check as CheckIcon,
  Close as CloseIcon,
  PriorityHigh as PriorityHighIcon,
  Drafts as DraftsIcon,
  NotificationsOff as NotificationsOffIcon
} from '@mui/icons-material';

// Types
interface NotificationLog {
  id: string;
  tenant_id: string;
  tenant_instance_id: string;
  template_key: string;
  recipient_user_id: string;
  subject: string;
  body: string;
  channel: string;
  status: string;
  priority: string;
  sent_at: string;
  opened_at?: string;
  clicked_at?: string;
  action_taken?: string;
  action_taken_at?: string;
  created_at: string;
  updated_at: string;
}

type FilterTab = 'All' | 'Unread' | 'Archived';
type Channel = 'email' | 'sms' | 'slack' | 'teams' | 'push';
type Priority = 'urgent' | 'high' | 'normal' | 'low';

import { useTenantCompat } from '../../contexts/AccessContext';

export const NotificationCenter: React.FC = () => {
  const { tenant, datasource } = useTenantCompat();
  const { user } = useAuth();
  const userId = user?.id; // Access ID from user object
  const theme = useTheme();
  
  const [notifications, setNotifications] = useState<NotificationLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<FilterTab>('All');
  const [selectedChannels, setSelectedChannels] = useState<Channel[]>([]);
  const [selectedPriorities, setSelectedPriorities] = useState<Priority[]>([]);

  // Modal State
  const [selectedNotification, setSelectedNotification] = useState<NotificationLog | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const fetchNotifications = useCallback(async () => {
    // Dev fallback: if auth context is missing, use these hardcoded values (matching seeded data)
    // ONLY in development mode.
    const isDev = (import.meta as any).env?.DEV;
    const effectiveTenantId = tenant?.id || (isDev ? '910638ba-a459-4a3f-bb2d-78391b0595f6' : '');
    const effectiveDatasourceId = datasource?.id || (isDev ? '982aef38-418f-46dc-acd0-35fe8f3b97b0' : '');
    const effectiveUserId = userId || (isDev ? 'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12' : '');

    if (!effectiveTenantId || !effectiveUserId) {
      if (isDev) console.warn('NotificationCenter: Missing tenant or user context.');
      setLoading(false);
      return;
    }

    try {
      setLoading(true);
      const url = `/api/bp-notifications/logs?tenant_id=${effectiveTenantId}&tenant_instance_id=${effectiveDatasourceId}&user_id=${effectiveUserId}`;
      
      const response = await fetch(url);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();
      setNotifications(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch notifications:', error);
      setNotifications([]);
    } finally {
      setLoading(false);
    }
  }, [tenant?.id, datasource?.id, userId]);

  useEffect(() => {
    fetchNotifications();
  }, [fetchNotifications]);

  // Filter notifications
  const filteredNotifications = useMemo(() => {
    return notifications.filter((notif) => {
      // Tab filter
      if (activeTab === 'Unread' && notif.opened_at) return false;
      if (activeTab === 'Archived' && !notif.opened_at) return false;

      // Channel filter
      if (selectedChannels.length > 0 && !selectedChannels.includes(notif.channel as Channel)) {
        return false;
      }

      // Priority filter
      if (selectedPriorities.length > 0 && !selectedPriorities.includes(notif.priority as Priority)) {
        return false;
      }

      return true;
    });
  }, [notifications, activeTab, selectedChannels, selectedPriorities]);

  const unreadCount = notifications.filter((n) => !n.opened_at).length;

  // Mark as read
  const markAsRead = useCallback(async (id: string) => {
    try {
      await fetch(`/api/bp-notifications/webhook/opened/${id}`, { method: 'POST' });
      setNotifications((prev) =>
        prev.map((n) => (n.id === id ? { ...n, opened_at: new Date().toISOString() } : n))
      );
    } catch (error) {
      console.error('Failed to mark as read:', error);
    }
  }, []);

  // Record action
  const recordAction = useCallback(async (id: string, action: 'approve' | 'reject') => {
    try {
      await fetch(`/api/bp-notifications/webhook/action/${id}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action }),
      });
      setNotifications((prev) =>
        prev.map((n) =>
          n.id === id
            ? { ...n, action_taken: action, action_taken_at: new Date().toISOString() }
            : n
        )
      );
    } catch (error) {
      console.error('Failed to record action:', error);
    }
  }, []);

  // Modal handling
  const handleOpenModal = (notification: NotificationLog) => {
    setSelectedNotification(notification);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setSelectedNotification(null);
  };

  const handleModalMarkAsRead = () => {
    if (selectedNotification) markAsRead(selectedNotification.id);
  };

  const handleModalApprove = () => {
    if (selectedNotification) {
      recordAction(selectedNotification.id, 'approve');
      handleCloseModal();
    }
  };

  const handleModalReject = () => {
    if (selectedNotification) {
      recordAction(selectedNotification.id, 'reject');
      handleCloseModal();
    }
  };

  // Helpers
  const formatRelativeTime = (dateString: string) => {
    const date = new Date(dateString);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    const diffMins = Math.floor(diffMs / 60000);
    const diffHours = Math.floor(diffMs / 3600000);
    const diffDays = Math.floor(diffMs / 86400000);

    if (diffMins < 60) return `${diffMins}m ago`;
    if (diffHours < 24) return `${diffHours}h ago`;
    return `${diffDays}d ago`;
  };

  const getChannelIcon = (channel: string) => {
    switch (channel) {
      case 'email': return <EmailIcon fontSize="small" />;
      case 'sms': return <SmsIcon fontSize="small" />;
      case 'slack': return <ChatIcon fontSize="small" />;
      case 'teams': return <GroupIcon fontSize="small" />;
      case 'push': return <NotificationsActiveIcon fontSize="small" />;
      default: return <NotificationsIcon fontSize="small" />;
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

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'grey.50', py: 4 }}>
      <Container maxWidth="lg">
        {/* Header */}
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 4 }}>
          <Stack direction="row" spacing={2} alignItems="center">
            <Paper
              elevation={3}
              sx={{
                width: 40,
                height: 40,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                bgcolor: 'primary.main',
                color: 'primary.contrastText',
                borderRadius: 2
              }}
            >
              <NotificationsIcon />
            </Paper>
            <Typography variant="h4" fontWeight="bold" color="text.primary">
              Inbox
            </Typography>
          </Stack>

          <Stack direction="row" spacing={2} alignItems="center">
            <IconButton size="large" sx={{ bgcolor: 'background.paper', boxShadow: 1 }}>
              <Badge badgeContent={unreadCount} color="error">
                <NotificationsIcon color="action" />
              </Badge>
            </IconButton>
            <Avatar 
              src="https://lh3.googleusercontent.com/aida-public/AB6AXuDw4WqPbUwbEtC67dPOdS0fvFXvetnIpCIkZCl_z4nQP7UG_m7DkaMsQOVZxRu66CL_oizDDj8HKDi_BgUB_hNcaPlxgs3uDbw6kzpnZ2zXZA0mJ8aOczcby0aILZHlyBGg7kYM745dLf_Ork4Y1QOFZBTeztCjaRgkguJq6x0rCr286Lu0JoGwaQkaxXxY-sZgZfc49bbLlew6MKccA2w0NVJEs73r2Jf_V3bEAFY9NLyX_eeoVS15LChyGLqVpEdSN98iLogP6UdQ"
              sx={{ width: 40, height: 40, boxShadow: 1 }}
            />
          </Stack>
        </Box>

        <Stack spacing={3}>
          {/* Filters */}
          <Paper elevation={0} sx={{ p: 0.5, bgcolor: 'background.paper', borderRadius: 3, border: 1, borderColor: 'divider' }}>
            <Tabs 
              value={activeTab} 
              onChange={(_, v) => setActiveTab(v)}
              variant="fullWidth"
              textColor="primary"
              indicatorColor="primary"
              sx={{
                '& .MuiTab-root': { borderRadius: 2, textTransform: 'none', fontWeight: 600, minHeight: 48 },
                '& .Mui-selected': { bgcolor: alpha(theme.palette.primary.main, 0.08) }
              }}
            >
              <Tab value="All" label="All" />
              <Tab value="Unread" label="Unread" />
              <Tab value="Archived" label="Archived" />
            </Tabs>
          </Paper>

          {/* Advanced Filters */}
          <Accordion elevation={0} sx={{ '&:before': { display: 'none' }, bgcolor: 'transparent' }}>
            <AccordionSummary 
              expandIcon={<ExpandMoreIcon />}
              sx={{ 
                bgcolor: 'background.paper', 
                borderRadius: 3,
                border: 1, 
                borderColor: 'divider',
                px: 3
              }}
            >
              <Stack direction="row" spacing={1} alignItems="center">
                <TuneIcon fontSize="small" color="action" />
                <Typography fontWeight={500}>Advanced Filters</Typography>
              </Stack>
            </AccordionSummary>
            <AccordionDetails sx={{ bgcolor: 'background.paper', borderRadius: 3, mt: 1, border: 1, borderColor: 'divider', p: 3 }}>
              <Stack spacing={3}>
                <Box>
                  <Typography variant="overline" color="text.secondary" fontWeight="bold">Channel</Typography>
                  <Box mt={1}>
                    <ToggleButtonGroup 
                      value={selectedChannels} 
                      onChange={(_, newValues) => setSelectedChannels(newValues)}
                      aria-label="channel filters"
                      size="small"
                      sx={{ flexWrap: 'wrap', gap: 1, border: 'none' }}
                    >
                      {(['email', 'sms', 'slack', 'teams', 'push'] as Channel[]).map((channel) => (
                        <ToggleButton 
                          key={channel} 
                          value={channel}
                          sx={{ 
                            borderRadius: '20px !important', 
                            border: '1px solid !important',
                            borderColor: 'divider',
                            textTransform: 'capitalize',
                            px: 2,
                            py: 0.5,
                            '&.Mui-selected': { 
                              bgcolor: 'primary.main', 
                              color: 'white',
                              '&:hover': { bgcolor: 'primary.dark' }
                            }
                          }}
                        >
                          <Stack direction="row" spacing={1} alignItems="center">
                            {getChannelIcon(channel)}
                            <span>{channel}</span>
                          </Stack>
                        </ToggleButton>
                      ))}
                    </ToggleButtonGroup>
                  </Box>
                </Box>
                
                <Box>
                  <Typography variant="overline" color="text.secondary" fontWeight="bold">Priority</Typography>
                  <Box mt={1}>
                    <ToggleButtonGroup 
                      value={selectedPriorities} 
                      onChange={(_, newValues) => setSelectedPriorities(newValues)}
                      aria-label="priority filters"
                      size="small"
                      sx={{ flexWrap: 'wrap', gap: 1, border: 'none' }}
                    >
                      {(['urgent', 'high', 'normal', 'low'] as Priority[]).map((priority) => (
                        <ToggleButton 
                          key={priority} 
                          value={priority}
                          sx={{ 
                            borderRadius: '20px !important', 
                            border: '1px solid !important',
                            borderColor: 'divider',
                            textTransform: 'capitalize',
                            px: 2,
                            py: 0.5,
                            '&.Mui-selected': { 
                              bgcolor: 'text.primary', 
                              color: 'background.paper',
                              '&:hover': { bgcolor: 'text.secondary' }
                            }
                          }}
                        >
                          {priority}
                        </ToggleButton>
                      ))}
                    </ToggleButtonGroup>
                  </Box>
                </Box>
              </Stack>
            </AccordionDetails>
          </Accordion>

          {/* List */}
          <Box>
            {loading ? (
              <Box display="flex" justifyContent="center" py={8}>
                <CircularProgress />
              </Box>
            ) : filteredNotifications.length === 0 ? (
              <Box display="flex" flexDirection="column" alignItems="center" justifyContent="center" py={8} textAlign="center">
                <NotificationsOffIcon sx={{ fontSize: 64, color: 'text.disabled', mb: 2 }} />
                <Typography variant="h6" color="text.secondary">No notifications found</Typography>
                <Typography variant="body2" color="text.disabled">Try adjusting your filters</Typography>
              </Box>
            ) : (
              <Stack spacing={2}>
                {filteredNotifications.map((notif) => (
                  <Card 
                    key={notif.id} 
                    elevation={0}
                    sx={{ 
                      borderRadius: 3, 
                      border: 1, 
                      borderColor: 'divider',
                      transition: 'all 0.2s',
                      '&:hover': { 
                        borderColor: 'primary.main',
                        boxShadow: theme.shadows[2],
                        transform: 'translateY(-1px)'
                      },
                      position: 'relative',
                      overflow: 'visible'
                    }}
                  >
                    {!notif.opened_at && (
                      <Box 
                        sx={{ 
                          position: 'absolute', 
                          left: -1, 
                          top: 24, 
                          bottom: 24, 
                          width: 4, 
                          bgcolor: 'primary.main', 
                          borderRadius: '0 4px 4px 0' 
                        }} 
                      />
                    )}
                    <CardActionArea 
                      onClick={() => handleOpenModal(notif)}
                      sx={{ p: 2, borderRadius: 3 }}
                    >
                      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={3} alignItems="flex-start">
                        {/* Icon */}
                        <Box sx={{ position: 'relative' }}>
                          <Avatar 
                            sx={{ 
                              bgcolor: alpha(theme.palette.primary.main, 0.05),
                              color: 'primary.main',
                              border: 1,
                              borderColor: 'divider'
                            }}
                          >
                            {getChannelIcon(notif.channel)}
                          </Avatar>
                          {notif.priority === 'urgent' && (
                            <Box 
                              sx={{ 
                                position: 'absolute', 
                                bottom: -4, 
                                right: -4, 
                                bgcolor: 'error.main', 
                                color: 'white', 
                                borderRadius: '50%', 
                                width: 20, 
                                height: 20, 
                                display: 'flex', 
                                alignItems: 'center', 
                                justifyContent: 'center',
                                border: '2px solid white'
                              }}
                            >
                              <PriorityHighIcon sx={{ fontSize: 12 }} />
                            </Box>
                          )}
                        </Box>

                        {/* Content */}
                        <Box flex={1}>
                          <Stack direction="row" justifyContent="space-between" alignItems="flex-start" spacing={2}>
                            <Box>
                              <Typography 
                                variant="subtitle1" 
                                fontWeight={!notif.opened_at ? 700 : 500}
                                color={!notif.opened_at ? 'text.primary' : 'text.secondary'}
                                gutterBottom
                              >
                                {notif.subject}
                              </Typography>
                              <Typography 
                                variant="body2" 
                                color="text.secondary" 
                                sx={{ 
                                  display: '-webkit-box',
                                  WebkitLineClamp: 2,
                                  WebkitBoxOrient: 'vertical',
                                  overflow: 'hidden'
                                }}
                              >
                                {notif.body}
                              </Typography>
                            </Box>
                            <Typography variant="caption" color="text.disabled" whiteSpace="nowrap">
                              {formatRelativeTime(notif.sent_at)}
                            </Typography>
                          </Stack>

                          {/* Footer / Actions */}
                          <Stack 
                            direction="row" 
                            alignItems="center" 
                            justifyContent="space-between" 
                            mt={2} 
                            pt={2} 
                            borderTop={1} 
                            borderColor="divider"
                          >
                            <Chip 
                              label={notif.priority} 
                              size="small" 
                              color={getPriorityColor(notif.priority) as any}
                              variant={notif.priority === 'urgent' ? 'filled' : 'outlined'}
                              sx={{ textTransform: 'capitalize', fontWeight: 600, height: 24 }}
                            />

                            <Stack direction="row" spacing={1}>
                              {!notif.opened_at && (
                                <Button
                                  size="small"
                                  startIcon={<DraftsIcon />}
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    markAsRead(notif.id);
                                  }}
                                  sx={{ color: 'text.secondary' }}
                                >
                                  Mark as read
                                </Button>
                              )}
                              
                              {notif.priority === 'urgent' && !notif.action_taken ? (
                                <Stack direction="row" spacing={1}>
                                  <Button
                                    variant="outlined"
                                    color="error"
                                    size="small"
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      recordAction(notif.id, 'reject');
                                    }}
                                  >
                                    Reject
                                  </Button>
                                  <Button
                                    variant="contained"
                                    color="primary"
                                    size="small"
                                    onClick={(e) => {
                                      e.stopPropagation();
                                      recordAction(notif.id, 'approve');
                                    }}
                                  >
                                    Approve
                                  </Button>
                                </Stack>
                              ) : (
                                <Button 
                                  size="small" 
                                  variant="outlined"
                                  onClick={(e) => {
                                    e.stopPropagation();
                                    handleOpenModal(notif);
                                  }}
                                >
                                  View Details
                                </Button>
                              )}
                            </Stack>
                          </Stack>
                        </Box>
                      </Stack>
                    </CardActionArea>
                  </Card>
                ))}
              </Stack>
            )}
          </Box>
        </Stack>
      </Container>
      
      <NotificationDetailModal
        notification={selectedNotification}
        isOpen={isModalOpen}
        onClose={handleCloseModal}
        onMarkAsRead={handleModalMarkAsRead}
        onApprove={handleModalApprove}
        onReject={handleModalReject}
      />
    </Box>
  );
};
