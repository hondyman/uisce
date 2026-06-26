import type { FC } from 'react';
import { Bundle } from '../types';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  Chip,
} from '@mui/material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Area,
  AreaChart,
} from 'recharts';

interface CapitalDeploymentChartProps {
  data?: Array<{
    quarter: string;
    committed: number;
    deployed: number;
    target: number;
  }>;
  title?: string;
  showArea?: boolean;
  compact?: boolean;
  selectedFunds?: string[];
  excelResults?: any;
  bundle?: Bundle;
}

const DEFAULT_DATA = [
  { quarter: 'Q1 2020', committed: 10000000, deployed: 2000000, target: 2500000 },
  { quarter: 'Q2 2020', committed: 15000000, deployed: 5000000, target: 3750000 },
  { quarter: 'Q3 2020', committed: 20000000, deployed: 8000000, target: 5000000 },
  { quarter: 'Q4 2020', committed: 25000000, deployed: 12000000, target: 6250000 },
  { quarter: 'Q1 2021', committed: 30000000, deployed: 16000000, target: 7500000 },
  { quarter: 'Q2 2021', committed: 35000000, deployed: 21000000, target: 8750000 },
  { quarter: 'Q3 2021', committed: 40000000, deployed: 26000000, target: 10000000 },
  { quarter: 'Q4 2021', committed: 45000000, deployed: 32000000, target: 11250000 },
];

export const CapitalDeploymentChart: FC<CapitalDeploymentChartProps> = ({
  data = DEFAULT_DATA,
  title = 'Capital Deployment Over Time',
  showArea = false,
  compact = false,
}) => {
  const formatCurrency = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(value);
  };

  const CustomTooltip = ({ active, payload, label }: any) => {
    if (active && payload && payload.length) {
      return (
        <Box sx={{
          backgroundColor: 'background.paper',
          p: 1,
          border: 1,
          borderColor: 'divider',
          borderRadius: 1,
          boxShadow: 1
        }}>
          <Typography variant="body2" fontWeight="bold">
            {label}
          </Typography>
          {payload.map((entry: any, index: number) => (
            <Typography key={index} variant="body2" sx={{ color: entry.color }}>
              {entry.name}: {formatCurrency(entry.value)}
            </Typography>
          ))}
        </Box>
      );
    }
    return null;
  };

  if (compact) {
    return (
      <Box>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {title}
        </Typography>
        <Box height={150}>
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="quarter"
                fontSize={10}
                angle={-45}
                textAnchor="end"
                height={60}
              />
              <YAxis fontSize={10} />
              <Tooltip content={<CustomTooltip />} />
              <Line
                type="monotone"
                dataKey="deployed"
                stroke="#8884d8"
                strokeWidth={2}
                dot={{ r: 3 }}
              />
              <Line
                type="monotone"
                dataKey="target"
                stroke="#82ca9d"
                strokeWidth={2}
                strokeDasharray="5 5"
                dot={{ r: 3 }}
              />
            </LineChart>
          </ResponsiveContainer>
        </Box>
      </Box>
    );
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          {title}
        </Typography>

        <Grid container spacing={2} sx={{ mb: 2 }}>
          <Grid item xs={12} sm={4}>
            <Box textAlign="center">
              <Typography variant="body2" color="text.secondary">
                Total Committed
              </Typography>
              <Typography variant="h6" color="primary">
                {formatCurrency(data[data.length - 1]?.committed || 0)}
              </Typography>
            </Box>
          </Grid>
          <Grid item xs={12} sm={4}>
            <Box textAlign="center">
              <Typography variant="body2" color="text.secondary">
                Total Deployed
              </Typography>
              <Typography variant="h6" color="success.main">
                {formatCurrency(data[data.length - 1]?.deployed || 0)}
              </Typography>
            </Box>
          </Grid>
          <Grid item xs={12} sm={4}>
            <Box textAlign="center">
              <Typography variant="body2" color="text.secondary">
                Deployment Pace
              </Typography>
              <Chip
                label={`${((data[data.length - 1]?.deployed || 0) / (data[data.length - 1]?.committed || 1) * 100).toFixed(1)}%`}
                color="primary"
                size="small"
              />
            </Box>
          </Grid>
        </Grid>

        <Box height={300}>
          <ResponsiveContainer width="100%" height="100%">
            {showArea ? (
              <AreaChart data={data}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="quarter" />
                <YAxis tickFormatter={formatCurrency} />
                <Tooltip content={<CustomTooltip />} />
                <Area
                  type="monotone"
                  dataKey="deployed"
                  stackId="1"
                  stroke="#8884d8"
                  fill="#8884d8"
                  fillOpacity={0.6}
                />
                <Area
                  type="monotone"
                  dataKey="target"
                  stackId="2"
                  stroke="#82ca9d"
                  fill="#82ca9d"
                  fillOpacity={0.3}
                />
              </AreaChart>
            ) : (
              <LineChart data={data}>
                <CartesianGrid strokeDasharray="3 3" />
                <XAxis dataKey="quarter" />
                <YAxis tickFormatter={formatCurrency} />
                <Tooltip content={<CustomTooltip />} />
                <Line
                  type="monotone"
                  dataKey="committed"
                  stroke="#ff7300"
                  strokeWidth={2}
                  name="Committed Capital"
                />
                <Line
                  type="monotone"
                  dataKey="deployed"
                  stroke="#8884d8"
                  strokeWidth={3}
                  name="Deployed Capital"
                />
                <Line
                  type="monotone"
                  dataKey="target"
                  stroke="#82ca9d"
                  strokeWidth={2}
                  strokeDasharray="5 5"
                  name="Target Deployment"
                />
              </LineChart>
            )}
          </ResponsiveContainer>
        </Box>
      </CardContent>
    </Card>
  );
};
