import React, { useState, useEffect } from 'react';
import { useTheme } from '@mui/material/styles';

interface TenantUpgradeStatus {
  tenantId: string;
  coreVersion: string;
  overlayVersion: string;
  totalTests: number;
  passed: number;
  failed: number;
  conflicts: number;
  snapshotId: string;
}

interface Conflict {
  id: string;
  tenantId: string;
  reason: string;
  path: string;
}

export const UpgradeDashboardPage: React.FC = () => {
  const { mode } = useTheme();
  const [statuses, setStatuses] = useState<TenantUpgradeStatus[]>([]);
  const [conflicts, setConflicts] = useState<Conflict[]>([]);
  const [selectedTenant, setSelectedTenant] = useState<string | null>(null);

  useEffect(() => {
    // Mock Data Fetch
    setStatuses([
      { tenantId: 'Client A', coreVersion: 'v1.3.0', overlayVersion: 'v1.3.0-ovl', totalTests: 120, passed: 118, failed: 2, conflicts: 0, snapshotId: 'snap-001' },
      { tenantId: 'Client B', coreVersion: 'v1.3.0', overlayVersion: 'v1.3.0-ovl', totalTests: 95, passed: 95, failed: 0, conflicts: 0, snapshotId: 'snap-002' },
      { tenantId: 'Client C', coreVersion: 'v1.3.0', overlayVersion: 'v1.3.0-ovl', totalTests: 110, passed: 109, failed: 1, conflicts: 1, snapshotId: 'snap-003' },
    ]);

    setConflicts([
      { id: 'c1', tenantId: 'Client C', reason: 'View "AccountForm" field risk_rating visibility rule invalid', path: 'views.AccountForm.fields.risk_rating' },
    ]);
  }, []);

  const handleResolveConflict = (id: string) => {
    alert(`Resolving conflict ${id}... (Mock Action)`);
    setConflicts(conflicts.filter(c => c.id !== id));
  };

  const bgColor = mode === 'dark' ? '#111827' : '#f9fafb';
  const textColor = mode === 'dark' ? '#f9fafb' : '#111827';
  const borderColor = mode === 'dark' ? '#374151' : '#e5e7eb';
  const headerBg = mode === 'dark' ? '#1f2937' : '#ffffff';

  return (
    <Box sx={{ bg: bgColor, color: textColor }}>
      {/* Header */}
      <div sx={{ mb: 8 }}>
        <h1 sx={{ fontSize: '3xl', fontWeight: 700, color: mode === 'dark' ? '#f9fafb' : '#111827' }}>Upgrade Dashboard</h1>
        <div sx={{ mt: 2, display: 'flex', gap: 6, color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>
          <span><strong>Core Version:</strong> v1.3.0</span>
          <span><strong>Snapshot:</strong> iceberg-snap-20251122-001</span>
          <span><strong>Timestamp:</strong> 22:45 EST</span>
        </div>
      </div>

      {/* Summary Table */}
      <Box sx={{ bg: 'white', borderRadius: 1, shadow: 1, overflow: 'hidden' }}>
        <div sx={{ px: 6, py: 4, borderBottom: `1px solid ${borderColor}`, fontWeight: 500 }}>
          <h2>Tenant Upgrade Status</h2>
        </div>
        <TableContainer component={Paper}>
          <Table sx={{ width: '100%' }}>
            <TableHead>
              <TableRow>
                <TableCell sx={{ px: 6, py: 3, fontSize: 'xs', fontWeight: 500, textTransform: 'uppercase', color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>Tenant</TableCell>
                <TableCell sx={{ px: 6, py: 3, fontSize: 'xs', fontWeight: 500, textTransform: 'uppercase', color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>Core Ver</TableCell>
                <TableCell sx={{ px: 6, py: 3, fontSize: 'xs', fontWeight: 500, textTransform: 'uppercase', color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>Overlay Ver</TableCell>
                <TableCell sx={{ px: 6, py: 3, fontSize: 'xs', fontWeight: 500, textTransform: 'uppercase', color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>Tests</TableCell>
                <TableCell sx={{ px: 6, py: 3, fontSize: 'xs', fontWeight: 500, textTransform: 'uppercase', color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>Pass/Fail</TableCell>
                <TableCell sx={{ px: 6, py: 3, fontSize: 'xs', fontWeight: 500, textTransform: 'uppercase', color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>Status</TableCell>
                <TableCell sx={{ px: 6, py: 3, fontSize: 'xs', fontWeight: 500, textTransform: 'uppercase', color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>Evidence</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {statuses.map((s) => (
                <TableRow key={s.tenantId} sx={{ hover: mode === 'dark' ? { bgColor: '#1f2937' } : { bgColor: '#f9fafr' }, cursor: 'pointer' }} onClick={() => setSelectedTenant(s.tenantId)}>
                  <TableCell sx={{ px: 6, py: 4, fontSize: 'md', fontWeight: 500, color: mode === 'dark' ? '#f9fafb' : '#111827' }}>{s.tenantId}</TableCell>
                  <TableCell sx={{ px: 6, py: 4, fontSize: 'sm', color: mode === 'dark' ? '#6b7280' : '#64748b' }}>{s.coreVersion}</TableCell>
                  <TableCell sx={{ px: 6, py: 4, fontSize: 'sm', color: mode === 'dark' ? '#6b7280' : '#64748b' }}>{s.overlayVersion}</TableCell>
                  <TableCell sx={{ px: 6, py: 4, fontSize: 'sm', color: mode === 'dark' ? '#6b7280' : '#64748b' }}>{s.totalTests}</TableCell>
                  <TableCell sx={{ px: 6, py: 4, fontSize: 'sm' }}>
                    <span sx={{ sx: { color: mode === 'dark' ? '#22c55e' : '#16a34a', fontWeight: 700 } }}>{s.passed}</span> /
                    <span sx={{ sx: { color: mode === 'dark' ? '#f87171' : '#ef4444', fontWeight: 700 } }}>{s.failed}</span>
                  </TableCell>
                  <TableCell sx={{ px: 6, py: 4, fontSize: 'sm' }}>
                    {s.conflicts > 0 ? (
                      <span sx={{ sx: { borderRadius: 1, bgColor: mode === 'dark' ? 'rgba(239, 68, 68, 0.1)' : 'rgba(239, 68, 68, 0.1)', color: '#f87171', px: 2, py: 1, fontSize: 'xs', fontWeight: 500 } }}>Conflicts ({s.conflicts})</span>
                    ) : s.failed > 0 ? (
                      <span sx={{ sx: { borderRadius: 1, bgColor: mode === 'dark' ? 'rgba(254, 202, 88, 0.1), color: '#fbbf24', px: 2, py: 1, fontSize: 'xs', fontWeight: 500 } }}>Issues</span>
                    ) : (
                      <span sx={{ sx: { borderRadius: 1, bgColor: mode === 'dark' ? 'rgba(34, 197, 94, 0.1), color: '#22c55e', px: 2, py: 1, fontSize: 'xs', fontWeight: 500 } }}>Ready</span>
                    )}
                  </TableCell>
                  <TableCell sx={{ px: 6, py: 4, fontSize: 'sm', color: mode === 'dark' ? '#9ca3af' : '#6b7280', textTransform: 'none' }}>{s.snapshotId}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      </Box>

      {/* Ops Panel: Conflicts */}
      {conflicts.length > 0 && (
        <Box sx={{ bg: 'white', borderRadius: 1, shadow: 1, borderLeft: `4px solid ${mode === 'dark' ? '#f87171' : '#dc2626'} }}}>
          <div sx={{ px: 6, py: 4, borderBottom: `1px solid ${borderColor}`, display: 'flex', justifyContent: 'between', items: 'center' }}>
            <h2 sx={{ fontSize: 'xl', fontWeight: 600, color: mode === 'dark' ? '#f87171' : '#dc262000' }}>Conflict Queue</h2>
            <span sx={{ sx: { bgColor: mode === 'dark' ? 'rgba(248, 113, 113, 0.1), color: '#f87171', px: 2, py: 1, borderRadius: 1, fontSize: 'xs', fontWeight: 500 } }}>Conflicts ({conflicts.length} Pending)</span>
          </div>
          <div sx={{ p: 6 }}>
            {conflicts.map((c) => (
              <div key={c.id} sx={{ sx: { bg: mode === 'dark' ? '#1f2937' : '#ffffff', p: 4, borderRadius: 1, mb: 2, border: `1px solid ${borderColor}` } }}>
                <div>
                  <p sx={{ fontWeight: 500, color: mode === 'dark' ? '#f9fafb' : '#111827' }}>{c.tenantId}</p>
                  <p sx={{ fontSize: 'sm', color: mode === 'dark' ? '#d1d5db' : '#6b7280' }}>{c.reason}</p>
                  <p sx={{ fontSize: 'xs', color: mode === 'dark' ? '#9ca3af' : '#9ca3af', fontFamily: 'monospace', mt: 1 }}>{c.path}</p>
                </div>
                <button
                  onClick={() => handleResolveConflict(c.id)}
                  sx={{
                    sx: { px: 4, py: 2, borderRadius: 1, background: mode === 'dark' ? '#dc2626' : '#dc2626', color: 'white', fontSize: 'sm' },
                    hover: { background: mode === 'dark' ? '#b91c1c' : '#b91c1c' }
                  }}
                >
                  Resolve
                </button>
              </div>
            ))}
          </div>
        </Box>
      )}

      {/* Drill Down (Mock) */}
      {selectedTenant && (
        <Box sx={{ bg: 'white', borderRadius: 1, shadow: 1, p: 6 }}>
          <h2 sx={{ fontSize: 'xl', fontWeight: 600, mb: 4 }}>Details for {selectedTenant}</h2>
          <div sx={{ display: 'grid', gridTemplateColumns: '1fr 2fr', gap: 4 }>
            <div sx={{ p: 4, borderRadius: 1, border: `1px solid ${borderColor}`, bg: mode === 'dark' ? '#f9fafr' : '#f9fafr' }}>
              <h3 sx={{ fontWeight: 500, mb: 2 }}>Test Categories</h3>
              <ul sx={{ display: 'space-y-2', fontSize: 'sm' }}>
                <li sx={{ display: 'flex', justifyContent: 'between' }}><span>Business Objects</span> <span sx={{ sx: { color: mode === 'dark' ? '#22c55e' : '#16a34a', fontWeight: 600 } }}>✔️ 58/58</span></li>
                <li sx={{ display: 'flex', justifyContent: 'between' }}><span>Business Processes</span> <span sx={{ sx: { color: mode === 'dark' ? '#22c55e' : '#16a34a', fontWeight: 600 } }}>✔️ 30/30</span></li>
                <li sx={{ display: 'flex', justifyContent: 'between' }}><span>UI Views</span> <span sx={{ sx: { color: mode === 'dark' ? '#f87171' : '#ef4444', fontWeight: 600 } }}>❌ 28/30</span></li>
                <li sx={{ display: 'flex', justifyContent: 'between' }}><span>Metrics</span> <span sx={{ sx: { color: mode === 'dark' ? '#22c55e' : '#16a34a', fontWeight: 600 } }}>✔️ 4/4</span></li>
              </ul>
            </div>
            <div sx={{ p: 4, borderRadius: 1, border: `1px solid ${borderColor}`, bg: mode === 'dark' ? '#f9fafr' : '#f9fafr' }}>
              <h3 sx={{ fontWeight: 500, mb: 2 }}>Actions</h3>
              <div sx={{ display: 'space-y-2' }}>
                <button sx={{ sx: { width: '100%', py: 2, borderRadius: 1, background: mode === 'dark' ? '#3b82f6' : '#3b82f6', color: 'white', fontSize: 'sm', hover: { background: mode === 'dark' ? '#2563eb' : '#2563eb' } }}>View UAR Log</button>
                <button sx={{ sx: { width: '100%', py: 2, borderRadius: 1, background: mode === 'dark' ? '#3b82f6' : '#3b82f6', color: 'white', fontSize: 'sm', hover: { background: mode === 'dark' ? '#2563eb' : '#2563eb' } }}>Download Iceberg Snapshot</button>
                <button sx={{ sx: { width: '100%', py: 2, borderRadius: 1, background: mode === 'dark' ? '#f3f4f6' : '#e5e7eb', color: '#111827', fontSize: 'sm', hover: { background: mode === 'dark' ? '#d1d5db' : '#d1d5db' } }}>Rollback to Previous Version</button>
              </div>
            </div>
          </div>
        </Box>
      )}
    </Box>
  );
};