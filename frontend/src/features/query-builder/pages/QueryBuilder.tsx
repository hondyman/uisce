import { useState, type KeyboardEvent } from 'react';
import {
  Box, TextField, Button, Typography, Paper, CircularProgress,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow, Chip, Grid, Tabs, Tab
} from '@mui/material';
import { RechartsBarChart } from '../../../components/ChartLoader';
import axios from 'axios';
// Inline adapter: a lightweight schema explorer placed in the Query Builder sidebar.
import { useEffect } from 'react';
import ConversationalQueryInterface from '../../../ConversationalQueryInterface';
import { useTenant } from '../../../contexts/TenantContext';
import { useAuth } from '../../../contexts/AuthContext';

const SchemaBrowserAdapter: React.FC<{ setDimensions: (d: string[]) => void; setMeasures: (m: string[]) => void; setView: (v: string) => void }>
  = ({ setDimensions, setMeasures, setView }) => {
  useEffect(() => {
    // Attempt to fetch the same schema endpoint used by the previous SchemaBrowser
    const fetchSchema = async () => {
      try {
        const resp = await fetch('/api/schema', { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({}) });
        if (!resp.ok) return;
        const data = await resp.json();
        // No structural contract here; this is a best-effort adapter.
        if (Array.isArray(data) && data.length > 0) {
          const first = data[0];
          if (first.columns && Array.isArray(first.columns)) {
            setDimensions(first.columns.slice(0, 3).map((c: any) => c.column_name));
            setMeasures([]);
            setView(first.table_name || '');
          }
        }
      } catch (e) {
        // ignore — this adapter is intentionally tolerant
      }
    };
    fetchSchema();
  }, [setDimensions, setMeasures, setView]);

  return <Box sx={{ minHeight: 200 }}>Schema explorer (compact)</Box>;
};

// A reusable component for chip-based input
const ChipInput: React.FC<{ label: string; chips: string[]; setChips: (c: string[]) => void }> = ({ label, chips, setChips }) => {
  const [inputValue, setInputValue] = useState<string>('');

  const handleAddChip = (e: KeyboardEvent<HTMLInputElement>) => {
    if ((e as KeyboardEvent<HTMLInputElement>).key === 'Enter' && inputValue.trim() !== '') {
      (e as KeyboardEvent<HTMLInputElement>).preventDefault();
      // Prevent duplicates
      if (!chips.includes(inputValue.trim())) {
        setChips([...chips, inputValue.trim()]);
      }
      setInputValue('');
    }
  };

  const handleDeleteChip = (chipToDelete: string) => {
    setChips(chips.filter((chip) => chip !== chipToDelete));
  };

  return (
    <Box>
      <TextField
        label={label}
        variant="outlined"
        value={inputValue}
        onChange={(e) => setInputValue(e.target.value)}
        onKeyDown={handleAddChip}
        fullWidth
        helperText="Type a value and press Enter to add."
      />
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, mt: 1, minHeight: '32px' }}>
        {chips.map((chip, index) => (
          <Chip
            key={index}
            label={chip}
            onDelete={() => handleDeleteChip(chip)}
          />
        ))}
      </Box>
    </Box>
  );
};

