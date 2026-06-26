import { useState } from 'react';
import type { FC } from 'react';
import {
  DataGrid,
  GridColDef,
} from '@mui/x-data-grid';
import {
  Chip,
  Stack,
  Select,
  MenuItem,
  Box,
} from '@mui/material';
import { useNotification } from '../../hooks/useNotification';
import { useQuery } from '@tanstack/react-query';
import ActionButton from '../ui/ActionButton';
import SVGIcon from '../relationship/SVGIcon';

/**
 * AuditLogViewer Component
 * 
 * Displays complete audit trail of all ABAC policy decisions and changes.
 * Supports filtering, search, and export for compliance.
 * 
 * Features:
 * - View all audit log entries
 * - Filter by action, result, user
 * - Search by resource or entity
 * - Export to CSV
 * - Multi-tenant scoped
 */

interface AuditLogEntry {
  id: string;
  timestamp: string;
  actor_id: string;
  actor_name?: string;
  action: string;
  resource: string;
  result: 'allow' | 'deny';
  reason?: string;
  ip_address?: string;
  user_agent?: string;
}

interface AuditLogViewerProps {
  tenantId: string;
  baseUrl?: string;
}

export const AuditLogViewer: FC<AuditLogViewerProps> = ({
  tenantId,
  baseUrl = '/api',
}) => {
  const notification = useNotification();
  const [filters, setFilters] = useState({
    action: undefined,
    result: undefined,
    days: 30,
  });

  // Fetch audit logs
  const { data: logs = [], isLoading, refetch } = useQuery<AuditLogEntry[]>({
    queryKey: ['audit-logs', tenantId, filters],
    queryFn: async () => {
      const params = new URLSearchParams();
      params.set('days', filters.days.toString());
      if (filters.action) params.set('action', filters.action);
      if (filters.result) params.set('result', filters.result);

      const response = await fetch(`${baseUrl}/abac/audit?${params}`, {
        headers: {
          'X-Tenant-ID': tenantId,
        },
      });
      if (!response.ok) throw new Error('Failed to load audit logs');
      return response.json();
    },
  });

  const handleExport = () => {
    const notification = useNotification();
    if (logs.length === 0) {
      notification.info('No logs to export');
      return;
    }

    // Convert to CSV
    const headers = [
      'Timestamp',
      'Actor',
      'Action',
      'Resource',
      'Result',
      'Reason',
      'IP Address',
    ];
    const rows = logs.map((log) => [
      new Date(log.timestamp).toLocaleString(),
      log.actor_name || log.actor_id,
      log.action,
      log.resource,
      log.result,
      log.reason || '',
      log.ip_address || '',
    ]);

    const csv = [
      headers.join(','),
      ...rows.map((row) => row.map((cell) => `"${cell}"`).join(',')),
    ].join('\n');

    // Download
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `audit-log-${new Date().toISOString()}.csv`;
    a.click();
  };

  const columns: GridColDef[] = [
    {
      field: 'timestamp',
      headerName: 'Timestamp',
      flex: 1,
      minWidth: 180,
      renderCell: (params) => new Date(params.value).toLocaleString(),
    },
    {
      field: 'actor_name',
      headerName: 'Actor',
      flex: 1,
      minWidth: 120,
      renderCell: (params) => params.value || params.row.actor_id,
    },
    {
      field: 'action',
      headerName: 'Action',
      flex: 1,
      minWidth: 140,
    },
    {
      field: 'resource',
      headerName: 'Resource',
      flex: 1,
      minWidth: 140,
    },
    {
      field: 'result',
      headerName: 'Result',
      flex: 0.5,
      minWidth: 100,
      renderCell: (params) => (
        <Chip
          label={params.value === 'allow' ? '✅ Allow' : '❌ Deny'}
          color={params.value === 'allow' ? 'success' : 'error'}
          size="small"
        />
      ),
    },
    {
      field: 'reason',
      headerName: 'Reason',
      flex: 1,
      minWidth: 140,
    },
    {
      field: 'ip_address',
      headerName: 'IP Address',
      flex: 1,
      minWidth: 130,
    },
  ];

  return (
    <Box className="audit-log-viewer">
      <Box className="audit-log-toolbar" sx={{ marginBottom: 2 }}>
        <Stack direction="row" spacing={2} className="audit-log-filters">
          <Select
            size="small"
            sx={{ width: 150 }}
            placeholder="Filter by action"
            value={filters.action || ''}
            onChange={(e) => setFilters({ ...filters, action: e.target.value === '' ? undefined : (e.target.value as any) })}
          >
            <MenuItem value="">All Actions</MenuItem>
            <MenuItem value="evaluate">Evaluate</MenuItem>
            <MenuItem value="create_policy">Create Policy</MenuItem>
            <MenuItem value="update_policy">Update Policy</MenuItem>
          </Select>

          <Select
            size="small"
            sx={{ width: 150 }}
            placeholder="Filter by result"
            value={filters.result || ''}
            onChange={(e) => setFilters({ ...filters, result: e.target.value === '' ? undefined : (e.target.value as any) })}
          >
            <MenuItem value="">All Results</MenuItem>
            <MenuItem value="allow">Allow</MenuItem>
            <MenuItem value="deny">Deny</MenuItem>
          </Select>

          <Select
            size="small"
            sx={{ width: 120 }}
            value={filters.days}
            onChange={(e) => setFilters({ ...filters, days: e.target.value as number })}
          >
            <MenuItem value={7}>Last 7 days</MenuItem>
            <MenuItem value={30}>Last 30 days</MenuItem>
            <MenuItem value={90}>Last 90 days</MenuItem>
          </Select>

          <ActionButton variant="secondary" onClick={() => refetch()}>
            <SVGIcon name="refresh" className="inline-block mr-2" ariaLabel="refresh" />
            Refresh
          </ActionButton>
          <ActionButton variant="primary" onClick={handleExport}>
            <SVGIcon name="download" className="inline-block mr-2" ariaLabel="download" />
            Export CSV
          </ActionButton>
        </Stack>
      </Box>

      <div style={{ height: 400, width: '100%' }}>
        <DataGrid
          columns={columns}
          rows={logs}
          getRowId={(row) => row.id}
          loading={isLoading}
          pageSizeOptions={[20, 50, 100]}
          initialState={{
            pagination: {
              paginationModel: { pageSize: 20, page: 0 },
            },
          }}
        />
      </div>
    </Box>
  );
};

export default AuditLogViewer;
