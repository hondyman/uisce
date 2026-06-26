import { useState, useEffect, useRef } from 'react';
import { devDebug, devWarn, devError } from '../utils/devLogger';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  LinearProgress,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Grid,
  Alert,
  CircularProgress,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Collapse,
  Divider,
  IconButton
} from '@mui/material';
import { Checkbox, Snackbar } from '@mui/material';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  ResponsiveContainer,
  PieChart,
  Pie,
  Cell,
  LineChart,
  Line
} from 'recharts';
import { Tooltip as RechartsTooltip } from 'recharts';
import { BarChart as BarChartIcon } from 'lucide-react';
import { useTenant } from '../contexts/TenantContext';
import { Tooltip } from '@mui/material';
import ExpandLess from '@mui/icons-material/ExpandLess';
import ExpandMore from '@mui/icons-material/ExpandMore';
import RefreshIcon from '@mui/icons-material/Refresh';
import FolderIcon from '@mui/icons-material/Folder';
import TableChartIcon from '@mui/icons-material/TableChart';
import AccountTreeIcon from '@mui/icons-material/AccountTree';

interface CatalogNode {
  id: string;
  node_name: string;
  catalog_type: string;
  parent_id?: string;
  properties?: {
    schema?: string;
    data_type?: string;
    is_nullable?: boolean;
    ordinal_position?: number;
  };
  qualified_path: string;
}

interface ProfileResult {
  Schema: string;
  TableName: string;
  ColumnName: string;
  DataType: string;
  Cardinality: number;
  MinLength?: number;
  MaxLength?: number;
  AvgLength?: number;
  FrequentValues?: string[];
  InferredPatterns?: string[];
}

const CHART_COLORS = ['#8884d8', '#82ca9d', '#ffc658', '#ff7f50', '#a28bd4', '#ffb347', '#87ceeb'];

type ProfilerPageProps = {
  preselectedSchema?: string | null
  preselectedTable?: string | null
  preselectedTables?: string[] | null
}

