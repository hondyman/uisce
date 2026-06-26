import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  IconButton,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Alert,
  Tooltip,
  Card,
  CardContent,
  Grid,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import DeleteIcon from '@mui/icons-material/Delete';
import RefreshIcon from '@mui/icons-material/Refresh';
import LockIcon from '@mui/icons-material/Lock';
import ScheduleIcon from '@mui/icons-material/Schedule';
import SecurityIcon from '@mui/icons-material/Security';
import { useQuery, useMutation } from '@apollo/client';
import { gql } from '@apollo/client';

// GraphQL Queries
const GET_SECRETS = gql`
  query GetSecrets($tenantId: uuid!) {
    secret_metadata(where: { tenant_id: { _eq: $tenantId }, deleted_at: { _is_null: true } }) {
      id
      name
      path
      secret_type
      description
      ttl
      tags
      attributes
      created_at
      updated_at
    }
  }
`;

const INSERT_SECRET = gql`
  mutation InsertSecret($object: secret_metadata_insert_input!) {
    insert_secret_metadata_one(object: $object) {
      id
    }
  }
`;

const UPDATE_SECRET = gql`
  mutation UpdateSecret($id: uuid!, $set: secret_metadata_set_input!) {
    update_secret_metadata_by_pk(pk_columns: { id: $id }, _set: $set) {
      id
    }
  }
`;

const DELETE_SECRET = gql`
  mutation DeleteSecret($id: uuid!) {
    update_secret_metadata_by_pk(pk_columns: { id: $id }, _set: { deleted_at: "now()" }) {
      id
    }
  }
`;

