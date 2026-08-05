import React, { useState, useEffect } from 'react';
import { apiClient } from '../../utils/apiClient';
import {
  Box, Paper, Typography, Button, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, IconButton, Dialog, DialogTitle, DialogContent,
  DialogActions, TextField, MenuItem, Chip, Alert
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import WarningIcon from '@mui/icons-material/Warning';

interface SoDRule {
  id: string;
  role_key_a: string;
  role_key_b: string;
  conflict_type: string;
}

const conflictTypes = [
  'TRADE_INITIATOR_VS_APPROVER',
  'POSITION_MANAGER_VS_RISK_MANAGER',
  'FRONT_OFFICE_VS_BACK_OFFICE',
  'CUSTODIAN_VS_AUDITOR',
  'ADMIN_VS_USER'
];

const commonRoles = [
  'trade_initiator',
  'trade_approver',
  'risk_manager',
  'portfolio_manager',
  'compliance_officer',
  'back_office',
  'admin',
  'auditor'
];

export const SoDRulesManager: React.FC = () => {
  const [rules, setRules] = useState<SoDRule[]>([]);
  const [openModal, setOpenModal] = useState(false);
  const [form, setForm] = useState({
    role_key_a: '',
    role_key_b: '',
    conflict_type: ''
  });

  const fetchRules = async () => {
    try {
      const res = await apiClient.get('/api/compliance/gsifi/sod');
      const data = await res.json();
      setRules(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch SoD rules:', error);
      setRules([]);
    }
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const handleSave = async () => {
    if (form.role_key_a === form.role_key_b) {
      alert('A role cannot conflict with itself');
      return;
    }
    try {
      await apiClient.post('/api/compliance/gsifi/sod', {
        body: JSON.stringify(form)
      });
      setOpenModal(false);
      setForm({ role_key_a: '', role_key_b: '', conflict_type: '' });
      fetchRules();
    } catch (error) {
      console.error('Failed to save SoD rule:', error);
    }
  };

  const handleDelete = async (ruleId: string) => {
    if (!confirm('Are you sure you want to delete this SoD rule?')) return;
    try {
      await apiClient.delete(`/api/compliance/gsifi/sod/${ruleId}`);
      fetchRules();
    } catch (error) {
      console.error('Failed to delete SoD rule:', error);
    }
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            Segregation of Duties (SoD)
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Define role conflict rules to prevent toxic role combinations
          </Typography>
        </Box>
        <Button
          variant="contained"
          color="warning"
          startIcon={<AddIcon />}
          onClick={() => setOpenModal(true)}
        >
          Add Conflict Rule
        </Button>
      </Box>

      <Alert severity="warning" sx={{ mb: 2 }}>
        <strong>Important:</strong> Users cannot be assigned both roles in a conflicting pair.
        SoD violations are checked during role assignment and will be blocked.
      </Alert>

      <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#fff3e0' }}>
              <TableCell sx={{ fontWeight: 600 }}>Role A</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Role B</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Conflict Type</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rules.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} align="center">
                  <Typography color="text.secondary" sx={{ py: 3 }}>
                    No SoD rules configured. Add conflict rules to prevent toxic role combinations.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              rules.map((rule) => (
                <TableRow key={rule.id} hover>
                  <TableCell>
                    <Chip
                      label={rule.role_key_a}
                      size="small"
                      variant="outlined"
                      sx={{ fontFamily: 'monospace' }}
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={rule.role_key_b}
                      size="small"
                      variant="outlined"
                      sx={{ fontFamily: 'monospace' }}
                    />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <WarningIcon fontSize="small" color="warning" />
                      <Typography variant="body2">{rule.conflict_type}</Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <IconButton
                      size="small"
                      color="error"
                      onClick={() => handleDelete(rule.id)}
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

      <Dialog open={openModal} onClose={() => setOpenModal(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Add SoD Conflict Rule</DialogTitle>
        <DialogContent>
          <Alert severity="info" sx={{ mt: 1, mb: 2 }}>
            Select two roles that should never be held by the same user simultaneously.
          </Alert>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
            <TextField
              select
              label="Role A"
              fullWidth
              value={form.role_key_a}
              onChange={(e) => setForm({ ...form, role_key_a: e.target.value })}
            >
              {commonRoles.map((r) => (
                <MenuItem key={r} value={r}>
                  {r}
                </MenuItem>
              ))}
            </TextField>

            <Box sx={{ textAlign: 'center' }}>
              <Typography variant="body2" color="text.secondary">
                cannot be held with
              </Typography>
            </Box>

            <TextField
              select
              label="Role B"
              fullWidth
              value={form.role_key_b}
              onChange={(e) => setForm({ ...form, role_key_b: e.target.value })}
            >
              {commonRoles.map((r) => (
                <MenuItem key={r} value={r}>
                  {r}
                </MenuItem>
              ))}
            </TextField>

            <TextField
              select
              label="Conflict Type"
              fullWidth
              value={form.conflict_type}
              onChange={(e) => setForm({ ...form, conflict_type: e.target.value })}
            >
              {conflictTypes.map((ct) => (
                <MenuItem key={ct} value={ct}>
                  {ct.replace(/_/g, ' ')}
                </MenuItem>
              ))}
            </TextField>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenModal(false)}>Cancel</Button>
          <Button variant="contained" color="warning" onClick={handleSave}>
            Save Rule
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
