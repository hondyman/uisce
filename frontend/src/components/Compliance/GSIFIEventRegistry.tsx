import React, { useState, useEffect } from 'react';
import { apiClient } from '../../utils/apiClient';
import {
  Box, Paper, Typography, Button, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, IconButton, Dialog, DialogTitle, DialogContent,
  DialogActions, TextField, MenuItem, Chip
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';

const eventCategories = [
  'Market Risk',
  'Credit Risk',
  'Liquidity Risk',
  'Capital',
  'Trade Lifecycle',
  'Model Risk',
  'Regulatory'
];

interface GSIFIEvent {
  id: string;
  tenant_id: string;
  event_key: string;
  category: string;
  description: string;
  schema_json: string;
  is_active: boolean;
  created_at: string;
}

export const GSIFIEventRegistry: React.FC = () => {
  const [events, setEvents] = useState<GSIFIEvent[]>([]);
  const [openModal, setOpenModal] = useState(false);
  const [form, setForm] = useState({
    event_key: '',
    category: '',
    description: '',
    schema_json: '{}'
  });

  const fetchEvents = async () => {
    try {
      const res = await apiClient.get('/api/compliance/gsifi/events');
      const data = await res.json();
      setEvents(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch G-SIFI events:', error);
      setEvents([]);
    }
  };

  useEffect(() => {
    fetchEvents();
  }, []);

  const handleSave = async () => {
    try {
      await apiClient.post('/api/compliance/gsifi/events', {
        body: JSON.stringify(form)
      });
      setOpenModal(false);
      setForm({ event_key: '', category: '', description: '', schema_json: '{}' });
      fetchEvents();
    } catch (error) {
      console.error('Failed to save event:', error);
    }
  };

  const getCategoryColor = (category: string) => {
    const colors: Record<string, string> = {
      'Market Risk': 'error',
      'Credit Risk': 'warning',
      'Liquidity Risk': 'info',
      'Capital': 'secondary',
      'Trade Lifecycle': 'success',
      'Model Risk': 'default',
      'Regulatory': 'primary'
    };
    return colors[category] || 'default';
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Typography variant="h5" fontWeight={700}>
          G-SIFI Event Registry
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setOpenModal(true)}
        >
          Register New Event
        </Button>
      </Box>

      <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
              <TableCell sx={{ fontWeight: 600 }}>Event Key</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Category</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Description</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Status</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {events.length === 0 ? (
              <TableRow>
                <TableCell colSpan={5} align="center">
                  <Typography color="text.secondary" sx={{ py: 3 }}>
                    No G-SIFI events registered. Click "Register New Event" to add one.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              events.map((evt) => (
                <TableRow key={evt.id} hover>
                  <TableCell sx={{ fontWeight: 600, fontFamily: 'monospace' }}>
                    {evt.event_key}
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={evt.category}
                      size="small"
                      color={getCategoryColor(evt.category) as any}
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>{evt.description || '-'}</TableCell>
                  <TableCell>
                    <Chip
                      label={evt.is_active ? 'Active' : 'Inactive'}
                      size="small"
                      color={evt.is_active ? 'success' : 'default'}
                    />
                  </TableCell>
                  <TableCell>
                    <IconButton size="small">
                      <EditIcon fontSize="small" />
                    </IconButton>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      <Dialog open={openModal} onClose={() => setOpenModal(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Register G-SIFI Event</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <TextField
              label="Event Key"
              placeholder="e.g., VAR_LIMIT_VIOLATION"
              fullWidth
              value={form.event_key}
              onChange={(e) => setForm({ ...form, event_key: e.target.value.toUpperCase().replace(/[^A-Z0-9_]/g, '_') })}
              helperText="Use uppercase with underscores (e.g., VAR_LIMIT_VIOLATION)"
            />
            <TextField
              select
              label="Category"
              fullWidth
              value={form.category}
              onChange={(e) => setForm({ ...form, category: e.target.value })}
            >
              {eventCategories.map((c) => (
                <MenuItem key={c} value={c}>
                  {c}
                </MenuItem>
              ))}
            </TextField>
            <TextField
              label="Description"
              fullWidth
              multiline
              rows={2}
              value={form.description}
              onChange={(e) => setForm({ ...form, description: e.target.value })}
            />
            <TextField
              label="JSON Schema"
              fullWidth
              multiline
              rows={4}
              value={form.schema_json}
              onChange={(e) => setForm({ ...form, schema_json: e.target.value })}
              helperText="JSON schema defining the event payload structure"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenModal(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave}>
            Save Event
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
