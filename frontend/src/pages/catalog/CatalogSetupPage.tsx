import React, { useState } from 'react';
import { Box, Tabs, Tab, Paper, Typography, Container } from '@mui/material';
import { NodeTypeSetupPage } from '../nodes/NodeTypeSetupPage';
import { EdgeTypeSetupPage } from '../edges/EdgeTypeSetupPage';
import AccountTreeIcon from '@mui/icons-material/AccountTree';
import DeviceHubIcon from '@mui/icons-material/DeviceHub';

interface TabPanelProps {
  children?: React.ReactNode;
  index: number;
  value: number;
}

function TabPanel(props: TabPanelProps) {
  const { children, value, index, ...other } = props;

  return (
    <Box
      role="tabpanel"
      hidden={value !== index}
      id={`catalog-tabpanel-${index}`}
      aria-labelledby={`catalog-tab-${index}`}
      sx={{ height: '100%' }}
      {...other}
    >
      {value === index && <Box sx={{ height: '100%' }}>{children}</Box>}
    </Box>
  );
}

export const CatalogSetupPage: React.FC = () => {
  const [currentTab, setCurrentTab] = useState(0);

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setCurrentTab(newValue);
  };

  return (
    <Box sx={{ 
      display: 'flex', 
      flexDirection: 'column', 
      height: '100%',
      bgcolor: 'background.default'
    }}>
      {/* Header Section */}
      <Box sx={{ 
        bgcolor: 'background.paper', 
        borderBottom: 1, 
        borderColor: 'divider',
        boxShadow: 1
      }}>
        <Container maxWidth="xl" sx={{ py: 3 }}>
          <Typography variant="h4" component="h1" gutterBottom sx={{ fontWeight: 600 }}>
            Catalog Setup
          </Typography>
          <Typography variant="body1" color="text.secondary">
            Configure node and edge types for your business glossary and semantic layer
          </Typography>
        </Container>
      </Box>

      {/* Tabs Navigation */}
      <Paper 
        elevation={0} 
        sx={{ 
          borderBottom: 1, 
          borderColor: 'divider',
          bgcolor: 'background.paper'
        }}
      >
        <Container maxWidth="xl">
          <Tabs 
            value={currentTab} 
            onChange={handleTabChange} 
            aria-label="catalog setup tabs"
            sx={{
              '& .MuiTab-root': {
                minHeight: 64,
                textTransform: 'none',
                fontSize: '1rem',
                fontWeight: 500,
              }
            }}
          >
            <Tab 
              icon={<AccountTreeIcon />} 
              iconPosition="start"
              label="Node Types" 
              id="catalog-tab-0" 
              aria-controls="catalog-tabpanel-0"
            />
            <Tab 
              icon={<DeviceHubIcon />} 
              iconPosition="start"
              label="Edge Types" 
              id="catalog-tab-1" 
              aria-controls="catalog-tabpanel-1"
            />
          </Tabs>
        </Container>
      </Paper>

      {/* Content Area */}
      <Box sx={{ flexGrow: 1, overflow: 'auto', bgcolor: 'background.default' }}>
        <TabPanel value={currentTab} index={0}>
          <NodeTypeSetupPage />
        </TabPanel>
        <TabPanel value={currentTab} index={1}>
          <EdgeTypeSetupPage />
        </TabPanel>
      </Box>
    </Box>
  );
};

