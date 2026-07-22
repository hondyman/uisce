import * as React from 'react';
import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Paper,
  Button,
  Typography,
  TextField,
  MenuItem,
  FormControl,
  InputLabel,
  Select,
  Autocomplete,
  Stack,
  Card,
  CardContent,
  IconButton,
  Divider,
  Grid,
  CircularProgress,
  Alert,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  List,
  ListItem,
  ListItemText
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  CheckCircle as CheckCircleIcon,
  Error as ErrorIcon,
  Save as SaveIcon,
  ExpandMore as ExpandMoreIcon
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';

interface AutoDiscoveryItem {
  termNodeId: string;
  termKey: string;
  termName: string;
  termType: string;
  identityRole: string;
  dataType: string;
  aggregationType: string;
  columnNodeId: string;
  columnKey: string;
  tableKey: string;
  sourceType: string;
}

interface BindingField {
  termNodeId: string;
  termName: string;
  fieldName: string;
  fieldRole: string; // KEY, DIMENSION, MEASURE, ATTRIBUTE
  bindingRequirement: string; // REQUIRED, OPTIONAL, BACKEND_SPECIFIC, CALCULATED
  sourceNodeId: string;
  sourceType: string; // COLUMN, EXPRESSION
  transformationType: string; // NONE, CAST, SQL_EXPR
  transformationSql: string;
  availableColumns: { id: string; name: string }[]; // For inline resolution
}

interface BindingRelationship {
  toBoId: string;
  relKey: string;
  cardinality: string;
  joinType: string;
  joinConditionSql: string;
}

interface Binding {
  id: string;
  backendId: string; // postgres, iceberg, starrocks
  drivingNodeId: string;
  isDefault: boolean;
  fields: BindingField[];
  relationships: BindingRelationship[];
  loading?: boolean;
}

interface BO {
  id?: string;
  key: string;
  name: string;
  displayName: string;
  technicalName: string;
  description: string;
  boTypeId: string;
  classificationNodeId: string;
  businessKeyNodeId: string;
  semanticIdNodeId: string;
  grainNodeId: string;
}

