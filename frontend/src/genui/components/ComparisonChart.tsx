import React from 'react';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Card, CardContent, Typography, Box, useTheme } from '@mui/material';
import { motion } from 'framer-motion';

interface ComparisonChartProps {
  metric: string;
  benchmark: string;
  period: string;
}

// Mock data generator
const generateData = (period: string) => {
  const data = [];
  const points = period === 'YTD' ? 10 : 30;
  for (let i = 0; i < points; i++) {
    data.push({
      date: `Day ${i + 1}`,
      metric: 100 + Math.random() * 20 - 10 + (i * 2), // Upward trend
      benchmark: 100 + Math.random() * 10 - 5 + (i * 1), // Slower trend
    });
  }
  return data;
};

export function ComparisonChart({ metric, benchmark, period }: ComparisonChartProps) {
  const data = generateData(period);
  const theme = useTheme();

  return (
    <motion.div 
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
    >
      <Card sx={{ my: 2, borderRadius: 3, border: '1px solid', borderColor: 'divider', boxShadow: theme.shadows[2] }}>
        <CardContent>
          <Box sx={{ mb: 2 }}>
            <Typography variant="h6" component="h3" sx={{ fontWeight: 'semibold', color: 'text.primary' }}>
              {metric} vs {benchmark}
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Period: {period}
            </Typography>
          </Box>
          
          <Box sx={{ height: 300, width: '100%' }}>
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data}>
                <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
                <XAxis dataKey="date" stroke={theme.palette.text.secondary} fontSize={12} />
                <YAxis stroke={theme.palette.text.secondary} fontSize={12} />
                <Tooltip 
                  contentStyle={{ 
                    backgroundColor: theme.palette.background.paper, 
                    border: 'none', 
                    borderRadius: '8px', 
                    boxShadow: theme.shadows[3],
                    color: theme.palette.text.primary
                  }}
                />
                <Legend />
                <Line 
                  type="monotone" 
                  dataKey="metric" 
                  name={metric} 
                  stroke={theme.palette.primary.main} 
                  strokeWidth={2} 
                  dot={false} 
                />
                <Line 
                  type="monotone" 
                  dataKey="benchmark" 
                  name={benchmark} 
                  stroke={theme.palette.secondary.main} 
                  strokeWidth={2} 
                  dot={false} 
                />
              </LineChart>
            </ResponsiveContainer>
          </Box>
        </CardContent>
      </Card>
    </motion.div>
  );
}

