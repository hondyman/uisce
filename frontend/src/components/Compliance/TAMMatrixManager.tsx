import React, { useState, useEffect } from 'react';
import { apiClient } from '../../utils/apiClient';
import {
  Box, Paper, Typography, Button, Table, TableBody, TableCell, TableContainer,
  TableHead, TableRow, IconButton, Dialog, DialogTitle, DialogContent,
  DialogActions, TextField, MenuItem, Switch, FormControlLabel, Chip
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';

interface TAMRule {
  id: string;
  asset_class: string;
  currency: string;
  min_amount: number;
  max_amount: number;
  required_approvers: number;
  requires_senior_manager: boolean;
  time_limit_hours: number;
}

const assetClasses = ['equity', 'fx', 'derivatives', 'fixed_income', 'commodity', 'crypto'];
const currencies = ['USD', 'EUR', 'GBP', 'JPY', 'CHF', 'AUD', 'CAD'];

export const TAMMatrixManager: React.FC = () => {
  const [rules, setRules] = useState<TAMRule[]>([]);
  const [openModal, setOpenModal] = useState(false);
  const [form, setForm] = useState({
    asset_class: 'equity',
    currency: 'USD',
    min_amount: 0,
    max_amount: 0,
    required_approvers: 1,
    requires_senior_manager: false,
    time_limit_hours: 24
  });

  const fetchRules = async () => {
    try {
      const res = await apiClient.get('/api/compliance/gsifi/tam');
      const data = await res.json();
      setRules(Array.isArray(data) ? data : []);
    } catch (error) {
      console.error('Failed to fetch TAM rules:', error);
      setRules([]);
    }
  };

  useEffect(() => {
    fetchRules();
  }, []);

  const handleSave = async () => {
    try {
      await apiClient.post('/api/compliance/gsifi/tam', {
        body: JSON.stringify(form)
      });
      setOpenModal(false);
      fetchRules();
    } catch (error) {
      console.error('Failed to save TAM rule:', error);
    }
  };

  const handleDelete = async (ruleId: string) => {
    if (!confirm('Are you sure you want to delete this TAM rule?')) return;
    try {
      await apiClient.delete(`/api/compliance/gsifi/tam/${ruleId}`);
      fetchRules();
    } catch (error) {
      console.error('Failed to delete TAM rule:', error);
    }
  };

  const formatCurrency = (amount: number, currency: string) => {
    if (amount === 0 && form.max_amount === 0) return '$0 (Auto-approve)';
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency,
      minimumFractionDigits: 0,
      maximumFractionDigits: 0
    }).format(amount);
  };

  const getApprovalBadgeColor = (approvers: number) => {
    if (approvers >= 3) return 'error';
    if (approvers === 2) return 'warning';
    return 'success';
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 2 }}>
        <Box>
          <Typography variant="h5" fontWeight={700}>
            Transaction Authorization Matrix (TAM)
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
            Configure approval thresholds for trade transactions
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setOpenModal(true)}
        >
          Add Threshold Rule
        </Button>
      </Box>

      <TableContainer component={Paper} sx={{ borderRadius: 2 }}>
        <Table>
          <TableHead>
            <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
              <TableCell sx={{ fontWeight: 600 }}>Asset Class</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Currency</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Min Amount</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Max Amount</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Approvers</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Senior Mgr</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Time Limit</TableCell>
              <TableCell sx={{ fontWeight: 600 }}>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {rules.length === 0 ? (
              <TableRow>
                <TableCell colSpan={8} align="center">
                  <Typography color="text.secondary" sx={{ py: 3 }}>
                    No TAM rules configured. Click "Add Threshold Rule" to create one.
                  </Typography>
                </TableCell>
              </TableRow>
            ) : (
              rules.map((rule) => (
                <TableRow key={rule.id} hover>
                  <TableCell>
                    <Chip
                      label={rule.asset_class.toUpperCase()}
                      size="small"
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>{rule.currency}</TableCell>
                  <TableCell>
                    {rule.min_amount === 0 ? (
                      <Chip label="Any" size="small" color="default" />
                    ) : (
                      formatCurrency(rule.min_amount, rule.currency)
                    )}
                  </TableCell>
                  <TableCell>
                    {rule.max_amount === 0 ? (
                      <Chip label="No Limit" size="small" color="info" />
                    ) : (
                      formatCurrency(rule.max_amount, rule.currency)
                    )}
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={`${rule.required_approvers} approver${rule.required_approvers > 1 ? 's' : ''}`}
                      size="small"
                      color={getApprovalBadgeColor(rule.required_approvers) as any}
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={rule.requires_senior_manager ? 'Required' : 'Not Required'}
                      size="small"
                      color={rule.requires_senior_manager ? 'warning' : 'default'}
                    />
                  </TableCell>
                  <TableCell>{rule.time_limit_hours}h</TableCell>
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
        <DialogTitle>Add TAM Rule</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                select
                label="Asset Class"
                fullWidth
                value={form.asset_class}
                onChange={(e) => setForm({ ...form, asset_class: e.target.value })}
              >
                {assetClasses.map((ac) => (
                  <MenuItem key={ac} value={ac}>
                    {ac.toUpperCase()}
                  </MenuItem>
                ))}
              </TextField>
              <TextField
                select
                label="Currency"
                fullWidth
                value={form.currency}
                onChange={(e) => setForm({ ...form, currency: e.target.value })}
              >
                {currencies.map((c) => (
                  <MenuItem key={c} value={c}>
                    {c}
                  </MenuItem>
                ))}
              </TextField>
            </Box>

            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                label="Min Amount"
                type="number"
                fullWidth
                value={form.min_amount}
                onChange={(e) => setForm({ ...form, min_amount: parseFloat(e.target.value) || 0 })}
                helperText="Trades above this amount require approval"
              />
              <TextField
                label="Max Amount"
                type="number"
                fullWidth
                value={form.max_amount}
                onChange={(e) => setForm({ ...form, max_amount: parseFloat(e.target.value) || 0 })}
                helperText="Set to 0 for no upper limit"
              />
            </Box>

            <Box sx={{ display: 'flex', gap: 2 }}>
              <TextField
                label="Required Approvers"
                type="number"
                fullWidth
                value={form.required_approvers}
                onChange={(e) => setForm({ ...form, required_approvers: parseInt(e.target.value) || 1 })}
                inputProps={{ min: 1, max: 5 }}
              />
              <TextField
                label="Time Limit (hours)"
                type="number"
                fullWidth
                value={form.time_limit_hours}
                onChange={(e) => setForm({ ...form, time_limit_hours: parseInt(e.target.value) || 24 })}
                inputProps={{ min: 1, max: 168 }}
              />
            </Box>

            <FormControlLabel
              control={
                <Switch
                  checked={form.requires_senior_manager}
                  onChange={(e) => setForm({ ...form, requires_senior_manager: e.target.checked })}
                />
              }
              label="Requires Senior Manager Approval"
            />
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenModal(false)}>Cancel</Button>
          <Button variant="contained" onClick={handleSave}>
            Save Rule
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
