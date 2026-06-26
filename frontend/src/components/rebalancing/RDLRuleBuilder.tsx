import React, { useState, useCallback, useMemo } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Button,
  Chip,
  IconButton,
  Tooltip,
  Accordion,
  AccordionSummary,
  AccordionDetails,
  Switch,
  FormControlLabel,
  Alert,
  Tabs,
  Tab,
  Paper,
  Divider,
  Grid,
  Slider,
  FormHelperText,
} from '@mui/material';
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  PlayArrow as TestIcon,
  Save as SaveIcon,
  Code as CodeIcon,
  Preview as PreviewIcon,
  ExpandMore as ExpandMoreIcon,
  ContentCopy as CopyIcon,
  Refresh as RefreshIcon,
  Check as CheckIcon,
  Warning as WarningIcon,
} from '@mui/icons-material';

// =============================================================================
// TYPES
// =============================================================================

export type RuleType =
  | 'tax_loss_harvesting'
  | 'wash_sale'
  | 'cppi_floor'
  | 'drift_trigger'
  | 'sector_limit'
  | 'concentration_limit'
  | 'esg_restriction'
  | 'cash_flow'
  | 'custom';

export interface RuleDefinition {
  tenant_id: string;
  rule_id: string;
  type: RuleType;
  version: string;
  name: string;
  description: string;
  jurisdiction?: string;
  parameters: Record<string, any>;
  expression: string;
  scoring_formula?: string;
  wash_sale_config?: WashSaleConfig;
  substitute_asset_rules?: SubstituteAssetRule[];
  schedule?: ScheduleConfig;
  notifications?: NotificationConfig;
  active: boolean;
  effective_from?: string;
  effective_to?: string;
}

export interface WashSaleConfig {
  enabled: boolean;
  window_days_before: number;
  window_days_after: number;
  check_household: boolean;
  check_ira: boolean;
  check_spouse: boolean;
}

export interface SubstituteAssetRule {
  original_ticker: string;
  substitute_ticker: string;
  correlation_min: number;
  same_sector: boolean;
  esg_compatible: boolean;
}

export interface ScheduleConfig {
  type: 'continuous' | 'daily' | 'weekly' | 'monthly' | 'quarterly';
  timezone: string;
  days_of_week?: number[];
  time_of_day?: string;
}

export interface NotificationConfig {
  email: boolean;
  in_app: boolean;
  webhook_url?: string;
  threshold_triggered: boolean;
}

export interface RDLRuleBuilderProps {
  tenantId: string;
  datasourceId?: string;
  initialRule?: Partial<RuleDefinition>;
  onSave?: (rule: RuleDefinition) => Promise<void>;
  onTest?: (rule: RuleDefinition) => Promise<{ passed: boolean; score?: number; message?: string }>;
  onCancel?: () => void;
}

// =============================================================================
// RULE TYPE CONFIGURATIONS
// =============================================================================

