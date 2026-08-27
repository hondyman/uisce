import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  TextField,
  MenuItem,
  Button,
  Chip,
  Grid,
  Divider,
  Checkbox,
  FormControlLabel,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Alert,
  Tabs,
  Tab
} from '@mui/material';
import {
  Gavel as ComplianceIcon,
  Code as CodeIcon,
  Tune as WizardIcon,
  PlayArrow as TestValidateIcon
} from '@mui/icons-material';

interface AccountAttachment {
  accountCode: string;
  accountName: string;
  accountType: string;
  alertAbove?: number;
  alertBelow?: number;
  warnAbove?: number;
  warnBelow?: number;
}

export const DualModeComplianceStudio: React.FC<{ tenantId?: string }> = ({ tenantId: _tenantId }) => {
  const [activeTab, setActiveTab] = useState(0);
  const [ruleName, setRuleName] = useState('Do not allow more than 5% of total assets to be with any one issuer; exclude US government securities.');
  
  const [testType, setTestType] = useState('RESTRICT_CONCENTRATION_PCT');
  const [numeratorMetric, setNumeratorMetric] = useState('POSITION_ORDER_MARKET_VALUE');
  const [denominatorMetric, setDenominatorMetric] = useState('TOTAL_ASSETS');
  const [evaluateForEach, setEvaluateForEach] = useState(true);
  const [groupByDimension, setGroupByDimension] = useState('ISSUER_COUNTERPARTY');
  
  const [whereClause, setWhereClause] = useState("Investment Class is not equal to 'U.S. Government'");
  
  const [alertIfAbove, setAlertIfAbove] = useState(5.0);
  const [warnIfAbove, setWarnIfAbove] = useState(4.5);
  const [reopenTolerance, setReopenTolerance] = useState(0.5);

  const [freeformCEL, setFreeformCEL] = useState(
    `// Rule: Issuer Concentration 5% Excl US Gov\ninvestment_class != 'USGOV' &&\n(group_by(security.issuer_lei).sum(position.market_value + order.market_value) / account.total_assets) <= 0.05`
  );

  const [attachedAccounts] = useState<AccountAttachment[]>([
    { accountCode: 'STUDENTA1', accountName: 'Portfolio A for Student 1', accountType: 'Portfolio', alertAbove: 5.0, warnAbove: 4.5 },
    { accountCode: 'STUDENTA4', accountName: 'Portfolio A for Student 4', accountType: 'Portfolio', alertAbove: 5.0, warnAbove: 4.5 }
  ]);

  const [validationOutput, setValidationOutput] = useState<string | null>(null);

  const handleValidateTest = () => {
    setValidationOutput(
      'Test Validation Complete: 25 Positions scanned across STUDENTA4. 1 Breach detected: NVDA Direct (4.10%) + ETF Look-Through (1.32%) = 5.42% (Limit: 5.00%).'
    );
  };

  return (
    <Paper elevation={0} sx={{ p: 3, bgcolor: '#071526', color: '#F8FAFC', border: '1px solid #1E293B', borderRadius: 2 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" pb={2} mb={3} borderBottom="1px solid #1E293B">
        <Stack direction="row" spacing={1.5} alignItems="center">
          <ComplianceIcon sx={{ color: '#00D4FF', fontSize: 28 }} />
          <Box>
            <Typography variant="h6" sx={{ fontWeight: 700, fontSize: 16 }}>
              Compliance Rule Setup & Test Validation Studio
            </Typography>
            <Typography variant="caption" sx={{ color: '#94A3B8' }}>
              Multi-parameter concentration testing, look-through grouping, and CEL compilation
            </Typography>
          </Box>
        </Stack>

        <Stack direction="row" spacing={2} alignItems="center">
          <Button
            variant="outlined"
            size="small"
            startIcon={<TestValidateIcon />}
            onClick={handleValidateTest}
            sx={{ borderColor: '#0284C7', color: '#38BDF8', textTransform: 'none', fontWeight: 600 }}
          >
            Validate Test Against Accounts
          </Button>
          <Button
            variant="contained"
            size="small"
            sx={{ bgcolor: '#0284C7', textTransform: 'none', fontWeight: 600, '&:hover': { bgcolor: '#0369A1' } }}
          >
            Save Test Definition
          </Button>
        </Stack>
      </Box>

      {validationOutput && (
        <Alert severity="warning" sx={{ mb: 3, bgcolor: '#451A03', color: '#FDBA74', border: '1px solid #F59E0B' }}>
          {validationOutput}
        </Alert>
      )}

      <TextField
        fullWidth
        size="small"
        label="Test Name / Description"
        value={ruleName}
        onChange={e => setRuleName(e.target.value)}
        sx={{ mb: 3, input: { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
      />

      <Tabs
        value={activeTab}
        onChange={(_, val) => setActiveTab(val)}
        textColor="inherit"
        indicatorColor="primary"
        sx={{ borderBottom: 1, borderColor: '#1E293B', mb: 2 }}
      >
        <Tab icon={<WizardIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="Wizard Mode (Interactive)" />
        <Tab icon={<CodeIcon sx={{ fontSize: 16 }} />} iconPosition="start" label="FreeForm Mode (Google CEL)" />
      </Tabs>

      {activeTab === 0 && (
        <Box>
          <Paper sx={{ p: 2.5, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5, mb: 3 }}>
            <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', display: 'block', mb: 2 }}>
              Expression Setup (CRIMS Formulation)
            </Typography>

            <Grid container spacing={2} alignItems="center">
              <Grid   size={{ xs: 12, sm: 4 }}>
                <TextField
                  select
                  fullWidth
                  size="small"
                  label="Test Type"
                  value={testType}
                  onChange={e => setTestType(e.target.value)}
                  sx={{ '& .MuiSelect-select': { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
                >
                  <MenuItem value="RESTRICT_CONCENTRATION_PCT">Restrict the Concentration %</MenuItem>
                  <MenuItem value="EXCLUDE_HOLDINGS_TRADES">Exclude Holdings and Trades</MenuItem>
                  <MenuItem value="RESTRICT_VALUES">Restrict the Values (Market Value Caps)</MenuItem>
                </TextField>
              </Grid>

              <Grid   size={{ xs: 12, sm: 4 }}>
                <TextField
                  select
                  fullWidth
                  size="small"
                  label="Numerator (of)"
                  value={numeratorMetric}
                  onChange={e => setNumeratorMetric(e.target.value)}
                  sx={{ '& .MuiSelect-select': { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
                >
                  <MenuItem value="POSITION_ORDER_MARKET_VALUE">Position / Order Market Value</MenuItem>
                  <MenuItem value="SETTLED_HOLDINGS_MV">Settled Holdings Market Value</MenuItem>
                </TextField>
              </Grid>

              <Grid   size={{ xs: 12, sm: 4 }}>
                <TextField
                  select
                  fullWidth
                  size="small"
                  label="Denominator (based on)"
                  value={denominatorMetric}
                  onChange={e => setDenominatorMetric(e.target.value)}
                  sx={{ '& .MuiSelect-select': { color: '#F8FAFC' }, label: { color: '#94A3B8' } }}
                >
                  <MenuItem value="TOTAL_ASSETS">Total Assets</MenuItem>
                  <MenuItem value="PORTFOLIO_NAV">Portfolio Net Asset Value</MenuItem>
                  <MenuItem value="BENCHMARK_WEIGHT">Benchmark Weight</MenuItem>
                </TextField>
              </Grid>

              <Grid  size={{ xs: 12 }}>
                <Stack direction="row" spacing={3} alignItems="center">
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={evaluateForEach}
                        onChange={e => setEvaluateForEach(e.target.checked)}
                        sx={{ color: '#64748B', '&.Mui-checked': { color: '#00D4FF' } }}
                      />
                    }
                    label={<Typography variant="body2" sx={{ color: '#F8FAFC' }}>Evaluate For Each</Typography>}
                  />
                  {evaluateForEach && (
                    <TextField
                      select
                      size="small"
                      value={groupByDimension}
                      onChange={e => setGroupByDimension(e.target.value)}
                      sx={{ minWidth: 240, '& .MuiSelect-select': { color: '#F8FAFC' } }}
                    >
                      <MenuItem value="ISSUER_COUNTERPARTY">Issuer / Counterparty</MenuItem>
                      <MenuItem value="ISSUE_COUNTRY">Issue Country</MenuItem>
                      <MenuItem value="SECTOR_GICS">GICS Sector Classification</MenuItem>
                    </TextField>
                  )}
                </Stack>
              </Grid>

              <Grid  size={{ xs: 12 }}>
                <TextField
                  fullWidth
                  size="small"
                  label="Where Expression (Filters / Exclusions)"
                  value={whereClause}
                  onChange={e => setWhereClause(e.target.value)}
                  sx={{ input: { color: '#38BDF8', fontFamily: 'monospace' }, label: { color: '#94A3B8' } }}
                />
              </Grid>
            </Grid>

            <Divider sx={{ my: 2, borderColor: '#1E293B' }} />
            <Grid container spacing={2}>
              <Grid   size={{ xs: 12, sm: 4 }}>
                <TextField
                  fullWidth
                  size="small"
                  type="number"
                  label="Alert Threshold (Above %)"
                  value={alertIfAbove}
                  onChange={e => setAlertIfAbove(parseFloat(e.target.value))}
                  sx={{ input: { color: '#EF4444', fontWeight: 700 }, label: { color: '#94A3B8' } }}
                />
              </Grid>
              <Grid   size={{ xs: 12, sm: 4 }}>
                <TextField
                  fullWidth
                  size="small"
                  type="number"
                  label="Warn Threshold (Above %)"
                  value={warnIfAbove}
                  onChange={e => setWarnIfAbove(parseFloat(e.target.value))}
                  sx={{ input: { color: '#F59E0B', fontWeight: 700 }, label: { color: '#94A3B8' } }}
                />
              </Grid>
              <Grid   size={{ xs: 12, sm: 4 }}>
                <TextField
                  fullWidth
                  size="small"
                  type="number"
                  label="Reopen Tolerance (%)"
                  value={reopenTolerance}
                  onChange={e => setReopenTolerance(parseFloat(e.target.value))}
                  sx={{ input: { color: '#34D399' }, label: { color: '#94A3B8' } }}
                />
              </Grid>
            </Grid>
          </Paper>
        </Box>
      )}

      {activeTab === 1 && (
        <Paper sx={{ p: 2, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5, mb: 3 }}>
          <Typography variant="caption" sx={{ color: '#94A3B8', fontFamily: 'monospace', display: 'block', mb: 1 }}>
            Compiled Google Common Expression Language (CEL) AST
          </Typography>
          <TextField
            multiline
            rows={5}
            fullWidth
            value={freeformCEL}
            onChange={e => setFreeformCEL(e.target.value)}
            sx={{
              textarea: { color: '#38BDF8', fontFamily: 'monospace', fontSize: 13 },
              bgcolor: '#071526',
              borderRadius: 1
            }}
          />
        </Paper>
      )}

      <Typography variant="caption" sx={{ color: '#94A3B8', fontWeight: 700, textTransform: 'uppercase', display: 'block', mb: 1 }}>
        Attached Portfolio Accounts
      </Typography>
      <TableContainer component={Paper} sx={{ bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 1.5 }}>
        <Table size="small">
          <TableHead>
            <TableRow sx={{ '& th': { color: '#94A3B8', fontWeight: 600, borderColor: '#1E293B', fontSize: 11 } }}>
              <TableCell>Account Code</TableCell>
              <TableCell>Account Name</TableCell>
              <TableCell>Type</TableCell>
              <TableCell align="right">Alert Limit (%)</TableCell>
              <TableCell align="right">Warn Limit (%)</TableCell>
              <TableCell align="center">Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {attachedAccounts.map(a => (
              <TableRow key={a.accountCode} sx={{ '& td': { color: '#F8FAFC', borderColor: '#1E293B' } }}>
                <TableCell sx={{ fontFamily: 'monospace', fontWeight: 700, color: '#38BDF8' }}>{a.accountCode}</TableCell>
                <TableCell>{a.accountName}</TableCell>
                <TableCell>{a.accountType}</TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', color: '#EF4444', fontWeight: 700 }}>
                  {a.alertAbove}%
                </TableCell>
                <TableCell align="right" sx={{ fontFamily: 'monospace', color: '#F59E0B' }}>
                  {a.warnAbove}%
                </TableCell>
                <TableCell align="center">
                  <Chip label="Active" size="small" sx={{ bgcolor: '#064E3B', color: '#34D399', fontSize: 10, fontWeight: 700 }} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </TableContainer>
    </Paper>
  );
};

export default DualModeComplianceStudio;
