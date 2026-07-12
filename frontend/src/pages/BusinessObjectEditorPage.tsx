import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Container,
  Paper,
  Tabs,
  Tab,
  Typography,
  TextField,
  Button,
  Stack,
  Divider,
  Grid,
  Card,
  CardContent,
  CircularProgress,
  Alert,
  Chip,
  MenuItem,
  FormControl,
  InputLabel,
  Select,
  Fab,
  Tooltip
} from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import {
  Save as SaveIcon,
  CheckCircle as CheckCircleIcon,
  Warning as WarningIcon,
  Shield as ShieldIcon,
  Storage as StorageIcon,
  FolderSpecial as FolderSpecialIcon,
  Publish as PublishIcon
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';

// --- Tab Order mandate ---
const TAB_KEYS = ['Definition', 'Bindings', 'Scope', 'Terms Matrix', 'Validation', 'Governance', 'Review'];

interface FieldMapping {
  id: string;
  field_name: string;
  semantic_term_name: string;
  source_column_name: string;
  field_role: string;
  aggregation_type: string;
}

interface ValidationRule {
  id: string;
  rule_name: string;
  rule_type: string;
  inherited_from_term: string;
  rule_definition: string;
}

interface BOData {
  id: string;
  name: string;
  technical_key: string;
  description: string;
  datasource_type: string;
  driving_node_id: string;
  driving_node_name: string;
  fields: FieldMapping[];
  validation_rules: ValidationRule[];
  governance: {
    abac_policy: string;
    lifecycle_state: string;
    owner: string;
  };
}

// --- SUB-COMPONENT: BusinessObjectBindingsTab ---
interface BindingsTabProps {
  datasourceType: string;
  drivingNodeId: string;
  drivingNodeName: string;
  onUpdate: (field: string, value: any) => void;
}

const BusinessObjectBindingsTab: React.FC<BindingsTabProps> = ({
  datasourceType,
  drivingNodeId,
  drivingNodeName,
  onUpdate
}) => {
  return (
    <Card variant="outlined" sx={{ mt: 2 }}>
      <CardContent>
        <Typography variant="h6" sx={{ display: 'flex', alignItems: 'center', mb: 2, fontWeight: 'bold' }}>
          <StorageIcon sx={{ mr: 1, color: 'primary.main' }} /> Physical Source Database Anchors
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Manage your physical data schema mappings. The Binding-First Architecture ensures all definitions inherit properties correctly from physical drives.
        </Typography>
        <Grid container spacing={3}>
          <Grid item xs={12} md={6}>
            <FormControl fullWidth>
              <InputLabel>Datasource Endpoint Engine</InputLabel>
              <Select
                value={datasourceType}
                label="Datasource Endpoint Engine"
                onChange={(e) => onUpdate('datasource_type', e.target.value)}
              >
                <MenuItem value="Postgres">Postgres (Multi-Tenant Local)</MenuItem>
                <MenuItem value="Iceberg">Iceberg (Unified Lakehouse)</MenuItem>
                <MenuItem value="StarRocks">StarRocks (Real-time OLAP)</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12} md={6}>
            <TextField
              label="Driving Table Binding ID (UUID)"
              value={drivingNodeId}
              onChange={(e) => onUpdate('driving_node_id', e.target.value)}
              fullWidth
              disabled // Gated for safety / Read-Only UUID checks (Rule 1.3 config-before-code)
              helperText="Driving Node Binding ID is immutable after creation. Use migration plans to change anchor tables."
            />
          </Grid>
          <Grid item xs={12}>
            <TextField
              label="Target Table Reference"
              value={drivingNodeName}
              fullWidth
              disabled
            />
          </Grid>
        </Grid>
      </CardContent>
    </Card>
  );
};

// --- SUB-COMPONENT: BusinessObjectFieldsTab ---
interface FieldsTabProps {
  fields: FieldMapping[];
  onFieldsChange: (newFields: FieldMapping[]) => void;
}

