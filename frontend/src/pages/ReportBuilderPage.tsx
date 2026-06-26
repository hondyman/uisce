import React from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { Box, IconButton, Typography, Tooltip } from '@mui/material';
import { ArrowBack as BackIcon } from '@mui/icons-material';
import SSRSReportBuilder from '../components/reporting/SSRSReportBuilder';

/**
 * ReportBuilderPage - Wrapper page for the SSRSReportBuilder
 * 
 * Routes:
 * - /reports/builder - Create new report
 * - /reports/:reportId/edit - Edit existing report
 */
export const ReportBuilderPage: React.FC = () => {
  const { reportId } = useParams<{ reportId?: string }>();
  const navigate = useNavigate();
  const isEditMode = Boolean(reportId);

  const handleBack = () => {
    navigate('/reports/library');
  };

  return (
    <Box sx={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      {/* Header with back navigation */}
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 2,
          px: 2,
          py: 1,
          borderBottom: 1,
          borderColor: 'divider',
          bgcolor: 'background.paper',
        }}
      >
        <Tooltip title="Back to Report Library">
          <IconButton onClick={handleBack} size="small">
            <BackIcon />
          </IconButton>
        </Tooltip>
        <Typography variant="h6" component="h1">
          {isEditMode ? 'Edit Report' : 'Create New Report'}
        </Typography>
        {isEditMode && (
          <Typography variant="body2" color="text.secondary">
            Report ID: {reportId}
          </Typography>
        )}
      </Box>

      {/* Report Builder */}
      <Box sx={{ flex: 1, overflow: 'hidden' }}>
        <SSRSReportBuilder />
      </Box>
    </Box>
  );
};

export default ReportBuilderPage;
