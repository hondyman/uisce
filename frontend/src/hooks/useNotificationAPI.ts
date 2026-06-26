import { useCallback } from 'react';
import { getEnv } from '@internal/pkg/env/getEnv';
import { useAuthFetch } from '../utils/authFetch';

export interface NotificationAPI {
  getUserNotifications: (userId: string, limit: number, offset: number) => Promise<any[]>;
  markAsRead: (notificationId: string) => Promise<void>;
  trackEngagement: (event: EngagementEvent) => Promise<void>;
  getUserPreferences: (userId: string) => Promise<UserNotificationPreferences>;
  updateUserPreferences: (userId: string, preferences: UserNotificationPreferences) => Promise<void>;
  getEngagementAnalytics: (startDate: string, endDate: string) => Promise<EngagementAnalytics>;
  // Campaign methods
  createCampaign: (campaign: NotificationCampaign) => Promise<NotificationCampaign>;
  getCampaign: (campaignId: string) => Promise<NotificationCampaign>;
  getActiveCampaigns: () => Promise<NotificationCampaign[]>;
  launchCampaign: (campaignId: string) => Promise<void>;
  pauseCampaign: (campaignId: string) => Promise<void>;
  resumeCampaign: (campaignId: string) => Promise<void>;
  stopCampaign: (campaignId: string) => Promise<void>;
  getCampaignAnalytics: (campaignId: string) => Promise<CampaignAnalytics>;
  // Template methods
  createNotificationTemplate: (template: NotificationTemplate) => Promise<NotificationTemplate>;
}

export interface EngagementEvent {
  notification_id: string;
  user_id: string;
  event_type: string;
  event_timestamp: string;
  action_id?: string;
  action_type?: string;
  additional_metadata?: any;
}

export interface UserNotificationPreferences {
  user_id: string;
  email_enabled: boolean;
  sms_enabled: boolean;
  push_enabled: boolean;
  in_app_enabled: boolean;
  quiet_hours_start?: string;
  quiet_hours_end?: string;
  timezone: string;
  channel_preferences: { [key: string]: boolean };
  type_preferences: { [key: string]: boolean };
  frequency_preferences: { [key: string]: string };
  created_at: string;
  updated_at: string;
}

export interface EngagementAnalytics {
  total_sent: number;
  total_opened: number;
  total_clicked: number;
  avg_open_rate: number;
  avg_click_rate: number;
  period_start: string;
  period_end: string;
}

