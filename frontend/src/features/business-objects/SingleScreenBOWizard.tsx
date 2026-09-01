import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Grid,
  Stack,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Button,
  IconButton,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Checkbox,
  Tabs,
  Tab,
  Alert,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Star as StarIcon,
  StarBorder as StarBorderIcon,
  Add as AddIcon,
  CheckCircleOutline as CheckCircleOutlineIcon,
  WarningAmber as WarningAmberIcon,
  RocketLaunch as RocketLaunchIcon,
  Save as SaveIcon,
  CategoryOutlined as CategoryOutlinedIcon,
  AccountTreeOutlined as AccountTreeOutlinedIcon,
  LayersOutlined as LayersOutlinedIcon
} from '@mui/icons-material';

interface ColumnMapping {
  columnNodeId: string;
  columnName: string;
  tableName: string;
  isPrimarySource: boolean;
}

interface EligibleTerm {
  termNodeId: string;
  termKey: string;
  termName: string;
  sourceType: 'DIRECT' | 'RELATED' | 'CALCULATED' | 'MANUAL';
  fieldRole: 'KEY' | 'DIMENSION' | 'MEASURE' | 'ATTRIBUTE';
  mappings: ColumnMapping[];
}

interface BindingState {
  backendId: string;
  backendName: string;
  drivingNodeId: string;
  drivingTableName: string;
  isDefault: boolean;
  eligibleTerms: EligibleTerm[];
  selectedTermKeys: string[];
  fieldMappings: Record<string, string>;
}

