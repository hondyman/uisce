import React, { useState, useEffect, useCallback } from 'react';
// no-op import removed; types available in other modules
import {
  Box,
  Typography,
  Paper,
  ToggleButton,
  ToggleButtonGroup,
  Alert,
  CircularProgress
} from '@mui/material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
  ReferenceLine
} from 'recharts';

interface IRRDataPoint {
  date: string;
  [fundId: string]: string | number;
}

import type { Bundle } from '../types';

interface IRRCurveChartProps {
  selectedFunds?: string[];
  excelResults?: Record<string, Record<string, any>> | null;
  bundle?: Bundle | null;
}

export const IRRCurveChart: React.FC<IRRCurveChartProps> = ({ selectedFunds = [] }) => {
  const [irrData, setIrrData] = useState<IRRDataPoint[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'net' | 'gross'>('net');
  const [benchmarkMode, setBenchmarkMode] = useState<'none' | 'quartile' | 'vintage'>('quartile');

  const loadIRRData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // In a real implementation, this would call your API
      const response = await fetch(`/api/funds/irr-curve?funds=${selectedFunds.join(',')}&type=${viewMode}`);
      if (!response.ok) {
        throw new Error('Failed to load IRR data');
      }
      const data = await response.json();
      setIrrData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load IRR data');
    } finally {
      setLoading(false);
    }
  }, [selectedFunds, viewMode]);

  useEffect(() => {
    if (selectedFunds.length > 0) {
      loadIRRData();
    }
  }, [selectedFunds, viewMode, loadIRRData]);

  const handleViewModeChange = (
    _event: React.MouseEvent<HTMLElement>,
    newMode: 'net' | 'gross' | null,
  ) => {
    if (newMode !== null) {
      setViewMode(newMode);
    }
  };

  const handleBenchmarkModeChange = (
    _event: React.MouseEvent<HTMLElement>,
    newMode: 'none' | 'quartile' | 'vintage' | null,
  ) => {
    if (newMode !== null) {
      setBenchmarkMode(newMode);
    }
  };

  const formatTooltip = (value: any, name: string) => {
    if (name === 'date') return value;
    return [`${Number(value).toFixed(2)}%`, name];
  };

  const formatYAxisTick = (value: number) => {
    return `${value}%`;
  };

  if (selectedFunds.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          Select funds to view IRR curves
        </Typography>
      </Paper>
    );
  }

  if (loading) {
    return (
      <Paper sx={{ p: 3, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress />
      </Paper>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        {error}
      </Alert>
    );
  }

  // Generate colors for different funds
  const colors = ['#8884d8', '#82ca9d', '#ffc658', '#ff7300', '#00ff00', '#ff00ff', '#00ffff', '#ff0000'];

  return (
    <Paper sx={{ p: 3 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h6">
          IRR Curve Analysis
        </Typography>

        <Box display="flex" gap={2}>
          <ToggleButtonGroup
            value={viewMode}
            exclusive
            onChange={handleViewModeChange}
            size="small"
          >
            <ToggleButton value="net">Net IRR</ToggleButton>
            <ToggleButton value="gross">Gross IRR</ToggleButton>
          </ToggleButtonGroup>

          <ToggleButtonGroup
            value={benchmarkMode}
            exclusive
            onChange={handleBenchmarkModeChange}
            size="small"
          >
            <ToggleButton value="none">No Benchmark</ToggleButton>
            <ToggleButton value="quartile">Quartiles</ToggleButton>
            <ToggleButton value="vintage">Vintage</ToggleButton>
          </ToggleButtonGroup>
        </Box>
      </Box>

      <Box sx={{ width: '100%', height: 400 }}>
        <ResponsiveContainer>
          <LineChart data={irrData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="date"
              tick={{ fontSize: 12 }}
              angle={-45}
              textAnchor="end"
              height={80}
            />
            <YAxis
              tickFormatter={formatYAxisTick}
              domain={['dataMin - 5', 'dataMax + 5']}
            />
            <Tooltip formatter={formatTooltip} />

            {/* Benchmark reference lines */}
            {benchmarkMode === 'quartile' && (
              <>
                <ReferenceLine y={15} stroke="#ff0000" strokeDasharray="5 5" label="75th Percentile" />
                <ReferenceLine y={10} stroke="#ffa500" strokeDasharray="5 5" label="Median" />
                <ReferenceLine y={5} stroke="#00ff00" strokeDasharray="5 5" label="25th Percentile" />
              </>
            )}

            {benchmarkMode === 'vintage' && (
              <ReferenceLine y={12} stroke="#0000ff" strokeDasharray="5 5" label="Vintage Average" />
            )}

            <Legend />

            {/* Fund IRR lines */}
            {selectedFunds.map((fundId, index) => (
              <Line
                key={fundId}
                type="monotone"
                dataKey={fundId}
                stroke={colors[index % colors.length]}
                strokeWidth={2}
                dot={{ r: 4 }}
                activeDot={{ r: 6 }}
                name={`Fund ${fundId}`}
              />
            ))}
          </LineChart>
        </ResponsiveContainer>
      </Box>

      <Box mt={2}>
        <Typography variant="body2" color="text.secondary">
          {viewMode === 'net' ? 'Net IRR' : 'Gross IRR'} over time for selected funds.
          {benchmarkMode === 'quartile' && ' Quartile benchmarks shown for peer comparison.'}
          {benchmarkMode === 'vintage' && ' Vintage year average shown for comparison.'}
        </Typography>
      </Box>
    </Paper>
  );
};
