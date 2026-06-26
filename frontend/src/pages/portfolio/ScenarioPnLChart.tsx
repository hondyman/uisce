import React, { useMemo } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  LabelList,
  ReferenceLine,
} from 'recharts';
import {
  Paper,
  Box,
  Typography,
  Grid,
  Card,
  Skeleton,
  Alert,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import { useMaterialTheme } from '../../hooks/useMaterialTheme';

interface ScenarioResult {
  scenario_id: string;
  name: string;
  pnl: number;
}

interface ScenarioPnLChartProps {
  data?: ScenarioResult[];
  isLoading?: boolean;
  error?: Error | null;
}

const formatCurrency = (value: number): string => {
  const absValue = Math.abs(value);
  if (absValue >= 1e6) {
    return `$${(value / 1e6).toFixed(2)}M`;
  } else if (absValue >= 1e3) {
    return `$${(value / 1e3).toFixed(2)}K`;
  }
  return `$${value.toFixed(0)}`;
};

export const ScenarioPnLChart: React.FC<ScenarioPnLChartProps> = ({
  data,
  isLoading,
  error,
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const { textColor, gridColor, borderColor, backgroundColor } = useMaterialTheme();

  const stats = useMemo(() => {
    if (!data || data.length === 0) {
      return {
        totalPnL: 0,
        avgPnL: 0,
        maxPnL: 0,
        minPnL: 0,
      };
    }

    const totalPnL = data.reduce((sum, s) => sum + s.pnl, 0);
    const avgPnL = totalPnL / data.length;
    const maxPnL = Math.max(...data.map((s) => s.pnl));
    const minPnL = Math.min(...data.map((s) => s.pnl));

    return { totalPnL, avgPnL, maxPnL, minPnL };
  }, [data]);

  if (isLoading) {
    return (
      <Box sx={{ display: 'grid', gap: 3 }}>
        <Paper elevation={1} sx={{ p: 3, backgroundColor, borderColor, border: 1, height: 450 }}>
          <Skeleton variant="text" width="40%" height={32} sx={{ mb: 2 }} />
          <Skeleton variant="rectangular" height={300} />
        </Paper>
        <Grid container spacing={2}>
          {[1, 2, 3, 4].map((i) => (
            <Grid item xs={12} sm={6} md={3} key={i}>
              <Skeleton variant="rectangular" height={100} />
            </Grid>
          ))}
        </Grid>
      </Box>
    );
  }

  if (error) {
    return (
      <Paper elevation={1} sx={{ p: 3, backgroundColor, borderColor, border: 1 }}>
        <Alert
          severity="error"
          sx={{
            backgroundColor: 'error.light',
            color: 'error.dark',
            '& .MuiAlert-icon': { color: 'error.main' },
          }}
        >
          {error?.message || 'Failed to load scenario data'}
        </Alert>
      </Paper>
    );
  }

  if (!data || data.length === 0) {
    return (
      <Paper elevation={1} sx={{ p: 3, backgroundColor, borderColor, border: 1 }}>
        <Typography
          variant="body2"
          color="textSecondary"
          sx={{ textAlign: 'center' }}
        >
          No scenario data available
        </Typography>
      </Paper>
    );
  }

  const chartData = data.map((scenario) => ({
    ...scenario,
    displayName: scenario.name,
  }));

  const chartHeight = isMobile ? 250 : 350;

  return (
    <Box sx={{ display: 'grid', gap: 3 }}>
      {/* Chart */}
      <Paper elevation={1} sx={{ backgroundColor, borderColor, border: 1 }}>
        <Box sx={{ p: 3 }}>
          <Typography
            variant="h6"
            component="h3"
            sx={{
              fontWeight: 700,
              mb: 3,
              color: textColor,
            }}
          >
            Scenario PnL Distribution
          </Typography>

          <Box sx={{ width: '100%', height: chartHeight }}>
            <ResponsiveContainer width="100%" height="100%">
              <BarChart
                data={chartData}
                margin={{
                  top: 40,
                  right: 30,
                  left: isMobile ? 40 : 60,
                  bottom: isMobile ? 60 : 80,
                }}
              >
                <CartesianGrid
                  strokeDasharray="3 3"
                  stroke={gridColor}
                  opacity={0.3}
                />
                <XAxis
                  dataKey="displayName"
                  angle={isMobile ? -35 : -45}
                  textAnchor="end"
                  height={isMobile ? 60 : 100}
                  tick={{
                    fontSize: isMobile ? 11 : 12,
                    fill: textColor,
                  }}
                />
                <YAxis
                  tick={{
                    fontSize: isMobile ? 11 : 12,
                    fill: textColor,
                  }}
                  tickFormatter={(value) => formatCurrency(value)}
                />
                <Tooltip
                  contentStyle={{
                    backgroundColor: theme.palette.background.paper,
                    border: `1px solid ${borderColor}`,
                    borderRadius: theme.shape.borderRadius,
                    color: textColor,
                  }}
                  formatter={(value: number) => [formatCurrency(value), 'PnL']}
                  labelFormatter={(label) => `Scenario: ${label}`}
                  labelStyle={{ color: textColor }}
                />
                <ReferenceLine
                  y={0}
                  stroke={textColor}
                  strokeDasharray="5 5"
                  isFront
                />
                <Bar
                  dataKey="pnl"
                  fill={theme.palette.primary.main}
                  radius={[8, 8, 0, 0]}
                  isAnimationActive={true}
                  shape={<CustomBar />}
                >
                  <LabelList
                    dataKey="pnl"
                    position="top"
                    formatter={(value: number) => formatCurrency(value)}
                    fill={textColor}
                    fontSize={11}
                    fontWeight={600}
                  />
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          </Box>
        </Box>
      </Paper>

      {/* Summary Statistics */}
      <Grid container spacing={2}>
        <Grid item xs={12} sm={6} md={3}>
          <Card
            elevation={1}
            sx={{
              p: 2.5,
              backgroundColor,
              borderColor,
              border: 1,
              height: '100%',
            }}
          >
            <Typography
              variant="caption"
              sx={{
                display: 'block',
                fontWeight: 700,
                color: 'textSecondary',
                mb: 1.5,
                textTransform: 'uppercase',
                letterSpacing: 0.5,
              }}
            >
              Total PnL
            </Typography>
            <Typography
              variant="h6"
              sx={{
                fontWeight: 700,
                color: stats.totalPnL >= 0 ? 'success.main' : 'error.main',
              }}
            >
              {formatCurrency(stats.totalPnL)}
            </Typography>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card
            elevation={1}
            sx={{
              p: 2.5,
              backgroundColor,
              borderColor,
              border: 1,
              height: '100%',
            }}
          >
            <Typography
              variant="caption"
              sx={{
                display: 'block',
                fontWeight: 700,
                color: 'textSecondary',
                mb: 1.5,
                textTransform: 'uppercase',
                letterSpacing: 0.5,
              }}
            >
              Average PnL
            </Typography>
            <Typography
              variant="h6"
              sx={{
                fontWeight: 700,
                color: stats.avgPnL >= 0 ? 'success.main' : 'error.main',
              }}
            >
              {formatCurrency(stats.avgPnL)}
            </Typography>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card
            elevation={1}
            sx={{
              p: 2.5,
              backgroundColor,
              borderColor,
              border: 1,
              height: '100%',
            }}
          >
            <Typography
              variant="caption"
              sx={{
                display: 'block',
                fontWeight: 700,
                color: 'textSecondary',
                mb: 1.5,
                textTransform: 'uppercase',
                letterSpacing: 0.5,
              }}
            >
              Best Case
            </Typography>
            <Typography
              variant="h6"
              sx={{
                fontWeight: 700,
                color: 'success.main',
              }}
            >
              {formatCurrency(stats.maxPnL)}
            </Typography>
          </Card>
        </Grid>

        <Grid item xs={12} sm={6} md={3}>
          <Card
            elevation={1}
            sx={{
              p: 2.5,
              backgroundColor,
              borderColor,
              border: 1,
              height: '100%',
            }}
          >
            <Typography
              variant="caption"
              sx={{
                display: 'block',
                fontWeight: 700,
                color: 'textSecondary',
                mb: 1.5,
                textTransform: 'uppercase',
                letterSpacing: 0.5,
              }}
            >
              Worst Case
            </Typography>
            <Typography
              variant="h6"
              sx={{
                fontWeight: 700,
                color: 'error.main',
              }}
            >
              {formatCurrency(stats.minPnL)}
            </Typography>
          </Card>
        </Grid>
      </Grid>
    </Box>
  );
};

// Custom bar component to color bars based on positive/negative values
interface CustomBarProps {
  fill?: string;
  x?: number;
  y?: number;
  width?: number;
  height?: number;
  payload?: ScenarioResult;
}

const CustomBar: React.FC<CustomBarProps> = (props) => {
  const { x = 0, y = 0, width = 0, height = 0, payload } = props;
  const theme = useTheme();

  if (!payload) return null;

  const isNegative = payload.pnl < 0;
  const barColor = isNegative 
    ? theme.palette.error.main
    : theme.palette.primary.main;

  return (
    <rect
      x={x}
      y={y}
      width={width}
      height={height}
      fill={barColor}
      rx={4}
      ry={4}
    />
  );
};

export default ScenarioPnLChart;
