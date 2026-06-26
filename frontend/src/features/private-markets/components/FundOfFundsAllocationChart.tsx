import type { FC } from 'react';
import { Bundle } from '../types';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  LinearProgress,
} from '@mui/material';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';

interface FundOfFundsAllocationChartProps {
  data?: Array<{
    fundName: string;
    allocation: number;
    committed: number;
    invested: number;
    available: number;
    performance: number;
  }>;
  title?: string;
  showTable?: boolean;
  compact?: boolean;
  selectedFunds?: string[];
  excelResults?: any;
  bundle?: Bundle;
}

const DEFAULT_DATA = [
  {
    fundName: 'Tech Growth Fund III',
    allocation: 25,
    committed: 25000000,
    invested: 20000000,
    available: 5000000,
    performance: 15.6,
  },
  {
    fundName: 'Infrastructure Partners II',
    allocation: 20,
    committed: 20000000,
    invested: 18000000,
    available: 2000000,
    performance: 12.3,
  },
  {
    fundName: 'Real Estate Opportunity',
    allocation: 15,
    committed: 15000000,
    invested: 12000000,
    available: 3000000,
    performance: 18.9,
  },
  {
    fundName: 'Healthcare Innovation',
    allocation: 40,
    committed: 40000000,
    invested: 35000000,
    available: 5000000,
    performance: 22.1,
  },
];

export const FundOfFundsAllocationChart: FC<FundOfFundsAllocationChartProps> = ({
  data = DEFAULT_DATA,
  title = 'Fund of Funds Allocation',
  showTable = true,
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

  const getPerformanceColor = (performance: number) => {
    if (performance >= 20) return 'success';
    if (performance >= 15) return 'primary';
    if (performance >= 10) return 'warning';
    return 'error';
  };

  if (compact) {
    return (
      <Box>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {title}
        </Typography>
        <Box height={200}>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="fundName"
                angle={-45}
                textAnchor="end"
                height={80}
                fontSize={10}
              />
              <YAxis fontSize={10} />
              <Tooltip
                formatter={(value: number) => [`${value}%`, 'Allocation']}
                labelFormatter={(label) => `Fund: ${label}`}
              />
              <Bar dataKey="allocation" fill="#8884d8" />
            </BarChart>
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

        <Box height={300} mb={3}>
          <ResponsiveContainer width="100%" height="100%">
            <BarChart data={data}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis
                dataKey="fundName"
                angle={-45}
                textAnchor="end"
                height={80}
              />
              <YAxis />
              <Tooltip
                formatter={(value: number) => [`${value}%`, 'Allocation']}
                labelFormatter={(label) => `Fund: ${label}`}
              />
              <Bar dataKey="allocation" fill="#8884d8" />
            </BarChart>
          </ResponsiveContainer>
        </Box>

        {showTable && (
          <TableContainer component={Paper} variant="outlined">
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Fund Name</TableCell>
                  <TableCell align="right">Allocation</TableCell>
                  <TableCell align="right">Committed</TableCell>
                  <TableCell align="right">Invested</TableCell>
                  <TableCell align="right">Available</TableCell>
                  <TableCell align="right">Performance</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {data.map((row, index) => (
                  <TableRow key={index}>
                    <TableCell component="th" scope="row">
                      <Typography variant="body2" fontWeight="medium">
                        {row.fundName}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {row.allocation}%
                      </Typography>
                      <LinearProgress
                        variant="determinate"
                        value={row.allocation}
                        sx={{ height: 4, mt: 0.5 }}
                      />
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {formatCurrency(row.committed)}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2">
                        {formatCurrency(row.invested)}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Typography variant="body2" color="success.main">
                        {formatCurrency(row.available)}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Chip
                        label={`${row.performance.toFixed(1)}%`}
                        size="small"
                        color={getPerformanceColor(row.performance)}
                        variant="outlined"
                      />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}
      </CardContent>
    </Card>
  );
};
