import { useState, useMemo, useEffect } from 'react';
import { devLog } from '../../../utils/devLogger';
import {
  Box,
  Typography,
  Card,
  CardContent,
  CardActions,
  Button,
  Chip,
  TextField,
  InputAdornment,
  IconButton,
  Dialog,
  DialogContent,
  DialogActions,
  Alert,
  Snackbar,
  Paper,
  List,
  ListItem,
  ListItemButton,
  ListItemText,
  ListItemIcon,
  Collapse,
  Checkbox,
  
  Tabs,
  Tab,
  Grid
} from '@mui/material';
import { CircularProgress } from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import {
  Search as SearchIcon,
  Clear as ClearIcon,
  Add as AddIcon,
  Functions as FunctionsIcon,
  TrendingUp as TrendingUpIcon,
  Assessment as AssessmentIcon,
  AccountBalance as AccountBalanceIcon,
  ExpandLess,
  ExpandMore,
  Timeline as _TimelineIcon,
  PieChart as PieChartIcon,
  Security as SecurityIcon,
  Calculate as CalculateIcon,
  Gavel as GavelIcon,
  Edit as EditIcon,
  TableChart as ExcelIcon
  ,ContentCopy as ContentCopyIcon
} from '@mui/icons-material';
import { libraryOptions as defaultLibraryOptions } from '../../../components/UnifiedSemanticBuilder/financialCalculations';
import MetricsViewer from '../../../components/MetricsViewer';
import { listCalculations, createCalculation, Calculation } from '../../../api/calculations';
import { listDomains, DataDomain } from '../../../api/domains';
import { MenuItem, Select, FormControl, InputLabel } from '@mui/material';

interface CalculationsLibraryPageProps {
  tenantId: string;
  datasourceId: string;
}

interface CalculationOption {
  name: string;
  title: string;
  type: string;
  sql: string;
  description?: string;
  sourceTable?: string;
  sourceColumn?: string;
  format?: string;
  aggregationType?: string;
  defaultValue?: string;
  category?: string;
  subcategory?: string;
  preAggregationTemplate?: any;
  backendEndpoint?: string;
  financial_calc?: {
    type: string;
    formula?: string;
    arguments?: Record<string, string>;
  };
  domain_id?: string;
  execution_type?: string;
  engine?: string;
}

interface CategoryFacet {
  name: string;
  label: string;
  icon: React.ReactNode;
  color: 'primary' | 'warning' | 'success' | 'default';
  subcategories: string[];
  count: number;
}

// _SubcategoryFacet was removed because it was declared but unused

interface CalculationsLibraryPageProps {
  tenantId: string;
  datasourceId: string;
}

