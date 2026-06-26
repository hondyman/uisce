// React import not required directly in this file (JSX runtime handles it)
// Box is referenced in JSX runtime; keep import to satisfy typing
import {
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

interface NAVEvolutionChartProps {
  fundId?: string;
  data?: any[];
  selectedFunds?: string[];
}

export const NAVEvolutionChart: React.FC<NAVEvolutionChartProps> = ({
  fundId: _fundId = 'fund-1',
  data,
  selectedFunds: _selectedFunds = []
}) => {
  const theme = useTheme();

  // Mock data for NAV evolution
  const mockData = [
    { quarter: 'Q1 2020', nav: 100, benchmark: 98 },
    { quarter: 'Q2 2020', nav: 95, benchmark: 97 },
    { quarter: 'Q3 2020', nav: 102, benchmark: 99 },
    { quarter: 'Q4 2020', nav: 108, benchmark: 101 },
    { quarter: 'Q1 2021', nav: 115, benchmark: 103 },
    { quarter: 'Q2 2021', nav: 118, benchmark: 105 },
    { quarter: 'Q3 2021', nav: 122, benchmark: 107 },
    { quarter: 'Q4 2021', nav: 128, benchmark: 109 },
    { quarter: 'Q1 2022', nav: 135, benchmark: 111 },
    { quarter: 'Q2 2022', nav: 142, benchmark: 113 },
    { quarter: 'Q3 2022', nav: 148, benchmark: 115 },
    { quarter: 'Q4 2022', nav: 152, benchmark: 117 },
  ];

  const chartData = data || mockData;

  return (
    <Paper sx={{ p: 2, height: 400 }}>
      <Typography variant="h6" gutterBottom>
        NAV Evolution
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Net Asset Value progression over time with benchmark comparison
      </Typography>

      <ResponsiveContainer width="100%" height="85%">
        <LineChart data={chartData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis
            dataKey="quarter"
            tick={{ fontSize: 12 }}
          />
          <YAxis
            tick={{ fontSize: 12 }}
            label={{ value: 'NAV ($)', angle: -90, position: 'insideLeft' }}
          />
          <Tooltip
            formatter={(value: number, name: string) => [
              `$${value.toLocaleString()}`,
              name === 'nav' ? 'Fund NAV' : 'Benchmark'
            ]}
          />
          <Legend />
          <Line
            type="monotone"
            dataKey="nav"
            stroke={theme.palette.primary.main}
            strokeWidth={2}
            name="Fund NAV"
            dot={{ r: 4 }}
          />
          <Line
            type="monotone"
            dataKey="benchmark"
            stroke={theme.palette.secondary.main}
            strokeWidth={2}
            strokeDasharray="5 5"
            name="Benchmark"
            dot={{ r: 4 }}
          />
        </LineChart>
      </ResponsiveContainer>
    </Paper>
  );
};
