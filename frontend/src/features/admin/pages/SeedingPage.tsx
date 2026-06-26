import React, { useState, useEffect } from 'react';
import { useTenant } from '../../../contexts/TenantContext';
import * as adminSeed from '../../../api/adminSeed';
import { Download, Trash2, Zap, CheckCircle } from 'lucide-react';

const SeedingPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
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

  return (
    <div className="min-h-screen p-6">
      <h1 className="text-2xl font-bold mb-2">Seeding Console</h1>
      <p className="text-sm text-gray-600 mb-4">Run seeding operations for validation rules, approval rules and assignments. Tenant scope is required.</p>

      {toast && (
        <div className={`mb-4 p-3 rounded ${toast.type === 'success' ? 'bg-green-50' : 'bg-red-50'}`}>
          {toast.message}
        </div>
      )}

      <div className="grid grid-cols-2 gap-4 mb-6">
        <div>
          <label className="block text-sm font-medium mb-1">Tenant ID</label>
          <input value={tenantId} onChange={(e) => setTenantId(e.target.value)} className="w-full p-2 border rounded" />
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Datasource ID</label>
          <input value={datasourceId} onChange={(e) => setDatasourceId(e.target.value)} className="w-full p-2 border rounded" />
        </div>
      </div>

      <div className="flex gap-3 mb-6">
        <button onClick={handleSeedAll} disabled={loading || !tenantId} className="px-4 py-2 bg-green-600 text-white rounded flex items-center gap-2">
          <Download className="w-4 h-4" /> Seed All
        </button>
        <button onClick={handleSeedValidation} disabled={loading || !tenantId || !datasourceId} className="px-4 py-2 bg-blue-600 text-white rounded flex items-center gap-2">
          <Zap className="w-4 h-4" /> Seed Validation Rules
        </button>
        <button onClick={handleSeedApproval} disabled={loading || !tenantId} className="px-4 py-2 bg-purple-600 text-white rounded flex items-center gap-2">
          <CheckCircle className="w-4 h-4" /> Seed Approval Rules
        </button>
        <button onClick={handleClear} disabled={loading || !tenantId} className="px-4 py-2 bg-red-600 text-white rounded flex items-center gap-2">
          <Trash2 className="w-4 h-4" /> Clear Seed
        </button>
      </div>

      <div className="bg-white p-4 rounded border">
        <h2 className="font-semibold mb-2">Result</h2>
        <pre className="text-xs whitespace-pre-wrap">{JSON.stringify(result, null, 2) || 'No results yet'}</pre>
      </div>
    </div>
  );
};

export default SeedingPage;
