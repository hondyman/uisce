"use client";

import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
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

  return (
    <motion.div 
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      className="p-4 bg-white rounded-xl shadow-lg border border-gray-100 my-4"
    >
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-gray-900">{metric} vs {benchmark}</h3>
        <p className="text-sm text-gray-500">Period: {period}</p>
      </div>
      
      <div className="h-[300px] w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data}>
            <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
            <XAxis dataKey="date" stroke="#9ca3af" fontSize={12} />
            <YAxis stroke="#9ca3af" fontSize={12} />
            <Tooltip 
              contentStyle={{ borderRadius: '8px', border: 'none', boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)' }}
            />
            <Legend />
            <Line 
              type="monotone" 
              dataKey="metric" 
              name={metric} 
              stroke="#2563eb" 
              strokeWidth={2} 
              dot={false} 
            />
            <Line 
              type="monotone" 
              dataKey="benchmark" 
              name={benchmark} 
              stroke="#9333ea" 
              strokeWidth={2} 
              dot={false} 
            />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </motion.div>
  );
}
