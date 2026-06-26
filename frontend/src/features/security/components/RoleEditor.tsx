import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControlLabel,
  Checkbox,
  Box,
  Alert,
} from '@mui/material';
import { Role } from '../types/security';

interface RoleEditorProps {
  open: boolean;
  role?: Role | null; // If null, creates a new role
  onClose: () => void;
  onSave: (role: Partial<Role>) => Promise<void>;
}

export const RoleEditor: React.FC<RoleEditorProps> = ({ open, role, onClose, onSave }) => {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isGlobalAdmin, setIsGlobalAdmin] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      if (role) {
        setName(role.role_name);
        setDescription(role.description || '');
        setIsGlobalAdmin(role.is_global_admin);
      } else {
        setName('');
        setDescription('');
        setIsGlobalAdmin(false);
      }
      setError(null);
    }
  }, [open, role]);

  const handleSubmit = async () => {
    if (!name.trim()) {
      setError('Role name is required');
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await onSave({
        role_name: name,
        description,
        is_global_admin: isGlobalAdmin,
      });
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to save role');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>{role ? 'Edit Role' : 'Create Role'}</DialogTitle>
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
          {error && <Alert severity="error">{error}</Alert>}
          <TextField
            label="Role Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={!!role} // Prevent renaming existing roles if desired, or allow it. Usually IDs are immutable but names might be changeable. Let's allow generic edit unless backend blocks it.
            // Backend API might use name as ID, so be careful. 
            // The current backend uses ID as PK but checks name uniqueness.
            // Requirement from previous error: "role name is required".
            required
          />
          <TextField
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            multiline
            rows={3}
          />
          <FormControlLabel
            control={
              <Checkbox
                checked={isGlobalAdmin}
                onChange={(e) => setIsGlobalAdmin(e.target.checked)}
              />
            }
            label="Is Global Admin"
          />
        </Box>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={saving}>
          Cancel
        </Button>
        <Button onClick={handleSubmit} variant="contained" disabled={saving}>
          {saving ? 'Saving...' : 'Save'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};
