import type { FC } from 'react';
import {
  Box,
  Paper,
  Typography,
  useTheme
} from '@mui/material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer
} from 'recharts';

interface GrossIRRChartProps {
  selectedFunds?: string[];
}

export const GrossIRRChart: FC<GrossIRRChartProps> = ({ selectedFunds: _selectedFunds = [] }) => {
  const theme = useTheme();

  // Mock IRR data
  const irrData = [
    { year: '2019', fund1: 12.5, fund2: 15.2, fund3: 8.7, benchmark: 10.3 },
    { year: '2020', fund1: 18.3, fund2: 22.1, fund3: 14.5, benchmark: 12.8 },
    { year: '2021', fund1: -5.2, fund2: -3.8, fund3: -8.1, benchmark: -2.5 },
    { year: '2022', fund1: 25.7, fund2: 28.4, fund3: 19.3, benchmark: 18.9 },
    { year: '2023', fund1: 22.1, fund2: 24.8, fund3: 16.7, benchmark: 15.4 }
  ];

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        Gross IRR Performance
      </Typography>

      <Box sx={{ height: 300 }}>
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={irrData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="year"
              fontSize={12}
            />
            <YAxis
              fontSize={12}
              label={{ value: 'IRR (%)', angle: -90, position: 'insideLeft' }}
            />
            <Tooltip
              formatter={(value: number) => [`${value}%`, '']}
              labelStyle={{ color: theme.palette.text.primary }}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="fund1"
              stroke={theme.palette.primary.main}
              strokeWidth={2}
              name="Fund Alpha"
            />
            <Line
              type="monotone"
              dataKey="fund2"
              stroke={theme.palette.secondary.main}
              strokeWidth={2}
              name="Fund Beta"
            />
            <Line
              type="monotone"
              dataKey="fund3"
              stroke={theme.palette.success.main}
              strokeWidth={2}
              name="Fund Gamma"
            />
            <Line
              type="monotone"
              dataKey="benchmark"
              stroke={theme.palette.grey[500]}
              strokeWidth={2}
              strokeDasharray="5 5"
              name="Benchmark"
            />
          </LineChart>
        </ResponsiveContainer>
      </Box>

      <Box sx={{ mt: 2, display: 'flex', gap: 3 }}>
        <Box>
          <Typography variant="body2" color="text.secondary">Latest IRR</Typography>
          <Typography variant="h6" color="primary.main">
            +22.1%
          </Typography>
        </Box>
        <Box>
          <Typography variant="body2" color="text.secondary">3-Year Avg</Typography>
          <Typography variant="h6" color="primary.main">
            +14.6%
          </Typography>
        </Box>
        <Box>
          <Typography variant="body2" color="text.secondary">Outperformance</Typography>
          <Typography variant="h6" color="success.main">
            +6.7%
          </Typography>
        </Box>
      </Box>
    </Paper>
  );
};