const BusinessObjectFieldsTab: React.FC<FieldsTabProps> = ({ fields, onFieldsChange }) => {
  const handleFieldUpdate = (idx: number, key: keyof FieldMapping, value: any) => {
    const updated = [...fields];
    updated[idx] = { ...updated[idx], [key]: value };
    onFieldsChange(updated);
  };

  const columns: GridColDef[] = [
    { field: 'field_name', headerName: 'Field Name', flex: 1, minWidth: 150 },
    { field: 'semantic_term_name', headerName: 'Glossary Term', flex: 1, minWidth: 150 },
    { field: 'source_column_name', headerName: 'Source Column Mapping', flex: 1, minWidth: 150 },
    {
      field: 'field_role',
      headerName: 'Field Role',
      width: 180,
      renderCell: (params: GridRenderCellParams) => {
        const idx = fields.findIndex(item => item.id === params.row.id);
        return (
          <FormControl fullWidth size="small" variant="standard" sx={{ mt: 0.5 }}>
            <Select
              value={params.value || 'ATTRIBUTE'}
              onChange={(e) => handleFieldUpdate(idx, 'field_role', e.target.value)}
            >
              <MenuItem value="KEY">KEY</MenuItem>
              <MenuItem value="DIMENSION">DIMENSION</MenuItem>
              <MenuItem value="MEASURE">MEASURE</MenuItem>
              <MenuItem value="ATTRIBUTE">ATTRIBUTE</MenuItem>
            </Select>
          </FormControl>
        );
      }
    },
    {
      field: 'aggregation_type',
      headerName: 'Aggregation',
      width: 180,
      renderCell: (params: GridRenderCellParams) => {
        const idx = fields.findIndex(item => item.id === params.row.id);
        return (
          <FormControl fullWidth size="small" variant="standard" sx={{ mt: 0.5 }}>
            <Select
              value={params.value || 'NONE'}
              onChange={(e) => handleFieldUpdate(idx, 'aggregation_type', e.target.value)}
            >
              <MenuItem value="NONE">NONE</MenuItem>
              <MenuItem value="SUM">SUM</MenuItem>
              <MenuItem value="AVG">AVG</MenuItem>
              <MenuItem value="MIN">MIN</MenuItem>
              <MenuItem value="MAX">MAX</MenuItem>
            </Select>
          </FormControl>
        );
      }
    }
  ];

  return (
    <Box sx={{ height: 400, width: '100%', mt: 2 }}>
      <DataGrid
        rows={fields}
        columns={columns}
        getRowId={(row) => row.id}
        disableRowSelectionOnClick
      />
    </Box>
  );
};

