import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Paper,
  Typography,
  Box,
  Chip,
  Alert
} from '@mui/material';
import ExtensionIcon from '@mui/icons-material/Extension';
import AddIcon from '@mui/icons-material/Add';

interface CustomAttributeModalProps {
  open: boolean;
  onClose: () => void;
  boId?: string;
  tenantId?: string;
}

export interface CustomAttribute {
  id?: string;
  tenant_id: string;
  bo_id: string;
  attribute_name: string;
  display_name: string;
  data_type: 'STRING' | 'NUMBER' | 'BOOLEAN' | 'DATE' | 'JSON';
  jsonb_path: string;
  is_filterable?: boolean;
}

export const CustomAttributeModal: React.FC<CustomAttributeModalProps> = ({
  open,
  onClose,
  boId = 'customers',
  tenantId = 'core',
}) => {
  const [attributes, setAttributes] = useState<CustomAttribute[]>([]);
  const [loading, setLoading] = useState(false);
  const [form, setForm] = useState<CustomAttribute>({
    tenant_id: tenantId,
    bo_id: boId,
    attribute_name: '',
    display_name: '',
    data_type: 'STRING',
    jsonb_path: '',
    is_filterable: true,
  });
  const [statusMessage, setStatusMessage] = useState<string | null>(null);

  const fetchAttributes = () => {
    setLoading(true);
    fetch(`/api/tenants/custom-attributes?tenant_id=${tenantId}&bo_id=${boId}`)
      .then((res) => res.json())
      .then((data) => {
        setAttributes(data.attributes || []);
        setLoading(false);
      })
      .catch(() => setLoading(false));
  };

  useEffect(() => {
    if (open) {
      fetchAttributes();
    }
  }, [open, boId, tenantId]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!form.attribute_name || !form.display_name) return;

    try {
      const payload = {
        ...form,
        tenant_id: tenantId,
        bo_id: boId,
        jsonb_path: form.jsonb_path || `config->custom->${form.attribute_name}`,
      };

      const res = await fetch('/api/tenants/custom-attributes', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (res.ok) {
        setStatusMessage(`Successfully registered custom field '${form.attribute_name}'!`);
        setForm({
          tenant_id: tenantId,
          bo_id: boId,
          attribute_name: '',
          display_name: '',
          data_type: 'STRING',
          jsonb_path: '',
          is_filterable: true,
        });
        fetchAttributes();
      }
    } catch (err: any) {
      setStatusMessage(`Error: ${err.message}`);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <ExtensionIcon sx={{ color: '#38bdf8' }} />
        Tenant Custom Attribute Registry ({boId})
      </DialogTitle>
      <DialogContent>
        <Typography variant="body2" color="textSecondary" mb={2}>
          Add upgrade-safe custom fields to Business Objects. Custom fields are stored in JSONB overlays without modifying physical database tables.
        </Typography>

        {statusMessage && (
          <Alert severity="info" sx={{ mb: 2 }} onClose={() => setStatusMessage(null)}>
            {statusMessage}
          </Alert>
        )}

        <Paper component="form" onSubmit={handleSubmit} sx={{ p: 2, bgcolor: '#1e293b', border: '1px solid #334155', mb: 3 }}>
          <Typography variant="subtitle2" color="#f8fafc" fontWeight="600" mb={2}>
            Register New Custom Field
          </Typography>
          <Box display="flex" gap={2} flexWrap="wrap" mb={2}>
            <TextField
              label="Technical Name"
              placeholder="e.g. charge_code"
              size="small"
              value={form.attribute_name}
              onChange={(e) => setForm({ ...form, attribute_name: e.target.value })}
              required
              sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
            />
            <TextField
              label="Display Name"
              placeholder="e.g. Internal Charge Code"
              size="small"
              value={form.display_name}
              onChange={(e) => setForm({ ...form, display_name: e.target.value })}
              required
              sx={{ bgcolor: '#0f172a', input: { color: '#fff' } }}
            />
            <FormControl size="small" sx={{ minWidth: 140, bgcolor: '#0f172a' }}>
              <InputLabel sx={{ color: '#94a3b8' }}>Data Type</InputLabel>
              <Select
                value={form.data_type}
                onChange={(e) => setForm({ ...form, data_type: e.target.value as any })}
                sx={{ color: '#fff' }}
              >
                <MenuItem value="STRING">STRING</MenuItem>
                <MenuItem value="NUMBER">NUMBER</MenuItem>
                <MenuItem value="BOOLEAN">BOOLEAN</MenuItem>
                <MenuItem value="DATE">DATE</MenuItem>
                <MenuItem value="JSON">JSON</MenuItem>
              </Select>
            </FormControl>
          </Box>
          <Box display="flex" justifyContent="flex-end">
            <Button type="submit" variant="contained" startIcon={<AddIcon />} sx={{ bgcolor: '#0284c7' }}>
              Register Field
            </Button>
          </Box>
        </Paper>

        <Typography variant="subtitle2" fontWeight="600" mb={1}>
          Registered Custom Attributes ({attributes.length})
        </Typography>
        <Paper sx={{ bgcolor: '#1e293b', border: '1px solid #334155' }}>
          <Table size="small">
            <TableHead sx={{ bgcolor: '#0f172a' }}>
              <TableRow>
                <TableCell sx={{ color: '#94a3b8' }}>Field Name</TableCell>
                <TableCell sx={{ color: '#94a3b8' }}>Display Label</TableCell>
                <TableCell sx={{ color: '#94a3b8' }}>Data Type</TableCell>
                <TableCell sx={{ color: '#94a3b8' }}>JSONB Overlay Path</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {attributes.length > 0 ? (
                attributes.map((attr) => (
                  <TableRow key={attr.id || attr.attribute_name}>
                    <TableCell sx={{ color: '#38bdf8', fontWeight: 600 }}>{attr.attribute_name}</TableCell>
                    <TableCell sx={{ color: '#f8fafc' }}>{attr.display_name}</TableCell>
                    <TableCell>
                      <Chip label={attr.data_type} size="small" sx={{ bgcolor: 'rgba(255,255,255,0.05)', color: '#fff' }} />
                    </TableCell>
                    <TableCell sx={{ color: '#cbd5e1', fontFamily: 'monospace', fontSize: '12px' }}>
                      {attr.jsonb_path}
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={4} align="center" sx={{ color: '#94a3b8', py: 3 }}>
                    No custom attributes registered for {boId} yet.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </Paper>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Close</Button>
      </DialogActions>
    </Dialog>
  );
};
