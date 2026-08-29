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
  MenuItem,
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

const ROLE_LEVELS: Role['role_level'][] = ['viewer', 'editor', 'admin', 'super_admin'];

export const RoleEditor: React.FC<RoleEditorProps> = ({ open, role, onClose, onSave }) => {
  const [roleKey, setRoleKey] = useState('');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [roleLevel, setRoleLevel] = useState<Role['role_level']>('viewer');
  const [isTemplate, setIsTemplate] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (open) {
      if (role) {
        setRoleKey(role.role_key);
        setName(role.role_name);
        setDescription(role.description || '');
        setRoleLevel(role.role_level);
        setIsTemplate(role.is_template);
      } else {
        setRoleKey('');
        setName('');
        setDescription('');
        setRoleLevel('viewer');
        setIsTemplate(false);
      }
      setError(null);
    }
  }, [open, role]);

  const handleSubmit = async () => {
    if (!name.trim()) {
      setError('Role name is required');
      return;
    }
    if (!role && !roleKey.trim()) {
      setError('Role key is required');
      return;
    }

    setSaving(true);
    setError(null);

    try {
      await onSave({
        role_key: roleKey,
        role_name: name,
        description,
        role_level: roleLevel,
        is_template: isTemplate,
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
          {!role && (
            <TextField
              label="Role Key"
              value={roleKey}
              onChange={(e) => setRoleKey(e.target.value)}
              helperText="Stable identifier, e.g. 'billing_specialist'"
              required
            />
          )}
          <TextField
            label="Role Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            required
          />
          <TextField
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            multiline
            rows={3}
          />
          <TextField
            select
            label="Role Level"
            value={roleLevel}
            onChange={(e) => setRoleLevel(e.target.value as Role['role_level'])}
          >
            {ROLE_LEVELS.map((level) => (
              <MenuItem key={level} value={level}>{level}</MenuItem>
            ))}
          </TextField>
          {!role && (
            <FormControlLabel
              control={
                <Checkbox
                  checked={isTemplate}
                  onChange={(e) => setIsTemplate(e.target.checked)}
                />
              }
              label="Gold Copy Template (inheritable by every tenant — only takes effect when you're scoped to the gold-copy tenant)"
            />
          )}
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
