import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip
} from '@mui/material';
import EditIcon from '@mui/icons-material/Edit';
import AssessmentIcon from '@mui/icons-material/Assessment';
import { useNavigate } from 'react-router-dom';
import ImpactChart from '../../components/investment/ImpactChart';

// Mock Data
const MOCK_CLIENTS = [
  { id: 'C001', name: 'Alice Johnson', status: 'On Track', lastRebalance: '2023-10-25' },
  { id: 'C002', name: 'Bob Smith', status: 'Drifted', lastRebalance: '2023-09-15' },
];

const MOCK_PORTFOLIO_WEIGHTS = [
  { sector: 'Technology', weight: 0.35 },
  { sector: 'Healthcare', weight: 0.15 },
  { sector: 'Energy', weight: 0.00 }, // Excluded
  { sector: 'Financials', weight: 0.10 },
  { sector: 'Consumer', weight: 0.20 },
];

const MOCK_BENCHMARK_WEIGHTS = [
  { sector: 'Technology', weight: 0.28 },
  { sector: 'Healthcare', weight: 0.13 },
  { sector: 'Energy', weight: 0.05 },
  { sector: 'Financials', weight: 0.12 },
  { sector: 'Consumer', weight: 0.15 },
];

const DirectIndexingPage: React.FC = () => {
  const navigate = useNavigate();
  const [selectedClient, setSelectedClient] = useState<string | null>(null);

  const handleEditValues = (clientId: string) => {
    navigate(`/investment/direct-indexing/${clientId}/values`);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Direct Indexing Console
      </Typography>

      <Grid container spacing={3}>
        {/* Client List */}
        <Grid item xs={12} md={6}>
          <Paper elevation={3}>
            <Box p={2}>
              <Typography variant="h6">Managed Accounts</Typography>
            </Box>
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Client Name</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Last Rebalance</TableCell>
                    <TableCell>Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {MOCK_CLIENTS.map((client) => (
                    <TableRow
                      key={client.id}
                      hover
                      selected={selectedClient === client.id}
                      onClick={() => setSelectedClient(client.id)}
                      sx={{ cursor: 'pointer' }}
                    >
                      <TableCell>{client.name}</TableCell>
                      <TableCell>
                        <Chip
                          label={client.status}
                          color={client.status === 'On Track' ? 'success' : 'warning'}
                          size="small"
                        />
                      </TableCell>
                      <TableCell>{client.lastRebalance}</TableCell>
                      <TableCell>
                        <Button
                          size="small"
                          startIcon={<EditIcon />}
                          onClick={(e) => {
                            e.stopPropagation();
                            handleEditValues(client.id);
                          }}
                        >
                          Values
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          </Paper>
        </Grid>

        {/* Impact Analysis */}
        <Grid item xs={12} md={6}>
          {selectedClient ? (
            <Box>
              <Paper elevation={3} sx={{ p: 2, mb: 3 }}>
                <Box display="flex" justifyContent="space-between" alignItems="center">
                  <Typography variant="h6">Portfolio Impact: {MOCK_CLIENTS.find(c => c.id === selectedClient)?.name}</Typography>
                  <Button startIcon={<AssessmentIcon />}>Full Report</Button>
                </Box>
                <Typography variant="body2" color="textSecondary" paragraph>
                  Comparing current portfolio allocation against S&P 500 benchmark.
                  Note the 0% allocation to Energy due to "Fossil Free" constraint.
                </Typography>
                <ImpactChart
                  portfolioWeights={MOCK_PORTFOLIO_WEIGHTS}
                  benchmarkWeights={MOCK_BENCHMARK_WEIGHTS}
                />
              </Paper>
            </Box>
          ) : (
            <Paper elevation={3} sx={{ p: 4, textAlign: 'center', height: '100%' }}>
              <Typography color="textSecondary">
                Select a client to view their portfolio impact analysis.
              </Typography>
            </Paper>
          )}
        </Grid>
      </Grid>
    </Box>
  );
};

export default DirectIndexingPage;
