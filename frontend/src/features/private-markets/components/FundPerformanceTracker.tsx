import type { FC } from 'react';
import { Bundle } from '../types';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Grid,
  LinearProgress,
  Chip,
} from '@mui/material';
import {
  TrendingUp as TrendingUpIcon,
  TrendingDown as TrendingDownIcon,
} from '@mui/icons-material';

interface FundPerformanceTrackerProps {
  fundId?: string;
  showDetails?: boolean;
  compact?: boolean;
  selectedFunds?: string[];
  excelResults?: any;
  bundle?: Bundle;
}

export const FundPerformanceTracker: FC<FundPerformanceTrackerProps> = ({
  fundId: _fundId = 'default',
  showDetails = true,
  compact = false,
}) => {
  // Mock data - in real implementation, this would come from props or API
  const performanceData = {
    tvpi: 1.85,
    irr: 15.6,
    benchmark: 12.3,
    status: 'performing',
    vintage: 2020,
    targetIrr: 18.0,
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'performing':
        return 'success';
      case 'underperforming':
        return 'warning';
      case 'at_risk':
        return 'error';
      default:
        return 'default';
    }
  };

  const getIrrProgress = () => {
    return (performanceData.irr / performanceData.targetIrr) * 100;
  };

  if (compact) {
    return (
      <Box display="flex" alignItems="center" gap={1}>
        <Typography variant="body2" color="text.secondary">
          TVPI:
        </Typography>
        <Typography variant="body2" fontWeight="bold">
          {performanceData.tvpi.toFixed(2)}x
        </Typography>
        <Chip
          label={`${performanceData.irr.toFixed(1)}% IRR`}
          size="small"
          color={getStatusColor(performanceData.status)}
          variant="outlined"
        />
      </Box>
    );
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="h6" gutterBottom>
          Fund Performance Tracker
        </Typography>

        <Grid container spacing={2}>
          <Grid item xs={12} sm={6}>
            <Box>
              <Typography variant="body2" color="text.secondary">
                TVPI
              </Typography>
              <Typography variant="h4" fontWeight="bold">
                {performanceData.tvpi.toFixed(2)}x
              </Typography>
            </Box>
          </Grid>

          <Grid item xs={12} sm={6}>
            <Box>
              <Typography variant="body2" color="text.secondary">
                IRR
              </Typography>
              <Box display="flex" alignItems="center" gap={1}>
                <Typography variant="h4" fontWeight="bold">
                  {performanceData.irr.toFixed(1)}%
                </Typography>
                {performanceData.irr > performanceData.benchmark ? (
                  <TrendingUpIcon color="success" />
                ) : (
                  <TrendingDownIcon color="error" />
                )}
              </Box>
            </Box>
          </Grid>

          {showDetails && (
            <>
              <Grid item xs={12}>
                <Box>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    IRR Progress vs Target ({performanceData.targetIrr.toFixed(1)}%)
                  </Typography>
                  <LinearProgress
                    variant="determinate"
                    value={Math.min(getIrrProgress(), 100)}
                    color={getIrrProgress() >= 100 ? 'success' : 'primary'}
                    sx={{ height: 8, borderRadius: 4 }}
                  />
                  <Typography variant="caption" color="text.secondary">
                    {getIrrProgress().toFixed(1)}% of target achieved
                  </Typography>
                </Box>
              </Grid>

              <Grid item xs={12}>
                <Box display="flex" justifyContent="space-between" alignItems="center">
                  <Box>
                    <Typography variant="body2" color="text.secondary">
                      Vintage Year
                    </Typography>
                    <Typography variant="body1">
                      {performanceData.vintage}
                    </Typography>
                  </Box>
                  <Chip
                    label={performanceData.status.replace('_', ' ').toUpperCase()}
                    color={getStatusColor(performanceData.status)}
                    size="small"
                  />
                </Box>
              </Grid>
            </>
          )}
        </Grid>
      </CardContent>
    </Card>
  );
};
