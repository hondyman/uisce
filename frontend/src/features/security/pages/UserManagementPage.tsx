import React, { useEffect, useState } from 'react';
import { Box, Typography, Button, IconButton, Tooltip, Chip, Avatar } from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { Security as SecurityIcon, Refresh as RefreshIcon } from '@mui/icons-material';
import { useUsers } from '../hooks/useUsers';
import { User } from '../types/security';
import { UserRoleAssigner } from '../components/UserRoleAssigner';

export const UserManagementPage: React.FC = () => {
  const { users, loading, error, fetchUsers } = useUsers();
  const [assignerOpen, setAssignerOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);

  useEffect(() => {
    fetchUsers();
  }, [fetchUsers]);

  const handleManageRoles = (user: User) => {
    setSelectedUser(user);
    setAssignerOpen(true);
  };

  const columns: GridColDef[] = [
    { 
        field: 'avatar', 
        headerName: '', 
        width: 60,
        sortable: false,
        renderCell: (params) => (
            <Avatar>{params.row.name?.charAt(0).toUpperCase()}</Avatar>
        )
    },
    { field: 'name', headerName: 'Name', flex: 1 },
    { field: 'email', headerName: 'Email', flex: 1.2 },
    { field: 'role', headerName: 'Primary Role', width: 130 }, // This is the old monolithic column
    { 
        field: 'is_core_admin', 
        headerName: 'Core Admin', 
        width: 120, 
        type: 'boolean'
    },
    { 
        field: 'is_active', 
        headerName: 'Status', 
        width: 120,
        renderCell: (params) => (
             params.value ? <Chip label="Active" color="success" size="small" /> : <Chip label="Inactive" size="small" />
        )
    },
    {
      field: 'actions',
      headerName: 'Actions',
      width: 150,
      sortable: false,
      renderCell: (params: GridRenderCellParams) => (
        <Box>
          <Tooltip title="Manage Roles">
            <IconButton onClick={() => handleManageRoles(params.row as User)} size="small" color="primary">
              <SecurityIcon fontSize="small" />
            </IconButton>
          </Tooltip>
        </Box>
      ),
    },
  ];

  return (
    <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
       <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">User Management</Typography>
        <Box>
           <Button startIcon={<RefreshIcon />} onClick={() => fetchUsers()}>
            Refresh
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
          rows={users}
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

      <UserRoleAssigner
        open={assignerOpen}
        user={selectedUser}
        onClose={() => setAssignerOpen(false)}
      />
    </Box>
  );
};
