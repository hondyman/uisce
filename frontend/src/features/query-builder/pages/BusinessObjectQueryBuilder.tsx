import React, { useState, useEffect } from 'react';
import axios from 'axios';
import { 
    Box, 
    Typography, 
    Button, 
    Paper, 
    List, 
    ListItem, 
    ListItemButton,
    ListItemText, 
    ListItemIcon,
    Divider, 
    CircularProgress, 
    Alert,
    TextField,
    Chip,
    IconButton,
    InputAdornment,
    Tabs,
    Tab,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Stack,
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    FormControl,
    InputLabel,
    Select,
    MenuItem,
    RadioGroup,
    FormControlLabel,
    Radio,
    Stepper,
    Step,
    StepLabel,
    Autocomplete,
    Card,
    CardContent,
    Tooltip
} from '@mui/material';
import { 
    Search as SearchIcon, 
    FilterList as FilterIcon, 
    PlayArrow as RunIcon, 
    Add as AddIcon,
    Delete as DeleteIcon,
    TableChart as TableIcon,
    Code as CodeIcon,
    Api as ApiIcon,
    Numbers as NumberIcon,
    Abc as StringIcon,
    CalendarToday as DateIcon,
    CheckCircle as BoolIcon,
    CheckCircle as CheckCircleIcon,
    Download as DownloadIcon,
    FileDownload as FileDownloadIcon,
    Tune as TuneIcon,
    SwapVert as ArrowUpDownIcon
} from '@mui/icons-material';
import {
    DndContext,
    closestCenter,
    KeyboardSensor,
    PointerSensor,
    useSensor,
    useSensors,
    DragEndEvent,
} from '@dnd-kit/core';
import {
    arrayMove,
    SortableContext,
    sortableKeyboardCoordinates,
    verticalListSortingStrategy,
    horizontalListSortingStrategy,
} from '@dnd-kit/sortable';
import {
    useSortable,
} from '@dnd-kit/sortable';
import { CSS } from '@dnd-kit/utilities';
import { Prism as SyntaxHighlighter } from 'react-syntax-highlighter';
import { vscDarkPlus } from 'react-syntax-highlighter/dist/esm/styles/prism';

import { useTenant } from '../../../contexts/TenantContext';
import { getRequiredTenantScope } from '../../../utils/tenantScope';

// --- Types ---
interface BOField {
    id: string;
    name: string;
    label: string;
    type: string;
}

interface BusinessObject {
    id: string;
    name: string;
    display_name: string;
    fields: BOField[];
}

interface Filter {
    id: string;
    field: string;
    operator: string;
    value: string;
}

// --- Sortable Chip Component ---
interface SortableChipProps {
    id: string;
    label: string;
    onDelete: () => void;
}

const SortableChip: React.FC<SortableChipProps> = ({ id, label, onDelete }) => {
    const {
        attributes,
        listeners,
        setNodeRef,
        transform,
        transition,
        isDragging,
    } = useSortable({ id });

    const style = {
        transform: CSS.Transform.toString(transform),
        transition,
        opacity: isDragging ? 0.5 : 1,
    };

    return (
        <Chip
            ref={setNodeRef}
            style={style}
            {...attributes}
            {...listeners}
            label={label}
            onDelete={onDelete}
            size="small"
            sx={{
                bgcolor: '#e3f2fd',
                color: '#1565c0',
                cursor: 'grab',
                '&:active': { cursor: 'grabbing' }
            }}
        />
    );
};

// --- Icons Helper ---
const FieldIcon = ({ type }: { type: string }) => {
    if (['integer', 'decimal', 'number'].includes(type)) return <NumberIcon fontSize="small" sx={{ color: '#4caf50' }} />;
    if (['date', 'datetime', 'timestamp'].includes(type)) return <DateIcon fontSize="small" sx={{ color: '#ff9800' }} />;
    if (['boolean'].includes(type)) return <BoolIcon fontSize="small" sx={{ color: '#9c27b0' }} />;
    return <StringIcon fontSize="small" sx={{ color: '#2196f3' }} />;
};

