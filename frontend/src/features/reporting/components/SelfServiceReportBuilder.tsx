import React, { useState, useEffect, useMemo } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Button,
  Grid,
  List,
  ListItemButton,
  ListItemText,
  ListItemIcon,
  Chip,
  Paper,
  TextField,
  Alert,
  Divider,
  Stack,
  IconButton,
  Tooltip,
  Collapse,
  Fade,
  Zoom,
  Avatar,
  Badge,
  InputAdornment,
  Menu,
  MenuItem,
  Select,
  FormControl,
  InputLabel,
  Switch,
  FormControlLabel,
  Slider,
  Tabs,
  Tab,
  LinearProgress,
  alpha,
} from '@mui/material';
import {
  Business as BusinessObjectIcon,
  Category as SemanticModelIcon,
  ViewColumn as FieldIcon,
  Functions as MeasureIcon,
  CheckCircle as CheckIcon,
  ArrowBack as BackIcon,
  ArrowForward as ForwardIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  BarChart as BarChartIcon,
  ShowChart as LineChartIcon,
  PieChart as PieChartIcon,
  TableChart as TableIcon,
  Close as CloseIcon,
  Search as SearchIcon,
  DragIndicator as DragIcon,
  FilterList as FilterIcon,
  Sort as SortIcon,
  Visibility as PreviewIcon,
  Save as SaveIcon,
  PlayArrow as RunIcon,
  Schedule as ScheduleIcon,
  Share as ShareIcon,
  Bookmark as BookmarkIcon,
  ExpandMore as ExpandIcon,
  ExpandLess as CollapseIcon,
  Star as StarIcon,
  Abc as TextIcon,
  Numbers as NumberIcon,
  CalendarMonth as DateIcon,
  ToggleOn as BooleanIcon,
  Key as KeyIcon,
  Link as LinkIcon,
  Tune as TuneIcon,
  AutoAwesome as AIIcon,
  Lightbulb as SuggestIcon,
  ContentCopy as CopyIcon,
  Download as DownloadIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';
import { useTenant } from '../../../contexts/TenantContext';
import { useNavigate } from 'react-router-dom';

// ============================================================================
// WORLD-CLASS REPORT BUILDER
// Premium Workday/Tableau-level experience
// ============================================================================

interface DataSource {
  id: string;
  name: string;
  displayName: string;
  description?: string;
  type: 'business_object' | 'semantic_model';
  icon: 'customer' | 'order' | 'product' | 'employee' | 'analytics' | 'performance';
  fields: Field[];
  measures?: Measure[];
  rowCount?: number;
}

interface Field {
  id: string;
  name: string;
  label: string;
  type: 'string' | 'number' | 'date' | 'boolean' | 'currency';
  isPrimaryKey?: boolean;
  isForeignKey?: boolean;
  isRequired?: boolean;
  description?: string;
}

interface Measure {
  id: string;
  name: string;
  label: string;
  aggregation: 'SUM' | 'COUNT' | 'AVG' | 'MIN' | 'MAX';
  format?: 'number' | 'currency' | 'percent';
  description?: string;
}

interface SelectedField {
  field: Field;
  sortDirection?: 'asc' | 'desc';
  aggregation?: string;
}

interface FilterCondition {
  id: string;
  field: Field;
  operator: 'equals' | 'not_equals' | 'contains' | 'starts_with' | 'greater_than' | 'less_than' | 'between' | 'in';
  value: string;
  value2?: string; // For 'between'
}

interface ReportConfig {
  name: string;
  description: string;
  dataSource: DataSource | null;
  columns: SelectedField[];
  measures: Measure[];
  filters: FilterCondition[];
  groupBy: Field[];
  sortBy: { field: Field; direction: 'asc' | 'desc' }[];
  limit: number;
  chartType: 'table' | 'bar' | 'line' | 'pie' | 'area' | 'scatter';
  showTotals: boolean;
  showPercentages: boolean;
}

// Premium color palette
const COLORS = {
  primary: '#1976d2',
  secondary: '#9c27b0',
  success: '#2e7d32',
  warning: '#ed6c02',
  info: '#0288d1',
  gradient: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)',
  cardBg: 'rgba(255, 255, 255, 0.9)',
  glassBg: 'rgba(255, 255, 255, 0.7)',
};

const getFieldTypeIcon = (type: string) => {
  switch (type) {
    case 'string': return <TextIcon fontSize="small" />;
    case 'number': return <NumberIcon fontSize="small" />;
    case 'currency': return <NumberIcon fontSize="small" color="success" />;
    case 'date': return <DateIcon fontSize="small" />;
    case 'boolean': return <BooleanIcon fontSize="small" />;
    default: return <FieldIcon fontSize="small" />;
  }
};

const getFieldTypeColor = (type: string) => {
  switch (type) {
    case 'string': return '#1976d2';
    case 'number': return '#2e7d32';
    case 'currency': return '#ed6c02';
    case 'date': return '#9c27b0';
    case 'boolean': return '#0288d1';
    default: return '#757575';
  }
};

