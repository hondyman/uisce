import React, { useEffect, useState } from 'react';
import { Box, Typography, Button, IconButton, Tooltip, Chip } from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { Add as AddIcon, Edit as EditIcon, Delete as DeleteIcon, Refresh as RefreshIcon, ContentCopy as CloneIcon } from '@mui/icons-material';
import { useRoles } from '../hooks/useRoles';
import { Role } from '../types/security';
import { RoleEditor } from '../components/RoleEditor';
import { format } from 'date-fns';

const ORIGIN_LABEL: Record<Role['origin'], { label: string; color: 'secondary' | 'info' | 'default' }> = {
  gold_copy: { label: 'Gold Copy', color: 'secondary' },
  extended: { label: 'Extended', color: 'info' },
  tenant: { label: 'Custom', color: 'default' },
};

export const RoleManagementPage: React.FC = () => {
  const { roles, loading, error, fetchRoles, createRole, updateRole, deleteRole, cloneRole } = useRoles();
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

  const handleClone = async (role: Role) => {
    const roleKey = window.prompt('Role key for the cloned role:', `${role.role_key}_custom`);
    if (!roleKey) return;
    const roleName = window.prompt('Role name for the cloned role:', `${role.role_name} (My Tenant)`) || role.role_name;
    await cloneRole(role.id, roleKey, roleName);
  };

  const handleSave = async (roleData: Partial<Role>) => {
    if (editingRole) {
      await updateRole(editingRole.id, roleData);
    } else {
      await createRole(roleData);
    }
  };

  const columns: GridColDef[] = [
    { field: 'role_name', headerName: 'Role Name', flex: 1 },
    { field: 'role_key', headerName: 'Key', flex: 0.7 },
    { field: 'description', headerName: 'Description', flex: 1.5 },
    { field: 'role_level', headerName: 'Level', width: 120 },
    {
      field: 'origin',
      headerName: 'Origin',
      width: 130,
      renderCell: (params: GridRenderCellParams) => {
        const meta = ORIGIN_LABEL[params.value as Role['origin']] || ORIGIN_LABEL.tenant;
        return <Chip label={meta.label} color={meta.color} size="small" />;
      },
    },
    {
      field: 'created_at',
      headerName: 'Created At',
      width: 180,
      valueFormatter: (params: any) => {
        if (!params.value) return '';
        return format(new Date(params.value), 'yyyy-MM-dd HH:mm');
      },
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 150,
      sortable: false,
      renderCell: (params: GridRenderCellParams) => {
        const role = params.row as Role;
        if (role.origin === 'gold_copy') {
          return (
            <Tooltip title="Clone to my tenant">
              <IconButton onClick={() => handleClone(role)} size="small">
                <CloneIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          );
        }
        return (
          <Box>
            <Tooltip title="Edit">
              <IconButton onClick={() => handleEdit(role)} size="small">
                <EditIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Tooltip title="Delete">
              <IconButton onClick={() => handleDelete(role.id)} size="small" color="error">
                <DeleteIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </Box>
        );
      },
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
          getRowId={(row) => row.id}
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