const RULE_TYPE_CONFIGS: Record<RuleType, {
  label: string;
  description: string;
  icon: string;
  defaultExpression: string;
  parameterSchema: Array<{
    name: string;
    label: string;
    type: 'number' | 'string' | 'boolean' | 'select' | 'slider';
    default: any;
    min?: number;
    max?: number;
    step?: number;
    options?: Array<{ value: any; label: string }>;
    helpText?: string;
    required?: boolean;
  }>;
}> = {
  tax_loss_harvesting: {
    label: 'Tax-Loss Harvesting',
    description: 'Identify positions with unrealized losses for tax optimization',
    icon: '💰',
    defaultExpression: 'input.unrealized_loss_pct >= params.min_loss_pct && input.days_held >= params.min_holding_days',
    parameterSchema: [
      { name: 'min_loss_pct', label: 'Minimum Loss %', type: 'slider', default: 5, min: 1, max: 50, step: 1, helpText: 'Minimum unrealized loss percentage to trigger' },
      { name: 'min_loss_usd', label: 'Minimum Loss ($)', type: 'number', default: 1000, min: 0, helpText: 'Minimum dollar amount of loss' },
      { name: 'min_holding_days', label: 'Minimum Holding Days', type: 'number', default: 31, min: 1, helpText: 'Days position must be held before harvesting' },
      { name: 'long_term_days', label: 'Long-Term Threshold', type: 'number', default: 366, min: 1, helpText: 'Days for long-term capital gains treatment' },
      { name: 'annual_loss_limit', label: 'Annual Loss Limit ($)', type: 'number', default: 3000, min: 0, helpText: 'Maximum annual loss deduction (US: $3,000)' },
      { name: 'estimated_tax_rate', label: 'Estimated Tax Rate', type: 'slider', default: 35, min: 0, max: 50, step: 0.5, helpText: 'Client estimated marginal tax rate' },
    ],
  },
  wash_sale: {
    label: 'Wash Sale Prevention',
    description: 'Prevent wash sale violations by tracking substantially identical purchases',
    icon: '🚫',
    defaultExpression: '!isInWashSaleWindow(input.household_id, input.ticker) && input.days_since_sale >= params.window_days',
    parameterSchema: [
      { name: 'window_days', label: 'Wash Sale Window (Days)', type: 'number', default: 31, min: 1, helpText: 'Days before/after sale to check for purchases' },
      { name: 'check_household', label: 'Check Household Accounts', type: 'boolean', default: true, helpText: 'Include all household accounts in check' },
      { name: 'check_ira', label: 'Check IRA Accounts', type: 'boolean', default: true, helpText: 'Include IRA accounts in wash sale check' },
      { name: 'check_options', label: 'Check Options', type: 'boolean', default: true, helpText: 'Include options on same underlying' },
      { name: 'etf_overlap_threshold', label: 'ETF Overlap Threshold', type: 'slider', default: 80, min: 50, max: 100, step: 5, helpText: 'ETF holdings overlap % considered substantially identical' },
    ],
  },
  cppi_floor: {
    label: 'CPPI Floor Protection',
    description: 'Constant Proportion Portfolio Insurance floor monitoring',
    icon: '🛡️',
    defaultExpression: 'input.nav / input.floor >= params.cushion_min && input.nav / input.floor <= params.cushion_max',
    parameterSchema: [
      { name: 'floor_pct', label: 'Floor (%)', type: 'slider', default: 80, min: 50, max: 95, step: 1, helpText: 'Minimum portfolio value to protect' },
      { name: 'multiplier', label: 'Multiplier (m)', type: 'slider', default: 4, min: 1, max: 10, step: 0.5, helpText: 'Risky asset multiplier for cushion' },
      { name: 'cushion_min', label: 'Min Cushion Ratio', type: 'number', default: 1.02, min: 1, helpText: 'Minimum cushion before rebalancing' },
      { name: 'cushion_max', label: 'Max Cushion Ratio', type: 'number', default: 1.15, min: 1, helpText: 'Maximum cushion before deploying more risk' },
      { name: 'risk_free_ticker', label: 'Risk-Free Asset', type: 'string', default: 'SHY', helpText: 'Ticker for risk-free asset allocation' },
      { name: 'emergency_liquidation', label: 'Emergency Liquidation', type: 'boolean', default: true, helpText: 'Enable emergency liquidation at floor breach' },
    ],
  },
  drift_trigger: {
    label: 'Drift-Based Rebalancing',
    description: 'Trigger rebalancing when allocation drifts from target',
    icon: '⚖️',
    defaultExpression: 'input.drift_pct >= params.drift_threshold || input.asset_drift_pct >= params.asset_drift_threshold',
    parameterSchema: [
      { name: 'drift_threshold', label: 'Portfolio Drift Threshold (%)', type: 'slider', default: 5, min: 1, max: 20, step: 0.5, helpText: 'Overall portfolio drift to trigger rebalance' },
      { name: 'asset_drift_threshold', label: 'Asset Drift Threshold (%)', type: 'slider', default: 10, min: 1, max: 30, step: 1, helpText: 'Single asset drift to trigger rebalance' },
      { name: 'min_trade_size', label: 'Min Trade Size ($)', type: 'number', default: 500, min: 0, helpText: 'Minimum trade size to execute' },
      { name: 'rebalance_frequency', label: 'Rebalance Check Frequency', type: 'select', default: 'daily', options: [
        { value: 'continuous', label: 'Continuous' },
        { value: 'daily', label: 'Daily' },
        { value: 'weekly', label: 'Weekly' },
        { value: 'monthly', label: 'Monthly' },
      ], helpText: 'How often to check drift' },
    ],
  },
  sector_limit: {
    label: 'Sector Concentration Limit',
    description: 'Limit exposure to any single sector',
    icon: '📊',
    defaultExpression: 'input.sector_weight <= params.max_sector_weight',
    parameterSchema: [
      { name: 'max_sector_weight', label: 'Max Sector Weight (%)', type: 'slider', default: 25, min: 5, max: 50, step: 1, helpText: 'Maximum allocation to any sector' },
      { name: 'min_sectors', label: 'Min Sectors Required', type: 'number', default: 5, min: 1, helpText: 'Minimum number of sectors to hold' },
      { name: 'exclude_cash', label: 'Exclude Cash', type: 'boolean', default: true, helpText: 'Exclude cash from sector calculations' },
    ],
  },
  concentration_limit: {
    label: 'Position Concentration Limit',
    description: 'Limit exposure to any single position',
    icon: '🎯',
    defaultExpression: 'input.position_weight <= params.max_position_weight',
    parameterSchema: [
      { name: 'max_position_weight', label: 'Max Position Weight (%)', type: 'slider', default: 10, min: 1, max: 50, step: 1, helpText: 'Maximum allocation to any position' },
      { name: 'issuer_limit', label: 'Issuer Limit (%)', type: 'slider', default: 15, min: 5, max: 50, step: 1, helpText: 'Maximum allocation to any issuer' },
      { name: 'top_n_limit', label: 'Top N Holdings Limit (%)', type: 'slider', default: 50, min: 20, max: 80, step: 5, helpText: 'Maximum allocation to top N holdings' },
      { name: 'top_n_count', label: 'Top N Count', type: 'number', default: 10, min: 1, helpText: 'Number of top holdings to consider' },
    ],
  },
  esg_restriction: {
    label: 'ESG Restriction',
    description: 'Enforce ESG screens and exclusions',
    icon: '🌱',
    defaultExpression: 'input.esg_score >= params.min_esg_score && !input.excluded_industry',
    parameterSchema: [
      { name: 'min_esg_score', label: 'Minimum ESG Score', type: 'slider', default: 50, min: 0, max: 100, step: 5, helpText: 'Minimum ESG score for holdings' },
      { name: 'exclude_tobacco', label: 'Exclude Tobacco', type: 'boolean', default: true },
      { name: 'exclude_weapons', label: 'Exclude Weapons', type: 'boolean', default: true },
      { name: 'exclude_gambling', label: 'Exclude Gambling', type: 'boolean', default: false },
      { name: 'exclude_fossil_fuels', label: 'Exclude Fossil Fuels', type: 'boolean', default: false },
      { name: 'carbon_intensity_max', label: 'Max Carbon Intensity', type: 'number', default: 100, min: 0, helpText: 'Maximum tons CO2e per $M revenue' },
    ],
  },
  cash_flow: {
    label: 'Cash Flow Management',
    description: 'Manage cash flows and liquidity requirements',
    icon: '💵',
    defaultExpression: 'input.cash_balance >= params.min_cash_pct * input.nav',
    parameterSchema: [
      { name: 'min_cash_pct', label: 'Minimum Cash (%)', type: 'slider', default: 2, min: 0, max: 20, step: 0.5, helpText: 'Minimum cash allocation' },
      { name: 'max_cash_pct', label: 'Maximum Cash (%)', type: 'slider', default: 10, min: 0, max: 30, step: 0.5, helpText: 'Maximum cash allocation' },
      { name: 'sweep_threshold', label: 'Sweep Threshold ($)', type: 'number', default: 10000, min: 0, helpText: 'Cash above this triggers investment' },
      { name: 'emergency_reserve_months', label: 'Emergency Reserve (Months)', type: 'number', default: 3, min: 0, helpText: 'Months of expenses to keep liquid' },
    ],
  },
  custom: {
    label: 'Custom Rule',
    description: 'Create a custom rule with CEL expression',
    icon: '⚙️',
    defaultExpression: 'true',
    parameterSchema: [],
  },
};

