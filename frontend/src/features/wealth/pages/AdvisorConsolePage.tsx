import React, { useState } from 'react';
import { 
  Box, 
  Typography, 
  Paper, 
  List, 
  ListItem, 
  ListItemText, 
  ListItemSecondaryAction, 
  IconButton, 
  Button, 
  Chip, 
  Divider,
  Grid,
  Card,
  CardContent,
  CardActions
} from '@mui/material';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import CancelIcon from '@mui/icons-material/Cancel';
import InfoIcon from '@mui/icons-material/Info';
import { devDebug } from '../../../utils/devLogger';

// Mock Data Types
interface AdvisorQueueItem {
  id: string;
  clientId: string;
  clientName: string;
  type: 'RECOMMENDATION' | 'ALERT';
  priority: 'HIGH' | 'MEDIUM' | 'LOW';
  title: string;
  description: string;
  status: 'OPEN' | 'RESOLVED' | 'REJECTED';
  generatedAt: string;
  context: {
    riskScore: number;
    lastContact: string;
    holdings: string[];
  };
}

// Mock Data
const MOCK_QUEUE: AdvisorQueueItem[] = [
  {
    id: '1',
    clientId: 'c_123',
    clientName: 'Alice Johnson',
    type: 'RECOMMENDATION',
    priority: 'HIGH',
    title: 'Tax-Loss Harvesting Opportunity',
    description: 'Sell TSLA to realize $5k loss. Wash sale risk checked.',
    status: 'OPEN',
    generatedAt: '2025-11-23T09:00:00Z',
    context: {
      riskScore: 75,
      lastContact: '2025-11-10',
      holdings: ['TSLA', 'AAPL', 'VTI']
    }
  },
  {
    id: '2',
    clientId: 'c_456',
    clientName: 'Bob Smith',
    type: 'ALERT',
    priority: 'MEDIUM',
    title: 'Portfolio Drift Alert',
    description: 'Equity allocation is 5% above target due to recent rally.',
    status: 'OPEN',
    generatedAt: '2025-11-23T10:30:00Z',
    context: {
      riskScore: 45,
      lastContact: '2025-10-15',
      holdings: ['SPY', 'BND', 'GLD']
    }
  }
];

export const AdvisorConsolePage: React.FC = () => {
  const [queue, setQueue] = useState<AdvisorQueueItem[]>(MOCK_QUEUE);
  const [selectedItem, setSelectedItem] = useState<AdvisorQueueItem | null>(null);

  const handleApprove = (id: string) => {
    devDebug(`Approved item ${id}`);
    setQueue(queue.filter(item => item.id !== id));
    if (selectedItem?.id === id) setSelectedItem(null);
  };

  const handleReject = (id: string) => {
    devDebug(`Rejected item ${id}`);
    setQueue(queue.filter(item => item.id !== id));
    if (selectedItem?.id === id) setSelectedItem(null);
  };

  return (
    <Box sx={{ p: 3, height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <Typography variant="h4" gutterBottom>
        Advisor Console
      </Typography>
      <Typography variant="subtitle1" color="textSecondary" gutterBottom>
        Review and approve AI-generated recommendations requiring human oversight.
      </Typography>

      <Grid container spacing={3} sx={{ flexGrow: 1, mt: 1 }}>
        {/* Queue List */}
        <Grid item xs={12} md={4}>
          <Paper sx={{ height: '100%', overflow: 'auto' }}>
            <List>
              {queue.map((item) => (
                <React.Fragment key={item.id}>
                  <ListItem 
                    button 
                    selected={selectedItem?.id === item.id}
                    onClick={() => setSelectedItem(item)}
                  >
                    <ListItemText
                      primary={
                        <Box display="flex" justifyContent="space-between" alignItems="center">
                          <Typography variant="subtitle1">{item.clientName}</Typography>
                          <Chip 
                            label={item.priority} 
                            size="small" 
                            color={item.priority === 'HIGH' ? 'error' : 'warning'} 
                          />
                        </Box>
                      }
                      secondary={
                        <>
                          <Typography variant="body2" color="textPrimary">
                            {item.title}
                          </Typography>
                          <Typography variant="caption" color="textSecondary">
                            {new Date(item.generatedAt).toLocaleTimeString()}
                          </Typography>
                        </>
                      }
                    />
                  </ListItem>
                  <Divider />
                </React.Fragment>
              ))}
              {queue.length === 0 && (
                <Box p={3} textAlign="center">
                  <Typography color="textSecondary">No items in queue.</Typography>
                </Box>
              )}
            </List>
          </Paper>
        </Grid>

        {/* Detail View */}
        <Grid item xs={12} md={8}>
          {selectedItem ? (
            <Paper sx={{ p: 3, height: '100%' }}>
              <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
                <Typography variant="h5">{selectedItem.title}</Typography>
                <Chip label={selectedItem.type} color="primary" variant="outlined" />
              </Box>
              
              <Typography variant="body1" paragraph>
                {selectedItem.description}
              </Typography>

              <Divider sx={{ my: 2 }} />

              <Typography variant="h6" gutterBottom>Client Context</Typography>
              <Grid container spacing={2}>
                <Grid item xs={4}>
                  <Card variant="outlined">
                    <CardContent>
                      <Typography color="textSecondary" gutterBottom>Risk Score</Typography>
                      <Typography variant="h4">{selectedItem.context.riskScore}</Typography>
                    </CardContent>
                  </Card>
                </Grid>
                <Grid item xs={4}>
                  <Card variant="outlined">
                    <CardContent>
                      <Typography color="textSecondary" gutterBottom>Last Contact</Typography>
                      <Typography variant="h6">{selectedItem.context.lastContact}</Typography>
                    </CardContent>
                  </Card>
                </Grid>
                <Grid item xs={4}>
                  <Card variant="outlined">
                    <CardContent>
                      <Typography color="textSecondary" gutterBottom>Holdings</Typography>
                      <Typography variant="body2">{selectedItem.context.holdings.join(', ')}</Typography>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>

              <Box mt={4} display="flex" gap={2} justifyContent="flex-end">
                <Button 
                  variant="outlined" 
                  color="error" 
                  startIcon={<CancelIcon />}
                  onClick={() => handleReject(selectedItem.id)}
                >
                  Reject
                </Button>
                <Button 
                  variant="contained" 
                  color="success" 
                  startIcon={<CheckCircleIcon />}
                  onClick={() => handleApprove(selectedItem.id)}
                >
                  Approve & Execute
                </Button>
              </Box>
            </Paper>
          ) : (
            <Paper sx={{ p: 3, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              <Typography color="textSecondary">Select an item to review details.</Typography>
            </Paper>
          )}
        </Grid>
      </Grid>
    </Box>
  );
};

export default AdvisorConsolePage;