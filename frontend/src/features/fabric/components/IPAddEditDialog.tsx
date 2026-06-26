import { useState, useEffect } from 'react';
import {
  Dialog,
  DialogContent,
  DialogActions,
  TextField,
  Button,
  // removed unused form components to satisfy lint
  Chip,
  Box,
  Typography,
  Alert,
  // SelectChangeEvent removed - not used
  InputAdornment,
  IconButton,
  Accordion,
  AccordionSummary,
  AccordionDetails
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import {
  Info as InfoIcon,
  ExpandMore as ExpandMoreIcon,
  Help as HelpIcon
} from '@mui/icons-material';
import { Tenant, IPWhitelistEntry, ALL_TENANTS_ID } from '../types/ipWhitelist';
import { 
  isValidIPAddress, 
  getIPPatternDescription, 
  suggestSimilarPatterns,
  formatIPForDisplay
} from '../utils/ipUtils';
import TenantTypeahead from './TenantTypeahead';

interface IPAddEditDialogProps {
  open: boolean;
  onClose: () => void;
  onSave: (ipData: {
    ipAddress: string;
    label?: string;
    description?: string;
    tenantIds: string[];
    allTenants?: boolean;
  }) => Promise<void>;
  tenants: Tenant[];
  editingEntry?: IPWhitelistEntry | null;
  initialTenantId?: string;
}

const IPAddEditDialog: React.FC<IPAddEditDialogProps> = ({
  open,
  onClose,
  onSave,
  tenants,
  editingEntry,
  initialTenantId
}) => {
  const [formData, setFormData] = useState({
    ipAddress: editingEntry?.ipAddress || '',
    label: editingEntry?.label || '',
    description: editingEntry?.description || '',
    tenantIds: editingEntry?.tenantIds || (initialTenantId ? [initialTenantId] : [])
  });
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [conflictWarning, setConflictWarning] = useState<string | null>(null);
  const [allTenants, setAllTenants] = useState<boolean>(() => {
    // If editing and entry has no tenantIds, treat as All Tenants
    if (editingEntry) {
      return ((editingEntry as any).allTenants === true) || (editingEntry.tenantIds?.length === 0);
    }
    return false;
  });

  const isEdit = !!editingEntry;
  const isValidIP = isValidIPAddress(formData.ipAddress);
  const patternDescription = formData.ipAddress ? getIPPatternDescription(formData.ipAddress) : '';
  const suggestions = formData.ipAddress ? suggestSimilarPatterns(formData.ipAddress) : [];

  // Check for potential conflicts (simplified - in real app, would fetch from API)
  useEffect(() => {
    if (formData.ipAddress && isValidIP) {
      // Simulate conflict check - in real app, you'd call an API
      const hasConflict = formData.ipAddress.includes('192.168.1.') && formData.ipAddress !== editingEntry?.ipAddress;
      setConflictWarning(hasConflict ? 'Warning: This IP pattern may overlap with existing entries' : null);
    } else {
      setConflictWarning(null);
    }
  }, [formData.ipAddress, isValidIP, editingEntry?.ipAddress]);

  const handleSubmit = async () => {
    if (!formData.ipAddress.trim()) {
      setError('IP Address is required');
      return;
    }

    if (!isValidIPAddress(formData.ipAddress)) {
      setError('Enter a valid IPv4, wildcard pattern (e.g., 192.168.*.*), or CIDR (e.g., 10.0.0.0/8)');
      return;
    }

    if (!allTenants && formData.tenantIds.length === 0) {
      setError('Select at least one tenant or enable "All Tenants"');
      return;
    }

    setLoading(true);
    setError(null);

    try {
  const allTenantsSelected = allTenants || formData.tenantIds.includes(ALL_TENANTS_ID);
  const tenantIds = allTenantsSelected ? [] : formData.tenantIds;

      await onSave({
        ipAddress: formData.ipAddress.trim(),
        label: formData.label.trim() || undefined,
        description: formData.description.trim() || undefined,
        tenantIds,
        allTenants: allTenantsSelected || undefined,
      });
      
      // Reset form
  setFormData({
        ipAddress: '',
        label: '',
        description: '',
        tenantIds: initialTenantId ? [initialTenantId] : []
      });
  setAllTenants(false);
      
      onClose();
    } catch (err: any) {
      setError(err.message || 'Failed to save IP address');
    } finally {
      setLoading(false);
    }
  };

  const handleCancel = () => {
    setFormData({
      ipAddress: editingEntry?.ipAddress || '',
      label: editingEntry?.label || '',
      description: editingEntry?.description || '',
      tenantIds: editingEntry?.tenantIds || (initialTenantId ? [initialTenantId] : [])
    });
    setError(null);
    onClose();
  };

  useEffect(() => {
    if (open && editingEntry) {
      setFormData({
        ipAddress: editingEntry.ipAddress,
        label: editingEntry.label || '',
        description: editingEntry.description || '',
        tenantIds: editingEntry.tenantIds
      });
  setAllTenants(((editingEntry as any).allTenants === true) || editingEntry.tenantIds.length === 0);
    }
  }, [open, editingEntry]);

  return (
    <Dialog open={open} onClose={handleCancel} maxWidth="md" fullWidth>
      <ModalHeader title={isEdit ? 'Edit IP Address' : 'Add IP Address'} onClose={handleCancel} />
      
      <DialogContent>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3, mt: 2 }}>
          {error && (
            <Alert severity="error" onClose={() => setError(null)}>
              {error}
            </Alert>
          )}

          <TextField
            label="IP Address or Pattern"
            value={formData.ipAddress}
            onChange={(e) => setFormData(prev => ({ ...prev, ipAddress: e.target.value }))}
            fullWidth
            required
            disabled={isEdit} // Don't allow editing IP address
            placeholder="e.g., 192.168.1.1, 192.168.*.*, or 10.0.0.0/8"
            helperText={
              formData.ipAddress && !isValidIP 
                ? "Invalid IP format" 
                : "Supports wildcards (*) and CIDR (a.b.c.d/nn)"
            }
            error={formData.ipAddress !== '' && !isValidIP}
            InputProps={{
              endAdornment: formData.ipAddress && (
                <InputAdornment position="end">
                  <IconButton size="small" onClick={() => setShowAdvanced(!showAdvanced)}>
                    <HelpIcon />
                  </IconButton>
                </InputAdornment>
              )
            }}
          />

          {/* IP Pattern Information */}
          {formData.ipAddress && isValidIP && (
            <Alert severity="info" icon={<InfoIcon />}>
              <Typography variant="body2">
                {patternDescription}
              </Typography>
            </Alert>
          )}

          {/* Conflict Warning */}
          {conflictWarning && (
            <Alert severity="warning">
              <Typography variant="body2">
                {conflictWarning}
              </Typography>
            </Alert>
          )}

          {/* Advanced IP Information */}
          {showAdvanced && formData.ipAddress && isValidIP && suggestions.length > 0 && (
            <Accordion>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography variant="subtitle2">Pattern Suggestions</Typography>
              </AccordionSummary>
              <AccordionDetails>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                  {suggestions.slice(0, 5).map((suggestion, index) => (
                    <Chip
                      key={index}
                      label={formatIPForDisplay(suggestion)}
                      size="small"
                      variant="outlined"
                      onClick={() => setFormData(prev => ({ ...prev, ipAddress: suggestion }))}
                      sx={{ cursor: 'pointer' }}
                    />
                  ))}
                </Box>
              </AccordionDetails>
            </Accordion>
          )}

          <TextField
            label="Label"
            value={formData.label}
            onChange={(e) => setFormData(prev => ({ ...prev, label: e.target.value }))}
            fullWidth
            placeholder="e.g., Office Network, VPN Gateway"
          />

          <TextField
            label="Description"
            value={formData.description}
            onChange={(e) => setFormData(prev => ({ ...prev, description: e.target.value }))}
            fullWidth
            multiline
            rows={3}
            placeholder="Additional details about this IP address or range"
          />

          {/* All Tenants toggle and tenant selector */}
          <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
            <Chip
              label={allTenants ? 'All Tenants: On' : 'All Tenants: Off'}
              color={allTenants ? 'info' : 'default'}
              variant={allTenants ? 'filled' : 'outlined'}
              onClick={() => setAllTenants(!allTenants)}
              clickable
            />
            {!allTenants && (
              <TenantTypeahead
            label="Assigned Tenants"
            value={formData.tenantIds}
            onChange={(tenantIds) => setFormData(prev => ({ ...prev, tenantIds }))}
            tenants={tenants}
            loading={loading}
            multiple={true}
            placeholder="Search and select tenants to assign..."
                helperText={formData.tenantIds.length === 0 ? "Select at least one tenant or enable All Tenants" : undefined}
                error={formData.tenantIds.length === 0}
                allowAllTenants={true}
              />
            )}
          </Box>
        </Box>
      </DialogContent>

      <DialogActions>
        <Button onClick={handleCancel} disabled={loading}>
          Cancel
        </Button>
        <Button 
          onClick={handleSubmit} 
          variant="contained" 
          disabled={loading || !formData.ipAddress.trim() || formData.tenantIds.length === 0}
        >
          {loading ? 'Saving...' : (isEdit ? 'Update' : 'Add')}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export default IPAddEditDialog;
