import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  IconButton,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Button,
  Divider,
  Alert,
  CircularProgress,
  Chip,
  Tabs,
  Tab,
  Switch,
  FormControlLabel,
  useTheme,
} from '@mui/material';
import {
  X,
  Play,
  Plus,
  Trash2,
  CheckCircle2,
  AlertCircle,
  Code2,
  Workflow,
  Edit3,
  Database,
  ExternalLink,
} from 'lucide-react';
import axios from '@/utils/axiosClient';
import { PipelineNodeData } from '../types/pipeline';

interface TileConfigDrawerProps {
  node: { id: string; data: PipelineNodeData } | null;
  onClose: () => void;
  onUpdateConfig: (nodeId: string, updatedData: Partial<PipelineNodeData>) => void;
}

export const TileConfigDrawer: React.FC<TileConfigDrawerProps> = ({
  node,
  onClose,
  onUpdateConfig,
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';

  if (!node) return null;

  const [label, setLabel] = useState(node.data.label);
  const [description, setDescription] = useState(node.data.description || '');
  const [config, setConfig] = useState<Record<string, any>>({ ...node.data.config });
  const [activeTab, setActiveTab] = useState(0);

  // Discovery lists
  const [apiEndpoints, setApiEndpoints] = useState<any[]>([]);
  const [workflows, setWorkflows] = useState<any[]>([]);
  const [loadingSchemas, setLoadingSchemas] = useState(false);

  // Live Test Step state
  const [testInput, setTestInput] = useState<string>(
    '[\n  {\n    "id": "11111111-2222-3333-4444-555555555555",\n    "account_number": "ACC-INST-99",\n    "account_name": "Acme Global Treasury",\n    "subtype_code": "institutional",\n    "base_currency": "USD",\n    "status": "active"\n  }\n]'
  );
  const [testResult, setTestResult] = useState<any>(null);
  const [testing, setTesting] = useState(false);
  const [testError, setTestError] = useState<string | null>(null);

  useEffect(() => {
    setLabel(node.data.label);
    setDescription(node.data.description || '');
    setConfig({ ...node.data.config });
    setTestResult(null);
    setTestError(null);
  }, [node.id]);

  useEffect(() => {
    // Load schema discoveries when opening API Caller or Workflow Caller
    const loadDiscoveries = async () => {
      try {
        setLoadingSchemas(true);
        if (node.data.subType === 'api_caller') {
          const res = await axios.get('/api/v1/data-pipelines/schema/api-endpoints');
          setApiEndpoints(res.data || []);
        } else if (node.data.subType === 'workflow_caller') {
          const res = await axios.get('/api/v1/data-pipelines/schema/workflows');
          setWorkflows(res.data || []);
        }
      } catch (e) {
        console.error('Failed to load schema discovery:', e);
      } finally {
        setLoadingSchemas(false);
      }
    };
    loadDiscoveries();
  }, [node.data.subType]);

  const handleConfigChange = (key: string, value: any) => {
    const updated = { ...config, [key]: value };
    setConfig(updated);
    onUpdateConfig(node.id, { config: updated });
  };

  // Run single-step live test
  const handleRunTestStep = async () => {
    try {
      setTesting(true);
      setTestError(null);
      let parsedInput = [];
      try {
        parsedInput = JSON.parse(testInput);
      } catch (err) {
        setTestError('Invalid Test Input JSON array');
        setTesting(false);
        return;
      }

      const res = await axios.post('/api/v1/data-pipelines/test-step', {
        node_type: node.data.category,
        sub_type: node.data.subType,
        config: config,
        input: parsedInput,
      });

      setTestResult(res.data);
    } catch (err: any) {
      setTestError(err.response?.data?.error || err.message || 'Test failed');
    } finally {
      setTesting(false);
    }
  };

  return (
    <Box
      sx={{
        width: 440,
        height: '100%',
        backgroundColor: theme.palette.background.paper,
        borderLeft: `1px solid ${theme.palette.divider}`,
        display: 'flex',
        flexDirection: 'column',
        boxShadow: isDark ? '-4px 0 20px rgba(0,0,0,0.4)' : '-4px 0 16px rgba(0,0,0,0.06)',
        position: 'relative',
        zIndex: 20,
      }}
    >
      {/* Header */}
      <Box sx={{ p: 2, borderBottom: `1px solid ${theme.palette.divider}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 800, color: theme.palette.text.primary }}>
              Tile Inspector
            </Typography>
            <Chip label={node.data.badge || node.data.category} size="small" color="primary" sx={{ height: 20, fontSize: '0.65rem', fontWeight: 700 }} />
          </Box>
          <Typography variant="caption" sx={{ color: theme.palette.text.secondary }}>
            Configure operator parameters, APIs, and business workflows
          </Typography>
        </Box>
        <IconButton size="small" onClick={onClose}>
          <X size={18} color={isDark ? '#94a3b8' : '#475569'} />
        </IconButton>
      </Box>

      {/* Tabs */}
      <Tabs
        value={activeTab}
        onChange={(_, v) => setActiveTab(v)}
        sx={{ borderBottom: `1px solid ${theme.palette.divider}`, px: 2, minHeight: 40 }}
      >
        <Tab label="Configuration" sx={{ minHeight: 40, fontSize: '0.8rem', fontWeight: 700 }} />
        <Tab label="Live Test Step" sx={{ minHeight: 40, fontSize: '0.8rem', fontWeight: 700 }} />
      </Tabs>

      {/* Tab 1: Config */}
      {activeTab === 0 && (
        <Box sx={{ flex: 1, overflowY: 'auto', p: 2.5 }}>
          <TextField
            label="Tile Title"
            size="small"
            fullWidth
            value={label}
            onChange={(e) => {
              setLabel(e.target.value);
              onUpdateConfig(node.id, { label: e.target.value });
            }}
            sx={{ mb: 2 }}
          />

          <TextField
            label="Description"
            size="small"
            fullWidth
            multiline
            rows={2}
            value={description}
            onChange={(e) => {
              setDescription(e.target.value);
              onUpdateConfig(node.id, { description: e.target.value });
            }}
            sx={{ mb: 2.5 }}
          />

          <Divider sx={{ my: 2 }} />
          <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1.5, color: theme.palette.text.primary }}>
            Operator Parameters
          </Typography>

          {/* 1. API Builder Invoker Config */}
          {node.data.subType === 'api_caller' && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <FormControl size="small" fullWidth>
                <InputLabel>Published API Builder Endpoint</InputLabel>
                <Select
                  value={config.endpoint_url || ''}
                  label="Published API Builder Endpoint"
                  onChange={(e) => {
                    const sel = apiEndpoints.find((ep) => ep.path === e.target.value);
                    if (sel) {
                      handleConfigChange('endpoint_url', sel.path);
                      handleConfigChange('method', sel.method);
                    } else {
                      handleConfigChange('endpoint_url', e.target.value);
                    }
                  }}
                >
                  {apiEndpoints.map((ep) => (
                    <MenuItem key={ep.id} value={ep.path}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Chip label={ep.method} size="small" sx={{ height: 18, fontSize: '0.6rem', fontWeight: 800 }} />
                        <Typography variant="body2" sx={{ fontSize: '0.8rem', fontWeight: 600 }}>
                          {ep.name} ({ep.path})
                        </Typography>
                      </Box>
                    </MenuItem>
                  ))}
                  <MenuItem value="custom">-- Custom Endpoint URL --</MenuItem>
                </Select>
              </FormControl>

              <Box sx={{ display: 'flex', gap: 1.5 }}>
                <FormControl size="small" sx={{ width: 120 }}>
                  <InputLabel>Method</InputLabel>
                  <Select
                    value={config.method || 'POST'}
                    label="Method"
                    onChange={(e) => handleConfigChange('method', e.target.value)}
                  >
                    <MenuItem value="GET">GET</MenuItem>
                    <MenuItem value="POST">POST</MenuItem>
                    <MenuItem value="PUT">PUT</MenuItem>
                    <MenuItem value="DELETE">DELETE</MenuItem>
                  </Select>
                </FormControl>

                <TextField
                  label="Endpoint Path / URL"
                  size="small"
                  fullWidth
                  value={config.endpoint_url || ''}
                  onChange={(e) => handleConfigChange('endpoint_url', e.target.value)}
                  placeholder="/api/v1/customers/verify-kyc"
                />
              </Box>

              <TextField
                label="Target Response Field"
                size="small"
                value={config.target_field || '_api_response'}
                onChange={(e) => handleConfigChange('target_field', e.target.value)}
                helperText="Store the API return object into this field on each stream record"
              />

              <FormControlLabel
                control={
                  <Switch
                    checked={config.merge_output ?? true}
                    onChange={(e) => handleConfigChange('merge_output', e.target.checked)}
                  />
                }
                label={<Typography variant="caption" sx={{ fontWeight: 600 }}>Merge Response Directly into Stream Record</Typography>}
              />
            </Box>
          )}

          {/* 2. Flow Builder / Workflow Invoker Config */}
          {node.data.subType === 'workflow_caller' && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <FormControl size="small" fullWidth>
                <InputLabel>Flow Builder Workflow</InputLabel>
                <Select
                  value={config.workflow_id || ''}
                  label="Flow Builder Workflow"
                  onChange={(e) => {
                    const sel = workflows.find((wf) => wf.id === e.target.value);
                    if (sel) {
                      handleConfigChange('workflow_id', sel.id);
                      handleConfigChange('workflow_name', sel.name);
                    }
                  }}
                >
                  {workflows.map((wf) => (
                    <MenuItem key={wf.id} value={wf.id}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Workflow size={14} color="#ec4899" />
                        <Typography variant="body2" sx={{ fontSize: '0.8rem', fontWeight: 600 }}>
                          {wf.name}
                        </Typography>
                      </Box>
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>

              <FormControl size="small" fullWidth>
                <InputLabel>Execution Mode</InputLabel>
                <Select
                  value={config.mode || 'sync'}
                  label="Execution Mode"
                  onChange={(e) => handleConfigChange('mode', e.target.value)}
                >
                  <MenuItem value="sync">Synchronous (Wait for workflow output before next tile)</MenuItem>
                  <MenuItem value="async">Asynchronous (Dispatch workflow & record run ID)</MenuItem>
                </Select>
              </FormControl>

              <TextField
                label="Workflow Name Override"
                size="small"
                value={config.workflow_name || ''}
                onChange={(e) => handleConfigChange('workflow_name', e.target.value)}
              />
            </Box>
          )}

          {/* 3. Business Object CRUD Config */}
          {node.data.subType === 'bo_crud' && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <FormControl size="small" fullWidth>
                <InputLabel>CRUD Action</InputLabel>
                <Select
                  value={config.operation || 'UPDATE'}
                  label="CRUD Action"
                  onChange={(e) => handleConfigChange('operation', e.target.value)}
                >
                  <MenuItem value="INSERT">CREATE / INSERT (Create new bitemporal entities)</MenuItem>
                  <MenuItem value="READ">READ / QUERY (Extract matching entities)</MenuItem>
                  <MenuItem value="UPDATE">UPDATE (Modify existing record by ID)</MenuItem>
                  <MenuItem value="DELETE">DELETE / SOFT_DELETE (Set valid_to = NOW())</MenuItem>
                </Select>
              </FormControl>

              <FormControl size="small" fullWidth>
                <InputLabel>Target STI Entity Table</InputLabel>
                <Select
                  value={config.table || 'oms.account'}
                  label="Target STI Entity Table"
                  onChange={(e) => handleConfigChange('table', e.target.value)}
                >
                  <MenuItem value="oms.account">oms.account (Accounts)</MenuItem>
                  <MenuItem value="oms.position">oms.position (Positions)</MenuItem>
                  <MenuItem value="oms.security">oms.security (Securities)</MenuItem>
                  <MenuItem value="oms.trade_order">oms.trade_order (Trade Orders)</MenuItem>
                  <MenuItem value="altinv.alternative_investment">altinv.alternative_investment (Alt Assets)</MenuItem>
                  <MenuItem value="cash_flow.settlement">cash_flow.settlement (Settlements)</MenuItem>
                  <MenuItem value="master.customer">master.customer (Customers)</MenuItem>
                  <MenuItem value="master.vendor">master.vendor (Vendors)</MenuItem>
                  <MenuItem value="master.personnel">master.personnel (Personnel)</MenuItem>
                  <MenuItem value="master.sales_ledger">master.sales_ledger (Sales Ledger)</MenuItem>
                </Select>
              </FormControl>
            </Box>
          )}

          {/* 4. STI Reader / Loader Config */}
          {(node.data.subType === 'bo_reader' || node.data.subType === 'bo_loader') && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <FormControl size="small" fullWidth>
                <InputLabel>STI Table Target</InputLabel>
                <Select
                  value={config.table || 'oms.trade_order'}
                  label="STI Table Target"
                  onChange={(e) => handleConfigChange('table', e.target.value)}
                >
                  <MenuItem value="oms.account">oms.account (Accounts)</MenuItem>
                  <MenuItem value="oms.position">oms.position (Positions)</MenuItem>
                  <MenuItem value="oms.security">oms.security (Securities)</MenuItem>
                  <MenuItem value="oms.trade_order">oms.trade_order (Trade Orders)</MenuItem>
                  <MenuItem value="altinv.alternative_investment">altinv.alternative_investment (Private Mkts)</MenuItem>
                  <MenuItem value="cash_flow.settlement">cash_flow.settlement (Settlements)</MenuItem>
                  <MenuItem value="master.customer">master.customer (Customers)</MenuItem>
                  <MenuItem value="master.vendor">master.vendor (Vendors)</MenuItem>
                  <MenuItem value="master.personnel">master.personnel (Personnel)</MenuItem>
                  <MenuItem value="master.sales_ledger">master.sales_ledger (Sales Ledger)</MenuItem>
                </Select>
              </FormControl>

              {node.data.subType === 'bo_reader' && (
                <TextField
                  label="Filter Subtype Code (Optional)"
                  size="small"
                  value={config.subtype_code || ''}
                  onChange={(e) => handleConfigChange('subtype_code', e.target.value)}
                  placeholder="e.g. institutional, sma"
                />
              )}
            </Box>
          )}

          {/* 5. Catalog Graph Config */}
          {(node.data.subType === 'catalog_reader' || node.data.subType === 'catalog_loader') && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <FormControl size="small" fullWidth>
                <InputLabel>Catalog Asset Type</InputLabel>
                <Select
                  value={config.catalog_type || 'TABLE'}
                  label="Catalog Asset Type"
                  onChange={(e) => handleConfigChange('catalog_type', e.target.value)}
                >
                  <MenuItem value="TABLE">TABLE (Physical Schema Tables)</MenuItem>
                  <MenuItem value="ATTRIBUTE">ATTRIBUTE (Physical Columns / Attributes)</MenuItem>
                  <MenuItem value="BUSINESS_OBJECT">BUSINESS_OBJECT (Core Entities)</MenuItem>
                  <MenuItem value="BLOOMBERG_FIELD">BLOOMBERG_FIELD (Bloomberg Data License Fields)</MenuItem>
                  <MenuItem value="SEMANTIC_TERM">SEMANTIC_TERM (Glossary Terms)</MenuItem>
                  <MenuItem value="METRIC">METRIC (Calculated Measures)</MenuItem>
                </Select>
              </FormControl>
            </Box>
          )}

          {/* 6. Column Mapper Config */}
          {node.data.subType === 'column_mapper' && (
            <Box>
              <Typography variant="caption" sx={{ color: theme.palette.text.secondary, mb: 1, display: 'block' }}>
                Map Source Column Headers to Target Schema Headers:
              </Typography>
              {Object.entries(config.mappings || {}).map(([tgt, src], idx) => (
                <Box key={idx} sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                  <TextField size="small" placeholder="Target" value={tgt} disabled sx={{ flex: 1 }} />
                  <Typography variant="caption" sx={{ fontWeight: 700, color: theme.palette.text.primary }}>←</Typography>
                  <TextField
                    size="small"
                    placeholder="Source"
                    value={String(src)}
                    onChange={(e) => {
                      const updated = { ...config.mappings, [tgt]: e.target.value };
                      handleConfigChange('mappings', updated);
                    }}
                    sx={{ flex: 1 }}
                  />
                  <IconButton
                    size="small"
                    color="error"
                    onClick={() => {
                      const updated = { ...config.mappings };
                      delete updated[tgt];
                      handleConfigChange('mappings', updated);
                    }}
                  >
                    <Trash2 size={14} />
                  </IconButton>
                </Box>
              ))}

              <Button
                size="small"
                startIcon={<Plus size={14} />}
                variant="outlined"
                sx={{ mt: 1 }}
                onClick={() => {
                  const targetName = prompt('Enter Target Field Name (e.g. account_number):');
                  if (targetName) {
                    const srcName = prompt('Enter Source Field Name (e.g. ext_acc):') || targetName;
                    handleConfigChange('mappings', {
                      ...(config.mappings || {}),
                      [targetName]: srcName,
                    });
                  }
                }}
              >
                Add Field Mapping
              </Button>
            </Box>
          )}

          {/* 7. Filter Config */}
          {node.data.subType === 'filter' && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <TextField
                label="Target Field"
                size="small"
                value={config.field || ''}
                onChange={(e) => handleConfigChange('field', e.target.value)}
                placeholder="e.g. status, amount"
              />
              <FormControl size="small" fullWidth>
                <InputLabel>Condition Operator</InputLabel>
                <Select
                  value={config.operator || 'eq'}
                  label="Condition Operator"
                  onChange={(e) => handleConfigChange('operator', e.target.value)}
                >
                  <MenuItem value="eq">Equal (=)</MenuItem>
                  <MenuItem value="neq">Not Equal (!=)</MenuItem>
                  <MenuItem value="gt">Greater Than (&gt;)</MenuItem>
                  <MenuItem value="lt">Less Than (&lt;)</MenuItem>
                  <MenuItem value="contains">Contains Substring</MenuItem>
                  <MenuItem value="not_null">Is Not Null / Not Empty</MenuItem>
                </Select>
              </FormControl>
              <TextField
                label="Condition Value"
                size="small"
                value={config.value || ''}
                onChange={(e) => handleConfigChange('value', e.target.value)}
              />
            </Box>
          )}

          {/* 8. Graph Synthesizer Config */}
          {node.data.subType === 'graph_synthesizer' && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <TextField
                label="Parent Table Field"
                size="small"
                value={config.parent_field || 'table_name'}
                onChange={(e) => handleConfigChange('parent_field', e.target.value)}
              />
              <TextField
                label="Child Column Field"
                size="small"
                value={config.child_field || 'column_name'}
                onChange={(e) => handleConfigChange('child_field', e.target.value)}
              />
              <TextField
                label="Data Type Field"
                size="small"
                value={config.data_type_field || 'data_type'}
                onChange={(e) => handleConfigChange('data_type_field', e.target.value)}
              />
              <FormControl size="small" fullWidth>
                <InputLabel>Edge Relationship Predicate</InputLabel>
                <Select
                  value={config.edge_predicate || 'COLUMN_OF'}
                  label="Edge Relationship Predicate"
                  onChange={(e) => handleConfigChange('edge_predicate', e.target.value)}
                >
                  <MenuItem value="COLUMN_OF">COLUMN_OF (Physical Column of Table)</MenuItem>
                  <MenuItem value="ATTRIBUTE_OF">ATTRIBUTE_OF (Attribute of Business Object)</MenuItem>
                  <MenuItem value="IS_CLASSIFIED_AS">IS_CLASSIFIED_AS (Semantic Term Link)</MenuItem>
                  <MenuItem value="CHILD_OF">CHILD_OF (Hierarchical Parent/Child)</MenuItem>
                </Select>
              </FormControl>
            </Box>
          )}

          {/* 9. Bloomberg Fields Mapper Config */}
          {node.data.subType === 'bloomberg_field_mapper' && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <TextField
                label="Catalog Qualified Path Prefix"
                size="small"
                value={config.category_prefix || 'bloomberg.fields'}
                onChange={(e) => handleConfigChange('category_prefix', e.target.value)}
                helperText="Prefix for catalog qualified_path (e.g. bloomberg.fields/144A_FLAG)"
              />
              <Alert severity="info" sx={{ fontSize: '0.75rem' }}>
                Converts raw <code>bb_fields.csv</code> rows into <code>BLOOMBERG_FIELD</code> catalog nodes, with all market sector eligibility flags (Comdty, Equity, Muni, Pfd, MMkt, Govt, Corp, Index, Curncy, Mtge) parsed into structured JSON properties.
              </Alert>
            </Box>
          )}
        </Box>
      )}

      {/* Tab 2: Live Test Step */}
      {activeTab === 1 && (
        <Box sx={{ flex: 1, overflowY: 'auto', p: 2.5, display: 'flex', flexDirection: 'column' }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 700, mb: 1, color: theme.palette.text.primary }}>
            Sample Input Records (JSON):
          </Typography>
          <TextField
            multiline
            rows={5}
            fullWidth
            value={testInput}
            onChange={(e) => setTestInput(e.target.value)}
            sx={{
              fontFamily: 'monospace',
              fontSize: '0.75rem',
              mb: 2,
              '& .MuiInputBase-input': {
                fontFamily: 'monospace',
                backgroundColor: isDark ? 'rgba(0,0,0,0.2)' : 'transparent',
              },
            }}
          />

          <Button
            variant="contained"
            color="primary"
            startIcon={testing ? <CircularProgress size={16} color="inherit" /> : <Play size={16} />}
            onClick={handleRunTestStep}
            disabled={testing}
            sx={{ mb: 2 }}
          >
            {testing ? 'Executing Step...' : 'Execute Live Test'}
          </Button>

          {testError && (
            <Alert severity="error" icon={<AlertCircle size={16} />} sx={{ mb: 2 }}>
              {testError}
            </Alert>
          )}

          {testResult && (
            <Box sx={{ flex: 1 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                <CheckCircle2 size={16} color="#10b981" />
                <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#10b981' }}>
                  Execution Succeeded in {testResult.execution_ms}ms
                </Typography>
              </Box>

              <Typography variant="caption" sx={{ color: theme.palette.text.secondary, display: 'block', mb: 0.5 }}>
                Transformed Output Records:
              </Typography>
              <Box
                sx={{
                  p: 1.5,
                  backgroundColor: isDark ? 'rgba(0, 0, 0, 0.3)' : '#f8fafc',
                  borderRadius: '6px',
                  border: `1px solid ${theme.palette.divider}`,
                  maxHeight: 250,
                  overflowY: 'auto',
                }}
              >
                <pre style={{ margin: 0, fontSize: '0.7rem', fontFamily: 'monospace', color: theme.palette.text.primary }}>
                  {JSON.stringify(testResult.output, null, 2)}
                </pre>
              </Box>
            </Box>
          )}
        </Box>
      )}
    </Box>
  );
};
