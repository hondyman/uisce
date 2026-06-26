import { Button } from '@mui/material';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { useWASMVersions, useActivateWASMVersion } from '../../api/wasmVersions';

export function WASMVersionTable({ moduleName }: { moduleName: string }) {
  const { data, isLoading } = useWASMVersions(moduleName);
  const activate = useActivateWASMVersion();

  const columns: GridColDef[] = [
    {
      field: 'version',
      headerName: 'Version',
      width: 120,
    },
    {
      field: 'build_hash',
      headerName: 'Build Hash',
      width: 200,
      renderCell: (params) => (
        <code style={{ fontSize: '0.75rem' }}>{params.value?.substring(0, 8)}</code>
      ),
    },
    {
      field: 'build_time',
      headerName: 'Build Time',
      width: 150,
      valueGetter: (params) => {
        try {
          return new Date(params.value).toLocaleString();
        } catch {
          return params.value;
        }
      },
    },
    {
      field: 'artifact_uri',
      headerName: 'Artifact',
      width: 250,
      renderCell: (params) => (
        <a href={params.value} target="_blank" rel="noreferrer" style={{ fontSize: '0.875rem' }}>
          {new URL(params.value).pathname.split('/').pop()}
        </a>
      ),
    },
    {
      field: 'is_active',
      headerName: 'Active',
      width: 100,
      renderCell: (params) => (params.value ? '✓ Yes' : 'No'),
    },
    {
      field: 'actions',
      headerName: '',
      width: 150,
      sortable: false,
      filterable: false,
      renderCell: (params) =>
        !params.row.is_active && (
          <Button
            variant="contained"
            size="small"
            onClick={() => activate.mutate(params.row.wasm_version_id)}
            disabled={activate.isPending}
          >
            {activate.isPending ? 'Activating…' : 'Activate'}
          </Button>
        ),
    },
  ];

  return (
    <div style={{ width: '100%', height: 600 }}>
      <DataGrid
        rows={data ?? []}
        columns={columns}
        loading={isLoading}
        getRowId={(row) => row.wasm_version_id}
        pageSizeOptions={[10, 25, 50]}
        initialState={{
          pagination: { paginationModel: { pageSize: 25, page: 0 } },
        }}
      />
    </div>
  );
}