const JURISDICTIONS = [
  { value: 'US', label: 'United States' },
  { value: 'UK', label: 'United Kingdom' },
  { value: 'EU', label: 'European Union' },
  { value: 'CA', label: 'Canada' },
  { value: 'AU', label: 'Australia' },
  { value: 'JP', label: 'Japan' },
  { value: 'GLOBAL', label: 'Global' },
];

// =============================================================================
// MAIN COMPONENT
// =============================================================================

export const RDLRuleBuilder: React.FC<RDLRuleBuilderProps> = ({
  tenantId,
  datasourceId,
  initialRule,
  onSave,
  onTest,
  onCancel,
}) => {
  // State
  const [activeTab, setActiveTab] = useState(0);
  const [rule, setRule] = useState<Partial<RuleDefinition>>({
    tenant_id: tenantId,
    rule_id: '',
    type: 'tax_loss_harvesting',
    version: '1.0.0',
    name: '',
    description: '',
    jurisdiction: 'US',
    parameters: {},
    expression: RULE_TYPE_CONFIGS.tax_loss_harvesting.defaultExpression,
    active: true,
    ...initialRule,
  });
  
  const [testResult, setTestResult] = useState<{ passed: boolean; score?: number; message?: string } | null>(null);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [isTesting, setIsTesting] = useState(false);

  // Get current rule type config
  const currentConfig = useMemo(() => {
    return RULE_TYPE_CONFIGS[rule.type || 'tax_loss_harvesting'];
  }, [rule.type]);

  // Initialize parameters with defaults when rule type changes
  const handleRuleTypeChange = useCallback((newType: RuleType) => {
    const config = RULE_TYPE_CONFIGS[newType];
    const defaultParams: Record<string, any> = {};
    
    config.parameterSchema.forEach(param => {
      defaultParams[param.name] = param.default;
    });

    setRule(prev => ({
      ...prev,
      type: newType,
      expression: config.defaultExpression,
      parameters: defaultParams,
    }));
    setTestResult(null);
  }, []);

  // Update a single parameter
  const handleParameterChange = useCallback((name: string, value: any) => {
    setRule(prev => ({
      ...prev,
      parameters: {
        ...prev.parameters,
        [name]: value,
      },
    }));
  }, []);

  // Validate rule before saving
  const validateRule = useCallback((): boolean => {
    const errors: string[] = [];

    if (!rule.rule_id?.trim()) {
      errors.push('Rule ID is required');
    }
    if (!rule.name?.trim()) {
      errors.push('Rule name is required');
    }
    if (!rule.expression?.trim()) {
      errors.push('CEL expression is required');
    }

    setValidationErrors(errors);
    return errors.length === 0;
  }, [rule]);

  // Handle save
  const handleSave = useCallback(async () => {
    if (!validateRule()) return;
    
    setIsSaving(true);
    try {
      if (onSave) {
        await onSave(rule as RuleDefinition);
      }
    } catch (error) {
      console.error('Failed to save rule:', error);
      setValidationErrors(['Failed to save rule. Please try again.']);
    } finally {
      setIsSaving(false);
    }
  }, [rule, validateRule, onSave]);

  // Handle test
  const handleTest = useCallback(async () => {
    if (!validateRule()) return;
    
    setIsTesting(true);
    setTestResult(null);
    try {
      if (onTest) {
        const result = await onTest(rule as RuleDefinition);
        setTestResult(result);
      } else {
        // Mock test result
        setTestResult({
          passed: true,
          score: 0.85,
          message: 'Rule syntax is valid',
        });
      }
    } catch (error) {
      setTestResult({
        passed: false,
        message: `Test failed: ${error}`,
      });
    } finally {
      setIsTesting(false);
    }
  }, [rule, validateRule, onTest]);

  // Generate JSON preview
  const jsonPreview = useMemo(() => {
    return JSON.stringify(rule, null, 2);
  }, [rule]);

  return (
    <Box sx={{ width: '100%', maxWidth: 1200 }}>
      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', mb: 3, gap: 2 }}>
        <Typography variant="h5" sx={{ fontWeight: 600 }}>
          {currentConfig.icon} Rule Builder
        </Typography>
        <Chip
          label={currentConfig.label}
          color="primary"
          variant="outlined"
          size="small"
        />
      </Box>

      {/* Validation Errors */}
      {validationErrors.length > 0 && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {validationErrors.map((error, idx) => (
            <div key={idx}>{error}</div>
          ))}
        </Alert>
      )}

      {/* Test Result */}
      {testResult && (
        <Alert
          severity={testResult.passed ? 'success' : 'error'}
          icon={testResult.passed ? <CheckIcon /> : <WarningIcon />}
          sx={{ mb: 2 }}
        >
          {testResult.message}
          {testResult.score !== undefined && (
            <Typography variant="body2" sx={{ mt: 0.5 }}>
              Score: {(testResult.score * 100).toFixed(1)}%
            </Typography>
          )}
        </Alert>
      )}

      {/* Tabs */}
      <Paper sx={{ mb: 2 }}>
        <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)}>
          <Tab label="Configuration" />
          <Tab label="Parameters" />
          <Tab label="Expression" />
          <Tab label="Advanced" />
          <Tab label="Preview" icon={<PreviewIcon />} iconPosition="start" />
        </Tabs>
      </Paper>

      {/* Tab Panels */}
      <Box>
        {/* Tab 0: Basic Configuration */}
        {activeTab === 0 && (
          <Card>
            <CardContent>
              <Grid container spacing={3}>
                <Grid item xs={12} md={6}>
                  <FormControl fullWidth>
                    <InputLabel>Rule Type</InputLabel>
                    <Select
                      value={rule.type}
                      label="Rule Type"
                      onChange={(e) => handleRuleTypeChange(e.target.value as RuleType)}
                    >
                      {Object.entries(RULE_TYPE_CONFIGS).map(([key, config]) => (
                        <MenuItem key={key} value={key}>
                          {config.icon} {config.label}
                        </MenuItem>
                      ))}
                    </Select>
                    <FormHelperText>{currentConfig.description}</FormHelperText>
                  </FormControl>
                </Grid>

                <Grid item xs={12} md={6}>
                  <FormControl fullWidth>
                    <InputLabel>Jurisdiction</InputLabel>
                    <Select
                      value={rule.jurisdiction || 'US'}
                      label="Jurisdiction"
                      onChange={(e) => setRule(prev => ({ ...prev, jurisdiction: e.target.value }))}
                    >
                      {JURISDICTIONS.map(j => (
                        <MenuItem key={j.value} value={j.value}>{j.label}</MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    required
                    label="Rule ID"
                    value={rule.rule_id}
                    onChange={(e) => setRule(prev => ({ ...prev, rule_id: e.target.value }))}
                    helperText="Unique identifier (e.g., TLH_US_STANDARD)"
                    placeholder="TLH_US_STANDARD"
                  />
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    label="Version"
                    value={rule.version}
                    onChange={(e) => setRule(prev => ({ ...prev, version: e.target.value }))}
                    placeholder="1.0.0"
                  />
                </Grid>

                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    required
                    label="Rule Name"
                    value={rule.name}
                    onChange={(e) => setRule(prev => ({ ...prev, name: e.target.value }))}
                    placeholder="US Tax-Loss Harvesting - Standard"
                  />
                </Grid>

                <Grid item xs={12}>
                  <TextField
                    fullWidth
                    multiline
                    rows={3}
                    label="Description"
                    value={rule.description}
                    onChange={(e) => setRule(prev => ({ ...prev, description: e.target.value }))}
                    placeholder="Describe what this rule does and when it should trigger..."
                  />
                </Grid>

                <Grid item xs={12}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={rule.active}
                        onChange={(e) => setRule(prev => ({ ...prev, active: e.target.checked }))}
                      />
                    }
                    label="Rule Active"
                  />
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        )}

        {/* Tab 1: Parameters */}
        {activeTab === 1 && (
          <Card>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 600 }}>
                {currentConfig.label} Parameters
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
                Configure the parameters for this rule type. These values will be available as <code>params.*</code> in the CEL expression.
              </Typography>

              <Grid container spacing={3}>
                {currentConfig.parameterSchema.map(param => (
                  <Grid item xs={12} md={6} key={param.name}>
                    {param.type === 'slider' ? (
                      <Box>
                        <Typography variant="body2" gutterBottom>
                          {param.label}: {rule.parameters?.[param.name] ?? param.default}
                          {param.name.includes('pct') || param.name.includes('rate') || param.name.includes('weight') ? '%' : ''}
                        </Typography>
                        <Slider
                          value={rule.parameters?.[param.name] ?? param.default}
                          onChange={(_, value) => handleParameterChange(param.name, value)}
                          min={param.min}
                          max={param.max}
                          step={param.step}
                          marks
                          valueLabelDisplay="auto"
                        />
                        {param.helpText && (
                          <FormHelperText>{param.helpText}</FormHelperText>
                        )}
                      </Box>
                    ) : param.type === 'boolean' ? (
                      <FormControlLabel
                        control={
                          <Switch
                            checked={rule.parameters?.[param.name] ?? param.default}
                            onChange={(e) => handleParameterChange(param.name, e.target.checked)}
                          />
                        }
                        label={
                          <Box>
                            {param.label}
                            {param.helpText && (
                              <Typography variant="caption" display="block" color="text.secondary">
                                {param.helpText}
                              </Typography>
                            )}
                          </Box>
                        }
                      />
                    ) : param.type === 'select' ? (
                      <FormControl fullWidth>
                        <InputLabel>{param.label}</InputLabel>
                        <Select
                          value={rule.parameters?.[param.name] ?? param.default}
                          label={param.label}
                          onChange={(e) => handleParameterChange(param.name, e.target.value)}
                        >
                          {param.options?.map(opt => (
                            <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>
                          ))}
                        </Select>
                        {param.helpText && <FormHelperText>{param.helpText}</FormHelperText>}
                      </FormControl>
                    ) : (
                      <TextField
                        fullWidth
                        type={param.type}
                        label={param.label}
                        value={rule.parameters?.[param.name] ?? param.default}
                        onChange={(e) => handleParameterChange(
                          param.name,
                          param.type === 'number' ? parseFloat(e.target.value) : e.target.value
                        )}
                        helperText={param.helpText}
                        inputProps={param.type === 'number' ? { min: param.min, max: param.max } : undefined}
                      />
                    )}
                  </Grid>
                ))}

                {currentConfig.parameterSchema.length === 0 && (
                  <Grid item xs={12}>
                    <Alert severity="info">
                      Custom rules have no predefined parameters. Define your parameters in the JSON below or in the Expression tab.
                    </Alert>
                    <TextField
                      fullWidth
                      multiline
                      rows={6}
                      label="Custom Parameters (JSON)"
                      value={JSON.stringify(rule.parameters || {}, null, 2)}
                      onChange={(e) => {
                        try {
                          setRule(prev => ({ ...prev, parameters: JSON.parse(e.target.value) }));
                        } catch { /* ignore parse errors while typing */ }
                      }}
                      sx={{ mt: 2, fontFamily: 'monospace' }}
                    />
                  </Grid>
                )}
              </Grid>
            </CardContent>
          </Card>
        )}

        {/* Tab 2: Expression */}
        {activeTab === 2 && (
          <Card>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 600 }}>
                CEL Expression
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Write the Common Expression Language (CEL) expression that determines when this rule triggers.
                Use <code>input.*</code> for evaluation data and <code>params.*</code> for configured parameters.
              </Typography>

              <TextField
                fullWidth
                multiline
                rows={6}
                value={rule.expression}
                onChange={(e) => setRule(prev => ({ ...prev, expression: e.target.value }))}
                placeholder="input.unrealized_loss_pct >= params.min_loss_pct && input.days_held >= params.min_holding_days"
                sx={{ fontFamily: 'monospace', fontSize: 14 }}
              />

              <Divider sx={{ my: 3 }} />

              <Typography variant="subtitle2" gutterBottom>
                Available Variables
              </Typography>
              
              <Accordion>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography variant="body2">Input Variables (input.*)</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <Typography variant="body2" component="div" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                    input.portfolio_id - Portfolio identifier<br/>
                    input.ticker - Security ticker<br/>
                    input.unrealized_loss_pct - Unrealized loss percentage<br/>
                    input.unrealized_loss_usd - Unrealized loss in USD<br/>
                    input.cost_basis - Position cost basis<br/>
                    input.current_value - Current market value<br/>
                    input.days_held - Days position has been held<br/>
                    input.account_type - Account type (TAXABLE, IRA, etc.)<br/>
                    input.nav - Portfolio net asset value<br/>
                    input.drift_pct - Portfolio drift percentage<br/>
                    input.sector_weight - Position sector weight<br/>
                    input.position_weight - Position weight in portfolio
                  </Typography>
                </AccordionDetails>
              </Accordion>

              <Accordion>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography variant="body2">Custom Functions</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  <Typography variant="body2" component="div" sx={{ fontFamily: 'monospace', fontSize: 12 }}>
                    isInWashSaleWindow(household_id, ticker) - Check if in wash sale window<br/>
                    hasRecentPurchase(household_id, ticker, days) - Check for recent purchases<br/>
                    daysSince(timestamp) - Calculate days since a date
                  </Typography>
                </AccordionDetails>
              </Accordion>

              <Box sx={{ mt: 2 }}>
                <Button
                  variant="outlined"
                  startIcon={<RefreshIcon />}
                  onClick={() => setRule(prev => ({ ...prev, expression: currentConfig.defaultExpression }))}
                >
                  Reset to Default
                </Button>
              </Box>
            </CardContent>
          </Card>
        )}

        {/* Tab 3: Advanced */}
        {activeTab === 3 && (
          <Card>
            <CardContent>
              <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 600 }}>
                Advanced Configuration
              </Typography>

              <Grid container spacing={3}>
                <Grid item xs={12}>
                  <Typography variant="subtitle2" gutterBottom>Scoring Formula (Optional)</Typography>
                  <TextField
                    fullWidth
                    multiline
                    rows={2}
                    value={rule.scoring_formula || ''}
                    onChange={(e) => setRule(prev => ({ ...prev, scoring_formula: e.target.value }))}
                    placeholder="input.unrealized_loss_usd * params.estimated_tax_rate / 100"
                    helperText="CEL expression to calculate a score/priority for this rule trigger"
                    sx={{ fontFamily: 'monospace' }}
                  />
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    type="date"
                    label="Effective From"
                    value={rule.effective_from || ''}
                    onChange={(e) => setRule(prev => ({ ...prev, effective_from: e.target.value }))}
                    InputLabelProps={{ shrink: true }}
                    helperText="Rule becomes active from this date"
                  />
                </Grid>

                <Grid item xs={12} md={6}>
                  <TextField
                    fullWidth
                    type="date"
                    label="Effective To"
                    value={rule.effective_to || ''}
                    onChange={(e) => setRule(prev => ({ ...prev, effective_to: e.target.value }))}
                    InputLabelProps={{ shrink: true }}
                    helperText="Rule expires after this date"
                  />
                </Grid>

                {/* Wash Sale Config for TLH rules */}
                {(rule.type === 'tax_loss_harvesting' || rule.type === 'wash_sale') && (
                  <Grid item xs={12}>
                    <Accordion>
                      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                        <Typography>Wash Sale Configuration</Typography>
                      </AccordionSummary>
                      <AccordionDetails>
                        <Grid container spacing={2}>
                          <Grid item xs={12} md={4}>
                            <TextField
                              fullWidth
                              type="number"
                              label="Window Days Before"
                              value={rule.wash_sale_config?.window_days_before ?? 30}
                              onChange={(e) => setRule(prev => ({
                                ...prev,
                                wash_sale_config: {
                                  ...prev.wash_sale_config,
                                  window_days_before: parseInt(e.target.value),
                                } as WashSaleConfig,
                              }))}
                            />
                          </Grid>
                          <Grid item xs={12} md={4}>
                            <TextField
                              fullWidth
                              type="number"
                              label="Window Days After"
                              value={rule.wash_sale_config?.window_days_after ?? 30}
                              onChange={(e) => setRule(prev => ({
                                ...prev,
                                wash_sale_config: {
                                  ...prev.wash_sale_config,
                                  window_days_after: parseInt(e.target.value),
                                } as WashSaleConfig,
                              }))}
                            />
                          </Grid>
                          <Grid item xs={12} md={4}>
                            <FormControlLabel
                              control={
                                <Switch
                                  checked={rule.wash_sale_config?.check_household ?? true}
                                  onChange={(e) => setRule(prev => ({
                                    ...prev,
                                    wash_sale_config: {
                                      ...prev.wash_sale_config,
                                      check_household: e.target.checked,
                                    } as WashSaleConfig,
                                  }))}
                                />
                              }
                              label="Check Household"
                            />
                          </Grid>
                        </Grid>
                      </AccordionDetails>
                    </Accordion>
                  </Grid>
                )}

                {/* Notifications */}
                <Grid item xs={12}>
                  <Accordion>
                    <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                      <Typography>Notifications</Typography>
                    </AccordionSummary>
                    <AccordionDetails>
                      <Grid container spacing={2}>
                        <Grid item xs={12} md={4}>
                          <FormControlLabel
                            control={
                              <Switch
                                checked={rule.notifications?.email ?? true}
                                onChange={(e) => setRule(prev => ({
                                  ...prev,
                                  notifications: {
                                    ...prev.notifications,
                                    email: e.target.checked,
                                  } as NotificationConfig,
                                }))}
                              />
                            }
                            label="Email Notifications"
                          />
                        </Grid>
                        <Grid item xs={12} md={4}>
                          <FormControlLabel
                            control={
                              <Switch
                                checked={rule.notifications?.in_app ?? true}
                                onChange={(e) => setRule(prev => ({
                                  ...prev,
                                  notifications: {
                                    ...prev.notifications,
                                    in_app: e.target.checked,
                                  } as NotificationConfig,
                                }))}
                              />
                            }
                            label="In-App Notifications"
                          />
                        </Grid>
                        <Grid item xs={12}>
                          <TextField
                            fullWidth
                            label="Webhook URL"
                            value={rule.notifications?.webhook_url || ''}
                            onChange={(e) => setRule(prev => ({
                              ...prev,
                              notifications: {
                                ...prev.notifications,
                                webhook_url: e.target.value,
                              } as NotificationConfig,
                            }))}
                            placeholder="https://your-webhook.com/notify"
                          />
                        </Grid>
                      </Grid>
                    </AccordionDetails>
                  </Accordion>
                </Grid>
              </Grid>
            </CardContent>
          </Card>
        )}

        {/* Tab 4: JSON Preview */}
        {activeTab === 4 && (
          <Card>
            <CardContent>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                  JSON Preview
                </Typography>
                <Tooltip title="Copy to clipboard">
                  <IconButton
                    onClick={() => navigator.clipboard.writeText(jsonPreview)}
                    size="small"
                  >
                    <CopyIcon />
                  </IconButton>
                </Tooltip>
              </Box>
              <Paper
                sx={{
                  p: 2,
                  bgcolor: 'grey.900',
                  color: 'grey.100',
                  fontFamily: 'monospace',
                  fontSize: 12,
                  overflow: 'auto',
                  maxHeight: 500,
                }}
              ><pre>{jsonPreview}</pre>
              </Paper>
            </CardContent>
          </Card>
        )}
      </Box>

      {/* Actions */}
      <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 2, mt: 3 }}>
        {onCancel && (
          <Button variant="outlined" onClick={onCancel}>
            Cancel
          </Button>
        )}
        <Button
          variant="outlined"
          startIcon={<TestIcon />}
          onClick={handleTest}
          disabled={isTesting}
        >
          {isTesting ? 'Testing...' : 'Test Rule'}
        </Button>
        <Button
          variant="contained"
          startIcon={<SaveIcon />}
          onClick={handleSave}
          disabled={isSaving}
        >
          {isSaving ? 'Saving...' : 'Save Rule'}
        </Button>
      </Box>
    </Box>
  );
};

export default RDLRuleBuilder;
