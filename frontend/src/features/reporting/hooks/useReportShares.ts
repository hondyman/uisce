import { useState, useCallback } from 'react';
import { fetchAPI } from '../../../api';

export interface ShareRecord {
  id: string;
  report_id: string;
  shared_by: string;
  recipient_id: string;
  recipient_name: string;
  recipient_email: string;
  recipient_role: string;
  recipient_organization: string;
  access_path: 'direct' | 'entitlement';
  permission: string;
  is_active: boolean;
  is_suspended: boolean;
  allow_export: boolean;
  allow_print: boolean;
  watermark: boolean;
  created_at: string;
  expires_at?: string;
  suspended_at?: string;
  last_login?: string;
}

export const useReportShares = (reportId?: string) => {
  const [shares, setShares] = useState<ShareRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchShares = useCallback(async () => {
    if (!reportId) return;
    setLoading(true);
    setError(null);
    try {
      const response = await fetchAPI<ShareRecord[]>(
        `/api/v1/reports/${reportId}/shares`
      );
      setShares(response || []);
    } catch (err: any) {
      setError(err.message || 'Failed to fetch shares');
    } finally {
      setLoading(false);
    }
  }, [reportId]);

  const addShare = useCallback(async (recipientId: string, permission = 'view') => {
    if (!reportId) return;
    const result = await fetchAPI<{ id: string }>(
      `/api/v1/reports/${reportId}/shares`,
      {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ recipient_id: recipientId, permission }),
      }
    );
    return result;
  }, [reportId]);

  const removeShare = useCallback(async (recipientId: string) => {
    if (!reportId) return;
    await fetchAPI(
      `/api/v1/reports/${reportId}/shares/${recipientId}`,
      { method: 'DELETE' }
    );
  }, [reportId]);

  const updateShare = useCallback(async (
    recipientId: string,
    updates: { suspend?: boolean; watermark?: boolean }
  ) => {
    if (!reportId) return;
    await fetchAPI(
      `/api/v1/reports/${reportId}/shares/${recipientId}`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(updates),
      }
    );
  }, [reportId]);

  const stopAllShares = useCallback(async () => {
    if (!reportId) return;
    await fetchAPI(
      `/api/v1/reports/${reportId}/shares/stop-all`,
      { method: 'POST' }
    );
  }, [reportId]);

  const cloneReport = useCallback(async () => {
    if (!reportId) return;
    const result = await fetchAPI<{ id: string; name: string }>(
      `/api/v1/reports/${reportId}/clone`,
      { method: 'POST' }
    );
    return result;
  }, [reportId]);

  return {
    shares,
    loading,
    error,
    fetchShares,
    addShare,
    removeShare,
    updateShare,
    stopAllShares,
    cloneReport,
  };
};
