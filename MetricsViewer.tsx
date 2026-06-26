import React, { useState, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Tabs,
  Tab,
  Card,
  CardContent,
  IconButton,
  Tooltip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Divider,
  Grid
} from '@mui/material';
import ModalHeader from './frontend/src/components/ModalHeader';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import CodeIcon from '@mui/icons-material/Code';
import InfoIcon from '@mui/icons-material/Info';
import CloseIcon from '@mui/icons-material/Close';
import { getAllMetrics, getDomains, getMetricsByDomain, getFunctionMappings, Metric, FunctionMapping } from './calculationLibrary';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`simple-tabpanel-${index}`}
      aria-labelledby={`simple-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

const MetricsViewer: React.FC = () => {
  const [selectedDomain, setSelectedDomain] = useState<string>('all');
  const [searchTerm, setSearchTerm] = useState<string>('');
  const [tabValue, setTabValue] = useState<number>(0);
  const [selectedMetric, setSelectedMetric] = useState<Metric | null>(null);
  const [detailDialogOpen, setDetailDialogOpen] = useState<boolean>(false);

  const domains = getDomains();
  const allMetrics = getAllMetrics();
  const functionMappings = getFunctionMappings();

  const filteredMetrics = useMemo(() => {
    let metrics = selectedDomain === 'all' ? allMetrics : getMetricsByDomain(selectedDomain);
    
    if (searchTerm) {
      metrics = metrics.filter(metric =>
        metric.node_id.toLowerCase().includes(searchTerm.toLowerCase()) ||
        metric.description.toLowerCase().includes(searchTerm.toLowerCase()) ||
        metric.category.toLowerCase().includes(searchTerm.toLowerCase())
      );
    }
    
    return metrics;
  }, [selectedDomain, searchTerm, allMetrics]);

  const handleTabChange = (event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const handleViewDetails = (metric: Metric) => {
    setSelectedMetric(metric);
    setDetailDialogOpen(true);
  };

  const handleCloseDialog = () => {
    setDetailDialogOpen(false);
    setSelectedMetric(null);
  };

  const renderMetricRow = (metric: Metric) => (
    <TableRow key={metric.node_id}>
      <TableCell>{metric.node_id}</TableCell>
      <TableCell>{metric.category}</TableCell>
      <TableCell>{metric.description}</TableCell>
      <TableCell>
        <Chip 
          label={metric.governance} 
          color={metric.governance === 'golden' ? 'success' : 'warning'} 
          size="small" 
        />
      </TableCell>
      <TableCell>
        {metric.audience.map(aud => (
          <Chip key={aud} label={aud} size="small" variant="outlined" sx={{ mr: 0.5 }} />
        ))}
      </TableCell>
      <TableCell>
        <Tooltip title="View Details">
          <IconButton size="small" onClick={() => handleViewDetails(metric)}>
            <InfoIcon />
          </IconButton>
        </Tooltip>
      </TableCell>
    </TableRow>
  );

  const renderFunctionMappingRow = (mapping: FunctionMapping) => (
    <TableRow key={mapping.dax}>
      <TableCell>{mapping.class}</TableCell>
      <TableCell><code>{mapping.dax}</code></TableCell>
      <TableCell><code>{mapping.neutral}</code></TableCell>
      <TableCell><code>{mapping.sql_server}</code></TableCell>
      <TableCell><code>{mapping.oracle}</code></TableCell>
      <TableCell><code>{mapping.postgres}</code></TableCell>
      <TableCell><code>{mapping.snowflake}</code></TableCell>
      <TableCell><code>{mapping.iceberg}</code></TableCell>
      <TableCell>{mapping.notes}</TableCell>
    </TableRow>
  );

  const renderMetricDetails = () => {
    if (!selectedMetric) return null;

    return (
      <Dialog
        open={detailDialogOpen}
        onClose={handleCloseDialog}
        maxWidth="lg"
        fullWidth
      >
          <ModalHeader title={(
            <Box display="flex" justifyContent="space-between" alignItems="center">
              <Typography variant="h6">{selectedMetric.node_id}</Typography>
              <IconButton onClick={handleCloseDialog}>
                <CloseIcon />
              </IconButton>
            </Box>
          )} onClose={handleCloseDialog} />
        <DialogContent>
          <Typography variant="body1" gutterBottom>
            <strong>Description:</strong> {selectedMetric.description}
          </Typography>
          <Typography variant="body2" gutterBottom>
            <strong>Domain:</strong> {selectedMetric.domain} | <strong>Category:</strong> {selectedMetric.category}
          </Typography>
          
          <Divider sx={{ my: 2 }} />
          
          <Typography variant="h6" gutterBottom>Formulas</Typography>
          <Box sx={{ mb: 2 }}>
            <Typography variant="body2"><strong>Neutral:</strong></Typography>
            <Paper sx={{ p: 1, bgcolor: 'grey.100', fontFamily: 'monospace' }}>
              {selectedMetric.neutral_formula}
            </Paper>
          </Box>
          <Box sx={{ mb: 2 }}>
            <Typography variant="body2"><strong>DAX:</strong></Typography>
            <Paper sx={{ p: 1, bgcolor: 'grey.100', fontFamily: 'monospace' }}>
              {selectedMetric.dax_formula}
            </Paper>
          </Box>
          
          <Divider sx={{ my: 2 }} />
          
          <Typography variant="h6" gutterBottom>SQL Translations</Typography>
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>SQL Server</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Paper sx={{ p: 1, bgcolor: 'grey.100', fontFamily: 'monospace', width: '100%' }}>
                {selectedMetric.sql_server}
              </Paper>
            </AccordionDetails>
          </Accordion>
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>Oracle</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Paper sx={{ p: 1, bgcolor: 'grey.100', fontFamily: 'monospace', width: '100%' }}>
                {selectedMetric.oracle}
              </Paper>
            </AccordionDetails>
          </Accordion>
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>PostgreSQL</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Paper sx={{ p: 1, bgcolor: 'grey.100', fontFamily: 'monospace', width: '100%' }}>
                {selectedMetric.postgres}
              </Paper>
            </AccordionDetails>
          </Accordion>
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>Snowflake</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Paper sx={{ p: 1, bgcolor: 'grey.100', fontFamily: 'monospace', width: '100%' }}>
                {selectedMetric.snowflake}
              </Paper>
            </AccordionDetails>
          </Accordion>
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography>Iceberg/Trino/Spark SQL</Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Paper sx={{ p: 1, bgcolor: 'grey.100', fontFamily: 'monospace', width: '100%' }}>
                {selectedMetric.iceberg}
              </Paper>
            </AccordionDetails>
          </Accordion>
          
          <Divider sx={{ my: 2 }} />
          
          <Typography variant="h6" gutterBottom>Preaggregation Configuration</Typography>
          <Box sx={{ mb: 1 }}>
            <Typography variant="body2">
              <strong>Enabled:</strong> {selectedMetric.preaggregation.enabled ? 'Yes' : 'No'} | 
              <strong>Grain:</strong> {selectedMetric.preaggregation.grain} | 
              <strong>Snapshot:</strong> {selectedMetric.preaggregation.snapshot ? 'Yes' : 'No'}
            </Typography>
          </Box>
          <Box sx={{ mb: 1 }}>
            <Typography variant="body2"><strong>Rollups:</strong> {selectedMetric.preaggregation.rollups.join(', ')}</Typography>
          </Box>
          {selectedMetric.preaggregation.partition_keys && (
            <Box sx={{ mb: 1 }}>
              <Typography variant="body2"><strong>Partition Keys:</strong> {selectedMetric.preaggregation.partition_keys.join(', ')}</Typography>
            </Box>
          )}
          
          <Divider sx={{ my: 2 }} />
          
          <Typography variant="h6" gutterBottom>Governance & Audience</Typography>
          <Box sx={{ mb: 1 }}>
            <Chip 
              label={selectedMetric.governance} 
              color={selectedMetric.governance === 'golden' ? 'success' : 'warning'} 
            />
          </Box>
          <Box>
            <Typography variant="body2"><strong>Audience:</strong></Typography>
            {selectedMetric.audience.map(aud => (
              <Chip key={aud} label={aud} size="small" variant="outlined" sx={{ mr: 0.5, mb: 0.5 }} />
            ))}
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Close</Button>
        </DialogActions>
      </Dialog>
    );
  };

  return (
    <Box sx={{ width: '100%', p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Financial Services Semantic Layer Metrics
      </Typography>
      
      <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tabs value={tabValue} onChange={handleTabChange} aria-label="metrics tabs">
          <Tab label="Metrics" />
          <Tab label="Function Mappings" />
        </Tabs>
      </Box>

      <TabPanel value={tabValue} index={0}>
        <Box sx={{ mb: 3 }}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} md={4}>
              <FormControl fullWidth>
                <InputLabel>Domain</InputLabel>
                <Select
                  value={selectedDomain}
                  label="Domain"
                  onChange={(e) => setSelectedDomain(e.target.value)}
                >
                  <MenuItem value="all">All Domains</MenuItem>
                  {domains.map(domain => (
                    <MenuItem key={domain} value={domain}>
                      {domain.replace('_', ' ').toUpperCase()}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} md={8}>
              <TextField
                fullWidth
                label="Search metrics..."
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
                placeholder="Search by ID, description, or category"
              />
            </Grid>
          </Grid>
        </Box>

        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell><strong>Node ID</strong></TableCell>
                <TableCell><strong>Category</strong></TableCell>
                <TableCell><strong>Description</strong></TableCell>
                <TableCell><strong>Governance</strong></TableCell>
                <TableCell><strong>Audience</strong></TableCell>
                <TableCell><strong>Actions</strong></TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredMetrics.map(renderMetricRow)}
            </TableBody>
          </Table>
        </TableContainer>

        <Typography variant="body2" sx={{ mt: 2, color: 'text.secondary' }}>
          Showing {filteredMetrics.length} of {allMetrics.length} metrics
        </Typography>
      </TabPanel>

      <TabPanel value={tabValue} index={1}>
        <Typography variant="h6" gutterBottom>
          DAX to SQL Function Mapping Appendix
        </Typography>
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell><strong>Class</strong></TableCell>
                <TableCell><strong>DAX Function</strong></TableCell>
                <TableCell><strong>Neutral</strong></TableCell>
                <TableCell><strong>SQL Server</strong></TableCell>
                <TableCell><strong>Oracle</strong></TableCell>
                <TableCell><strong>PostgreSQL</strong></TableCell>
                <TableCell><strong>Snowflake</strong></TableCell>
                <TableCell><strong>Iceberg/Trino/Spark</strong></TableCell>
                <TableCell><strong>Notes</strong></TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {functionMappings.map(renderFunctionMappingRow)}
            </TableBody>
          </Table>
        </TableContainer>
      </TabPanel>

      {renderMetricDetails()}
    </Box>
  );
};

export default MetricsViewer;
