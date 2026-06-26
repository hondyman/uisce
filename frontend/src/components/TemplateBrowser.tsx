import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardActions,
  Button,
  Typography,
  Grid,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  CircularProgress,
  Alert,
  Chip,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
} from '@mui/material';
import TemplateOutlinedIcon from '@mui/icons-material/TemplateOutlined';
import AddIcon from '@mui/icons-material/Add';

interface Template {
  id: string;
  name: string;
  description: string;
  category: string;
  businessObject: string;
  status: 'draft' | 'approved' | 'deprecated';
  usageCount: number;
  parameterSchema: any;
}

interface TemplatePreview {
  template: Template;
  sampleParameters: Record<string, any>;
  previewSteps: any[];
}

/**
 * TemplateBrowser Component
 *
 * Allows users to:
 * 1. Browse available rule templates
 * 2. Filter by business object and category
 * 3. Preview templates with sample parameters
 * 4. Instantiate rules from templates with custom parameters
 *
 * Features:
 * - Template discovery by category
 * - Parameter configuration UI (auto-generated from JSON schema)
 * - Rule preview before creation
 * - Usage statistics per template
 */
export const TemplateBrowser: React.FC<{
  businessObject: string;
  onRuleCreated?: (ruleId: string) => void;
}> = ({ businessObject, onRuleCreated }) => {
  const [templates, setTemplates] = useState<Template[]>([]);
  const [selectedTemplate, setSelectedTemplate] = useState<Template | null>(null);
  const [preview, setPreview] = useState<TemplatePreview | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Instantiation dialog state
  const [openInstantiate, setOpenInstantiate] = useState(false);
  const [ruleName, setRuleName] = useState('');
  const [parameters, setParameters] = useState<Record<string, any>>({});
  const [instantiating, setInstantiating] = useState(false);

  // Filter state
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [categories, setCategories] = useState<string[]>([]);

  // Load templates on mount
  useEffect(() => {
    loadTemplates();
  }, [businessObject]);

  const loadTemplates = async () => {
    try {
      setLoading(true);
      setError(null);

      const response = await fetch(
        `/api/v1/templates?businessObject=${businessObject}&status=approved`,
        {
          headers: {
            'X-Tenant-ID': localStorage.getItem('tenantId') || '00000000-0000-0000-0000-000000000001',
          },
        }
      );

      if (!response.ok) {
        throw new Error('Failed to load templates');
      }

      const data = await response.json();
      setTemplates(data || []);

      // Extract unique categories
      const cats = [...new Set((data || []).map((t: Template) => t.category))];
      setCategories(cats);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load templates';
      setError(message);
      console.error('Error loading templates:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectTemplate = async (template: Template) => {
    setSelectedTemplate(template);
    setRuleName(`${template.name} - ${new Date().toLocaleDateString()}`);
    setParameters({}); // Reset parameters

    try {
      const response = await fetch(`/api/v1/templates/${template.id}/preview`, {
        headers: {
          'X-Tenant-ID': localStorage.getItem('tenantId') || '00000000-0000-0000-0000-000000000001',
        },
      });

      if (response.ok) {
        const data = await response.json();
        setPreview(data);
        setParameters(data.sampleParameters || {});
      }
    } catch (err) {
      console.error('Error loading preview:', err);
    }
  };

  const handleParameterChange = (paramName: string, value: any) => {
    setParameters((prev) => ({
      ...prev,
      [paramName]: value,
    }));
  };

  const handleInstantiate = async () => {
    if (!selectedTemplate || !ruleName) {
      setError('Rule name is required');
      return;
    }

    try {
      setInstantiating(true);
      setError(null);

      const response = await fetch(`/api/v1/templates/${selectedTemplate.id}/create-rule`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': localStorage.getItem('tenantId') || '00000000-0000-0000-0000-000000000001',
          'X-User-ID': localStorage.getItem('userId') || 'user-001',
        },
        body: JSON.stringify({
          ruleName,
          parameters,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to create rule from template');
      }

      const newRule = await response.json();

      // Success feedback
      setOpenInstantiate(false);
      setRuleName('');
      setParameters({});
      setSelectedTemplate(null);
      setPreview(null);

      if (onRuleCreated) {
        onRuleCreated(newRule.id);
      }

      // Show success message
      alert(`Rule "${ruleName}" created successfully!`);
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to create rule';
      setError(message);
    } finally {
      setInstantiating(false);
    }
  };

  // Render parameter input based on schema
  const renderParameterInput = (paramName: string, paramDef: any) => {
    const value = parameters[paramName];
    const label = paramDef.description || paramName;

    if (paramDef.type === 'number') {
      return (
        <TextField
          key={paramName}
          type="number"
          label={label}
          value={value || paramDef.default || ''}
          onChange={(e) => handleParameterChange(paramName, parseFloat(e.target.value))}
          inputProps={{
            min: paramDef.minimum,
            max: paramDef.maximum,
          }}
          fullWidth
          margin="normal"
          helperText={paramDef.description}
        />
      );
    }

    return (
      <TextField
        key={paramName}
        label={label}
        value={value || paramDef.default || ''}
        onChange={(e) => handleParameterChange(paramName, e.target.value)}
        fullWidth
        margin="normal"
        multiline={paramDef.type === 'text'}
        rows={paramDef.type === 'text' ? 2 : 1}
        placeholder={paramDef.example || label}
        helperText={paramDef.description}
      />
    );
  };

  const filteredTemplates =
    selectedCategory === ''
      ? templates
      : templates.filter((t) => t.category === selectedCategory);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={400}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box sx={{ p: 2 }}>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6}>
          <FormControl fullWidth>
            <InputLabel>Filter by Category</InputLabel>
            <Select
              value={selectedCategory}
              label="Filter by Category"
              onChange={(e) => setSelectedCategory(e.target.value)}
            >
              <MenuItem value="">All Categories</MenuItem>
              {categories.map((cat) => (
                <MenuItem key={cat} value={cat}>
                  {cat}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        </Grid>
        <Grid item xs={12} sm={6}>
          <Typography variant="subtitle2">
            Templates Available: {filteredTemplates.length}
          </Typography>
        </Grid>
      </Grid>

      {filteredTemplates.length === 0 ? (
        <Alert severity="info">
          No templates available for "{businessObject}". Create a new rule or contact your administrator.
        </Alert>
      ) : (
        <Grid container spacing={2}>
          {/* Template List */}
          <Grid item xs={12} md={6}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              <TemplateOutlinedIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
              Available Templates
            </Typography>

            <List sx={{ maxHeight: 500, overflow: 'auto' }}>
              {filteredTemplates.map((template) => (
                <ListItemButton
                  key={template.id}
                  selected={selectedTemplate?.id === template.id}
                  onClick={() => handleSelectTemplate(template)}
                  sx={{
                    mb: 1,
                    border: '1px solid #e0e0e0',
                    borderRadius: 1,
                    p: 1,
                  }}
                >
                  <ListItemText
                    primary={template.name}
                    secondary={
                      <>
                        <Typography variant="caption" display="block">
                          {template.description}
                        </Typography>
                        <Box sx={{ mt: 0.5 }}>
                          <Chip
                            label={template.category}
                            size="small"
                            variant="outlined"
                            sx={{ mr: 1 }}
                          />
                          <Chip
                            label={`Used ${template.usageCount} times`}
                            size="small"
                            variant="filled"
                            color={template.usageCount > 0 ? 'success' : 'default'}
                          />
                        </Box>
                      </>
                    }
                  />
                </ListItemButton>
              ))}
            </List>
          </Grid>

          {/* Template Preview & Instantiation */}
          <Grid item xs={12} md={6}>
            {preview ? (
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>
                    {preview.template.name}
                  </Typography>
                  <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                    {preview.template.description}
                  </Typography>

                  <Typography variant="subtitle2" sx={{ mt: 2, mb: 1 }}>
                    Configure Parameters:
                  </Typography>

                  {preview.template.parameterSchema.properties ? (
                    Object.entries(preview.template.parameterSchema.properties).map(
                      ([paramName, paramDef]: any) =>
                        renderParameterInput(paramName, paramDef)
                    )
                  ) : (
                    <Typography variant="body2" color="textSecondary">
                      No parameters required for this template.
                    </Typography>
                  )}

                  <Typography variant="subtitle2" sx={{ mt: 3, mb: 1 }}>
                    New Rule Name:
                  </Typography>
                  <TextField
                    label="Rule Name"
                    value={ruleName}
                    onChange={(e) => setRuleName(e.target.value)}
                    fullWidth
                    placeholder="e.g., US Weekend Override Rule"
                  />

                  <Typography variant="subtitle2" sx={{ mt: 3, mb: 1 }}>
                    Preview (Sample Steps):
                  </Typography>
                  <Box
                    sx={{
                      bgcolor: '#f5f5f5',
                      p: 2,
                      borderRadius: 1,
                      maxHeight: 200,
                      overflow: 'auto',
                    }}
                  >
                    <pre style={{ fontSize: '0.75rem', margin: 0 }}>
                      {JSON.stringify(preview.previewSteps, null, 2)}
                    </pre>
                  </Box>
                </CardContent>

                <CardActions>
                  <Button
                    variant="contained"
                    color="primary"
                    startIcon={<AddIcon />}
                    onClick={() => setOpenInstantiate(true)}
                    disabled={!ruleName || instantiating}
                  >
                    {instantiating ? 'Creating...' : 'Create Rule from Template'}
                  </Button>
                </CardActions>
              </Card>
            ) : (
              <Alert severity="info">
                Select a template from the list to see preview and configure parameters.
              </Alert>
            )}
          </Grid>
        </Grid>
      )}

      {/* Confirmation Dialog */}
      <Dialog open={openInstantiate} onClose={() => !instantiating && setOpenInstantiate(false)}>
        <DialogTitle>Create Rule from Template</DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ mb: 2 }}>
            Create a new rule "{ruleName}" from the template "{selectedTemplate?.name}"?
          </Typography>
          <Box sx={{ bgcolor: '#f5f5f5', p: 1, borderRadius: 1 }}>
            <Typography variant="caption">
              <strong>Business Object:</strong> {selectedTemplate?.businessObject}
            </Typography>
            <br />
            <Typography variant="caption">
              <strong>Template:</strong> {selectedTemplate?.name}
            </Typography>
            <br />
            <Typography variant="caption">
              <strong>Parameters:</strong> {Object.keys(parameters).length} configured
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenInstantiate(false)} disabled={instantiating}>
            Cancel
          </Button>
          <Button
            onClick={handleInstantiate}
            variant="contained"
            color="primary"
            disabled={instantiating}
          >
            {instantiating ? 'Creating...' : 'Confirm'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default TemplateBrowser;