export interface NotificationCampaign {
  id: string;
  name: string;
  description: string;
  type: string;
  status: string;
  target_users: string[];
  user_segment: string;
  steps: NotificationCampaignStep[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface NotificationCampaignStep {
  id: string;
  step_number: number;
  template_id: string;
  delay_hours: number;
  trigger_event?: string;
  condition?: string;
  sent_count: number;
  open_rate: number;
  click_rate: number;
}

export interface CampaignAnalytics {
  campaign_id: string;
  total_sent: number;
  total_opened: number;
  total_clicked: number;
  total_converted: number;
  open_rate: number;
  click_rate: number;
  conversion_rate: number;
  step_performance: StepPerformance[];
}

export interface StepPerformance {
  step_number: number;
  sent_count: number;
  open_rate: number;
  click_rate: number;
  conversion_rate: number;
}

export interface NotificationTemplate {
  id: string;
  name: string;
  type: string;
  subject: string;
  title: string;
  message: string;
  rich_content?: any;
  variables: string[];
  channels: string[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

const API_BASE_URL = getEnv('', 'VITE_API_BASE_URL', 'http://localhost:29080') as string;

export const useNotificationAPI = (): NotificationAPI => {
  const { authFetch } = useAuthFetch();
  const getUserNotifications = useCallback(async (
    userId: string,
    limit: number,
    offset: number
  ): Promise<any[]> => {
  const resp = await authFetch<any[]>(`${API_BASE_URL}/api/notifications/user/${userId}?limit=${limit}&offset=${offset}`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch notifications: ${resp.status}`);
  return resp.data || [];
  }, []);

  const markAsRead = useCallback(async (notificationId: string): Promise<void> => {
  const resp = await authFetch(`${API_BASE_URL}/api/notifications/${notificationId}/read`, { method: 'POST' });
  if (!resp.ok) throw new Error(`Failed to mark notification as read: ${resp.status}`);
  }, []);

  const trackEngagement = useCallback(async (event: EngagementEvent): Promise<void> => {
  const resp = await authFetch(`${API_BASE_URL}/api/notifications/engagement`, { method: 'POST', json: event });
  if (!resp.ok) throw new Error(`Failed to track engagement: ${resp.status}`);
  }, []);

  const getUserPreferences = useCallback(async (userId: string): Promise<UserNotificationPreferences> => {
  const resp = await authFetch<UserNotificationPreferences>(`${API_BASE_URL}/api/notifications/preferences/${userId}`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch user preferences: ${resp.status}`);
  return resp.data as UserNotificationPreferences;
  }, []);

  const updateUserPreferences = useCallback(async (
    userId: string,
    preferences: UserNotificationPreferences
  ): Promise<void> => {
  const resp = await authFetch(`${API_BASE_URL}/api/notifications/preferences/${userId}`, { method: 'PUT', json: preferences });
  if (!resp.ok) throw new Error(`Failed to update user preferences: ${resp.status}`);
  }, []);

  const getEngagementAnalytics = useCallback(async (
    startDate: string,
    endDate: string
  ): Promise<EngagementAnalytics> => {
  const resp = await authFetch<EngagementAnalytics>(`${API_BASE_URL}/api/notifications/analytics?start=${startDate}&end=${endDate}`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch engagement analytics: ${resp.status}`);
  return resp.data as EngagementAnalytics;
  }, []);

  // Campaign methods
  const createCampaign = useCallback(async (campaign: NotificationCampaign): Promise<NotificationCampaign> => {
  const resp = await authFetch<NotificationCampaign>(`${API_BASE_URL}/api/notifications/campaigns`, { method: 'POST', json: campaign });
  if (!resp.ok) throw new Error(`Failed to create campaign: ${resp.status}`);
  return resp.data as NotificationCampaign;
  }, []);

  const getCampaign = useCallback(async (campaignId: string): Promise<NotificationCampaign> => {
  const resp = await authFetch<NotificationCampaign>(`${API_BASE_URL}/api/notifications/campaigns/${campaignId}`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch campaign: ${resp.status}`);
  return resp.data as NotificationCampaign;
  }, []);

  const getActiveCampaigns = useCallback(async (): Promise<NotificationCampaign[]> => {
  const resp = await authFetch<NotificationCampaign[]>(`${API_BASE_URL}/api/notifications/campaigns/active`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch active campaigns: ${resp.status}`);
  return resp.data as NotificationCampaign[];
  }, []);

  const launchCampaign = useCallback(async (campaignId: string): Promise<void> => {
  const resp = await authFetch(`${API_BASE_URL}/api/notifications/campaigns/${campaignId}/launch`, { method: 'POST' });
  if (!resp.ok) throw new Error(`Failed to launch campaign: ${resp.status}`);
  }, []);

  const pauseCampaign = useCallback(async (campaignId: string): Promise<void> => {
  const resp = await authFetch(`${API_BASE_URL}/api/notifications/campaigns/${campaignId}/pause`, { method: 'POST' });
  if (!resp.ok) throw new Error(`Failed to pause campaign: ${resp.status}`);
  }, []);

  const resumeCampaign = useCallback(async (campaignId: string): Promise<void> => {
  const resp = await authFetch(`${API_BASE_URL}/api/notifications/campaigns/${campaignId}/resume`, { method: 'POST' });
  if (!resp.ok) throw new Error(`Failed to resume campaign: ${resp.status}`);
  }, []);

  const stopCampaign = useCallback(async (campaignId: string): Promise<void> => {
  const resp = await authFetch(`${API_BASE_URL}/api/notifications/campaigns/${campaignId}/stop`, { method: 'POST' });
  if (!resp.ok) throw new Error(`Failed to stop campaign: ${resp.status}`);
  }, []);

  const getCampaignAnalytics = useCallback(async (campaignId: string): Promise<CampaignAnalytics> => {
  const resp = await authFetch<CampaignAnalytics>(`${API_BASE_URL}/api/notifications/campaigns/${campaignId}/analytics`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch campaign analytics: ${resp.status}`);
  return resp.data as CampaignAnalytics;
  }, []);

  const createNotificationTemplate = useCallback(async (template: NotificationTemplate): Promise<NotificationTemplate> => {
  const resp = await authFetch<NotificationTemplate>(`${API_BASE_URL}/api/notifications/templates`, { method: 'POST', json: template });
  if (!resp.ok) throw new Error(`Failed to create notification template: ${resp.status}`);
  return resp.data as NotificationTemplate;
  }, []);

  return {
    getUserNotifications,
    markAsRead,
    trackEngagement,
    getUserPreferences,
    updateUserPreferences,
    getEngagementAnalytics,
    createCampaign,
    getCampaign,
    getActiveCampaigns,
    launchCampaign,
    pauseCampaign,
    resumeCampaign,
    stopCampaign,
    getCampaignAnalytics,
    createNotificationTemplate,
  };
};
