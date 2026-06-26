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
  PieChart,
  Pie,
  Cell,
  ResponsiveContainer,
  Tooltip,
  Legend,
} from 'recharts';

interface PortfolioCompositionChartProps {
  data?: Array<{
    name: string;
    value: number;
    color: string;
  }>;
  title?: string;
  showLegend?: boolean;
  compact?: boolean;
  selectedFunds?: string[];
  excelResults?: any;
  bundle?: Bundle;
}

const DEFAULT_DATA = [
  { name: 'Venture Capital', value: 45, color: '#8884d8' },
  { name: 'Private Equity', value: 30, color: '#82ca9d' },
  { name: 'Real Estate', value: 15, color: '#ffc658' },
  { name: 'Infrastructure', value: 10, color: '#ff7300' },
];

export const PortfolioCompositionChart: FC<PortfolioCompositionChartProps> = ({
  data = DEFAULT_DATA,
  title = 'Portfolio Composition',
  showLegend = true,
  compact = false,
}) => {
  const renderCustomizedLabel = (entry: any) => {
    return `${entry.name}: ${entry.value}%`;
  };

  if (compact) {
    return (
      <Box>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {title}
        </Typography>
        <Box display="flex" flexWrap="wrap" gap={1}>
          {data.map((item, index) => (
            <Chip
              key={index}
              label={`${item.name}: ${item.value}%`}
              size="small"
              sx={{
                backgroundColor: item.color,
                color: 'white',
                '& .MuiChip-label': { fontSize: '0.75rem' }
              }}
            />
          ))}
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

        <Box height={300}>
          <ResponsiveContainer width="100%" height="100%">
            <PieChart>
              <Pie
                data={data}
                cx="50%"
                cy="50%"
                labelLine={false}
                label={renderCustomizedLabel}
                outerRadius={80}
                fill="#8884d8"
                dataKey="value"
              >
                {data.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={entry.color} />
                ))}
              </Pie>
              {showLegend && (
                <Legend
                  verticalAlign="bottom"
                  height={36}
                  formatter={(value, entry: any) => (
                    <Box component="span" sx={{ color: entry.color }}>
                      {value}: {entry.payload.value}%
                    </Box>
                  )}
                />
              )}
              <Tooltip
                formatter={(value: number) => [`${value}%`, 'Allocation']}
              />
            </PieChart>
          </ResponsiveContainer>
        </Box>

        <Grid container spacing={1} sx={{ mt: 2 }}>
          {data.map((item, index) => (
            <Grid item xs={6} sm={3} key={index}>
              <Box display="flex" alignItems="center" gap={1}>
                <Box
                  sx={{
                    width: 12,
                    height: 12,
                    backgroundColor: item.color,
                    borderRadius: '50%',
                  }}
                />
                <Typography variant="caption">
                  {item.name}
                </Typography>
              </Box>
            </Grid>
          ))}
        </Grid>
      </CardContent>
    </Card>
  );
};