const CalculationsLibraryPage: React.FC<CalculationsLibraryPageProps> = () => {
  const [calculations, setCalculations] = useState<CalculationOption[]>(defaultLibraryOptions);
  const [domains, setDomains] = useState<DataDomain[]>([]);
  const [loading, setLoading] = useState(false);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(new Set());
  const [selectedSubcategories, setSelectedSubcategories] = useState<Set<string>>(new Set());
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set());
  const [editingCalculation, setEditingCalculation] = useState<Partial<CalculationOption> | null>(null);
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [snackbarMessage, setSnackbarMessage] = useState('');
  const [activeTab, setActiveTab] = useState(0);
  // Test/run calculation UI state
  const [testModalOpen, setTestModalOpen] = useState(false);
  const [testLoading, setTestLoading] = useState(false);
  const [testCalculation, setTestCalculation] = useState<CalculationOption | null>(null);
  const [testRequestBody, setTestRequestBody] = useState<any>(null);
  const [testResponseBody, setTestResponseBody] = useState<any>(null);
  const [testError, setTestError] = useState<string | null>(null);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  useEffect(() => {
    const fetchCalculations = async () => {
      try {
        setLoading(true);
        // 1. Fetch Legacy Calculations
        const apiCalculations = await listCalculations();
        const mappedLegacy: CalculationOption[] = apiCalculations.map(c => ({
          name: c.name,
          title: c.title,
          type: c.engine_type,
          sql: c.formula,
          description: c.description,
          category: c.category,
          subcategory: c.subcategory,
          backendEndpoint: '/api/calculations/execute',
          financial_calc: {
            type: c.engine_type,
            formula: c.formula,
            arguments: c.arguments as Record<string, string>
          }
        }));

        // 2. Fetch Semantic Terms
        const semanticRes = await fetch('/api/semantic-terms');
        const semanticData = await semanticRes.json();
        const semanticTerms: any[] = semanticData.data || [];
        
        const mappedSemantic: CalculationOption[] = semanticTerms
          .filter(t => t.node_name.startsWith('financial.'))
          .map(t => ({
            name: t.node_name,
            title: t.display_name || t.node_name.replace('financial.', '').replace(/_/g, ' '),
            type: t.type,
            sql: t.expression || '',
            description: t.description,
            category: t.tags?.[0] || 'Financial',
            subcategory: t.tags?.[1] || '',
            backendEndpoint: '/api/semantic-terms/explain', // Use explain for deep dive/execution
            execution_type: 'plugin'
          }));
        
        // Merge everything
        const allFetched = [...mappedLegacy, ...mappedSemantic];
        const existingNames = new Set(defaultLibraryOptions.map(c => c.name));
        const newCalculations = allFetched.filter(c => !existingNames.has(c.name));
        
        setCalculations([...defaultLibraryOptions, ...newCalculations]);
      } catch (err) {
        console.error('Failed to fetch calculations:', err);
        setSnackbarMessage('Failed to load calculations from server.');
        setSnackbarOpen(true);
      } finally {
        setLoading(false);
      }
    };



    const fetchDomains = async () => {
      try {
        const apiDomains = await listDomains();
        setDomains(apiDomains);
      } catch (err) {
        console.error('Failed to fetch domains:', err);
      }
    };

    fetchCalculations();
    fetchDomains();
  }, []);

  // Define category structure with subcategories
  const categoryStructure: CategoryFacet[] = useMemo(() => {
    const structure: CategoryFacet[] = [
      {
        name: 'Performance',
        label: 'Performance',
        icon: <TrendingUpIcon />,
        color: 'primary',
        subcategories: ['Returns', 'Growth', 'Valuation', 'IRR'],
        count: 0
      },
      {
        name: 'Risk',
        label: 'Risk',
        icon: <AssessmentIcon />,
        color: 'warning',
        subcategories: ['Volatility', 'Drawdown', 'Correlation', 'Market Risk', 'Credit Risk'],
        count: 0
      },
      {
        name: 'Private Markets',
        label: 'Private Markets',
        icon: <AccountBalanceIcon />,
        color: 'success',
        subcategories: ['Performance', 'Multiples', 'Cash Flow', 'Liquidity', 'Valuation'],
        count: 0
      },
      {
        name: 'Insurance',
        label: 'Insurance',
        icon: <SecurityIcon />,
        color: 'success',
        subcategories: ['Underwriting', 'Reserving', 'Solvency', 'Profitability'],
        count: 0
      },
      {
        name: 'Banking',
        label: 'Banking & Lending',
        icon: <GavelIcon />,
        color: 'default',
        subcategories: ['Risk', 'Profitability', 'Regulatory'],
        count: 0
      },
      {
        name: 'Quant Finance',
        label: 'Quant Finance',
        icon: <CalculateIcon />,
        color: 'warning',
        subcategories: ['Market Risk', 'Derivatives Pricing', 'Fixed Income'],
        count: 0
      },
      {
        name: 'Risk Management',
        label: 'Risk Management',
        icon: <SecurityIcon />,
        color: 'warning',
        subcategories: ['Market Risk', 'Credit Risk', 'Operational Risk'],
        count: 0
      },
      {
        name: 'Compliance & Regulatory',
        label: 'Compliance & Regulatory',
        icon: <GavelIcon />,
        color: 'default',
        subcategories: ['Banking/Basel III', 'Insurance/Solvency II', 'AML/KYC', 'Market Conduct'],
        count: 0
      },
      {
        name: 'Wealth',
        label: 'Wealth Management',
        icon: <PieChartIcon />,
        color: 'default',
        subcategories: ['Allocation', 'Diversification'],
        count: 0
      }
    ];

    // Calculate counts for each category
    structure.forEach(cat => {
      cat.count = calculations.filter(calc => calc.category === cat.name).length;
    });

    return structure;
  }, []);

  // Filter calculations based on search and selected facets
  const filteredCalculations = useMemo(() => {
    return calculations.filter(calc => {
      const matchesSearch = calc.title.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           calc.description?.toLowerCase().includes(searchTerm.toLowerCase()) ||
                           calc.name.toLowerCase().includes(searchTerm.toLowerCase());

      const matchesCategory = selectedCategories.size === 0 ||
                             selectedCategories.has(calc.category || 'General');

      const matchesSubcategory = selectedSubcategories.size === 0 ||
                                selectedSubcategories.has(calc.subcategory || '');

      return matchesSearch && matchesCategory && matchesSubcategory;
    });
  }, [searchTerm, selectedCategories, selectedSubcategories]);

  const handleCategoryToggle = (categoryName: string) => {
    const newSelected = new Set(selectedCategories);
    if (newSelected.has(categoryName)) {
      newSelected.delete(categoryName);
    } else {
      newSelected.add(categoryName);
    }
    setSelectedCategories(newSelected);
  };

  const handleSubcategoryToggle = (subcategoryName: string) => {
    const newSelected = new Set(selectedSubcategories);
    if (newSelected.has(subcategoryName)) {
      newSelected.delete(subcategoryName);
    } else {
      newSelected.add(subcategoryName);
    }
    setSelectedSubcategories(newSelected);
  };

  const handleCategoryExpand = (categoryName: string) => {
    const newExpanded = new Set(expandedCategories);
    if (newExpanded.has(categoryName)) {
      newExpanded.delete(categoryName);
    } else {
      newExpanded.add(categoryName);
    }
    setExpandedCategories(newExpanded);
  };

  const clearAllFilters = () => {
    setSelectedCategories(new Set());
    setSelectedSubcategories(new Set());
    setSearchTerm('');
  };

  const handleOpenEditor = (calculation: CalculationOption | null) => {
    setEditingCalculation(calculation ? { ...calculation } : { type: 'measure' });
  };

  const handleSaveCalculation = async (calculation: Partial<CalculationOption>) => {
    devLog('Saving calculation:', calculation);
    
    try {
      // Save to backend
      const newCalc: Calculation = {
        node_id: calculation.name || '',
        name: calculation.name || '',
        title: calculation.title || '',
        description: calculation.description,
        formula: calculation.sql || '',
        engine_type: calculation.type || 'formula',
        return_type: 'number', // Default
        category: calculation.category,
        subcategory: calculation.subcategory,
        arguments: calculation.financial_calc?.arguments,
        domain_id: calculation.domain_id,
        execution_type: calculation.execution_type,
        engine: calculation.engine
      };


      await createCalculation(newCalc);

      setSnackbarMessage(`Successfully saved "${calculation.title}" to the library.`);
      setSnackbarOpen(true);
      setEditingCalculation(null);
      
      // Refresh list
      const apiCalculations = await listCalculations();
      const mappedCalculations: CalculationOption[] = apiCalculations.map(c => ({
          name: c.name,
          title: c.title,
          type: c.engine_type,
          sql: c.formula,
          description: c.description,
          category: c.category,
          subcategory: c.subcategory,
          backendEndpoint: '/api/calculations/execute',
          financial_calc: {
            type: c.engine_type,
            formula: c.formula,
            arguments: c.arguments as Record<string, string>
          }
      }));
      
      const existingNames = new Set(defaultLibraryOptions.map(c => c.name));
      const newCalculations = mappedCalculations.filter(c => !existingNames.has(c.name));
      setCalculations([...defaultLibraryOptions, ...newCalculations]);

    } catch (err) {
      console.error('Failed to save calculation:', err);
      setSnackbarMessage('Failed to save calculation.');
      setSnackbarOpen(true);
    }
  };

  // Run a test call for a calculation and show modal with results
  const handleTestCalculation = async (calculation: CalculationOption) => {
    setTestCalculation(calculation);
    setTestLoading(true);
    setTestError(null);
    setTestResponseBody(null);

    // Choose endpoint: prefer calculation.backendEndpoint else default run
    const endpoint = calculation.backendEndpoint || '/api/calc/run';

    // Build request body. If calling calc/run we send a Template-like payload.
    let body: any = {};
    if (endpoint.includes('/calc/run')) {
      body = {
        node_id: calculation.name,
        financial_calc: calculation.financial_calc || { type: 'formula', formula: calculation.sql }
      };
    } else {
      // Other backend endpoints: send a small financial_calc wrapper
      body = { financial_calc: calculation.financial_calc || { type: 'formula', formula: calculation.sql } };
    }

    setTestRequestBody(body);
    setTestModalOpen(true);

    try {
      const res = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify(body)
      });

      const text = await res.text();
      let parsed;
      try {
        parsed = JSON.parse(text);
      } catch (err) {
        parsed = text;
      }

      if (!res.ok) {
        setTestError(`Status ${res.status}: ${res.statusText} - ${typeof parsed === 'string' ? parsed : JSON.stringify(parsed)}`);
      } else {
        setTestResponseBody(parsed);
      }
    } catch (err: any) {
      setTestError(err?.message || String(err));
    } finally {
      setTestLoading(false);
    }
  };

  // Load a built-in sample payload for IRR/XIRR style calculations
  const handleLoadSamplePayload = () => {
    const sample = {
      node_id: testCalculation?.name || 'sample_irr',
      financial_calc: {
        type: 'xirr',
        cash_flows_dated: [
          { amount: -5000000, date: '2018-01-01' },
          { amount: 1000000, date: '2019-01-01' },
          { amount: 20000000, date: '2020-01-01' }
        ],
        guess: 0.1
      }
    };

    setTestRequestBody(sample);
  };
  const getCategoryIcon = (category?: string) => {
    switch (category) {
      case 'Performance':
        return <TrendingUpIcon color="primary" />;
      case 'Risk':
        return <AssessmentIcon color="warning" />;
      case 'Private Markets':
        return <AccountBalanceIcon color="success" />;
      case 'Insurance':
        return <SecurityIcon color="info" />;
      case 'Banking':
        return <GavelIcon color="action" />;
      case 'Quant Finance':
        return <CalculateIcon color="secondary" />;
      default:
        return <FunctionsIcon color="action" />;
    }
  };

  const getCategoryColor = (category?: string): 'primary' | 'secondary' | 'info' | 'success' | 'warning' | 'default' => {
    switch (category) {
      case 'Performance':
        return 'primary';
      case 'Risk':
        return 'warning';
      case 'Private Markets':
        return 'success';
      case 'Insurance':
        return 'info';
      case 'Quant Finance':
        return 'secondary';
      case 'Banking':
        return 'default';
      default:
        return 'default';
    }
  };

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh' }}>
      {/* Sidebar with Facets */}
      <Paper
        sx={{
          width: 320,
          flexShrink: 0,
          p: 2,
          borderRadius: 0,
          borderRight: 1,
          borderColor: 'divider'
        }}
      >
        <Typography variant="h6" gutterBottom>
          Filters
        </Typography>

        {/* Search */}
        <TextField
          placeholder="Search calculations..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
            endAdornment: searchTerm && (
              <InputAdornment position="end">
                <IconButton size="small" onClick={() => setSearchTerm('')}>
                  <ClearIcon />
                </IconButton>
              </InputAdornment>
            )
          }}
          sx={{ mb: 2, width: '100%' }}
          size="small"
        />

        {/* Clear Filters */}
        {(selectedCategories.size > 0 || selectedSubcategories.size > 0 || searchTerm) && (
          <Button
            onClick={clearAllFilters}
            size="small"
            sx={{ mb: 2 }}
            variant="outlined"
          >
            Clear All Filters
          </Button>
        )}

        {/* Categories */}
        <Typography variant="subtitle2" sx={{ mb: 1, mt: 2 }}>
          Categories
        </Typography>
        <List dense>
          {categoryStructure.map((category) => (
            <Box key={category.name}>
              <ListItem disablePadding>
                <ListItemButton
                  onClick={() => handleCategoryToggle(category.name)}
                  sx={{ pl: 0, pr: 0 }}
                >
                  <Checkbox
                    edge="start"
                    checked={selectedCategories.has(category.name)}
                    tabIndex={-1}
                    disableRipple
                    size="small"
                  />
                  <ListItemIcon sx={{ minWidth: 32 }}>
                    {category.icon}
                  </ListItemIcon>
                  <ListItemText
                    primary={
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                        <Typography variant="body2">{category.label}</Typography>
                        <Chip
                          label={category.count}
                          size="small"
                          variant="outlined"
                          sx={{ height: 16, fontSize: '0.7rem' }}
                        />
                      </Box>
                    }
                  />
                  {category.subcategories.length > 0 && (
                    <IconButton
                      size="small"
                      onClick={(e) => {
                        e.stopPropagation();
                        handleCategoryExpand(category.name);
                      }}
                    >
                      {expandedCategories.has(category.name) ? <ExpandLess /> : <ExpandMore />}
                    </IconButton>
                  )}
                </ListItemButton>
              </ListItem>

              {/* Subcategories */}
              {category.subcategories.length > 0 && (
                <Collapse in={expandedCategories.has(category.name)} timeout="auto" unmountOnExit>
                  <List component="div" disablePadding dense>
                    {category.subcategories.map((subcategory) => (
                      <ListItem key={subcategory} disablePadding sx={{ pl: 4 }}>
                        <ListItemButton
                          onClick={() => handleSubcategoryToggle(subcategory)}
                          sx={{ pl: 0, pr: 0 }}
                        >
                          <Checkbox
                            edge="start"
                            checked={selectedSubcategories.has(subcategory)}
                            tabIndex={-1}
                            disableRipple
                            size="small"
                          />
                          <ListItemText
                            primary={
                              <Typography variant="body2" color="text.secondary">
                                {subcategory}
                              </Typography>
                            }
                          />
                        </ListItemButton>
                      </ListItem>
                    ))}
                  </List>
                </Collapse>
              )}
            </Box>
          ))}
        </List>
      </Paper>

      {/* Main Content */}
      <Box sx={{ flexGrow: 1, p: 3 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
          <Typography variant="h4" gutterBottom>
            Calculations Library
          </Typography>
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => handleOpenEditor(null)}
          >
            Add Calculation
          </Button>
        </Box>
        <Typography variant="body1" color="text.secondary" sx={{ mb: 3 }}>
          Browse and add pre-built financial and analytical calculations to your semantic models
        </Typography>

        {/* Tabs */}
        <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
          <Tabs value={activeTab} onChange={handleTabChange} aria-label="calculations tabs">
            <Tab label="Standard Calculations" />
            <Tab label="Semantic Layer Metrics" />
          </Tabs>
        </Box>

        {/* Tab Content */}
        {activeTab === 0 && (
          <>
            {/* Active Filters Display */}
            {(selectedCategories.size > 0 || selectedSubcategories.size > 0) && (
              <Box sx={{ mb: 2, display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                <Typography variant="body2" color="text.secondary">
                  Active filters:
                </Typography>
                {Array.from(selectedCategories).map(category => (
                  <Chip
                    key={category}
                    label={category}
                    size="small"
                    onDelete={() => handleCategoryToggle(category)}
                    color="primary"
                    variant="outlined"
                  />
                ))}
                {Array.from(selectedSubcategories).map(subcategory => (
                  <Chip
                    key={subcategory}
                    label={subcategory}
                    size="small"
                    onDelete={() => handleSubcategoryToggle(subcategory)}
                    color="secondary"
                    variant="outlined"
                  />
                ))}
              </Box>
            )}

            {/* Results Count */}
            <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
              {filteredCalculations.length} calculation{filteredCalculations.length !== 1 ? 's' : ''} found
            </Typography>

            {/* Calculations Grid */}
            <Grid container spacing={3}>
              {filteredCalculations.map((calculation) => (
                <Grid item xs={12} sm={6} md={4} key={calculation.name}>
                  <Card sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
                    <CardContent sx={{ flexGrow: 1 }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                        {getCategoryIcon(calculation.category)}
                        <Typography variant="h6" sx={{ ml: 1 }}>
                          {calculation.title}
                        </Typography>
                      </Box>

                      <Chip
                        label={calculation.category || 'General'}
                        size="small"
                        color={getCategoryColor(calculation.category)}
                        sx={{ mb: 2 }}
                      />

                      {calculation.financial_calc?.type === 'excel_formula' && (
                        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                          <ExcelIcon sx={{ mr: 1, color: 'green' }} />
                          <Typography variant="body2" color="green" sx={{ fontWeight: 'bold' }}>
                            Excel Formula
                          </Typography>
                        </Box>
                      )}

                      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                        {calculation.description}
                      </Typography>

                      <Paper variant="outlined" sx={{ p: 1, backgroundColor: 'grey.50', mb: 2, overflowX: 'auto' }}>
                        <Typography variant="caption" component="pre" sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-all', fontFamily: 'monospace', color: 'text.secondary' }}>
                          {calculation.sql}
                        </Typography>
                      </Paper>

                      {calculation.backendEndpoint && (
                        <Alert severity="info" sx={{ mb: 2, fontSize: '0.75rem', py: 0.5 }}>
                          Uses backend calculation service for accuracy
                        </Alert>
                      )}

                      {calculation.preAggregationTemplate && (
                        <Alert severity="success" sx={{ fontSize: '0.75rem', py: 0.5 }}>
                          Includes automatic pre-aggregation setup
                        </Alert>
                      )}
                    </CardContent>

                    <CardActions>
                      <Button
                        size="small"
                        startIcon={<EditIcon />}
                        onClick={() => handleOpenEditor(calculation)}
                        color="secondary"
                      >
                        Edit
                      </Button>
                      <Button
                        size="small"
                        startIcon={testLoading && testCalculation?.name === calculation.name ? <CircularProgress size={16} /> : <TrendingUpIcon />}
                        onClick={() => {
                          // open test modal and start test
                          setTestCalculation(calculation);
                          handleTestCalculation(calculation);
                        }}
                        color="primary"
                      >
                        Test
                      </Button>
                    </CardActions>
                  </Card>
                </Grid>
              ))}
            </Grid>

            {filteredCalculations.length === 0 && (
              <Box sx={{ textAlign: 'center', py: 8 }}>
                <Typography variant="h6" color="text.secondary">
                  No calculations found
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  Try adjusting your search or filters
                </Typography>
              </Box>
            )}
          </>
        )}

        {activeTab === 1 && (
          <Box>
            <Typography variant="h6" gutterBottom>
              Multi-Dialect Semantic Layer Metrics
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
              Browse your governed financial services metrics with cross-engine SQL translations
            </Typography>
            <MetricsViewer />
          </Box>
        )}
      </Box>

      <CalculationEditorModal
        open={!!editingCalculation}
        onClose={() => setEditingCalculation(null)}
        onSave={handleSaveCalculation}
        calculation={editingCalculation}
        domains={domains}
      />
      <TestResultDialog
        open={testModalOpen}
        onClose={() => { setTestModalOpen(false); setTestCalculation(null); setTestRequestBody(null); setTestResponseBody(null); setTestError(null); }}
        loading={testLoading}
        calculation={testCalculation}
        requestBody={testRequestBody}
        responseBody={testResponseBody}
        error={testError}
        onLoadSample={handleLoadSamplePayload}
      />
      {/* Success Snackbar */}
      <Snackbar
        open={snackbarOpen}
        autoHideDuration={4000}
        onClose={() => setSnackbarOpen(false)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert onClose={() => setSnackbarOpen(false)} severity="success">
          {snackbarMessage}
        </Alert>
      </Snackbar>
    </Box>
  );
};

interface CalculationEditorModalProps {
  open: boolean;
  onClose: () => void;
  onSave: (calculation: Partial<CalculationOption>) => void;
  calculation: Partial<CalculationOption> | null;
  domains: DataDomain[];
}

const CalculationEditorModal: React.FC<CalculationEditorModalProps> = ({ open, onClose, onSave, calculation, domains }) => {
  const [formData, setFormData] = useState<Partial<CalculationOption>>({});
  const [errors, setErrors] = useState<Partial<Record<keyof CalculationOption, string>>>({});
  
  // Domain selection state
  const [selectedLevel1, setSelectedLevel1] = useState<string>('');
  const [selectedLevel2, setSelectedLevel2] = useState<string>('');
  const [selectedLevel3, setSelectedLevel3] = useState<string>('');

  useEffect(() => {
    if (calculation) {
      setFormData(calculation);
      // Initialize domain selection if domain_id is present
      if (calculation.domain_id) {
        const domain = domains.find(d => d.id === calculation.domain_id);
        if (domain) {
           // Logic to back-fill levels would go here if we had full hierarchy path
           // For now, just setting the ID if it matches a level 3
           if (domain.level === 3) {
             setSelectedLevel3(domain.id);
             const parent = domains.find(d => d.id === domain.parent_id);
             if (parent) {
               setSelectedLevel2(parent.id);
               const grandParent = domains.find(d => d.id === parent.parent_id);
               if (grandParent) setSelectedLevel1(grandParent.id);
             }
           } else if (domain.level === 2) {
             setSelectedLevel2(domain.id);
             const parent = domains.find(d => d.id === domain.parent_id);
             if (parent) setSelectedLevel1(parent.id);
           } else {
             setSelectedLevel1(domain.id);
           }
        }
      }
    } else {
      setFormData({ type: 'measure', execution_type: 'realtime', engine: 'internal' });
      setSelectedLevel1('');
      setSelectedLevel2('');
      setSelectedLevel3('');
    }
    setErrors({});
  }, [calculation, open, domains]);

  // Filter domains by level and parent
  const level1Domains = useMemo(() => domains.filter(d => d.level === 1), [domains]);
  const level2Domains = useMemo(() => domains.filter(d => d.level === 2 && d.parent_id === selectedLevel1), [domains, selectedLevel1]);
  const level3Domains = useMemo(() => domains.filter(d => d.level === 3 && d.parent_id === selectedLevel2), [domains, selectedLevel2]);

  const handleDomainChange = (level: number, value: string) => {
    if (level === 1) {
      setSelectedLevel1(value);
      setSelectedLevel2('');
      setSelectedLevel3('');
      setFormData(prev => ({ ...prev, domain_id: value }));
    } else if (level === 2) {
      setSelectedLevel2(value);
      setSelectedLevel3('');
      setFormData(prev => ({ ...prev, domain_id: value }));
    } else if (level === 3) {
      setSelectedLevel3(value);
      setFormData(prev => ({ ...prev, domain_id: value }));
    }
  };

  const handleChange = (field: keyof CalculationOption, value: any) => {
    setFormData(prev => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors(prev => {
        const newErrors = { ...prev };
        delete newErrors[field];
        return newErrors;
      });
    }
  };

  const validate = () => {
    const newErrors: Partial<Record<keyof CalculationOption, string>> = {};
    if (!formData.name) newErrors.name = 'Name is required (e.g., my_custom_calc).';
    if (!formData.title) newErrors.title = 'Title is required.';
    if (!formData.sql) newErrors.sql = 'SQL formula is required.';
    if (!formData.category) newErrors.category = 'Category is required.';
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSave = () => {
    if (validate()) {
      onSave(formData);
    }
  };

  const isNew = !calculation?.name;

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
  <ModalHeader title={isNew ? 'Add New Calculation' : `Edit: ${calculation?.title}`} onClose={onClose} />
      <DialogContent>
        <Grid container spacing={2} sx={{ pt: 2 }}>
          <Grid item xs={6}>
            <TextField
              label="Name (ID)"
              value={formData.name || ''}
              onChange={(e) => handleChange('name', e.target.value)}
              fullWidth
              required
              error={!!errors.name}
              helperText={errors.name || "Unique identifier, e.g., my_xirr_calc"}
              disabled={!isNew}
            />
          </Grid>
          <Grid item xs={6}>
            <TextField
              label="Title"
              value={formData.title || ''}
              onChange={(e) => handleChange('title', e.target.value)}
              fullWidth
              required
              error={!!errors.title}
              helperText={errors.title}
            />
          </Grid>
          <Grid item xs={12}>
            <TextField
              label="Description"
              value={formData.description || ''}
              onChange={(e) => handleChange('description', e.target.value)}
              fullWidth
              multiline
              rows={2}
            />
          </Grid>
          <Grid item xs={6}>
            <TextField label="Category" value={formData.category || ''} onChange={(e) => handleChange('category', e.target.value)} fullWidth required error={!!errors.category} helperText={errors.category} />
          </Grid>
          <Grid item xs={6}>
            <TextField label="Subcategory" value={formData.subcategory || ''} onChange={(e) => handleChange('subcategory', e.target.value)} fullWidth />
          </Grid>
          
          {/* Domain Selection */}
          <Grid item xs={12}>
            <Typography variant="subtitle2" gutterBottom>Domain Classification</Typography>
            <Grid container spacing={2}>
              <Grid item xs={4}>
                <FormControl fullWidth size="small">
                  <InputLabel>Domain Level 1</InputLabel>
                  <Select
                    value={selectedLevel1}
                    label="Domain Level 1"
                    onChange={(e) => handleDomainChange(1, e.target.value)}
                  >
                    {level1Domains.map(d => <MenuItem key={d.id} value={d.id}>{d.name}</MenuItem>)}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={4}>
                <FormControl fullWidth size="small" disabled={!selectedLevel1}>
                  <InputLabel>Domain Level 2</InputLabel>
                  <Select
                    value={selectedLevel2}
                    label="Domain Level 2"
                    onChange={(e) => handleDomainChange(2, e.target.value)}
                  >
                    {level2Domains.map(d => <MenuItem key={d.id} value={d.id}>{d.name}</MenuItem>)}
                  </Select>
                </FormControl>
              </Grid>
              <Grid item xs={4}>
                <FormControl fullWidth size="small" disabled={!selectedLevel2}>
                  <InputLabel>Domain Level 3</InputLabel>
                  <Select
                    value={selectedLevel3}
                    label="Domain Level 3"
                    onChange={(e) => handleDomainChange(3, e.target.value)}
                  >
                    {level3Domains.map(d => <MenuItem key={d.id} value={d.id}>{d.name}</MenuItem>)}
                  </Select>
                </FormControl>
              </Grid>
            </Grid>
          </Grid>

          {/* Execution Settings */}
          <Grid item xs={6}>
            <FormControl fullWidth>
              <InputLabel>Execution Type</InputLabel>
              <Select
                value={formData.execution_type || 'realtime'}
                label="Execution Type"
                onChange={(e) => handleChange('execution_type', e.target.value)}
              >
                <MenuItem value="realtime">Realtime</MenuItem>
                <MenuItem value="batch">Batch</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={6}>
            <FormControl fullWidth>
              <InputLabel>Engine</InputLabel>
              <Select
                value={formData.engine || 'internal'}
                label="Engine"
                onChange={(e) => handleChange('engine', e.target.value)}
              >
                <MenuItem value="internal">Internal</MenuItem>
                <MenuItem value="cube">Cube</MenuItem>
                <MenuItem value="spark">Spark</MenuItem>
              </Select>
            </FormControl>
          </Grid>
          <Grid item xs={12}>
            <Typography variant="subtitle2" gutterBottom>SQL / Formula</Typography>
            <Paper variant="outlined" sx={{ p: 1, borderColor: errors.sql ? 'error.main' : 'divider' }}>
              <TextField
                value={formData.sql || ''}
                onChange={(e) => handleChange('sql', e.target.value)}
                fullWidth
                multiline
                rows={4}
                variant="standard"
                placeholder="e.g., SUM(revenue) / SUM(users)"
                InputProps={{ disableUnderline: true, sx: { fontFamily: 'monospace' } }}
              />
            </Paper>
            {errors.sql && <Typography color="error" variant="caption">{errors.sql}</Typography>}
          </Grid>
        </Grid>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained">
          {isNew ? 'Add to Library' : 'Save Changes'}
        </Button>
      </DialogActions>
    </Dialog>
  );
};

  interface TestResultDialogProps {
    open: boolean;
    onClose: () => void;
    loading: boolean;
    calculation: CalculationOption | null;
    requestBody: any;
    responseBody: any;
    error: string | null;
  }

  const TestResultDialog: React.FC<TestResultDialogProps & { onLoadSample?: () => void }> = ({ open, onClose, loading, calculation, requestBody, responseBody, error, onLoadSample }) => {
    const copyText = async (text: string) => {
      try {
        await navigator.clipboard.writeText(text);
      } catch (e) {
        // ignore
      }
    };

    return (
      <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
        <ModalHeader title={`Test: ${calculation?.title || 'Calculation'}`} onClose={onClose} />
        <DialogContent>
          <Box sx={{ mb: 2, display: 'flex', gap: 1, alignItems: 'center' }}>
            <Button size="small" variant="outlined" onClick={onLoadSample}>
              Load sample payload
            </Button>
            <Typography variant="caption" color="text.secondary">(fills an IRR/XIRR example you can run immediately)</Typography>
          </Box>
          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2">Request</Typography>
            <Paper variant="outlined" sx={{ p: 1, mt: 1, maxHeight: 220, overflow: 'auto' }}>
              <Typography component="pre" sx={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
                {requestBody ? JSON.stringify(requestBody, null, 2) : '—'}
              </Typography>
            </Paper>
          </Box>

          <Box sx={{ mb: 2 }}>
            <Typography variant="subtitle2">Response</Typography>
            <Paper variant="outlined" sx={{ p: 1, mt: 1, maxHeight: 320, overflow: 'auto' }}>
              {loading ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', p: 2 }}><CircularProgress /></Box>
              ) : error ? (
                <Alert severity="error">{error}</Alert>
              ) : (
                <Typography component="pre" sx={{ whiteSpace: 'pre-wrap', fontFamily: 'monospace' }}>
                  {responseBody ? (typeof responseBody === 'string' ? responseBody : JSON.stringify(responseBody, null, 2)) : '—'}
                </Typography>
              )}
            </Paper>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button
            startIcon={<ContentCopyIcon />}
            onClick={async () => copyText(requestBody ? JSON.stringify(requestBody, null, 2) : '')}
          >
            Copy Request
          </Button>
          <Button
            startIcon={<ContentCopyIcon />}
            onClick={async () => copyText(responseBody ? (typeof responseBody === 'string' ? responseBody : JSON.stringify(responseBody, null, 2)) : '')}
            disabled={!responseBody}
          >
            Copy Response
          </Button>
          <Box sx={{ flex: '1 1 auto' }} />
          <Button onClick={onClose}>Close</Button>
        </DialogActions>
      </Dialog>
    );
  };

export default CalculationsLibraryPage;
