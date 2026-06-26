import React, { useState } from 'react';
import {
  Box,
  Typography,
  Card,
  CardContent,
  Grid,
  TextField,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  CircularProgress,
  Chip,
} from '@mui/material';
import PlayArrowIcon from '@mui/icons-material/PlayArrow';
import AutoGraphIcon from '@mui/icons-material/AutoGraph';

interface SimulationResult {
  portfolioId: string;
  portfolioName: string;
  originalNav: number;
  newNav: number;
  deltaPercent: number;
  impactedPositions: number;
}

export interface SecurityImpactSimulationProps {
  securityId: string;
  currentPrice: number;
  currentRating?: string;
}

export const SecurityImpactSimulation: React.FC<SecurityImpactSimulationProps> = ({
  // securityId,
  currentPrice,
  currentRating = 'A',
}) => {
  const [simulatedPrice, setSimulatedPrice] = useState<number>(currentPrice);
  const [simulatedRating, setSimulatedRating] = useState<string>(currentRating);
  const [isSimulating, setIsSimulating] = useState(false);
  const [results, setResults] = useState<SimulationResult[] | null>(null);

  const handleSimulate = async () => {
    setIsSimulating(true);
    // Simulate API call to the lineage execution engine
    setTimeout(() => {
      // Mock results based on the delta

      
      setResults([
        {
          portfolioId: 'port-1',
          portfolioName: 'Global Equity Growth',
          originalNav: 15420000,
          newNav: 15420000 + (1000 * (simulatedPrice - currentPrice)), // Assuming 1000 shares
          deltaPercent: ((simulatedPrice - currentPrice) / currentPrice) * 1.5, // 1.5% weight
          impactedPositions: 1,
        },
        {
          portfolioId: 'port-2',
          portfolioName: 'US Large Cap Core',
          originalNav: 45800000,
          newNav: 45800000 + (5000 * (simulatedPrice - currentPrice)), // Assuming 5000 shares
          deltaPercent: ((simulatedPrice - currentPrice) / currentPrice) * 3.2, // 3.2% weight
          impactedPositions: 1,
        },
        {
          portfolioId: 'port-3',
          portfolioName: 'Balanced Strategy',
          originalNav: 8900000,
          newNav: 8900000 + (250 * (simulatedPrice - currentPrice)), // Assuming 250 shares
          deltaPercent: ((simulatedPrice - currentPrice) / currentPrice) * 0.8, // 0.8% weight
          impactedPositions: 1,
        }
      ]);
      setIsSimulating(false);
    }, 1500);
  };

  const formatCurrency = (val: number) => {
    return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(val);
  };

  const formatPercent = (val: number) => {
    const sign = val > 0 ? '+' : '';
    return `${sign}${val.toFixed(2)}%`;
  };

  return (
    <Card elevation={2}>
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 3, gap: 1 }}>
          <AutoGraphIcon color="primary" />
          <Typography variant="h6">
            Downstream Impact Simulation
          </Typography>
        </Box>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          Simulate how changes to this security's core attributes will propagate through the semantic graph and impact portfolio NAV calculations.
        </Typography>

        <Grid container spacing={3} sx={{ mb: 4 }} alignItems="flex-end">
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              label="Simulated Price ($)"
              type="number"
              value={simulatedPrice}
              onChange={(e) => setSimulatedPrice(Number(e.target.value))}
              InputProps={{ inputProps: { min: 0, step: 0.01 } }}
            />
          </Grid>
          <Grid item xs={12} md={4}>
            <TextField
              fullWidth
              label="Simulated Credit Rating"
              value={simulatedRating}
              onChange={(e) => setSimulatedRating(e.target.value)}
            />
          </Grid>
          <Grid item xs={12} md={4}>
            <Button
              variant="contained"
              color="primary"
              fullWidth
              size="large"
              startIcon={isSimulating ? <CircularProgress size={20} color="inherit" /> : <PlayArrowIcon />}
              onClick={handleSimulate}
              disabled={isSimulating || simulatedPrice === currentPrice}
              sx={{ height: 56 }}
            >
              Initialize Simulation
            </Button>
          </Grid>
        </Grid>

        {results && (
          <Box>
            <Typography variant="subtitle1" fontWeight="bold" gutterBottom>
              Simulated Portfolio Impact
            </Typography>
            <TableContainer component={Paper} elevation={0} variant="outlined">
              <Table size="small">
                <TableHead>
                  <TableRow sx={{ backgroundColor: 'action.hover' }}>
                    <TableCell>Portfolio</TableCell>
                    <TableCell align="right">Original NAV</TableCell>
                    <TableCell align="right">Simulated NAV</TableCell>
                    <TableCell align="right">NAV Delta</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {results.map((r) => (
                    <TableRow key={r.portfolioId}>
                      <TableCell sx={{ fontWeight: 500 }}>{r.portfolioName}</TableCell>
                      <TableCell align="right">{formatCurrency(r.originalNav)}</TableCell>
                      <TableCell align="right">
                        <Typography sx={{ fontWeight: 'bold' }}>
                          {formatCurrency(r.newNav)}
                        </Typography>
                      </TableCell>
                      <TableCell align="right">
                        <Chip 
                          label={formatPercent(r.deltaPercent)}
                          color={r.deltaPercent > 0 ? 'success' : r.deltaPercent < 0 ? 'error' : 'default'}
                          size="small"
                          sx={{ fontWeight: 'bold' }}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Box>
        )}
      </CardContent>
    </Card>
  );
};
