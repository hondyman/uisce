import type { FC } from 'react';
import { gql, useQuery } from '@apollo/client';
import {
  Box,
  Typography,
  Alert,
  CircularProgress,
  Paper,
  TableContainer,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  Button,
  Chip,
} from '@mui/material';
import { Link as RouterLink } from 'react-router-dom';
import { format } from 'date-fns';

const GET_LATEST_EVALUATIONS = gql`
  query GetLatestEvaluations {
    drift_reports(order_by: { generated_at: desc }, limit: 10) {
      id
      generated_at
      severity_summary
    }
  }
`;

const getDecisionFromSummary = (summary: any) => {
  if (summary?.breaking > 0) {
    return { label: 'BLOCK', color: 'error' as const };
  }
  if (summary?.medium > 0) {
    return { label: 'WARN', color: 'warning' as const };
  }
  return { label: 'ALLOW', color: 'success' as const };
};

const DashboardPage: FC = () => {
  const { data, loading, error } = useQuery(GET_LATEST_EVALUATIONS);

  return (
    <Box sx={{ p: 3 }}>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4">Policy Dashboard</Typography>
        <Button component={RouterLink} to="/fabric/simulate" variant="contained">
          Go to Simulation Lab
        </Button>
      </Box>

      <Typography paragraph color="text.secondary">
        An overview of the most recent policy evaluations.
      </Typography>

      <Paper>
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Run ID</TableCell>
                <TableCell>Evaluated At</TableCell>
                <TableCell>Decision</TableCell>
                <TableCell>Violations (B/M/L)</TableCell>
                <TableCell>Actions</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {loading && (
                <TableRow>
                  <TableCell colSpan={5} align="center">
                    <CircularProgress />
                  </TableCell>
                </TableRow>
              )}
              {error && (
                <TableRow>
                  <TableCell colSpan={5}>
                    <Alert severity="error">Failed to load evaluations: {error.message}</Alert>
                  </TableCell>
                </TableRow>
              )}
              {data?.drift_reports.map((report: any) => {
                const decision = getDecisionFromSummary(report.severity_summary);
                const summary = report.severity_summary || {};
                return (
                  <TableRow key={report.id}>
                    <TableCell sx={{ fontFamily: 'monospace' }}>{report.id.substring(0, 8)}...</TableCell>
                    <TableCell>{format(new Date(report.generated_at), 'yyyy-MM-dd HH:mm')}</TableCell>
                    <TableCell>
                      <Chip label={decision.label} color={decision.color} size="small" />
                    </TableCell>
                    <TableCell>
                      {summary.breaking || 0} / {summary.medium || 0} / {summary.low || 0}
                    </TableCell>
                    <TableCell>
                      <Button component={RouterLink} to={`/fabric/reports/${report.id}`} size="small">
                        View Report
                      </Button>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      </Paper>
    </Box>
  );
};

export default DashboardPage;