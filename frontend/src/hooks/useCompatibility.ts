import { useState, useCallback, useMemo, useEffect } from 'react';
import { getCompatibilityReport, type ExtensionCompatibilityRow } from '../services/extensions';

interface Params {
  datasourceId: string;
  issueLevelFilter: 'all' | 'error' | 'warning';
  issueCodeFilter: string;
}

const useCompatibility = ({ datasourceId, issueLevelFilter, issueCodeFilter }: Params) => {
  const [compat, setCompat] = useState<ExtensionCompatibilityRow[] | null>(null);
  const [compatErr, setCompatErr] = useState<string | null>(null);
  const [compatLoading, setCompatLoading] = useState(false);

  const refreshCompatibility = useCallback(async () => {
    try {
      setCompatLoading(true);
      setCompatErr(null);
      const res = await getCompatibilityReport(datasourceId);
      setCompat(res.report || []);
    } catch (e: unknown) {
      const msg = e instanceof Error
        ? e.message
        : (e && typeof e === 'object' && 'message' in e)
        ? String((e as Record<string, unknown>)['message'])
        : 'Failed to load compatibility';
      setCompatErr(msg);
    } finally {
      setCompatLoading(false);
    }
  }, [datasourceId]);

  // derive filteredCompat
  const filteredCompat = useMemo(() => {
    if (!compat) return null;
    if (issueLevelFilter === 'all' && !issueCodeFilter) return compat;
    return compat.map((row) => ({
      ...row,
      issues: row.issues?.filter((i) => {
        if (issueLevelFilter !== 'all' && i.level !== issueLevelFilter) return false;
        if (issueCodeFilter && !i.code.toLowerCase().includes(issueCodeFilter.toLowerCase())) return false;
        return true;
      }) || [],
    }));
  }, [compat, issueLevelFilter, issueCodeFilter]);

  // auto-refresh when datasourceId changes
  useEffect(() => { refreshCompatibility(); }, [datasourceId, refreshCompatibility]);

  return { compat, compatErr, compatLoading, refreshCompatibility, filteredCompat };
};

export default useCompatibility;
