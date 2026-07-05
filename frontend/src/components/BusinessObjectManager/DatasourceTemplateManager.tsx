import { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Stack,
  Alert,
  IconButton,
} from '@mui/material';
import { Add as AddIcon, Delete as DeleteIcon } from '@mui/icons-material';
import { fetchPhysicalBackends, createPhysicalBackend } from './bindingWizard.service';
import { useNotification } from '../../hooks/useNotification';

export default function DatasourceTemplateManager() {
  const notification = useNotification();
  const [backends, setBackends] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [saving, setSaving] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const data = await fetchPhysicalBackends();
      setBackends(data);
    } catch (err: any) {
      notification.error('Failed to load physical backends.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleSave = async () => {
    if (!name.trim()) return;
    setSaving(true);
    try {
      await createPhysicalBackend({ backendName: name, description });
      notification.success('Physical backend (Datasource Template) created.');
      setOpen(false);
      setName('');
      setDescription('');
      load();
    } catch (err: any) {
      notification.error(err?.message || 'Failed to create physical backend.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight="bold">
            Datasource Templates (Physical Backends)
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Define templates representing database engines/connections that tenants can inherit.
          </Typography>
        </Box>
        <Button startIcon={<AddIcon />} variant="contained" onClick={() => setOpen(true)}>
          Create Template
        </Button>
      </Stack>

      {loading ? (
        <Typography>Loading templates...</Typography>
      ) : backends.length === 0 ? (
        <Alert severity="info">No datasource templates (physical backends) defined yet.</Alert>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Backend Name</TableCell>
                <TableCell>Description</TableCell>
                <TableCell>Backend ID (UUID)</TableCell>
                <TableCell>Created At</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {backends.map((b) => (
                <TableRow key={b.backendId}>
                  <TableCell style={{ fontWeight: 'bold' }}>{b.backendName}</TableCell>
                  <TableCell>{b.description || '-'}</TableCell>
                  <TableCell style={{ fontFamily: 'monospace', fontSize: '0.85rem' }}>
                    {b.backendId}
                  </TableCell>
                  <TableCell>{b.createdAt ? new Date(b.createdAt).toLocaleDateString() : '-'}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Dialog open={open} onClose={() => setOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create Physical Backend Template</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <TextField
              label="Backend Name"
              placeholder="e.g. Northwinds PostgreSQL, Snowflake Data Warehouse"
              fullWidth
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
            <TextField
              label="Description"
              placeholder="Provide a description of this physical data source template"
              fullWidth
              multiline
              rows={3}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button onClick={handleSave} variant="contained" disabled={saving || !name.trim()}>
            Create
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