export const BusinessObjectWizardPage: React.FC<{ tenantId?: string }> = ({ tenantId }) => {
  const navigate = useNavigate();
  const { tenant } = useTenant();
  const activeTenantId = tenantId || tenant?.id || '00000000-0000-0000-0000-000000000000';

  // State Definitions
  const [bo, setBo] = useState<BO>({
    key: '',
    name: '',
    displayName: '',
    technicalName: '',
    description: '',
    boTypeId: '',
    classificationNodeId: '',
    businessKeyNodeId: '',
    semanticIdNodeId: '',
    grainNodeId: ''
  });

  const [bindings, setBindings] = useState<Binding[]>([
    {
      id: '1',
      backendId: 'postgres',
      drivingNodeId: '',
      isDefault: true,
      fields: [],
      relationships: []
    }
  ]);

  const [drivingTables, setDrivingTables] = useState<{ id: string; label: string }[]>([]);
  const [businessObjectsList, setBusinessObjectsList] = useState<{ id: string; name: string }[]>([]);
  const [submitError, setSubmitError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  // Fetch Driving Tables & Target BOs
  useEffect(() => {
    const fetchData = async () => {
      try {
        const tblRes = await fetch(`/api/v1/bo/driving-tables?tenant_id=${activeTenantId}`);
        if (tblRes.ok) {
          const data = await tblRes.json();
          setDrivingTables(data || []);
        } else {
          // Fallbacks for testing
          setDrivingTables([
            { id: '11111111-1111-1111-1111-111111111111', label: 'public.customers' },
            { id: '22222222-2222-2222-2222-222222222222', label: 'public.orders' },
            { id: '33333333-3333-3333-3333-333333333333', label: 'public.products' }
          ]);
        }

        const boRes = await fetch('/api/business-objects');
        if (boRes.ok) {
          const boData = await boRes.json();
          const items = Object.entries(boData).map(([id, item]: any) => ({
            id: id,
            name: item.name || item.displayName
          }));
          setBusinessObjectsList(items);
        }
      } catch {
        // Fallbacks
        setDrivingTables([
          { id: '11111111-1111-1111-1111-111111111111', label: 'public.customers' },
          { id: '22222222-2222-2222-2222-222222222222', label: 'public.orders' },
          { id: '33333333-3333-3333-3333-333333333333', label: 'public.products' }
        ]);
      }
    };
    fetchData();
  }, [activeTenantId]);

  // Handle BO Metadata Updates
  const handleBoChange = (field: keyof BO, value: string) => {
    setBo(prev => {
      const next = { ...prev, [field]: value };
      if (field === 'name') {
        const keyVal = value.toLowerCase().replace(/[^a-z0-9_]/g, '_').replace(/_+/g, '_');
        next.key = keyVal;
        next.technicalName = keyVal;
      }
      return next;
    });
  };

  // Add a new expandable Backend Binding panel
  const addBinding = () => {
    const nextId = (bindings.length + 1).toString();
    setBindings(prev => [
      ...prev,
      {
        id: nextId,
        backendId: 'starrocks',
        drivingNodeId: '',
        isDefault: false,
        fields: [],
        relationships: []
      }
    ]);
  };

  const removeBinding = (id: string) => {
    setBindings(prev => prev.filter(b => b.id !== id));
  };

  // Perform Auto-discovery when driving table is selected
  const handleDrivingTableChange = async (bindingId: string, drivingNodeId: string) => {
    setBindings(prev => prev.map(b => b.id === bindingId ? { ...b, drivingNodeId, loading: true } : b));
    if (!drivingNodeId) return;

    try {
      const response = await fetch(`/api/business-objects/auto-discovery?driving_node_id=${drivingNodeId}`);
      if (response.ok) {
        const items: AutoDiscoveryItem[] = await response.json();

        // Group auto-discovery items by term node id to find ambiguity (multiple columns for one term)
        const termGroups: Record<string, AutoDiscoveryItem[]> = {};
        items.forEach(item => {
          if (!termGroups[item.termNodeId]) {
            termGroups[item.termNodeId] = [];
          }
          termGroups[item.termNodeId].push(item);
        });

        const fields: BindingField[] = Object.entries(termGroups).map(([termNodeId, list]) => {
          const primary = list[0];
          return {
            termNodeId,
            termName: primary.termName,
            fieldName: primary.termKey,
            fieldRole: primary.identityRole || 'DIMENSION',
            bindingRequirement: 'REQUIRED',
            sourceNodeId: primary.columnNodeId,
            sourceType: 'COLUMN',
            transformationType: 'NONE',
            transformationSql: '',
            availableColumns: list.map(item => ({
              id: item.columnNodeId,
              name: item.columnKey
            }))
          };
        });

        setBindings(prev => prev.map(b => b.id === bindingId ? { ...b, fields, loading: false } : b));
      } else {
        setBindings(prev => prev.map(b => b.id === bindingId ? { ...b, loading: false } : b));
      }
    } catch {
      setBindings(prev => prev.map(b => b.id === bindingId ? { ...b, loading: false } : b));
    }
  };

  const updateFieldProperty = (bindingId: string, termNodeId: string, property: keyof BindingField, value: any) => {
    setBindings(prev => prev.map(b => {
      if (b.id !== bindingId) return b;
      const updatedFields = b.fields.map(f => f.termNodeId === termNodeId ? { ...f, [property]: value } : f);
      return { ...b, fields: updatedFields };
    }));
  };

  // Relationship actions
  const addRelationship = (bindingId: string) => {
    setBindings(prev => prev.map(b => {
      if (b.id !== bindingId) return b;
      return {
        ...b,
        relationships: [
          ...b.relationships,
          { toBoId: '', relKey: '', cardinality: '1:M', joinType: 'LEFT', joinConditionSql: '' }
        ]
      };
    }));
  };

  const updateRelationship = (bindingId: string, relIdx: number, property: keyof BindingRelationship, value: any) => {
    setBindings(prev => prev.map(b => {
      if (b.id !== bindingId) return b;
      const updated = [...b.relationships];
      updated[relIdx] = { ...updated[relIdx], [property]: value };
      return { ...b, relationships: updated };
    }));
  };

  const removeRelationship = (bindingId: string, relIdx: number) => {
    setBindings(prev => prev.map(b => {
      if (b.id !== bindingId) return b;
      return { ...b, relationships: b.relationships.filter((_, idx) => idx !== relIdx) };
    }));
  };

  // Validation Check Rules (Section 6 Enforcement)
  const validationSummary = (() => {
    const allTermIds = Array.from(new Set(bindings.flatMap(b => b.fields.map(f => f.termNodeId))));
    const errors: string[] = [];
    const successes: string[] = [];

    allTermIds.forEach(termId => {
      // Find term requirement across any configuration
      const sampleField = bindings.flatMap(b => b.fields).find(f => f.termNodeId === termId);
      if (!sampleField) return;

      const termName = sampleField.termName;
      const req = sampleField.bindingRequirement;

      const boundCount = bindings.filter(b => b.fields.find(f => f.termNodeId === termId && f.sourceNodeId)).length;

      if (req === 'REQUIRED' && boundCount < bindings.length) {
        errors.push(`Field "${termName}" is REQUIRED but not mapped in all active backends.`);
      } else if (req === 'BACKEND_SPECIFIC' && boundCount === 0) {
        errors.push(`Field "${termName}" is BACKEND_SPECIFIC but is not mapped in any backend.`);
      } else {
        successes.push(`Field "${termName}" is fully resolved.`);
      }
    });

    return { errors, successes };
  })();

  const handleSave = async () => {
    if (!bo.name || !bo.key) {
      setSubmitError('Name and Technical Key are required.');
      return;
    }

    setIsSaving(true);
    setSubmitError(null);

    const payload = {
      tenantId: activeTenantId,
      modelId: bo.boTypeId || '00000000-0000-0000-0000-000000000000',
      businessObject: {
        boKey: bo.key,
        boName: bo.name,
        boTypeId: bo.boTypeId,
        classificationNodeId: bo.classificationNodeId,
        businessKeyNodeId: bo.businessKeyNodeId,
        semanticIdNodeId: bo.semanticIdNodeId,
        grainNodeId: bo.grainNodeId
      },
      bindings: bindings.map(b => ({
        backendId: b.backendId,
        drivingNodeId: b.drivingNodeId,
        isDefault: b.isDefault,
        temporalOverride: 'NONE',
        fields: b.fields.map(f => ({
          termNodeId: f.termNodeId,
          fieldName: f.fieldName,
          fieldRole: f.fieldRole,
          bindingRequirement: f.bindingRequirement,
          sourceNodeId: f.sourceNodeId,
          sourceType: f.sourceType,
          transformationType: f.transformationType,
          transformationSql: f.transformationSql
        })),
        relationships: b.relationships.map(r => ({
          toBoId: r.toBoId,
          relKey: r.relKey,
          cardinality: r.cardinality,
          joinType: r.joinType,
          joinConditionSql: r.joinConditionSql
        }))
      })),
      publish: true
    };

    try {
      const response = await fetch('/api/business-objects/save', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || 'Failed to save Business Object atomically');
      }

      navigate('/business-objects');
    } catch (err: any) {
      setSubmitError(err.message || 'An error occurred while publishing.');
    } finally {
      setIsSaving(false);
    }
  };

  return (
    <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
      <Paper sx={{ p: 4, borderRadius: 3, boxShadow: '0 8px 30px rgba(0,0,0,0.12)', bgcolor: 'background.paper' }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 4 }}>
          <Typography variant="h4" component="h1" sx={{ fontWeight: 800 }}>
            Business Object Studio
          </Typography>
          <Button
            variant="contained"
            color="success"
            startIcon={isSaving ? <CircularProgress size={20} color="inherit" /> : <SaveIcon />}
            onClick={handleSave}
            disabled={isSaving || validationSummary.errors.length > 0}
            sx={{ px: 4, py: 1.5, borderRadius: 2, fontWeight: 'bold' }}
          >
            Publish Business Object
          </Button>
        </Stack>

        {submitError && (
          <Alert severity="error" sx={{ mb: 4, borderRadius: 2 }}>{submitError}</Alert>
        )}

        <Grid container spacing={4}>
          {/* Left / Metadata Panel */}
          <Grid size={{ xs: 12, lg: 4 }}>
            <Card variant="outlined" sx={{ borderRadius: 2, height: '100%' }}>
              <CardContent sx={{ p: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 3 }}>
                  1. Business Object Identity
                </Typography>
                <Stack spacing={3}>
                  <TextField
                    required
                    label="Name"
                    value={bo.name}
                    onChange={(e) => handleBoChange('name', e.target.value)}
                    fullWidth
                  />
                  <TextField
                    required
                    label="Technical Key"
                    value={bo.key}
                    onChange={(e) => handleBoChange('key', e.target.value)}
                    fullWidth
                    helperText="Unique snake_case identifier"
                  />
                  <TextField
                    label="Description"
                    value={bo.description}
                    onChange={(e) => handleBoChange('description', e.target.value)}
                    multiline
                    rows={3}
                    fullWidth
                  />
                  <Divider sx={{ my: 1 }} />
                  <TextField
                    label="Level 3 Classification Node"
                    value={bo.classificationNodeId}
                    onChange={(e) => handleBoChange('classificationNodeId', e.target.value)}
                    fullWidth
                  />
                  <TextField
                    label="Business Key (BK)"
                    value={bo.businessKeyNodeId}
                    onChange={(e) => handleBoChange('businessKeyNodeId', e.target.value)}
                    fullWidth
                  />
                  <TextField
                    label="Semantic ID (SID)"
                    value={bo.semanticIdNodeId}
                    onChange={(e) => handleBoChange('semanticIdNodeId', e.target.value)}
                    fullWidth
                  />
                </Stack>
              </CardContent>
            </Card>
          </Grid>

          {/* Bindings area */}
          <Grid size={{ xs: 12, lg: 5 }}>
            <Stack spacing={3}>
              <Stack direction="row" justifyContent="space-between" alignItems="center">
                <Typography variant="h6" sx={{ fontWeight: 'bold' }}>
                  2. Physical Bindings
                </Typography>
                <Button variant="outlined" startIcon={<AddIcon />} onClick={addBinding}>
                  Add Backend
                </Button>
              </Stack>

              {bindings.map((b, idx) => (
                <Accordion key={b.id} defaultExpanded={idx === 0} sx={{ border: '1px solid rgba(0,0,0,0.12)', borderRadius: 2 }}>
                  <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                    <Stack direction="row" spacing={2} alignItems="center" sx={{ width: '100%' }}>
                      <Chip label={b.backendId.toUpperCase()} color="primary" variant="outlined" size="small" />
                      <Typography sx={{ fontWeight: 'bold', flexGrow: 1 }}>
                        {drivingTables.find(t => t.id === b.drivingNodeId)?.label || 'No Driving Table Anchor'}
                      </Typography>
                      {bindings.length > 1 && (
                        <IconButton size="small" color="error" onClick={(e) => { e.stopPropagation(); removeBinding(b.id); }}>
                          <DeleteIcon />
                        </IconButton>
                      )}
                    </Stack>
                  </AccordionSummary>
                  <AccordionDetails>
                    <Stack spacing={3}>
                      <Grid container spacing={2}>
                        <Grid size={{ xs: 6 }}>
                          <FormControl fullWidth size="small">
                            <InputLabel>Backend</InputLabel>
                            <Select
                              value={b.backendId}
                              label="Backend"
                              onChange={(e) => setBindings(prev => prev.map(item => item.id === b.id ? { ...item, backendId: e.target.value } : item))}
                            >
                              <MenuItem value="postgres">Postgres</MenuItem>
                              <MenuItem value="starrocks">StarRocks</MenuItem>
                              <MenuItem value="iceberg">Iceberg</MenuItem>
                            </Select>
                          </FormControl>
                        </Grid>
                        <Grid size={{ xs: 6 }}>
                          <Autocomplete
                            size="small"
                            options={drivingTables}
                            getOptionLabel={(option) => option.label}
                            value={drivingTables.find(t => t.id === b.drivingNodeId) || null}
                            onChange={(_, newValue) => handleDrivingTableChange(b.id, newValue ? newValue.id : '')}
                            renderInput={(params) => <TextField {...params} label="Driving Table" />}
                          />
                        </Grid>
                      </Grid>

                      {b.loading && (
                        <Box sx={{ display: 'flex', justifyContent: 'center', py: 2 }}>
                          <CircularProgress size={24} />
                        </Box>
                      )}

                      {/* Fields inside binding */}
                      {b.fields.length > 0 && (
                        <Box>
                          <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1.5 }}>
                            Field Term mappings
                          </Typography>
                          <Stack spacing={2}>
                            {b.fields.map(f => (
                              <Paper key={f.termNodeId} variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
                                <Grid container spacing={2} alignItems="center">
                                  <Grid size={{ xs: 12, sm: 4 }}>
                                    <Typography variant="body2" sx={{ fontWeight: 'bold' }}>{f.termName}</Typography>
                                    <Typography variant="caption" color="text.secondary">{f.fieldName}</Typography>
                                  </Grid>
                                  <Grid size={{ xs: 12, sm: 4 }}>
                                    {f.availableColumns.length > 1 ? (
                                      <FormControl fullWidth size="small">
                                        <InputLabel>Source column (Ambiguous)</InputLabel>
                                        <Select
                                          value={f.sourceNodeId}
                                          label="Source column"
                                          onChange={(e) => updateFieldProperty(b.id, f.termNodeId, 'sourceNodeId', e.target.value)}
                                        >
                                          {f.availableColumns.map(col => (
                                            <MenuItem key={col.id} value={col.id}>{col.name}</MenuItem>
                                          ))}
                                        </Select>
                                      </FormControl>
                                    ) : (
                                      <Typography variant="body2">{f.availableColumns[0]?.name || 'Unbound'}</Typography>
                                    )}
                                  </Grid>
                                  <Grid size={{ xs: 12, sm: 4 }}>
                                    <FormControl fullWidth size="small">
                                      <Select
                                        value={f.bindingRequirement}
                                        onChange={(e) => updateFieldProperty(b.id, f.termNodeId, 'bindingRequirement', e.target.value)}
                                      >
                                        <MenuItem value="REQUIRED">REQUIRED</MenuItem>
                                        <MenuItem value="OPTIONAL">OPTIONAL</MenuItem>
                                        <MenuItem value="BACKEND_SPECIFIC">BACKEND_SPECIFIC</MenuItem>
                                        <MenuItem value="CALCULATED">CALCULATED</MenuItem>
                                      </Select>
                                    </FormControl>
                                  </Grid>
                                </Grid>
                              </Paper>
                            ))}
                          </Stack>
                        </Box>
                      )}

                      {/* Relationships */}
                      <Box>
                        <Stack direction="row" justifyContent="space-between" alignItems="center" sx={{ mb: 1 }}>
                          <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>
                            Relationships
                          </Typography>
                          <Button size="small" startIcon={<AddIcon />} onClick={() => addRelationship(b.id)}>
                            Add Link
                          </Button>
                        </Stack>
                        <Stack spacing={2}>
                          {b.relationships.map((r, rIdx) => (
                            <Paper key={rIdx} variant="outlined" sx={{ p: 2, borderRadius: 2 }}>
                              <Grid container spacing={2} alignItems="center">
                                <Grid size={{ xs: 12, sm: 4 }}>
                                  <FormControl fullWidth size="small">
                                    <InputLabel>Related BO</InputLabel>
                                    <Select
                                      value={r.toBoId}
                                      label="Related BO"
                                      onChange={(e) => updateRelationship(b.id, rIdx, 'toBoId', e.target.value)}
                                    >
                                      {businessObjectsList.map(item => (
                                        <MenuItem key={item.id} value={item.id}>{item.name}</MenuItem>
                                      ))}
                                    </Select>
                                  </FormControl>
                                </Grid>
                                <Grid size={{ xs: 6, sm: 4 }}>
                                  <TextField
                                    size="small"
                                    label="Join Condition SQL"
                                    value={r.joinConditionSql}
                                    placeholder="e.g. public.customers.id = public.orders.customer_id"
                                    onChange={(e) => updateRelationship(b.id, rIdx, 'joinConditionSql', e.target.value)}
                                    fullWidth
                                  />
                                </Grid>
                                <Grid size={{ xs: 6, sm: 3 }}>
                                  <FormControl fullWidth size="small">
                                    <Select
                                      value={r.cardinality}
                                      onChange={(e) => updateRelationship(b.id, rIdx, 'cardinality', e.target.value)}
                                    >
                                      <MenuItem value="1:M">1:M (One-to-Many)</MenuItem>
                                      <MenuItem value="M:1">M:1 (Many-to-One)</MenuItem>
                                      <MenuItem value="1:1">1:1 (One-to-One)</MenuItem>
                                    </Select>
                                  </FormControl>
                                </Grid>
                                <Grid size={{ xs: 12, sm: 1 }}>
                                  <IconButton size="small" color="error" onClick={() => removeRelationship(b.id, rIdx)}>
                                    <DeleteIcon />
                                  </IconButton>
                                </Grid>
                              </Grid>
                            </Paper>
                          ))}
                        </Stack>
                      </Box>
                    </Stack>
                  </AccordionDetails>
                </Accordion>
              ))}
            </Stack>
          </Grid>

          {/* Validation summary right panel */}
          <Grid size={{ xs: 12, lg: 3 }}>
            <Card variant="outlined" sx={{ borderRadius: 2, height: '100%', bgcolor: 'rgba(0,0,0,0.02)' }}>
              <CardContent sx={{ p: 3 }}>
                <Typography variant="h6" sx={{ fontWeight: 'bold', mb: 3 }}>
                  3. Publish Readiness
                </Typography>
                {validationSummary.errors.length > 0 ? (
                  <Stack spacing={2} sx={{ mb: 3 }}>
                    <Alert severity="error" icon={<ErrorIcon />} sx={{ borderRadius: 2 }}>
                      <Typography variant="body2" sx={{ fontWeight: 'bold' }}>Coverage checks failed</Typography>
                      All REQUIRED fields must be bound in all backends before publishing.
                    </Alert>
                    <List dense>
                      {validationSummary.errors.map((err, i) => (
                        <ListItem key={i} sx={{ px: 0 }}>
                          <ListItemText primary={err} primaryTypographyProps={{ variant: 'caption', color: 'error.main' }} />
                        </ListItem>
                      ))}
                    </List>
                  </Stack>
                ) : (
                  <Alert severity="success" icon={<CheckCircleIcon />} sx={{ mb: 3, borderRadius: 2 }}>
                    Ready to publish! All schema checks passed.
                  </Alert>
                )}

                <Divider sx={{ my: 2 }} />

                <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1.5 }}>
                  Coverage Resolved
                </Typography>
                <List dense>
                  {validationSummary.successes.map((suc, i) => (
                    <ListItem key={i} sx={{ px: 0 }}>
                      <ListItemText primary={suc} primaryTypographyProps={{ variant: 'caption', color: 'success.main' }} />
                    </ListItem>
                  ))}
                </List>
              </CardContent>
            </Card>
          </Grid>
        </Grid>
      </Paper>
    </Container>
  );
};

export default BusinessObjectWizardPage;
