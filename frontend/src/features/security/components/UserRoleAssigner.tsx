import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  List,
  ListItem,
  ListItemText,
  ListItemSecondaryAction,
  IconButton,
  Typography,
  Box,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  CircularProgress,
  Chip
} from '@mui/material';
import { Delete as DeleteIcon, Add as AddIcon } from '@mui/icons-material';
import { Role, User } from '../types/security';
import { useRoles } from '../hooks/useRoles';
import { useUsers } from '../hooks/useUsers';

interface UserRoleAssignerProps {
  open: boolean;
  user: User | null;
  onClose: () => void;
}

export const UserRoleAssigner: React.FC<UserRoleAssignerProps> = ({ open, user, onClose }) => {
  const { roles: allRoles, fetchRoles: fetchAllRoles } = useRoles();
  const { fetchUserRoles, assignRole, revokeRole, loading } = useUsers();
  const [userRoles, setUserRoles] = useState<Role[]>([]);
  const [selectedroleId, setSelectedRoleId] = useState('');
  const [processing, setProcessing] = useState(false);

  useEffect(() => {
    if (open && user) {
      fetchAllRoles();
      loadUserRoles();
    }
  }, [open, user, fetchAllRoles]);

  const loadUserRoles = async () => {
    if (user) {
      const roles = await fetchUserRoles(user.id);
      setUserRoles(roles);
    }
  };

  const handleAssign = async () => {
    if (!user || !selectedroleId) return;
    setProcessing(true);
    try {
      await assignRole(user.id, selectedroleId);
      await loadUserRoles();
      setSelectedRoleId('');
    } finally {
      setProcessing(false);
    }
  };

  const handleRevoke = async (roleId: string) => {
    if (!user) return;
    if (window.confirm('Revoke this role?')) {
        setProcessing(true);
        try {
            await revokeRole(user.id, roleId);
            await loadUserRoles();
        } finally {
            setProcessing(false);
        }
    }
  };

  // Filter out roles already assigned
  const availableRoles = allRoles.filter(
    (role) => !userRoles.find((ur) => ur.role_id === role.role_id)
  );

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>Manage Roles for {user?.name}</DialogTitle>
      <DialogContent>
        {loading && !userRoles.length ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}><CircularProgress /></Box>
        ) : (
            <>
                <List dense>
                    {userRoles.map((role) => (
                        <ListItem key={role.role_id}>
                            <ListItemText
                                primary={role.role_name}
                                secondary={role.is_global_admin ? 'Global Admin' : 'Tenant Role'}
                            />
                            <ListItemSecondaryAction>
                                <IconButton edge="end" aria-label="delete" onClick={() => handleRevoke(role.role_id)} disabled={processing}>
                                    <DeleteIcon />
                                </IconButton>
                            </ListItemSecondaryAction>
                        </ListItem>
                    ))}
                    {userRoles.length === 0 && (
                        <Typography variant="body2" color="textSecondary" sx={{ py: 2, textAlign: 'center' }}>
                            No roles assigned.
                        </Typography>
                    )}
                </List>
                
                <Box sx={{ mt: 3, display: 'flex', gap: 1 }}>
                    <FormControl fullWidth size="small">
                        <InputLabel>Add Role</InputLabel>
                        <Select
                            value={selectedroleId}
                            label="Add Role"
                            onChange={(e) => setSelectedRoleId(e.target.value)}
                            disabled={processing}
                        >
                            {availableRoles.map((role) => (
                                <MenuItem key={role.role_id} value={role.role_id}>
                                    {role.role_name}
                                </MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    <Button
                        variant="contained"
                        onClick={handleAssign}
                        disabled={!selectedroleId || processing}
                        startIcon={<AddIcon />}
                    >
                        Add
                    </Button>
                </Box>
            </>
        )}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
