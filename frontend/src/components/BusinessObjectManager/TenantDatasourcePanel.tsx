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
  MenuItem,
  FormControl,
  InputLabel,
  Select,
} from '@mui/material';
import { Add as AddIcon } from '@mui/icons-material';
import { fetchTenantDatasources, createTenantDatasource, fetchPhysicalBackends } from './bindingWizard.service';
import { useNotification } from '../../hooks/useNotification';

export default function TenantDatasourcePanel() {
  const notification = useNotification();
  const [datasources, setDatasources] = useState<any[]>([]);
  const [backends, setBackends] = useState<any[]>([]);
  const [loading, setLoading] = useState(true);
  const [open, setOpen] = useState(false);

  // Form State
  const [host, setHost] = useState('');
  const [port, setPort] = useState(5432);
  const [databaseName, setDatabaseName] = useState('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [saving, setSaving] = useState(false);

  const load = async () => {
    setLoading(true);
    try {
      const [dsData, pbData] = await Promise.all([
        fetchTenantDatasources(),
        fetchPhysicalBackends(),
      ]);
      setDatasources(dsData);
      setBackends(pbData);
    } catch (err: any) {
      notification.error('Failed to load datasource connection configurations.');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    load();
  }, []);

  const handleSave = async () => {
    if (!host || !databaseName) return;
    setSaving(true);
    try {
      await createTenantDatasource({
        datasourceType: 'postgres',
        host,
        port: Number(port),
        databaseName,
        username,
        password,
      });
      notification.success('Tenant connection configuration saved.');
      setOpen(false);
      // Reset form
      setHost('');
      setPort(5432);
      setDatabaseName('');
      setUsername('');
      setPassword('');
      load();
    } catch (err: any) {
      notification.error(err?.message || 'Failed to save connection configuration.');
    } finally {
      setSaving(false);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 3 }}>
        <Box>
          <Typography variant="h5" fontWeight="bold">
            Tenant Datasource Connections
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Configure tenant-specific database connection parameters and credentials.
          </Typography>
        </Box>
        <Button startIcon={<AddIcon />} variant="contained" onClick={() => setOpen(true)}>
          Add Connection
        </Button>
      </Stack>

      {loading ? (
        <Typography>Loading connections...</Typography>
      ) : datasources.length === 0 ? (
        <Alert severity="info">No custom connection configurations saved for this tenant yet.</Alert>
      ) : (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Host</TableCell>
                <TableCell>Port</TableCell>
                <TableCell>Database</TableCell>
                <TableCell>User</TableCell>
                <TableCell>Status</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {datasources.map((ds) => (
                <TableRow key={ds.datasourceId}>
                  <TableCell style={{ fontWeight: 'bold' }}>{ds.host}</TableCell>
                  <TableCell>{ds.port}</TableCell>
                  <TableCell>{ds.databaseName}</TableCell>
                  <TableCell>{ds.username || '-'}</TableCell>
                  <TableCell>
                    <Box
                      component="span"
                      sx={{
                        px: 1,
                        py: 0.5,
                        borderRadius: 1,
                        fontSize: '0.75rem',
                        fontWeight: 'bold',
                        backgroundColor:
                          ds.provisioningStatus === 'PROVISIONED'
                            ? 'success.light'
                            : 'warning.light',
                        color:
                          ds.provisioningStatus === 'PROVISIONED'
                            ? 'success.contrastText'
                            : 'warning.contrastText',
                      }}
                    >
                      {ds.provisioningStatus}
                    </Box>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Dialog open={open} onClose={() => setOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Configure Connection Credentials</DialogTitle>
        <DialogContent>
          <Stack spacing={3} sx={{ mt: 1 }}>
            <FormControl fullWidth>
              <InputLabel>Type</InputLabel>
              <Select value="postgres" label="Type" disabled>
                <MenuItem value="postgres">PostgreSQL</MenuItem>
              </Select>
            </FormControl>
            <TextField
              label="Database Host"
              placeholder="e.g. 100.84.50.65"
              fullWidth
              value={host}
              onChange={(e) => setHost(e.target.value)}
              required
            />
            <TextField
              label="Port"
              type="number"
              fullWidth
              value={port}
              onChange={(e) => setPort(Number(e.target.value))}
            />
            <TextField
              label="Database Name"
              placeholder="e.g. northwinds"
              fullWidth
              value={databaseName}
              onChange={(e) => setDatabaseName(e.target.value)}
              required
            />
            <TextField
              label="Username"
              fullWidth
              value={username}
              onChange={(e) => setUsername(e.target.value)}
            />
            <TextField
              label="Password"
              type="password"
              fullWidth
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Stack>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Cancel</Button>
          <Button
            onClick={handleSave}
            variant="contained"
            disabled={saving || !host || !databaseName}
          >
            Save Connection
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