interface SecretMetadata {
  id: string;
  name: string;
  path: string;
  secret_type: string;
  description?: string;
  ttl?: string;
  tags?: string[];
  attributes?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

interface SecretsConfigPageProps {
  tenantId: string;
}

const SECRET_TYPES = [
  { value: 'kv-v2', label: 'Key-Value (Vault)' },
  { value: 'database', label: 'Database Credentials' },
  { value: 'aws', label: 'AWS Secrets Manager' },
  { value: 'azure', label: 'Azure Key Vault' },
  { value: 'api-key', label: 'API Key' },
];

export default function SecretsConfigPage({ tenantId }: SecretsConfigPageProps) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingSecret, setEditingSecret] = useState<SecretMetadata | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    path: '',
    secret_type: 'kv-v2',
    description: '',
    ttl: '',
    tags: '',
  });

  const { data, loading, refetch } = useQuery(GET_SECRETS, {
    variables: { tenantId },
    skip: !tenantId,
  });

  const [insertSecret] = useMutation(INSERT_SECRET);
  const [updateSecret] = useMutation(UPDATE_SECRET);
  const [deleteSecret] = useMutation(DELETE_SECRET);

  const secrets: SecretMetadata[] = data?.secret_metadata || [];

  const handleOpenDialog = (secret?: SecretMetadata) => {
    if (secret) {
      setEditingSecret(secret);
      setFormData({
        name: secret.name,
        path: secret.path,
        secret_type: secret.secret_type,
        description: secret.description || '',
        ttl: secret.ttl || '',
        tags: secret.tags?.join(', ') || '',
      });
    } else {
      setEditingSecret(null);
      setFormData({
        name: '',
        path: '',
        secret_type: 'kv-v2',
        description: '',
        ttl: '',
        tags: '',
      });
    }
    setDialogOpen(true);
  };

  const handleSave = async () => {
    const secretData = {
      tenant_id: tenantId,
      name: formData.name,
      path: formData.path,
      secret_type: formData.secret_type,
      description: formData.description || null,
      ttl: formData.ttl || null,
      tags: formData.tags ? formData.tags.split(',').map((t) => t.trim()) : [],
    };

    try {
      if (editingSecret) {
        await updateSecret({ variables: { id: editingSecret.id, set: secretData } });
      } else {
        await insertSecret({ variables: { object: secretData } });
      }
      setDialogOpen(false);
      refetch();
    } catch (error) {
      console.error('Error saving secret:', error);
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm('Delete this secret configuration?')) return;
    try {
      await deleteSecret({ variables: { id } });
      refetch();
    } catch (error) {
      console.error('Error deleting secret:', error);
    }
  };

  const getTypeChipColor = (type: string) => {
    switch (type) {
      case 'database': return 'primary';
      case 'api-key': return 'secondary';
      case 'aws': return 'warning';
      case 'azure': return 'info';
      default: return 'default';
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <SecurityIcon /> Secrets Configuration
          </Typography>
          <Typography color="text.secondary">
            Manage secret paths, rotation policies, and access controls
          </Typography>
        </Box>
        <Button variant="contained" startIcon={<AddIcon />} onClick={() => handleOpenDialog()}>
          Add Secret
        </Button>
      </Box>

      {/* Stats Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Total Secrets</Typography>
              <Typography variant="h4">{secrets.length}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>With Rotation</Typography>
              <Typography variant="h4">{secrets.filter(s => s.ttl).length}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>Database Creds</Typography>
              <Typography variant="h4">{secrets.filter(s => s.secret_type === 'database').length}</Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>API Keys</Typography>
              <Typography variant="h4">{secrets.filter(s => s.secret_type === 'api-key').length}</Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Secrets Table */}
      <Paper>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Path</TableCell>
              <TableCell>Type</TableCell>
              <TableCell>Rotation</TableCell>
              <TableCell>Tags</TableCell>
              <TableCell>Updated</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={7} align="center">Loading...</TableCell>
              </TableRow>
            ) : secrets.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  No secrets configured. Click "Add Secret" to create one.
                </TableCell>
              </TableRow>
            ) : (
              secrets.map((secret) => (
                <TableRow key={secret.id} hover>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LockIcon fontSize="small" color="action" />
                      {secret.name}
                    </Box>
                  </TableCell>
                  <TableCell>
                    <code style={{ fontSize: '0.85em' }}>{secret.path}</code>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={SECRET_TYPES.find(t => t.value === secret.secret_type)?.label || secret.secret_type}
                      size="small"
                      color={getTypeChipColor(secret.secret_type) as any}
                    />
                  </TableCell>
                  <TableCell>
                    {secret.ttl ? (
                      <Chip icon={<ScheduleIcon />} label={secret.ttl} size="small" variant="outlined" />
                    ) : (
                      <Typography color="text.secondary" variant="caption">None</Typography>
                    )}
                  </TableCell>
                  <TableCell>
                    {secret.tags?.map((tag) => (
                      <Chip key={tag} label={tag} size="small" sx={{ mr: 0.5 }} />
                    ))}
                  </TableCell>
                  <TableCell>
                    {new Date(secret.updated_at).toLocaleDateString()}
                  </TableCell>
                  <TableCell align="right">
                    <Tooltip title="Edit">
                      <IconButton size="small" onClick={() => handleOpenDialog(secret)}>
                        <EditIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Rotate Now">
                      <IconButton size="small" color="primary">
                        <RefreshIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                    <Tooltip title="Delete">
                      <IconButton size="small" color="error" onClick={() => handleDelete(secret.id)}>
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </Tooltip>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </Paper>

      {/* Add/Edit Dialog */}
      <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle>{editingSecret ? 'Edit Secret' : 'Add Secret'}</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              label="Name"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              required
              fullWidth
            />
            <TextField
              label="Path"
              value={formData.path}
              onChange={(e) => setFormData({ ...formData, path: e.target.value })}
              required
              fullWidth
              placeholder="secret/myapp/database"
              helperText="Vault path or cloud secret identifier"
            />
            <FormControl fullWidth>
              <InputLabel>Secret Type</InputLabel>
              <Select
                value={formData.secret_type}
                onChange={(e) => setFormData({ ...formData, secret_type: e.target.value })}
                label="Secret Type"
              >
                {SECRET_TYPES.map((type) => (
                  <MenuItem key={type.value} value={type.value}>{type.label}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <TextField
              label="Description"
              value={formData.description}
              onChange={(e) => setFormData({ ...formData, description: e.target.value })}
              multiline
              rows={2}
              fullWidth
            />
            <TextField
              label="Rotation TTL"
              value={formData.ttl}
              onChange={(e) => setFormData({ ...formData, ttl: e.target.value })}
              placeholder="7 days"
              helperText="Leave empty for no auto-rotation"
              fullWidth
            />
            <TextField
              label="Tags"
              value={formData.tags}
              onChange={(e) => setFormData({ ...formData, tags: e.target.value })}
              placeholder="production, critical, database"
              helperText="Comma-separated tags"
              fullWidth
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDialogOpen(false)}>Cancel</Button>
          <Button onClick={handleSave} variant="contained" disabled={!formData.name || !formData.path}>
            {editingSecret ? 'Update' : 'Create'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
