import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  CircularProgress,
  DataGrid,
  GridColDef,
  Paper,
  ToggleButton,
  ToggleButtonGroup,
  Tooltip,
  Typography,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip as ChartTooltip,
  Legend,
  ResponsiveContainer,
  Cell,
} from 'recharts';
import { useMemo, useState } from 'react';
import { useScenarioComparison } from '../../hooks/useScenarioComparison';
import type { SimulationResult, ScenarioComparison, StressScenario } from '../../types/scenarios';

export interface MultiScenarioComparisonProps {
  scenarios: StressScenario[];
  scenarioResults: Map<string, SimulationResult[]>;
  onSelectScenario?: (scenarioId: string) => void;
  isLoading?: boolean;
}

type MetricType = 'pnl' | 'variance' | 'confidence';

/**
 * Multi-scenario comparison dashboard with charts and data grid
 * Displays comparative metrics, variance analysis, and portfolio-level results
 * 
 * Features:
 * - Clustered bar charts for scenario comparison
 * - Data grid with sortable/filterable columns
 * - Metric toggles (PnL, Volatility, Confidence)
 * - Aggregated statistics sidebar
 * - 100% Material UI design
 * - Dark mode support
 * - Responsive layout
 */
export function MultiScenarioComparison({
  scenarios,
  scenarioResults,
  onSelectScenario,
  isLoading = false,
}: MultiScenarioComparisonProps) {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const isTablet = useMediaQuery(theme.breakpoints.down('lg'));

  const [activeMetric, setActiveMetric] = useState<MetricType>('pnl');
  const [sortBy, setSortBy] = useState<'name' | 'avgPnL' | 'variance'>('name');
  const [filterMinPnL, setFilterMinPnL] = useState<number | null>(null);
  const [filterMaxPnL, setFilterMaxPnL] = useState<number | null>(null);

  const { comparison, isCalculating } = useScenarioComparison();

  // Prepare chart data - compare PnL across scenarios
  const chartData = useMemo(() => {
    if (!comparison || comparison.scenarios.length === 0) return [];

    return comparison.scenarios.map(s => ({
      name: s.scenarioName,
      scenarioId: s.scenarioId,
      pnl: parseFloat(s.avgPnL.toFixed(2)),
      variance: parseFloat(s.variance.toFixed(2)),
      confidence: parseFloat(s.avgConfidence.toFixed(1)),
      portfolioCount: s.portfolioCount,
    }));
  }, [comparison]);

  // Prepare data grid rows - all portfolios with per-scenario PnL
  const gridRows = useMemo(() => {
    if (scenarios.length === 0) return [];

    // Get all unique portfolio IDs
    const portfolioIds = new Set<string>();
    scenarios.forEach(scenario => {
      const results = scenarioResults.get(scenario.id) || [];
      results.forEach(r => portfolioIds.add(r.portfolioId));
    });

    // Build rows with PnL for each scenario
    return Array.from(portfolioIds).map((portfolioId, index) => {
      const row: any = {
        id: portfolioId,
        portfolioId,
        index: index + 1,
      };

      scenarios.forEach(scenario => {
        const results = scenarioResults.get(scenario.id) || [];
        const result = results.find(r => r.portfolioId === portfolioId);
        if (result) {
          row[`pnl_${scenario.id}`] = result.pnl;
          row[`confidence_${scenario.id}`] = result.confidence;
        }
      });

      return row;
    });
  }, [scenarios, scenarioResults]);

  // Filter rows based on PnL range
  const filteredRows = useMemo(() => {
    return gridRows.filter(row => {
      if (filterMinPnL !== null && row[`pnl_${activeMetric}`] < filterMinPnL) return false;
      if (filterMaxPnL !== null && row[`pnl_${activeMetric}`] > filterMaxPnL) return false;
      return true;
    });
  }, [gridRows, filterMinPnL, filterMaxPnL, activeMetric]);

  // Dynamic data grid columns
  const gridColumns = useMemo((): GridColDef[] => {
    const baseCols: GridColDef[] = [
      {
        field: 'portfolioId',
        headerName: 'Portfolio',
        flex: 1,
        minWidth: 150,
      },
    ];

    // Add columns for each scenario
    scenarios.forEach(scenario => {
      if (activeMetric === 'pnl') {
        baseCols.push({
          field: `pnl_${scenario.id}`,
          headerName: `${scenario.name} PnL (M)`,
          flex: 1,
          minWidth: 140,
          renderCell: params => {
            const value = params.value as number | undefined;
            if (value === undefined) return '-';
            const isNegative = value < 0;
            return (
              <Tooltip title={`Confidence: ${params.row[`confidence_${scenario.id}`]}%`}>
                <span
                  style={{
                    color: isNegative ? theme.palette.error.main : theme.palette.success.main,
                    fontWeight: 600,
                  }}
                >
                  {value > 0 ? '+' : ''}{value.toFixed(2)}
                </span>
              </Tooltip>
            );
          },
        });
      } else if (activeMetric === 'confidence') {
        baseCols.push({
          field: `confidence_${scenario.id}`,
          headerName: `${scenario.name} Confidence (%)`,
          flex: 1,
          minWidth: 140,
          renderCell: params => {
            const value = params.value as number | undefined;
            if (value === undefined) return '-';
            return (
              <Box display="flex" alignItems="center" gap={1}>
                <Box
                  sx={{
                    width: 60,
                    height: 6,
                    backgroundColor: theme.palette.action.disabledBackground,
                    borderRadius: 3,
                    overflow: 'hidden',
                  }}
                >
                  <Box
                    sx={{
                      width: `${value}%`,
                      height: '100%',
                      backgroundColor:
                        value >= 90
                          ? theme.palette.success.main
                          : value >= 70
                          ? theme.palette.info.main
                          : theme.palette.warning.main,
                      transition: 'width 0.3s ease',
                    }}
                  />
                </Box>
                <Typography variant="body2">{value.toFixed(1)}%</Typography>
              </Box>
            );
          },
        });
      }
    });

    return baseCols;
  }, [scenarios, activeMetric, theme]);

  if (isLoading || isCalculating) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={400}>
        <CircularProgress />
      </Box>
    );
  }

  if (scenarios.length === 0) {
    return (
      <Alert severity="info">
        No scenarios to compare. Run at least one simulation to see comparisons.
      </Alert>
    );
  }

  // Chart colors for scenarios
  const chartColors = [
    theme.palette.primary.main,
    theme.palette.secondary.main,
    theme.palette.info.main,
    theme.palette.success.main,
    theme.palette.warning.main,
  ];

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      {/* Top Controls */}
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          flexWrap: 'wrap',
          gap: 2,
        }}
      >
        <Box>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Compare Metrics
          </Typography>
          <ToggleButtonGroup
            value={activeMetric}
            exclusive
            onChange={(e, newMetric) => {
              if (newMetric !== null) setActiveMetric(newMetric);
            }}
            size="small"
          >
            <ToggleButton value="pnl">PnL</ToggleButton>
            <ToggleButton value="variance">Variance</ToggleButton>
            <ToggleButton value="confidence">Confidence</ToggleButton>
          </ToggleButtonGroup>
        </Box>

        <Box>
          <Typography variant="body2" color="textSecondary" sx={{ mb: 1 }}>
            {filteredRows.length} of {gridRows.length} portfolios
          </Typography>
          <Button size="small" onClick={() => setFilterMinPnL(null)}>
            Clear Filters
          </Button>
        </Box>
      </Box>

      {/* Main Layout: Chart + Sidebar */}
      <Box sx={{ display: 'flex', gap: 2, flexDirection: isMobile ? 'column' : 'row' }}>
        {/* Chart Section */}
        <Card sx={{ flex: 1, minHeight: 300 }}>
          <CardContent>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Scenario {activeMetric === 'pnl' ? 'PnL' : activeMetric === 'variance' ? 'Volatility' : 'Confidence'} Comparison
            </Typography>

            <ResponsiveContainer width="100%" height={300}>
              <BarChart
                data={chartData}
                margin={{ top: 20, right: 30, left: 0, bottom: 40 }}
              >
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis
                  dataKey="name"
                  height={80}
                  interval={0}
                  tick={{
                    fontSize: 12,
                  }}
                  angle={-45}
                  textAnchor="end"
                  fill={theme.palette.text.secondary}
                />
                <YAxis
                  label={{ value: activeMetric.toUpperCase(), angle: -90, position: 'insideLeft' }}
                  fill={theme.palette.text.secondary}
                />
                <ChartTooltip
                  contentStyle={{
                    backgroundColor: theme.palette.background.paper,
                    border: `1px solid ${theme.palette.divider}`,
                    borderRadius: 8,
                  }}
                  formatter={(value: number) => [value.toFixed(2), '']}
                />
                <Legend />

                {activeMetric === 'pnl' && (
                  <Bar dataKey="pnl" fill={theme.palette.primary.main} name="Avg PnL (M)" />
                )}
                {activeMetric === 'variance' && (
                  <Bar dataKey="variance" fill={theme.palette.info.main} name="Volatility" />
                )}
                {activeMetric === 'confidence' && (
                  <Bar dataKey="confidence" fill={theme.palette.success.main} name="Confidence (%)" />
                )}
              </BarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        {/* Sidebar: Aggregates */}
        {!isMobile && (
          <Box sx={{ width: 280 }}>
            <Card sx={{ mb: 2 }}>
              <CardContent>
                <Typography variant="subtitle2" color="textSecondary" sx={{ mb: 2 }}>
                  AGGREGATED IMPACT
                </Typography>

                {comparison?.scenarios.map((scenario, idx) => (
                  <Box key={scenario.scenarioId} sx={{ mb: 2, pb: 2, borderBottom: '1px solid', borderColor: 'divider' }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                      <Typography variant="caption" sx={{ fontWeight: 600 }}>
                        {scenario.scenarioName}
                      </Typography>
                      <Chip
                        label={`${scenario.portfolioCount} portfolios`}
                        size="small"
                        sx={{
                          backgroundColor: chartColors[idx % chartColors.length],
                          color: 'white',
                          height: 20,
                          fontSize: '0.7rem',
                        }}
                      />
                    </Box>

                    <Box sx={{ fontSize: '0.875rem', mb: 0.5 }}>
                      <Typography variant="caption" display="block">
                        Avg PnL: <strong>{scenario.avgPnL.toFixed(2)}M</strong>
                      </Typography>
                      <Typography variant="caption" display="block">
                        Variance: <strong>{scenario.variance.toFixed(2)}</strong>
                      </Typography>
                      <Typography variant="caption" display="block">
                        Confidence: <strong>{scenario.avgConfidence.toFixed(1)}%</strong>
                      </Typography>
                    </Box>

                    <Box sx={{ display: 'flex', gap: 1, mt: 1 }}>
                      <Typography variant="caption" sx={{ color: scenario.minPnL < 0 ? 'error.main' : 'success.main' }}>
                        Min: {scenario.minPnL.toFixed(2)}M
                      </Typography>
                      <Typography variant="caption">|</Typography>
                      <Typography variant="caption" sx={{ color: scenario.maxPnL > 0 ? 'success.main' : 'error.main' }}>
                        Max: {scenario.maxPnL.toFixed(2)}M
                      </Typography>
                    </Box>
                  </Box>
                ))}
              </CardContent>
            </Card>
          </Box>
        )}
      </Box>

      {/* Data Grid */}
      <Card>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 2 }}>
            Portfolio Comparison
          </Typography>

          <Box sx={{ height: 400, width: '100%' }}>
            <DataGrid
              rows={filteredRows}
              columns={gridColumns}
              pageSizeOptions={[5, 10, 25]}
              initialState={{
                pagination: { paginationModel: { pageSize: 10 } },
              }}
              disableSelectionOnClick
              sx={{
                '& .MuiDataGrid-cell': {
                  borderColor: theme.palette.divider,
                },
                '& .MuiDataGrid-row:hover': {
                  backgroundColor: theme.palette.action.hover,
                },
              }}
            />
          </Box>
        </CardContent>
      </Card>

      {/* Stats Footer */}
      {comparison && (
        <Paper
          sx={{
            p: 2,
            backgroundColor: theme.palette.action.hover,
            display: 'flex',
            justifyContent: 'space-around',
            flexWrap: 'wrap',
            gap: 2,
          }}
        >
          <Box textAlign="center">
            <Typography variant="caption" color="textSecondary">
              Compared Scenarios
            </Typography>
            <Typography variant="h6">{comparison.metadata.comparedScenarioCount}</Typography>
          </Box>

          <Box textAlign="center">
            <Typography variant="caption" color="textSecondary">
              Total Portfolios
            </Typography>
            <Typography variant="h6">{comparison.metadata.totalPortfolios}</Typography>
          </Box>

          <Box textAlign="center">
            <Typography variant="caption" color="textSecondary">
              Best Case (Avg PnL)
            </Typography>
            <Typography
              variant="h6"
              sx={{
                color:
                  Math.max(...comparison.scenarios.map(s => s.avgPnL)) > 0
                    ? theme.palette.success.main
                    : theme.palette.error.main,
              }}
            >
              {Math.max(...comparison.scenarios.map(s => s.avgPnL)).toFixed(2)}M
            </Typography>
          </Box>

          <Box textAlign="center">
            <Typography variant="caption" color="textSecondary">
              Worst Case (Avg PnL)
            </Typography>
            <Typography
              variant="h6"
              sx={{
                color:
                  Math.min(...comparison.scenarios.map(s => s.avgPnL)) < 0
                    ? theme.palette.error.main
                    : theme.palette.success.main,
              }}
            >
              {Math.min(...comparison.scenarios.map(s => s.avgPnL)).toFixed(2)}M
            </Typography>
          </Box>
        </Paper>
      )}
    </Box>
  );
}

export type { MultiScenarioComparisonProps };
