import React, { useState, useEffect } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Box,
  TableContainer,
  Paper,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  IconButton,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Checkbox,
  FormControlLabel,
  Typography,
  Chip,
  Tooltip,
  FormHelperText,
  Switch,
  Divider,
} from '@mui/material';
import { useTheme } from '@mui/material/styles';
import {
  SlidersHorizontal,
  Plus,
  Trash2,
  Edit2,
  Sparkles,
  UserCheck,
  Database,
  Lock,
  Calendar,
} from 'lucide-react';
import { ReportParameter, ParameterType } from './builderSerialization';

type Props = {
  open: boolean;
  onClose: () => void;
  parameters: ReportParameter[];
  onAdd: (param: Omit<ReportParameter, 'id'>) => void;
  onUpdate: (param: ReportParameter) => void;
  onDelete: (paramId: string) => void;
  isReadOnly?: boolean;
  onClone?: () => void;
};

const PARAMETER_PRESETS: Omit<ReportParameter, 'id'>[] = [
  { name: 'Year', type: 'number', prompt: 'Reporting Calendar Year', defaultValue: String(new Date().getFullYear()), allowBlank: false, allowMultiple: false, sourceType: 'manual' },
  { name: 'AccountID', type: 'string', prompt: 'Assigned User Account', defaultValue: '', userContextKey: 'accountId', lockForUser: true, sourceType: 'context', allowBlank: false },
  { name: 'ClientID', type: 'string', prompt: 'Client Identifier', defaultValue: 'client-001', userContextKey: 'clientId', sourceType: 'context', allowBlank: false },
  { name: 'StartDate', type: 'date', prompt: 'Valuation Start Date', defaultValue: 'START_OF_YEAR', sourceType: 'manual', allowBlank: false },
  { name: 'EndDate', type: 'date', prompt: 'Valuation End Date', defaultValue: 'TODAY', sourceType: 'manual', allowBlank: false },
  { name: 'Status', type: 'string', prompt: 'Account / Asset Status', defaultValue: 'Active', sourceType: 'manual', allowBlank: true },
  { name: 'BranchCode', type: 'string', prompt: 'Assigned Branch / Region', defaultValue: '', userContextKey: 'branchId', lockForUser: true, sourceType: 'context', allowBlank: false },
  { name: 'PortfolioList', type: 'list', prompt: 'Select Portfolios', defaultValue: '', allowMultiple: true, sourceType: 'query', querySql: 'SELECT id, name FROM oms.portfolio ORDER BY name', allowBlank: true },
];