const QueryBuilder = () => {
  const { tenant, datasource } = useTenant();
  const { user } = useAuth();
  const [activeTab, setActiveTab] = useState(0);
  const [view, setView] = useState('orders');
  const [dimensions, setDimensions] = useState(['order_date', 'ship_country']);
  const [measures, setMeasures] = useState(['total_freight']);
  const [filters, setFilters] = useState('');

  const [results, setResults] = useState<any[] | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  

  const handleExecuteQuery = async () => {
    setLoading(true);
    setError('');
    setResults(null);

    const queryPayload = {
      view,
      dimensions,
      measures,
      filters,
    };

    try {
      const apiBase = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8000';
      const response = await axios.post(`${apiBase.replace(/\/$/, '')}/api/query`, queryPayload);
      setResults(response.data);
    } catch (err) {
      const e: any = err
      setError(e?.response?.data?.details || e?.message || 'An unexpected error occurred.');
    } finally {
      setLoading(false);
    }
  };

  const getTableHeaders = () => {
    if (!results || results.length === 0) return [];
    return Object.keys(results[0]);
  };

  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Query Builder
      </Typography>
      <Tabs value={activeTab} onChange={(_, newValue) => setActiveTab(newValue)} sx={{ borderBottom: 1, borderColor: 'divider' }}>
        <Tab label="Query Builder" />
        <Tab label="NL Query" />
      </Tabs>
      {activeTab === 0 && (
        <Grid container spacing={3} sx={{ mt: 2 }}>
          <Grid item xs={12} md={4}>
            <Typography variant="h6" gutterBottom>Schema</Typography>
            <Paper sx={{ p: 2, height: '100%' }}>
              <SchemaBrowserAdapter
                setDimensions={setDimensions as any}
                setMeasures={setMeasures as any}
                setView={setView as any}
              />
            </Paper>
          </Grid>
          <Grid item xs={12} md={8}>
            <Typography variant="h6" gutterBottom>Query</Typography>
            <Paper sx={{ p: 2, mb: 3 }}>
              <Box component="form" noValidate autoComplete="off" sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                <TextField
                  label="View Name"
                  variant="outlined"
                  value={view}
                  onChange={(e) => setView(e.target.value)}
                  fullWidth
                />
                <ChipInput label="Dimensions" chips={dimensions} setChips={setDimensions} />
                <ChipInput label="Measures" chips={measures} setChips={setMeasures} />
                <TextField
                  label="Filters (SQL WHERE clause)"
                  variant="outlined"
                  multiline
                  rows={4}
                  value={filters}
                  onChange={(e) => setFilters(e.target.value)}
                  fullWidth
                />
                <Box>
                  <Button variant="contained" onClick={handleExecuteQuery} disabled={loading}>
                    {loading ? <CircularProgress size={24} /> : 'Execute Query'}
                  </Button>
                </Box>
              </Box>
            </Paper>
          </Grid>
        </Grid>
      )}
      {activeTab === 1 && (
        <Box sx={{ mt: 2 }}>
          <ConversationalQueryInterface
            currentDatasource={datasource?.id || ''}
            currentUser={user?.id || ''}
            currentTenant={tenant?.id || ''}
          />
        </Box>
      )}

      {activeTab === 0 && error && (
        <Typography color="error" sx={{ mt: 2 }}>
          Error: {error}
        </Typography>
      )}

      {activeTab === 0 && results && results.length > 0 && (
        <Box>
          <Typography variant="h5" gutterBottom sx={{ mt: 4 }}>
            Results
          </Typography>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Typography variant="h6" gutterBottom>Chart</Typography>
              <Paper sx={{ p: 2, height: 400 }}>
                <RechartsBarChart data={results} xKey={dimensions[0]} yKey={measures[0]} />
              </Paper>
            </Grid>
            <Grid item xs={12} md={6}>
              <Typography variant="h6" gutterBottom>Table</Typography>
              <TableContainer component={Paper} sx={{maxHeight: 440}}>
                <Table stickyHeader sx={{ minWidth: 650 }} aria-label="simple table">
                  <TableHead>
                    <TableRow>
                      {getTableHeaders().map((header) => (
                        <TableCell key={header}><b>{header}</b></TableCell>
                      ))}
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {results.map((row, index) => (
                      <TableRow key={index}>
                        {getTableHeaders().map((header) => (
                          <TableCell key={header}>
                            {typeof row[header] === 'object' ? JSON.stringify(row[header]) : row[header]}
                          </TableCell>
                        ))}
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Grid>
          </Grid>
        </Box>
      )}
    </Box>
  );
};

export default QueryBuilder;