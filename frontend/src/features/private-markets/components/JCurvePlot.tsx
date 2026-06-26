import React, { useState, useEffect, useCallback } from 'react';
import { Bundle } from '../types';
import {
  Box,
  Typography,
  Paper,
  ToggleButton,
  ToggleButtonGroup,
  Alert,
  CircularProgress,
  Grid,
  Card,
  CardContent
} from '@mui/material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  AreaChart,
  Area,
  ReferenceLine
} from 'recharts';

interface CashFlowData {
  date: string;
  cumulativeNet: number;
  cumulativeGross: number;
  netCashFlow: number;
  grossCashFlow: number;
  [fundId: string]: string | number;
}

interface JCurvePlotProps {
  selectedFunds?: string[];
  excelResults?: Record<string, Record<string, any>> | null;
  bundle?: Bundle | null;
}

export const JCurvePlot: React.FC<JCurvePlotProps> = ({ selectedFunds = [] }) => {
  const [cashFlowData, setCashFlowData] = useState<CashFlowData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [viewMode, setViewMode] = useState<'net' | 'gross'>('net');
  const [chartType, setChartType] = useState<'line' | 'area'>('area');

  const loadCashFlowData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      // In a real implementation, this would call your API
      const response = await fetch(`/api/funds/cashflow?funds=${selectedFunds.join(',')}`);
      if (!response.ok) {
        throw new Error('Failed to load cash flow data');
      }
      const data = await response.json();
      setCashFlowData(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load cash flow data');
    } finally {
      setLoading(false);
    }
  }, [selectedFunds]);

  useEffect(() => {
    if (selectedFunds.length > 0) {
      loadCashFlowData();
    }
  }, [selectedFunds, loadCashFlowData]);
  const handleViewModeChange = (
    _event: React.MouseEvent<HTMLElement>,
    newMode: 'net' | 'gross' | null,
  ) => {
    if (newMode !== null) {
      setViewMode(newMode);
    }
  };

  const handleChartTypeChange = (
    _event: React.MouseEvent<HTMLElement>,
    newType: 'line' | 'area' | null,
  ) => {
    if (newType !== null) {
      setChartType(newType);
    }
  };

  const formatTooltip = (value: any, name: string) => {
    if (name === 'date') return value;
    return [`$${Number(value).toLocaleString()}`, name];
  };

  const formatYAxisTick = (value: number) => {
    return `$${(value / 1000000).toFixed(1)}M`;
  };

  // Find the inflection point (where cumulative cash flow becomes positive)
  const findInflectionPoint = (data: CashFlowData[]) => {
    for (let i = 0; i < data.length - 1; i++) {
      const current = viewMode === 'net' ? data[i].cumulativeNet : data[i].cumulativeGross;
      const next = viewMode === 'net' ? data[i + 1].cumulativeNet : data[i + 1].cumulativeGross;
      if (current < 0 && next >= 0) {
        return data[i + 1].date;
      }
    }
    return null;
  };

  const inflectionPoint = findInflectionPoint(cashFlowData);

  if (selectedFunds.length === 0) {
    return (
      <Paper sx={{ p: 3, textAlign: 'center' }}>
        <Typography variant="h6" color="text.secondary">
          Select funds to view J-curve analysis
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

  const dataKey = viewMode === 'net' ? 'cumulativeNet' : 'cumulativeGross';
  const title = viewMode === 'net' ? 'Net J-Curve' : 'Gross J-Curve';

  return (
    <Paper sx={{ p: 3 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h6">
          J-Curve Analysis
        </Typography>

        <Box display="flex" gap={2}>
          <ToggleButtonGroup
            value={viewMode}
            exclusive
            onChange={handleViewModeChange}
            size="small"
          >
            <ToggleButton value="net">Net Cash Flow</ToggleButton>
            <ToggleButton value="gross">Gross Cash Flow</ToggleButton>
          </ToggleButtonGroup>

          <ToggleButtonGroup
            value={chartType}
            exclusive
            onChange={handleChartTypeChange}
            size="small"
          >
            <ToggleButton value="area">Area Chart</ToggleButton>
            <ToggleButton value="line">Line Chart</ToggleButton>
          </ToggleButtonGroup>
        </Box>
      </Box>

      <Grid container spacing={2} sx={{ mb: 2 }}>
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" color="primary">
                J-Curve Pattern
              </Typography>
              <Typography variant="body2">
                The J-curve shows the typical cash flow pattern of private equity funds:
                negative cash flows during investment period, turning positive during harvest.
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" color="secondary">
                Inflection Point
              </Typography>
              <Typography variant="body2">
                {inflectionPoint
                  ? `Cash flow positive starting ${inflectionPoint}`
                  : 'Inflection point not yet reached'
                }
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} md={4}>
          <Card>
            <CardContent>
              <Typography variant="h6" color="success.main">
                Current Status
              </Typography>
              <Typography variant="body2">
                {cashFlowData.length > 0 && cashFlowData[cashFlowData.length - 1][dataKey] >= 0
                  ? 'Fund has reached cash flow positive'
                  : 'Fund still in investment phase'
                }
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      <Box sx={{ width: '100%', height: 400 }}>
        <ResponsiveContainer>
          {chartType === 'area' ? (
            <AreaChart data={cashFlowData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="date"
                tick={{ fontSize: 12 }}
                angle={-45}
                textAnchor="end"
                height={80}
              />
              <YAxis tickFormatter={formatYAxisTick} />
              <Tooltip formatter={formatTooltip} />

              {/* Zero line */}
              <ReferenceLine y={0} stroke="#000" strokeWidth={2} />

              {/* Inflection point */}
              {inflectionPoint && (
                <ReferenceLine
                  x={inflectionPoint}
                  stroke="#ff0000"
                  strokeDasharray="5 5"
                  label="Inflection Point"
                />
              )}

              <Area
                type="monotone"
                dataKey={dataKey}
                stroke="#8884d8"
                fill="#8884d8"
                fillOpacity={0.6}
                name={title}
              />
            </AreaChart>
          ) : (
            <LineChart data={cashFlowData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="date"
                tick={{ fontSize: 12 }}
                angle={-45}
                textAnchor="end"
                height={80}
              />
              <YAxis tickFormatter={formatYAxisTick} />
              <Tooltip formatter={formatTooltip} />

              {/* Zero line */}
              <ReferenceLine y={0} stroke="#000" strokeWidth={2} />

              {/* Inflection point */}
              {inflectionPoint && (
                <ReferenceLine
                  x={inflectionPoint}
                  stroke="#ff0000"
                  strokeDasharray="5 5"
                  label="Inflection Point"
                />
              )}

              <Line
                type="monotone"
                dataKey={dataKey}
                stroke="#8884d8"
                strokeWidth={3}
                dot={{ r: 4 }}
                activeDot={{ r: 6 }}
                name={title}
              />
            </LineChart>
          )}
        </ResponsiveContainer>
      </Box>

      <Box mt={2}>
        <Typography variant="body2" color="text.secondary">
          J-curve shows cumulative {viewMode} cash flows over time. The "J" shape represents
          the typical pattern: negative cash flows during investment period, becoming positive
          during the harvest phase. The inflection point marks when the fund becomes cash flow positive.
        </Typography>
      </Box>
    </Paper>
  );
};
