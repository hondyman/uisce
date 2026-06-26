import React, { useEffect, useState } from 'react';
import {
  Box,
  Button,
  Container,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  Chip,
  IconButton
} from '@mui/material';
import RefreshIcon from '@mui/icons-material/Refresh';
import EditIcon from '@mui/icons-material/Edit';

interface QuotaDef {
  Limit: number;
  Window: number;
}

// Map<TenantID, Map<Resource, QuotaDef>>
type QuotasMap = Record<string, Record<string, QuotaDef>>;

interface FlatQuota {
  tenantID: string;
  resource: string;
  limit: number;
  window: number;
}

export const CapacityManagementPage: React.FC = () => {
  const [quotas, setQuotas] = useState<FlatQuota[]>([]);
  const [loading, setLoading] = useState(false);
  const [editOpen, setEditOpen] = useState(false);
  const [currentQuota, setCurrentQuota] = useState<FlatQuota | null>(null);

  const fetchQuotas = async () => {
    setLoading(true);
    try {
      const res = await fetch('/api/admin/quotas');
      if (res.ok) {
        const data: QuotasMap = await res.json();
        // Flatten
        const flat: FlatQuota[] = [];
        Object.entries(data).forEach(([tenantID, resources]) => {
          Object.entries(resources).forEach(([resource, def]) => {
            flat.push({
              tenantID,
              resource,
              limit: def.Limit,
              window: def.Window
            });
          });
        });
        setQuotas(flat);
      }
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchQuotas();
  }, []);

  const handleEdit = (quota: FlatQuota) => {
    setCurrentQuota({ ...quota });
    setEditOpen(true);
  };

  const handleSave = async () => {
    if (!currentQuota) return;
    try {
      const res = await fetch('/api/admin/quotas', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          tenant_id: currentQuota.tenantID,
          resource: currentQuota.resource,
          limit_value: Number(currentQuota.limit),
          window_seconds: Number(currentQuota.window)
        })
      });
      if (res.ok) {
        setEditOpen(false);
        fetchQuotas();
      } else {
        alert('Failed to update quota');
      }
    } catch (err) {
      console.error(err);
    }
  };

  return (
    <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
        <Typography variant="h4" component="h1" sx={{ fontWeight: 'bold' }}>
          Capacity Management
        </Typography>
        <Button startIcon={<RefreshIcon />} onClick={fetchQuotas} variant="outlined">
          Refresh
        </Button>
      </Box>

      <TableContainer component={Paper} elevation={2}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>Tenant</TableCell>
              <TableCell>Resource</TableCell>
              <TableCell align="right">Limit</TableCell>
              <TableCell align="right">Window (s)</TableCell>
              <TableCell align="right">Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {quotas.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center">No quotas defined or loading...</TableCell>
              </TableRow>
            ) : (
              quotas.map((q) => (
                <TableRow key={`${q.tenantID}-${q.resource}`}>
                  <TableCell>
                    <Chip label={q.tenantID} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell>{q.resource}</TableCell>
                  <TableCell align="right">{q.limit}</TableCell>
                  <TableCell align="right">{q.window}s</TableCell>
                  <TableCell align="right">
                    <IconButton size="small" onClick={() => handleEdit(q)}>
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={editOpen} onClose={() => setEditOpen(false)}>
        <DialogTitle>Edit Quota</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1, minWidth: 300 }}>
             <TextField
              label="Tenant ID"
              value={currentQuota?.tenantID}
              disabled
              variant="filled"
            />
            <TextField
              label="Resource"
              value={currentQuota?.resource}
              disabled
              variant="filled"
            />
            <TextField
              label="Limit"
              type="number"
              value={currentQuota?.limit}
              onChange={(e) => setCurrentQuota(prev => prev ? { ...prev, limit: Number(e.target.value) } : null)}
            />
            <TextField
              label="Window (seconds)"
              type="number"
              value={currentQuota?.window}
              onChange={(e) => setCurrentQuota(prev => prev ? { ...prev, window: Number(e.target.value) } : null)}
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setEditOpen(false)}>Cancel</Button>
          <Button onClick={handleSave} variant="contained">Save</Button>
        </DialogActions>
      </Dialog>
    </Container>
  );
};
