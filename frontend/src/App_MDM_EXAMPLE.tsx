import React, { useState } from 'react';
import { Box, Button, Stack, Container } from '@mui/material';
import { createTheme, ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';

// Import all dashboard components
import { SemanticRuleBuilderDashboard } from './components/MDM_MUI_Dashboard';
import { RuleImpactComparison } from './components/MDM_MUI_RuleComparison';
import { ImpactAnalysisDashboard } from './components/MDM_MUI_ImpactAnalysis';
import { RealtimeNotificationsDashboard } from './components/MDM_MUI_RealtimeNotifications';

// Create custom theme for Material-UI v7
const theme = createTheme({
  palette: {
    primary: {
      main: '#137fec',
    },
    background: {
      default: '#f6f7f8',
    },
    success: {
      main: '#10b981',
    },
    warning: {
      main: '#f59e0b',
    },
    error: {
      main: '#ef4444',
    },
  },
  typography: {
    fontFamily: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  },
});

type PageType = 'dashboard' | 'comparison' | 'impact' | 'realtime';

export default function App() {
  const [currentPage, setCurrentPage] = useState<PageType>('dashboard');

  const pages: Record<PageType, React.ReactNode> = {
    dashboard: <SemanticRuleBuilderDashboard />,
    comparison: <RuleImpactComparison />,
    impact: <ImpactAnalysisDashboard />,
    realtime: <RealtimeNotificationsDashboard />,
  };

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <Box sx={{ minHeight: '100vh' }}>
        {/* Navigation Bar */}
        <Box sx={{ display: 'flex', gap: 1, p: 2, bgcolor: '#f3f4f6', borderBottom: '1px solid #e5e7eb' }}>
          <Button
            variant={currentPage === 'dashboard' ? 'contained' : 'outlined'}
            onClick={() => setCurrentPage('dashboard')}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            Rule Builder
          </Button>
          <Button
            variant={currentPage === 'comparison' ? 'contained' : 'outlined'}
            onClick={() => setCurrentPage('comparison')}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            Comparison
          </Button>
          <Button
            variant={currentPage === 'impact' ? 'contained' : 'outlined'}
            onClick={() => setCurrentPage('impact')}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            Impact
          </Button>
          <Button
            variant={currentPage === 'realtime' ? 'contained' : 'outlined'}
            onClick={() => setCurrentPage('realtime')}
            sx={{ textTransform: 'none', fontWeight: 600 }}
          >
            Real-time
          </Button>
        </Box>

        {/* Render current page */}
        {pages[currentPage]}
      </Box>
    </ThemeProvider>
  );
}
