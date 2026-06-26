import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Container,
  Paper,
  Tabs,
  Tab,
  Typography,
  Button,
  Grid,
  Card,
  CardContent,
  Chip as _Chip,
  CircularProgress,
  Alert,
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import AddIcon from '@mui/icons-material/Add';
import RefreshIcon from '@mui/icons-material/Refresh';
import ValidationRuleEditor from './ValidationRuleEditor';
import ValidationResultsPanel from './ValidationResultsPanel';
import RealTimeValidationPanel from './RealTimeValidationPanel';
import ValidationHistoryPanel from './ValidationHistoryPanel';
import ValidationRulesList from './ValidationRulesList';

const useStyles = makeStyles({
  unitText: {
    fontSize: '0.6em',
  },
});

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`validation-tabpanel-${index}`}
      aria-labelledby={`validation-tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ p: 3 }}>{children}</Box>}
    </div>
  );
}

interface ValidationStats {
  totalRules: number;
  enabledRules: number;
  recentValidations: number;
  successRate: number;
  averageExecutionTime: number;
}

interface ApiError {
  message: string;
  code: string;
}

const ValidationDashboard: React.FC = () => {
  const classes = useStyles();
  const [tabValue, setTabValue] = useState(0);
  const [stats, setStats] = useState<ValidationStats | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  const fetchStats = useCallback(async () => {
    try {
      setRefreshing(true);
      const tenantId = localStorage.getItem('selected_tenant')
        ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id
        : null;
      const datasourceId = localStorage.getItem('selected_datasource')
        ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id
        : null;

      if (!tenantId || !datasourceId) {
        setError({
          message: 'Please select a tenant and datasource first',
          code: 'SCOPE_REQUIRED',
        });
        return;
      }

      const response = await fetch(
        `/api/validations/metrics?tenant_id=${tenantId}&tenant_instance_id=${datasourceId}`,
        {
          headers: {
            'X-Tenant-ID': tenantId,
            'X-Tenant-Datasource-ID': datasourceId,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch stats: ${response.statusText}`);
      }

      const data = await response.json();
      setStats(data);
      setError(null);
    } catch (err) {
      setError({
        message: err instanceof Error ? err.message : 'Failed to fetch statistics',
        code: 'FETCH_ERROR',
      });
    } finally {
      setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    fetchStats().finally(() => setLoading(false));
  }, [fetchStats]);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setTabValue(newValue);
  };

  const StatCard: React.FC<{ label: string; value: string | number; unit?: string }> = ({
    label,
    value,
    unit,
  }) => (
    <Card sx={{ minHeight: 120, display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
      <CardContent>
        <Typography color="textSecondary" gutterBottom>
          {label}
        </Typography>
        <Typography variant="h4">
          {value}
          {unit && <span className={classes.unitText}>{unit}</span>}
        </Typography>
      </CardContent>
    </Card>
  );

  return (
    <Container maxWidth="lg" sx={{ py: 4 }}>
      <Box sx={{ mb: 4 }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
          <Typography variant="h4" component="h1">
            Validation Dashboard
          </Typography>
          <Box sx={{ display: 'flex', gap: 2 }}>
            <Button
              variant="outlined"
              startIcon={<RefreshIcon />}
              onClick={fetchStats}
              disabled={refreshing}
            >
              Refresh
            </Button>
            <Button variant="contained" startIcon={<AddIcon />}>
              Create Rule
            </Button>
          </Box>
        </Box>

        {error && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {error.message}
          </Alert>
        )}

        {loading ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
            <CircularProgress />
          </Box>
        ) : stats ? (
          <Grid container spacing={2} sx={{ mb: 4 }}>
            <Grid item xs={12} sm={6} md={3}>
              <StatCard label="Total Rules" value={stats.totalRules} />
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <StatCard label="Enabled Rules" value={stats.enabledRules} />
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <StatCard label="Recent Validations" value={stats.recentValidations} />
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <StatCard label="Success Rate" value={stats.successRate.toFixed(1)} unit="%" />
            </Grid>
          </Grid>
        ) : null}
      </Box>

      <Paper>
        <Tabs
          value={tabValue}
          onChange={handleTabChange}
          aria-label="validation dashboard tabs"
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab label="Real-Time Validation" id="validation-tab-0" aria-controls="validation-tabpanel-0" />
          <Tab label="Rule Editor" id="validation-tab-1" aria-controls="validation-tabpanel-1" />
          <Tab label="Rules Library" id="validation-tab-2" aria-controls="validation-tabpanel-2" />
          <Tab label="Results" id="validation-tab-3" aria-controls="validation-tabpanel-3" />
          <Tab label="History" id="validation-tab-4" aria-controls="validation-tabpanel-4" />
        </Tabs>

        <TabPanel value={tabValue} index={0}>
          <RealTimeValidationPanel />
        </TabPanel>

        <TabPanel value={tabValue} index={1}>
          <ValidationRuleEditor />
        </TabPanel>

        <TabPanel value={tabValue} index={2}>
          <ValidationRulesList />
        </TabPanel>

        <TabPanel value={tabValue} index={3}>
          <ValidationResultsPanel />
        </TabPanel>

        <TabPanel value={tabValue} index={4}>
          <ValidationHistoryPanel />
        </TabPanel>
      </Paper>
    </Container>
  );
};

export default ValidationDashboard;