export default function BusinessObjectEditorPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { tenant } = useTenant();
  
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  
  const [boData, setBoData] = useState<BOData | null>(null);

  // Fetch BO details
  useEffect(() => {
    const fetchBODetails = async () => {
      setLoading(true);
      try {
        const response = await fetch(`/api/business-objects/${id}`);
        if (response.ok) {
          const data = await response.json();
          setBoData(data);
        } else {
          // Dynamic Demo Fallback
          setBoData({
            id: id || 'demo-bo-uuid',
            name: 'Monthly Revenue Analytics',
            technical_key: 'monthly_revenue_analytics',
            description: 'Analyzes financial revenues aggregated monthly by regional nodes.',
            datasource_type: 'Postgres',
            driving_node_id: '11111111-1111-1111-1111-111111111111',
            driving_node_name: 'sales_ledger',
            fields: [
              {
                id: 'f1',
                field_name: 'total_revenue',
                semantic_term_name: 'Revenue Amount',
                source_column_name: 'amount',
                field_role: 'MEASURE',
                aggregation_type: 'SUM'
              },
              {
                id: 'f2',
                field_name: 'region_code',
                semantic_term_name: 'Region Identifier',
                source_column_name: 'region_id',
                field_role: 'DIMENSION',
                aggregation_type: 'NONE'
              }
            ],
            validation_rules: [
              {
                id: 'v1',
                rule_name: 'Non-Negative Revenue',
                rule_type: 'CHECK CONSTRAINT',
                inherited_from_term: 'Revenue Amount',
                rule_definition: 'value >= 0'
              },
              {
                id: 'v2',
                rule_name: 'Valid Region Code',
                rule_type: 'FOREIGN KEY CHECK',
                inherited_from_term: 'Region Identifier',
                rule_definition: 'region_id IS NOT NULL'
              }
            ],
            governance: {
              abac_policy: 'finance-officer-only',
              lifecycle_state: 'APPROVED',
              owner: 'John Doe'
            }
          });
        }
      } catch {
        // demo fallback
        setBoData({
          id: id || 'demo-bo-uuid',
          name: 'Monthly Revenue Analytics',
          technical_key: 'monthly_revenue_analytics',
          description: 'Analyzes financial revenues aggregated monthly by regional nodes.',
          datasource_type: 'Postgres',
          driving_node_id: '11111111-1111-1111-1111-111111111111',
          driving_node_name: 'sales_ledger',
          fields: [
            {
              id: 'f1',
              field_name: 'total_revenue',
              semantic_term_name: 'Revenue Amount',
              source_column_name: 'amount',
              field_role: 'MEASURE',
              aggregation_type: 'SUM'
            },
            {
              id: 'f2',
              field_name: 'region_code',
              semantic_term_name: 'Region Identifier',
              source_column_name: 'region_id',
              field_role: 'DIMENSION',
              aggregation_type: 'NONE'
            }
          ],
          validation_rules: [
            {
              id: 'v1',
              rule_name: 'Non-Negative Revenue',
              rule_type: 'CHECK CONSTRAINT',
              inherited_from_term: 'Revenue Amount',
              rule_definition: 'value >= 0'
            }
          ],
          governance: {
            abac_policy: 'finance-officer-only',
            lifecycle_state: 'APPROVED',
            owner: 'John Doe'
          }
        });
      } finally {
        setLoading(false);
      }
    };
    fetchBODetails();
  }, [id]);

  const handleUpdateField = (key: string, value: any) => {
    if (!boData) return;
    setBoData({ ...boData, [key]: value });
  };

  const handleSave = async () => {
    if (!boData) return;
    setSaving(true);
    try {
      const response = await fetch(`/api/business-objects/${boData.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(boData)
      });
      if (response.ok) {
        navigate('/business-objects');
      } else {
        const text = await response.text();
        setError(text || 'Failed to save modifications');
      }
    } catch {
      setError('A network error occurred while updating the Business Object.');
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', p: 8 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!boData) {
    return <Alert severity="error">Business Object not found.</Alert>;
  }

  return (
    <Container maxWidth="xl" sx={{ mt: 4, mb: 4 }}>
      <Paper sx={{ p: 4, borderRadius: 3, boxShadow: '0 4px 20px rgba(0,0,0,0.08)' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Box>
            <Typography variant="h5" component="h1" sx={{ fontWeight: 'bold' }}>
              Edit: {boData.name}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Technical Key: {boData.technical_key}
            </Typography>
          </Box>
          <Button
            variant="contained"
            color="primary"
            startIcon={saving ? <CircularProgress size={20} color="inherit" /> : <SaveIcon />}
            onClick={handleSave}
            disabled={saving}
          >
            Save Changes
          </Button>
        </Box>

        <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
          <Tabs
            value={activeTab}
            onChange={(_, val) => setActiveTab(val)}
            variant="scrollable"
            scrollButtons="auto"
          >
            {TAB_KEYS.map((tab, idx) => (
              <Tab key={idx} label={tab} id={`editor-tab-${idx}`} />
            ))}
          </Tabs>
        </Box>

        {error && <Alert severity="error" sx={{ mb: 3 }}>{error}</Alert>}

        {/* TAB 1: DEFINITION */}
        {activeTab === 0 && (
          <Stack spacing={3} sx={{ maxWidth: 600 }}>
            <TextField
              label="Business Object Name"
              value={boData.name}
              onChange={(e) => handleUpdateField('name', e.target.value)}
              fullWidth
            />
            <TextField
              label="Technical Key (auto-formatted)"
              value={boData.technical_key}
              disabled
              fullWidth
              helperText="Immutable after publishing."
            />
            <TextField
              label="Description"
              value={boData.description}
              onChange={(e) => handleUpdateField('description', e.target.value)}
              multiline
              rows={4}
              fullWidth
            />
          </Stack>
        )}

        {/* TAB 2: BINDINGS */}
        {activeTab === 1 && (
          <BusinessObjectBindingsTab
            datasourceType={boData.datasource_type}
            drivingNodeId={boData.driving_node_id}
            drivingNodeName={boData.driving_node_name}
            onUpdate={handleUpdateField}
          />
        )}

        {/* TAB 3: SCOPE */}
        {activeTab === 2 && (
          <Card variant="outlined" sx={{ mt: 2 }}>
            <CardContent>
              <Typography variant="h6" gutterBottom sx={{ fontWeight: 'bold' }}>
                Scope & Relationship Graph
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                Visualize semantic edges, parent nodes, and relationships configured inside this Business Object context.
              </Typography>
              <Box sx={{ p: 4, bgcolor: '#f4f6f8', borderRadius: 2, display: 'flex', justifyContent: 'center', alignItems: 'center', height: 250, border: '1px dashed #cfd8dc' }}>
                <Typography color="text.secondary" variant="subtitle2">
                  [Relationship Graph Visualization Canvas: {boData.driving_node_name} Node Anchor Active]
                </Typography>
              </Box>
            </CardContent>
          </Card>
        )}

        {/* TAB 4: TERMS MATRIX */}
        {activeTab === 3 && (
          <BusinessObjectFieldsTab
            fields={boData.fields}
            onFieldsChange={(newFields) => handleUpdateField('fields', newFields)}
          />
        )}

        {/* TAB 5: VALIDATION */}
        {activeTab === 4 && (
          <Stack spacing={3}>
            <Alert severity="info">
              Validation Rules are defined and managed inside the Glossary semantic terms level. They are <strong>read-only</strong> inside the Business Object manager.
            </Alert>
            {boData.validation_rules.map((rule) => (
              <Card key={rule.id} variant="outlined">
                <CardContent sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <Box>
                    <Typography variant="subtitle1" sx={{ fontWeight: 'bold' }}>
                      {rule.rule_name}
                    </Typography>
                    <Typography variant="caption" display="block" color="text.secondary" sx={{ mb: 1 }}>
                      Type: {rule.rule_type} | Inherited From: {rule.inherited_from_term}
                    </Typography>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace', p: 1, bgcolor: '#f5f5f5', borderRadius: 1 }}>
                      {rule.rule_definition}
                    </Typography>
                  </Box>
                  <Chip label="ACTIVE" color="success" size="small" variant="outlined" />
                </CardContent>
              </Card>
            ))}
          </Stack>
        )}

        {/* TAB 6: GOVERNANCE */}
        {activeTab === 5 && (
          <Grid container spacing={3}>
            <Grid item xs={12} md={4}>
              <Card variant="outlined">
                <CardContent>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    ABAC Auth Policy
                  </Typography>
                  <Chip icon={<ShieldIcon />} label={boData.governance.abac_policy} color="primary" />
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} md={4}>
              <Card variant="outlined">
                <CardContent>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Lifecycle State
                  </Typography>
                  <Chip label={boData.governance.lifecycle_state} color="success" variant="outlined" />
                </CardContent>
              </Card>
            </Grid>
            <Grid item xs={12} md={4}>
              <Card variant="outlined">
                <CardContent>
                  <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                    Owner Matrix
                  </Typography>
                  <Typography variant="body1" sx={{ fontWeight: 'bold' }}>
                    {boData.governance.owner}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          </Grid>
        )}

        {/* TAB 7: REVIEW */}
        {activeTab === 6 && (
          <Stack spacing={3}>
            <Typography variant="h6" sx={{ fontWeight: 'bold' }}>Review & Publish</Typography>
            <Typography variant="body2" color="text.secondary">
              Review current specs before triggering cache invalidation across distributed services (Distributed Cache Rule 8.3).
            </Typography>
            <Paper variant="outlined" sx={{ p: 3 }}>
              <Grid container spacing={2}>
                <Grid item xs={6}><Typography variant="body2" color="text.secondary">Name:</Typography></Grid>
                <Grid item xs={6}><Typography variant="body2" sx={{ fontWeight: 'bold' }}>{boData.name}</Typography></Grid>
                <Grid item xs={6}><Typography variant="body2" color="text.secondary">Technical Key:</Typography></Grid>
                <Grid item xs={6}><Typography variant="body2" sx={{ fontWeight: 'bold' }}>{boData.technical_key}</Typography></Grid>
                <Grid item xs={6}><Typography variant="body2" color="text.secondary">Bindings Source:</Typography></Grid>
                <Grid item xs={6}><Typography variant="body2" sx={{ fontWeight: 'bold' }}>{boData.datasource_type} ({boData.driving_node_name})</Typography></Grid>
                <Grid item xs={6}><Typography variant="body2" color="text.secondary">Total Fields Mapped:</Typography></Grid>
                <Grid item xs={6}><Typography variant="body2" sx={{ fontWeight: 'bold' }}>{boData.fields.length}</Typography></Grid>
              </Grid>
            </Paper>
          </Stack>
        )}

        {/* Review Floating Action Button */}
        {activeTab === 6 && (
          <Box sx={{ position: 'fixed', bottom: 32, right: 32 }}>
            <Tooltip title="Publish and Sync Metadata">
              <Fab color="success" aria-label="publish" onClick={handleSave}>
                <PublishIcon />
              </Fab>
            </Tooltip>
          </Box>
        )}
      </Paper>
    </Container>
  );
}
