import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Typography,
  Paper,
  Alert,
  CircularProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip
} from '@mui/material';
import { ToggleButton, ToggleButtonGroup } from '@mui/material';
import {
  BarChart,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Bar,
  ReferenceLine
} from 'recharts';
import { Legend } from 'recharts';

interface BenchmarkData {
  fundId: string;
  fundName: string;
  vintage: number;
  netPme: number;
  grossPme: number;
  benchmarkIndex: string;
  benchmarkReturn: number;
  excessReturn: number;
  alpha: number;
}

interface BenchmarkComparisonProps {
  selectedFunds?: string[];
}

export const BenchmarkComparison: React.FC<BenchmarkComparisonProps> = ({ selectedFunds = [] }) => {
  const [benchmarkData, setBenchmarkData] = useState<BenchmarkData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pmeType, setPmeType] = useState<'net' | 'gross'>('net');
  const [benchmarkIndex, setBenchmarkIndex] = useState<'sp500' | 'msciaworld' | 'custom'>('msciaworld');
  const [viewMode, setViewMode] = useState<'chart' | 'table'>('chart');

  const loadBenchmarkData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // In a real implementation, this would call your API
      const response = await fetch(
        `/api/funds/benchmark?funds=${selectedFunds.join(',')}&pmeType=${pmeType}&benchmark=${benchmarkIndex}`
      );
      if (!response.ok) {
        throw new Error('Failed to load benchmark data');
      }
      const data = await response.json();
      setBenchmarkData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load benchmark data');
    } finally {
      setLoading(false);
    }
  }, [selectedFunds, pmeType, benchmarkIndex]);

  useEffect(() => {
    if (selectedFunds.length > 0) {
      loadBenchmarkData();
    }
  }, [selectedFunds, pmeType, benchmarkIndex, loadBenchmarkData]);

  const handlePmeTypeChange = (
    _event: React.MouseEvent<HTMLElement>,
    newType: 'net' | 'gross' | null,
  ) => {
    if (newType !== null) {
      setPmeType(newType);
    }
  };

  const handleBenchmarkChange = (
    _event: React.MouseEvent<HTMLElement>,
    newBenchmark: 'sp500' | 'msciaworld' | 'custom' | null,
  ) => {
    if (newBenchmark !== null) {
      setBenchmarkIndex(newBenchmark);
    }
  };

  const handleViewModeChange = (
    _event: React.MouseEvent<HTMLElement>,
    newMode: 'chart' | 'table' | null,
  ) => {
    if (newMode !== null) {
      setViewMode(newMode);
    }
  };

  const formatTooltip = (value: any, name: string) => {
    if (name === 'benchmarkReturn') return [`${Number(value).toFixed(2)}%`, 'Benchmark Return'];
    if (name === 'excessReturn') return [`${Number(value).toFixed(2)}%`, 'Excess Return'];
    return [`${Number(value).toFixed(2)}x`, name];
  };

  const formatYAxisTick = (value: number) => {
    return `${value.toFixed(2)}x`;
  };

  const getBenchmarkLabel = (index: string) => {
    switch (index) {
      case 'sp500': return 'S&P 500';
      case 'msciaworld': return 'MSCI ACWI';
      case 'custom': return 'Custom Benchmark';
      default: return index;
    }
  };

  const getPerformanceColor = (pme: number) => {
    if (pme > 1.5) return '#4caf50';
    if (pme > 1.0) return '#ff9800';
    return '#f44336';
  };

  if (selectedFunds.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          Select funds to view benchmark comparison
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

  const chartData = benchmarkData.map(fund => ({
    name: fund.fundName,
    pme: pmeType === 'net' ? fund.netPme : fund.grossPme,
    benchmark: fund.benchmarkReturn / 100, // Convert to multiple
    excess: fund.excessReturn / 100,
    vintage: fund.vintage
  }));

  return (
    <Paper sx={{ p: 3 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h6">
          Benchmark Comparison Analysis
        </Typography>

        <Box display="flex" gap={2}>
          <ToggleButtonGroup
            value={pmeType}
            exclusive
            onChange={handlePmeTypeChange}
            size="small"
          >
            <ToggleButton value="net">Net PME</ToggleButton>
            <ToggleButton value="gross">Gross PME</ToggleButton>
          </ToggleButtonGroup>

          <ToggleButtonGroup
            value={benchmarkIndex}
            exclusive
            onChange={handleBenchmarkChange}
            size="small"
          >
            <ToggleButton value="msciaworld">MSCI ACWI</ToggleButton>
            <ToggleButton value="sp500">S&P 500</ToggleButton>
            <ToggleButton value="custom">Custom</ToggleButton>
          </ToggleButtonGroup>

          <ToggleButtonGroup
            value={viewMode}
            exclusive
            onChange={handleViewModeChange}
            size="small"
          >
            <ToggleButton value="chart">Chart</ToggleButton>
            <ToggleButton value="table">Table</ToggleButton>
          </ToggleButtonGroup>
        </Box>
      </Box>

      <Typography variant="body2" color="text.secondary" mb={3}>
        Comparing fund performance against {getBenchmarkLabel(benchmarkIndex)} benchmark.
        PME shows fund returns as a multiple of the benchmark's return over the same period.
      </Typography>

      {viewMode === 'chart' ? (
        <Box sx={{ width: '100%', height: 400 }}>
          <ResponsiveContainer>
            <BarChart data={chartData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="name"
                tick={{ fontSize: 12 }}
                angle={-45}
                textAnchor="end"
                height={80}
              />
              <YAxis tickFormatter={formatYAxisTick} />
              <Tooltip formatter={formatTooltip} />
              <Legend />

              {/* Benchmark reference line */}
              <ReferenceLine y={1.0} stroke="#666" strokeDasharray="5 5" label="Benchmark" />

              <Bar dataKey="pme" fill="#8884d8" name="Fund PME" />
              <Bar dataKey="benchmark" fill="#82ca9d" name="Benchmark Multiple" />
            </BarChart>
          </ResponsiveContainer>
        </Box>
      ) : (
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Fund Name</TableCell>
                <TableCell>Vintage</TableCell>
                <TableCell align="right">PME</TableCell>
                <TableCell align="right">Benchmark Return</TableCell>
                <TableCell align="right">Excess Return</TableCell>
                <TableCell>Performance</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {benchmarkData.map((fund) => {
                const pme = pmeType === 'net' ? fund.netPme : fund.grossPme;
                return (
                  <TableRow key={fund.fundId}>
                    <TableCell>{fund.fundName}</TableCell>
                    <TableCell>{fund.vintage}</TableCell>
                    <TableCell align="right">
                      <Typography sx={{ color: getPerformanceColor(pme) }}>
                        {pme.toFixed(2)}x
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      {fund.benchmarkReturn.toFixed(1)}%
                    </TableCell>
                    <TableCell align="right">
                      <Typography sx={{ color: fund.excessReturn > 0 ? 'success.main' : 'error.main' }}>
                        {fund.excessReturn > 0 ? '+' : ''}{fund.excessReturn.toFixed(1)}%
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={pme > 1.0 ? 'Outperforming' : 'Underperforming'}
                        size="small"
                        color={pme > 1.0 ? 'success' : 'error'}
                        variant="outlined"
                      />
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      <Box mt={3} p={2} bgcolor="grey.50" borderRadius={1}>
        <Typography variant="body2" color="text.secondary">
          <strong>Understanding PME:</strong><br />
          Public Market Equivalent (PME) measures fund performance relative to a public market benchmark.
          A PME &gt; 1.0x means the fund outperformed the benchmark. Net PME accounts for fees, while Gross PME shows performance before fees.
        </Typography>
      </Box>
    </Paper>
  );
};