const ParameterEditor: React.FC<{
  param: Partial<ReportParameter> | null;
  onSave: (param: any) => void;
  onCancel: () => void;
}> = ({ param, onSave, onCancel }) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const [formData, setFormData] = useState<Partial<ReportParameter>>({});

  useEffect(() => {
    setFormData(param || {
      name: '',
      type: 'string',
      prompt: '',
      defaultValue: '',
      allowBlank: false,
      allowMultiple: false,
      sourceType: 'manual',
    });
  }, [param]);

  const handleChange = (field: keyof ReportParameter, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
  };

  if (!param) return null;

  const C = {
    bg: isDark ? '#0B132B' : '#FFFFFF',
    bgAlt: isDark ? 'rgba(15, 23, 42, 0.6)' : '#F8FAFC',
    text: isDark ? '#F8FAFC' : '#0F172A',
    textMuted: isDark ? '#94A3B8' : '#64748B',
    border: isDark ? 'rgba(255,255,255,0.1)' : 'rgba(0,0,0,0.1)',
    accent: '#0D9488',
    accentLight: isDark ? '#2DD4BF' : '#0D9488',
  };

  return (
    <Dialog
      open={!!param}
      onClose={onCancel}
      maxWidth="sm"
      fullWidth
      PaperProps={{
        sx: {
          bgcolor: C.bg,
          color: C.text,
          borderRadius: 2,
          border: `1px solid ${C.border}`,
        },
      }}
    >
      <DialogTitle sx={{ fontWeight: 800, color: C.text, borderBottom: `1px solid ${C.border}` }}>
        {param.id ? 'Edit Enterprise Report Parameter' : 'Add Enterprise Report Parameter'}
      </DialogTitle>
      <DialogContent sx={{ p: 2.5, pt: 3 }}>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
          {/* Parameter Name & Data Type */}
          <Box sx={{ display: 'flex', gap: 2 }}>
            <TextField
              fullWidth
              size="small"
              label="Parameter Name"
              value={formData.name || ''}
              onChange={(e) => handleChange('name', e.target.value.replace(/[^a-zA-Z0-9_]/g, ''))}
              helperText={formData.name ? `Reference as @${formData.name}` : 'Alphanumeric name (e.g. AccountID)'}
              sx={{
                '& .MuiInputBase-input': { color: C.text, fontFamily: 'monospace', fontWeight: 700 },
                '& label': { color: C.textMuted },
              }}
            />

            <FormControl fullWidth size="small">
              <InputLabel sx={{ color: C.textMuted }}>Data Type</InputLabel>
              <Select
                value={formData.type || 'string'}
                label="Data Type"
                onChange={(e) => handleChange('type', e.target.value)}
                sx={{ color: C.text, '& .MuiSvgIcon-root': { color: C.text } }}
              >
                <MenuItem value="string">String (Text)</MenuItem>
                <MenuItem value="number">Number (Numeric)</MenuItem>
                <MenuItem value="date">Date (YYYY-MM-DD)</MenuItem>
                <MenuItem value="boolean">Boolean (True / False)</MenuItem>
                <MenuItem value="list">List / Array (Options)</MenuItem>
              </Select>
            </FormControl>
          </Box>

          {/* Display Label / Prompt */}
          <TextField
            fullWidth
            size="small"
            label="Prompt / Display Label"
            value={formData.prompt || ''}
            onChange={(e) => handleChange('prompt', e.target.value)}
            placeholder="e.g. Select Assigned Account"
            sx={{
              '& .MuiInputBase-input': { color: C.text },
              '& label': { color: C.textMuted },
            }}
          />

          {/* Source Mode Selector */}
          <FormControl fullWidth size="small">
            <InputLabel sx={{ color: C.textMuted }}>Parameter Source Mode</InputLabel>
            <Select
              value={formData.sourceType || 'manual'}
              label="Parameter Source Mode"
              onChange={(e) => handleChange('sourceType', e.target.value)}
              sx={{ color: C.text, '& .MuiSvgIcon-root': { color: C.text } }}
            >
              <MenuItem value="manual">Manual / Static Default Input</MenuItem>
              <MenuItem value="context">User Context Auto-Binding (Personalization)</MenuItem>
              <MenuItem value="query">Dynamic SQL Query (Cascading Dropdown)</MenuItem>
            </Select>
            <FormHelperText sx={{ color: C.textMuted }}>
              {formData.sourceType === 'context'
                ? 'Automatically populates value from the authenticated user session.'
                : formData.sourceType === 'query'
                ? 'Executes a background query to fetch available choices.'
                : 'Standard consumer-input parameter with a default fallback.'}
            </FormHelperText>
          </FormControl>

          {/* Context Mapping */}
          {formData.sourceType === 'context' && (
            <Paper sx={{ p: 2, bgcolor: C.bgAlt, border: `1px solid ${C.border}`, borderRadius: 1.5 }}>
              <Typography variant="caption" sx={{ fontWeight: 800, color: C.accentLight, display: 'flex', alignItems: 'center', gap: 0.8, mb: 1.5 }}>
                <UserCheck size={14} /> User Profile Session Mapping
              </Typography>
              <FormControl fullWidth size="small">
                <InputLabel sx={{ color: C.textMuted }}>User Context Key</InputLabel>
                <Select
                  value={formData.userContextKey || ''}
                  label="User Context Key"
                  onChange={(e) => handleChange('userContextKey', e.target.value)}
                  sx={{ color: C.text }}
                >
                  <MenuItem value="accountId">User Account ID (accountId)</MenuItem>
                  <MenuItem value="clientId">Client Identifier (clientId)</MenuItem>
                  <MenuItem value="branchId">Branch / Region Code (branchId)</MenuItem>
                  <MenuItem value="tenantId">Tenant Organization ID (tenantId)</MenuItem>
                  <MenuItem value="user.id">User Primary Key (userId)</MenuItem>
                  <MenuItem value="region">Geographic Region (region)</MenuItem>
                </Select>
              </FormControl>

              <FormControlLabel
                sx={{ mt: 1.5 }}
                control={
                  <Switch
                    size="small"
                    checked={!!formData.lockForUser}
                    onChange={(e) => handleChange('lockForUser', e.target.checked)}
                  />
                }
                label={<Typography variant="caption" sx={{ color: C.text }}>Lock Parameter (Read-only for consumers to enforce security)</Typography>}
              />
            </Paper>
          )}

          {/* SQL Query Dynamic Options */}
          {formData.sourceType === 'query' && (
            <Paper sx={{ p: 2, bgcolor: C.bgAlt, border: `1px solid ${C.border}`, borderRadius: 1.5 }}>
              <Typography variant="caption" sx={{ fontWeight: 800, color: C.accentLight, display: 'flex', alignItems: 'center', gap: 0.8, mb: 1 }}>
                <Database size={14} /> Dynamic Options SQL Query
              </Typography>
              <TextField
                fullWidth
                multiline
                rows={2}
                size="small"
                value={formData.querySql || ''}
                onChange={(e) => handleChange('querySql', e.target.value)}
                placeholder="SELECT id AS value, name AS label FROM oms.account ORDER BY name"
                sx={{
                  '& .MuiInputBase-input': { color: C.text, fontFamily: 'monospace', fontSize: '0.78rem' },
                }}
              />
            </Paper>
          )}

          {/* Default Value & Relative Date Shortcuts */}
          <Box>
            <TextField
              fullWidth
              size="small"
              label="Default Fallback Value"
              value={formData.defaultValue || ''}
              onChange={(e) => handleChange('defaultValue', e.target.value)}
              placeholder={formData.type === 'date' ? 'e.g. TODAY, YTD, START_OF_MONTH' : 'e.g. 2026 or client-001'}
              helperText={formData.type === 'date' ? 'Supports relative keywords: TODAY, YTD, START_OF_MONTH, PREV_MONTH, PREV_QUARTER' : undefined}
              sx={{
                '& .MuiInputBase-input': { color: C.text, fontFamily: 'monospace' },
                '& label': { color: C.textMuted },
              }}
            />
            {formData.type === 'date' && (
              <Box sx={{ display: 'flex', gap: 0.5, mt: 1, flexWrap: 'wrap' }}>
                {['TODAY', 'YTD', 'START_OF_MONTH', 'PREV_MONTH', 'PREV_QUARTER'].map((kw) => (
                  <Chip
                    key={kw}
                    size="small"
                    label={kw}
                    onClick={() => handleChange('defaultValue', kw)}
                    sx={{ height: 18, fontSize: '0.62rem', cursor: 'pointer', bgcolor: formData.defaultValue === kw ? C.accent : C.bgAlt, color: formData.defaultValue === kw ? '#FFF' : C.textMuted }}
                  />
                ))}
              </Box>
            )}
          </Box>

          {/* Checkboxes */}
          <Box sx={{ display: 'flex', gap: 2 }}>
            <FormControlLabel
              control={<Checkbox size="small" checked={!!formData.allowBlank} onChange={(e) => handleChange('allowBlank', e.target.checked)} sx={{ color: C.textMuted }} />}
              label={<Typography variant="caption" sx={{ color: C.text }}>Allow Blank / Null</Typography>}
            />
            <FormControlLabel
              control={<Checkbox size="small" checked={!!formData.allowMultiple} onChange={(e) => handleChange('allowMultiple', e.target.checked)} sx={{ color: C.textMuted }} />}
              label={<Typography variant="caption" sx={{ color: C.text }}>Multi-Select (IN Clause Array)</Typography>}
            />
          </Box>
        </Box>
      </DialogContent>
      <DialogActions sx={{ p: 2, borderTop: `1px solid ${C.border}` }}>
        <Button onClick={onCancel} sx={{ color: C.textMuted, textTransform: 'none' }}>
          Cancel
        </Button>
        <Button
          onClick={() => onSave(formData)}
          variant="contained"
          disabled={!formData.name}
          sx={{ bgcolor: C.accent, color: '#FFF', fontWeight: 800, textTransform: 'none', '&:hover': { bgcolor: '#0F766E' } }}
        >
          Save Parameter
        </Button>
      </DialogActions>
    </Dialog>
  );
};

