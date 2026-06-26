import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  FormControlLabel,
  Switch,
  Chip,
  Button,
  Alert,
  Divider,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Tabs,
  Tab,
  Card,
  CardContent,
} from '@mui/material';
import {
  ExpandMore as ExpandMoreIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  ContentCopy as CopyIcon,
  PlayArrow as PreviewIcon,
  Save as SaveIcon,
  Schedule as ScheduleIcon,
  Storage as StorageIcon,
  Speed as SpeedIcon,
  Build as BuildIcon,
} from '@mui/icons-material';
import yaml from 'js-yaml';

interface PreAggConfig {
  name: string;
  cubeName: string;
  measures: string[];
  dimensions: string[];
  timeDimension: string;
  granularity: 'second' | 'minute' | 'hour' | 'day' | 'week' | 'month' | 'quarter' | 'year';
  partitionGranularity: '' | 'day' | 'week' | 'month' | 'year';
  scheduledRefresh: boolean;
  refreshKey: RefreshKeyConfig;
  buildRangeStart: string;
  buildRangeEnd: string;
  indexes: IndexConfig[];
  external: boolean;
  rollupLambda: boolean;
  allowNonStrictDateRangeMatch: boolean;
}

interface RefreshKeyConfig {
  type: 'every' | 'sql' | 'incremental';
  every?: string;
  sql?: string;
  updateWindow?: string;
  incrementalColumn?: string;
}

interface IndexConfig {
  name: string;
  columns: string[];
  type: 'regular' | 'aggregate';
}

const GRANULARITIES = [
  { value: 'second', label: 'Second', premium: true },
  { value: 'minute', label: 'Minute' },
  { value: 'hour', label: 'Hour' },
  { value: 'day', label: 'Day' },
  { value: 'week', label: 'Week' },
  { value: 'month', label: 'Month' },
  { value: 'quarter', label: 'Quarter' },
  { value: 'year', label: 'Year' },
];

const PARTITION_GRANULARITIES = [
  { value: '', label: 'No Partitioning' },
  { value: 'hour', label: 'Hourly (Premium)', premium: true },
  { value: 'day', label: 'Daily' },
  { value: 'week', label: 'Weekly' },
  { value: 'month', label: 'Monthly' },
  { value: 'year', label: 'Yearly' },
];

