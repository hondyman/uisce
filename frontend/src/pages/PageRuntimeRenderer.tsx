import React, { useState, useEffect, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Grid,
  Card,
  CardContent,
  CircularProgress,
  Alert,
  Divider,
  IconButton,
  Collapse,
  Stack,
  FormControlLabel,
  Switch,
  MenuItem,
} from '@mui/material';
import {
  Save as SaveIcon,
  ArrowBack as BackIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
} from '@mui/icons-material';
import { useNotification } from '../hooks/useNotification';

// Types
interface FieldConfig {
  id: string;
  name: string;
  label: string;
  type: 'text' | 'number' | 'date' | 'boolean' | 'email' | 'currency' | 'reference';
  required?: boolean;
  helpText?: string;
}

interface SectionConfig {
  id: string;
  title: string;
  collapsible?: boolean;
  columns: 1 | 2 | 3 | 4;
  fieldIds: string[];
}

interface PageLayoutConfig {
  id: string;
  name: string;
  primaryBO: string;
  layoutType: 'form' | 'list' | 'detail';
  sections: SectionConfig[];
  fields: FieldConfig[];
  pipelineId?: string;
}

interface PageRuntimeRendererProps {
  mode?: 'create' | 'edit' | 'view';
}

const API_BASE = '/api';