const BusinessObjectQueryBuilder: React.FC = () => {
    const { tenant, datasource } = useTenant();
    
    // Data State
    const [businessObjects, setBusinessObjects] = useState<BusinessObject[]>([]);
    const [selectedBO, setSelectedBO] = useState<BusinessObject | null>(null);
    const [searchTerm, setSearchTerm] = useState('');
    
    // Builder State
    const [selectedFields, setSelectedFields] = useState<string[]>([]);
    const [filters, setFilters] = useState<Filter[]>([]);
    const [filterDialogOpen, setFilterDialogOpen] = useState(false);
    const [editingFilterId, setEditingFilterId] = useState<string | null>(null);
    
    // Results State
    const [activeTab, setActiveTab] = useState(0);
    const [generatedSQL, setGeneratedSQL] = useState<string>('');
    const [queryResult, setQueryResult] = useState<any[]>([]);
    const [resultColumns, setResultColumns] = useState<Array<{ name: string; type: string }>>([]);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    // Export Wizard State
    const [exportWizardOpen, setExportWizardOpen] = useState(false);

    // Drag and Drop Sensors
    const sensors = useSensors(
        useSensor(PointerSensor),
        useSensor(KeyboardSensor, {
            coordinateGetter: sortableKeyboardCoordinates,
        })
    );

    // Initial Fetch
    useEffect(() => {
        const fetchBOs = async () => {
            let tenantId = tenant?.id;
            let datasourceId = datasource?.id;

            if (!tenantId || !datasourceId) {
                try {
                    const scope = getRequiredTenantScope();
                    tenantId = scope.tenantId;
                    datasourceId = scope.datasourceId;
                } catch (e: any) {
                    setError(e?.message || 'Tenant scope not selected');
                    return;
                }
            }
            try {
                const res = await axios.get('/api/business-objects', {
                    headers: {
                        'X-Tenant-ID': tenantId,
                        ...(datasourceId ? { 'X-Tenant-Datasource-ID': datasourceId } : {})
                    }
                });

                const data = res.data;

                // Normalization Logic (robust to different response shapes)
                let rawList: any[] = [];
                if (Array.isArray(data)) rawList = data;
                else if (data && typeof data === 'object') {
                    if (Array.isArray((data as any).businessObjects)) rawList = (data as any).businessObjects;
                    else if (Array.isArray((data as any).business_objects)) rawList = (data as any).business_objects;
                    else if (Array.isArray((data as any).items)) rawList = (data as any).items;
                    else if (Array.isArray((data as any).data)) rawList = (data as any).data;
                    else rawList = Object.values(data);
                }

                const normalizedList: BusinessObject[] = rawList
                    .filter(item => item && typeof item === 'object' && (item.id || item.name))
                    .map(item => ({
                        id: String(item.id ?? item.name),
                        name: item.name,
                        display_name: item.displayName || item.display_name || item.name,
                        fields: [] // Lazy load later
                    }));

                setBusinessObjects(normalizedList);
                if (normalizedList.length === 0) {
                    setError('No Business Objects available for the current tenant/datasource');
                }
            } catch (err) {
                console.error("Failed to fetch BOs", err);
                setError("Failed to load Business Objects");
            }
        };
        fetchBOs();
    }, [tenant, datasource]);

    // Independent BO details fetcher
    const loadBODetails = async (bo: BusinessObject) => {
        if (bo.fields.length > 0) return bo; // Already loaded

        try {
            const res = await axios.get(`/api/business-objects/${bo.id}`, {
                headers: { 'X-Tenant-ID': tenant?.id, 'X-Tenant-Datasource-ID': datasource?.id }
            });
            const fullBO = res.data;
            const activeFields = [...(fullBO.coreFields || []), ...(fullBO.customFields || [])];
            return {
                ...bo,
                fields: activeFields.map((f: any) => ({
                    id: f.id,
                    name: f.name || f.key,
                    label: f.displayName || f.name,
                    type: f.type || 'string'
                }))
            };
        } catch (e) {
            console.error(e);
            return bo;
        }
    };

    const handleBOSelect = async (bo: BusinessObject) => {
        setLoading(true);
        const detailedBO = await loadBODetails(bo);
        setSelectedBO(detailedBO);
        setSelectedFields([]);
        setFilters([]);
        setGeneratedSQL('');
        setQueryResult([]);
        setLoading(false);
    };

    const toggleField = (fieldId: string) => {
        setSelectedFields(prev => 
            prev.includes(fieldId) ? prev.filter(f => f !== fieldId) : [...prev, fieldId]
        );
    };

    const getFieldLabel = (fieldId: string) =>
        selectedBO?.fields.find(f => f.id === fieldId)?.label || fieldId;

    // Drag and Drop Handler
    const handleDragEnd = (event: DragEndEvent) => {
        const { active, over } = event;

        if (over && active.id !== over.id) {
            setSelectedFields((items) => {
                const oldIndex = items.indexOf(active.id as string);
                const newIndex = items.indexOf(over.id as string);

                return arrayMove(items, oldIndex, newIndex);
            });
        }
    };

    const runQuery = async () => {
        if (!selectedBO || selectedFields.length === 0) return;
        setLoading(true);
        setError(null);
        
        try {
            // Build WHERE clause from filters
            const whereClause = filters
                .filter(f => f.field && f.operator && (f.value || ['is_null', 'is_not_null'].includes(f.operator)))
                .map(f => {
                    switch (f.operator) {
                        case 'equals':
                            return `${f.field} = '${f.value}'`;
                        case 'not_equals':
                            return `${f.field} != '${f.value}'`;
                        case 'contains':
                            return `${f.field} LIKE '%${f.value}%'`;
                        case 'starts_with':
                            return `${f.field} LIKE '${f.value}%'`;
                        case 'ends_with':
                            return `${f.field} LIKE '%${f.value}'`;
                        case 'greater_than':
                            return `${f.field} > ${isNaN(Number(f.value)) ? `'${f.value}'` : f.value}`;
                        case 'less_than':
                            return `${f.field} < ${isNaN(Number(f.value)) ? `'${f.value}'` : f.value}`;
                        case 'greater_or_equal':
                            return `${f.field} >= ${isNaN(Number(f.value)) ? `'${f.value}'` : f.value}`;
                        case 'less_or_equal':
                            return `${f.field} <= ${isNaN(Number(f.value)) ? `'${f.value}'` : f.value}`;
                        case 'is_null':
                            return `${f.field} IS NULL`;
                        case 'is_not_null':
                            return `${f.field} IS NOT NULL`;
                        default:
                            return '';
                    }
                })
                .filter(Boolean);

            // 1. Generate SQL
            const payload = {
                businessObjectId: selectedBO.id,
                selectedFields: selectedFields,
                whereClause: whereClause.length > 0 ? whereClause.join(' AND ') : null,
                limit: 100
            };
            const sqlRes = await axios.post('/api/business-objects/generate-sql', payload, {
                headers: { 'X-Tenant-ID': tenant?.id }
            });
            const generatedSql = sqlRes.data.sql;
            setGeneratedSQL(generatedSql);
            
            // 2. Execute the generated SQL with auto-routing based on BO's datasource
            const execRes = await axios.post('/api/business-objects/execute-sql', {
                sql: generatedSql,
                limit: 100,
                business_object_id: selectedBO.id  // Pass BO ID for automatic datasource routing
            }, {
                headers: { 'X-Tenant-ID': tenant?.id }
            });
            
            if (execRes.data.error) {
                setError(execRes.data.error);
                setQueryResult([]);
                setResultColumns([]);
            } else {
                setQueryResult(execRes.data.rows || []);
                setResultColumns(execRes.data.columns || []);
            }
            
            setActiveTab(0); // Switch to Data tab
        } catch (err: any) {
            console.error("Query failed", err);
            setError(err.response?.data?.error || err.message || "Failed to execute query");
            setQueryResult([]);
        } finally {
            setLoading(false);
        }
    };

    // Filtered Fields for Sidebar
    const visibleFields = selectedBO?.fields.filter(f => 
        f.label.toLowerCase().includes(searchTerm.toLowerCase()) || 
        f.name.toLowerCase().includes(searchTerm.toLowerCase())
    ) || [];

    return (
        <Box sx={{ display: 'flex', height: 'calc(100vh - 64px)', bgcolor: '#f5f5f5' }}>
            
            {/* Left Pane: Sidebar (BOs & Fields) */}
            <Paper sx={{ width: 320, display: 'flex', flexDirection: 'column', borderRight: '1px solid #ddd', borderRadius: 0 }}>
                {/* BO Selector */}
                <Box sx={{ p: 2, borderBottom: '1px solid #eee' }}>
                    <Typography variant="overline" color="text.secondary">Subject Area</Typography>
                    <TextField
                        select
                        fullWidth
                        size="small"
                        value={selectedBO?.id || ''}
                        onChange={(e) => {
                            const selectedId = e.target.value;
                            const bo = businessObjects.find(b => b.id === selectedId);
                            if (bo) handleBOSelect(bo);
                        }}
                        SelectProps={{ native: true }}
                        inputProps={{ 'aria-label': 'Subject Area' }}
                        sx={{ mt: 1 }}
                    >
                        <option value="" disabled>Select Business Object...</option>
                        {businessObjects.map(bo => (
                            <option key={bo.id} value={bo.id}>{bo.display_name}</option>
                        ))}
                    </TextField>
                    {error && (
                        <Typography variant="caption" color="error" sx={{ display: 'block', mt: 1 }}>
                            {error}
                        </Typography>
                    )}
                </Box>

                {/* Field Search & List */}
                <Box sx={{ p: 2, pb: 1 }}>
                    <TextField
                        fullWidth
                        size="small"
                        placeholder="Search fields..."
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                        InputProps={{
                            startAdornment: <InputAdornment position="start"><SearchIcon fontSize="small" /></InputAdornment>
                        }}
                    />
                </Box>
                
                <List dense sx={{ flex: 1, overflow: 'auto', px: 1 }}>
                    {loading && !selectedBO && <Box sx={{ p: 2, textAlign: 'center' }}><CircularProgress size={20} /></Box>}
                    
                    {selectedBO && visibleFields.map(field => (
                        <ListItemButton 
                            key={field.id} 
                            onClick={() => toggleField(field.id)}
                            selected={selectedFields.includes(field.id)}
                            sx={{ borderRadius: 1, mb: 0.5 }}
                        >
                            <ListItemIcon sx={{ minWidth: 32 }}>
                                <FieldIcon type={field.type} />
                            </ListItemIcon>
                            <ListItemText 
                                primary={field.label} 
                                secondary={field.name} 
                                primaryTypographyProps={{ variant: 'body2' }}
                                secondaryTypographyProps={{ variant: 'caption', sx: { fontSize: '0.65rem' } }}
                            />
                            {selectedFields.includes(field.id) && <CheckCircleIcon fontSize="small" color="primary" sx={{ ml: 1 }} />} 
                            {/* CheckCircleIcon alias defined above as BoolIcon, but lets fix import collision manually: using pure CSS/Icon */}
                            {selectedFields.includes(field.id) && (
                                <Box sx={{ width: 8, height: 8, borderRadius: '50%', bgcolor: 'primary.main' }} />
                            )}
                        </ListItemButton>
                    ))}
                </List>
            </Paper>

            {/* Middle Pane: Query Builder Canvas */}
            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', p: 2, overflow: 'hidden' }}>
                <Paper sx={{ p: 2, mb: 2, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Box>
                        <Typography variant="h6">Untitled Query</Typography>
                        <Typography variant="caption" color="text.secondary">
                            {selectedFields.length} fields selected
                        </Typography>
                    </Box>
                    <Box>
                        {/* <Button startIcon={<FilterIcon />} sx={{ mr: 1 }}>Filter</Button> */}
                        <Button 
                            variant="contained" 
                            color="primary" 
                            startIcon={<RunIcon />}
                            onClick={runQuery}
                            disabled={!selectedBO || selectedFields.length === 0 || loading}
                        >
                            Run Query
                        </Button>
                    </Box>
                </Paper>

                {/* Selected Columns & Filters Area */}
                <Paper sx={{ p: 0, mb: 2, minHeight: 120, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
                     <Box sx={{ p: 1, bgcolor: '#f9f9f9', borderBottom: '1px solid #eee', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <Typography variant="caption" fontWeight="bold" color="text.secondary">SELECTED COLUMNS</Typography>
                        {queryResult.length > 0 && (
                            <Button
                                size="small"
                                startIcon={<FileDownloadIcon />}
                                onClick={() => setExportWizardOpen(true)}
                                sx={{ fontSize: '0.75rem' }}
                            >
                                Export
                            </Button>
                        )}
                     </Box>
                     <Box sx={{ p: 2, overflow: 'auto' }}>
                        {selectedFields.length === 0 && (
                            <Typography variant="body2" color="text.secondary" fontStyle="italic">
                                Select fields from the sidebar to modify your query...
                            </Typography>
                        )}
                        {selectedFields.length > 0 && (
                            <DndContext
                                sensors={sensors}
                                collisionDetection={closestCenter}
                                onDragEnd={handleDragEnd}
                            >
                                <SortableContext items={selectedFields} strategy={horizontalListSortingStrategy}>
                                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                                        {selectedFields.map(f => (
                                            <SortableChip
                                                key={f}
                                                id={f}
                                                label={getFieldLabel(f)}
                                                onDelete={() => toggleField(f)}
                                            />
                                        ))}
                                    </Box>
                                </SortableContext>
                            </DndContext>
                        )}
                     </Box>
                </Paper>

                {/* WHERE Clause Filter Builder */}
                <Paper sx={{ p: 3, mb: 2 }}>
                  <Typography variant="h6" gutterBottom>WHERE Clause Builder</Typography>
                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    Add filter conditions to narrow down your query results
                  </Typography>
                  
                  {/* Filter Editor */}
                  {filters.length === 0 ? (
                    <Button
                      variant="outlined"
                      startIcon={<AddIcon />}
                      onClick={() => {
                        setFilters([{ id: `filter-${Date.now()}`, field: '', operator: 'equals', value: '' }]);
                        setFilterDialogOpen(true);
                      }}
                      size="small"
                    >
                      Add Filter
                    </Button>
                  ) : (
                    <>
                      {filters.map((filter) => (
                        <Card key={filter.id} variant="outlined" sx={{ mb: 2 }}>
                          <CardContent sx={{ p: 2, '&:last-child': { pb: 2 } }}>
                            <Stack direction="row" spacing={1} alignItems="center">
                              <Autocomplete
                                options={selectedBO?.fields?.map(f => f.name) || []}
                                value={filter.field || null}
                                onChange={(_, val) => {
                                  setFilters(filters.map(f => 
                                    f.id === filter.id ? { ...f, field: val || '' } : f
                                  ));
                                }}
                                isOptionEqualToValue={(option, value) => option === value}
                                renderInput={(params) => <TextField {...params} label="Field" size="small" />}
                                sx={{ flex: 1 }}
                                noOptionsText="No fields"
                              />

                                                            <FormControl size="small" sx={{ minWidth: 100 }}>
                                                                <InputLabel id={`operator-label-${filter.id}`}>Operator</InputLabel>
                                <Select
                                                                    labelId={`operator-label-${filter.id}`}
                                  aria-label="Operator"
                                  value={filter.operator}
                                  onChange={(e) => {
                                    setFilters(filters.map(f => 
                                      f.id === filter.id ? { ...f, operator: e.target.value } : f
                                    ));
                                  }}
                                  label="Operator"
                                >
                                  <MenuItem value="equals">=</MenuItem>
                                  <MenuItem value="not_equals">!=</MenuItem>
                                  <MenuItem value="contains">LIKE</MenuItem>
                                  <MenuItem value="greater_than">&gt;</MenuItem>
                                  <MenuItem value="less_than">&lt;</MenuItem>
                                  <MenuItem value="greater_or_equal">&gt;=</MenuItem>
                                  <MenuItem value="less_or_equal">&lt;=</MenuItem>
                                </Select>
                              </FormControl>

                              <TextField
                                value={filter.value}
                                onChange={(e) => {
                                  setFilters(filters.map(f => 
                                    f.id === filter.id ? { ...f, value: e.target.value } : f
                                  ));
                                }}
                                placeholder="Value"
                                size="small"
                                sx={{ flex: 1 }}
                              />

                              <IconButton
                                size="small"
                                onClick={() => setFilters(filters.filter(f => f.id !== filter.id))}
                              >
                                <DeleteIcon fontSize="small" />
                              </IconButton>
                            </Stack>
                          </CardContent>
                        </Card>
                      ))}

                      <Stack direction="row" spacing={1} sx={{ mt: 2 }}>
                        <Button
                          variant="outlined"
                          size="small"
                          startIcon={<AddIcon />}
                          onClick={() => {
                            setFilters([...filters, { id: `filter-${Date.now()}`, field: '', operator: 'equals', value: '' }]);
                          }}
                        >
                          Add Another Filter
                        </Button>
                        <Button
                          size="small"
                          variant="text"
                          color="error"
                          onClick={() => setFilters([])}
                        >
                          Clear All
                        </Button>
                      </Stack>
                    </>
                  )}
                </Paper>

                 {/* Results Area */}
                 <Paper sx={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
                    <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
                        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} textColor="primary" indicatorColor="primary">
                            <Tab icon={<TableIcon fontSize="small"/>} label="Results" iconPosition="start" />
                            <Tab icon={<CodeIcon fontSize="small"/>} label="Generated SQL" iconPosition="start" />
                            <Tab icon={<ApiIcon fontSize="small"/>} label="JSON / API" iconPosition="start" />
                        </Tabs>
                    </Box>
                    
                    <Box sx={{ flex: 1, overflow: 'auto', p: 0, position: 'relative' }}>
                        {loading && (
                            <Box sx={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', bgcolor: 'rgba(255,255,255,0.7)', zIndex: 10 }}>
                                <CircularProgress />
                            </Box>
                        )}
                        
                        {error && (
                            <Box sx={{ p: 2 }}>
                                <Alert severity="error">{error}</Alert>
                            </Box>
                        )}

                        {/* Tab 0: Results Table */}
                        {activeTab === 0 && (
                            <Box sx={{ p: 0, height: '100%' }}>
                                {queryResult.length > 0 ? (
                                    <TableContainer sx={{ height: '100%' }}>
                                        <Table stickyHeader size="small">
                                            <TableHead>
                                                <TableRow>
                                                    {resultColumns.map(col => (
                                                        <TableCell key={col.name} sx={{ fontWeight: 'bold', bgcolor: '#f9f9f9' }}>
                                                            {col.name}
                                                        </TableCell>
                                                    ))}
                                                </TableRow>
                                            </TableHead>
                                            <TableBody>
                                                {queryResult.map((row, i) => (
                                                    <TableRow key={i} hover>
                                                        {resultColumns.map(col => (
                                                            <TableCell key={col.name}>{row[col.name] || '-'}</TableCell>
                                                        ))}
                                                    </TableRow>
                                                ))}
                                            </TableBody>
                                        </Table>
                                    </TableContainer>
                                ) : (
                                    <Box sx={{ p: 4, textAlign: 'center', color: 'text.secondary' }}>
                                        <Typography>No results to display. Run a query to see data.</Typography>
                                    </Box>
                                )}
                            </Box>
                        )}

                        {/* Tab 1: SQL */}
                        {activeTab === 1 && (
                            <Box sx={{ height: '100%', bgcolor: '#1e1e1e', fontSize: '13px' }}>
                                <SyntaxHighlighter 
                                    language="sql" 
                                    style={vscDarkPlus} 
                                    customStyle={{ margin: 0, height: '100%', padding: '16px', background: 'transparent' }}
                                >
                                    {generatedSQL || '-- No SQL generated yet'}
                                </SyntaxHighlighter>
                            </Box>
                        )}

                        {/* Tab 2: API */}
                        {activeTab === 2 && (
                            <Box sx={{ p: 2 }}>
                                <Typography variant="h6" gutterBottom>REST API Endpoint</Typography>
                                <Paper variant="outlined" sx={{ p: 2, mb: 2, bgcolor: '#f5f5f5', fontFamily: 'monospace' }}>
                                    POST /api/business-objects/generate-sql
                                </Paper>
                                <Typography variant="subtitle2" gutterBottom>Request Payload</Typography>
                                <SyntaxHighlighter language="json" style={vscDarkPlus} customStyle={{ borderRadius: '4px' }}>
                                    {JSON.stringify({
                                        business_object_id: selectedBO?.id,
                                        selected_fields: selectedFields,
                                        filters: filters,
                                        limit: 100
                                    }, null, 2)}
                                </SyntaxHighlighter>
                            </Box>
                        )}
                    </Box>
                 </Paper>
            </Box>

            {/* Export Wizard */}
            <ExportWizard
                open={exportWizardOpen}
                onClose={() => setExportWizardOpen(false)}
                data={queryResult}
                columns={resultColumns}
                queryName={selectedBO?.display_name}
            />

        </Box>
    );
};

// --- Export Wizard Component ---
interface ExportWizardProps {
    open: boolean;
    onClose: () => void;
    data: any[];
    columns: Array<{ name: string; type: string }>;
    queryName?: string;
}

const ExportWizard: React.FC<ExportWizardProps> = ({ open, onClose, data, columns, queryName = "Query Results" }) => {
    const [activeStep, setActiveStep] = useState(0);
    const [exportFormat, setExportFormat] = useState<'csv' | 'xml' | 'json'>('csv');
    const [includeHeaders, setIncludeHeaders] = useState(true);
    const [fileName, setFileName] = useState(`${queryName.replace(/[^a-zA-Z0-9]/g, '_')}_${new Date().toISOString().split('T')[0]}`);

    const steps = ['Format Selection', 'Options', 'Download'];

    const handleNext = () => {
        setActiveStep((prev) => prev + 1);
    };

    const handleBack = () => {
        setActiveStep((prev) => prev - 1);
    };

    const handleDownload = () => {
        let content = '';
        let mimeType = '';
        let extension = '';

        switch (exportFormat) {
            case 'csv':
                content = exportToCSV(data, columns, includeHeaders);
                mimeType = 'text/csv';
                extension = 'csv';
                break;
            case 'xml':
                content = exportToXML(data, columns, queryName);
                mimeType = 'application/xml';
                extension = 'xml';
                break;
            case 'json':
                content = exportToJSON(data, columns);
                mimeType = 'application/json';
                extension = 'json';
                break;
        }

        const blob = new Blob([content], { type: mimeType });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${fileName}.${extension}`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        onClose();
        setActiveStep(0);
    };

    const exportToCSV = (data: any[], columns: Array<{ name: string; type: string }>, includeHeaders: boolean): string => {
        const headers = columns.map(col => `"${col.name}"`).join(',');
        const rows = data.map(row =>
            columns.map(col => {
                const value = row[col.name] || '';
                // Escape quotes and wrap in quotes if contains comma, quote, or newline
                const stringValue = String(value);
                if (stringValue.includes(',') || stringValue.includes('"') || stringValue.includes('\n')) {
                    return `"${stringValue.replace(/"/g, '""')}"`;
                }
                return stringValue;
            }).join(',')
        );

        return includeHeaders ? [headers, ...rows].join('\n') : rows.join('\n');
    };

    const exportToXML = (data: any[], columns: Array<{ name: string; type: string }>, rootName: string): string => {
        const escapeXml = (str: string) => {
            return str.replace(/[<>&'"]/g, (char) => {
                const xmlChars: { [key: string]: string } = {
                    '<': '&lt;',
                    '>': '&gt;',
                    '&': '&amp;',
                    "'": '&apos;',
                    '"': '&quot;'
                };
                return xmlChars[char] || char;
            });
        };

        const rootElement = rootName.replace(/[^a-zA-Z0-9]/g, '_');
        let xml = `<?xml version="1.0" encoding="UTF-8"?>\n<${rootElement}>\n`;

        data.forEach((row, index) => {
            xml += `  <row id="${index + 1}">\n`;
            columns.forEach(col => {
                const value = row[col.name] || '';
                xml += `    <${col.name}>${escapeXml(String(value))}</${col.name}>\n`;
            });
            xml += `  </row>\n`;
        });

        xml += `</${rootElement}>`;
        return xml;
    };

    const exportToJSON = (data: any[], columns: Array<{ name: string; type: string }>): string => {
        // Clean the data to only include selected columns
        const cleanData = data.map(row => {
            const cleanRow: any = {};
            columns.forEach(col => {
                cleanRow[col.name] = row[col.name] || null;
            });
            return cleanRow;
        });

        return JSON.stringify(cleanData, null, 2);
    };

    return (
        <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
            <DialogTitle>Export Query Results</DialogTitle>
            <DialogContent>
                <Stepper activeStep={activeStep} sx={{ mb: 3 }}>
                    {steps.map((label) => (
                        <Step key={label}>
                            <StepLabel>{label}</StepLabel>
                        </Step>
                    ))}
                </Stepper>

                {activeStep === 0 && (
                    <Box>
                        <Typography variant="h6" gutterBottom>Choose Export Format</Typography>
                        <FormControl component="fieldset">
                            <RadioGroup
                                value={exportFormat}
                                onChange={(e) => setExportFormat(e.target.value as 'csv' | 'xml' | 'json')}
                            >
                                <FormControlLabel value="csv" control={<Radio />} label="CSV (Comma Separated Values)" />
                                <FormControlLabel value="xml" control={<Radio />} label="XML (Extensible Markup Language)" />
                                <FormControlLabel value="json" control={<Radio />} label="JSON (JavaScript Object Notation)" />
                            </RadioGroup>
                        </FormControl>
                    </Box>
                )}

                {activeStep === 1 && (
                    <Box>
                        <Typography variant="h6" gutterBottom>Export Options</Typography>
                        <TextField
                            fullWidth
                            label="File Name"
                            value={fileName}
                            onChange={(e) => setFileName(e.target.value)}
                            sx={{ mb: 2 }}
                        />
                        {exportFormat === 'csv' && (
                            <FormControlLabel
                                control={
                                    <Radio
                                        checked={includeHeaders}
                                        onChange={(e) => setIncludeHeaders(e.target.checked)}
                                    />
                                }
                                label="Include column headers"
                            />
                        )}
                        <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
                            {data.length} rows will be exported
                        </Typography>
                    </Box>
                )}

                {activeStep === 2 && (
                    <Box>
                        <Typography variant="h6" gutterBottom>Ready to Download</Typography>
                        <Typography>
                            Format: {exportFormat.toUpperCase()}
                        </Typography>
                        <Typography>
                            File: {fileName}.{exportFormat}
                        </Typography>
                        <Typography>
                            Records: {data.length}
                        </Typography>
                    </Box>
                )}
            </DialogContent>
            <DialogActions>
                <Button onClick={onClose}>Cancel</Button>
                {activeStep > 0 && <Button onClick={handleBack}>Back</Button>}
                {activeStep < steps.length - 1 ? (
                    <Button onClick={handleNext} variant="contained">Next</Button>
                ) : (
                    <Button onClick={handleDownload} variant="contained" startIcon={<DownloadIcon />}>
                        Download
                    </Button>
                )}
            </DialogActions>
        </Dialog>
    );
};

export default BusinessObjectQueryBuilder;
