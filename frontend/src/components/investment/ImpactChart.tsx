import React from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer
} from 'recharts';
import { Box, Typography, Paper } from '@mui/material';

interface ImpactChartProps {
  portfolioWeights: { sector: string; weight: number }[];
  benchmarkWeights: { sector: string; weight: number }[];
}

const ImpactChart: React.FC<ImpactChartProps> = ({ portfolioWeights, benchmarkWeights }) => {
  // Merge data for the chart
  const data = benchmarkWeights.map((bm) => {
    const pf = portfolioWeights.find((p) => p.sector === bm.sector);
    return {
      sector: bm.sector,
      Benchmark: (bm.weight * 100).toFixed(1),
      Portfolio: (pf ? pf.weight * 100 : 0).toFixed(1),
    };
  });

  return (
    <Paper elevation={3} sx={{ p: 2, height: 400 }}>
      <Typography variant="h6" gutterBottom>
        Sector Allocation Impact
      </Typography>
      <ResponsiveContainer width="100%" height="90%">
        <BarChart
          data={data}
          margin={{
            top: 20,
            right: 30,
            left: 20,
            bottom: 5,
          }}
        >
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="sector" />
          <YAxis label={{ value: '% Allocation', angle: -90, position: 'insideLeft' }} />
          <Tooltip />
          <Legend />
          <Bar dataKey="Benchmark" fill="#8884d8" />
          <Bar dataKey="Portfolio" fill="#82ca9d" />
        </BarChart>
      </ResponsiveContainer>
    </Paper>
  );
};

export default ImpactChart;