// Sample data sources
const SAMPLE_DATA_SOURCES: DataSource[] = [
  {
    id: 'ds_customers',
    name: 'customers',
    displayName: 'Customers',
    description: 'Customer master data including contacts and addresses',
    type: 'business_object',
    icon: 'customer',
    rowCount: 91,
    fields: [
      { id: 'f1', name: 'customer_id', label: 'Customer ID', type: 'string', isPrimaryKey: true },
      { id: 'f2', name: 'company_name', label: 'Company Name', type: 'string', isRequired: true },
      { id: 'f3', name: 'contact_name', label: 'Contact Name', type: 'string' },
      { id: 'f4', name: 'contact_title', label: 'Contact Title', type: 'string' },
      { id: 'f5', name: 'city', label: 'City', type: 'string' },
      { id: 'f6', name: 'region', label: 'Region', type: 'string' },
      { id: 'f7', name: 'country', label: 'Country', type: 'string' },
      { id: 'f8', name: 'phone', label: 'Phone', type: 'string' },
      { id: 'f9', name: 'fax', label: 'Fax', type: 'string' },
    ],
  },
  {
    id: 'ds_orders',
    name: 'orders',
    displayName: 'Orders',
    description: 'Order transactions with shipping information',
    type: 'business_object',
    icon: 'order',
    rowCount: 830,
    fields: [
      { id: 'f10', name: 'order_id', label: 'Order ID', type: 'number', isPrimaryKey: true },
      { id: 'f11', name: 'customer_id', label: 'Customer ID', type: 'string', isForeignKey: true },
      { id: 'f12', name: 'employee_id', label: 'Employee ID', type: 'number', isForeignKey: true },
      { id: 'f13', name: 'order_date', label: 'Order Date', type: 'date', isRequired: true },
      { id: 'f14', name: 'required_date', label: 'Required Date', type: 'date' },
      { id: 'f15', name: 'shipped_date', label: 'Shipped Date', type: 'date' },
      { id: 'f16', name: 'freight', label: 'Freight', type: 'currency' },
      { id: 'f17', name: 'ship_name', label: 'Ship Name', type: 'string' },
      { id: 'f18', name: 'ship_city', label: 'Ship City', type: 'string' },
      { id: 'f19', name: 'ship_country', label: 'Ship Country', type: 'string' },
    ],
  },
  {
    id: 'ds_products',
    name: 'products',
    displayName: 'Products',
    description: 'Product catalog with pricing and inventory',
    type: 'business_object',
    icon: 'product',
    rowCount: 77,
    fields: [
      { id: 'f20', name: 'product_id', label: 'Product ID', type: 'number', isPrimaryKey: true },
      { id: 'f21', name: 'product_name', label: 'Product Name', type: 'string', isRequired: true },
      { id: 'f22', name: 'supplier_id', label: 'Supplier ID', type: 'number', isForeignKey: true },
      { id: 'f23', name: 'category_id', label: 'Category ID', type: 'number', isForeignKey: true },
      { id: 'f25', name: 'units_in_stock', label: 'Units in Stock', type: 'number' },
      { id: 'f26', name: 'units_on_order', label: 'Units on Order', type: 'number' },
      { id: 'f27', name: 'reorder_level', label: 'Reorder Level', type: 'number' },
      { id: 'f28', name: 'discontinued', label: 'Discontinued', type: 'boolean' },
    ],
  },
  {
    id: 'ds_sales_analytics',
    name: 'sales_analytics',
    displayName: 'Sales Analytics',
    description: 'Pre-aggregated sales metrics and KPIs',
    type: 'semantic_model',
    icon: 'analytics',
    fields: [
      { id: 'f30', name: 'order_date', label: 'Order Date', type: 'date' },
      { id: 'f31', name: 'customer_name', label: 'Customer', type: 'string' },
      { id: 'f32', name: 'product_name', label: 'Product', type: 'string' },
      { id: 'f33', name: 'category', label: 'Category', type: 'string' },
      { id: 'f34', name: 'country', label: 'Country', type: 'string' },
      { id: 'f35', name: 'region', label: 'Region', type: 'string' },
    ],
    measures: [
      { id: 'm1', name: 'total_revenue', label: 'Total Revenue', aggregation: 'SUM', format: 'currency' },
      { id: 'm2', name: 'order_count', label: 'Order Count', aggregation: 'COUNT', format: 'number' },
      { id: 'm3', name: 'avg_order_value', label: 'Avg Order Value', aggregation: 'AVG', format: 'currency' },
      { id: 'm4', name: 'units_sold', label: 'Units Sold', aggregation: 'SUM', format: 'number' },
      { id: 'm5', name: 'customer_count', label: 'Unique Customers', aggregation: 'COUNT', format: 'number' },
    ],
  },
  {
    id: 'ds_product_performance',
    name: 'product_performance',
    displayName: 'Product Performance',
    description: 'Product sales and inventory analytics',
    type: 'semantic_model',
    icon: 'performance',
    fields: [
      { id: 'f40', name: 'product_name', label: 'Product', type: 'string' },
      { id: 'f41', name: 'category', label: 'Category', type: 'string' },
      { id: 'f42', name: 'supplier', label: 'Supplier', type: 'string' },
      { id: 'f43', name: 'discontinued', label: 'Status', type: 'boolean' },
    ],
    measures: [
      { id: 'm10', name: 'revenue', label: 'Revenue', aggregation: 'SUM', format: 'currency' },
      { id: 'm11', name: 'units_sold', label: 'Units Sold', aggregation: 'SUM', format: 'number' },
      { id: 'm12', name: 'stock_level', label: 'Current Stock', aggregation: 'SUM', format: 'number' },
      { id: 'm13', name: 'reorder_needed', label: 'Needs Reorder', aggregation: 'COUNT', format: 'number' },
    ],
  },
];

