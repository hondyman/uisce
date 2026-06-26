import React, { useState, useEffect } from 'react';
import {
  Drawer,
  Box,
  Typography,
  TextField,
  FormControl,
  FormControlLabel,
  Checkbox,
  Select,
  MenuItem,
  InputLabel,
  Button,
  Divider,
  IconButton,
  Alert,
  CircularProgress,
  Chip,
  Stack,
  Autocomplete,
} from '@mui/material';
import {
  Close as CloseIcon,
  Save as SaveIcon,
  Refresh as RefreshIcon,
  Functions as FunctionsIcon,
} from '@mui/icons-material';
import { useTenant } from '../../contexts/TenantContext';

// ============================================================================
// Types
// ============================================================================

interface TermWithMetadata {
  term_id: string;
  term_name: string;
  term_title?: string;
  source_column?: string;
  data_type?: string;
  is_calculation?: boolean;
  metadata_id?: string;
  display_name?: string;
  description?: string;
  group_name?: string;
  required: boolean;
  visible: boolean;
  format?: string;
  precision?: number;
  currency_code?: string;
  date_format?: string;
  aggregation?: string;
  sort_order?: number;
}

interface TermMetadataFormData {
  display_name: string;
  description: string;
  group_name: string;
  required: boolean;
  visible: boolean;
  format: string;
  precision: number;
  currency_code: string;
  date_format: string;
  aggregation: string;
  sort_order: number;
}

interface TermMetadataEditorDrawerProps {
  open: boolean;
  onClose: () => void;
  boId: string;
  term: TermWithMetadata | null;
  existingGroups?: string[];
  onSave?: () => void;
}

const FORMAT_OPTIONS = [
  { value: 'string', label: 'Text' },
  { value: 'number', label: 'Number' },
  { value: 'integer', label: 'Integer' },
  { value: 'currency', label: 'Currency' },
  { value: 'percent', label: 'Percentage' },
  { value: 'date', label: 'Date' },
  { value: 'boolean', label: 'Boolean' },
];

const AGGREGATION_OPTIONS = [
  { value: 'none', label: 'None' },
  { value: 'sum', label: 'Sum' },
  { value: 'avg', label: 'Average' },
  { value: 'min', label: 'Minimum' },
  { value: 'max', label: 'Maximum' },
  { value: 'count', label: 'Count' },
];

const DATE_FORMAT_OPTIONS = [
  { value: 'YYYY-MM-DD', label: 'YYYY-MM-DD' },
  { value: 'MM/DD/YYYY', label: 'MM/DD/YYYY' },
  { value: 'DD/MM/YYYY', label: 'DD/MM/YYYY' },
  { value: 'MMM DD, YYYY', label: 'MMM DD, YYYY' },
];

const CURRENCY_OPTIONS = ['USD', 'EUR', 'GBP', 'JPY', 'CAD', 'AUD', 'CHF'];

// ============================================================================
// Component
// ============================================================================

