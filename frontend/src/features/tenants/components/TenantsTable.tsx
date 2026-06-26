import React, { useState, useMemo, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { DataGrid, GridColDef } from '@mui/x-data-grid';
import { Box, Chip, Link, IconButton } from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import { ProfessionalSearchInput } from '../../../components/common/ProfessionalSearchInput';
import type { Tenant } from '../../../types';

interface TenantsTableProps {
  tenants: Tenant[];
  onEditTenant: (tenant: Tenant) => void;
  onDeleteTenant: (tenantId: string) => void;
  // optional tenant id to focus/highlight when the table mounts
  focusTenantId?: string | null;
}

const TenantsTable: React.FC<TenantsTableProps> = ({
  tenants, onEditTenant, onDeleteTenant
  , focusTenantId = null
}) => {
  const navigate = useNavigate();
  const [searchValue, setSearchValue] = useState('');
  // simplified: ERD modal removed
  console.log('TenantsTable tenants:', tenants);

  const rows = tenants.map(tenant => ({
    id: String(tenant.id),
    tenantId: String(tenant.id),
    tenantName: tenant.display_name || tenant.name || String(tenant.id),
    displayName: tenant.display_name || tenant.name || String(tenant.id),
    gold_copy: tenant.gold_copy,
    // Be defensive about the instances property coming from different APIs or shapes
    // Some backends may populate tenant_instances, instances or Instances — accept either
    instanceCount: (tenant.tenant_instances?.length ?? (tenant as any).instances?.length ?? (tenant as any).Instances?.length) || 0,
    isInstance: false,
    is_active: tenant.is_active,
  }));

  // Filter rows based on search value
  const filteredRows = useMemo(() => {
    if (!searchValue.trim()) return rows;
    const searchLower = searchValue.toLowerCase();
    return rows.filter(row =>
      row.tenantName.toLowerCase().includes(searchLower) ||
      row.displayName.toLowerCase().includes(searchLower)
    );
  }, [rows, searchValue]);

  // compute which row id corresponds to the tenant-level row (not instance)
  const focusedRowId = useMemo(() => {
    if (!focusTenantId) return null;
    const r = rows.find(row => !row.isInstance && String(row.tenantId) === String(focusTenantId));
    return r ? r.id : null;
  }, [focusTenantId, rows]);

  const [selectionModel, setSelectionModel] = useState<string[]>(focusedRowId ? [String(focusedRowId)] : []);

  useEffect(() => {
    if (focusedRowId) setSelectionModel([String(focusedRowId)]);
  }, [focusedRowId]);

  // ref map for row elements to support scrollIntoView
  const rowRefs = useRef<Record<string, HTMLDivElement | null>>({});

  useEffect(() => {
    if (!focusedRowId) return;
    // give DataGrid a tick to render rows
    const id = String(focusedRowId);
    setTimeout(() => {
      const el = rowRefs.current[id];
      if (el && typeof el.scrollIntoView === 'function') {
        el.scrollIntoView({ behavior: 'smooth', block: 'center' });
      } else {
        // fallback: attempt to find by data-id in DOM
        const dom = document.querySelector(`[data-id="${id}"]`);
        if (dom && (dom as HTMLElement).scrollIntoView) {
          (dom as HTMLElement).scrollIntoView({ behavior: 'smooth', block: 'center' });
        }
      }
    }, 150);
  }, [focusedRowId]);

  const columns: GridColDef[] = [
    { field: 'tenantName', headerName: 'Tenant', flex: 1, groupable: true },
    {
      field: 'displayName',
      headerName: 'Name',
      flex: 1,
      renderCell: (params) => {
        // For tenant rows, make it a clickable link to the tenant detail page
        const tenant = tenants.find(t => String(t.id) === String(params.row.tenantId));
        if (!tenant) return params.value;
        
        if (tenant.gold_copy) {
           console.log('Rendering Gold Copy for:', tenant.display_name, tenant);
        }

        return (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Link
              component="button"
              variant="body2"
              onClick={() => navigate(`/tenants/${params.row.tenantId}`)}
              sx={{ textAlign: 'left', fontWeight: 'bold' }}
            >
              {params.value}
            </Link>
            {tenant.gold_copy === true && (
              <Chip
                label="Gold Copy"
                size="small"
                color="warning"
                sx={{ height: 20, fontSize: '0.7rem', fontWeight: 'bold' }}
              />
            )}
          </Box>
        );
      },
    },
    {
      field: 'instanceCount',
      headerName: 'Instances',
      width: 120,
      renderCell: (params) => (
        <Link
          component="button"
          variant="body2"
          onClick={() => navigate(`/tenants/${params.row.tenantId}`)}
          sx={{ textAlign: 'center', fontWeight: 'bold', minWidth: '40px' }}
        >
          {params.value}
        </Link>
      ),
    },
    {
      field: 'is_active',
      headerName: 'Status',
      width: 120,
      renderCell: (params) => (
        <Chip label={params.value ? 'Active' : 'Inactive'} color={params.value ? 'success' : 'default'} size="small" />
      ),
    },
  // catalog column removed
    {
      field: 'actions',
      headerName: 'Actions',
      width: 100,
      renderCell: (params) => {
        const tenant = tenants.find(t => String(t.id) === String(params.row.tenantId));
        if (!tenant) return null;
        return (
          <Box sx={{ display: 'flex', gap: 1 }}>
            <IconButton
              size="small"
              onClick={() => onEditTenant(tenant)}
              aria-label="Edit tenant"
              sx={{ color: 'primary.main' }}
            >
              <EditIcon fontSize="small" />
            </IconButton>
            <span title={params.row.gold_copy ? "Gold Copy tenants cannot be deleted" : "Delete tenant"}>
              <IconButton
                size="small"
                onClick={() => onDeleteTenant(String(params.row.id))}
                aria-label="Delete tenant"
                sx={{ color: 'error.main' }}
                disabled={params.row.gold_copy}
              >
                <DeleteIcon fontSize="small" />
              </IconButton>
            </span>
          </Box>
        );
      },
    },
  ];

  // Custom toolbar with search input only
  const CustomToolbar = () => (
    <Box sx={{ p: 2, display: 'flex', gap: 2, alignItems: 'center' }}>
      <ProfessionalSearchInput
        value={searchValue}
        onChange={setSearchValue}
        placeholder="Search tenants and instances..."
        size="md"
        variant="default"
      />
    </Box>
  );

  return (
    <Box sx={{ height: 650, width: '100%' }}>
      <DataGrid
        rows={filteredRows}
        columns={columns}
        // highlight matching tenant row with a class and keep it selected so it's easier to spot
        getRowClassName={(params) => (String(params.row.tenantId) === String(focusTenantId) && !params.row.isInstance ? 'focused-tenant-row' : '')}
        selectionModel={selectionModel}
  onSelectionModelChange={(newModel: any) => setSelectionModel(newModel as string[])}
        getRowHeight={() => 'auto'}
        componentsProps={{
          row: {
            ref: (el: any) => {
              if (!el || !el?.dataset) return;
              const id = String(el.dataset.id);
              rowRefs.current[id] = el as HTMLDivElement;
            }
          }
        }}
        slots={{ toolbar: CustomToolbar }}
        sx={{
          '& .focused-tenant-row': {
            backgroundColor: (theme) => theme.palette.action.selected,
            '&:hover': { backgroundColor: (theme) => theme.palette.action.hover },
          }
        }}
        initialState={{
          columns: { columnVisibilityModel: { tenantName: true } },
          // @ts-ignore: 'grouping' is a valid property for DataGridPro, but not directly in GridInitialStateCommunity
          // This is a workaround for the current MUI X DataGrid type definitions.
          grouping: { model: ['tenantName'] }, 
        }}
      />

  {/* ERD modal removed */}
    </Box>
  );
};

export default TenantsTable;