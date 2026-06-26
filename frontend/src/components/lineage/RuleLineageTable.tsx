import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useRuleLineage } from '../../api/ruleLineage';
import { StatusBadge } from '../design/StatusBadge';

export function RuleLineageTable({ ruleId }: { ruleId: string }) {
  const { data, isLoading } = useRuleLineage(ruleId, {});

  const columns: GridColDef[] = [
    {
      field: 'valuation_date',
      headerName: 'Date',
      width: 150,
    },
    {
      field: 'portfolio_id',
      headerName: 'Portfolio',
      width: 150,
    },
    {
      field: 'status',
      headerName: 'Status',
      width: 120,
      renderCell: (params) => <StatusBadge status={params.value} />,
    },
    {
      field: 'metric_value',
      headerName: 'Metric Value',
      width: 150,
      type: 'number',
      valueGetter: (params) => parseFloat(params.value),
      renderCell: (params) => params.value?.toFixed(6),
    },
    {
      field: 'threshold_value',
      headerName: 'Threshold',
      width: 150,
      type: 'number',
      valueGetter: (params) => parseFloat(params.value),
      renderCell: (params) => params.value?.toFixed(6),
    },
    {
      field: 'etl_run_id',
      headerName: 'ETL Run',
      width: 200,
      renderCell: (params) => (
        <a href={`/console/etl-runs/${params.value}`} style={{ textDecoration: 'none' }}>
          {params.value}
        </a>
      ),
    },
  ];

  return (
    <div style={{ width: '100%', height: 600 }}>
      <DataGrid
        rows={data ?? []}
        columns={columns}
        loading={isLoading}
        getRowId={(row) => `${row.valuation_date}-${row.portfolio_id}`}
        pageSizeOptions={[10, 25, 50, 100]}
        initialState={{
          pagination: { paginationModel: { pageSize: 25, page: 0 } },
        }}
      />
    </div>
  );
}
