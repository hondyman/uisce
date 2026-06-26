/**
 * ImpactReport - Displays simulation results with impact analysis
 * Part of the Uisce Visual Rule Builder
 */
import React from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Alert,
  LinearProgress,
  Divider,
  Chip,
  Table,
  TableBody,
  TableRow,
  TableCell,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material';
import {
  Warning as WarningIcon,
  CheckCircle as CheckCircleIcon,
  ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';

export interface ImpactReportData {
  total_records: number;
  affected_records: number;
  impact_percent: number;
  sample_matches: Record<string, any>[];
  time_range: string;
  warning?: string;
}

interface ImpactReportProps {
  data: ImpactReportData | null;
  loading?: boolean;
  error?: string | null;
}

export const ImpactReport: React.FC<ImpactReportProps> = ({
  data,
  loading,
  error,
}) => {
  if (loading) {
    return (
      <Card>
        <CardContent>
          <Typography variant="body2" gutterBottom>
            Running impact simulation against historical data...
          </Typography>
          <LinearProgress />
        </CardContent>
      </Card>
    );
  }

  if (error) {
    return (
      <Alert severity="error">
        <Typography variant="body2">{error}</Typography>
      </Alert>
    );
  }

  if (!data) {
    return null;
  }

  const isHighImpact = data.impact_percent > 10;
  const isMediumImpact = data.impact_percent > 5 && data.impact_percent <= 10;

  return (
    <Card variant="outlined">
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 2 }}>
          {isHighImpact ? (
            <WarningIcon color="error" sx={{ fontSize: 40 }} />
          ) : isMediumImpact ? (
            <WarningIcon color="warning" sx={{ fontSize: 40 }} />
          ) : (
            <CheckCircleIcon color="success" sx={{ fontSize: 40 }} />
          )}
          <Box sx={{ flex: 1 }}>
            <Typography variant="h6" gutterBottom>
              Simulation Results
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Based on {data.time_range} of historical data
            </Typography>
          </Box>
        </Box>

        {data.warning && (
          <Alert 
            severity={isHighImpact ? 'error' : isMediumImpact ? 'warning' : 'info'}
            sx={{ mb: 2 }}
          >
            {data.warning}
          </Alert>
        )}

        <Box sx={{ display: 'flex', gap: 4, mb: 3 }}>
          <Box sx={{ textAlign: 'center' }}>
            <Typography variant="h4" color="primary.main">
              {data.total_records.toLocaleString()}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Total Records
            </Typography>
          </Box>
          <Divider orientation="vertical" flexItem />
          <Box sx={{ textAlign: 'center' }}>
            <Typography variant="h4" color={isHighImpact ? 'error.main' : 'warning.main'}>
              {data.affected_records.toLocaleString()}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Would Be Affected
            </Typography>
          </Box>
          <Divider orientation="vertical" flexItem />
          <Box sx={{ textAlign: 'center' }}>
            <Typography variant="h4" color={isHighImpact ? 'error.main' : 'text.primary'}>
              {data.impact_percent.toFixed(2)}%
            </Typography>
            <Typography variant="caption" color="text.secondary">
              Impact Rate
            </Typography>
          </Box>
        </Box>

        <Box sx={{ mb: 2 }}>
          <Typography variant="caption" color="text.secondary" gutterBottom display="block">
            Impact Level
          </Typography>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <LinearProgress
              variant="determinate"
              value={Math.min(data.impact_percent, 100)}
              sx={{ flex: 1, height: 10, borderRadius: 5 }}
              color={isHighImpact ? 'error' : isMediumImpact ? 'warning' : 'success'}
            />
            <Chip
              size="small"
              label={isHighImpact ? 'High' : isMediumImpact ? 'Medium' : 'Low'}
              color={isHighImpact ? 'error' : isMediumImpact ? 'warning' : 'success'}
            />
          </Box>
        </Box>

        {data.sample_matches && data.sample_matches.length > 0 && (
          <Accordion>
            <AccordionSummary expandIcon={<ExpandMoreIcon />}>
              <Typography variant="subtitle2">
                Sample Matches ({data.sample_matches.length})
              </Typography>
            </AccordionSummary>
            <AccordionDetails>
              <Table size="small">
                <TableBody>
                  {data.sample_matches.slice(0, 5).map((match, index) => (
                    <TableRow key={index}>
                      {Object.entries(match).slice(0, 4).map(([key, value]) => (
                        <TableCell key={key}>
                          <Typography variant="caption" color="text.secondary">
                            {key}:
                          </Typography>{' '}
                          <Typography variant="body2" component="span">
                            {String(value)}
                          </Typography>
                        </TableCell>
                      ))}
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </AccordionDetails>
          </Accordion>
        )}
      </CardContent>
    </Card>
  );
};

export default ImpactReport;