export const TermMetadataEditorDrawer: React.FC<TermMetadataEditorDrawerProps> = ({
  open,
  onClose,
  boId,
  term,
  existingGroups = [],
  onSave,
}) => {
  const { tenant } = useTenant();
  const tenantId = tenant?.id || '';

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [formData, setFormData] = useState<TermMetadataFormData>({
    display_name: '',
    description: '',
    group_name: '',
    required: false,
    visible: true,
    format: 'string',
    precision: 2,
    currency_code: 'USD',
    date_format: 'YYYY-MM-DD',
    aggregation: 'none',
    sort_order: 0,
  });

  // Load metadata when drawer opens
  useEffect(() => {
    if (open && term && boId) {
      loadMetadata();
    }
  }, [open, term, boId]);

  const loadMetadata = async () => {
    if (!term) return;

    setLoading(true);
    setError(null);

    try {
      const response = await fetch(`/api/bo/${boId}/term/${term.term_id}/metadata`, {
        headers: {
          'X-Tenant-ID': tenantId,
        },
      });

      if (!response.ok) {
        throw new Error('Failed to load metadata');
      }

      const data = await response.json();

      setFormData({
        display_name: data.display_name || term.term_title || term.term_name || '',
        description: data.description || '',
        group_name: data.group_name || '',
        required: data.required || false,
        visible: data.visible !== false,
        format: data.format || 'string',
        precision: data.precision ?? 2,
        currency_code: data.currency_code || 'USD',
        date_format: data.date_format || 'YYYY-MM-DD',
        aggregation: data.aggregation || 'none',
        sort_order: data.sort_order ?? 0,
      });
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!term) return;

    setSaving(true);
    setError(null);

    try {
      const response = await fetch(`/api/bo/${boId}/term/${term.term_id}/metadata`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId,
        },
        body: JSON.stringify(formData),
      });

      if (!response.ok) {
        const errText = await response.text();
        throw new Error(errText || 'Failed to save metadata');
      }

      onSave?.();
      onClose();
    } catch (err: any) {
      setError(err.message);
    } finally {
      setSaving(false);
    }
  };

  const handleChange = (field: keyof TermMetadataFormData, value: any) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const isNumericFormat = ['number', 'integer', 'currency', 'percent'].includes(formData.format);

  if (!term) return null;

  return (
    <Drawer
      anchor="right"
      open={open}
      onClose={onClose}
      PaperProps={{
        sx: { width: 480, p: 0 },
      }}
    >
      {/* Header */}
      <Box
        sx={{
          p: 2,
          borderBottom: 1,
          borderColor: 'divider',
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          bgcolor: 'background.default',
        }}
      >
        <Box>
          <Typography variant="h6">Edit Term Metadata</Typography>
          <Typography variant="body2" color="text.secondary">
            {term.term_name}
          </Typography>
        </Box>
        <IconButton onClick={onClose}>
          <CloseIcon />
        </IconButton>
      </Box>

      {/* Content */}
      <Box sx={{ p: 3, overflowY: 'auto', flex: 1 }}>
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
            <CircularProgress />
          </Box>
        )}

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error}
          </Alert>
        )}

        {!loading && (
          <Stack spacing={3}>
            {/* Term Info */}
            {term.is_calculation && (
              <Alert severity="info" icon={<FunctionsIcon />}>
                This is a calculated field. Some settings may be restricted.
              </Alert>
            )}

            <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
              {term.source_column && (
                <Chip label={`Column: ${term.source_column}`} size="small" variant="outlined" />
              )}
              {term.data_type && (
                <Chip label={`Type: ${term.data_type}`} size="small" variant="outlined" />
              )}
            </Box>

            <Divider />

            {/* Section A: Basic Info */}
            <Typography variant="subtitle2" color="text.secondary">
              Basic Information
            </Typography>

            <TextField
              label="Display Name"
              value={formData.display_name}
              onChange={(e) => handleChange('display_name', e.target.value)}
              fullWidth
              helperText="Name shown to end users"
            />

            <TextField
              label="Description"
              value={formData.description}
              onChange={(e) => handleChange('description', e.target.value)}
              fullWidth
              multiline
              rows={2}
              helperText="Tooltip / help text"
            />

            <Autocomplete
              freeSolo
              options={existingGroups}
              value={formData.group_name}
              onChange={(_, value) => handleChange('group_name', value || '')}
              onInputChange={(_, value) => handleChange('group_name', value)}
              renderInput={(params) => (
                <TextField
                  {...params}
                  label="Group / Section"
                  helperText="Group related fields together"
                />
              )}
            />

            <Box sx={{ display: 'flex', gap: 2 }}>
              <FormControlLabel
                control={
                  <Checkbox
                    checked={formData.required}
                    onChange={(e) => handleChange('required', e.target.checked)}
                  />
                }
                label="Required"
              />
              <FormControlLabel
                control={
                  <Checkbox
                    checked={formData.visible}
                    onChange={(e) => handleChange('visible', e.target.checked)}
                  />
                }
                label="Visible"
              />
            </Box>

            <Divider />

            {/* Section B: Formatting */}
            <Typography variant="subtitle2" color="text.secondary">
              Formatting
            </Typography>

            <FormControl fullWidth>
              <InputLabel>Format</InputLabel>
              <Select
                value={formData.format}
                label="Format"
                onChange={(e) => handleChange('format', e.target.value)}
              >
                {FORMAT_OPTIONS.map((opt) => (
                  <MenuItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            {isNumericFormat && (
              <TextField
                label="Decimal Places"
                type="number"
                value={formData.precision}
                onChange={(e) => handleChange('precision', parseInt(e.target.value) || 0)}
                inputProps={{ min: 0, max: 10 }}
                fullWidth
              />
            )}

            {formData.format === 'currency' && (
              <FormControl fullWidth>
                <InputLabel>Currency Code</InputLabel>
                <Select
                  value={formData.currency_code}
                  label="Currency Code"
                  onChange={(e) => handleChange('currency_code', e.target.value)}
                >
                  {CURRENCY_OPTIONS.map((code) => (
                    <MenuItem key={code} value={code}>
                      {code}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            )}

            {formData.format === 'date' && (
              <FormControl fullWidth>
                <InputLabel>Date Format</InputLabel>
                <Select
                  value={formData.date_format}
                  label="Date Format"
                  onChange={(e) => handleChange('date_format', e.target.value)}
                >
                  {DATE_FORMAT_OPTIONS.map((opt) => (
                    <MenuItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            )}

            <Divider />

            {/* Section C: Aggregation */}
            <Typography variant="subtitle2" color="text.secondary">
              Aggregation Behavior
            </Typography>

            <FormControl fullWidth>
              <InputLabel>Aggregation</InputLabel>
              <Select
                value={formData.aggregation}
                label="Aggregation"
                onChange={(e) => handleChange('aggregation', e.target.value)}
              >
                {AGGREGATION_OPTIONS.filter((opt) => {
                  // Count only for non-numeric
                  if (opt.value === 'count') return !isNumericFormat;
                  // Sum/Avg/Min/Max only for numeric
                  if (['sum', 'avg', 'min', 'max'].includes(opt.value)) return isNumericFormat;
                  return true;
                }).map((opt) => (
                  <MenuItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <Divider />

            {/* Section D: Ordering */}
            <Typography variant="subtitle2" color="text.secondary">
              Ordering
            </Typography>

            <TextField
              label="Sort Order"
              type="number"
              value={formData.sort_order}
              onChange={(e) => handleChange('sort_order', parseInt(e.target.value) || 0)}
              fullWidth
              helperText="Lower numbers appear first within the group"
            />
          </Stack>
        )}
      </Box>

      {/* Footer */}
      <Box
        sx={{
          p: 2,
          borderTop: 1,
          borderColor: 'divider',
          display: 'flex',
          justifyContent: 'flex-end',
          gap: 2,
        }}
      >
        <Button onClick={onClose}>Cancel</Button>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={saving}
          startIcon={saving ? <CircularProgress size={16} /> : <SaveIcon />}
        >
          {saving ? 'Saving...' : 'Save Changes'}
        </Button>
      </Box>
    </Drawer>
  );
};

export default TermMetadataEditorDrawer;
