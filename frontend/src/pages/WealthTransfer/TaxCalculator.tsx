import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  TextField,
  Button,
  Grid,
  Card,
  CardContent,
  Slider,
  MenuItem,
  Divider,
  Table,
  TableBody,
  TableCell,
  TableRow,
  Alert,
  Chip,
} from '@mui/material';
import {
  Calculate as CalculateIcon,
  TrendingUp as GrowthIcon,
  AttachMoney as MoneyIcon,
} from '@mui/icons-material';

interface TaxCalculatorProps {
  familyId: string;
}

export const TaxCalculator: React.FC<TaxCalculatorProps> = ({ familyId }) => {
  const [inputs, setInputs] = useState({
    grossEstate: 25000000,
    state: 'NY',
    priorGifts: 0,
    charitableDeductions: 0,
    spouseAlive: true,
    growthRate: 7,
    yearsToProject: 10,
  });

  const [results, setResults] = useState<any>(null);
  const [loading, setLoading] = useState(false);

  const calculateTax = async () => {
    setLoading(true);
    try {
      // Calculate federal tax
      const federalResponse = await fetch('/api/wealth-transfer/tax/estate/federal', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          gross_estate_value: inputs.grossEstate,
          prior_lifetime_gifts: inputs.priorGifts,
          charitable_deductions: inputs.charitableDeductions,
        }),
      });
      const federalData = await federalResponse.json();

      // Calculate state tax
      const stateResponse = await fetch('/api/wealth-transfer/tax/estate/state', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          gross_estate_value: inputs.grossEstate,
          state_code: inputs.state,
        }),
      });
      const stateData = await stateResponse.json();

      setResults({
        federal: federalData,
        state: stateData,
        combined: federalData.tax_owed + stateData.tax_owed,
        effectiveRate: ((federalData.tax_owed + stateData.tax_owed) / inputs.grossEstate) * 100,
      });
    } catch (error) {
      console.error('Failed to calculate tax:', error);
    } finally {
      setLoading(false);
    }
  };

  const calculateFutureValue = () => {
    const fv = inputs.grossEstate * Math.pow(1 + inputs.growthRate / 100, inputs.yearsToProject);
    return fv;
  };

  const calculateProjectedTax = () => {
    const futureValue = calculateFutureValue();
    const federalExemption = 13990000;
    const taxableAmount = Math.max(0, futureValue - federalExemption - inputs.priorGifts);
    const federalTax = taxableAmount * 0.40;

    // Simplified state tax
    const stateTax = futureValue > 6000000 ? (futureValue - 6000000) * 0.16 : 0;

    return federalTax + stateTax;
  };

  const formatCurrency = (value: number): string => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const stateOptions = [
    { code: 'CA', name: 'California' },
    { code: 'NY', name: 'New York' },
    { code: 'FL', name: 'Florida' },
    { code: 'TX', name: 'Texas' },
    { code: 'CT', name: 'Connecticut' },
    { code: 'MA', name: 'Massachusetts' },
    { code: 'IL', name: 'Illinois' },
    { code: 'WA', name: 'Washington' },
  ];

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Estate Tax Calculator & Projections
      </Typography>

      <Grid container spacing={3}>
        {/* Input Panel */}
        <Grid item xs={12} md={6}>
          <Paper elevation={2} sx={{ p: 3 }}>
            <Typography variant="subtitle1" gutterBottom fontWeight="bold">
              Current Situation
            </Typography>

            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5 }}>
              <TextField
                label="Gross Estate Value"
                type="number"
                value={inputs.grossEstate}
                onChange={(e) => setInputs({ ...inputs, grossEstate: parseFloat(e.target.value) || 0 })}
                InputProps={{
                  startAdornment: <MoneyIcon sx={{ mr: 1, color: 'grey.500' }} />,
                }}
                fullWidth
              />

              <TextField
                label="Primary State"
                select
                value={inputs.state}
                onChange={(e) => setInputs({ ...inputs, state: e.target.value })}
                fullWidth
              >
                {stateOptions.map((state) => (
                  <MenuItem key={state.code} value={state.code}>
                    {state.name}
                  </MenuItem>
                ))}
              </TextField>

              <TextField
                label="Prior Lifetime Gifts"
                type="number"
                value={inputs.priorGifts}
                onChange={(e) => setInputs({ ...inputs, priorGifts: parseFloat(e.target.value) || 0 })}
                InputProps={{
                  startAdornment: <MoneyIcon sx={{ mr: 1, color: 'grey.500' }} />,
                }}
                fullWidth
              />

              <TextField
                label="Charitable Deductions"
                type="number"
                value={inputs.charitableDeductions}
                onChange={(e) => setInputs({ ...inputs, charitableDeductions: parseFloat(e.target.value) || 0 })}
                InputProps={{
                  startAdornment: <MoneyIcon sx={{ mr: 1, color: 'grey.500' }} />,
                }}
                fullWidth
              />

              <TextField
                label="Spouse Status"
                select
                value={inputs.spouseAlive.toString()}
                onChange={(e) => setInputs({ ...inputs, spouseAlive: e.target.value === 'true' })}
                fullWidth
              >
                <MenuItem value="true">Spouse Alive (Portability Available)</MenuItem>
                <MenuItem value="false">Single / Widowed</MenuItem>
              </TextField>

              <Divider />

              <Typography variant="subtitle2" gutterBottom>
                Projection Parameters
              </Typography>

              <Box>
                <Typography variant="caption" gutterBottom display="block">
                  Annual Growth Rate: {inputs.growthRate}%
                </Typography>
                <Slider
                  value={inputs.growthRate}
                  onChange={(_, value) => setInputs({ ...inputs, growthRate: value as number })}
                  min={0}
                  max={15}
                  step={0.5}
                  marks={[
                    { value: 0, label: '0%' },
                    { value: 7, label: '7%' },
                    { value: 15, label: '15%' },
                  ]}
                  valueLabelDisplay="auto"
                />
              </Box>

              <Box>
                <Typography variant="caption" gutterBottom display="block">
                  Years to Project: {inputs.yearsToProject}
                </Typography>
                <Slider
                  value={inputs.yearsToProject}
                  onChange={(_, value) => setInputs({ ...inputs, yearsToProject: value as number })}
                  min={1}
                  max={30}
                  step={1}
                  marks={[
                    { value: 1, label: '1y' },
                    { value: 10, label: '10y' },
                    { value: 30, label: '30y' },
                  ]}
                  valueLabelDisplay="auto"
                />
              </Box>

              <Button
                variant="contained"
                size="large"
                startIcon={<CalculateIcon />}
                onClick={calculateTax}
                disabled={loading}
                fullWidth
              >
                {loading ? 'Calculating...' : 'Calculate Tax'}
              </Button>
            </Box>
          </Paper>
        </Grid>

        {/* Results Panel */}
        <Grid item xs={12} md={6}>
          <Paper elevation={2} sx={{ p: 3, height: '100%' }}>
            <Typography variant="subtitle1" gutterBottom fontWeight="bold">
              Tax Calculation Results
            </Typography>

            {!results ? (
              <Alert severity="info" sx={{ mt: 2 }}>
                Enter your estate details and click "Calculate Tax" to see results.
              </Alert>
            ) : (
              <Box sx={{ mt: 2 }}>
                {/* Current Tax */}
                <Card variant="outlined" sx={{ mb: 2 }}>
                  <CardContent>
                    <Typography variant="subtitle2" color="text.secondary" gutterBottom>
                      Current Estate Tax (Today)
                    </Typography>
                    <Typography variant="h3" color="error" gutterBottom>
                      {formatCurrency(results.combined)}
                    </Typography>
                    <Chip
                      label={`Effective Rate: ${results.effectiveRate.toFixed(1)}%`}
                      color="error"
                      size="small"
                    />
                  </CardContent>
                </Card>

                {/* Tax Breakdown Table */}
                <Table size="small">
                  <TableBody>
                    <TableRow>
                      <TableCell><strong>Gross Estate</strong></TableCell>
                      <TableCell align="right">{formatCurrency(inputs.grossEstate)}</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>Less: Charitable Deductions</TableCell>
                      <TableCell align="right">-{formatCurrency(inputs.charitableDeductions)}</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell>Less: Federal Exemption</TableCell>
                      <TableCell align="right">-{formatCurrency(13990000)}</TableCell>
                    </TableRow>
                    <TableRow>
                      <TableCell><strong>Taxable Estate (Federal)</strong></TableCell>
                      <TableCell align="right"><strong>{formatCurrency(results.federal.taxable_amount)}</strong></TableCell>
                    </TableRow>
                    <TableRow sx={{ bgcolor: 'error.50' }}>
                      <TableCell><strong>Federal Estate Tax (40%)</strong></TableCell>
                      <TableCell align="right"><strong>{formatCurrency(results.federal.tax_owed)}</strong></TableCell>
                    </TableRow>
                    <TableRow sx={{ bgcolor: 'warning.50' }}>
                      <TableCell><strong>State Estate Tax ({inputs.state})</strong></TableCell>
                      <TableCell align="right"><strong>{formatCurrency(results.state.tax_owed)}</strong></TableCell>
                    </TableRow>
                    <TableRow sx={{ bgcolor: 'error.100' }}>
                      <TableCell><strong>Total Estate Tax</strong></TableCell>
                      <TableCell align="right"><strong>{formatCurrency(results.combined)}</strong></TableCell>
                    </TableRow>
                  </TableBody>
                </Table>

                <Divider sx={{ my: 2 }} />

                {/* Future Projection */}
                <Card variant="outlined" sx={{ bgcolor: 'success.50' }}>
                  <CardContent>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
                      <GrowthIcon color="success" />
                      <Typography variant="subtitle2" color="text.secondary">
                        Projected in {inputs.yearsToProject} Years
                      </Typography>
                    </Box>
                    <Typography variant="body2" gutterBottom>
                      <strong>Estate Value:</strong> {formatCurrency(calculateFutureValue())}
                    </Typography>
                    <Typography variant="body2" gutterBottom>
                      <strong>Projected Tax:</strong> {formatCurrency(calculateProjectedTax())}
                    </Typography>
                    <Alert severity="warning" sx={{ mt: 2 }}>
                      <Typography variant="caption">
                        Without planning, estate could grow to {formatCurrency(calculateFutureValue())}, 
                        resulting in {formatCurrency(calculateProjectedTax())} in estate taxes.
                      </Typography>
                    </Alert>
                  </CardContent>
                </Card>
              </Box>
            )}
          </Paper>
        </Grid>

        {/* Tax Savings Scenarios */}
        {results && (
          <Grid item xs={12}>
            <Paper elevation={2} sx={{ p: 3 }}>
              <Typography variant="subtitle1" gutterBottom fontWeight="bold">
                Potential Tax Savings with Planning
              </Typography>

              <Grid container spacing={2} sx={{ mt: 1 }}>
                <Grid item xs={12} md={4}>
                  <Card variant="outlined">
                    <CardContent>
                      <Typography variant="subtitle2" gutterBottom>
                        Annual Gifting Strategy
                      </Typography>
                      <Typography variant="h5" color="success.main">
                        Save {formatCurrency(results.combined * 0.30)}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        ~30% tax reduction through systematic gifting
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} md={4}>
                  <Card variant="outlined">
                    <CardContent>
                      <Typography variant="subtitle2" gutterBottom>
                        SLAT + Gifting Combination
                      </Typography>
                      <Typography variant="h5" color="success.main">
                        Save {formatCurrency(results.combined * 0.60)}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        ~60% tax reduction with comprehensive planning
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>

                <Grid item xs={12} md={4}>
                  <Card variant="outlined">
                    <CardContent>
                      <Typography variant="subtitle2" gutterBottom>
                        Dynasty Trust (Multi-Gen)
                      </Typography>
                      <Typography variant="h5" color="success.main">
                        Save {formatCurrency(results.combined * 1.5)}
                      </Typography>
                      <Typography variant="caption" color="text.secondary">
                        Estate tax eliminated across 3 generations
                      </Typography>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>

              <Alert severity="info" sx={{ mt: 2 }}>
                Click "Generate Estate Plan" from the main dashboard to get personalized recommendations.
              </Alert>
            </Paper>
          </Grid>
        )}
      </Grid>
    </Box>
  );
};
