import { Box } from '@mui/material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';

export interface TrendChartProps {
  data: any[];
  metricKey: string;
  threshold?: number;
  height?: number;
  color?: string;
  thresholdColor?: string;
}

export function TrendChart({
  data,
  metricKey,
  threshold,
  height = 250,
  color = '#3498DB',
  thresholdColor = '#E74C3C',
}: TrendChartProps) {
  return (
    <Box sx={{ width: '100%', height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data}>
          <XAxis
            dataKey="valuation_date"
            angle={-45}
            textAnchor="end"
            height={80}
            tick={{ fontSize: 12 }}
          />
          <YAxis tick={{ fontSize: 12 }} />
          <Tooltip
            contentStyle={{
              backgroundColor: 'rgba(255, 255, 255, 0.95)',
              border: '1px solid #ccc',
              borderRadius: '4px',
            }}
          />
          <Line
            type="monotone"
            dataKey={metricKey}
            stroke={color}
            strokeWidth={2}
            dot={false}
            isAnimationActive={false}
          />
          {threshold !== undefined && (
            <ReferenceLine
              y={threshold}
              stroke={thresholdColor}
              strokeDasharray="4 4"
              label={{ value: 'Threshold', position: 'right' }}
            />
          )}
        </LineChart>
      </ResponsiveContainer>
    </Box>
  );
}