export const WorldClassReportBuilder: React.FC = () => {
  const { tenant } = useTenant();
  const navigate = useNavigate();
  
  const [activeStep, setActiveStep] = useState(0);
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedSource, setExpandedSource] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [showPreview, setShowPreview] = useState(true);
  const [previewData, setPreviewData] = useState<any[]>([]);
  
  const [config, setConfig] = useState<ReportConfig>({
    name: '',
    description: '',
    dataSource: null,
    columns: [],
    measures: [],
    filters: [],
    groupBy: [],
    sortBy: [],
    limit: 100,
    chartType: 'table',
    showTotals: true,
    showPercentages: false,
  });

  // Filter data sources by search
  const filteredSources = useMemo(() => {
    if (!searchQuery) return SAMPLE_DATA_SOURCES;
    const query = searchQuery.toLowerCase();
    return SAMPLE_DATA_SOURCES.filter(
      ds => ds.displayName.toLowerCase().includes(query) ||
            ds.description?.toLowerCase().includes(query) ||
            ds.fields.some(f => f.label.toLowerCase().includes(query))
    );
  }, [searchQuery]);

  // Generate preview data
  useEffect(() => {
    if (config.dataSource && config.columns.length > 0) {
      generatePreviewData();
    }
  }, [config.dataSource, config.columns, config.measures]);

  const generatePreviewData = () => {
    // Mock preview data based on selected columns
    const mockData = [];
    for (let i = 0; i < 5; i++) {
      const row: any = {};
      config.columns.forEach(col => {
        switch (col.field.type) {
          case 'string':
            row[col.field.name] = `Sample ${col.field.label} ${i + 1}`;
            break;
          case 'number':
            row[col.field.name] = Math.floor(Math.random() * 1000);
            break;
          case 'currency':
            row[col.field.name] = `$${(Math.random() * 10000).toFixed(2)}`;
            break;
          case 'date':
            row[col.field.name] = new Date(Date.now() - Math.random() * 86400000 * 30).toLocaleDateString();
            break;
          case 'boolean':
            row[col.field.name] = Math.random() > 0.5 ? 'Yes' : 'No';
            break;
        }
      });
      config.measures.forEach(m => {
        row[m.name] = m.format === 'currency' 
          ? `$${(Math.random() * 100000).toFixed(2)}`
          : Math.floor(Math.random() * 10000);
      });
      mockData.push(row);
    }
    setPreviewData(mockData);
  };

  const handleSelectDataSource = (source: DataSource) => {
    setConfig(prev => ({
      ...prev,
      dataSource: source,
      columns: [],
      measures: [],
      filters: [],
    }));
    setActiveStep(1);
  };

  const handleToggleField = (field: Field) => {
    setConfig(prev => {
      const exists = prev.columns.find(c => c.field.id === field.id);
      if (exists) {
        return { ...prev, columns: prev.columns.filter(c => c.field.id !== field.id) };
      } else {
        return { ...prev, columns: [...prev.columns, { field }] };
      }
    });
  };

  const handleToggleMeasure = (measure: Measure) => {
    setConfig(prev => {
      const exists = prev.measures.find(m => m.id === measure.id);
      if (exists) {
        return { ...prev, measures: prev.measures.filter(m => m.id !== measure.id) };
      } else {
        return { ...prev, measures: [...prev.measures, measure] };
      }
    });
  };

  const handleAddFilter = () => {
    if (!config.dataSource || config.dataSource.fields.length === 0) return;
    const newFilter: FilterCondition = {
      id: `filter_${Date.now()}`,
      field: config.dataSource.fields[0],
      operator: 'equals',
      value: '',
    };
    setConfig(prev => ({ ...prev, filters: [...prev.filters, newFilter] }));
  };

  const handleRemoveFilter = (filterId: string) => {
    setConfig(prev => ({
      ...prev,
      filters: prev.filters.filter(f => f.id !== filterId),
    }));
  };

  const handleSaveReport = async () => {
    setIsLoading(true);
    // Simulate save
    await new Promise(resolve => setTimeout(resolve, 1000));
    setIsLoading(false);
    navigate('/reports/library');
  };

  const isFieldSelected = (fieldId: string) => config.columns.some(c => c.field.id === fieldId);
  const isMeasureSelected = (measureId: string) => config.measures.some(m => m.id === measureId);

  const canProceed = () => {
    switch (activeStep) {
      case 0: return config.dataSource !== null;
      case 1: return config.columns.length > 0 || config.measures.length > 0;
      case 2: return true; // Filters optional
      case 3: return config.name.trim() !== '';
      default: return true;
    }
  };

  if (!tenant) {
    return (
      <Box sx={{ p: 3 }}>
        <Alert severity="warning">Please select a tenant to create reports.</Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ 
      minHeight: '100vh',
      bgcolor: '#f5f7fa',
    }}>
      {/* Premium Header */}
      <Paper 
        elevation={0} 
        sx={{ 
          background: COLORS.gradient,
          color: 'white',
          py: 2,
          px: 3,
          borderRadius: 0,
        }}
      >
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
            <IconButton color="inherit" onClick={() => navigate('/reports/library')}>
              <CloseIcon />
            </IconButton>
            <Box>
              <Typography variant="h5" fontWeight="bold">
                {config.name || 'New Report'}
              </Typography>
              <Typography variant="body2" sx={{ opacity: 0.9 }}>
                {tenant.display_name || tenant.name} • {config.dataSource?.displayName || 'Select a data source'}
              </Typography>
            </Box>
          </Box>
          
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Tooltip title="Save as Draft">
              <IconButton color="inherit" disabled={!canProceed()}>
                <BookmarkIcon />
              </IconButton>
            </Tooltip>
            <Button 
              variant="contained" 
              color="inherit"
              sx={{ color: COLORS.primary, bgcolor: 'white', '&:hover': { bgcolor: 'rgba(255,255,255,0.9)' } }}
              startIcon={<RunIcon />}
              onClick={handleSaveReport}
              disabled={activeStep < 3 || !canProceed()}
            >
              Run Report
            </Button>
          </Box>
        </Box>
        
        {/* Progress Steps */}
        <Box sx={{ display: 'flex', gap: 3, mt: 3 }}>
          {['Data Source', 'Columns', 'Filters', 'Configure'].map((label, index) => (
            <Box 
              key={label}
              onClick={() => index <= activeStep && setActiveStep(index)}
              sx={{ 
                display: 'flex', 
                alignItems: 'center', 
                gap: 1,
                cursor: index <= activeStep ? 'pointer' : 'default',
                opacity: index <= activeStep ? 1 : 0.5,
                transition: 'all 0.2s',
                '&:hover': index <= activeStep ? { transform: 'translateY(-2px)' } : {},
              }}
            >
              <Avatar 
                sx={{ 
                  width: 28, 
                  height: 28, 
                  fontSize: '0.85rem',
                  bgcolor: index < activeStep ? 'rgba(255,255,255,0.9)' : 
                           index === activeStep ? 'white' : 'rgba(255,255,255,0.3)',
                  color: index <= activeStep ? COLORS.primary : 'white',
                  fontWeight: 'bold',
                }}
              >
                {index < activeStep ? <CheckIcon sx={{ fontSize: 16 }} /> : index + 1}
              </Avatar>
              <Typography 
                variant="body2" 
                fontWeight={index === activeStep ? 'bold' : 'normal'}
              >
                {label}
              </Typography>
              {index < 3 && (
                <Box sx={{ width: 40, height: 2, bgcolor: 'rgba(255,255,255,0.3)', ml: 1 }} />
              )}
            </Box>
          ))}
        </Box>
      </Paper>

      {isLoading && <LinearProgress />}

      <Box sx={{ p: 3 }}>
        <Grid container spacing={3}>
          {/* Main Content */}
          <Grid item xs={12} md={showPreview && activeStep > 0 ? 7 : 12}>
            <Fade in={true}>
              <Box>
                {/* Step 0: Choose Data Source */}
                {activeStep === 0 && (
                  <Card sx={{ borderRadius: 2 }}>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
                        <Box>
                          <Typography variant="h6" fontWeight="bold">
                            Choose Your Data Source
                          </Typography>
                          <Typography variant="body2" color="text.secondary">
                            Select a business object or semantic model to power your report
                          </Typography>
                        </Box>
                        <TextField
                          size="small"
                          placeholder="Search data sources..."
                          value={searchQuery}
                          onChange={(e) => setSearchQuery(e.target.value)}
                          InputProps={{
                            startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment>,
                          }}
                          sx={{ width: 280 }}
                        />
                      </Box>

                      <Divider sx={{ mb: 2 }} />

                      {/* Business Objects */}
                      <Typography variant="overline" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
                        Business Objects
                      </Typography>
                      <Grid container spacing={2} sx={{ mb: 3 }}>
                        {filteredSources.filter(s => s.type === 'business_object').map(source => (
                          <Grid item xs={12} sm={6} md={4} key={source.id}>
                            <Paper
                              variant="outlined"
                              onClick={() => handleSelectDataSource(source)}
                              sx={{
                                p: 2,
                                cursor: 'pointer',
                                transition: 'all 0.2s',
                                border: config.dataSource?.id === source.id ? '2px solid' : '1px solid',
                                borderColor: config.dataSource?.id === source.id ? 'primary.main' : 'divider',
                                bgcolor: config.dataSource?.id === source.id ? alpha(COLORS.primary, 0.05) : 'transparent',
                                '&:hover': {
                                  borderColor: 'primary.main',
                                  transform: 'translateY(-2px)',
                                  boxShadow: 2,
                                },
                              }}
                            >
                              <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
                                <Avatar sx={{ bgcolor: alpha(COLORS.primary, 0.1), color: COLORS.primary }}>
                                  <BusinessObjectIcon />
                                </Avatar>
                                <Box sx={{ flex: 1 }}>
                                  <Typography variant="subtitle1" fontWeight="bold">
                                    {source.displayName}
                                  </Typography>
                                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                                    {source.description}
                                  </Typography>
                                  <Stack direction="row" spacing={1}>
                                    <Chip label={`${source.fields.length} fields`} size="small" variant="outlined" />
                                    {source.rowCount && (
                                      <Chip label={`${source.rowCount.toLocaleString()} rows`} size="small" variant="outlined" />
                                    )}
                                  </Stack>
                                </Box>
                              </Box>
                            </Paper>
                          </Grid>
                        ))}
                      </Grid>

                      {/* Semantic Models */}
                      <Typography variant="overline" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
                        Semantic Models
                      </Typography>
                      <Grid container spacing={2}>
                        {filteredSources.filter(s => s.type === 'semantic_model').map(source => (
                          <Grid item xs={12} sm={6} md={4} key={source.id}>
                            <Paper
                              variant="outlined"
                              onClick={() => handleSelectDataSource(source)}
                              sx={{
                                p: 2,
                                cursor: 'pointer',
                                transition: 'all 0.2s',
                                border: config.dataSource?.id === source.id ? '2px solid' : '1px solid',
                                borderColor: config.dataSource?.id === source.id ? 'secondary.main' : 'divider',
                                bgcolor: config.dataSource?.id === source.id ? alpha(COLORS.secondary, 0.05) : 'transparent',
                                '&:hover': {
                                  borderColor: 'secondary.main',
                                  transform: 'translateY(-2px)',
                                  boxShadow: 2,
                                },
                              }}
                            >
                              <Box sx={{ display: 'flex', alignItems: 'flex-start', gap: 2 }}>
                                <Avatar sx={{ bgcolor: alpha(COLORS.secondary, 0.1), color: COLORS.secondary }}>
                                  <SemanticModelIcon />
                                </Avatar>
                                <Box sx={{ flex: 1 }}>
                                  <Typography variant="subtitle1" fontWeight="bold">
                                    {source.displayName}
                                  </Typography>
                                  <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
                                    {source.description}
                                  </Typography>
                                  <Stack direction="row" spacing={1}>
                                    <Chip label={`${source.fields.length} dimensions`} size="small" color="secondary" variant="outlined" />
                                    <Chip label={`${source.measures?.length || 0} measures`} size="small" color="primary" variant="outlined" />
                                  </Stack>
                                </Box>
                              </Box>
                            </Paper>
                          </Grid>
                        ))}
                      </Grid>
                    </CardContent>
                  </Card>
                )}

                {/* Step 1: Select Columns */}
                {activeStep === 1 && config.dataSource && (
                  <Card sx={{ borderRadius: 2 }}>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                        <Box>
                          <Typography variant="h6" fontWeight="bold">
                            Select Columns
                          </Typography>
                          <Typography variant="body2" color="text.secondary">
                            Choose the fields to include in your report
                          </Typography>
                        </Box>
                        <Box sx={{ display: 'flex', gap: 1 }}>
                          <Chip 
                            label={`${config.columns.length} columns`} 
                            color="primary" 
                            size="small" 
                          />
                          {config.measures.length > 0 && (
                            <Chip 
                              label={`${config.measures.length} measures`} 
                              color="secondary" 
                              size="small" 
                            />
                          )}
                        </Box>
                      </Box>

                      <Grid container spacing={2}>
                        {/* Available Fields */}
                        <Grid item xs={12} md={6}>
                          <Paper variant="outlined" sx={{ height: 400, overflow: 'hidden' }}>
                            <Box sx={{ p: 1.5, bgcolor: 'grey.50', borderBottom: 1, borderColor: 'divider' }}>
                              <Typography variant="subtitle2" fontWeight="bold">
                                Available Fields
                              </Typography>
                            </Box>
                            <List sx={{ height: 'calc(100% - 48px)', overflow: 'auto' }}>
                              {config.dataSource.fields.map(field => (
                                <ListItemButton
                                  key={field.id}
                                  onClick={() => handleToggleField(field)}
                                  sx={{
                                    borderLeft: '3px solid',
                                    borderColor: isFieldSelected(field.id) ? 'primary.main' : 'transparent',
                                    bgcolor: isFieldSelected(field.id) ? alpha(COLORS.primary, 0.08) : 'transparent',
                                  }}
                                >
                                  <ListItemIcon sx={{ minWidth: 36, color: getFieldTypeColor(field.type) }}>
                                    {getFieldTypeIcon(field.type)}
                                  </ListItemIcon>
                                  <ListItemText
                                    primary={
                                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                                        {field.label}
                                        {field.isPrimaryKey && <KeyIcon sx={{ fontSize: 14, color: 'warning.main' }} />}
                                        {field.isForeignKey && <LinkIcon sx={{ fontSize: 14, color: 'info.main' }} />}
                                      </Box>
                                    }
                                    secondary={field.name}
                                    primaryTypographyProps={{ fontWeight: isFieldSelected(field.id) ? 'bold' : 'normal' }}
                                  />
                                  {isFieldSelected(field.id) && <CheckIcon color="primary" />}
                                </ListItemButton>
                              ))}
                            </List>
                          </Paper>
                        </Grid>

                        {/* Measures (for semantic models) */}
                        {config.dataSource.type === 'semantic_model' && config.dataSource.measures && (
                          <Grid item xs={12} md={6}>
                            <Paper variant="outlined" sx={{ height: 400, overflow: 'hidden' }}>
                              <Box sx={{ p: 1.5, bgcolor: 'secondary.50', borderBottom: 1, borderColor: 'divider' }}>
                                <Typography variant="subtitle2" fontWeight="bold" color="secondary.main">
                                  Measures
                                </Typography>
                              </Box>
                              <List sx={{ height: 'calc(100% - 48px)', overflow: 'auto' }}>
                                {config.dataSource.measures.map(measure => (
                                  <ListItemButton
                                    key={measure.id}
                                    onClick={() => handleToggleMeasure(measure)}
                                    sx={{
                                      borderLeft: '3px solid',
                                      borderColor: isMeasureSelected(measure.id) ? 'secondary.main' : 'transparent',
                                      bgcolor: isMeasureSelected(measure.id) ? alpha(COLORS.secondary, 0.08) : 'transparent',
                                    }}
                                  >
                                    <ListItemIcon sx={{ minWidth: 36 }}>
                                      <MeasureIcon color="secondary" />
                                    </ListItemIcon>
                                    <ListItemText
                                      primary={measure.label}
                                      secondary={`${measure.aggregation} • ${measure.format}`}
                                      primaryTypographyProps={{ fontWeight: isMeasureSelected(measure.id) ? 'bold' : 'normal' }}
                                    />
                                    {isMeasureSelected(measure.id) && <CheckIcon color="secondary" />}
                                  </ListItemButton>
                                ))}
                              </List>
                            </Paper>
                          </Grid>
                        )}
                      </Grid>
                    </CardContent>
                  </Card>
                )}

                {/* Step 2: Add Filters */}
                {activeStep === 2 && config.dataSource && (
                  <Card sx={{ borderRadius: 2 }}>
                    <CardContent>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
                        <Box>
                          <Typography variant="h6" fontWeight="bold">
                            Add Filters (Optional)
                          </Typography>
                          <Typography variant="body2" color="text.secondary">
                            Filter your data to show only what matters
                          </Typography>
                        </Box>
                        <Button
                          variant="outlined"
                          startIcon={<AddIcon />}
                          onClick={handleAddFilter}
                        >
                          Add Filter
                        </Button>
                      </Box>

                      {config.filters.length === 0 ? (
                        <Paper 
                          variant="outlined" 
                          sx={{ 
                            p: 4, 
                            textAlign: 'center',
                            bgcolor: 'grey.50',
                            borderStyle: 'dashed',
                          }}
                        >
                          <FilterIcon sx={{ fontSize: 48, color: 'grey.400', mb: 2 }} />
                          <Typography variant="body1" color="text.secondary">
                            No filters added yet
                          </Typography>
                          <Typography variant="body2" color="text.secondary">
                            Click "Add Filter" to narrow down your results
                          </Typography>
                        </Paper>
                      ) : (
                        <Stack spacing={2}>
                          {config.filters.map((filter, index) => (
                            <Paper key={filter.id} variant="outlined" sx={{ p: 2 }}>
                              <Grid container spacing={2} alignItems="center">
                                <Grid item xs={12} sm={3}>
                                  <FormControl fullWidth size="small">
                                    <InputLabel>Field</InputLabel>
                                    <Select
                                      value={filter.field.id}
                                      label="Field"
                                      onChange={(e) => {
                                        const newField = config.dataSource?.fields.find(f => f.id === e.target.value);
                                        if (newField) {
                                          setConfig(prev => ({
                                            ...prev,
                                            filters: prev.filters.map(f => 
                                              f.id === filter.id ? { ...f, field: newField } : f
                                            ),
                                          }));
                                        }
                                      }}
                                    >
                                      {config.dataSource?.fields.map(f => (
                                        <MenuItem key={f.id} value={f.id}>{f.label}</MenuItem>
                                      ))}
                                    </Select>
                                  </FormControl>
                                </Grid>
                                <Grid item xs={12} sm={3}>
                                  <FormControl fullWidth size="small">
                                    <InputLabel>Operator</InputLabel>
                                    <Select
                                      value={filter.operator}
                                      label="Operator"
                                      onChange={(e) => {
                                        setConfig(prev => ({
                                          ...prev,
                                          filters: prev.filters.map(f =>
                                            f.id === filter.id ? { ...f, operator: e.target.value as any } : f
                                          ),
                                        }));
                                      }}
                                    >
                                      <MenuItem value="equals">Equals</MenuItem>
                                      <MenuItem value="not_equals">Not Equals</MenuItem>
                                      <MenuItem value="contains">Contains</MenuItem>
                                      <MenuItem value="starts_with">Starts With</MenuItem>
                                      <MenuItem value="greater_than">Greater Than</MenuItem>
                                      <MenuItem value="less_than">Less Than</MenuItem>
                                    </Select>
                                  </FormControl>
                                </Grid>
                                <Grid item xs={12} sm={5}>
                                  <TextField
                                    fullWidth
                                    size="small"
                                    label="Value"
                                    value={filter.value}
                                    onChange={(e) => {
                                      setConfig(prev => ({
                                        ...prev,
                                        filters: prev.filters.map(f =>
                                          f.id === filter.id ? { ...f, value: e.target.value } : f
                                        ),
                                      }));
                                    }}
                                  />
                                </Grid>
                                <Grid item xs={12} sm={1}>
                                  <IconButton onClick={() => handleRemoveFilter(filter.id)} color="error">
                                    <DeleteIcon />
                                  </IconButton>
                                </Grid>
                              </Grid>
                            </Paper>
                          ))}
                        </Stack>
                      )}
                    </CardContent>
                  </Card>
                )}

                {/* Step 3: Configure & Save */}
                {activeStep === 3 && (
                  <Card sx={{ borderRadius: 2 }}>
                    <CardContent>
                      <Typography variant="h6" fontWeight="bold" gutterBottom>
                        Configure Your Report
                      </Typography>

                      <Grid container spacing={3}>
                        <Grid item xs={12} md={6}>
                          <TextField
                            fullWidth
                            label="Report Name"
                            value={config.name}
                            onChange={(e) => setConfig(prev => ({ ...prev, name: e.target.value }))}
                            placeholder="Enter a descriptive name..."
                            required
                          />
                        </Grid>
                        <Grid item xs={12} md={6}>
                          <TextField
                            fullWidth
                            label="Description"
                            value={config.description}
                            onChange={(e) => setConfig(prev => ({ ...prev, description: e.target.value }))}
                            placeholder="What does this report show?"
                          />
                        </Grid>

                        <Grid item xs={12}>
                          <Divider sx={{ my: 1 }} />
                          <Typography variant="subtitle2" gutterBottom sx={{ mt: 2 }}>
                            Visualization
                          </Typography>
                          <Grid container spacing={1}>
                            {[
                              { type: 'table', icon: <TableIcon />, label: 'Table' },
                              { type: 'bar', icon: <BarChartIcon />, label: 'Bar Chart' },
                              { type: 'line', icon: <LineChartIcon />, label: 'Line Chart' },
                              { type: 'pie', icon: <PieChartIcon />, label: 'Pie Chart' },
                            ].map(({ type, icon, label }) => (
                              <Grid item key={type}>
                                <Paper
                                  variant="outlined"
                                  onClick={() => setConfig(prev => ({ ...prev, chartType: type as any }))}
                                  sx={{
                                    p: 2,
                                    cursor: 'pointer',
                                    textAlign: 'center',
                                    minWidth: 100,
                                    border: config.chartType === type ? '2px solid' : '1px solid',
                                    borderColor: config.chartType === type ? 'primary.main' : 'divider',
                                    bgcolor: config.chartType === type ? alpha(COLORS.primary, 0.05) : 'transparent',
                                    '&:hover': { borderColor: 'primary.main' },
                                  }}
                                >
                                  {icon}
                                  <Typography variant="body2" sx={{ mt: 1 }}>{label}</Typography>
                                </Paper>
                              </Grid>
                            ))}
                          </Grid>
                        </Grid>

                        <Grid item xs={12} md={6}>
                          <Typography variant="subtitle2" gutterBottom>
                            Result Limit
                          </Typography>
                          <Slider
                            value={config.limit}
                            onChange={(_, value) => setConfig(prev => ({ ...prev, limit: value as number }))}
                            min={10}
                            max={10000}
                            step={10}
                            valueLabelDisplay="auto"
                            marks={[
                              { value: 100, label: '100' },
                              { value: 1000, label: '1K' },
                              { value: 5000, label: '5K' },
                              { value: 10000, label: '10K' },
                            ]}
                          />
                        </Grid>

                        <Grid item xs={12} md={6}>
                          <Typography variant="subtitle2" gutterBottom>
                            Options
                          </Typography>
                          <FormControlLabel
                            control={
                              <Switch
                                checked={config.showTotals}
                                onChange={(e) => setConfig(prev => ({ ...prev, showTotals: e.target.checked }))}
                              />
                            }
                            label="Show Totals"
                          />
                          <FormControlLabel
                            control={
                              <Switch
                                checked={config.showPercentages}
                                onChange={(e) => setConfig(prev => ({ ...prev, showPercentages: e.target.checked }))}
                              />
                            }
                            label="Show Percentages"
                          />
                        </Grid>
                      </Grid>
                    </CardContent>
                  </Card>
                )}

                {/* Navigation Buttons */}
                <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 3 }}>
                  <Button
                    variant="outlined"
                    startIcon={<BackIcon />}
                    onClick={() => setActiveStep(prev => prev - 1)}
                    disabled={activeStep === 0}
                  >
                    Back
                  </Button>
                  
                  {activeStep < 3 ? (
                    <Button
                      variant="contained"
                      endIcon={<ForwardIcon />}
                      onClick={() => setActiveStep(prev => prev + 1)}
                      disabled={!canProceed()}
                    >
                      Continue
                    </Button>
                  ) : (
                    <Button
                      variant="contained"
                      startIcon={<SaveIcon />}
                      onClick={handleSaveReport}
                      disabled={!canProceed() || isLoading}
                      sx={{
                        background: COLORS.gradient,
                        '&:hover': { background: COLORS.gradient, filter: 'brightness(1.1)' },
                      }}
                    >
                      Save & Run Report
                    </Button>
                  )}
                </Box>
              </Box>
            </Fade>
          </Grid>

          {/* Live Preview Panel */}
          {showPreview && activeStep > 0 && (
            <Grid item xs={12} md={5}>
              <Zoom in={true}>
                <Card sx={{ borderRadius: 2, position: 'sticky', top: 16 }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                      <Typography variant="h6" fontWeight="bold">
                        Preview
                      </Typography>
                      <IconButton size="small" onClick={() => setShowPreview(false)}>
                        <CloseIcon />
                      </IconButton>
                    </Box>

                    {previewData.length > 0 ? (
                      <Paper variant="outlined" sx={{ overflow: 'auto', maxHeight: 400 }}>
                        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
                          <thead>
                            <tr style={{ background: '#f5f5f5' }}>
                              {config.columns.map(col => (
                                <th key={col.field.id} style={{ padding: 8, textAlign: 'left', borderBottom: '1px solid #e0e0e0' }}>
                                  {col.field.label}
                                </th>
                              ))}
                              {config.measures.map(m => (
                                <th key={m.id} style={{ padding: 8, textAlign: 'right', borderBottom: '1px solid #e0e0e0', color: COLORS.secondary }}>
                                  {m.label}
                                </th>
                              ))}
                            </tr>
                          </thead>
                          <tbody>
                            {previewData.map((row, i) => (
                              <tr key={i} style={{ borderBottom: '1px solid #f0f0f0' }}>
                                {config.columns.map(col => (
                                  <td key={col.field.id} style={{ padding: 8 }}>
                                    {row[col.field.name]}
                                  </td>
                                ))}
                                {config.measures.map(m => (
                                  <td key={m.id} style={{ padding: 8, textAlign: 'right', fontFamily: 'monospace' }}>
                                    {row[m.name]}
                                  </td>
                                ))}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </Paper>
                    ) : (
                      <Box sx={{ textAlign: 'center', py: 4 }}>
                        <PreviewIcon sx={{ fontSize: 48, color: 'grey.400', mb: 1 }} />
                        <Typography color="text.secondary">
                          Select columns to see a preview
                        </Typography>
                      </Box>
                    )}

                    <Box sx={{ mt: 2, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <Typography variant="caption" color="text.secondary">
                        Showing {previewData.length} sample rows
                      </Typography>
                      <Button size="small" startIcon={<RefreshIcon />} onClick={generatePreviewData}>
                        Refresh
                      </Button>
                    </Box>
                  </CardContent>
                </Card>
              </Zoom>
            </Grid>
          )}
        </Grid>
      </Box>
    </Box>
  );
};

export default WorldClassReportBuilder;
