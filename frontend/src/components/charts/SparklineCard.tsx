import { Card, CardContent, Typography, Box } from '@mui/material';
import { Sparkline, SparklineProps } from './Sparkline';

export interface SparklineCardProps extends Omit<SparklineProps, 'metricKey'> {
  title: string;
  metricKey?: string;
}

export function SparklineCard({
  title,
  data,
  metricKey = 'value',
  color,
}: SparklineCardProps) {
  const latest = data.length > 0 ? data[data.length - 1][metricKey] : 0;
  const previous = data.length > 1 ? data[data.length - 2][metricKey] : latest;
  const change = previous !== 0 ? ((latest - previous) / previous) * 100 : 0;
  const changeColor = change > 0 ? '#2ECC71' : '#E74C3C';

  return (
    <Card>
      <CardContent>
        <Typography color="textSecondary" gutterBottom>
          {title}
        </Typography>
        <Typography variant="h6">
          {typeof latest === 'number' ? latest.toFixed(2) : latest}
        </Typography>
        <Box sx={{ mt: 1, color: changeColor, fontSize: '0.875rem' }}>
          {change > 0 ? '↑' : '↓'} {Math.abs(change).toFixed(1)}%
        </Box>
        <Box sx={{ mt: 2 }}>
          <Sparkline data={data} metricKey={metricKey} color={color} height={40} />
        </Box>
      </CardContent>
    </Card>
  );
}
