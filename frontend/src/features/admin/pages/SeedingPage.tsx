import React, { useState, useEffect } from 'react';
import { useTheme } from '@mui/material/styles';
import { useTenant } from '../../../contexts/TenantContext';
import * as adminSeed from '../../../api/adminSeed';
import { Download, Trash2, Zap, CheckCircle } from 'lucide-react';

const SeedingPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const { mode } = useTheme();
  const [tenantId, setTenantId] = useState('');
  const [datasourceId, setDatasourceId] = useState('');
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<any>(null);
  const [toast, setToast] = useState<{ type: 'success'|'error', message: string } | null>(null);

  useEffect(() => {
    if (tenant && datasource) {
      setTenantId(tenant.id);
      setDatasourceId(datasource.id);
      return;
    }

    try {
      const rawTenant = window.localStorage.getItem('selected_tenant');
      const rawDatasource = window.localStorage.getItem('selected_datasource');
      if (rawTenant) {
        try { const p = JSON.parse(rawTenant); if (p?.id) setTenantId(p.id); } catch (e) {}
      }
      if (rawDatasource) {
        try { const p = JSON.parse(rawDatasource); if (p?.id) setDatasourceId(p.id); } catch (e) {}
      }
    } catch (e) {}

    const params = new URLSearchParams(window.location.search);
    if (!tenantId) setTenantId(params.get('tenantId') || '');
    if (!datasourceId) setDatasourceId(params.get('datasourceId') || '');
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tenant, datasource]);

  const showToast = (type: 'success'|'error', message: string) => {
    setToast({ type, message });
    setTimeout(() => setToast(null), 4000);
  };

  const handleSeedAll = async () => {
    if (!tenantId) return showToast('error', 'Tenant is required');
    setLoading(true);
    try {
      const res = await adminSeed.seedAll(tenantId, datasourceId || undefined);
      if (!res.ok) throw new Error(res.data?.message || `HTTP ${res.status}`);
      setResult(res.data || null);
      showToast('success', 'Seed completed');
    } catch (e) {
      showToast('error', e instanceof Error ? e.message : 'Seed failed');
    } finally { setLoading(false); }
  };

  const handleSeedValidation = async () => {
    if (!tenantId || !datasourceId) return showToast('error', 'Tenant and datasource required for validation rules');
    setLoading(true);
    try {
      const res = await adminSeed.seedValidationRules(tenantId, datasourceId);
      if (!res.ok) throw new Error(res.data?.message || `HTTP ${res.status}`);
      setResult(res.data || null);
      showToast('success', 'Validation rules seeded');
    } catch (e) {
      showToast('error', e instanceof Error ? e.message : 'Seed failed');
    } finally { setLoading(false); }
  };

  const handleSeedApproval = async () => {
    if (!tenantId) return showToast('error', 'Tenant required');
    setLoading(true);
    try {
      const res = await adminSeed.seedApprovalRules(tenantId);
      if (!res.ok) throw new Error(res.data?.message || `HTTP ${res.status}`);
      setResult(res.data || null);
      showToast('success', 'Approval rules seeded');
    } catch (e) {
      showToast('error', e instanceof Error ? e.message : 'Seed failed');
    } finally { setLoading(false); }
  };

  const handleClear = async () => {
    if (!tenantId) return showToast('error', 'Tenant required');
    if (!confirm('Clear seeded rules? This is destructive.')) return;
    setLoading(true);
    try {
      const res = await adminSeed.clearSeed(tenantId, datasourceId || undefined);
      if (!res.ok) throw new Error(res.data?.message || `HTTP ${res.status}`);
      setResult(null);
      showToast('success', 'Seed cleared');
    } catch (e) {
      showToast('error', e instanceof Error ? e.message : 'Clear failed');
    } finally { setLoading(false); }
  };

  const bgColor = mode === 'dark' ? '#111827' : '#f9fafb';
  const textColor = mode === 'dark' ? '#f9fafb' : '#111827';
  const borderColor = mode === 'dark' ? '#374151' : '#e5e7eb';

  return (
    <Box sx={{ bg: bgColor, color: textColor }}>
      <h1 sx={{ fontSize: '2xl', fontWeight: 700, mb: 2 }}>Seeding Console</h1>
      <p sx={{ fontSize: 'sm', color: 'text.secondary', mb: 4 }}>
        Run seeding operations for validation rules, approval rules and assignments. Tenant scope is required.
      </p>

      {toast && (
        <Box sx={{ mb: 4, p: 3, borderRadius: 1 }}>
          {toast.message}
        </Box>
      )}

      <div sx={{ display: 'grid', gridTemplateColumns: 'repeat(2, 1fr)', gap: 4, mb: 6 }}>
        <div>
          <label sx={{ display: 'block', fontSize: 'sm', fontWeight: 500, mb: 1 }}>Tenant ID</label>
          <input
            value={tenantId}
            onChange={(e) => setTenantId(e.target.value)}
            sx={{ width: '100', p: 2, borderRadius: 1, border: `1px solid ${borderColor}` }}
          />
        </div>
        <div>
          <label sx={{ display: 'block', fontSize: 'sm', fontWeight: 500, mb: 1 }}>Datasource ID</label>
          <input
            value={datasourceId}
            onChange={(e) => setDatasourceId(e.target.value)}
            sx={{ width: '100', p: 2, borderRadius: 1, border: `1px solid ${borderColor}` }}
          />
        </div>
      </div>

      <div sx={{ display: 'flex', gap: 3, mb: 6 }}>
        <button
          onClick={handleSeedAll}
          disabled={loading || !tenantId}
          sx={{
            px: 4, py: 2, borderRadius: 1, color: 'white',
            display: 'flex', items: 'center', gap: 2, fontSize: 'sm',
            background: mode === 'dark' ? '#10b981' : '#10b981'
          }}
        >
          <Download sx={{ width: 4, height: 4 }} /> Seed All
        </button>
        <button
          onClick={handleSeedValidation}
          disabled={loading || !tenantId || !datasourceId}
          sx={{
            px: 4, py: 2, borderRadius: 1, color: 'white',
            display: 'flex', items: 'center', gap: 2, fontSize: 'sm',
            background: mode === 'dark' ? '#3b82f6' : '#3b82f6'
          }}
        >
          <Zap sx={{ width: 4, height: 4 }} /> Seed Validation Rules
        </button>
        <button
          onClick={handleSeedApproval}
          disabled={loading || !tenantId}
          sx={{
            px: 4, py: 2, borderRadius: 1, color: 'white',
            display: 'flex', items: 'center', gap: 2, fontSize: 'sm',
            background: mode === 'dark' ? '#a855f7' : '#a855f7'
          }}
        >
          <CheckCircle sx={{ width: 4, height: 4 }} /> Seed Approval Rules
        </button>
        <button
          onClick={handleClear}
          disabled={loading || !tenantId}
          sx={{
            px: 4, py: 2, borderRadius: 1, color: 'white',
            display: 'flex', items: 'center', gap: 2, fontSize: 'sm',
            background: mode === 'dark' ? '#dc2626' : '#dc2626'
          }}
        >
          <Trash2 sx={{ width: 4, height: 4 }} /> Clear Seed
        </button>
      </div>

      <Box sx={{ bg: 'white', p: 4, borderRadius: 1, border: `1px solid ${borderColor}` }}>
        <h2 sx={{ fontWeight: 500, mb: 2 }}>Result</h2>
        <pre sx={{ fontSize: 'xs', whitePre: 'wrap' }}>{JSON.stringify(result, null, 2) || 'No results yet'}</pre>
      </Box>
    </Box>
  );
};

export default SeedingPage;