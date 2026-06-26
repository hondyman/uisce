import React, { useState } from 'react';
import {
  Box,
  Typography,
  Alert,
  Paper,
  CircularProgress,
  TableContainer,
  FormControlLabel,
  Switch,
} from '@mui/material';
import { gql, useQuery } from '@apollo/client';
import useBlockableNavigate from '../../../components/RouteBlocker/useBlockableNavigate';
import DriftReportTable from '../components/DriftReportTable';

/**
 * GraphQL query to fetch a list of drift reports.
 * This aligns with the 'Dashboard Query' from the blueprint.
 */
const GET_DRIFT_REPORTS = gql`
  query GetDriftReports($limit: Int!, $offset: Int!, $where: drift_reports_with_severity_flag_bool_exp) {
    drift_reports_with_severity_flag(
      limit: $limit, 
      offset: $offset, 
      order_by: { generated_at: desc },
      where: $where
    ) {
      id
      generated_at
      schema_hash
      severity_summary
      has_severity_change
    }
  }
`;

/**
 * TypeScript interface for a single drift report summary.
 * This matches the data structure returned by the GetDriftReports query.
 */
interface DriftReportSummary {
  id: string;
  generated_at: string; // ISO timestamp
  schema_hash: string;
  severity_summary: {
    breaking?: number;
    medium?: number;
    low?: number;
  };
  has_severity_change: boolean;
}

/**
 * DriftReportDashboard Component
 * Purpose: Renders the main dashboard for viewing all drift reports.
 * Implements the 'DriftDashboardPage' from the component tree blueprint.
 */
const DriftReportDashboard: React.FC = () => {
  const navigate = useBlockableNavigate();
  const [onlySeverityChanges, setOnlySeverityChanges] = useState(false);

  const { loading, error, data } = useQuery<{ drift_reports_with_severity_flag: DriftReportSummary[] }>(GET_DRIFT_REPORTS, {
    variables: {
      limit: 50,
      offset: 0,
      where: onlySeverityChanges ? { has_severity_change: { _eq: true } } : {},
    },
    fetchPolicy: 'network-only',
  });

  const handleRowClick = (reportId: string) => {
    void navigate(`/fabric/reports/${reportId}`);
  };

  return (
    <Box sx={{ flexGrow: 1, p: 3, display: 'flex', flexDirection: 'column' }}>
      {/* PageHeader */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="h4" gutterBottom>
          Drift Reports
        </Typography>
        <Typography variant="body2" color="text.secondary">
          A historical log of all detected schema drift events. Click a report to see details.
        </Typography>
      </Box>

      {/* FilterBar (Placeholder) */}
      <Paper sx={{ p: 2, mb: 3, display: 'flex', gap: 2 }}>
        <FormControlLabel
          control={
            <Switch
              checked={onlySeverityChanges}
              onChange={(e) => setOnlySeverityChanges(e.target.checked)}
            />
          }
          label="Show only reports with severity changes"
        />
      </Paper>

      {/* DriftReportTable */}
      <Paper sx={{ flexGrow: 1, overflow: 'hidden' }}>
        <TableContainer sx={{ height: '100%' }}>
          {loading && (
            <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100%' }}>
              <CircularProgress />
              <Typography sx={{ ml: 2 }}>Loading Reports...</Typography>
            </Box>
          )}
          {error && <Alert severity="error" sx={{ m: 2 }}>Failed to load drift reports: {error.message}</Alert>}
          {!loading && !error && data && (
            <DriftReportTable reports={data.drift_reports_with_severity_flag} onSelectReport={handleRowClick} />
          )}
        </TableContainer>
      </Paper>
    </Box>
  );
};

export default DriftReportDashboard;