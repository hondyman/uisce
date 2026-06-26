/**
 * Field Permission Editor
 * Manage field-level security and masking rules
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Container,
  FormControl,
  Grid,
  InputAdornment,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  Chip,
  Radio,
  RadioGroup,
  FormControlLabel,
  AppBar,
  Toolbar,
  Avatar,
  IconButton,
} from '@mui/material';
import {
  Search as SearchIcon,
  Add as AddIcon,
  Security as SecurityIcon,
  Visibility as VisibilityIcon,
  VisibilityOff as VisibilityOffIcon,
  Edit as EditIcon,
  Lock as LockIcon,
  ExpandMore as ExpandMoreIcon,
  GridView as GridViewIcon,
  BackupTable as BackupTableIcon,
  PersonOutline as PersonOutlineIcon,
  Language as LanguageIcon,
  Translate as TranslateIcon,
  LightMode as LightModeIcon,
  Notifications as NotificationsIcon,
  Settings as SettingsIcon,
  Autorenew as AutorenewIcon,
} from '@mui/icons-material';
import { 
  Autocomplete,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Stack,
  FormHelperText
} from '@mui/material';

// Mock Data
// Mock Data removed in favor of API fetch
// const MOCK_FIELDS = ...

const MOCK_ROLES = [
  { id: 'role-1', name: 'Trader' },
  { id: 'role-2', name: 'Compliance Officer' },
  { id: 'role-3', name: 'Analyst' },
];

const CONTEXT_OPTIONS = [
  { id: 'client_profile', label: 'Client Profile' },
  { id: 'trade_execution', label: 'Trade Execution' },
  { id: 'kyc_document', label: 'KYC Document' },
];

interface FieldPermissionEditorProps {
  tenant: { id: string; display_name: string };
  datasource: { id: string; source_name: string };
}

export const FieldPermissionEditor: React.FC<FieldPermissionEditorProps> = ({ tenant, datasource }) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedContext, setSelectedContext] = useState('client_profile');
  const [selectedRole, setSelectedRole] = useState<{ id: string; name: string } | null>(MOCK_ROLES[0]);
  const [fields, setFields] = useState<any[]>([]); // Store fetched fields

  useEffect(() => {
    // Fetch business terms from API
    const fetchTerms = async () => {
      try {
        const response = await fetch(`/api/semantic-terms?tenant_instance_id=${datasource?.id || ''}`);
        if (response.ok) {
          const data = await response.json();
          const mappedFields = (data.data || []).map((term: any) => ({
            id: term.id,
            name: term.node_name,
            key: term.qualified_path || term.node_name.toLowerCase().replace(/\s+/g, '_'),
            category: term.properties?.category || 'General', // Fallback category
            ...term
          }));
          setFields(mappedFields);
        }
      } catch (error) {
        console.error('Failed to fetch semantic terms:', error);
      }
    };

    fetchTerms();
  }, [datasource?.id]);
  
  // Permissions state: Record<FieldID, Record<RoleID, Permission>>
  const [permissions, setPermissions] = useState<Record<string, Record<string, string>>>({
    '1': { 'role-1': 'read', 'role-2': 'mask', 'role-3': 'none' },
    '2': { 'role-1': 'read', 'role-2': 'mask', 'role-3': 'none' },
    '3': { 'role-1': 'none', 'role-2': 'read', 'role-3': 'none' },
    '4': { 'role-1': 'write', 'role-2': 'read', 'role-3': 'read' },
  });

  // Masking Rule Dialog State
  const [openMaskingDialog, setOpenMaskingDialog] = useState(false);
  const [maskingRuleForm, setMaskingRuleForm] = useState({
    fieldId: null as string | null,
    type: 'partial',
    pattern: ''
  });

  const handleOpenMaskingDialog = () => {
    setMaskingRuleForm({ fieldId: null, type: 'partial', pattern: '' });
    setOpenMaskingDialog(true);
  };

  const handleCloseMaskingDialog = () => {
    setOpenMaskingDialog(false);
  };

  const handleSaveMaskingRule = () => {
    // TODO: Connect to backend
    console.log('Saving masking rule:', maskingRuleForm);
    handleCloseMaskingDialog();
  };

  const handlePermissionChange = (fieldId: string, value: string) => {
    if (!selectedRole) return;
    setPermissions(prev => ({
      ...prev,
      [fieldId]: {
        ...(prev[fieldId] || {}),
        [selectedRole.id]: value
      }
    }));
  };

  const currentPermission = (fieldId: string) => {
    if (!selectedRole) return 'none';
    return permissions[fieldId]?.[selectedRole.id] || 'none';
  };

  return (
    <Box sx={{ minHeight: '100vh', bgcolor: 'background.default', display: 'flex', flexDirection: 'column' }}>
      {/* Header - Replicating the provided design's header */}
      <AppBar position="static" elevation={1} sx={{ bgcolor: '#1d66d5' }}> {/* Primary color from provided config */}
        <Container maxWidth="xl">
          <Toolbar disableGutters sx={{ minHeight: 64, justifyContent: 'space-between' }}>
            <Box display="flex" alignItems="center" gap={4}>
               <Typography variant="h6" fontWeight="bold" color="inherit">
                  SemLayer
               </Typography>
               <Box sx={{ display: { xs: 'none', md: 'flex' }, gap: 3 }}>
                  <Button color="inherit" endIcon={<ExpandMoreIcon fontSize="small" />} sx={{ textTransform: 'none', fontWeight: 500 }}>Setup</Button>
                  <Button color="inherit" startIcon={<GridViewIcon fontSize="small" />} endIcon={<ExpandMoreIcon fontSize="small" />} sx={{ textTransform: 'none', fontWeight: 500 }}>Tenants</Button>
                  <Button color="inherit" startIcon={<SecurityIcon fontSize="small" />} endIcon={<ExpandMoreIcon fontSize="small" />} sx={{ textTransform: 'none', fontWeight: 500 }}>Security</Button>
                  <Button color="inherit" startIcon={<BackupTableIcon fontSize="small" />} endIcon={<ExpandMoreIcon fontSize="small" />} sx={{ textTransform: 'none', fontWeight: 500 }}>System</Button>
                  <Button 
                    variant="outlined" 
                    color="inherit" 
                    startIcon={<PersonOutlineIcon fontSize="small" />} 
                    sx={{ textTransform: 'none', fontWeight: 500, borderColor: 'rgba(255,255,255,0.2)', bgcolor: 'rgba(255,255,255,0.1)', '&:hover': { bgcolor: 'rgba(255,255,255,0.2)' } }}
                  >
                    Operator
                  </Button>
                   <Button color="inherit" startIcon={<LanguageIcon fontSize="small" />} endIcon={<ExpandMoreIcon fontSize="small" />} sx={{ textTransform: 'none', fontWeight: 500 }}>All Tenants</Button>
               </Box>
            </Box>
            <Box display="flex" alignItems="center" gap={1}>
                <IconButton color="inherit"><TranslateIcon fontSize="small" /></IconButton>
                <IconButton color="inherit"><LightModeIcon fontSize="small" /></IconButton>
                <IconButton color="inherit"><NotificationsIcon fontSize="small" /></IconButton>
                <IconButton color="inherit"><SettingsIcon fontSize="small" /></IconButton>
            </Box>
          </Toolbar>
        </Container>
      </AppBar>

      <Box component="main" sx={{ flexGrow: 1, p: { xs: 3, lg: 4 } }}>
        <Container maxWidth={false}>
          {/* Page Title & Action */}
          <Box display="flex" flexDirection={{ xs: 'column', sm: 'row' }} justifyContent="space-between" alignItems={{ xs: 'flex-start', sm: 'center' }} mb={3} gap={2}>
            <Box>
              <Box display="flex" alignItems="center" gap={1.5} mb={0.5}>
                <SecurityIcon sx={{ fontSize: 32, color: 'text.secondary' }} />
                <Typography variant="h4" fontWeight={700} color="text.primary">
                  Field Permission Editor
                </Typography>
              </Box>
              <Typography variant="body1" color="text.secondary">
                Configure field-level security and masking rules for My Client
              </Typography>
            </Box>
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleOpenMaskingDialog}
              sx={{ 
                bgcolor: 'background.paper', 
                color: 'text.primary', 
                textTransform: 'none', 
                fontWeight: 600,
                border: 1,
                borderColor: 'divider',
                boxShadow: 0,
                '&:hover': { bgcolor: 'action.hover', boxShadow: 0 }
              }}
            >
              Add Masking Rule
            </Button>
          </Box>
          
          {/* Masking Rule Dialog */}
          <Dialog open={openMaskingDialog} onClose={handleCloseMaskingDialog} maxWidth="sm" fullWidth>
            <DialogTitle sx={{ fontWeight: 700 }}>Add Masking Rule</DialogTitle>
            <DialogContent dividers>
              <Stack spacing={3} pt={1}>
                <Autocomplete
                  options={fields}
                  getOptionLabel={(option) => option.name}
                  value={fields.find(f => f.id === maskingRuleForm.fieldId) || null}
                  onChange={(_, newValue) => setMaskingRuleForm(prev => ({ ...prev, fieldId: newValue?.id || null }))}
                  renderInput={(params) => (
                    <TextField 
                      {...params} 
                      label="Select Field" 
                      placeholder="Search business terms..." 
                      helperText="Select a field from the business glossary"
                    />
                  )}
                />

                <FormControl fullWidth>
                  <InputLabel>Masking Type</InputLabel>
                  <Select
                    value={maskingRuleForm.type}
                    label="Masking Type"
                    onChange={(e) => setMaskingRuleForm(prev => ({ ...prev, type: e.target.value }))}
                  >
                    <MenuItem value="full">Full Masking (******)</MenuItem>
                    <MenuItem value="partial">Partial Masking (Last 4 visible)</MenuItem>
                    <MenuItem value="hash">Hashing (SHA-256)</MenuItem>
                    <MenuItem value="custom">Custom Pattern</MenuItem>
                  </Select>
                </FormControl>

                {maskingRuleForm.type === 'custom' && (
                  <TextField
                    label="Custom Pattern"
                    value={maskingRuleForm.pattern}
                    onChange={(e) => setMaskingRuleForm(prev => ({ ...prev, pattern: e.target.value }))}
                    placeholder="e.g., XXX-XX-####"
                    fullWidth
                    helperText="Use X for masked characters and # for visible"
                  />
                )}
              </Stack>
            </DialogContent>
            <DialogActions sx={{ p: 2.5 }}>
              <Button onClick={handleCloseMaskingDialog} color="inherit">Cancel</Button>
              <Button 
                onClick={handleSaveMaskingRule} 
                variant="contained" 
                disabled={!maskingRuleForm.fieldId}
              >
                Save Rule
              </Button>
            </DialogActions>
          </Dialog>

          {/* Filters */}
          <Box mb={3}>
             <Grid container spacing={2} alignItems="center">
                <Grid item xs={12} sm={3}>
                   <Autocomplete
                      options={MOCK_ROLES}
                      getOptionLabel={(option) => option.name}
                      value={selectedRole}
                      onChange={(_, newValue) => setSelectedRole(newValue)}
                      renderInput={(params) => <TextField {...params} label="Select Role" size="small" sx={{ bgcolor: 'background.paper' }} />}
                    />
                </Grid>
                <Grid item xs={12} sm={3}>
                   <FormControl fullWidth size="small">
                      <InputLabel>Context</InputLabel>
                      <Select
                        label="Context"
                        value={selectedContext}
                        onChange={(e) => setSelectedContext(e.target.value)}
                        sx={{ bgcolor: 'background.paper' }}
                      >
                         {CONTEXT_OPTIONS.map(opt => (
                           <MenuItem key={opt.id} value={opt.id}>{opt.label}</MenuItem>
                         ))}
                      </Select>
                   </FormControl>
                </Grid>
                <Grid item xs={12} sm>
                    <TextField
                      fullWidth
                      size="small"
                      placeholder="Search fields..."
                      value={searchTerm}
                      onChange={(e) => setSearchTerm(e.target.value)}
                      InputProps={{
                        startAdornment: (
                          <InputAdornment position="start">
                            <SearchIcon color="action" />
                          </InputAdornment>
                        ),
                      }}
                       sx={{ bgcolor: 'background.paper' }}
                    />
                </Grid>
             </Grid>
          </Box>

          {/* Table */}
          <Paper variant="outlined" sx={{ borderRadius: 2, overflow: 'hidden' }}>
            <TableContainer>
              <Table>
                <TableHead sx={{ bgcolor: 'action.hover' }}>
                  <TableRow>
                    <TableCell sx={{ width: '30%', fontWeight: 600, color: 'text.secondary', textTransform: 'uppercase', fontSize: '0.75rem' }}>Field Name</TableCell>
                    <TableCell sx={{ width: '20%', fontWeight: 600, color: 'text.secondary', textTransform: 'uppercase', fontSize: '0.75rem' }}>Category</TableCell>
                    <TableCell sx={{ width: '50%', fontWeight: 600, color: 'text.secondary', textTransform: 'uppercase', fontSize: '0.75rem' }}>
                        Permission for <Box component="span" color="primary.main">{selectedRole?.name || '...'}</Box>
                    </TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {fields.map((field) => (
                    <TableRow key={field.id} hover>
                      <TableCell>
                        <Typography variant="subtitle2" fontWeight={600} color="text.primary">{field.name}</Typography>
                        <Typography variant="caption" color="text.secondary">{field.key}</Typography>
                      </TableCell>
                      <TableCell>
                         <Chip 
                            label={field.category} 
                            size="small" 
                            color={field.category === 'PII' ? 'error' : 'success'} 
                            variant="outlined"
                            sx={{ fontWeight: 600, borderRadius: 4, height: 24, bgcolor: field.category === 'PII' ? 'error.lighter' : 'success.lighter', border: 1, borderColor: field.category === 'PII' ? 'error.light' : 'success.light' }} 
                         />
                      </TableCell>
                      <TableCell>
                          <Paper variant="outlined" sx={{ display: 'inline-flex', borderRadius: 2, overflow: 'hidden', opacity: selectedRole ? 1 : 0.5, pointerEvents: selectedRole ? 'auto' : 'none' }}>
                              {['none', 'read', 'write', 'mask'].map((option, index) => {
                                  const isSelected = currentPermission(field.id) === option;
                                  return (
                                      <React.Fragment key={option}>
                                          {index > 0 && <Box sx={{ width: 1, bgcolor: 'divider' }} />}
                                          <Box 
                                            onClick={() => handlePermissionChange(field.id, option)}
                                            sx={{ 
                                                display: 'flex', 
                                                alignItems: 'center', 
                                                px: 2, 
                                                py: 1, 
                                                cursor: 'pointer',
                                                bgcolor: isSelected ? 'action.selected' : 'transparent',
                                                '&:hover': { bgcolor: isSelected ? 'action.selected' : 'action.hover' },
                                                color: isSelected ? 'primary.main' : 'text.secondary',
                                            }}
                                          >
                                              {option === 'none' && <VisibilityOffIcon fontSize="small" sx={{ mr: 1 }} />}
                                              {option === 'read' && <VisibilityIcon fontSize="small" sx={{ mr: 1 }} />}
                                              {option === 'write' && <EditIcon fontSize="small" sx={{ mr: 1 }} />}
                                              {option === 'mask' && <LockIcon fontSize="small" sx={{ mr: 1 }} />}
                                              <Typography variant="caption" fontWeight={600} sx={{ textTransform: 'uppercase' }}>{option}</Typography>
                                              <Radio 
                                                checked={isSelected} 
                                                size="small" 
                                                sx={{ display: 'none' }} 
                                              />
                                          </Box>
                                      </React.Fragment>
                                  );
                              })}
                          </Paper>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>

        </Container>
      </Box>
    </Box>
  );
};
