import { Box } from '@mui/material';
import { LineChart, Line, ResponsiveContainer } from 'recharts';

export interface SparklineProps {
  data: any[];
  metricKey: string;
  color?: string;
  height?: number;
}

export function Sparkline({
  data,
  metricKey,
  color = '#2ECC71',
  height = 40,
}: SparklineProps) {
  return (
    <Box sx={{ width: '100%', height }}>
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={data}>
          <Line
            type="monotone"
            dataKey={metricKey}
            stroke={color}
            strokeWidth={2}
            dot={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </Box>
  );
}