export default function ProfilerPage({ preselectedSchema, preselectedTable, preselectedTables }: ProfilerPageProps) {
  const { tenant, datasource } = useTenant();
  
  // State for schema/table/column selection
  const [schemas, setSchemas] = useState<CatalogNode[]>([]);
  const [tables, setTables] = useState<CatalogNode[]>([]);
  const [columns, setColumns] = useState<CatalogNode[]>([]);
  const [selectedSchema, setSelectedSchema] = useState<string>('');
  const [selectedTable, setSelectedTable] = useState<string>('');
  const [loading, setLoading] = useState(false);
  const [expandedSchemas, setExpandedSchemas] = useState<Record<string, boolean>>({});
  const [selectedTables, setSelectedTables] = useState<string[]>([]);
  const [selectedSchemas, setSelectedSchemas] = useState<string[]>([]);
  const [runScope, setRunScope] = useState<'table'|'schema'|'selected'>('table');
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [snackbarMessage, setSnackbarMessage] = useState<string>('');
  // TODO: use auto-run flag to trigger run when tables/columns are fully loaded
  const runButtonRef = useRef<HTMLButtonElement | null>(null);
  
  // State for profiler results (include optional ColumnId when results are enriched)
  const [profileResults, setProfileResults] = useState<(ProfileResult & { ColumnId?: string })[]>([]);
  const [selectedColumn, setSelectedColumn] = useState<ProfileResult | null>(null);
  const [showColumnDetail, setShowColumnDetail] = useState(false);
  const [profiling, setProfiling] = useState(false);
  const [profileProgress, setProfileProgress] = useState<string>('');
  // Pagination for profiler results
  const [page, setPage] = useState<number>(0); // zero-based
  const [limit, setLimit] = useState<number>(100);
  const [hasMore, setHasMore] = useState<boolean>(false);

  const getHeaders = () => {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };
    if (tenant?.id) {
      headers['X-Tenant-ID'] = tenant.id;
    }
    if (datasource?.id) {
      headers['X-Tenant-Datasource-ID'] = datasource.id;
    }
    return headers;
  };

  // Fetch schemas (callable by refresh button)
  const fetchSchemas = async () => {
    if (!tenant?.id || !datasource?.id) return;
    setLoading(true);
    try {
      const response = await fetch(
        `/api/catalog/nodes?type=schema&tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&limit=100`,
        { headers: getHeaders() }
      );
      if (response.ok) {
        const data = await response.json();
        if (Array.isArray(data)) {
          setSchemas(data);
        }
      }
    } catch (error) {
      devError('Failed to fetch schemas:', error);
    } finally {
      setLoading(false);
    }
  };

  // Fetch schemas on component mount
  useEffect(() => {
    fetchSchemas();
  }, [tenant?.id, datasource?.id]);

  // Load existing profile results when component mounts or tenant/datasource/schema/table changes
  useEffect(() => {
    if (tenant?.id && datasource?.id) {
      fetchResultsOnce(0, limit);
    }
  }, [tenant?.id, datasource?.id, selectedSchema, selectedTable]);  // Clear profile results when schema selection changes
  useEffect(() => {
    setProfileResults([]);
    setPage(0);
  }, [selectedSchema]);

  // Clear profile results when table selection changes
  useEffect(() => {
    setProfileResults([]);
    setPage(0);
  }, [selectedTable]);

  // apply preselection if provided
  useEffect(() => {
    if (preselectedSchema) {
      setSelectedSchema(preselectedSchema);
    }
    if (preselectedTable) {
      setSelectedTable(preselectedTable);
    }
    if (preselectedTables && Array.isArray(preselectedTables)) {
      setSelectedTables(preselectedTables);
      // expand schemas that contain selected tables
      const expanded: Record<string, boolean> = {};
      preselectedTables.forEach(t => {
        // try to find table in tables list and expand its schema if found
        const match = tables.find(tbl => tbl.node_name === t);
        if (match && match.qualified_path) {
          const parts = match.qualified_path.split('/').filter(Boolean);
          if (parts.length > 0) expanded[parts[0]] = true;
        }
      });
      setExpandedSchemas(prev => ({ ...prev, ...expanded }));
      // request auto-run; actual run will wait until tables/columns are available
  setRunScope('selected');
    }
  }, [preselectedSchema, preselectedTable, preselectedTables, tables]);

  // Fetch tables when schema is selected
  useEffect(() => {
    const fetchTables = async () => {
      if (!selectedSchema || !tenant?.id || !datasource?.id) {
        setTables([]);
        return;
      }
      
      setLoading(true);
      try {
        const response = await fetch(
          `/api/catalog/nodes?type=table&q=${selectedSchema}&tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&limit=100`,
          { headers: getHeaders() }
        );
        if (response.ok) {
          const data = await response.json();
          if (Array.isArray(data)) {
            // Filter tables that belong to the selected schema
            const filteredTables = data.filter(table => 
              table.qualified_path.startsWith(`/${selectedSchema}/`)
            );
            setTables(filteredTables);
          }
        }
      } catch (error) {
        devError('Failed to fetch tables:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchTables();
  }, [selectedSchema, tenant?.id, datasource?.id]);

  // If preselectedTables were provided by the scanner, and the tables list has been loaded,
  // trigger an automatic profiling run for the selected tables.
  useEffect(() => {
    if (!preselectedTables || !Array.isArray(preselectedTables) || preselectedTables.length === 0) return;
    // ensure tables for the selected schema are loaded and contain the requested names
    const allResolved = preselectedTables.every(tn => tables.some(t => t.node_name === tn));
    if (allResolved) {
      setSelectedTables(preselectedTables);
      setRunScope('selected');
      // slight delay to allow UI state updates before running
      setTimeout(() => { runProfiling(); }, 0);
    }
  }, [preselectedTables, tables]);

  // Fetch columns when table is selected
  useEffect(() => {
    const fetchColumns = async () => {
      if (!selectedTable || !tenant?.id || !datasource?.id) {
        setColumns([]);
        return;
      }
      
      setLoading(true);
      try {
        // Find the selected table object to get its ID
        const table = tables.find(t => t.node_name === selectedTable);
        if (!table) return;

        const response = await fetch(
          `/api/catalog/nodes?type=column&tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&limit=100`,
          { headers: getHeaders() }
        );
        if (response.ok) {
          const data = await response.json();
          if (Array.isArray(data)) {
            // Filter columns that belong to the selected table
            const filteredColumns = data.filter(column => 
              column.parent_id === table.id
            ).sort((a, b) => (a.properties?.ordinal_position || 0) - (b.properties?.ordinal_position || 0));
            setColumns(filteredColumns);
          }
        }
      } catch (error) {
        devError('Failed to fetch columns:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchColumns();
  }, [selectedTable, tables, tenant?.id, datasource?.id]);

  // Run profiling on the selected table
  const runProfiling = async (opts?: { schemas?: string[]; tables?: string[] }) => {
    if (!tenant?.id || !datasource?.id) return;

    // determine node ids based on explicit opts or current selection state
    const schemasToUse = opts?.schemas ?? selectedSchemas.length > 0 ? selectedSchemas : (selectedSchema ? [selectedSchema] : []);
    const tablesToUse = opts?.tables ?? (selectedTables.length > 0 ? selectedTables : (selectedTable ? [selectedTable] : []));

    let nodeIds: string[] = [];

    // If specific tables were chosen, get their column node IDs
    if (tablesToUse && tablesToUse.length > 0) {
      const tableNodeIds = tablesToUse.map(name => tables.find(t => t.node_name === name)).filter(Boolean).map((t:any)=>t.id);
      // For each table, get all its column node IDs
      const columnPromises = tableNodeIds.map(async (tableId) => {
        try {
          const response = await fetch(`/api/catalog/nodes?type=column&parent_id=${encodeURIComponent(tableId)}&limit=1000`, { headers: getHeaders() });
          if (response.ok) {
            const columnData = await response.json();
            return Array.isArray(columnData) ? columnData.map((c: any) => c.id) : [];
          }
        } catch (err) {
          devWarn('Failed to fetch columns for table', tableId);
        }
        return [];
      });
      const columnArrays = await Promise.all(columnPromises);
      nodeIds = columnArrays.flat();
    } else if (schemasToUse && schemasToUse.length > 0) {
      // Get all column node IDs for the selected schemas
      try {
        const response = await fetch(`/api/catalog/nodes?type=column&tenant_id=${tenant.id}&tenant_instance_id=${datasource.id}&limit=10000`, { headers: getHeaders() });
        if (response.ok) {
          const columnData = await response.json();
          if (Array.isArray(columnData)) {
            // Filter columns that belong to the selected schemas
            const schemaColumns = columnData.filter(col => 
              schemasToUse.some(schema => col.qualified_path.startsWith(`/${schema}/`))
            );
            nodeIds = schemaColumns.map(col => col.id);
          }
        }
      } catch (error) {
        devError('Failed to fetch columns for schemas:', error);
        setProfileProgress('Error fetching columns for schemas');
        return;
      }
    } else {
      setProfileProgress('No schema or table selected to profile');
      return;
    }

    setProfiling(true);
    setProfileProgress('Starting profiler...');
    setSnackbarMessage(`Profiling started (${runScope}) - running ${nodeIds.length} node(s)`);
    setSnackbarOpen(true);

    try {
      const response = await fetch('/api/profiler/profile', {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ node_ids: nodeIds }),
      });

      if (response.ok) {
        const result = await response.json();
        const jobId = result.jobId;
        // Poll for results; pass filters when meaningful (single table or schema)
        const filters: any = {};
        if (runScope === 'table') filters.table = selectedTable;
        if (runScope === 'schema') filters.schema = selectedSchema;
        pollForResults(jobId, page, limit, filters);
      } else {
        devError('Failed to start profiling:', response.statusText);
        setProfileProgress('Failed to start profiling');
      }
    } catch (error) {
      devError('Failed to start profiling:', error);
      setProfileProgress('Error starting profiling');
    }
  };
  const pollForResults = async (_jobId: string, pageParam = 0, limitParam = limit, filters?: { schema?: string; table?: string }) => {
    let attempts = 0;
    const maxAttempts = 30; // 30 attempts * 2 seconds = 1 minute max
    
    const poll = async () => {
      try {
        attempts++;
        setProfileProgress(`Profiling in progress... (${attempts}/${maxAttempts})`);
        
  const qs = new URLSearchParams();
  const schemaToUse = filters?.schema ?? selectedSchema;
  const tableToUse = filters?.table ?? selectedTable;
  if (schemaToUse) qs.set('schema', schemaToUse);
  if (tableToUse) qs.set('table', tableToUse);
        // paging
        qs.set('limit', String(limitParam));
        qs.set('offset', String(pageParam * limitParam));
        const url = `/api/profiler/results${qs.toString() ? `?${qs.toString()}` : ''}`;
        const response = await fetch(url, {
          headers: getHeaders(),
        });
        
        if (response.ok) {
          const data = await response.json();
          if (data.profiles && Array.isArray(data.profiles)) {
            // For multi-table runs we may not have columns loaded; fetch columns for all involved tables
            const tableNames: string[] = Array.from(new Set(data.profiles.map((p: any) => String(p.TableName))));
            const tableNameToId: Record<string, string> = {};
            tableNames.forEach((tn) => {
              const t: any = tables.find((x: any) => x.node_name === tn);
              if (t) tableNameToId[tn] = t.id;
            });

            const columnNameToId: Record<string, string> = {};
            // fetch columns for each table id
            const fetchedColumns: any[] = [];
            await Promise.all(Object.values(tableNameToId).map(async (tableId) => {
              try {
                const resp = await fetch(`/api/catalog/nodes?type=column&parent_id=${encodeURIComponent(tableId)}&limit=100`, { headers: getHeaders() });
                if (!resp.ok) return;
                const colData = await resp.json();
                if (Array.isArray(colData)) {
                  colData.forEach((c: any) => { if (c.node_name) columnNameToId[c.node_name] = c.id });
                  fetchedColumns.push(...colData);
                }
              } catch (err) {
                // ignore
              }
            }));

            // merge fetched columns into columns state (avoid duplicates by id)
            if (fetchedColumns.length > 0) {
              setColumns(prev => {
                const byId: Record<string, any> = {};
                (prev || []).forEach((c: any) => { byId[c.id] = c });
                fetchedColumns.forEach((c: any) => { if (c && c.id) byId[c.id] = c });
                return Object.values(byId);
              });
            }

            const profilesWithIds = data.profiles.map((p: any) => {
              const colId = columnNameToId[p.ColumnName] || (columns.find(c => c.node_name === p.ColumnName)?.id);
              return colId ? { ...p, ColumnId: colId } : p;
            });

            setProfileResults(profilesWithIds);
            setHasMore(Array.isArray(profilesWithIds) && profilesWithIds.length >= limitParam);
            setProfiling(false);
            setProfileProgress('');
            return;
          }
        }
        
        if (attempts < maxAttempts) {
          setTimeout(poll, 2000); // Poll every 2 seconds
        } else {
          setProfiling(false);
          setProfileProgress('Profiling timed out');
        }
      } catch (error) {
        devError('Failed to poll for results:', error);
        setProfiling(false);
        setProfileProgress('Error polling for results');
      }
    };
    
    poll();
  };

  // One-off fetch for a particular page/limit (no polling)
  const fetchResultsOnce = async (pageParam = 0, limitParam = limit) => {
    try {
      setProfileProgress('Fetching profiler results...');
      const qs = new URLSearchParams();
      if (selectedSchema) qs.set('schema', selectedSchema);
      if (selectedTable) qs.set('table', selectedTable);
      qs.set('limit', String(limitParam));
      qs.set('offset', String(pageParam * limitParam));
      const url = `/api/profiler/results${qs.toString() ? `?${qs.toString()}` : ''}`;
      const response = await fetch(url, { headers: getHeaders() });
      if (response.ok) {
        const data = await response.json();
        if (data.profiles && Array.isArray(data.profiles)) {
          const tableNames: string[] = Array.from(new Set(data.profiles.map((p: any) => String(p.TableName))));
          const tableNameToId: Record<string, string> = {};
          tableNames.forEach((tn) => {
            const t: any = tables.find((x: any) => x.node_name === tn);
            if (t) tableNameToId[tn] = t.id;
          });

          const columnNameToId: Record<string, string> = {};
          const fetchedColumns: any[] = [];
          await Promise.all(Object.values(tableNameToId).map(async (tableId) => {
            try {
              const resp = await fetch(`/api/catalog/nodes?type=column&parent_id=${encodeURIComponent(tableId)}&limit=100`, { headers: getHeaders() });
              if (!resp.ok) return;
              const colData = await resp.json();
              if (Array.isArray(colData)) {
                colData.forEach((c: any) => { if (c.node_name) columnNameToId[c.node_name] = c.id });
                fetchedColumns.push(...colData);
              }
            } catch (err) {}
          }));

          if (fetchedColumns.length > 0) {
            setColumns(prev => {
              const byId: Record<string, any> = {};
              (prev || []).forEach((c: any) => { byId[c.id] = c });
              fetchedColumns.forEach((c: any) => { if (c && c.id) byId[c.id] = c });
              return Object.values(byId);
            });
          }

          const profilesWithIds = data.profiles.map((p: any) => {
            const colId = columnNameToId[p.ColumnName] || (columns.find(c => c.node_name === p.ColumnName)?.id);
            return colId ? { ...p, ColumnId: colId } : p;
          });

          setProfileResults(profilesWithIds);
          setHasMore(Array.isArray(profilesWithIds) && profilesWithIds.length >= limitParam);
        } else {
          devDebug('No profiles found in response');
        }
      } else {
        devError('API response not ok:', response.status, response.statusText);
        const errorText = await response.text();
        devError('Error response body:', errorText);
      }
    } catch (error) {
      devError('Failed to fetch profiler results:', error);
    } finally {
      setProfileProgress('');
    }
  };

  // Pagination controls
  const handlePrev = () => {
    if (page <= 0) return;
    const newPage = page - 1;
    setPage(newPage);
    fetchResultsOnce(newPage, limit);
  };

  const handleNext = () => {
    if (!hasMore) return;
    const newPage = page + 1;
    setPage(newPage);
    fetchResultsOnce(newPage, limit);
  };

  const handleLimitChange = (e: any) => {
    const newLimit = Number(e.target.value) || 100;
    setLimit(newLimit);
    setPage(0);
    fetchResultsOnce(0, newLimit);
  };

  const handleColumnClick = (profile: ProfileResult) => {
    setSelectedColumn(profile);
    setShowColumnDetail(true);
  };

  const toggleSchema = (schemaName: string) => {
    setExpandedSchemas(prev => ({ ...prev, [schemaName]: !prev[schemaName] }));
    // set active selected schema for context
    setSelectedSchema(schemaName);
    setSelectedTable('');
    setProfileResults([]);
  };

  const toggleSchemaSelection = (schemaName: string) => {
    setSelectedSchemas(prev => prev.includes(schemaName) ? prev.filter(s => s !== schemaName) : [...prev, schemaName]);
    // clear table selection when schemas change to avoid confusion
    setSelectedTable('');
    setProfileResults([]);
  };

  const toggleTableSelection = (tableName: string) => {
    setSelectedTables(prev => {
      if (prev.includes(tableName)) return prev.filter(t => t !== tableName);
      return [...prev, tableName];
    });
  };

  // Keyboard shortcut: 'p' to run profiling on selected table
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.key.toLowerCase() === 'p') {
        if (selectedTable || selectedTables.length > 0 || selectedSchemas.length > 0) runProfiling();
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [selectedTable, tables]);

  // Focus run button when preselection changes (e.g., coming from Scanner)
  useEffect(() => {
    if (preselectedTable && runButtonRef.current) {
      runButtonRef.current.focus();
    }
  }, [preselectedTable]);

  const renderColumnDetailDialog = () => {
    if (!selectedColumn) return null;

    // Use actual frequent values data (assume they're ordered by frequency)
    const frequentValuesData = (selectedColumn.FrequentValues || []).slice(0, 10).map((value, index) => ({
      value: value === '<nil>' ? 'NULL' : value,
      // Estimate frequency based on position (most frequent first)
      count: Math.max(1, selectedColumn.Cardinality - index * Math.floor(selectedColumn.Cardinality / 10)),
      index
    }));

    const patternsData = (selectedColumn.InferredPatterns || []).map((pattern, index) => ({
      pattern,
      // All patterns detected have equal weight for now
      frequency: 1,
      index
    }));

    return (
      <Dialog
        open={showColumnDetail}
        onClose={() => setShowColumnDetail(false)}
        maxWidth="lg"
        fullWidth
      >
        <DialogTitle>
          Column Profile: {selectedColumn.Schema}.{selectedColumn.TableName}.{selectedColumn.ColumnName}
        </DialogTitle>
        <DialogContent>
          <Grid container spacing={3}>
            {/* Basic Info */}
            <Grid item xs={12}>
              <Card>
                <CardContent>
                  <Typography variant="h6" gutterBottom>Basic Information</Typography>
                  <Grid container spacing={2}>
                    <Grid item xs={3}>
                      <Typography variant="body2" color="textSecondary">Data Type</Typography>
                      <Chip label={selectedColumn.DataType} color="primary" />
                    </Grid>
                    <Grid item xs={3}>
                      <Typography variant="body2" color="textSecondary">Cardinality</Typography>
                      <Typography variant="h6">{selectedColumn.Cardinality.toLocaleString()}</Typography>
                    </Grid>
                    <Grid item xs={3}>
                      <Typography variant="body2" color="textSecondary">Min Length</Typography>
                      <Typography variant="h6">{selectedColumn.MinLength || 'N/A'}</Typography>
                    </Grid>
                    <Grid item xs={3}>
                      <Typography variant="body2" color="textSecondary">Max Length</Typography>
                      <Typography variant="h6">{selectedColumn.MaxLength || 'N/A'}</Typography>
                    </Grid>
                  </Grid>
                </CardContent>
              </Card>
            </Grid>

            {/* Frequent Values Chart */}
            {frequentValuesData.length > 0 && (
              <Grid item xs={12} md={6}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>Most Frequent Values</Typography>
                    <ResponsiveContainer width="100%" height={300}>
                      <BarChart data={frequentValuesData}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis 
                          dataKey="value" 
                          angle={-45}
                          textAnchor="end"
                          height={80}
                        />
                        <YAxis />
                        <RechartsTooltip />
                        <Bar dataKey="count" fill="#8884d8" />
                      </BarChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>
              </Grid>
            )}

            {/* Data Patterns */}
            {patternsData.length > 0 && (
              <Grid item xs={12} md={6}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>Data Patterns</Typography>
                    <ResponsiveContainer width="100%" height={300}>
                      <PieChart>
                        <Pie
                          data={patternsData}
                          cx="50%"
                          cy="50%"
                          outerRadius={80}
                          fill="#8884d8"
                          dataKey="frequency"
                          label={({ pattern }) => pattern}
                        >
                          {patternsData.map((_, index) => (
                            <Cell key={`cell-${index}`} fill={CHART_COLORS[index % CHART_COLORS.length]} />
                          ))}
                        </Pie>
                        <RechartsTooltip />
                      </PieChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>
              </Grid>
            )}

            {/* Length Distribution (if applicable) */}
            {selectedColumn.MinLength !== undefined && selectedColumn.MaxLength !== undefined && (
              <Grid item xs={12}>
                <Card>
                  <CardContent>
                    <Typography variant="h6" gutterBottom>Length Distribution</Typography>
                    <ResponsiveContainer width="100%" height={200}>
                      <LineChart
                        data={[
                          { length: selectedColumn.MinLength, frequency: 10 },
                          { length: selectedColumn.AvgLength || ((selectedColumn.MinLength + selectedColumn.MaxLength) / 2), frequency: 50 },
                          { length: selectedColumn.MaxLength, frequency: 5 }
                        ]}
                      >
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="length" />
                        <YAxis />
                        <RechartsTooltip />
                        <Line type="monotone" dataKey="frequency" stroke="#8884d8" strokeWidth={2} />
                      </LineChart>
                    </ResponsiveContainer>
                  </CardContent>
                </Card>
              </Grid>
            )}
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setShowColumnDetail(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    );
  };

  return (
    <Box sx={{ p: 3 }}>
      
      {!tenant?.id || !datasource?.id ? (
        <Alert severity="warning" sx={{ mb: 3 }}>
          Please select a tenant and datasource to use the profiler.
        </Alert>
      ) : (
        <Box sx={{ display: 'flex', gap: 3, flexDirection: { xs: 'column', md: 'row' } }}>
          {/* Sidebar copied from DbScanner for consistent UX; full-width on xs so it stacks above content */}
          <Box sx={{ width: { xs: '100%', md: '360px' }, flexShrink: 0 }}>
              <Paper sx={{ p: 2, mb: 2, display: 'flex', alignItems: 'center', gap: 1 }}>
                <AccountTreeIcon />
                <Box>
                  <Typography variant="h6">Database</Typography>
                  <Typography variant="body2" color="text.secondary">Schemas and tables</Typography>
                </Box>
                <Box sx={{ flex: 1 }} />
                <IconButton size="small" aria-label="refresh-schemas" onClick={async () => { setTables([]); setColumns([]); await fetchSchemas(); }}><RefreshIcon /></IconButton>
              </Paper>

              <Paper sx={{ maxHeight: '72vh', overflow: 'auto', p: 1 }}>
                {loading ? (
                  <Box sx={{ p: 3, textAlign: 'center' }}><CircularProgress size={28} /></Box>
                ) : schemas.length === 0 ? (
                  <Box sx={{ p: 2 }}><Typography>No schemas found</Typography></Box>
                ) : (
                  <List>
                    {schemas.map(schema => (
                      <div key={schema.id}>
                        <ListItem
                          secondaryAction={expandedSchemas[schema.node_name] ? <ExpandLess /> : <ExpandMore />}
                          button
                          onClick={() => toggleSchema(schema.node_name)}
                          data-testid={`schema-${schema.node_name}`}
                        >
                          <ListItemIcon>
                            <Checkbox
                              edge="start"
                              size="small"
                              checked={selectedSchemas.includes(schema.node_name)}
                              onClick={(e) => { e.stopPropagation(); toggleSchemaSelection(schema.node_name); }}
                              inputProps={{ 'aria-label': `select-schema-${schema.node_name}` }}
                            />
                          </ListItemIcon>
                          <ListItemIcon><FolderIcon fontSize="small" /></ListItemIcon>
                          <ListItemText primary={schema.node_name} />
                        </ListItem>

                        <Collapse in={Boolean(expandedSchemas[schema.node_name])} timeout="auto" unmountOnExit>
                          <List component="div" disablePadding>
                            {tables
                              .filter(t => t.qualified_path?.startsWith(`/${schema.node_name}/`))
                              .map((table) => (
                                <ListItem key={table.id} button sx={{ pl: 4 }} onClick={() => { setSelectedSchema(schema.node_name); setSelectedTable(table.node_name); setProfileResults([]); }} data-testid={`table-${table.node_name}`}>
                                  <ListItemIcon>
                                    <Checkbox
                                      edge="start"
                                      size="small"
                                      checked={selectedTables.includes(table.node_name)}
                                      onClick={(e) => { e.stopPropagation(); toggleTableSelection(table.node_name); }}
                                      inputProps={{ 'aria-label': `select-${table.node_name}` }}
                                    />
                                  </ListItemIcon>
                                  <ListItemIcon><TableChartIcon fontSize="small" /></ListItemIcon>
                                  <ListItemText primary={table.node_name} />
                                </ListItem>
                              ))}
                          </List>
                        </Collapse>
                        <Divider />
                      </div>
                    ))}
                  </List>
                )}
              </Paper>
            </Box>
          <Box sx={{ flex: 1 }}>
            {/* Run profiler action (compact) */}
            <Card sx={{ mb: 3 }}>
              <CardContent>
                <Typography variant="h6" gutterBottom>Run Profiler</Typography>
                <Grid container spacing={2} alignItems="center">
                  <Grid item xs={12}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, justifyContent: 'space-between' }}>
                      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
                        <Typography variant="body2">
                          Schemas: {selectedSchemas.length > 0 ? selectedSchemas.join(', ') : (selectedSchema ? selectedSchema : 'None')}
                        </Typography>
                        <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center', flexWrap: 'wrap' }}>
                          {selectedTables.length > 0 ? selectedTables.map(t => <Chip key={t} label={t} size="small" />) : (
                            <Typography variant="body2" color="text.secondary">No tables selected</Typography>
                          )}
                        </Box>
                      </Box>

                      <Box sx={{ marginLeft: 'auto' }}>
                        <Tooltip title="Run (press 'p')">
                          <span>
                            <Button
                              variant="contained"
                              onClick={() => void runProfiling()}
                              disabled={profiling || (selectedTables.length === 0 && selectedSchemas.length === 0 && !selectedTable)}
                              startIcon={profiling ? <CircularProgress size={20} /> : <BarChartIcon />}
                              size="small"
                              data-testid="profiler-run-button"
                              ref={(el: HTMLButtonElement | null) => { runButtonRef.current = el }}
                            >
                              {profiling ? 'Profiling...' : 'Run'}
                            </Button>
                          </span>
                        </Tooltip>
                      </Box>
                    </Box>
                  </Grid>
                </Grid>
              </CardContent>
            </Card>

              {profiling && (
                <Box sx={{ mt: 2 }}>
                  <Typography variant="body2" color="textSecondary" gutterBottom>
                    {profileProgress}
                  </Typography>
                  <LinearProgress />
                </Box>
              )}

          {/* Columns Table */}
          {selectedTable && columns.length > 0 && (
            <Card sx={{ mb: 3 }}>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Columns in {selectedSchema}.{selectedTable}
                </Typography>
                <TableContainer component={Paper} sx={{ maxHeight: 400 }}>
                  <Table stickyHeader size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Column Name</TableCell>
                        <TableCell>Data Type</TableCell>
                        <TableCell>Nullable</TableCell>
                        <TableCell>Position</TableCell>
                        <TableCell>Profiled</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {columns.map((column) => {
                const profileResult = profileResults.find(p => p.ColumnName === column.node_name || p.ColumnId === column.id);
                const highlighted = profileResults.some(p => p.ColumnId === column.id) || Boolean(profileResult);
                        return (
                          <TableRow 
                            key={column.id}
                            hover
                            style={{ cursor: profileResult ? 'pointer' : 'default', backgroundColor: highlighted ? 'rgba(76, 175, 80, 0.08)' : undefined }}
                            onClick={() => profileResult && handleColumnClick(profileResult)}
                            data-testid={`profiler-column-${column.node_name}`}
                          >
                            <TableCell sx={{ fontWeight: 500 }}>
                              {column.node_name}
                            </TableCell>
                            <TableCell>
                              <Chip 
                                label={column.properties?.data_type || 'unknown'} 
                                size="small"
                                color="secondary"
                              />
                            </TableCell>
                            <TableCell>
                              {column.properties?.is_nullable ? 'Yes' : 'No'}
                            </TableCell>
                            <TableCell>
                              {column.properties?.ordinal_position || '-'}
                            </TableCell>
                            <TableCell>
                              {profileResult ? (
                                <Chip 
                                  label={`${profileResult.Cardinality} unique`}
                                  size="small"
                                  color="success"
                                />
                              ) : (
                                <Typography variant="body2" color="textSecondary">
                                  Not profiled
                                </Typography>
                              )}
                            </TableCell>
                          </TableRow>
                        );
                      })}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          )}

          {/* Profile Results Summary */}
          {profileResults.length > 0 && (
            <Card>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Profile Summary
                  {selectedTable && selectedSchema && ` for ${selectedSchema}.${selectedTable}`}
                  {selectedSchema && !selectedTable && ` for ${selectedSchema} schema`}
                  {selectedSchemas.length > 0 && ` for ${selectedSchemas.length} schema(s)`}
                  {!selectedSchema && !selectedTable && !selectedSchemas.length && ' (All Available)'}
                </Typography>
                <Grid container spacing={2} alignItems="center">
                  <Grid item xs={12} sm={2}>
                    <Typography variant="body2" color="textSecondary">Columns</Typography>
                    <Typography variant="h5">{profileResults.length}</Typography>
                  </Grid>
                  <Grid item xs={12} sm={2}>
                    <Typography variant="body2" color="textSecondary">Avg Cardinality</Typography>
                    <Typography variant="h5">
                      {Math.round(profileResults.reduce((sum, p) => sum + p.Cardinality, 0) / profileResults.length).toLocaleString()}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={2}>
                    <Typography variant="body2" color="textSecondary">Data Types</Typography>
                    <Typography variant="h5">
                      {new Set(profileResults.map(p => p.DataType)).size}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={2}>
                    <Typography variant="body2" color="textSecondary">Tables</Typography>
                    <Typography variant="h5">
                      {new Set(profileResults.map(p => p.TableName)).size}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={2}>
                    <Typography variant="body2" color="textSecondary">Patterns Found</Typography>
                    <Typography variant="h5">
                      {profileResults.reduce((sum, p) => sum + (p.InferredPatterns?.length || 0), 0)}
                    </Typography>
                  </Grid>
                  <Grid item xs={12} sm={2}>
                    <FormControl size="small" fullWidth>
                      <InputLabel>Limit</InputLabel>
                      <Select value={limit} label="Limit" onChange={handleLimitChange}>
                        <MenuItem value={25}>25</MenuItem>
                        <MenuItem value={50}>50</MenuItem>
                        <MenuItem value={100}>100</MenuItem>
                        <MenuItem value={200}>200</MenuItem>
                      </Select>
                    </FormControl>
                  </Grid>
                  
                  <Grid item xs={12}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: 1 }}>
                      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                        <Typography variant="body2" color="textSecondary">
                          Data type breakdown:
                        </Typography>
                        <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                          {Object.entries(
                            profileResults.reduce((acc: Record<string, number>, p) => {
                              acc[p.DataType] = (acc[p.DataType] || 0) + 1;
                              return acc;
                            }, {})
                          ).map(([type, count]) => (
                            <Chip key={type} label={`${type} (${count})`} size="small" variant="outlined" />
                          ))}
                        </Box>
                      </Box>
                      
                      <Box sx={{ display: 'flex', gap: 1, alignItems: 'center' }}>
                        <Button onClick={handlePrev} disabled={page <= 0} size="small">Prev</Button>
                        <Typography variant="body2" color="textSecondary">
                          Page {page + 1}
                        </Typography>
                        <Button onClick={handleNext} disabled={!hasMore} size="small">Next</Button>
                      </Box>
                    </Box>
                    
                    <Typography variant="body2" color="textSecondary" sx={{ mt: 1 }}>
                      💡 Click any column row below to see detailed statistics and charts
                    </Typography>
                  </Grid>
                </Grid>
              </CardContent>
            </Card>
          )}
          {/* Detailed profile results list (works for multi-table runs too) */}
          {profileResults.length > 0 && selectedTable && (
            <Card sx={{ mt: 2 }}>
              <CardContent>
                <Typography variant="h6" gutterBottom>
                  Profile Results ({profileResults.length} columns)
                  {selectedTable && selectedSchema && ` for ${selectedSchema}.${selectedTable}`}
                  {selectedSchema && !selectedTable && ` for ${selectedSchema} schema`}
                  {selectedSchemas.length > 0 && ` for ${selectedSchemas.length} schema(s)`}
                </Typography>
                <TableContainer component={Paper} sx={{ maxHeight: 500 }}>
                  <Table stickyHeader size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Schema.Table.Column</TableCell>
                        <TableCell>Type</TableCell>
                        <TableCell>Cardinality</TableCell>
                        <TableCell>Length Stats</TableCell>
                        <TableCell>Patterns</TableCell>
                        <TableCell>Top Values</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {profileResults.map((p: any) => (
                        <TableRow 
                          key={`${p.Schema}.${p.TableName}.${p.ColumnName}`} 
                          data-testid={`profile-result-${p.ColumnName}`}
                          hover
                          sx={{ cursor: 'pointer' }}
                          onClick={() => handleColumnClick(p)}
                        >
                          <TableCell sx={{ fontWeight: 500 }}>
                            {p.Schema}.{p.TableName}.{p.ColumnName}
                          </TableCell>
                          <TableCell>
                            <Chip label={p.DataType} size="small" color="secondary" />
                          </TableCell>
                          <TableCell>
                            <Chip 
                              label={`${p.Cardinality} unique`} 
                              size="small" 
                              color={p.Cardinality > 1000 ? "success" : p.Cardinality > 100 ? "warning" : "default"}
                            />
                          </TableCell>
                          <TableCell>
                            {p.MinLength !== undefined && p.MaxLength !== undefined ? (
                              <Typography variant="body2" sx={{ fontSize: '0.75rem' }}>
                                {p.MinLength}-{p.MaxLength}
                                {p.AvgLength !== undefined && p.AvgLength !== null && (
                                  <span> (avg: {p.AvgLength.toFixed(1)})</span>
                                )}
                              </Typography>
                            ) : (
                              <Typography variant="body2" color="textSecondary">-</Typography>
                            )}
                          </TableCell>
                          <TableCell>
                            {p.InferredPatterns && p.InferredPatterns.length > 0 ? (
                              <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                                {p.InferredPatterns.slice(0, 2).map((pattern: string, idx: number) => (
                                  <Chip key={idx} label={pattern} size="small" variant="outlined" />
                                ))}
                                {p.InferredPatterns.length > 2 && (
                                  <Chip label={`+${p.InferredPatterns.length - 2}`} size="small" variant="outlined" />
                                )}
                              </Box>
                            ) : (
                              <Typography variant="body2" color="textSecondary">-</Typography>
                            )}
                          </TableCell>
                          <TableCell>
                            {p.FrequentValues && p.FrequentValues.length > 0 ? (
                              <Typography variant="body2" sx={{ fontSize: '0.75rem', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                                {p.FrequentValues.slice(0, 3).map((val: string) => 
                                  val === '<nil>' ? 'NULL' : val
                                ).join(', ')}
                                {p.FrequentValues.length > 3 && '...'}
                              </Typography>
                            ) : (
                              <Typography variant="body2" color="textSecondary">-</Typography>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              </CardContent>
            </Card>
          )}
          </Box>
        </Box>
      )}

      {renderColumnDetailDialog()}
      <Snackbar
        open={snackbarOpen}
        autoHideDuration={4000}
        onClose={() => setSnackbarOpen(false)}
        message={snackbarMessage}
      />
    </Box>
  );
}