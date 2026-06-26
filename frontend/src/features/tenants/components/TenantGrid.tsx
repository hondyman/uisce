// React default import removed (not used as a value)
import { devLog } from '../../utils/devLogger';
import { DataGrid } from '@mui/x-data-grid';
import { Button } from '@mui/material';
import './TenantGrid.css';

interface TenantGridProps {
  tenants: any[];
}

const columns = [
  { field: 'id', headerName: 'ID', width: 90 },
  {
    field: 'name',
    headerName: 'Name',
    width: 150,
  },
  {
    field: 'instance',
    headerName: 'Instance',
    width: 150,
  },
  {
    field: 'actions',
    headerName: 'Actions',
    width: 150,
    renderCell: (params: any) => (
      <Button
        variant="contained"
        color="primary"
        onClick={() => {
          // Handle button click for the row
          devLog('Clicked row:', params.row);
        }}
      >
        View
      </Button>
    ),
  },
];

export default function TenantGrid({ tenants }: TenantGridProps) {
  const rows = tenants.map((tenant) => ({
    id: tenant._id,
    name: tenant.name,
    instance: tenant.instance,
  }));

  return (
  <div className="tenant-grid-container">
      <DataGrid
        rows={rows}
        columns={columns}
        // pageSize is now controlled via pagination model in newer MUI x versions
        initialState={{ pagination: { paginationModel: { pageSize: 5 } } }}
        pageSizeOptions={[5]}
    checkboxSelection
    disableRowSelectionOnClick
      />
    </div>
  );
}