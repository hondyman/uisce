/**
 * Report Viewer Component
 * 
 * Displays a rendered report with parameter input,
 * format selection, and export options.
 */

import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  IconButton,
  CircularProgress,
  Alert,
  Drawer,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Divider,
  Toolbar,
  Tooltip,
  Chip,
  FormControlLabel,
  Switch,
} from '@mui/material';
import {
  Download,
  Print,
  Refresh,
  ZoomIn,
  ZoomOut,
  Close,
  ChevronRight,
} from '@mui/icons-material';
import {
  useReportDefinition,
  useReportInstance,
  useRenderReport,
  useDownloadReport,
} from '../../hooks/useSemanticReporting';
import { Parameter, RenderReportRequest } from '../../api/semanticReporting';

interface ReportViewerProps {
  reportId: string;
  extensionId?: string;
  initialParameters?: Record<string, any>;
  onClose?: () => void;
}

const ReportViewer: React.FC<ReportViewerProps> = ({
  reportId,
  extensionId,
  initialParameters = {},
  onClose,
}) => {
  const [parameters, setParameters] = useState<Record<string, any>>(initialParameters);
  const [outputFormat, setOutputFormat] = useState<'pdf' | 'html' | 'excel'>('html');
  const [parametersOpen, setParametersOpen] = useState(true);
  const [zoom, setZoom] = useState(100);
  const [currentInstanceId, setCurrentInstanceId] = useState<string | null>(null);

  // Fetch report definition
  const { data: definition, isLoading: loadingDef } = useReportDefinition(reportId);

  // Fetch current instance if we have one
  const { data: instance } = useReportInstance(
    currentInstanceId || undefined
  );

  const renderMutation = useRenderReport();
  const downloadMutation = useDownloadReport();

  // Initialize parameters from definition defaults
  useEffect(() => {
    if (definition?.parameters_schema) {
      const defaults: Record<string, any> = {};
      definition.parameters_schema.forEach((param: Parameter) => {
        if (param.default_value !== undefined) {
          defaults[param.name] = param.default_value;
        }
      });
      setParameters({ ...defaults, ...initialParameters });
    }
  }, [definition, initialParameters]);

  const handleRun = async () => {
    if (!definition) return;

    const request: RenderReportRequest = {
      report_definition_id: reportId,
      report_extension_id: extensionId,
      output_format: outputFormat,
      parameters,
    };

    try {
      const result = await renderMutation.mutateAsync(request);
      setCurrentInstanceId(result.id);
    } catch (err) {
      console.error('Failed to run report:', err);
    }
  };

  const handleDownload = () => {
    if (currentInstanceId) {
      downloadMutation.mutate(currentInstanceId);
    }
  };

  const handlePrint = () => {
    window.print();
  };

  const updateParameter = (name: string, value: any) => {
    setParameters(prev => ({ ...prev, [name]: value }));
  };

  const renderParameterInput = (param: Parameter) => {
    const value = parameters[param.name] ?? param.default_value ?? '';

    switch (param.type) {
      case 'string':
        return (
          <TextField
            key={param.name}
            label={param.label}
            value={value}
            onChange={(e) => updateParameter(param.name, e.target.value)}
            fullWidth
            size="small"
            required={param.required}
            helperText={param.description}
            sx={{ mb: 2 }}
          />
        );

      case 'number':
        return (
          <TextField
            key={param.name}
            label={param.label}
            type="number"
            value={value}
            onChange={(e) => updateParameter(param.name, Number(e.target.value))}
            fullWidth
            size="small"
            required={param.required}
            helperText={param.description}
            inputProps={{
              min: param.validation?.min,
              max: param.validation?.max,
            }}
            sx={{ mb: 2 }}
          />
        );

      case 'date':
        return (
          <TextField
            key={param.name}
            label={param.label}
            type="date"
            value={value}
            onChange={(e) => updateParameter(param.name, e.target.value)}
            fullWidth
            size="small"
            required={param.required}
            helperText={param.description}
            InputLabelProps={{ shrink: true }}
            sx={{ mb: 2 }}
          />
        );

      case 'boolean':
        return (
          <FormControlLabel
            key={param.name}
            control={
              <Switch
                checked={Boolean(value)}
                onChange={(e) => updateParameter(param.name, e.target.checked)}
              />
            }
            label={param.label}
            sx={{ mb: 2 }}
          />
        );

      case 'select':
        return (
          <FormControl key={param.name} fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>{param.label}</InputLabel>
            <Select
              value={value}
              label={param.label}
              onChange={(e) => updateParameter(param.name, e.target.value)}
              required={param.required}
            >
              {param.options?.map((opt) => (
                <MenuItem key={opt.value} value={opt.value}>
                  {opt.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        );

      case 'multiselect':
        return (
          <FormControl key={param.name} fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>{param.label}</InputLabel>
            <Select
              multiple
              value={Array.isArray(value) ? value : []}
              label={param.label}
              onChange={(e) => updateParameter(param.name, e.target.value)}
              required={param.required}
              renderValue={(selected) => (
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {(selected as string[]).map((val) => (
                    <Chip 
                      key={val} 
                      label={param.options?.find(o => o.value === val)?.label || val} 
                      size="small" 
                    />
                  ))}
                </Box>
              )}
            >
              {param.options?.map((opt) => (
                <MenuItem key={opt.value} value={opt.value}>
                  {opt.label}
                </MenuItem>
              ))}
            </Select>
          </FormControl>
        );

      default:
        return null;
    }
  };

  if (loadingDef) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
        <CircularProgress />
      </Box>
    );
  }

  if (!definition) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="error">Report definition not found</Alert>
      </Box>
    );
  }

  const isGenerating = instance?.status === 'pending' || instance?.status === 'generating';
  const isComplete = instance?.status === 'completed';
  const isFailed = instance?.status === 'failed';

  return (
    <Box sx={{ display: 'flex', height: '100%' }}>
      {/* Parameters Drawer */}
      <Drawer
        variant="persistent"
        anchor="left"
        open={parametersOpen}
        sx={{
          width: 300,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: 300,
            position: 'relative',
          },
        }}
      >
        <Box sx={{ p: 2 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
            <Typography variant="h6">Parameters</Typography>
            <IconButton size="small" onClick={() => setParametersOpen(false)}>
              <Close />
            </IconButton>
          </Box>

          {definition.parameters_schema?.map(renderParameterInput)}

          <Divider sx={{ my: 2 }} />

          <FormControl fullWidth size="small" sx={{ mb: 2 }}>
            <InputLabel>Output Format</InputLabel>
            <Select
              value={outputFormat}
              label="Output Format"
              onChange={(e) => setOutputFormat(e.target.value as 'pdf' | 'html' | 'excel')}
            >
              {definition.output_formats.map((format) => (
                <MenuItem key={format} value={format}>
                  {format.toUpperCase()}
                </MenuItem>
              ))}
            </Select>
          </FormControl>

          <Button
            variant="contained"
            fullWidth
            onClick={handleRun}
            disabled={renderMutation.isPending || isGenerating}
            startIcon={renderMutation.isPending || isGenerating ? <CircularProgress size={20} /> : <Refresh />}
          >
            {renderMutation.isPending || isGenerating ? 'Generating...' : 'Run Report'}
          </Button>
        </Box>
      </Drawer>

      {/* Main Content */}
      <Box sx={{ flexGrow: 1, display: 'flex', flexDirection: 'column' }}>
        {/* Toolbar */}
        <Toolbar variant="dense" sx={{ borderBottom: 1, borderColor: 'divider' }}>
          {!parametersOpen && (
            <IconButton onClick={() => setParametersOpen(true)} sx={{ mr: 1 }}>
              <ChevronRight />
            </IconButton>
          )}
          
          <Typography variant="subtitle1" sx={{ flexGrow: 1 }}>
            {definition.display_name}
            {extensionId && <Chip label="Extended" size="small" sx={{ ml: 1 }} />}
          </Typography>

          <Box sx={{ display: 'flex', gap: 1 }}>
            <Tooltip title="Zoom Out">
              <IconButton size="small" onClick={() => setZoom(z => Math.max(50, z - 10))}>
                <ZoomOut />
              </IconButton>
            </Tooltip>
            <Typography sx={{ minWidth: 40, textAlign: 'center', lineHeight: '32px' }}>
              {zoom}%
            </Typography>
            <Tooltip title="Zoom In">
              <IconButton size="small" onClick={() => setZoom(z => Math.min(200, z + 10))}>
                <ZoomIn />
              </IconButton>
            </Tooltip>
            
            <Divider orientation="vertical" flexItem sx={{ mx: 1 }} />
            
            <Tooltip title="Download">
              <IconButton 
                size="small" 
                onClick={handleDownload}
                disabled={!isComplete}
              >
                <Download />
              </IconButton>
            </Tooltip>
            <Tooltip title="Print">
              <IconButton 
                size="small" 
                onClick={handlePrint}
                disabled={!isComplete}
              >
                <Print />
              </IconButton>
            </Tooltip>

            {onClose && (
              <>
                <Divider orientation="vertical" flexItem sx={{ mx: 1 }} />
                <IconButton size="small" onClick={onClose}>
                  <Close />
                </IconButton>
              </>
            )}
          </Box>
        </Toolbar>

        {/* Report Content */}
        <Box 
          sx={{ 
            flexGrow: 1, 
            overflow: 'auto', 
            bgcolor: 'grey.100',
            p: 2,
            display: 'flex',
            justifyContent: 'center',
          }}
        >
          {!currentInstanceId && (
            <Box sx={{ textAlign: 'center', py: 8 }}>
              <Typography color="text.secondary">
                Configure parameters and click "Run Report" to generate
              </Typography>
            </Box>
          )}

          {isGenerating && (
            <Box sx={{ textAlign: 'center', py: 8 }}>
              <CircularProgress sx={{ mb: 2 }} />
              <Typography>Generating report...</Typography>
            </Box>
          )}

          {isFailed && (
            <Alert severity="error" sx={{ maxWidth: 500 }}>
              <Typography variant="subtitle1">Report generation failed</Typography>
              <Typography variant="body2">
                {instance?.error_message || 'An unexpected error occurred'}
              </Typography>
            </Alert>
          )}

          {isComplete && instance?.output_url && (
            <Paper
              elevation={3}
              sx={{
                width: `${8.5 * zoom}px`,
                minHeight: `${11 * zoom}px`,
                transform: `scale(${zoom / 100})`,
                transformOrigin: 'top center',
                bgcolor: 'white',
                overflow: 'hidden',
              }}
            >
              {outputFormat === 'html' ? (
                <Box
                  component="iframe"
                  src={instance.output_url}
                  sx={{ width: '100%', height: '100%', border: 'none' }}
                  title="Report Content"
                />
              ) : (
                <Box sx={{ p: 3, textAlign: 'center' }}>
                  <Typography>
                    Report generated successfully.
                  </Typography>
                  <Button
                    variant="contained"
                    startIcon={<Download />}
                    onClick={handleDownload}
                    sx={{ mt: 2 }}
                  >
                    Download {outputFormat.toUpperCase()}
                  </Button>
                </Box>
              )}
            </Paper>
          )}
        </Box>

        {/* Status Bar */}
        {instance && (
          <Box sx={{ px: 2, py: 1, borderTop: 1, borderColor: 'divider', bgcolor: 'grey.50' }}>
            <Typography variant="caption" color="text.secondary">
              Status: {instance.status}
              {instance.generation_time_ms && ` • Generated in ${instance.generation_time_ms}ms`}
              {instance.completed_at && ` • ${new Date(instance.completed_at).toLocaleString()}`}
            </Typography>
          </Box>
        )}
      </Box>
    </Box>
  );
};

export default ReportViewer;
