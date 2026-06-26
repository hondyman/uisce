import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
  Grid,
  Card,
  CardContent,
  Alert,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Divider,
} from '@mui/material';
import {
  Add as AddIcon,
  CheckCircle as CheckIcon,
  Warning as WarningIcon,
  Error as ErrorIcon,
  Gavel as TrustIcon,
} from '@mui/icons-material';
import { format } from 'date-fns';

interface Trust {
  entity_id: string;
  entity_name: string;
  entity_type: string;
  formation_date: string;
  formation_state: string;
  is_revocable: boolean;
  current_value: number;
  tax_id_number?: string;
  grantor_names: string[];
  trustee_names: string[];
  beneficiary_names: string[];
  last_tax_filing_date?: string;
  next_tax_filing_due?: string;
}

interface ComplianceIssue {
  issue_type: string;
  severity: 'ERROR' | 'WARNING' | 'INFO';
  description: string;
  recommendation: string;
}

interface TrustManagementPanelProps {
  familyId: string;
}

export const TrustManagementPanel: React.FC<TrustManagementPanelProps> = ({ familyId }) => {
  const [trusts, setTrusts] = useState<Trust[]>([]);
  const [selectedTrust, setSelectedTrust] = useState<Trust | null>(null);
  const [complianceIssues, setComplianceIssues] = useState<ComplianceIssue[]>([]);
  const [newTrustDialog, setNewTrustDialog] = useState(false);
  const [loading, setLoading] = useState(false);

  const [formData, setFormData] = useState({
    entity_type: 'SLAT',
    entity_name: '',
    formation_date: format(new Date(), 'yyyy-MM-dd'),
    formation_state: 'CA',
    is_revocable: false,
    grantor_member_ids: [] as string[],
    trustee_member_ids: [] as string[],
    beneficiary_member_ids: [] as string[],
  });

  useEffect(() => {
    loadTrusts();
  }, [familyId]);

  const loadTrusts = async () => {
    try {
      const response = await fetch(`/api/wealth-transfer/families/${familyId}/trusts`);
      const data = await response.json();
      setTrusts(data || []);
    } catch (error) {
      console.error('Failed to load trusts:', error);
    }
  };

  const loadComplianceIssues = async (trustId: string) => {
    try {
      const response = await fetch(`/api/wealth-transfer/trusts/${trustId}/compliance`);
      const data = await response.json();
      setComplianceIssues(data.issues || []);
    } catch (error) {
      console.error('Failed to load compliance issues:', error);
    }
  };

  const handleCreateTrust = async () => {
    setLoading(true);
    try {
      const response = await fetch('/api/wealth-transfer/trusts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ...formData,
          family_id: familyId,
        }),
      });

      if (response.ok) {
        setNewTrustDialog(false);
        loadTrusts();
        setFormData({
          entity_type: 'SLAT',
          entity_name: '',
          formation_date: format(new Date(), 'yyyy-MM-dd'),
          formation_state: 'CA',
          is_revocable: false,
          grantor_member_ids: [],
          trustee_member_ids: [],
          beneficiary_member_ids: [],
        });
      }
    } catch (error) {
      console.error('Failed to create trust:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleViewTrust = async (trust: Trust) => {
    setSelectedTrust(trust);
    await loadComplianceIssues(trust.entity_id);
  };

  const formatCurrency = (value: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const getTrustTypeDescription = (type: string): string => {
    const descriptions: { [key: string]: string } = {
      'SLAT': 'Spousal Lifetime Access Trust',
      'GRAT': 'Grantor Retained Annuity Trust',
      'QPRT': 'Qualified Personal Residence Trust',
      'ILIT': 'Irrevocable Life Insurance Trust',
      'DYNASTY_TRUST': 'Dynasty Trust',
      'CRT': 'Charitable Remainder Trust',
      'CLT': 'Charitable Lead Trust',
      'QTIP': 'Qualified Terminable Interest Property Trust',
      'GST': 'Generation-Skipping Trust',
      'SNT': 'Special Needs Trust',
    };
    return descriptions[type] || type;
  };

  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'ERROR':
        return <ErrorIcon color="error" />;
      case 'WARNING':
        return <WarningIcon color="warning" />;
      default:
        return <CheckIcon color="info" />;
    }
  };

  const getSeverityColor = (severity: string): 'error' | 'warning' | 'success' => {
    switch (severity) {
      case 'ERROR':
        return 'error';
      case 'WARNING':
        return 'warning';
      default:
        return 'success';
    }
  };

  return (
    <Box>
      {/* Header */}
      <Box sx={{ display: 'flex', gap: 2, mb: 3, alignItems: 'center' }}>
        <Typography variant="h6">Trust & Entity Management</Typography>
        <Box sx={{ flexGrow: 1 }} />
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => setNewTrustDialog(true)}
        >
          Create New Trust
        </Button>
      </Box>

      {/* Summary Cards */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} md={4}>
          <Card elevation={2}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Total Trusts
              </Typography>
              <Typography variant="h4">
                {trusts.length}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Across {new Set(trusts.map(t => t.entity_type)).size} different types
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card elevation={2}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Total Trust Value
              </Typography>
              <Typography variant="h4">
                {formatCurrency(trusts.reduce((sum, t) => sum + t.current_value, 0))}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Aggregate across all trusts
              </Typography>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={4}>
          <Card elevation={2}>
            <CardContent>
              <Typography color="text.secondary" gutterBottom variant="body2">
                Compliance Status
              </Typography>
              <Typography variant="h4" color={trusts.filter(t => !t.tax_id_number && !t.is_revocable).length > 0 ? 'error' : 'success'}>
                {trusts.filter(t => t.tax_id_number || t.is_revocable).length}/{trusts.length}
              </Typography>
              <Typography variant="caption" color="text.secondary">
                Trusts in good standing
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Trusts Table */}
      <TableContainer component={Paper} elevation={2}>
        <Table>
          <TableHead>
            <TableRow sx={{ bgcolor: 'grey.100' }}>
              <TableCell><strong>Trust Name</strong></TableCell>
              <TableCell><strong>Type</strong></TableCell>
              <TableCell><strong>Formation Date</strong></TableCell>
              <TableCell align="right"><strong>Value</strong></TableCell>
              <TableCell><strong>Status</strong></TableCell>
              <TableCell><strong>Tax Filing</strong></TableCell>
              <TableCell><strong>Actions</strong></TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {trusts.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  <Box sx={{ py: 4 }}>
                    <TrustIcon sx={{ fontSize: 48, color: 'grey.400', mb: 2 }} />
                    <Typography color="text.secondary">
                      No trusts created yet. Click "Create New Trust" to get started.
                    </Typography>
                  </Box>
                </TableCell>
              </TableRow>
            ) : (
              trusts.map((trust) => (
                <TableRow key={trust.entity_id} hover>
                  <TableCell>
                    <Typography variant="body2" fontWeight="medium">
                      {trust.entity_name}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption" display="block">
                      {trust.entity_type}
                    </Typography>
                    <Typography variant="caption" color="text.secondary">
                      {getTrustTypeDescription(trust.entity_type)}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {format(new Date(trust.formation_date), 'MMM d, yyyy')}
                    <Typography variant="caption" display="block" color="text.secondary">
                      {trust.formation_state}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography variant="body2" fontWeight="medium">
                      {formatCurrency(trust.current_value)}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    {trust.is_revocable ? (
                      <Chip label="Revocable" size="small" color="default" />
                    ) : (
                      <Chip label="Irrevocable" size="small" color="primary" />
                    )}
                  </TableCell>
                  <TableCell>
                    {trust.last_tax_filing_date ? (
                      <Typography variant="caption">
                        Last: {format(new Date(trust.last_tax_filing_date), 'MM/dd/yyyy')}
                      </Typography>
                    ) : (
                      <Chip label="No Filings" size="small" color="warning" />
                    )}
                  </TableCell>
                  <TableCell>
                    <Button size="small" onClick={() => handleViewTrust(trust)}>
                      View Details
                    </Button>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* Trust Details Dialog */}
      <Dialog
        open={selectedTrust !== null}
        onClose={() => setSelectedTrust(null)}
        maxWidth="md"
        fullWidth
      >
        {selectedTrust && (
          <>
            <DialogTitle>
              {selectedTrust.entity_name}
              <Typography variant="caption" display="block" color="text.secondary">
                {getTrustTypeDescription(selectedTrust.entity_type)}
              </Typography>
            </DialogTitle>
            <DialogContent dividers>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle2" gutterBottom>
                    Trust Information
                  </Typography>
                  <Typography variant="body2">
                    <strong>Formation Date:</strong> {format(new Date(selectedTrust.formation_date), 'MMMM d, yyyy')}
                  </Typography>
                  <Typography variant="body2">
                    <strong>Formation State:</strong> {selectedTrust.formation_state}
                  </Typography>
                  <Typography variant="body2">
                    <strong>Status:</strong> {selectedTrust.is_revocable ? 'Revocable' : 'Irrevocable'}
                  </Typography>
                  <Typography variant="body2">
                    <strong>Current Value:</strong> {formatCurrency(selectedTrust.current_value)}
                  </Typography>
                  {selectedTrust.tax_id_number && (
                    <Typography variant="body2">
                      <strong>Tax ID:</strong> {selectedTrust.tax_id_number}
                    </Typography>
                  )}
                </Grid>

                <Grid item xs={12} md={6}>
                  <Typography variant="subtitle2" gutterBottom>
                    Parties
                  </Typography>
                  <Typography variant="body2">
                    <strong>Grantor(s):</strong> {selectedTrust.grantor_names.join(', ')}
                  </Typography>
                  <Typography variant="body2">
                    <strong>Trustee(s):</strong> {selectedTrust.trustee_names.join(', ')}
                  </Typography>
                  <Typography variant="body2">
                    <strong>Beneficiaries:</strong> {selectedTrust.beneficiary_names.join(', ')}
                  </Typography>
                </Grid>

                {complianceIssues.length > 0 && (
                  <Grid item xs={12}>
                    <Divider sx={{ my: 2 }} />
                    <Typography variant="subtitle2" gutterBottom>
                      Compliance Issues
                    </Typography>
                    <List dense>
                      {complianceIssues.map((issue, index) => (
                        <ListItem key={index}>
                          <ListItemIcon>
                            {getSeverityIcon(issue.severity)}
                          </ListItemIcon>
                          <ListItemText
                            primary={issue.description}
                            secondary={issue.recommendation}
                          />
                        </ListItem>
                      ))}
                    </List>
                  </Grid>
                )}

                {complianceIssues.length === 0 && (
                  <Grid item xs={12}>
                    <Alert severity="success" icon={<CheckIcon />}>
                      No compliance issues detected. Trust is in good standing.
                    </Alert>
                  </Grid>
                )}
              </Grid>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setSelectedTrust(null)}>Close</Button>
              <Button variant="outlined">Update Tax Filing</Button>
              <Button variant="outlined" color="error">Terminate Trust</Button>
            </DialogActions>
          </>
        )}
      </Dialog>

      {/* New Trust Dialog */}
      <Dialog open={newTrustDialog} onClose={() => setNewTrustDialog(false)} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Trust</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 2 }}>
            <TextField
              label="Trust Type"
              select
              value={formData.entity_type}
              onChange={(e) => setFormData({ ...formData, entity_type: e.target.value })}
              fullWidth
            >
              <MenuItem value="SLAT">Spousal Lifetime Access Trust (SLAT)</MenuItem>
              <MenuItem value="GRAT">Grantor Retained Annuity Trust (GRAT)</MenuItem>
              <MenuItem value="ILIT">Irrevocable Life Insurance Trust (ILIT)</MenuItem>
              <MenuItem value="DYNASTY_TRUST">Dynasty Trust</MenuItem>
              <MenuItem value="CRT">Charitable Remainder Trust (CRT)</MenuItem>
              <MenuItem value="QTIP">QTIP Trust</MenuItem>
            </TextField>

            <TextField
              label="Trust Name"
              value={formData.entity_name}
              onChange={(e) => setFormData({ ...formData, entity_name: e.target.value })}
              fullWidth
              placeholder="e.g., Smith Family SLAT 2025"
            />

            <TextField
              label="Formation Date"
              type="date"
              value={formData.formation_date}
              onChange={(e) => setFormData({ ...formData, formation_date: e.target.value })}
              fullWidth
              InputLabelProps={{ shrink: true }}
            />

            <TextField
              label="Formation State"
              select
              value={formData.formation_state}
              onChange={(e) => setFormData({ ...formData, formation_state: e.target.value })}
              fullWidth
            >
              <MenuItem value="CA">California</MenuItem>
              <MenuItem value="NY">New York</MenuItem>
              <MenuItem value="FL">Florida</MenuItem>
              <MenuItem value="TX">Texas</MenuItem>
              <MenuItem value="NV">Nevada</MenuItem>
              <MenuItem value="DE">Delaware</MenuItem>
            </TextField>

            <TextField
              label="Revocable?"
              select
              value={formData.is_revocable.toString()}
              onChange={(e) => setFormData({ ...formData, is_revocable: e.target.value === 'true' })}
              fullWidth
            >
              <MenuItem value="false">Irrevocable (most estate planning trusts)</MenuItem>
              <MenuItem value="true">Revocable (living trust)</MenuItem>
            </TextField>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setNewTrustDialog(false)}>Cancel</Button>
          <Button
            variant="contained"
            onClick={handleCreateTrust}
            disabled={loading || !formData.entity_name}
          >
            {loading ? 'Creating...' : 'Create Trust'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
