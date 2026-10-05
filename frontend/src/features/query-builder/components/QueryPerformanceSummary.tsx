import React from 'react';
import { Typography } from '@mui/material';

/** Minimal stub for plan metrics summary. */
const QueryPerformanceSummary: React.FC<{ plan?: unknown }> = () => (
  <Typography variant="caption" color="text.secondary">
    Performance summary unavailable in this build.
  </Typography>
);

export default QueryPerformanceSummary;