const PreAggregationBuilder: React.FC = () => {
  const [config, setConfig] = useState<PreAggConfig>({
    name: 'main_rollup',
    cubeName: 'Orders',
    measures: ['Orders.count', 'Orders.totalAmount'],
    dimensions: ['Orders.status', 'Orders.category'],
    timeDimension: 'Orders.createdAt',
    granularity: 'day',
    partitionGranularity: 'month',
    scheduledRefresh: true,
    refreshKey: {
      type: 'every',
      every: '1 hour',
    },
    buildRangeStart: '',
    buildRangeEnd: '',
    indexes: [],
    external: true,
    rollupLambda: false,
    allowNonStrictDateRangeMatch: false,
  });

  const [newMeasure, setNewMeasure] = useState('');
  const [newDimension, setNewDimension] = useState('');
  const [previewDialogOpen, setPreviewDialogOpen] = useState(false);
  const [tabValue, setTabValue] = useState(0);

  const updateConfig = (key: keyof PreAggConfig, value: any) => {
    setConfig((prev) => ({ ...prev, [key]: value }));
  };

  const addMeasure = () => {
    if (newMeasure && !config.measures.includes(newMeasure)) {
      updateConfig('measures', [...config.measures, newMeasure]);
      setNewMeasure('');
    }
  };

  const removeMeasure = (measure: string) => {
    updateConfig('measures', config.measures.filter((m) => m !== measure));
  };

  const addDimension = () => {
    if (newDimension && !config.dimensions.includes(newDimension)) {
      updateConfig('dimensions', [...config.dimensions, newDimension]);
      setNewDimension('');
    }
  };

  const removeDimension = (dim: string) => {
    updateConfig('dimensions', config.dimensions.filter((d) => d !== dim));
  };

  const addIndex = () => {
    const newIndex: IndexConfig = {
      name: `idx_${config.indexes.length + 1}`,
      columns: [],
      type: 'regular',
    };
    updateConfig('indexes', [...config.indexes, newIndex]);
  };

  const removeIndex = (idx: number) => {
    updateConfig('indexes', config.indexes.filter((_, i) => i !== idx));
  };

  const generateYaml = () => {
    const preAgg: any = {
      [config.name]: {
        measures: config.measures,
        dimensions: config.dimensions,
        time_dimension: config.timeDimension,
        granularity: config.granularity,
      },
    };

    // Add partitioning (Premium feature)
    if (config.partitionGranularity) {
      preAgg[config.name].partition_granularity = config.partitionGranularity;
    }

    // Add scheduled refresh
    if (config.scheduledRefresh) {
      preAgg[config.name].scheduled_refresh = true;
    }

    // Add refresh key
    if (config.refreshKey.type === 'every') {
      preAgg[config.name].refresh_key = {
        every: config.refreshKey.every,
      };
    } else if (config.refreshKey.type === 'sql') {
      preAgg[config.name].refresh_key = {
        sql: config.refreshKey.sql,
      };
    } else if (config.refreshKey.type === 'incremental') {
      preAgg[config.name].refresh_key = {
        incremental: true,
        update_window: config.refreshKey.updateWindow,
      };
    }

    // Add build range (Premium feature for historical backfill)
    if (config.buildRangeStart) {
      preAgg[config.name].build_range_start = {
        sql: `SELECT '${config.buildRangeStart}'::timestamp`,
      };
    }
    if (config.buildRangeEnd) {
      preAgg[config.name].build_range_end = {
        sql: `SELECT '${config.buildRangeEnd}'::timestamp`,
      };
    }

    // Add external storage (uses StarRocks instead of Cube Store)
    if (config.external) {
      preAgg[config.name].external = true;
    }

    // Add rollup lambda (Premium real-time feature)
    if (config.rollupLambda) {
      preAgg[config.name].rollup_lambda = true;
    }

    // Add indexes (Premium feature)
    if (config.indexes.length > 0) {
      preAgg[config.name].indexes = {};
      config.indexes.forEach((idx) => {
        preAgg[config.name].indexes[idx.name] = {
          columns: idx.columns,
          type: idx.type,
        };
      });
    }

    // Allow non-strict date range matching (Premium feature)
    if (config.allowNonStrictDateRangeMatch) {
      preAgg[config.name].allow_non_strict_date_range_match = true;
    }

    // Generate full cube schema structure
    const cubeSchema = {
      cubes: [
        {
          name: config.cubeName,
          pre_aggregations: preAgg,
        },
      ],
    };

    return yaml.dump(cubeSchema, { indent: 2, lineWidth: 120 });
  };

  const copyToClipboard = () => {
    navigator.clipboard.writeText(generateYaml());
  };

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" gutterBottom>
            Pre-Aggregation Builder
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Configure premium pre-aggregation rollups with partitioning, incremental refresh, and external storage
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 2 }}>
          <Button
            variant="outlined"
            startIcon={<PreviewIcon />}
            onClick={() => setPreviewDialogOpen(true)}
          >
            Preview YAML
          </Button>
          <Button variant="contained" startIcon={<SaveIcon />}>
            Save Definition
          </Button>
        </Box>
      </Box>

      <Grid container spacing={3}>
        {/* Basic Configuration */}
        <Grid item xs={12} md={8}>
          <Paper sx={{ p: 3, mb: 3 }}>
            <Typography variant="h6" gutterBottom>
              Basic Configuration
            </Typography>
            <Grid container spacing={2}>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Pre-Aggregation Name"
                  value={config.name}
                  onChange={(e) => updateConfig('name', e.target.value)}
                  helperText="Unique identifier for this rollup"
                />
              </Grid>
              <Grid item xs={12} sm={6}>
                <TextField
                  fullWidth
                  label="Cube Name"
                  value={config.cubeName}
                  onChange={(e) => updateConfig('cubeName', e.target.value)}
                  helperText="Target cube for pre-aggregation"
                />
              </Grid>
            </Grid>
          </Paper>

          {/* Measures & Dimensions */}
          <Paper sx={{ p: 3, mb: 3 }}>
            <Tabs value={tabValue} onChange={(_, v) => setTabValue(v)} sx={{ mb: 2 }}>
              <Tab label="Measures" />
              <Tab label="Dimensions" />
              <Tab label="Time Dimension" />
            </Tabs>

            {tabValue === 0 && (
              <Box>
                <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                  <TextField
                    size="small"
                    placeholder="e.g., Orders.totalAmount"
                    value={newMeasure}
                    onChange={(e) => setNewMeasure(e.target.value)}
                    onKeyPress={(e) => e.key === 'Enter' && addMeasure()}
                    sx={{ flexGrow: 1 }}
                  />
                  <Button variant="outlined" onClick={addMeasure} startIcon={<AddIcon />}>
                    Add
                  </Button>
                </Box>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                  {config.measures.map((m) => (
                    <Chip
                      key={m}
                      label={m}
                      onDelete={() => removeMeasure(m)}
                      color="primary"
                      variant="outlined"
                    />
                  ))}
                </Box>
              </Box>
            )}

            {tabValue === 1 && (
              <Box>
                <Box sx={{ display: 'flex', gap: 1, mb: 2 }}>
                  <TextField
                    size="small"
                    placeholder="e.g., Orders.status"
                    value={newDimension}
                    onChange={(e) => setNewDimension(e.target.value)}
                    onKeyPress={(e) => e.key === 'Enter' && addDimension()}
                    sx={{ flexGrow: 1 }}
                  />
                  <Button variant="outlined" onClick={addDimension} startIcon={<AddIcon />}>
                    Add
                  </Button>
                </Box>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                  {config.dimensions.map((d) => (
                    <Chip
                      key={d}
                      label={d}
                      onDelete={() => removeDimension(d)}
                      color="secondary"
                      variant="outlined"
                    />
                  ))}
                </Box>
              </Box>
            )}

            {tabValue === 2 && (
              <Box>
                <TextField
                  fullWidth
                  label="Time Dimension"
                  value={config.timeDimension}
                  onChange={(e) => updateConfig('timeDimension', e.target.value)}
                  helperText="Required for time-series pre-aggregations"
                  sx={{ mb: 2 }}
                />
                <Grid container spacing={2}>
                  <Grid item xs={6}>
                    <FormControl fullWidth>
                      <InputLabel>Granularity</InputLabel>
                      <Select
                        value={config.granularity}
                        label="Granularity"
                        onChange={(e) => updateConfig('granularity', e.target.value)}
                      >
                        {GRANULARITIES.map((g) => (
                          <MenuItem key={g.value} value={g.value}>
                            {g.label}
                            {g.premium && (
                              <Chip label="Premium" size="small" color="warning" sx={{ ml: 1 }} />
                            )}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  </Grid>
                  <Grid item xs={6}>
                    <FormControl fullWidth>
                      <InputLabel>Partition Granularity</InputLabel>
                      <Select
                        value={config.partitionGranularity}
                        label="Partition Granularity"
                        onChange={(e) => updateConfig('partitionGranularity', e.target.value)}
                      >
                        {PARTITION_GRANULARITIES.map((g) => (
                          <MenuItem key={g.value} value={g.value}>
                            {g.label}
                            {g.premium && (
                              <Chip label="Premium" size="small" color="warning" sx={{ ml: 1 }} />
                            )}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  </Grid>
                </Grid>
              </Box>
            )}
          </Paper>

          {/* Refresh Configuration */}
          <Accordion defaultExpanded>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <ScheduleIcon sx={{ mr: 1 }} />
              <Typography>Refresh Configuration</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Grid container spacing={2}>
                <Grid item xs={12}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={config.scheduledRefresh}
                        onChange={(e) => updateConfig('scheduledRefresh', e.target.checked)}
                      />
                    }
                    label="Enable Scheduled Refresh"
                  />
                </Grid>

                <Grid item xs={12} sm={4}>
                  <FormControl fullWidth>
                    <InputLabel>Refresh Key Type</InputLabel>
                    <Select
                      value={config.refreshKey.type}
                      label="Refresh Key Type"
                      onChange={(e) =>
                        updateConfig('refreshKey', { ...config.refreshKey, type: e.target.value as any })
                      }
                    >
                      <MenuItem value="every">Time Interval</MenuItem>
                      <MenuItem value="sql">SQL Query</MenuItem>
                      <MenuItem value="incremental">
                        Incremental{' '}
                        <Chip label="Premium" size="small" color="warning" sx={{ ml: 1 }} />
                      </MenuItem>
                    </Select>
                  </FormControl>
                </Grid>

                {config.refreshKey.type === 'every' && (
                  <Grid item xs={12} sm={8}>
                    <TextField
                      fullWidth
                      label="Refresh Interval"
                      value={config.refreshKey.every || ''}
                      onChange={(e) =>
                        updateConfig('refreshKey', { ...config.refreshKey, every: e.target.value })
                      }
                      placeholder="e.g., 1 hour, 30 minutes"
                    />
                  </Grid>
                )}

                {config.refreshKey.type === 'sql' && (
                  <Grid item xs={12} sm={8}>
                    <TextField
                      fullWidth
                      multiline
                      rows={2}
                      label="Refresh Key SQL"
                      value={config.refreshKey.sql || ''}
                      onChange={(e) =>
                        updateConfig('refreshKey', { ...config.refreshKey, sql: e.target.value })
                      }
                      placeholder="SELECT MAX(updated_at) FROM source_table"
                    />
                  </Grid>
                )}

                {config.refreshKey.type === 'incremental' && (
                  <>
                    <Grid item xs={12} sm={4}>
                      <TextField
                        fullWidth
                        label="Update Window"
                        value={config.refreshKey.updateWindow || ''}
                        onChange={(e) =>
                          updateConfig('refreshKey', {
                            ...config.refreshKey,
                            updateWindow: e.target.value,
                          })
                        }
                        placeholder="e.g., 1 day"
                        helperText="Lookback window for incremental updates"
                      />
                    </Grid>
                    <Grid item xs={12} sm={4}>
                      <TextField
                        fullWidth
                        label="Incremental Column"
                        value={config.refreshKey.incrementalColumn || ''}
                        onChange={(e) =>
                          updateConfig('refreshKey', {
                            ...config.refreshKey,
                            incrementalColumn: e.target.value,
                          })
                        }
                        placeholder="e.g., updated_at"
                      />
                    </Grid>
                  </>
                )}
              </Grid>
            </AccordionDetails>
          </Accordion>

          {/* Build Range Configuration */}
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <BuildIcon sx={{ mr: 1 }} />
              <Typography>Build Range (Historical Backfill)</Typography>
              <Chip label="Premium" size="small" color="warning" sx={{ ml: 2 }} />
            </AccordionSummary>
            <AccordionDetails>
              <Alert severity="info" sx={{ mb: 2 }}>
                Configure historical data range for initial pre-aggregation build
              </Alert>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    type="date"
                    label="Build Range Start"
                    value={config.buildRangeStart}
                    onChange={(e) => updateConfig('buildRangeStart', e.target.value)}
                    InputLabelProps={{ shrink: true }}
                  />
                </Grid>
                <Grid item xs={6}>
                  <TextField
                    fullWidth
                    type="date"
                    label="Build Range End"
                    value={config.buildRangeEnd}
                    onChange={(e) => updateConfig('buildRangeEnd', e.target.value)}
                    InputLabelProps={{ shrink: true }}
                  />
                </Grid>
              </Grid>
            </AccordionDetails>
          </Accordion>

          {/* Indexes Configuration */}
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <SpeedIcon sx={{ mr: 1 }} />
              <Typography>Indexes</Typography>
              <Chip label="Premium" size="small" color="warning" sx={{ ml: 2 }} />
            </AccordionSummary>
            <AccordionDetails>
              <Alert severity="info" sx={{ mb: 2 }}>
                Add indexes to optimize pre-aggregation query performance
              </Alert>
              <Button startIcon={<AddIcon />} onClick={addIndex} sx={{ mb: 2 }}>
                Add Index
              </Button>
              {config.indexes.map((idx, i) => (
                <Paper key={i} sx={{ p: 2, mb: 2 }} variant="outlined">
                  <Grid container spacing={2} alignItems="center">
                    <Grid item xs={3}>
                      <TextField
                        size="small"
                        label="Index Name"
                        value={idx.name}
                        onChange={(e) => {
                          const newIndexes = [...config.indexes];
                          newIndexes[i].name = e.target.value;
                          updateConfig('indexes', newIndexes);
                        }}
                      />
                    </Grid>
                    <Grid item xs={5}>
                      <TextField
                        size="small"
                        label="Columns (comma-separated)"
                        value={idx.columns.join(', ')}
                        onChange={(e) => {
                          const newIndexes = [...config.indexes];
                          newIndexes[i].columns = e.target.value.split(',').map((c) => c.trim());
                          updateConfig('indexes', newIndexes);
                        }}
                      />
                    </Grid>
                    <Grid item xs={3}>
                      <FormControl fullWidth size="small">
                        <InputLabel>Type</InputLabel>
                        <Select
                          value={idx.type}
                          label="Type"
                          onChange={(e) => {
                            const newIndexes = [...config.indexes];
                            newIndexes[i].type = e.target.value as any;
                            updateConfig('indexes', newIndexes);
                          }}
                        >
                          <MenuItem value="regular">Regular</MenuItem>
                          <MenuItem value="aggregate">Aggregate</MenuItem>
                        </Select>
                      </FormControl>
                    </Grid>
                    <Grid item xs={1}>
                      <IconButton color="error" onClick={() => removeIndex(i)}>
                        <DeleteIcon />
                      </IconButton>
                    </Grid>
                  </Grid>
                </Paper>
              ))}
            </AccordionDetails>
          </Accordion>
        </Grid>

        {/* Side Panel - Premium Features */}
        <Grid item xs={12} md={4}>
          <Paper sx={{ p: 3, mb: 3 }}>
            <Typography variant="h6" gutterBottom>
              <StorageIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
              Storage Options
            </Typography>
            <FormControlLabel
              control={
                <Switch
                  checked={config.external}
                  onChange={(e) => updateConfig('external', e.target.checked)}
                  color="success"
                />
              }
              label={
                <Box>
                  <Typography>External Storage (StarRocks)</Typography>
                  <Typography variant="caption" color="text.secondary">
                    Use StarRocks for HA pre-aggregation storage
                  </Typography>
                </Box>
              }
            />
            <Divider sx={{ my: 2 }} />
            <FormControlLabel
              control={
                <Switch
                  checked={config.rollupLambda}
                  onChange={(e) => updateConfig('rollupLambda', e.target.checked)}
                  color="warning"
                />
              }
              label={
                <Box>
                  <Typography>
                    Rollup Lambda
                    <Chip label="Premium" size="small" color="warning" sx={{ ml: 1 }} />
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    Combine pre-agg with real-time data
                  </Typography>
                </Box>
              }
            />
            <Divider sx={{ my: 2 }} />
            <FormControlLabel
              control={
                <Switch
                  checked={config.allowNonStrictDateRangeMatch}
                  onChange={(e) => updateConfig('allowNonStrictDateRangeMatch', e.target.checked)}
                />
              }
              label={
                <Box>
                  <Typography>Allow Non-Strict Date Range</Typography>
                  <Typography variant="caption" color="text.secondary">
                    Enable queries outside build range
                  </Typography>
                </Box>
              }
            />
          </Paper>

          <Card variant="outlined">
            <CardContent>
              <Typography variant="h6" gutterBottom color="primary">
                Premium Features Enabled
              </Typography>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                {config.partitionGranularity && (
                  <Chip label={`Partitioning: ${config.partitionGranularity}`} color="success" size="small" />
                )}
                {config.external && (
                  <Chip label="External Storage" color="success" size="small" />
                )}
                {config.rollupLambda && (
                  <Chip label="Rollup Lambda" color="warning" size="small" />
                )}
                {config.refreshKey.type === 'incremental' && (
                  <Chip label="Incremental Refresh" color="warning" size="small" />
                )}
                {config.indexes.length > 0 && (
                  <Chip label={`${config.indexes.length} Custom Indexes`} color="info" size="small" />
                )}
              </Box>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Preview Dialog */}
      <Dialog
        open={previewDialogOpen}
        onClose={() => setPreviewDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          Pre-Aggregation YAML Preview
          <Tooltip title="Copy to Clipboard">
            <IconButton onClick={copyToClipboard} sx={{ float: 'right' }}>
              <CopyIcon />
            </IconButton>
          </Tooltip>
        </DialogTitle>
        <DialogContent>
          <Box
            component="pre"
            sx={{
              backgroundColor: 'grey.900',
              color: 'grey.100',
              p: 2,
              borderRadius: 1,
              overflow: 'auto',
              fontSize: '0.85rem',
              fontFamily: 'monospace',
            }}
          >
            {generateYaml()}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPreviewDialogOpen(false)}>Close</Button>
          <Button variant="contained" onClick={copyToClipboard} startIcon={<CopyIcon />}>
            Copy YAML
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default PreAggregationBuilder;
