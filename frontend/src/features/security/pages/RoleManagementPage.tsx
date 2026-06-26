import React, { useEffect, useState } from 'react';
import { Box, Typography, Button, IconButton, Tooltip, Chip } from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon, Refresh as RefreshIcon } from '@mui/icons-material';
import { useRoles } from '../hooks/useRoles';
import { Role } from '../types/security';
import { RoleEditor } from '../components/RoleEditor';
import { format } from 'date-fns';

export const RoleManagementPage: React.FC = () => {
  const { roles, loading, error, fetchRoles, createRole, updateRole, deleteRole } = useRoles();
  const [editorOpen, setEditorOpen] = useState(false);
  const [editingRole, setEditingRole] = useState<Role | null>(null);

  useEffect(() => {
    fetchRoles();
  }, [fetchRoles]);

  const handleCreate = () => {
    setEditingRole(null);
    setEditorOpen(true);
  };

  const handleEdit = (role: Role) => {
    setEditingRole(role);
    setEditorOpen(true);
  };

  const handleDelete = async (roleId: string) => {
    if (window.confirm('Are you sure you want to delete this role?')) {
      await deleteRole(roleId);
    }
  };

  const handleSave = async (roleData: Partial<Role>) => {
    if (editingRole) {
      await updateRole(editingRole.role_id, roleData);
    } else {
      await createRole(roleData);
    }
  };

  const columns: GridColDef[] = [
    { field: 'role_name', headerName: 'Role Name', flex: 1 },
    { field: 'description', headerName: 'Description', flex: 1.5 },
    {
      field: 'is_global_admin',
      headerName: 'Type',
      width: 150,
      renderCell: (params: GridRenderCellParams) => (
        params.value ? (
          <Chip label="Global Admin" color="secondary" size="small" />
        ) : (
          <Chip label="Tenant Role" size="small" />
        )
      ),
    },
    {
      field: 'created_at',
      headerName: 'Created At',
      width: 180,
      valueFormatter: (params) => {
        if (!params.value) return '';
        return format(new Date(params.value), 'yyyy-MM-dd HH:mm');
      },
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 120,
      sortable: false,
      renderCell: (params: GridRenderCellParams) => (
        <Box>
          <Tooltip title="Edit">
            <IconButton onClick={() => handleEdit(params.row as Role)} size="small">
              <EditIcon fontSize="small" />
            </IconButton>
          </Tooltip>
          <Tooltip title="Delete">
            <IconButton onClick={() => handleDelete(params.row.role_id)} size="small" color="error">
              <DeleteIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      ),
    },
  ];

  return (
    <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">Role Management</Typography>
        <Box>
           <Button startIcon={<RefreshIcon />} onClick={() => fetchRoles()} sx={{ mr: 1 }}>
            Refresh
          </Button>
          <Button variant="contained" startIcon={<AddIcon />} onClick={handleCreate}>
            Create Role
          </Button>
        </Box>
      </Box>

      {error && (
        <Typography color="error" sx={{ mb: 2 }}>
          {error}
        </Typography>
      )}

      <Box sx={{ flexGrow: 1 }}>
        <DataGrid
          rows={roles}
          columns={columns}
          getRowId={(row) => row.role_id}
          loading={loading}
          disableRowSelectionOnClick
          initialState={{
             pagination: { paginationModel: { pageSize: 25 } },
          }}
          pageSizeOptions={[25, 50, 100]}
        />
      </Box>

      <RoleEditor
        open={editorOpen}
        role={editingRole}
        onClose={() => setEditorOpen(false)}
        onSave={handleSave}
      />
    </Box>
  );
};
