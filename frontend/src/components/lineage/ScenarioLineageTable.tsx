import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useScenarioLineage } from '../../api/scenarioLineage';

export function ScenarioLineageTable({ scenarioId }: { scenarioId: string }) {
  const { data, isLoading } = useScenarioLineage(scenarioId, {});

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
      field: 'pnl',
      headerName: 'P&L',
      width: 150,
      type: 'number',
      renderCell: (params) => {
        const value = params.value;
        const color = value > 0 ? '#2ECC71' : '#E74C3C';
        return <span style={{ color }}>{value?.toFixed(2)}</span>;
      },
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
