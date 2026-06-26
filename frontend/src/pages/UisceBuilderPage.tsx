/**
 * UisceBuilderPage - Visual Rule Builder Page
 * No-code compliance rule creation with impact simulation
 * 
 * Business owners see logic and impact, never code.
 */
import React, { useState, useCallback } from 'react';
import {
  Box,
  Container,
  Typography,
  Paper,
  Tabs,
  Tab,
  Alert,
  Snackbar,
  Breadcrumbs,
  Link,
} from '@mui/material';
import {
  Build as BuildIcon,
  History as HistoryIcon,
  Settings as SettingsIcon,
} from '@mui/icons-material';
import { useTenant } from '../contexts/TenantContext';
import { UisceRuleBuilder, UIRule } from '../components/uisce';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index }) => (
  <Box role="tabpanel" hidden={value !== index} sx={{ py: 3 }}>
    {value === index && children}
  </Box>
);

export const UisceBuilderPage: React.FC = () => {
  const { tenant, datasource } = useTenant();
  const [activeTab, setActiveTab] = useState(0);
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  });

  const handlePublish = useCallback(async (rule: UIRule, cueCode: string) => {
    try {
      // Call backend to persist the rule
      const response = await fetch(
        `/api/validation-rules?tenant_id=${tenant?.id}&tenant_instance_id=${datasource?.id}`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            name: rule.name,
            description: rule.description,
            rule_type: 'cue',
            script_content: cueCode,
            severity: rule.severity || 'error',
            is_active: true,
            ui_definition: JSON.stringify(rule),
          }),
        }
      );

      if (!response.ok) {
        const error = await response.json().catch(() => ({}));
        throw new Error(error.error || 'Failed to publish rule');
      }

      setSnackbar({
        open: true,
        message: `Rule "${rule.name}" published successfully!`,
        severity: 'success',
      });
    } catch (error) {
      const err = error instanceof Error ? error.message : 'Unknown error';
      setSnackbar({
        open: true,
        message: `Failed to publish: ${err}`,
        severity: 'error',
      });
    }
  }, [tenant?.id, datasource?.id]);

  if (!tenant || !datasource) {
    return (
      <Container maxWidth="lg" sx={{ py: 4 }}>
        <Alert severity="warning">
          Please select a tenant and datasource to use the Uisce Builder.
        </Alert>
      </Container>
    );
  }

  return (
    <Container maxWidth="lg" sx={{ py: 3 }}>
      {/* Breadcrumbs */}
      <Breadcrumbs sx={{ mb: 2 }}>
        <Link underline="hover" color="inherit" href="/">
          Home
        </Link>
        <Link underline="hover" color="inherit" href="/core/validation-rules">
          Validation Rules
        </Link>
        <Typography color="text.primary">Visual Builder</Typography>
      </Breadcrumbs>

      {/* Header */}
      <Paper sx={{ mb: 3, p: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <BuildIcon sx={{ fontSize: 40, color: 'primary.main' }} />
          <Box>
            <Typography variant="h4" fontWeight="bold">
              Uisce Compliance Builder
            </Typography>
            <Typography variant="body2" color="text.secondary">
              Create validation rules visually • See impact before you publish • No code required
            </Typography>
          </Box>
        </Box>
      </Paper>

      {/* Tabs */}
      <Paper sx={{ mb: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(_, newValue) => setActiveTab(newValue)}
          sx={{ borderBottom: 1, borderColor: 'divider' }}
        >
          <Tab icon={<BuildIcon />} label="Build Rule" iconPosition="start" />
          <Tab icon={<HistoryIcon />} label="Published Rules" iconPosition="start" />
          <Tab icon={<SettingsIcon />} label="Settings" iconPosition="start" disabled />
        </Tabs>

        <TabPanel value={activeTab} index={0}>
          <UisceRuleBuilder
            tenantId={tenant.id}
            datasourceId={datasource.id}
            onPublish={handlePublish}
          />
        </TabPanel>

        <TabPanel value={activeTab} index={1}>
          <Box sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              Published Rules
            </Typography>
            <Typography color="text.secondary">
              View and manage rules created with the visual builder.
            </Typography>
            <Alert severity="info" sx={{ mt: 2 }}>
              Switch to the{' '}
              <Link href="/core/validation-rules">Validation Rules</Link> page to manage all rules.
            </Alert>
          </Box>
        </TabPanel>

        <TabPanel value={activeTab} index={2}>
          <Box sx={{ p: 3 }}>
            <Typography>Settings coming soon...</Typography>
          </Box>
        </TabPanel>
      </Paper>

      {/* Snackbar */}
      <Snackbar
        open={snackbar.open}
        autoHideDuration={5000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
      >
        <Alert
          onClose={() => setSnackbar({ ...snackbar, open: false })}
          severity={snackbar.severity}
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Container>
  );
};

export default UisceBuilderPage;
