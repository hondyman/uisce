/**
 * User Role Assignment Component
 * Assign roles to users within tenant context
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  InputAdornment,
  Alert,
} from '@mui/material';
import {
  Search as SearchIcon,
  PersonAdd as PersonAddIcon,
  Delete as DeleteIcon,
  CheckCircle as CheckCircleIcon,
} from '@mui/icons-material';

interface User {
  id: string;
  email: string;
  name: string;
}

interface UserRole {
  id: string;
  user_id: string;
  role_id: string;
  user: User;
  assigned_at: string;
  expires_at?: string;
  is_active: boolean;
}

interface UserRoleAssignmentProps {
  roleId: string;
  roleName: string;
  tenantId: string;
  datasourceId: string;
  onClose: () => void;
}

export const UserRoleAssignment: React.FC<UserRoleAssignmentProps> = ({
  roleId,
  roleName,
  tenantId,
  datasourceId,
  onClose,
}) => {
  const [assignedUsers, setAssignedUsers] = useState<UserRole[]>([]);
  const [availableUsers, setAvailableUsers] = useState<User[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [isAssigning, setIsAssigning] = useState(false);
  const [selectedUserId, setSelectedUserId] = useState<string>('');
  const [loading, setLoading] = useState(true);

  // Fetch assigned users
  const fetchAssignedUsers = async () => {
    try {
      const response = await fetch(
        `/api/rbac/roles/${roleId}/users?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`
      );
      const data = await response.json();
      setAssignedUsers(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch assigned users:', error);
      setAssignedUsers([]);
    }
  };

  // Fetch available users (not yet assigned)
  const fetchAvailableUsers = async () => {
    try {
      const response = await fetch(
        `/api/users?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`
      );
      const data = await response.json();
      const users = Array.isArray(data) ? data : [];
      
      // Filter out already assigned users
      const assignedUserIds = new Set(assignedUsers.map(ur => ur.user_id));
      setAvailableUsers(users.filter((u: User) => !assignedUserIds.has(u.id)));
    } catch (error) {
      console.error('Failed to fetch users:', error);
      setAvailableUsers([]);
    } finally {
      setLoading(false);
    }
  };

  // Assign role to user
  const assignRole = async (userId: string) => {
    try {
      const response = await fetch(
        `/api/rbac/roles/${roleId}/assign?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            user_id: userId,
            scope_type: 'global',
          }),
        }
      );

      if (response.ok) {
        await fetchAssignedUsers();
        await fetchAvailableUsers();
        setIsAssigning(false);
        setSelectedUserId('');
      }
    } catch (error) {
      console.error('Failed to assign role:', error);
    }
  };

  // Unassign role from user
  const unassignRole = async (userId: string) => {
    if (!confirm('Are you sure you want to remove this role from the user?')) {
      return;
    }

    try {
      await fetch(
        `/api/rbac/roles/${roleId}/unassign/${userId}?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        { method: 'DELETE' }
      );
      await fetchAssignedUsers();
      await fetchAvailableUsers();
    } catch (error) {
      console.error('Failed to unassign role:', error);
    }
  };

  useEffect(() => {
    fetchAssignedUsers();
  }, [roleId, tenantId, datasourceId]);

  useEffect(() => {
    if (assignedUsers.length >= 0) {
      fetchAvailableUsers();
    }
  }, [assignedUsers]);

  const filteredUsers = availableUsers.filter(user =>
    user.name?.toLowerCase().includes(searchTerm.toLowerCase()) ||
    user.email?.toLowerCase().includes(searchTerm.toLowerCase())
  );

  return (
    <Box sx={{ p: 3 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={3}>
        <Box>
          <Typography variant="h5" fontWeight={600}>
            Assign Users to Role
          </Typography>
          <Typography variant="body2" color="text.secondary" mt={0.5}>
            Role: <strong>{roleName}</strong>
          </Typography>
        </Box>
        <Button onClick={onClose} sx={{ textTransform: 'none' }}>
          Close
        </Button>
      </Box>

      <Alert severity="info" sx={{ mb: 3 }}>
        Users assigned to this role will have permissions scoped to this tenant only.
        Super admin roles can be configured separately for cross-tenant access.
      </Alert>

      {/* Assigned Users */}
      <Box mb={4}>
        <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
          <Typography variant="h6" fontWeight={600}>
            Assigned Users ({assignedUsers.length})
          </Typography>
          <Button
            variant="contained"
            startIcon={<PersonAddIcon />}
            onClick={() => setIsAssigning(true)}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Assign User
          </Button>
        </Box>

        <Paper elevation={2} sx={{ borderRadius: 2, overflow: 'hidden' }}>
          <TableContainer>
            <Table>
              <TableHead>
                <TableRow sx={{ bgcolor: 'grey.50' }}>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>User</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>Email</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>Assigned</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>Status</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="subtitle2" fontWeight={600}>Actions</Typography>
                  </TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {assignedUsers.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} align="center" sx={{ py: 4 }}>
                      <Typography color="text.secondary">
                        No users assigned to this role yet
                      </Typography>
                    </TableCell>
                  </TableRow>
                ) : (
                  assignedUsers.map((userRole) => (
                    <TableRow key={userRole.id} hover>
                      <TableCell>
                        <Typography variant="body2" fontWeight={500}>
                          {userRole.user?.name || 'Unknown User'}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {userRole.user?.email || 'N/A'}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Typography variant="body2" color="text.secondary">
                          {new Date(userRole.assigned_at).toLocaleDateString()}
                        </Typography>
                      </TableCell>
                      <TableCell>
                        <Chip
                          icon={<CheckCircleIcon />}
                          label={userRole.is_active ? 'Active' : 'Inactive'}
                          size="small"
                          color={userRole.is_active ? 'success' : 'default'}
                          variant="outlined"
                        />
                      </TableCell>
                      <TableCell>
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => unassignRole(userRole.user_id)}
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      </Box>

      {/* Assign User Dialog */}
      <Dialog
        open={isAssigning}
        onClose={() => setIsAssigning(false)}
        maxWidth="sm"
        fullWidth
        PaperProps={{ elevation: 8, sx: { borderRadius: 2 } }}
      >
        <DialogTitle>
          <Typography variant="h6" fontWeight={600}>
            Assign User to {roleName}
          </Typography>
        </DialogTitle>
        <DialogContent>
          <Box mt={2}>
            <TextField
              fullWidth
              placeholder="Search users..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon />
                  </InputAdornment>
                ),
              }}
              sx={{ mb: 2 }}
            />

            <Box sx={{ maxHeight: 300, overflow: 'auto' }}>
              {loading ? (
                <Typography color="text.secondary" align="center" py={4}>
                  Loading users...
                </Typography>
              ) : filteredUsers.length === 0 ? (
                <Typography color="text.secondary" align="center" py={4}>
                  No available users found
                </Typography>
              ) : (
                filteredUsers.map((user) => (
                  <Paper
                    key={user.id}
                    elevation={selectedUserId === user.id ? 3 : 1}
                    sx={{
                      p: 2,
                      mb: 1,
                      cursor: 'pointer',
                      border: selectedUserId === user.id ? 2 : 0,
                      borderColor: 'primary.main',
                      '&:hover': { bgcolor: 'action.hover' },
                    }}
                    onClick={() => setSelectedUserId(user.id)}
                  >
                    <Typography variant="body2" fontWeight={500}>
                      {user.name}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {user.email}
                    </Typography>
                  </Paper>
                ))
              )}
            </Box>
          </Box>
        </DialogContent>
        <DialogActions sx={{ px: 3, pb: 2 }}>
          <Button onClick={() => setIsAssigning(false)} sx={{ textTransform: 'none' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={() => selectedUserId && assignRole(selectedUserId)}
            disabled={!selectedUserId}
            sx={{ textTransform: 'none', fontWeight: 500 }}
          >
            Assign Role
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