export const SingleScreenBOWizard: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [boName, setBoName] = useState('Customer');
  const [boKey, setBoKey] = useState('customer');
  const [boType, setBoType] = useState('ENTITY');
  const [businessKey, setBusinessKey] = useState('customer_bk');
  const [semanticId, setSemanticId] = useState('customer_sid');

  const [bindings, setBindings] = useState<BindingState[]>([
    {
      backendId: 'a0000001-0000-4000-a000-000000000001',
      backendName: 'Northwind Postgres (Hot)',
      drivingNodeId: 'b4c9e2c7-1c4c-5c2b-ac2b-2b3c4d5e6f7a',
      drivingTableName: 'Customers',
      isDefault: true,
      selectedTermKeys: ['customer_bk', 'customer_sid', 'company_name', 'country', 'city'],
      fieldMappings: {
        customer_bk: 'col-cust-id',
        customer_sid: 'col-cust-uuid',
        company_name: 'col-comp-name',
        country: 'col-country',
        city: 'col-city'
      },
      eligibleTerms: [
        {
          termNodeId: 't-1',
          termKey: 'customer_bk',
          termName: 'Customer Business Key',
          sourceType: 'DIRECT',
          fieldRole: 'KEY',
          mappings: [{ columnNodeId: 'col-cust-id', columnName: 'CustomerID', tableName: 'Customers', isPrimarySource: true }]
        },
        {
          termNodeId: 't-2',
          termKey: 'customer_sid',
          termName: 'Customer Semantic ID',
          sourceType: 'DIRECT',
          fieldRole: 'KEY',
          mappings: [{ columnNodeId: 'col-cust-uuid', columnName: 'CustomerUUID', tableName: 'Customers', isPrimarySource: true }]
        },
        {
          termNodeId: 't-3',
          termKey: 'company_name',
          termName: 'Company Name',
          sourceType: 'DIRECT',
          fieldRole: 'DIMENSION',
          mappings: [{ columnNodeId: 'col-comp-name', columnName: 'CompanyName', tableName: 'Customers', isPrimarySource: true }]
        },
        {
          termNodeId: 't-4',
          termKey: 'country',
          termName: 'Billing Country',
          sourceType: 'DIRECT',
          fieldRole: 'DIMENSION',
          mappings: [{ columnNodeId: 'col-country', columnName: 'Country', tableName: 'Customers', isPrimarySource: true }]
        },
        {
          termNodeId: 't-5',
          termKey: 'city',
          termName: 'City',
          sourceType: 'DIRECT',
          fieldRole: 'DIMENSION',
          mappings: [{ columnNodeId: 'col-city', columnName: 'City', tableName: 'Customers', isPrimarySource: true }]
        }
      ]
    },
    {
      backendId: 'a0000002-0000-4000-a000-000000000002',
      backendName: 'CRM Snowflake (OLAP)',
      drivingNodeId: 'c5d1e2f3-4a5b-6c7d-8e9f-0a1b2c3d4e5f',
      drivingTableName: 'CRM_CONTACTS',
      isDefault: false,
      selectedTermKeys: ['customer_bk', 'company_name', 'country'],
      fieldMappings: {
        customer_bk: 'col-snow-id',
        company_name: 'col-snow-name',
        country: 'col-snow-country'
      },
      eligibleTerms: []
    }
  ]);

  const [activeTermTab, setActiveTermTab] = useState<number>(0);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [publishAlert, setPublishAlert] = useState<{ severity: 'success' | 'warning' | 'error'; message: string } | null>(null);

  const handleSetDefaultBinding = (index: number) => {
    setBindings(bindings.map((b, idx) => ({ ...b, isDefault: idx === index })));
  };

  const handleToggleTerm = (bIdx: number, termKey: string, defaultColumnId?: string) => {
    setBindings(prev => {
      const copy = [...prev];
      const target = { ...copy[bIdx] };
      if (target.selectedTermKeys.includes(termKey)) {
        target.selectedTermKeys = target.selectedTermKeys.filter(k => k !== termKey);
        delete target.fieldMappings[termKey];
      } else {
        target.selectedTermKeys = [...target.selectedTermKeys, termKey];
        if (defaultColumnId) {
          target.fieldMappings[termKey] = defaultColumnId;
        }
      }
      copy[bIdx] = target;
      return copy;
    });
  };

  const handleSaveOrPublish = async (publish: boolean) => {
    setIsSubmitting(true);
    setPublishAlert(null);

    setTimeout(() => {
      setIsSubmitting(false);
      setPublishAlert({
        severity: 'success',
        message: publish ? 'Business Object published actively across all backends!' : 'Draft configuration saved successfully.'
      });
    }, 600);
  };

  return (
    <Box sx={{ p: 4, bgcolor: '#0f172a', minHeight: '100vh', color: '#f8fafc' }}>
      <Stack spacing={4}>
        <Box display="flex" justifyContent="space-between" alignItems="center">
          <Box>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#38bdf8' }}>
              Business Object Management Studio
            </Typography>
            <Typography variant="body2" sx={{ color: '#94a3b8' }}>
              Single-screen semantic contract authoring, auto-discovery & multi-backend execution mesh
            </Typography>
          </Box>
          <Stack direction="row" spacing={2}>
            <Button
              variant="outlined"
              color="inherit"
              startIcon={<SaveIcon />}
              onClick={() => handleSaveOrPublish(false)}
              disabled={isSubmitting}
              sx={{ borderColor: '#334155' }}
            >
              Save Draft
            </Button>
            <Button
              variant="contained"
              color="primary"
              startIcon={<RocketLaunchIcon />}
              onClick={() => handleSaveOrPublish(true)}
              disabled={isSubmitting}
              sx={{ bgcolor: '#0284c7', '&:hover': { bgcolor: '#0369a1' } }}
            >
              Validate & Publish
            </Button>
          </Stack>
        </Box>

        {publishAlert && (
          <Alert severity={publishAlert.severity} variant="filled">
            {publishAlert.message}
          </Alert>
        )}

        {/* 1. BO Definition Shell */}
        <Paper sx={{ p: 3, bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155', borderRadius: 2 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2, color: '#e2e8f0' }}>
            1. Semantic Object Definition
          </Typography>
          <Grid container spacing={3}>
            <Grid   size={{ xs: 12, sm: 4 }}>
              <TextField
                fullWidth
                size="small"
                label="Business Object Name"
                value={boName}
                onChange={e => setBoName(e.target.value)}
                sx={{ '& .MuiInputBase-root': { bgcolor: '#0f172a', color: '#f8fafc' } }}
                InputLabelProps={{ sx: { color: '#94a3b8' } }}
              />
            </Grid>
            <Grid   size={{ xs: 12, sm: 4 }}>
              <TextField
                fullWidth
                size="small"
                label="Machine Key (Technical)"
                value={boKey}
                onChange={e => setBoKey(e.target.value)}
                sx={{ '& .MuiInputBase-root': { bgcolor: '#0f172a', color: '#f8fafc' } }}
                InputLabelProps={{ sx: { color: '#94a3b8' } }}
              />
            </Grid>
            <Grid   size={{ xs: 12, sm: 4 }}>
              <FormControl fullWidth size="small">
                <InputLabel sx={{ color: '#94a3b8' }}>Object Type</InputLabel>
                <Select
                  value={boType}
                  label="Object Type"
                  onChange={e => setBoType(e.target.value)}
                  sx={{ bgcolor: '#0f172a', color: '#f8fafc' }}
                >
                  <MenuItem value="ENTITY">ENTITY</MenuItem>
                  <MenuItem value="FACT">FACT</MenuItem>
                  <MenuItem value="DIMENSION">DIMENSION</MenuItem>
                </Select>
              </FormControl>
            </Grid>

            <Grid   size={{ xs: 12, sm: 4 }}>
              <TextField
                fullWidth
                size="small"
                label="Business Key (*_bk)"
                value={businessKey}
                onChange={e => setBusinessKey(e.target.value)}
                sx={{ '& .MuiInputBase-root': { bgcolor: '#0f172a', color: '#f8fafc' } }}
                InputLabelProps={{ sx: { color: '#94a3b8' } }}
              />
            </Grid>
            <Grid   size={{ xs: 12, sm: 4 }}>
              <TextField
                fullWidth
                size="small"
                label="Semantic ID (*_sid)"
                value={semanticId}
                onChange={e => setSemanticId(e.target.value)}
                sx={{ '& .MuiInputBase-root': { bgcolor: '#0f172a', color: '#f8fafc' } }}
                InputLabelProps={{ sx: { color: '#94a3b8' } }}
              />
            </Grid>
            <Grid   size={{ xs: 12, sm: 4 }}>
              <TextField
                fullWidth
                size="small"
                label="Grain Definition"
                value={businessKey}
                disabled
                sx={{ '& .MuiInputBase-root': { bgcolor: '#0f172a', color: '#64748b' } }}
                InputLabelProps={{ sx: { color: '#94a3b8' } }}
              />
            </Grid>
          </Grid>
        </Paper>

        {/* 2. Bindings & Scoped Term Selectors */}
        <Box>
          <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, color: '#e2e8f0' }}>
              2. Physical Bindings & Auto-Discovered Scopes
            </Typography>
            <Button startIcon={<AddIcon />} variant="outlined" size="small" sx={{ borderColor: '#38bdf8', color: '#38bdf8' }}>
              Add Backend Binding
            </Button>
          </Box>

          {bindings.map((b, bIdx) => (
            <Accordion
              key={b.backendId}
              defaultExpanded={b.isDefault}
              sx={{ bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155', mb: 2, borderRadius: 1 }}
            >
              <AccordionSummary expandIcon={<ExpandMoreIcon sx={{ color: '#94a3b8' }} />}>
                <Stack direction="row" spacing={2} alignItems="center" sx={{ width: '100%' }}>
                  <IconButton
                    size="small"
                    onClick={e => {
                      e.stopPropagation();
                      handleSetDefaultBinding(bIdx);
                    }}
                  >
                    {b.isDefault ? <StarIcon sx={{ color: '#f59e0b' }} /> : <StarBorderIcon sx={{ color: '#64748b' }} />}
                  </IconButton>
                  <Typography variant="subtitle2" sx={{ fontWeight: 700, color: b.isDefault ? '#38bdf8' : '#f8fafc' }}>
                    {b.backendName}
                  </Typography>
                  <Chip
                    label={`Driving Table: ${b.drivingTableName}`}
                    size="small"
                    sx={{ bgcolor: '#0f172a', color: '#94a3b8', border: '1px solid #334155' }}
                  />
                  <Box sx={{ flexGrow: 1 }} />
                  <Chip
                    label={`${b.selectedTermKeys.length} Terms Bound`}
                    size="small"
                    color={b.selectedTermKeys.length > 0 ? 'success' : 'default'}
                    variant="outlined"
                  />
                </Stack>
              </AccordionSummary>
              <AccordionDetails sx={{ borderTop: '1px solid #334155' }}>
                <Box mb={3}>
                  <Tabs
                    value={activeTermTab}
                    onChange={(_, val) => setActiveTermTab(val)}
                    textColor="inherit"
                    indicatorColor="primary"
                    sx={{ borderBottom: '1px solid #334155' }}
                  >
                    <Tab label="Direct Terms (Driving Table)" icon={<CategoryOutlinedIcon />} iconPosition="start" />
                    <Tab label="Related Terms (FK Graph)" icon={<AccountTreeOutlinedIcon />} iconPosition="start" />
                    <Tab label="Calculated AST Measures" icon={<LayersOutlinedIcon />} iconPosition="start" />
                  </Tabs>
                </Box>

                {/* Terms Selection Grid */}
                <TableContainer component={Paper} sx={{ bgcolor: '#0f172a', border: '1px solid #334155' }}>
                  <Table size="small">
                    <TableHead>
                      <TableRow sx={{ '& th': { color: '#94a3b8', fontWeight: 600, borderColor: '#334155' } }}>
                        <TableCell padding="checkbox">Select</TableCell>
                        <TableCell>Semantic Term Key</TableCell>
                        <TableCell>Term Name</TableCell>
                        <TableCell>Role</TableCell>
                        <TableCell>Physical Column Mapping</TableCell>
                        <TableCell>Status</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {b.eligibleTerms.map(term => {
                        const isSelected = b.selectedTermKeys.includes(term.termKey);
                        const mappedColId = b.fieldMappings[term.termKey];

                        return (
                          <TableRow
                            key={term.termKey}
                            hover
                            sx={{ '& td': { color: '#f8fafc', borderColor: '#334155' } }}
                          >
                            <TableCell padding="checkbox">
                              <Checkbox
                                checked={isSelected}
                                onChange={() => handleToggleTerm(bIdx, term.termKey, term.mappings[0]?.columnNodeId)}
                                sx={{ color: '#64748b', '&.Mui-checked': { color: '#38bdf8' } }}
                              />
                            </TableCell>
                            <TableCell sx={{ fontFamily: 'monospace', color: '#38bdf8' }}>{term.termKey}</TableCell>
                            <TableCell>{term.termName}</TableCell>
                            <TableCell>
                              <Chip label={term.fieldRole} size="small" sx={{ bgcolor: '#1e293b', color: '#e2e8f0' }} />
                            </TableCell>
                            <TableCell>
                              {term.mappings.length === 1 ? (
                                <Typography variant="caption" sx={{ fontFamily: 'monospace', color: '#34d399' }}>
                                  {term.mappings[0].tableName}.{term.mappings[0].columnName}
                                </Typography>
                              ) : (
                                <Select
                                  size="small"
                                  value={mappedColId || ''}
                                  onChange={e => {
                                    const colId = e.target.value;
                                    setBindings(prev => {
                                      const copy = [...prev];
                                      copy[bIdx].fieldMappings[term.termKey] = colId;
                                      return copy;
                                    });
                                  }}
                                  sx={{ bgcolor: '#1e293b', color: '#f8fafc', height: 30, fontSize: 12 }}
                                >
                                  {term.mappings.map(m => (
                                    <MenuItem key={m.columnNodeId} value={m.columnNodeId} sx={{ fontSize: 12 }}>
                                      {m.tableName}.{m.columnName} {m.isPrimarySource ? '(Primary)' : ''}
                                    </MenuItem>
                                  ))}
                                </Select>
                              )}
                            </TableCell>
                            <TableCell>
                              {isSelected && mappedColId ? (
                                <Chip icon={<CheckCircleOutlineIcon />} label="RESOLVED" size="small" color="success" />
                              ) : isSelected ? (
                                <Chip icon={<WarningAmberIcon />} label="UNRESOLVED" size="small" color="warning" />
                              ) : (
                                <Typography variant="caption" sx={{ color: '#64748b' }}>
                                  Not Added
                                </Typography>
                              )}
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </TableContainer>
              </AccordionDetails>
            </Accordion>
          ))}
        </Box>

        {/* 3. Cross-Backend Validation & Coverage HUD */}
        <Paper sx={{ p: 3, bgcolor: '#1e293b', color: '#f8fafc', border: '1px solid #334155', borderRadius: 2 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 600, mb: 2, color: '#e2e8f0' }}>
            3. Cross-Backend Coverage Matrix
          </Typography>
          <Grid container spacing={2}>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #334155', textAlign: 'center' }}>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                  Identity Requirements
                </Typography>
                <Typography variant="h6" sx={{ color: '#34d399', fontWeight: 700 }}>
                  Passed (BK + SID)
                </Typography>
              </Paper>
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #334155', textAlign: 'center' }}>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                  Default Backend Resolution
                </Typography>
                <Typography variant="h6" sx={{ color: '#38bdf8', fontWeight: 700 }}>
                  5/5 Fields (100%)
                </Typography>
              </Paper>
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #334155', textAlign: 'center' }}>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                  Secondary Backend Coverage
                </Typography>
                <Typography variant="h6" sx={{ color: '#fbbf24', fontWeight: 700 }}>
                  3/5 Fields (Partial)
                </Typography>
              </Paper>
            </Grid>
            <Grid   size={{ xs: 12, sm: 3 }}>
              <Paper sx={{ p: 2, bgcolor: '#0f172a', border: '1px solid #334155', textAlign: 'center' }}>
                <Typography variant="caption" sx={{ color: '#94a3b8' }}>
                  Publish Eligibility Status
                </Typography>
                <Typography variant="h6" sx={{ color: '#34d399', fontWeight: 700 }}>
                  READY_TO_PUBLISH
                </Typography>
              </Paper>
            </Grid>
          </Grid>
        </Paper>
      </Stack>
    </Box>
  );
};

export default SingleScreenBOWizard;
