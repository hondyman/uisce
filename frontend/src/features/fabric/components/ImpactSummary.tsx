import type { FC, ReactNode } from 'react';
import { Box, Typography, Paper, Grid, Chip } from '@mui/material';
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward';
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward';

interface Summary {
  total_runs: number;
  changed_decisions: number;
  blocks_added: number;
  blocks_removed: number;
  top_new_violation_codes: string[];
  top_removed_violation_codes: string[];
}

interface ImpactSummaryProps {
  summary: Summary;
}

const StatCard: FC<{ title: string; value: ReactNode }> = ({ title, value }) => (
  <Grid item xs={6} sm={3}>
    <Paper variant="outlined" sx={{ p: 2, textAlign: 'center' }}>
      <Typography color="text.secondary" gutterBottom>
        {title}
      </Typography>
      <Typography variant="h5" component="div">
        {value}
      </Typography>
    </Paper>
  </Grid>
);

const ImpactSummary: FC<ImpactSummaryProps> = ({ summary }) => {
  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Impact Summary
      </Typography>
      <Grid container spacing={2}>
        <StatCard title="Runs Compared" value={summary.total_runs} />
        <StatCard title="Decisions Changed" value={summary.changed_decisions} />
        <StatCard
          title="Blocks Added"
          value={
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'error.main' }}>
              <ArrowUpwardIcon fontSize="small" />
              {summary.blocks_added}
            </Box>
          }
        />
        <StatCard
          title="Blocks Removed"
          value={
            <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'success.main' }}>
              <ArrowDownwardIcon fontSize="small" />
              {summary.blocks_removed}
            </Box>
          }
        />
      </Grid>
      <Box sx={{ mt: 2 }}>
        <Typography variant="subtitle2">Top New Violations:</Typography>
        {summary.top_new_violation_codes.map((code) => (
          <Chip key={code} label={code} size="small" sx={{ mr: 1 }} color="error" />
        ))}
      </Box>
    </Box>
  );
};

export default ImpactSummary;
