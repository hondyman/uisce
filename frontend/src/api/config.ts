/**
 * API Configuration for SemlayerEDM
 * 
 * This file contains centralized configuration for backend API integration
 */

// API Base URL - configured for local development
export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';

// Default Tenant ID - in a real application, this would come from authentication
// Format must be a valid UUID v4
export const DEFAULT_TENANT_ID = import.meta.env.VITE_DEFAULT_TENANT_ID || '550e8400-e29b-41d4-a716-446655440000';

// API Request Headers
export const getApiHeaders = (tenantId: string = DEFAULT_TENANT_ID) => ({
  'Content-Type': 'application/json',
  'X-Tenant-ID': tenantId,
  'Authorization': localStorage.getItem('auth_token') ? `Bearer ${localStorage.getItem('auth_token')}` : '',
});

// API Response Interceptor
export const handleApiError = (error: any): string => {
  if (error.response?.data?.error) {
    return error.response.data.error;
  }
  if (error.message) {
    return error.message;
  }
  return 'An unexpected error occurred';
};

// Export Service Configuration
export const EXPORT_CONFIG = {
  FORMATS: ['csv', 'json', 'parquet'] as const,
  DEFAULT_FORMAT: 'csv' as const,
  RETENTION_DAYS: 7,
  PRESIGNED_URL_EXPIRY_HOURS: 24,
};

// Scheduler Configuration
export const SCHEDULER_CONFIG = {
  SCHEDULE_TYPES: ['once', 'daily', 'weekly', 'monthly', 'cron'] as const,
  DEFAULT_SCHEDULE_TYPE: 'daily' as const,
  DEFAULT_TIMEZONE: 'UTC',
  TIMEZONES: [
    'UTC',
    'America/New_York',
    'America/Chicago',
    'America/Denver',
    'America/Los_Angeles',
    'Europe/London',
    'Europe/Paris',
    'Europe/Berlin',
    'Asia/Tokyo',
    'Asia/Shanghai',
    'Australia/Sydney',
  ],
};

// Common API Utilities
export const apiCall = async (
  endpoint: string,
  options: RequestInit = {},
  tenantId?: string
) => {
  const headers = getApiHeaders(tenantId);
  const response = await fetch(`${API_BASE_URL}${endpoint}`, {
    ...options,
    headers: {
      ...headers,
      ...(options.headers as Record<string, string>),
    },
  });

  if (!response.ok) {
    const data = await response.json().catch(() => null);
    throw new Error(data?.error || `API Error: ${response.statusText}`);
  }

  return response.json();
};
