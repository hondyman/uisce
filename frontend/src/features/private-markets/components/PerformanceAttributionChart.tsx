// React import not required directly in this file (JSX runtime handles it)
import { Bundle } from '../types';
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

interface PerformanceAttributionChartProps {
  selectedFunds?: string[];
  excelResults?: Record<string, Record<string, any>> | null;
  bundle?: Bundle | null;
}

export const PerformanceAttributionChart: React.FC<PerformanceAttributionChartProps> = ({
  selectedFunds: _selectedFunds = []
}) => {
  const theme = useTheme();

  // Mock performance attribution data
  const attributionData = [
    { quarter: 'Q1 2022', totalReturn: 8.5, benchmark: 6.2, alpha: 2.3 },
    { quarter: 'Q2 2022', totalReturn: 12.1, benchmark: 7.8, alpha: 4.3 },
    { quarter: 'Q3 2022', totalReturn: -3.2, benchmark: -1.5, alpha: -1.7 },
    { quarter: 'Q4 2022', totalReturn: 15.8, benchmark: 9.4, alpha: 6.4 },
    { quarter: 'Q1 2023', totalReturn: 9.7, benchmark: 5.9, alpha: 3.8 },
    { quarter: 'Q2 2023', totalReturn: 11.3, benchmark: 8.1, alpha: 3.2 },
    { quarter: 'Q3 2023', totalReturn: 7.4, benchmark: 4.7, alpha: 2.7 },
    { quarter: 'Q4 2023', totalReturn: 13.2, benchmark: 9.8, alpha: 3.4 }
  ];

  return (
    <Paper sx={{ p: 2 }}>
      <Typography variant="h6" gutterBottom>
        Performance Attribution
      </Typography>

      <Box sx={{ height: 300 }}>
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={attributionData}>
            <CartesianGrid strokeDasharray="3 3" />
            <XAxis
              dataKey="quarter"
              fontSize={12}
            />
            <YAxis
              fontSize={12}
              label={{ value: 'Return (%)', angle: -90, position: 'insideLeft' }}
            />
            <Tooltip
              formatter={(value: number) => [`${value}%`, '']}
              labelStyle={{ color: theme.palette.text.primary }}
            />
            <Legend />
            <Line
              type="monotone"
              dataKey="totalReturn"
              stroke={theme.palette.primary.main}
              strokeWidth={2}
              name="Total Return"
            />
            <Line
              type="monotone"
              dataKey="benchmark"
              stroke={theme.palette.secondary.main}
              strokeWidth={2}
              name="Benchmark"
            />
            <Line
              type="monotone"
              dataKey="alpha"
              stroke={theme.palette.success.main}
              strokeWidth={2}
              name="Alpha"
            />
          </LineChart>
          </ResponsiveContainer>
        </Box>

      <Box sx={{ mt: 2, display: 'flex', gap: 3 }}>
        <Box>
          <Typography variant="body2" color="text.secondary">Latest Alpha</Typography>
          <Typography variant="h6" color="success.main">
            +3.4%
          </Typography>
        </Box>
        <Box>
          <Typography variant="body2" color="text.secondary">Avg Alpha</Typography>
          <Typography variant="h6" color="success.main">
            +2.8%
          </Typography>
        </Box>
        <Box>
          <Typography variant="body2" color="text.secondary">Benchmark</Typography>
          <Typography variant="h6">
            +9.8%
          </Typography>
        </Box>
      </Box>
    </Paper>
  );
};
