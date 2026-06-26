import React, { useState, useEffect, useMemo } from 'react';
import { gql, useMutation } from '@apollo/client';
import {
  Box,
  Typography,
  Alert,
  CircularProgress,
  Paper,
  Grid,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  FormHelperText,
} from '@mui/material';
import { Chart as ChartJS, Tooltip, Legend, CategoryScale, LinearScale } from 'chart.js';
import { MatrixController, MatrixElement } from 'chartjs-chart-matrix';
import { Chart } from 'react-chartjs-2';
import { subMonths, format } from 'date-fns';
import { useDrillDown } from '../../../contexts/DrillDownContext';

ChartJS.register(MatrixController, MatrixElement, Tooltip, Legend, CategoryScale, LinearScale);

const SIMULATE_HISTORICAL_REPLAY = gql`
  mutation SimulateHistoricalReplay($fromDate: date!, $toDate: date!, $bucketSize: String!) {
    simulate_historical_replay(from_date: $fromDate, to_date: $toDate, bucket_size: $bucketSize) {
      policy_id
      policy_name
      time_bucket
      total_runs
      blocked_runs
      top_violation_codes
    }
  }
`;

interface ReplayCell {
  policy_id: string;
  policy_name: string;
  time_bucket: string;
  total_runs: number;
  blocked_runs: number;
  top_violation_codes: string[];
}

interface ReplayData {
  simulate_historical_replay: ReplayCell[];
}

const PolicyHistoricalReplayPage: React.FC = () => {
  const [bucketSize, setBucketSize] = useState('week');
  const [runReplay, { data, loading, error }] = useMutation<ReplayData>(SIMULATE_HISTORICAL_REPLAY);
  const { showDrillDown } = useDrillDown();

  useEffect(() => {
    // Run on initial load
    const toDate = new Date();
    const fromDate = subMonths(toDate, 3);
    runReplay({
      variables: {
        fromDate: format(fromDate, 'yyyy-MM-dd'),
        toDate: format(toDate, 'yyyy-MM-dd'),
        bucketSize,
      },
    });
  }, [runReplay, bucketSize]);
  
  const chartConfig = useMemo(() => {
    if (!data?.simulate_historical_replay) {
      return null;
    }

    const replayData = data.simulate_historical_replay;
    const policies: string[] = [...new Set(replayData.map((r) => r.policy_name))].sort();
    const buckets: string[] = [...new Set(replayData.map((r) => r.time_bucket))].sort();

    const dataset = replayData.map((r) => ({
      x: r.time_bucket,
      y: r.policy_name,
      policyId: r.policy_id,
      v: r.total_runs > 0 ? r.blocked_runs / r.total_runs : 0,
      blocked: r.blocked_runs,
      total: r.total_runs,
      topCodes: r.top_violation_codes,
    }));

    const chartData = {
      datasets: [
        {
          label: 'Policy Block Rate',
          data: dataset,
          backgroundColor: (ctx: any) => {
            if (!ctx.raw) return 'rgba(0,0,0,0.1)';
            const value = ctx.raw.v;
            if (value > 0.5) return 'rgba(215, 48, 39, 0.8)'; // High block rate
            if (value > 0.1) return 'rgba(252, 141, 89, 0.8)'; // Medium
            if (value > 0) return 'rgba(254, 224, 144, 0.8)'; // Low
            return 'rgba(224, 224, 224, 0.5)'; // No blocks
          },
          borderColor: 'grey',
          borderWidth: 1,
          width: ({ chart }: any) => (chart.chartArea || {}).width / buckets.length - 1,
          height: ({ chart }: any) => (chart.chartArea || {}).height / policies.length - 1,
        },
      ],
    };

    const options = {
      responsive: true,
      maintainAspectRatio: false,
      scales: {
        x: {
          type: 'category' as const,
          labels: buckets,
          position: 'top' as const,
          ticks: {
            autoSkip: true,
            maxRotation: 90,
            minRotation: 45,
          },
        },
        y: {
          type: 'category' as const,
          labels: policies,
          offset: true,
        },
      },
      plugins: {
        legend: {
          display: false,
        },
        tooltip: {
          callbacks: {
            title: (ctx: any) => `${ctx[0].raw.y} @ ${ctx[0].raw.x}`,
            label: (ctx: any) => {
              const d = ctx.raw;
              const percentage = (d.v * 100).toFixed(1);
              return [
                `${percentage}% blocked (${d.blocked}/${d.total} runs)`,
                `Top violations: ${d.topCodes.join(', ') || 'N/A'}`,
              ];
            },
          },
        },
      },
      onClick: (_evt: any, elements: any[]) => {
        if (elements.length > 0) {
          const { raw } = elements[0].element.$context;
          const toDate = new Date();
          const fromDate = subMonths(toDate, 3);

          showDrillDown('historical', {
            policyId: raw.policyId,
            policyName: raw.y,
            bucket: raw.x,
            fromDate: format(fromDate, 'yyyy-MM-dd'),
            toDate: format(toDate, 'yyyy-MM-dd'),
            bucketSize: bucketSize,
          });
        }
      },
    };
    // Add bucketSize to dependency array as it's used in the effect
    return { chartData, options };
  }, [data, bucketSize, showDrillDown]);

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Policy Impact Heatmap
      </Typography>
      <Typography paragraph color="text.secondary">
        Visualize the block rate of policies over time. Darker cells indicate a higher percentage of changes being blocked.
      </Typography>

      <Paper sx={{ p: 2, mb: 3 }}>
        <Grid container spacing={2} alignItems="center">
          <Grid item>
            <FormControl size="small">
              <InputLabel id="bucket-size-label">Bucket Size</InputLabel>
              <Select
                labelId="bucket-size-label"
                value={bucketSize}
                label="Bucket Size"
                onChange={(e) => setBucketSize(e.target.value)}
              >
                <MenuItem value="week">Week</MenuItem>
                <MenuItem value="month">Month</MenuItem>
              </Select>
              <FormHelperText>Group results by week or month.</FormHelperText>
            </FormControl>
          </Grid>
        </Grid>
      </Paper>

      <Paper sx={{ p: 3, height: '70vh', position: 'relative' }}>
        {loading && (
          <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
            <CircularProgress />
            <Typography sx={{ ml: 2 }}>Running Historical Replay...</Typography>
          </Box>
        )}
        {error && <Alert severity="error">Failed to run replay: {error.message}</Alert>}
        {chartConfig && (
          <Chart type={"matrix" as any} data={chartConfig.chartData} options={chartConfig.options} />
        )}
        {!loading && !error && !chartConfig && (
          <Alert severity="info">No historical data found for the selected period.</Alert>
        )}
      </Paper>
    </Box>
  );
};

export default PolicyHistoricalReplayPage;