export const PageRuntimeRenderer: React.FC<PageRuntimeRendererProps> = ({ mode: propMode }) => {
  const { pageId, instanceId } = useParams<{ pageId: string; instanceId?: string }>();
  const navigate = useNavigate();
  const notification = useNotification();


  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [layout, setLayout] = useState<PageLayoutConfig | null>(null);
  const [formData, setFormData] = useState<Record<string, any>>({});
  const [collapsedSections, setCollapsedSections] = useState<Set<string>>(new Set());

  const mode = propMode || (instanceId ? 'edit' : 'create');

  // Fetch layout and instance data
  useEffect(() => {
    const fetchData = async () => {
      setLoading(true);
      setError(null);

      try {
        // Fetch layout
        const layoutRes = await fetch(`${API_BASE}/v1/page-layouts/${pageId}`);
        if (!layoutRes.ok) throw new Error('Failed to load page layout');
        const layoutData = await layoutRes.json();

        // Parse layout_json
        const config: PageLayoutConfig = {
          id: layoutData.id,
          name: layoutData.name,
          primaryBO: layoutData.primary_bo,
          layoutType: layoutData.layout_type,
          pipelineId: layoutData.pipeline_id,
          ...layoutData.layout_json,
        };
        setLayout(config);

        // Fetch instance if editing
        if (instanceId) {
          const instanceRes = await fetch(`${API_BASE}/business-objects/${config.primaryBO}/instances/${instanceId}`);
          if (instanceRes.ok) {
            const instanceData = await instanceRes.json();
            setFormData({
              ...instanceData.coreFieldValues,
              ...instanceData.customFieldValues,
            });
          }
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setLoading(false);
      }
    };

    if (pageId) {
      fetchData();
    }
  }, [pageId, instanceId]);

  // Handle field change
  const handleChange = useCallback((fieldId: string, value: any) => {
    setFormData(prev => ({ ...prev, [fieldId]: value }));
  }, []);

  // Toggle section collapse
  const toggleSection = (sectionId: string) => {
    setCollapsedSections(prev => {
      const next = new Set(prev);
      if (next.has(sectionId)) {
        next.delete(sectionId);
      } else {
        next.add(sectionId);
      }
      return next;
    });
  };

  // Save form
  const handleSave = async () => {
    if (!layout) return;

    setSaving(true);
    try {
      // Run pipeline if configured
      if (layout.pipelineId) {
        const pipelineRes = await fetch(`${API_BASE}/v1/pipelines/${layout.pipelineId}/execute`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ formData }),
        });

        if (!pipelineRes.ok) {
          const err = await pipelineRes.json();
          // Handling OPA Policy Violations
          if (pipelineRes.status === 403 && err.message?.includes('Policy Violation')) {
             throw new Error(`Governance Blocked: ${err.message}`);
          }
          throw new Error(err.message || 'Pipeline validation failed');
        }
      }

      // Save instance
      const url = instanceId
        ? `${API_BASE}/business-objects/${layout.primaryBO}/instances/${instanceId}`
        : `${API_BASE}/business-objects/${layout.primaryBO}/instances`;

      const res = await fetch(url, {
        method: instanceId ? 'PUT' : 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          coreFieldValues: formData,
          customFieldValues: {},
        }),
      });

      if (!res.ok) throw new Error('Failed to save');

      notification.success(instanceId ? 'Updated successfully' : 'Created successfully');
      navigate(-1);
    } catch (err) {
      notification.error(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  };

  // Delete instance
  const handleDelete = async () => {
    if (!layout || !instanceId) return;
    if (!window.confirm('Are you sure you want to delete this record?')) return;

    setSaving(true);
    try {
      const res = await fetch(`${API_BASE}/business-objects/${layout.primaryBO}/instances/${instanceId}`, {
        method: 'DELETE',
      });
      if (!res.ok) throw new Error('Failed to delete');

      notification.success('Deleted successfully');
      navigate(-1);
    } catch (err) {
      notification.error(err instanceof Error ? err.message : 'Delete failed');
    } finally {
      setSaving(false);
    }
  };

  // Render field input
  const renderField = (field: FieldConfig) => {
    const value = formData[field.id] ?? '';

    switch (field.type) {
      case 'boolean':
        return (
          <FormControlLabel
            control={
              <Switch
                checked={!!value}
                onChange={e => handleChange(field.id, e.target.checked)}
                disabled={mode === 'view'}
              />
            }
            label={field.label}
          />
        );

      case 'date':
        return (
          <TextField
            fullWidth
            size="small"
            type="date"
            label={field.label}
            value={value}
            onChange={e => handleChange(field.id, e.target.value)}
            required={field.required}
            helperText={field.helpText}
            InputLabelProps={{ shrink: true }}
            disabled={mode === 'view'}
          />
        );

      case 'number':
      case 'currency':
        return (
          <TextField
            fullWidth
            size="small"
            type="number"
            label={field.label}
            value={value}
            onChange={e => handleChange(field.id, parseFloat(e.target.value) || 0)}
            required={field.required}
            helperText={field.helpText}
            disabled={mode === 'view'}
            InputProps={field.type === 'currency' ? { startAdornment: '$' } : undefined}
          />
        );

      case 'email':
        return (
          <TextField
            fullWidth
            size="small"
            type="email"
            label={field.label}
            value={value}
            onChange={e => handleChange(field.id, e.target.value)}
            required={field.required}
            helperText={field.helpText}
            disabled={mode === 'view'}
          />
        );

      default:
        return (
          <TextField
            fullWidth
            size="small"
            label={field.label}
            value={value}
            onChange={e => handleChange(field.id, e.target.value)}
            required={field.required}
            helperText={field.helpText}
            disabled={mode === 'view'}
          />
        );
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight="400px">
        <CircularProgress />
      </Box>
    );
  }

  if (error || !layout) {
    return (
      <Box p={3}>
        <Alert severity="error">{error || 'Page layout not found'}</Alert>
        <Button startIcon={<BackIcon />} onClick={() => navigate(-1)} sx={{ mt: 2 }}>
          Go Back
        </Button>
      </Box>
    );
  }

  const fieldsMap = new Map(layout.fields?.map(f => [f.id, f]) || []);

  return (
    <Box p={3} maxWidth="1200px" mx="auto">
      {/* Header */}
      <Paper sx={{ p: 2, mb: 3 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Stack direction="row" alignItems="center" spacing={2}>
            <IconButton onClick={() => navigate(-1)}>
              <BackIcon />
            </IconButton>
            <Typography variant="h5">
              {mode === 'create' ? `New ${layout.name}` : mode === 'edit' ? `Edit ${layout.name}` : layout.name}
            </Typography>
          </Stack>
          <Stack direction="row" spacing={1}>
            {mode === 'edit' && (
              <Button
                color="error"
                startIcon={<DeleteIcon />}
                onClick={handleDelete}
                disabled={saving}
              >
                Delete
              </Button>
            )}
            {mode !== 'view' && (
              <Button
                variant="contained"
                startIcon={<SaveIcon />}
                onClick={handleSave}
                disabled={saving}
              >
                {saving ? 'Saving...' : 'Save'}
              </Button>
            )}
          </Stack>
        </Stack>
      </Paper>

      {/* Sections */}
      {layout.sections?.map(section => {
        const isCollapsed = collapsedSections.has(section.id);

        return (
          <Card key={section.id} sx={{ mb: 2 }}>
            <CardContent sx={{ pb: section.collapsible && isCollapsed ? '16px !important' : undefined }}>
              <Stack
                direction="row"
                justifyContent="space-between"
                alignItems="center"
                onClick={() => section.collapsible && toggleSection(section.id)}
                sx={{ cursor: section.collapsible ? 'pointer' : 'default' }}
              >
                <Typography variant="h6">{section.title}</Typography>
                {section.collapsible && (
                  <IconButton size="small">
                    {isCollapsed ? <ExpandIcon /> : <CollapseIcon />}
                  </IconButton>
                )}
              </Stack>

              <Collapse in={!isCollapsed}>
                <Divider sx={{ my: 2 }} />
                <Grid container spacing={2}>
                  {section.fieldIds?.map(fieldId => {
                    const field = fieldsMap.get(fieldId);
                    if (!field) return null;

                    return (
                      <Grid item xs={12} sm={12 / section.columns} key={fieldId}>
                        {renderField(field)}
                      </Grid>
                    );
                  })}
                </Grid>
              </Collapse>
            </CardContent>
          </Card>
        );
      })}
    </Box>
  );
};

export default PageRuntimeRenderer;
