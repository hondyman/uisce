import { useCallback } from 'react';
import { getEnv } from '@internal/pkg/env/getEnv';
import { useAuthFetch } from '../utils/authFetch';

export interface Dashboard {
  id: string;
  name: string;
  description?: string;
  widgets: DashboardWidget[];
  layout: 'grid' | 'freeform';
  theme?: string;
  isPublic: boolean;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

export interface DashboardWidget {
  id: string;
  type: string;
  title: string;
  position: { x: number; y: number };
  size: { width: number; height: number };
  config: any;
  data?: any;
}

export interface DashboardTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  widgets: Omit<DashboardWidget, 'id'>[];
  previewImage?: string;
  isDefault: boolean;
}

interface DashboardService {
  getDashboards: (userId: string) => Promise<Dashboard[]>;
  getDashboard: (dashboardId: string) => Promise<Dashboard>;
  saveDashboard: (dashboard: Dashboard) => Promise<Dashboard>;
  deleteDashboard: (dashboardId: string) => Promise<void>;
  getPublicDashboards: () => Promise<Dashboard[]>;
  getDashboardTemplates: () => Promise<DashboardTemplate[]>;
  duplicateDashboard: (dashboardId: string, newName: string) => Promise<Dashboard>;
}

const API_BASE_URL = getEnv('', 'VITE_API_BASE_URL', 'http://localhost:29080') as string;

export const useDashboardService = (): DashboardService => {
  const { authFetch } = useAuthFetch();
  const getDashboards = useCallback(async (userId: string): Promise<Dashboard[]> => {
    const resp = await authFetch<{ dashboards: Dashboard[] }>(`${API_BASE_URL}/api/dashboards/user/${userId}`, { method: 'GET' });
    if (!resp.ok) throw new Error(`Failed to fetch dashboards: ${resp.status}`);
    return resp.data?.dashboards || [];
  }, []);

  const getDashboard = useCallback(async (dashboardId: string): Promise<Dashboard> => {
  const resp = await authFetch<Dashboard>(`${API_BASE_URL}/api/dashboards/${dashboardId}`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch dashboard: ${resp.status}`);
  return resp.data as Dashboard;
  }, []);

  const saveDashboard = useCallback(async (dashboard: Dashboard): Promise<Dashboard> => {
    const method = dashboard.id.startsWith('dashboard-') ? 'POST' : 'PUT';
    const url = method === 'POST'
      ? `${API_BASE_URL}/api/dashboards`
      : `${API_BASE_URL}/api/dashboards/${dashboard.id}`;
  const resp = await authFetch<Dashboard>(url, { method, json: dashboard });
  if (!resp.ok) throw new Error(`Failed to save dashboard: ${resp.status}`);
  return resp.data as Dashboard;
  }, []);

  const deleteDashboard = useCallback(async (dashboardId: string): Promise<void> => {
  const resp = await authFetch(`${API_BASE_URL}/api/dashboards/${dashboardId}`, { method: 'DELETE' });
  if (!resp.ok) throw new Error(`Failed to delete dashboard: ${resp.status}`);
  }, []);

  const getPublicDashboards = useCallback(async (): Promise<Dashboard[]> => {
  const resp = await authFetch<{ dashboards: Dashboard[] }>(`${API_BASE_URL}/api/dashboards/public`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch public dashboards: ${resp.status}`);
  return resp.data?.dashboards || [];
  }, []);

  const getDashboardTemplates = useCallback(async (): Promise<DashboardTemplate[]> => {
  const resp = await authFetch<{ templates: DashboardTemplate[] }>(`${API_BASE_URL}/api/dashboards/templates`, { method: 'GET' });
  if (!resp.ok) throw new Error(`Failed to fetch dashboard templates: ${resp.status}`);
  return resp.data?.templates || [];
  }, []);

  const duplicateDashboard = useCallback(async (dashboardId: string, newName: string): Promise<Dashboard> => {
  const resp = await authFetch<Dashboard>(`${API_BASE_URL}/api/dashboards/${dashboardId}/duplicate`, { method: 'POST', json: { name: newName } });
  if (!resp.ok) throw new Error(`Failed to duplicate dashboard: ${resp.status}`);
  return resp.data as Dashboard;
  }, []);

  return {
    getDashboards,
    getDashboard,
    saveDashboard,
    deleteDashboard,
    getPublicDashboards,
    getDashboardTemplates,
    duplicateDashboard,
  };
};
