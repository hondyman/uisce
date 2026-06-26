import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Tabs,
  Tab,
  LinearProgress,
  Chip,
} from '@mui/material';
import {
  TrendingUp as UpIcon,
  TrendingDown as DownIcon,
} from '@mui/icons-material';
import { LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, PieChart, Pie, Cell } from 'recharts';
import { WidgetProps } from '../Dashboard';

interface PortfolioData {
  total_value: number;
  total_return_ytd: number;
  total_return_ytd_pct: number;
  day_change: number;
  day_change_pct: number;
  allocation: Array<{
    asset_class: string;
    value: number;
    percentage: number;
    color: string;
  }>;
  performance_history: Array<{
    date: string;
    value: number;
  }>;
}

export const PortfolioWidget: React.FC<WidgetProps> = () => {
  const [data, setData] = useState<PortfolioData | null>(null);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState(0);

  useEffect(() => {
    fetchPortfolioData();
  }, []);

  const fetchPortfolioData = async () => {
    try {
      const response = await fetch('/api/portfolio/summary');
      const portfolioData = await response.json();
      setData(portfolioData);

      // Track widget view
      await fetch('/api/portal/analytics', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          event_type: 'WIDGET_VIEW',
          event_data: { widget_id: 'portfolio' },
        }),
      });
    } catch (error) {
      console.error('Failed to fetch portfolio data:', error);
    } finally {
      setLoading(false);
    }
  };

  if (loading) {
    return <LinearProgress />;
  }

  if (!data) {
    return (
      <Box sx={{ textAlign: 'center', py: 4, color: 'text.secondary' }}>
        No portfolio data available
      </Box>
    );
  }

  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Portfolio Summary */}
      <Box sx={{ mb: 2 }}>
        <Typography variant="h4">
          ${data.total_value.toLocaleString()}
        </Typography>
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', mt: 1 }}>
          <Chip
            icon={data.total_return_ytd >= 0 ? <UpIcon /> : <DownIcon />}
            label={`${data.total_return_ytd >= 0 ? '+' : ''}$${Math.abs(data.total_return_ytd).toLocaleString()} (${data.total_return_ytd_pct.toFixed(2)}%)`}
            color={data.total_return_ytd >= 0 ? 'success' : 'error'}
            size="small"
          />
          <Typography variant="caption" color="text.secondary">
            YTD Return
          </Typography>
        </Box>
        <Typography variant="body2" color={data.day_change >= 0 ? 'success.main' : 'error.main'} sx={{ mt: 0.5 }}>
          {data.day_change >= 0 ? '+' : ''}${data.day_change.toFixed(2)} ({data.day_change_pct.toFixed(2)}%) Today
        </Typography>
      </Box>

      {/* Tabs */}
      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ borderBottom: 1, borderColor: 'divider', mb: 2 }}>
        <Tab label="Performance" />
        <Tab label="Allocation" />
      </Tabs>

      {/* Tab Content */}
      <Box sx={{ flex: 1, minHeight: 0 }}>
        {tab === 0 && (
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={data.performance_history}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis />
              <Tooltip />
              <Line type="monotone" dataKey="value" stroke="#1976d2" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        )}

        {tab === 1 && (
          <Box sx={{ height: '100%', display: 'flex' }}>
            <ResponsiveContainer width="60%" height="100%">
              <PieChart>
                <Pie
                  data={data.allocation}
                  dataKey="value"
                  nameKey="asset_class"
                  cx="50%"
                  cy="50%"
                  outerRadius={80}
                  label
                >
                  {data.allocation.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>

            <Box sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 1, justifyContent: 'center' }}>
              {data.allocation.map((item) => (
                <Box key={item.asset_class} sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                  <Box sx={{ width: 12, height: 12, bgcolor: item.color, borderRadius: '50%' }} />
                  <Typography variant="caption">
                    {item.asset_class}: {item.percentage.toFixed(1)}%
                  </Typography>
                </Box>
              ))}
            </Box>
          </Box>
        )}
      </Box>
    </Box>
  );
};
