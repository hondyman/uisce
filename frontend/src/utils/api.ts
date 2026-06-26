import { devDebug } from '../utils/devLogger';
import apiClient from './apiClient';

/**
 * API utility functions for making requests to the backend
 */

/**
 * Get the full API URL for a given endpoint - DEPRECATED: use apiClient instead
 */
export const getApiUrl = (endpoint: string): string => {
  const apiBaseUrl = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';
  const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint;
  return `${apiBaseUrl}/api/${cleanEndpoint}`;
};

/**
 * Make a GET request to the API
 */
export const apiGet = async (endpoint: string): Promise<any> => {
  try {
    const resp = await apiClient(endpoint);
    return resp;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error(`API request failed: ${error}`);
  }
};

/**
 * Make a POST request to the API
 */
export const apiPost = async (endpoint: string, data?: any): Promise<any> => {
  // Check if demo mode is enabled via environment variable
  const isDemoMode = import.meta.env.VITE_DEMO_MODE === 'true';

  // Demo mode: simulate authentication endpoints
  if (isDemoMode && (endpoint === 'auth/login' || endpoint === 'auth/register')) {
    await new Promise(resolve => setTimeout(resolve, 1000));
    return {
      user: {
        id: 'demo-user-123',
        email: data?.email || 'demo@semlayer.com',
        name: data?.name || 'Demo User',
        role: 'admin',
        organization: data?.organization || 'Demo Organization',
        permissions: ['read', 'write', 'admin'],
        is_active: true
      },
      access_token: 'demo-jwt-token-' + Date.now(),
      refresh_token: 'demo-refresh-token-' + Date.now(),
      token_type: 'Bearer',
      expires_in: 3600
    };
  }
  // ... other demo mode checks
  if (isDemoMode && (endpoint === 'auth/forgot-password' || endpoint === 'auth/reset-password' || endpoint === 'auth/logout' || endpoint === 'auth/refresh')) {
    await new Promise(resolve => setTimeout(resolve, 500));
    return { message: 'Demo mode success' };
  }

  try {
    const resp = await apiClient(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
    return resp;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error(`API request failed: ${error}`);
  }
};

/**
 * Make a PUT request to the API
 */
export const apiPut = async (endpoint: string, data?: any): Promise<any> => {
  try {
    const resp = await apiClient(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
    return resp;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error(`API request failed: ${error}`);
  }
};

/**
 * Make a DELETE request to the API
 */
export const apiDelete = async (endpoint: string): Promise<any> => {
  try {
    const resp = await apiClient(endpoint, {
      method: 'DELETE',
    });
    return resp;
  } catch (error) {
    if (error instanceof Error) {
      throw error;
    }
    throw new Error(`API request failed: ${error}`);
  }
};