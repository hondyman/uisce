import React, { useMemo } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';
import {
  Paper,
  Box,
  Typography,
  Skeleton,
  Alert,
  Grid,
  Card,
  useTheme,
  useMediaQuery,
} from '@mui/material';
import { useMaterialTheme } from '../../hooks/useMaterialTheme';

interface FactorExposure {
  factor_id: string;
  exposure: number;
}

interface FactorExposureChartProps {
  data?: FactorExposure[];
  isLoading?: boolean;
  error?: Error | null;
}

export const FactorExposureChart: React.FC<FactorExposureChartProps> = ({
  data,
  isLoading,
  error,
}) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const { textColor, gridColor, borderColor, backgroundColor } = useMaterialTheme();

  const stats = useMemo(() => {
    if (!data || data.length === 0) {
      return { maxExposure: 0, minExposure: 0, avgExposure: 0 };
    }
    const exposures = data.map((f) => f.exposure);
    return {
      maxExposure: Math.max(...exposures),
      minExposure: Math.min(...exposures),
      avgExposure: exposures.reduce((a, b) => a + b, 0) / exposures.length,
    };
  }, [data]);

  if (isLoading) {
    return (
      <Paper
        elevation={1}
        sx={{
          p: 3,
          backgroundColor,
          borderColor,
          border: 1,
          height: 450,
        }}
      >
        <Skeleton variant="text" width="40%" height={32} sx={{ mb: 2 }} />
        <Skeleton variant="rectangular" height={300} sx={{ mb: 2 }} />
        <Grid container spacing={2}>
          {[1, 2, 3].map((i) => (
            <Grid item xs={12} sm={6} md={4} key={i}>
              <Skeleton variant="text" height={24} />
            </Grid>
          ))}
        </Grid>
      </Paper>
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
          {error?.message || 'Failed to load factor exposure data'}
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
          No factor exposure data available
        </Typography>
      </Paper>
    );
  }

  const chartData = data.map((factor) => ({
    ...factor,
    name: factor.factor_id,
  }));

  const chartHeight = isMobile ? 250 : 300;

  return (
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
          Factor Exposures
        </Typography>

        <Box sx={{ width: '100%', height: chartHeight }}>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart
              data={chartData}
              margin={{
                top: 20,
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
                dataKey="name"
                angle={isMobile ? -35 : -45}
                textAnchor="end"
                height={isMobile ? 60 : 80}
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
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: theme.palette.background.paper,
                  border: `1px solid ${borderColor}`,
                  borderRadius: theme.shape.borderRadius,
                  color: textColor,
                }}
                formatter={(value: number) => [
                  `${value.toFixed(4)}`,
                  'Exposure',
                ]}
                labelStyle={{ color: textColor }}
              />
              <ReferenceLine
                y={0}
                stroke={textColor}
                strokeDasharray="5 5"
                isFront
              />
              <Bar
                dataKey="exposure"
                fill={theme.palette.primary.main}
                radius={[8, 8, 0, 0]}
                isAnimationActive={true}
              />
            </BarChart>
          </ResponsiveContainer>
        </Box>

        {/* Summary Statistics */}
        <Grid container spacing={2} sx={{ mt: 1 }}>
          <Grid item xs={12} sm={4}>
            <Card
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: theme.palette.action.hover,
                border: `1px solid ${borderColor}`,
              }}
            >
              <Typography
                variant="caption"
                sx={{
                  display: 'block',
                  fontWeight: 700,
                  color: 'textSecondary',
                  mb: 1,
                  textTransform: 'uppercase',
                  letterSpacing: 0.5,
                }}
              >
                Max Exposure
              </Typography>
              <Typography
                variant="h6"
                sx={{
                  fontWeight: 700,
                  color: 'primary.main',
                }}
              >
                {stats.maxExposure.toFixed(4)}
              </Typography>
            </Card>
          </Grid>

          <Grid item xs={12} sm={4}>
            <Card
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: theme.palette.action.hover,
                border: `1px solid ${borderColor}`,
              }}
            >
              <Typography
                variant="caption"
                sx={{
                  display: 'block',
                  fontWeight: 700,
                  color: 'textSecondary',
                  mb: 1,
                  textTransform: 'uppercase',
                  letterSpacing: 0.5,
                }}
              >
                Avg Exposure
              </Typography>
              <Typography
                variant="h6"
                sx={{
                  fontWeight: 700,
                  color: 'text.primary',
                }}
              >
                {stats.avgExposure.toFixed(4)}
              </Typography>
            </Card>
          </Grid>

          <Grid item xs={12} sm={4}>
            <Card
              elevation={0}
              sx={{
                p: 2,
                backgroundColor: theme.palette.action.hover,
                border: `1px solid ${borderColor}`,
              }}
            >
              <Typography
                variant="caption"
                sx={{
                  display: 'block',
                  fontWeight: 700,
                  color: 'textSecondary',
                  mb: 1,
                  textTransform: 'uppercase',
                  letterSpacing: 0.5,
                }}
              >
                Min Exposure
              </Typography>
              <Typography
                variant="h6"
                sx={{
                  fontWeight: 700,
                  color: stats.minExposure < 0 ? 'error.main' : 'warning.main',
                }}
              >
                {stats.minExposure.toFixed(4)}
              </Typography>
            </Card>
          </Grid>
        </Grid>
      </Box>
    </Paper>
  );
};

export default FactorExposureChart;
