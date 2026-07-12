import React, { useState, useEffect } from 'react';
import { useParams } from 'react-router-dom';
import { 
  Box, Typography, Paper, Button, Drawer, Stack, TextField,
  Divider, CircularProgress, Alert, Snackbar 
} from '@mui/material';
import { DataGrid, GridColDef, GridActionsCellItem } from '@mui/x-data-grid';
import { useForm, Controller } from 'react-hook-form';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { useTenant } from '../contexts/TenantContext';
import { ConfigurableNavigationSidebar } from '../components/ConfigurableNavigationSidebar';

interface WidgetConfig {
  field_id: string;
  technical_key: string;
  field_name: string;
  data_type: string;
  component_widget: string;
  is_visible_in_grid: boolean;
  is_editable: boolean;
  grid_width: number;
}

interface PageBlueprint {
  blueprint_id: string;
  page_key: string;
  bo_id: string;
  title: string;
  layout_type: string;
  widgets: WidgetConfig[];
}

export const DynamicDataProductPage: React.FC = () => {
  const { pageKey } = useParams<{ pageKey: string }>();
  const { tenant } = useTenant();
  const [blueprint, setBlueprint] = useState<PageBlueprint | null>(null);
  const [records, setRecords] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [selectedRecord, setSelectedRecord] = useState<any | null>(null);
  const [toastMessage, setToastMessage] = useState<string | null>(null);

  const { control, handleSubmit, reset } = useForm();

  const fetchPageMetadataAndData = async () => {
    setLoading(true);
    try {
      // Step 1: Fetch runtime layout configurations matching our route parameters
      // We pass the active tenant-id as a query param or header (assuming US-WEST default or Northwind tenant context)
      const tenantId = tenant?.id || "11111111-1111-1111-1111-111111111111"; // Default to seed tenant id
      const metaRes = await fetch(`/api/v1/layout/pages/${pageKey}/resolve?tenant_id=${tenantId}`);
      if (!metaRes.ok) throw new Error("Target page blueprint could not be processed by registry.");
      const metaData: PageBlueprint = await metaRes.json();
      setBlueprint(metaData);

      // Step 2: Use Data Product contracts to pull query streams instantly
      const dataRes = await fetch(`/api/v1/data/${pageKey}/v1.0.0?tenant_id=${tenantId}`);
      if (dataRes.ok) {
        const dataRecords = await dataRes.json();
        setRecords(dataRecords.map((r: any, idx: number) => ({ id: r.id || idx, ...r })));
      } else {
        setRecords([]);
      }
    } catch (err: any) {
      setError(err.message || "An unhandled framework exception triggered during schema resolution.");
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (pageKey) {
      fetchPageMetadataAndData();
    }
  }, [pageKey, tenant?.id]);

  if (loading) return <Box display="flex" justifyContent="center" py={10}><CircularProgress /></Box>;
  if (error || !blueprint) return <Alert severity="error" sx={{ m: 3 }}>{error || "Page definition lost."}</Alert>;

  const openFormDrawer = (record: any = null) => {
    setSelectedRecord(record);
    reset(record || {});
    setDrawerOpen(true);
  };

  const handleFormSubmit = async (data: any) => {
    const method = selectedRecord ? 'PUT' : 'POST';
    const tenantId = tenant?.id || "11111111-1111-1111-1111-111111111111";
    const endpoint = selectedRecord 
      ? `/api/v1/data/${blueprint.page_key}/v1.0.0/${selectedRecord.id}?tenant_id=${tenantId}`
      : `/api/v1/data/${blueprint.page_key}/v1.0.0?tenant_id=${tenantId}`;

    try {
      const response = await fetch(endpoint, {
        method,
        headers: { 
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId
        },
        body: JSON.stringify(data)
      });
      if (!response.ok) throw new Error("CRUD transaction aborted by governance safety gate.");
      
      setToastMessage(`Record ${selectedRecord ? 'updated' : 'instantiated'} successfully.`);
      setDrawerOpen(false);
      fetchPageMetadataAndData(); // Refresh the grid
    } catch (e: any) {
      setError(e.message);
    }
  };

  const handleDeleteRecord = async (id: any) => {
    const tenantId = tenant?.id || "11111111-1111-1111-1111-111111111111";
    const endpoint = `/api/v1/data/${blueprint.page_key}/v1.0.0/${id}?tenant_id=${tenantId}`;
    try {
      const response = await fetch(endpoint, {
        method: 'DELETE',
        headers: { 'X-Tenant-ID': tenantId }
      });
      if (!response.ok) throw new Error("Delete transaction aborted by governance safety gate.");
      setToastMessage("Record deleted successfully.");
      fetchPageMetadataAndData();
    } catch (e: any) {
      setError(e.message);
    }
  };

  // Build grid properties out of abstract parameters mapping cleanly to widget parameters
  const columns: GridColDef[] = blueprint.widgets
    .filter(w => w.is_visible_in_grid)
    .map(w => ({
      field: w.technical_key,
      headerName: w.field_name,
      width: w.grid_width,
      type: w.data_type === 'integer' || w.data_type === 'number' ? 'number' : 'string'
    }));

  columns.push({
    field: 'actions',
    type: 'actions',
    width: 100,
    getActions: (params) => [
      <GridActionsCellItem 
        key="edit"
        icon={<EditIcon />} 
        label="Edit" 
        onClick={() => openFormDrawer(params.row)} 
      />,
      <GridActionsCellItem 
        key="delete"
        icon={<DeleteIcon color="error" />} 
        label="Delete" 
        onClick={() => handleDeleteRecord(params.row.id)} 
      />
    ]
  });

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      <ConfigurableNavigationSidebar />
      <Box sx={{ flexGrow: 1, p: 4, bgcolor: '#f8fafc' }}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={4}>
          <Typography variant="h4" fontWeight="700" color="slate.900">{blueprint.title}</Typography>
          <Button variant="contained" startIcon={<AddIcon />} onClick={() => openFormDrawer()}>
            Add Entry
          </Button>
        </Box>

        <Paper variant="outlined" sx={{ height: 550, borderRadius: 2, overflow: 'hidden', border: '1px solid #e2e8f0' }}>
          <DataGrid rows={records} columns={columns} disableRowSelectionOnClick density="comfortable" />
        </Paper>

      <Drawer anchor="right" open={drawerOpen} onClose={() => setDrawerOpen(false)}>
        <Box sx={{ width: 450, p: 4, display: 'flex', flexDirection: 'column', height: '100%' }}>
          <Typography variant="h6" fontWeight="600" mb={1}>
            {selectedRecord ? 'Modify Record Parameter Block' : 'Instantiate New Semantic Record'}
          </Typography>
          <Typography variant="caption" color="text.secondary" mb={3}>
            Fields are bounded directly to underlying business object constraints.
          </Typography>
          <Divider />

          <form onSubmit={handleSubmit(handleFormSubmit)} style={{ flexGrow: 1, marginTop: '24px', position: 'relative' }}>
            <Stack spacing={3}>
              {blueprint.widgets.filter(w => w.is_editable).map((w) => (
                <Controller
                  key={w.field_id}
                  name={w.technical_key}
                  control={control}
                  defaultValue=""
                  render={({ field }) => (
                    <TextField
                      {...field}
                      label={w.field_name}
                      fullWidth
                      size="small"
                      type={w.data_type === 'integer' || w.data_type === 'number' ? 'number' : 'text'}
                    />
                  )}
                />
              ))}
            </Stack>

            <Box sx={{ position: 'absolute', bottom: 24, right: 0, left: 0, display: 'flex', gap: 2 }}>
              <Button variant="outlined" fullWidth onClick={() => setDrawerOpen(false)}>Cancel</Button>
              <Button type="submit" variant="contained" fullWidth>Save Transaction</Button>
            </Box>
          </form>
        </Box>
      </Drawer>

      <Snackbar open={Boolean(toastMessage)} autoHideDuration={4000} onClose={() => setToastMessage(null)} message={toastMessage} />
      </Box>
    </Box>
  );
};
