import React, { useState, useEffect } from 'react';
import {
  Box,
  Button,
  Card,
  Dialog,
  Grid,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TextField,
  Typography,
  Chip,
  IconButton,
  Snackbar,
  Alert,
  Tooltip,
  FormControlLabel,
  Checkbox,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import {
  Edit as EditIcon,
  Delete as DeleteIcon,
  PlayArrow as PlayArrowIcon,
  ContentCopy as ContentCopyIcon,
  Visibility as VisibilityIcon,
  MoreVert as MoreVertIcon,
} from '@mui/icons-material';
import { Monaco } from '@monaco-editor/react';
import Editor from '@monaco-editor/react';

// ============================================================================
// Types
// ============================================================================

interface TemplateParamDef {
  name: string;
  type: 'string' | 'number' | 'bool';
  required: boolean;
  default?: any;
  help?: string;
}

interface SemanticQuery {
  datasource: string;
  version?: string;
  select: string[];
  filters: any[];
  order_by?: any[];
  limit?: number;
}

interface SemanticQueryTemplate {
  id: string;
  tenant_id?: string;
  name: string;
  description?: string;
  datasource: string;
  version: string;
  semantic_query: SemanticQuery;
  parameters: TemplateParamDef[];
  created_by: string;
  created_at: string;
  updated_at: string;
  visibility: string;
  tags: string[];
  deprecated: boolean;
}

interface TemplateRunResponse {
  datasource: string;
  version: string;
  sql: string;
  rows: any[];
  count: number;
  executed_at: string;
  duration_ms: number;
}

// ============================================================================
// API Client
// ============================================================================

const templateApi = {
  async createTemplate(template: Partial<SemanticQueryTemplate>) {
    const response = await fetch('/api/semantic/templates', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'tenant-1' },
      body: JSON.stringify(template),
    });
    return response.json();
  },

  async listTemplates(datasource?: string, version?: string) {
    const params = new URLSearchParams();
    if (datasource) params.append('datasource', datasource);
    if (version) params.append('version', version);

    const response = await fetch(`/api/semantic/templates?${params}`, {
      headers: { 'X-Tenant-ID': 'tenant-1' },
    });
    return response.json();
  },

  async getTemplate(id: string) {
    const response = await fetch(`/api/semantic/templates/${id}`, {
      headers: { 'X-Tenant-ID': 'tenant-1' },
    });
    return response.json();
  },

  async updateTemplate(id: string, changes: any, changeMessage?: string) {
    const response = await fetch(`/api/semantic/templates/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'tenant-1' },
      body: JSON.stringify({ ...changes, change_message: changeMessage }),
    });
    return response.json();
  },

  async deleteTemplate(id: string) {
    await fetch(`/api/semantic/templates/${id}`, {
      method: 'DELETE',
      headers: { 'X-Tenant-ID': 'tenant-1' },
    });
  },

  async runTemplate(id: string, params: Record<string, any>) {
    const response = await fetch(`/api/semantic/templates/${id}/run`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': 'tenant-1' },
      body: JSON.stringify({ params }),
    });
    return response.json();
  },
};

// ============================================================================
// TemplateListPanel - Browse & Select Templates
// ============================================================================

interface TemplateListPanelProps {
  onSelectTemplate: (template: SemanticQueryTemplate) => void;
  datasource?: string;
  version?: string;
}

const TemplateListPanel: React.FC<TemplateListPanelProps> = ({
  onSelectTemplate,
  datasource,
  version,
}) => {
  const [templates, setTemplates] = useState<SemanticQueryTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [filteredDatasource, setFilteredDatasource] = useState(datasource || '');
  const [filteredVersion, setFilteredVersion] = useState(version || '');

  useEffect(() => {
    loadTemplates();
  }, [filteredDatasource, filteredVersion]);

  const loadTemplates = async () => {
    setLoading(true);
    try {
      const result = await templateApi.listTemplates(filteredDatasource, filteredVersion);
      setTemplates(result.templates || []);
    } catch (error) {
      console.error('Failed to load templates:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Paper sx={{ p: 2, height: '100%', overflow: 'auto' }}>
      <Typography variant="h6" sx={{ mb: 2 }}>
        Templates
      </Typography>

      <Box sx={{ mb: 2, display: 'flex', gap: 1 }}>
        <TextField
          size="small"
          placeholder="Datasource"
          value={filteredDatasource}
          onChange={(e) => setFilteredDatasource(e.target.value)}
        />
        <TextField
          size="small"
          placeholder="Version"
          value={filteredVersion}
          onChange={(e) => setFilteredVersion(e.target.value)}
        />
      </Box>

      <List sx={{ maxHeight: 400, overflow: 'auto' }}>
        {templates.map((template) => (
          <ListItem key={template.id} disablePadding>
            <ListItemButton
              onClick={() => onSelectTemplate(template)}
              sx={{
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'flex-start',
              }}
            >
              <Typography variant="subtitle2">{template.name}</Typography>
              <Typography variant="caption" color="textSecondary">
                {template.datasource} v{template.version}
              </Typography>
            </ListItemButton>
          </ListItem>
        ))}
      </List>
    </Paper>
  );
};

// ============================================================================
// ParameterEditor - Collect Parameter Values for Execution
// ============================================================================

interface ParameterEditorProps {
  parameters: TemplateParamDef[];
  values: Record<string, any>;
  onChange: (values: Record<string, any>) => void;
}

const ParameterEditor: React.FC<ParameterEditorProps> = ({
  parameters,
  values,
  onChange,
}) => {
  const handleChangePara = (name: string, value: any) => {
    onChange({ ...values, [name]: value });
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Typography variant="subtitle1">Parameters</Typography>

      {parameters.map((param) => (
        <Box key={param.name} sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          <Typography variant="body2" sx={{ fontWeight: 500 }}>
            {param.name}
            {param.required && <span style={{ color: 'red' }}> *</span>}
          </Typography>

          {param.type === 'string' && (
            <TextField
              size="small"
              placeholder={param.help || `Enter ${param.name}`}
              value={values[param.name] || param.default || ''}
              onChange={(e) => handleChangePara(param.name, e.target.value)}
            />
          )}

          {param.type === 'number' && (
            <TextField
              size="small"
              type="number"
              placeholder={param.help || `Enter ${param.name}`}
              value={values[param.name] || param.default || ''}
              onChange={(e) => handleChangePara(param.name, parseFloat(e.target.value))}
            />
          )}

          {param.type === 'bool' && (
            <FormControlLabel
              control={
                <Checkbox
                  checked={values[param.name] || param.default || false}
                  onChange={(e) => handleChangePara(param.name, e.target.checked)}
                />
              }
              label={param.help || param.name}
            />
          )}
        </Box>
      ))}
    </Box>
  );
};

// ============================================================================
// TemplateEditor - Create & Edit Templates
// ============================================================================

interface TemplateEditorProps {
  template?: SemanticQueryTemplate;
  onSave: (template: Partial<SemanticQueryTemplate>, changeMessage?: string) => void;
}

const TemplateEditor: React.FC<TemplateEditorProps> = ({ template, onSave }) => {
  const [name, setName] = useState(template?.name || '');
  const [description, setDescription] = useState(template?.description || '');
  const [datasource, setDatasource] = useState(template?.datasource || '');
  const [version, setVersion] = useState(template?.version || 'v1');
  const [visibility, setVisibility] = useState(template?.visibility || 'private');
  const [semanticQuery, setSemanticQuery] = useState(
    JSON.stringify(template?.semantic_query || {}, null, 2)
  );
  const [parameters, setParameters] = useState<TemplateParamDef[]>(
    template?.parameters || []
  );
  const [changeMessage, setChangeMessage] = useState('');

  const handleSave = () => {
    const newTemplate: Partial<SemanticQueryTemplate> = {
      name,
      description,
      datasource,
      version,
      visibility,
      semantic_query: JSON.parse(semanticQuery),
      parameters,
    };

    onSave(newTemplate, changeMessage);
  };

  const addParameter = () => {
    setParameters([
      ...parameters,
      {
        name: `param_${Date.now()}`,
        type: 'string',
        required: false,
      },
    ]);
  };

  const updateParameter = (index: number, changes: Partial<TemplateParamDef>) => {
    const updated = [...parameters];
    updated[index] = { ...updated[index], ...changes };
    setParameters(updated);
  };

  const removeParameter = (index: number) => {
    setParameters(parameters.filter((_, i) => i !== index));
  };

  return (
    <Grid container spacing={2} sx={{ p: 2 }}>
      {/* Template Metadata */}
      <Grid item xs={12}>
        <Paper sx={{ p: 2 }}>
          <Typography variant="h6" sx={{ mb: 2 }}>
            Template Details
          </Typography>

          <Grid container spacing={2}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Template Name"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </Grid>

            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Description"
                multiline
                rows={2}
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </Grid>

            <Grid item xs={6}>
              <TextField
                fullWidth
                label="Datasource"
                value={datasource}
                onChange={(e) => setDatasource(e.target.value)}
              />
            </Grid>

            <Grid item xs={6}>
              <TextField
                fullWidth
                label="Version"
                value={version}
                onChange={(e) => setVersion(e.target.value)}
              />
            </Grid>

            <Grid item xs={6}>
              <FormControl fullWidth>
                <InputLabel>Visibility</InputLabel>
                <Select
                  value={visibility}
                  label="Visibility"
                  onChange={(e) => setVisibility(e.target.value)}
                >
                  <MenuItem value="private">Private</MenuItem>
                  <MenuItem value="team">Team</MenuItem>
                  <MenuItem value="public">Public</MenuItem>
                </Select>
              </FormControl>
            </Grid>
          </Grid>
        </Paper>
      </Grid>

      {/* Semantic Query */}
      <Grid item xs={12}>
        <Paper sx={{ p: 2 }}>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Semantic Query (JSON)
          </Typography>
          <Box sx={{ border: '1px solid #ccc', borderRadius: 1, height: 300 }}>
            <Editor
              height="100%"
              defaultLanguage="json"
              value={semanticQuery}
              onChange={(value) => setSemanticQuery(value || '')}
              options={{ minimap: { enabled: false } }}
            />
          </Box>
        </Paper>
      </Grid>

      {/* Parameters */}
      <Grid item xs={12}>
        <Paper sx={{ p: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">Parameters</Typography>
            <Button size="small" onClick={addParameter}>
              + Add Parameter
            </Button>
          </Box>

          <TableContainer>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
                  <TableCell>Name</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>Required</TableCell>
                  <TableCell>Default</TableCell>
                  <TableCell>Help</TableCell>
                  <TableCell>Actions</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {parameters.map((param, idx) => (
                  <TableRow key={idx}>
                    <TableCell>
                      <TextField
                        size="small"
                        value={param.name}
                        onChange={(e) => updateParameter(idx, { name: e.target.value })}
                      />
                    </TableCell>
                    <TableCell>
                      <Select
                        size="small"
                        value={param.type}
                        onChange={(e) => updateParameter(idx, { type: e.target.value as any })}
                      >
                        <MenuItem value="string">string</MenuItem>
                        <MenuItem value="number">number</MenuItem>
                        <MenuItem value="bool">bool</MenuItem>
                      </Select>
                    </TableCell>
                    <TableCell>
                      <Checkbox
                        checked={param.required}
                        onChange={(e) => updateParameter(idx, { required: e.target.checked })}
                      />
                    </TableCell>
                    <TableCell>
                      <TextField
                        size="small"
                        value={param.default || ''}
                        onChange={(e) => updateParameter(idx, { default: e.target.value })}
                      />
                    </TableCell>
                    <TableCell>
                      <TextField
                        size="small"
                        value={param.help || ''}
                        onChange={(e) => updateParameter(idx, { help: e.target.value })}
                      />
                    </TableCell>
                    <TableCell>
                      <IconButton
                        size="small"
                        onClick={() => removeParameter(idx)}
                      >
                        <DeleteIcon fontSize="small" />
                      </IconButton>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        </Paper>
      </Grid>

      {/* Change Message (for versioning) */}
      {template && (
        <Grid item xs={12}>
          <TextField
            fullWidth
            label="Change Message"
            placeholder="Describe what changed in this version"
            value={changeMessage}
            onChange={(e) => setChangeMessage(e.target.value)}
            multiline
            rows={2}
          />
        </Grid>
      )}

      {/* Save Button */}
      <Grid item xs={12}>
        <Button variant="contained" onClick={handleSave} fullWidth>
          {template ? 'Update Template' : 'Create Template'}
        </Button>
      </Grid>
    </Grid>
  );
};

// ============================================================================
// TemplateRunner - Execute Template with Parameters
// ============================================================================

interface TemplateRunnerProps {
  template: SemanticQueryTemplate;
  onClose: () => void;
}

const TemplateRunner: React.FC<TemplateRunnerProps> = ({ template, onClose }) => {
  const [paramValues, setParamValues] = useState<Record<string, any>>({});
  const [loading, setLoading] = useState(false);
  const [result, setResult] = useState<TemplateRunResponse | null>(null);
  const [sqlViewerOpen, setSqlViewerOpen] = useState(false);

  const handleRun = async () => {
    setLoading(true);
    try {
      const response = await templateApi.runTemplate(template.id, paramValues);
      setResult(response);
    } catch (error) {
      console.error('Failed to run template:', error);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog open onClose={onClose} maxWidth="lg" fullWidth>
      <Box sx={{ p: 3 }}>
        <Typography variant="h5" sx={{ mb: 2 }}>
          Run: {template.name}
        </Typography>

        {!result ? (
          <Box>
            <ParameterEditor
              parameters={template.parameters}
              values={paramValues}
              onChange={setParamValues}
            />

            <Box sx={{ mt: 3, display: 'flex', gap: 1 }}>
              <Button variant="contained" onClick={handleRun} disabled={loading}>
                {loading ? 'Executing...' : 'Execute'}
              </Button>
              <Button variant="outlined" onClick={onClose}>
                Cancel
              </Button>
            </Box>
          </Box>
        ) : (
          <Box>
            <Alert severity="success" sx={{ mb: 2 }}>
              Query executed successfully in {result.duration_ms}ms - {result.count} rows
            </Alert>

            <Box sx={{ mb: 2 }}>
              <Button
                variant="outlined"
                size="small"
                onClick={() => setSqlViewerOpen(!sqlViewerOpen)}
              >
                {sqlViewerOpen ? 'Hide' : 'Show'} SQL
              </Button>
            </Box>

            {sqlViewerOpen && (
              <Paper sx={{ p: 2, mb: 2, backgroundColor: '#f5f5f5' }}>
                <Typography variant="body2" sx={{ fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>
                  {result.sql}
                </Typography>
              </Paper>
            )}

            <Typography variant="subtitle2" sx={{ mb: 1 }}>
              Results ({result.count} rows)
            </Typography>

            <TableContainer sx={{ maxHeight: 400 }}>
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
                    {result.rows.length > 0 &&
                      Object.keys(result.rows[0]).map((col) => (
                        <TableCell key={col}>{col}</TableCell>
                      ))}
                  </TableRow>
                </TableHead>
                <TableBody>
                  {result.rows.map((row, idx) => (
                    <TableRow key={idx}>
                      {Object.values(row).map((cell, cidx) => (
                        <TableCell key={cidx}>{String(cell)}</TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        )}
      </Box>
    </Dialog>
  );
};

// ============================================================================
// TemplatesTab - Main Template Management Component
// ============================================================================

export const TemplatesTab: React.FC = () => {
  const [mode, setMode] = useState<'list' | 'edit' | 'create'>('list');
  const [selectedTemplate, setSelectedTemplate] = useState<SemanticQueryTemplate | null>(null);
  const [runnerTemplate, setRunnerTemplate] = useState<SemanticQueryTemplate | null>(null);
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  });

  const handleCreateClick = () => {
    setSelectedTemplate(null);
    setMode('create');
  };

  const handleEditClick = (template: SemanticQueryTemplate) => {
    setSelectedTemplate(template);
    setMode('edit');
  };

  const handleSaveTemplate = async (template: Partial<SemanticQueryTemplate>, changeMessage?: string) => {
    try {
      if (selectedTemplate) {
        await templateApi.updateTemplate(selectedTemplate.id, template, changeMessage);
        setSnackbar({ open: true, message: 'Template updated successfully', severity: 'success' });
      } else {
        await templateApi.createTemplate(template);
        setSnackbar({ open: true, message: 'Template created successfully', severity: 'success' });
      }

      setMode('list');
      setSelectedTemplate(null);
    } catch (error) {
      setSnackbar({ open: true, message: 'Error saving template', severity: 'error' });
    }
  };

  return (
    <Box sx={{ display: 'flex', height: '100%', flexDirection: 'column' }}>
      {/* Header */}
      <Box sx={{ p: 2, borderBottom: '1px solid #e0e0e0' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Typography variant="h5">Semantic Query Templates</Typography>

          {mode === 'list' && (
            <Button variant="contained" onClick={handleCreateClick}>
              + New Template
            </Button>
          )}

          {mode !== 'list' && (
            <Box>
              <Button
                variant="outlined"
                onClick={() => {
                  setMode('list');
                  setSelectedTemplate(null);
                }}
              >
                Back
              </Button>
            </Box>
          )}
        </Box>
      </Box>

      {/* Content */}
      <Box sx={{ flex: 1, overflow: 'auto' }}>
        {mode === 'list' && (
          <Grid container spacing={2} sx={{ p: 2, height: '100%' }}>
            <Grid item xs={4}>
              <TemplateListPanel onSelectTemplate={(t) => handleEditClick(t)} />
            </Grid>

            <Grid item xs={8}>
              {selectedTemplate && (
                <Paper sx={{ p: 2 }}>
                  <Typography variant="h6">{selectedTemplate.name}</Typography>
                  <Typography variant="body2" color="textSecondary">
                    {selectedTemplate.description}
                  </Typography>

                  <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
                    <Button
                      variant="contained"
                      startIcon={<PlayArrowIcon />}
                      onClick={() => setRunnerTemplate(selectedTemplate)}
                    >
                      Run
                    </Button>

                    <Button
                      variant="outlined"
                      startIcon={<EditIcon />}
                      onClick={() => handleEditClick(selectedTemplate)}
                    >
                      Edit
                    </Button>

                    <Button
                      variant="outlined"
                      color="error"
                      startIcon={<DeleteIcon />}
                      onClick={async () => {
                        await templateApi.deleteTemplate(selectedTemplate.id);
                        setSelectedTemplate(null);
                        setSnackbar({ open: true, message: 'Template deleted', severity: 'success' });
                      }}
                    >
                      Delete
                    </Button>
                  </Box>
                </Paper>
              )}
            </Grid>
          </Grid>
        )}

        {(mode === 'edit' || mode === 'create') && (
          <TemplateEditor
            template={selectedTemplate || undefined}
            onSave={handleSaveTemplate}
          />
        )}
      </Box>

      {/* Runner Dialog */}
      {runnerTemplate && (
        <TemplateRunner
          template={runnerTemplate}
          onClose={() => setRunnerTemplate(null)}
        />
      )}

      {/* Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={3000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert severity={snackbar.severity}>{snackbar.message}</Alert>
      </Snackbar>
    </Box>
  );
};

export default TemplatesTab;
