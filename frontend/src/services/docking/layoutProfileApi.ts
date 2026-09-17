import { devError, devLog } from '../../utils/devLogger';
import { DeskWorkspaceState } from './LayoutManager';

export interface LayoutProfileSummary {
  id: string;
  profile_name: string;
  schema_version: string;
  updated_at: string;
}

export interface ServerLayoutResponse {
  id: string;
  tenant_id: string;
  user_id: string;
  profile_name: string;
  schema_version: string;
  layout_data: DeskWorkspaceState;
  created_at: string;
  updated_at: string;
}

const LAYOUT_API_BASE = '/api/user/preferences/workspace-layout';

function getAuthHeaders(): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  const token = typeof localStorage !== 'undefined' ? localStorage.getItem('auth_token') : null;
  if (token && !token.includes('demo')) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

/**
 * Fetches the user's roaming workspace layout profile from PostgreSQL.
 * Returns null if 404 (no server profile saved yet) or on network failure.
 */
export async function fetchServerLayoutProfile(
  profileName: string = 'default'
): Promise<DeskWorkspaceState | null> {
  try {
    const url = `${LAYOUT_API_BASE}?profile=${encodeURIComponent(profileName)}`;
    const res = await fetch(url, {
      method: 'GET',
      headers: getAuthHeaders(),
    });

    if (res.status === 404) {
      devLog(`[layoutProfileApi] No server profile found for "${profileName}" (first session)`);
      return null;
    }

    if (!res.ok) {
      devLog(`[layoutProfileApi] Failed to fetch layout profile: status ${res.status}`);
      return null;
    }

    const data: ServerLayoutResponse = await res.json();
    return data.layout_data || null;
  } catch (err) {
    devError('[layoutProfileApi] Network error fetching server layout profile:', err);
    return null;
  }
}

/**
 * Persists the user's workspace layout profile to PostgreSQL.
 * Single-Writer Rule: Call ONLY from the hub window (/workspace).
 */
export async function saveServerLayoutProfile(
  state: DeskWorkspaceState,
  profileName: string = 'default'
): Promise<boolean> {
  try {
    const payload = {
      profile_name: profileName,
      schema_version: 'v1',
      layout_data: state,
    };

    const res = await fetch(LAYOUT_API_BASE, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(payload),
    });

    if (!res.ok) {
      devError(`[layoutProfileApi] Failed to save layout profile: status ${res.status}`);
      return false;
    }

    devLog(`[layoutProfileApi] Saved workspace layout profile "${profileName}" to server`);
    return true;
  } catch (err) {
    devError('[layoutProfileApi] Network error saving layout profile to server:', err);
    return false;
  }
}

/**
 * Lists all roaming layout profile summaries for the authenticated user.
 */
export async function listServerLayoutProfiles(): Promise<LayoutProfileSummary[]> {
  try {
    const res = await fetch(`${LAYOUT_API_BASE}/profiles`, {
      method: 'GET',
      headers: getAuthHeaders(),
    });

    if (!res.ok) {
      devLog(`[layoutProfileApi] Failed to list layout profiles: status ${res.status}`);
      return [];
    }

    return await res.json();
  } catch (err) {
    devError('[layoutProfileApi] Network error listing layout profiles:', err);
    return [];
  }
}
