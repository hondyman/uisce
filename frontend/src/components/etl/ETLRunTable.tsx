import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useETLRuns } from '../../api/etlRuns';
import { StatusBadge } from '../design/StatusBadge';

export interface ETLRunTableProps {
  tenantId?: string;
  onRowClick?: (id: string) => void;
}

export function ETLRunTable({ tenantId, onRowClick }: ETLRunTableProps) {
  const { data, isLoading } = useETLRuns({
    tenant_id: tenantId,
    limit: 200,
  });

  const rows = data ?? [];

  const columns: GridColDef[] = [
    {
      field: 'valuation_date',
      headerName: 'Valuation Date',
      width: 150,
      type: 'string',
    },
    {
      field: 'status',
      headerName: 'Status',
      width: 120,
      renderCell: (params) => <StatusBadge status={params.value} />,
    },
    {
      field: 'rules_evaluated',
      headerName: 'Rules',
      width: 100,
      type: 'number',
    },
    {
      field: 'scenarios_evaluated',
      headerName: 'Scenarios',
      width: 120,
      type: 'number',
    },
    {
      field: 'wasm_version',
      headerName: 'WASM Version',
      width: 150,
    },
    {
      field: 'orchestrator_version',
      headerName: 'Orchestrator',
      width: 150,
    },
    {
      field: 'duration',
      headerName: 'Duration',
      width: 120,
      valueGetter: (params) => {
        if (!params.row.completed_at) return '—';
        const start = new Date(params.row.started_at).getTime();
        const end = new Date(params.row.completed_at).getTime();
        return `${((end - start) / 1000).toFixed(1)}s`;
      },
    },
  ];

  return (
    <div style={{ width: '100%', height: 600 }}>
      <DataGrid
        rows={rows}
        columns={columns}
        loading={isLoading}
        getRowId={(row) => row.etl_run_id}
        onRowClick={(params) => onRowClick?.(params.row.etl_run_id)}
        sx={{ cursor: onRowClick ? 'pointer' : 'default' }}
        pageSizeOptions={[10, 25, 50, 100]}
        initialState={{
          pagination: { paginationModel: { pageSize: 25, page: 0 } },
        }}
      />
    </div>
  );
}
