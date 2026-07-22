import type { FC } from 'react';
import { useParams, Link as RouterLink } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Box, Typography, Alert, CircularProgress, Breadcrumbs, Link } from '@mui/material';
import DriftReportDetail, { DriftReport } from '../components/DriftReportDetail';
import { apiFetch } from '../../../lib/apiClient';

const DriftReportDetailPage: FC = () => {
  const { reportId } = useParams<{ reportId: string }>();

  const { data, isLoading, error } = useQuery({
    queryKey: ['drift-report', reportId],
    queryFn: async () => {
      const res = await apiFetch(`/api/rest/drift-reports/${reportId}`);
      if (!res.ok) throw new Error(await res.text());
      return res.json() as Promise<DriftReport>;
    },
    enabled: !!reportId,
  });

  if (isLoading) {
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
        Failed to load report details: {error?.message}
      </Alert>
    );
  }

  if (!data) {
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
      <DriftReportDetail report={data} />
    </Box>
  );
};

export default DriftReportDetailPage;