export const ParametersDialog: React.FC<Props> = ({
  open,
  onClose,
  parameters,
  onAdd,
  onUpdate,
  onDelete,
  isReadOnly = false,
  onClone,
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  const [editingParam, setEditingParam] = useState<Partial<ReportParameter> | null>(null);

  const C = {
    bg: isDark ? '#0B132B' : '#FFFFFF',
    bgAlt: isDark ? 'rgba(15, 23, 42, 0.6)' : '#F8FAFC',
    cardBg: isDark ? 'rgba(15, 23, 42, 0.6)' : '#FFFFFF',
    text: isDark ? '#F8FAFC' : '#0F172A',
    textMuted: isDark ? '#94A3B8' : '#64748B',
    border: isDark ? 'rgba(255,255,255,0.08)' : 'rgba(0,0,0,0.08)',
    accent: '#0D9488',
    accentLight: isDark ? '#2DD4BF' : '#0D9488',
    accentBg: isDark ? 'rgba(13, 148, 136, 0.15)' : 'rgba(13, 148, 136, 0.08)',
  };

  const handleSave = (paramData: ReportParameter) => {
    if (paramData.id) {
      onUpdate(paramData);
    } else {
      onAdd(paramData);
    }
    setEditingParam(null);
  };

  const handleAddPreset = (preset: Omit<ReportParameter, 'id'>) => {
    onAdd(preset);
  };

  return (
    <>
      <Dialog
        open={open}
        onClose={onClose}
        maxWidth="md"
        fullWidth
        PaperProps={{
          sx: {
            bgcolor: C.bg,
            color: C.text,
            borderRadius: 2.5,
            border: `1px solid ${C.border}`,
          },
        }}
      >
        <DialogTitle sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', borderBottom: `1px solid ${C.border}`, pb: 1.5 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
            <SlidersHorizontal size={22} color={C.accentLight} />
            <Box>
              <Typography variant="subtitle1" sx={{ fontWeight: 800, color: C.text }}>
                Enterprise Parameters &amp; User Context Engine
              </Typography>
              <Typography variant="caption" sx={{ color: C.textMuted }}>
                {isReadOnly
                  ? 'Core Report Template Parameters (Read-Only for client tenants).'
                  : 'Configure query variables, dynamic SQL dropdowns, and automatic user profile personalization.'}
              </Typography>
            </Box>
          </Box>
          {isReadOnly ? (
            onClone && (
              <Button
                variant="contained"
                size="small"
                onClick={() => {
                  onClose();
                  onClone();
                }}
                sx={{ bgcolor: C.accent, color: '#FFF', fontWeight: 800, textTransform: 'none', borderRadius: 1.5 }}
              >
                Clone to Edit
              </Button>
            )
          ) : (
            <Button
              variant="contained"
              size="small"
              startIcon={<Plus size={15} />}
              onClick={() => setEditingParam({})}
              sx={{ bgcolor: C.accent, color: '#FFF', fontWeight: 800, textTransform: 'none', borderRadius: 1.5 }}
            >
              Add Parameter
            </Button>
          )}
        </DialogTitle>

        <DialogContent sx={{ p: 3 }}>
          {/* Read-Only Notice for Core Template */}
          {isReadOnly && (
            <Paper sx={{ p: 1.5, mb: 2, bgcolor: isDark ? 'rgba(245, 158, 11, 0.12)' : '#FEF3C7', border: '1px solid rgba(245, 158, 11, 0.35)', borderRadius: 1.5 }}>
              <Typography variant="caption" sx={{ color: isDark ? '#FCD34D' : '#92400E', fontWeight: 700, display: 'block' }}>
                🔒 Core Report Template (Read-Only Parameters):
              </Typography>
              <Typography variant="caption" sx={{ color: isDark ? '#FDE68A' : '#78350F', display: 'block', mt: 0.2 }}>
                Parameters on master templates cannot be modified directly by client tenants. You can provide runtime values in the Preview tab, or click <strong>Clone to Edit</strong> to create a customizable copy for your tenant.
              </Typography>
            </Paper>
          )}

          {/* Quick Presets Bar (Hidden if Read-Only) */}
          {!isReadOnly && (
            <Box sx={{ mb: 2.5, p: 1.5, bgcolor: C.bgAlt, border: `1px solid ${C.border}`, borderRadius: 2 }}>
              <Typography variant="caption" sx={{ color: C.textMuted, fontWeight: 700, display: 'flex', alignItems: 'center', gap: 0.5, mb: 1 }}>
                <Sparkles size={13} color={C.accentLight} /> Enterprise Parameter Templates:
              </Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.8 }}>
                {PARAMETER_PRESETS.map((preset) => {
                  const alreadyExists = parameters.some((p) => p.name.toLowerCase() === preset.name.toLowerCase());
                  return (
                    <Chip
                      key={preset.name}
                      size="small"
                      label={`+ ${preset.name} (${preset.sourceType === 'context' ? 'User Context' : preset.type})`}
                      disabled={alreadyExists}
                      onClick={() => !alreadyExists && handleAddPreset(preset)}
                      sx={{
                        cursor: alreadyExists ? 'default' : 'pointer',
                        fontSize: '0.72rem',
                        fontWeight: 700,
                        fontFamily: 'monospace',
                        bgcolor: alreadyExists ? (isDark ? 'rgba(255,255,255,0.04)' : '#E2E8F0') : C.accentBg,
                        color: alreadyExists ? C.textMuted : C.accentLight,
                        border: `1px solid ${alreadyExists ? 'transparent' : C.accentBg}`,
                        '&:hover': alreadyExists ? {} : { bgcolor: C.accent, color: '#FFF' },
                      }}
                    />
                  );
                })}
              </Box>
            </Box>
          )}

          {/* Parameters Table */}
          {parameters.length === 0 ? (
            <Paper sx={{ p: 4, textAlign: 'center', bgcolor: C.bgAlt, border: `1px dashed ${C.border}`, borderRadius: 2 }}>
              <Typography variant="body2" sx={{ color: C.textMuted, mb: 1.5 }}>
                No parameters defined for this report.
              </Typography>
              <Typography variant="caption" sx={{ color: C.textMuted, display: 'block', mb: 2 }}>
                Parameters allow users to filter report queries at runtime or pass arguments when executing via the SDK / REST API.
              </Typography>
              {!isReadOnly && (
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={<Plus size={14} />}
                  onClick={() => setEditingParam({})}
                  sx={{ borderColor: C.accentLight, color: C.accentLight, textTransform: 'none', fontWeight: 700 }}
                >
                  Create Custom Parameter
                </Button>
              )}
            </Paper>
          ) : (
            <TableContainer component={Paper} sx={{ bgcolor: C.cardBg, border: `1px solid ${C.border}`, borderRadius: 2 }}>
              <Table size="small">
                <TableHead sx={{ bgcolor: isDark ? 'rgba(0,0,0,0.3)' : '#F1F5F9' }}>
                  <TableRow>
                    <TableCell sx={{ color: C.textMuted, fontWeight: 700, fontSize: '0.72rem' }}>Parameter / Variable</TableCell>
                    <TableCell sx={{ color: C.textMuted, fontWeight: 700, fontSize: '0.72rem' }}>Source &amp; Type</TableCell>
                    <TableCell sx={{ color: C.textMuted, fontWeight: 700, fontSize: '0.72rem' }}>Prompt Label</TableCell>
                    <TableCell sx={{ color: C.textMuted, fontWeight: 700, fontSize: '0.72rem' }}>Default / Mapping</TableCell>
                    {!isReadOnly && <TableCell sx={{ color: C.textMuted, fontWeight: 700, fontSize: '0.72rem', textAlign: 'right' }}>Actions</TableCell>}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {parameters.map((param) => (
                    <TableRow key={param.id} sx={{ '&:hover': { bgcolor: isDark ? 'rgba(255,255,255,0.03)' : 'rgba(0,0,0,0.02)' } }}>
                      <TableCell>
                        <Box>
                          <Typography variant="body2" sx={{ fontFamily: 'monospace', fontWeight: 800, color: isDark ? '#38BDF8' : '#0284C7', fontSize: '0.8rem', display: 'flex', alignItems: 'center', gap: 0.5 }}>
                            @{param.name}
                            {param.lockForUser && <Lock size={12} color="#F59E0B" />}
                          </Typography>
                          <Typography variant="caption" sx={{ color: C.textMuted, fontSize: '0.65rem' }}>
                            Parameters!{param.name}.Value
                          </Typography>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
                          <Chip
                            size="small"
                            label={param.type}
                            sx={{ height: 18, fontSize: '0.65rem', fontWeight: 700, bgcolor: isDark ? 'rgba(255,255,255,0.05)' : '#E2E8F0', color: C.text }}
                          />
                          {param.sourceType === 'context' && (
                            <Chip
                              size="small"
                              label="User Context"
                              sx={{ height: 18, fontSize: '0.6rem', fontWeight: 700, bgcolor: 'rgba(245, 158, 11, 0.15)', color: '#F59E0B' }}
                            />
                          )}
                          {param.sourceType === 'query' && (
                            <Chip
                              size="small"
                              label="SQL Query"
                              sx={{ height: 18, fontSize: '0.6rem', fontWeight: 700, bgcolor: 'rgba(59, 130, 246, 0.15)', color: '#3B82F6' }}
                            />
                          )}
                        </Box>
                      </TableCell>
                      <TableCell sx={{ color: C.text, fontSize: '0.78rem' }}>
                        {param.prompt || param.name}
                      </TableCell>
                      <TableCell sx={{ color: C.accentLight, fontFamily: 'monospace', fontSize: '0.78rem', fontWeight: 600 }}>
                        {param.userContextKey ? (
                          <span style={{ color: '#F59E0B' }}>User.{param.userContextKey}</span>
                        ) : param.defaultValue ? (
                          param.defaultValue
                        ) : (
                          <span style={{ color: C.textMuted, fontStyle: 'italic' }}>None</span>
                        )}
                      </TableCell>
                      {!isReadOnly && (
                        <TableCell sx={{ textAlign: 'right' }}>
                          <IconButton size="small" onClick={() => setEditingParam(param)} sx={{ color: C.textMuted, '&:hover': { color: isDark ? '#38BDF8' : '#0284C7' } }}>
                            <Edit2 size={14} />
                          </IconButton>
                          <IconButton size="small" onClick={() => onDelete(param.id)} sx={{ color: C.textMuted, '&:hover': { color: '#EF4444' } }}>
                            <Trash2 size={14} />
                          </IconButton>
                        </TableCell>
                      )}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </DialogContent>

        <DialogActions sx={{ p: 2, borderTop: `1px solid ${C.border}` }}>
          <Button onClick={onClose} variant="contained" sx={{ bgcolor: C.accent, color: '#FFF', fontWeight: 800, textTransform: 'none' }}>
            Done
          </Button>
        </DialogActions>
      </Dialog>

      <ParameterEditor param={editingParam} onSave={handleSave} onCancel={() => setEditingParam(null)} />
    </>
  );
};

export default ParametersDialog;
