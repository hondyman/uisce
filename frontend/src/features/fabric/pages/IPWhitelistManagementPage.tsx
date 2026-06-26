import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Tabs,
  Tab,
  Typography,
  Container,
  Button,
  Stack,
} from '@mui/material';
import {
  Computer as ComputerIcon,
  Business as BusinessIcon,
  Add as AddIcon,
} from '@mui/icons-material';
import { useIPWhitelistAPI } from '../hooks/useIPWhitelist';
import IPManagementView from '../components/IPManagementView';
import TenantManagementView from '../components/TenantManagementView';
import { useNotification } from '../../../hooks/useNotification';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

const TabPanel: React.FC<TabPanelProps> = ({ children, value, index, ...other }) => {
  return (
    <div
      role="tabpanel"
      hidden={value !== index}
      id={`tabpanel-${index}`}
      aria-labelledby={`tab-${index}`}
      {...other}
    >
      {value === index && <Box sx={{ py: 3 }}>{children}</Box>}
    </div>
  );
};

const IPWhitelistManagementPage: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const api = useIPWhitelistAPI();
  const notification = useNotification();

  const handleTabChange = (_: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <Container maxWidth="xl" sx={{ py: 3 }}>
      {/* Page Header */}
      <Box sx={{ mb: 4 }}>
        <Stack spacing={1}>
          <Typography variant="h4" component="h1" sx={{ fontWeight: 900 }}>
            IP Whitelist Management
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Configure global allow-lists and assign dedicated IP ranges to specific tenants to ensure secure access control.
          </Typography>
        </Stack>
      </Box>

      <Paper sx={{ width: '100%' }}>
        <Box sx={{ borderBottom: 1, borderColor: 'divider' }}>
          <Tabs
            value={activeTab}
            onChange={handleTabChange}
            aria-label="IP whitelist management tabs"
            variant="fullWidth"
          >
            <Tab
              icon={<ComputerIcon />}
              label="IP Address View"
              id="tab-0"
              aria-controls="tabpanel-0"
              iconPosition="start"
            />
            <Tab
              icon={<BusinessIcon />}
              label="Tenant View"
              id="tab-1"
              aria-controls="tabpanel-1"
              iconPosition="start"
            />
          </Tabs>
        </Box>

        <TabPanel value={activeTab} index={0}>
          <IPManagementView />
        </TabPanel>

        <TabPanel value={activeTab} index={1}>
          <TenantManagementView />
        </TabPanel>
      </Paper>
    </Container>
  );
};

export default IPWhitelistManagementPage;
