import type { FC } from 'react';
import { useParams, Link as RouterLink } from 'react-router-dom';
import { gql, useQuery } from '@apollo/client';
import { Box, Typography, Alert, CircularProgress, Breadcrumbs, Link } from '@mui/material';
import DriftReportDetail, { DriftReport } from '../components/DriftReportDetail';

/**
 * GraphQL query to fetch a single drift report by its primary key.
 * This aligns with the 'Detail Query' from the blueprint.
 */
const GET_DRIFT_REPORT_DETAIL = gql`
  query GetDriftReportDetail($id: uuid!) {
    drift_reports_by_pk(id: $id) {
      id
      generated_at
      schema_hash
      severity_summary
      changelog_md
      raw_report
      drift_log_entries(order_by: { severity: asc }) {
        id
        severity
        qualified_path
        explanation
        explain
      }
    }
  }
`;

const DriftReportDetailPage: FC = () => {
  const { reportId } = useParams<{ reportId: string }>();
  const { data, loading, error } = useQuery<{ drift_reports_by_pk: DriftReport }>(
    GET_DRIFT_REPORT_DETAIL,
    {
      variables: { id: reportId! },
      skip: !reportId,
    }
  );

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '80vh' }}>
        <CircularProgress />
        <Typography sx={{ ml: 2 }}>Loading Report Details...</Typography>
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ m: 2 }}>
        Failed to load report details: {error.message}
      </Alert>
    );
  }

  if (!data?.drift_reports_by_pk) {
    return (
      <Alert severity="warning" sx={{ m: 2 }}>
        Report not found. It may have been deleted.
      </Alert>
    );
  }

  return (
    <Box sx={{ flexGrow: 1, p: 3 }}>
      <Breadcrumbs aria-label="breadcrumb" sx={{ mb: 2 }}>
        <Link component={RouterLink} underline="hover" color="inherit" to="/fabric/reports">
          Drift Reports
        </Link>
        <Typography color="text.primary">Report {reportId?.substring(0, 8)}...</Typography>
      </Breadcrumbs>
      <DriftReportDetail report={data.drift_reports_by_pk} />
    </Box>
  );
};

export default DriftReportDetailPage;