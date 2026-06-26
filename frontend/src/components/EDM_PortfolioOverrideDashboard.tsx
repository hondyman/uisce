import React, { useState } from 'react';
import {
  Box,
  Card,
  CardContent,
  CardHeader,
  Button,
  Tabs,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  Chip,
  Grid,
  Alert,
  LinearProgress,
  Typography,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  TextField,
  MenuItem,
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import TrendingUpIcon from '@mui/icons-material/TrendingUp';
import TrendingDownIcon from '@mui/icons-material/TrendingDown';
import FolderIcon from '@mui/icons-material/Folder';
import WarningIcon from '@mui/icons-material/Warning';

interface PortfolioOverride {
  id: string;
  portfolio: string;
  assetClass: string;
  overrideType: string;
  impact: number;
  status: 'active' | 'pending' | 'rejected';
  createdDate: string;
  exposureChange: number;
}

interface Portfolio {
  id: string;
  name: string;
  healthScore: number;
  activeOverrides: number;
}

const mockPortfolios: Portfolio[] = [
  { id: 'P-001', name: 'US Equity Prime', healthScore: 92, activeOverrides: 3 },
  { id: 'P-002', name: 'Fixed Income Core', healthScore: 87, activeOverrides: 1 },
  { id: 'P-003', name: 'Emerging Markets', healthScore: 78, activeOverrides: 5 },
];

const mockOverrides: PortfolioOverride[] = [
  {
    id: 'OR-001',
    portfolio: 'US Equity Prime',
    assetClass: 'Large Cap',
    overrideType: 'Source Selection',
    impact: 2.3,
    status: 'active',
    createdDate: '2026-02-15',
    exposureChange: 1.8,
  },
  {
    id: 'OR-002',
    portfolio: 'Fixed Income Core',
    assetClass: 'Bonds',
    overrideType: 'Valuation Adjustment',
    impact: -0.8,
    status: 'pending',
    createdDate: '2026-02-18',
    exposureChange: -0.5,
  },
  {
    id: 'OR-003',
    portfolio: 'Emerging Markets',
    assetClass: 'Equities',
    overrideType: 'Risk Weight',
    impact: 0.5,
    status: 'active',
    createdDate: '2026-02-19',
    exposureChange: 0.2,
  },
];

export const PortfolioOverrideDashboard: React.FC = () => {
  const [tab, setTab] = useState(0);
  const [openDialog, setOpenDialog] = useState(false);
  const [selectedPortfolio, setSelectedPortfolio] = useState<string>('P-001');
  const [overrideType, setOverrideType] = useState('source-selection');
  const [portfolios] = useState(mockPortfolios);
  const [overrides] = useState(mockOverrides);

  const handleOpenDialog = () => {
    setOpenDialog(true);
  };

  const handleCloseDialog = () => {
    setOpenDialog(false);
  };

  const getHealthColor = (score: number): 'success' | 'warning' | 'error' => {
    if (score >= 85) return 'success';
    if (score >= 70) return 'warning';
    return 'error';
  };

  const getStatusColor = (status: string): 'default' | 'success' | 'warning' | 'error' => {
    if (status === 'active') return 'success';
    if (status === 'pending') return 'warning';
    return 'error';
  };

  const selectedPortfolioData = portfolios.find((p) => p.id === selectedPortfolio);

  return (
    <Box sx={{ p: 3 }}>
      <Card>
        <CardHeader
          title="Portfolio Override Dashboard"
          subheader="Manage and track data source overrides by portfolio"
          action={
            <Button
              variant="contained"
              startIcon={<AddIcon />}
              onClick={handleOpenDialog}
            >
              Create Override
            </Button>
          }
        />
        <CardContent>
          <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ mb: 3 }}>
            <Tab label="Portfolio Health" />
            <Tab label="Override History" />
            <Tab label="Impact Analysis" />
            <Tab label="Templates" />
          </Tabs>

          {/* Portfolio Health Tab */}
          {tab === 0 && (
            <Grid container spacing={3}>
              {portfolios.map((portfolio) => (
                <Grid item xs={12} sm={6} md={4} key={portfolio.id}>
                  <Card
                    sx={{
                      cursor: 'pointer',
                      border: selectedPortfolio === portfolio.id ? '2px solid #137fec' : '1px solid #e5e7eb',
                      transition: 'all 0.2s',
                      '&:hover': { boxShadow: 2 },
                    }}
                    onClick={() => setSelectedPortfolio(portfolio.id)}
                  >
                    <CardContent>
                      <Box sx={{ display: 'flex', alignItems: 'start', justifyContent: 'space-between', mb: 2 }}>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <FolderIcon sx={{ color: '#137fec' }} />
                          <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>
                            {portfolio.name}
                          </Typography>
                        </Box>
                      </Box>
                      <Box sx={{ mb: 2 }}>
                        <Typography variant="caption" color="textSecondary">
                          Health Score
                        </Typography>
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mt: 0.5 }}>
                          <LinearProgress
                            variant="determinate"
                            value={portfolio.healthScore}
                            sx={{ flex: 1, height: 8, borderRadius: 4 }}
                            color={getHealthColor(portfolio.healthScore)}
                          />
                          <Typography variant="body2" sx={{ fontWeight: 600, minWidth: 40 }}>
                            {portfolio.healthScore}/100
                          </Typography>
                        </Box>
                      </Box>
                      <Chip
                        label={`${portfolio.activeOverrides} active overrides`}
                        size="small"
                        color={portfolio.activeOverrides > 0 ? 'warning' : 'success'}
                        variant="outlined"
                      />
                    </CardContent>
                  </Card>
                </Grid>
              ))}
            </Grid>
          )}

          {/* Override History Tab */}
          {tab === 1 && (
            <Box>
              <Alert severity="info" sx={{ mb: 2 }}>
                Showing {overrides.length} overrides across all portfolios. Click portfolio cards above to filter.
              </Alert>
              <TableContainer component={Paper}>
                <Table>
                  <TableHead sx={{ bgcolor: '#f3f4f6' }}>
                    <TableRow>
                      <TableCell sx={{ fontWeight: 700 }}>Override ID</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Portfolio</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Asset Class</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Type</TableCell>
                      <TableCell align="right" sx={{ fontWeight: 700 }}>
                        Impact
                      </TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Status</TableCell>
                      <TableCell sx={{ fontWeight: 700 }}>Created</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {overrides.map((override) => (
                      <TableRow key={override.id} hover>
                        <TableCell sx={{ fontWeight: 600, color: '#137fec' }}>
                          {override.id}
                        </TableCell>
                        <TableCell>{override.portfolio}</TableCell>
                        <TableCell>{override.assetClass}</TableCell>
                        <TableCell>
                          <Chip label={override.overrideType} size="small" variant="outlined" />
                        </TableCell>
                        <TableCell align="right">
                          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-end', gap: 0.5 }}>
                            {override.impact > 0 ? (
                              <TrendingUpIcon sx={{ color: 'success.main', fontSize: 18 }} />
                            ) : (
                              <TrendingDownIcon sx={{ color: 'error.main', fontSize: 18 }} />
                            )}
                            <Typography
                              variant="body2"
                              sx={{
                                fontWeight: 600,
                                color: override.impact > 0 ? 'success.main' : 'error.main',
                              }}
                            >
                              {override.impact > 0 ? '+' : ''}{override.impact.toFixed(2)}%
                            </Typography>
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={override.status.charAt(0).toUpperCase() + override.status.slice(1)}
                            color={getStatusColor(override.status)}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>{override.createdDate}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>
            </Box>
          )}

          {/* Impact Analysis Tab */}
          {tab === 2 && (
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <Card>
                  <CardHeader title="Portfolio Impact Summary" />
                  <CardContent>
                    <Box sx={{ space: 2 }}>
                      <Box sx={{ mb: 2 }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                          Total Exposure Change
                        </Typography>
                        <Typography variant="h5" sx={{ color: 'success.main', fontWeight: 700 }}>
                          +2.3%
                        </Typography>
                        <Typography variant="caption" color="textSecondary">
                          Across all active overrides
                        </Typography>
                      </Box>
                      <Box sx={{ mb: 2 }}>
                        <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                          Risk Adjusted Return
                        </Typography>
                        <Typography variant="h5" sx={{ color: '#137fec', fontWeight: 700 }}>
                          +1.8%
                        </Typography>
                        <Typography variant="caption" color="textSecondary">
                          Annualized basis points
                        </Typography>
                      </Box>
                      <Alert severity="warning">
                        Consider distributing overrides across more asset classes to reduce concentration risk.
                      </Alert>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>

              <Grid item xs={12} md={6}>
                <Card>
                  <CardHeader title="Scenario Analysis" />
                  <CardContent>
                    <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                        What-if analysis: Impact of enabling pending overrides
                    </Typography>
                    <Box sx={{ p: 2, bgcolor: '#f3f4f6', borderRadius: 1 }}>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                        <Typography variant="body2">Current State</Typography>
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>
                          +2.3%
                        </Typography>
                      </Box>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                        <Typography variant="body2">With Pending</Typography>
                        <Typography variant="body2" sx={{ fontWeight: 600, color: 'warning.main' }}>
                          +1.5%
                        </Typography>
                      </Box>
                      <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                        <Typography variant="body2">Delta</Typography>
                        <Typography variant="body2" sx={{ fontWeight: 600, color: 'error.main' }}>
                          -0.8%
                        </Typography>
                      </Box>
                    </Box>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>
          )}

          {/* Templates Tab */}
          {tab === 3 && (
            <Grid container spacing={2}>
              <Grid item xs={12}>
                <Alert severity="info">
                  Save frequently used override configurations as templates for faster deployment.
                </Alert>
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <Card>
                  <CardContent>
                    <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                      📋 Standard Source Override
                    </Typography>
                    <Typography variant="caption" color="textSecondary" sx={{ mb: 2, display: 'block' }}>
                      Primary source replacement with fallback configuration
                    </Typography>
                    <Button size="small" variant="outlined" fullWidth>
                      Use Template
                    </Button>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <Card>
                  <CardContent>
                    <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                      ⚖️ Risk Weight Adjustment
                    </Typography>
                    <Typography variant="caption" color="textSecondary" sx={{ mb: 2, display: 'block' }}>
                        Temporary risk weight increase for upgrades
                    </Typography>
                    <Button size="small" variant="outlined" fullWidth>
                      Use Template
                    </Button>
                  </CardContent>
                </Card>
              </Grid>
              <Grid item xs={12} sm={6} md={4}>
                <Card>
                  <CardContent>
                    <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 1 }}>
                      🔄 Multi-Portfolio Sync
                    </Typography>
                    <Typography variant="caption" color="textSecondary" sx={{ mb: 2, display: 'block' }}>
                        Apply same override across multiple portfolios
                    </Typography>
                    <Button size="small" variant="outlined" fullWidth>
                      Use Template
                    </Button>
                  </CardContent>
                </Card>
              </Grid>
            </Grid>
          )}
        </CardContent>
      </Card>

      {/* Create Override Dialog */}
      <Dialog open={openDialog} onClose={handleCloseDialog} maxWidth="sm" fullWidth>
        <DialogTitle>Create New Override</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <TextField
            select
            label="Portfolio"
            value={selectedPortfolio}
            onChange={(e) => setSelectedPortfolio(e.target.value)}
            fullWidth
            sx={{ mb: 2 }}
          >
            {portfolios.map((p) => (
              <MenuItem key={p.id} value={p.id}>
                {p.name}
              </MenuItem>
            ))}
          </TextField>

          <TextField
            select
            label="Override Type"
            value={overrideType}
            onChange={(e) => setOverrideType(e.target.value)}
            fullWidth
            sx={{ mb: 2 }}
          >
            <MenuItem value="source-selection">Source Selection</MenuItem>
            <MenuItem value="risk-weight">Risk Weight Adjustment</MenuItem>
            <MenuItem value="valuation">Valuation Adjustment</MenuItem>
            <MenuItem value="confidence">Confidence Score</MenuItem>
          </TextField>

          <TextField
            label="Effective Date"
            type="date"
            fullWidth
            InputLabelProps={{ shrink: true }}
            sx={{ mb: 2 }}
          />

          <TextField
            label="Reason for Override"
            multiline
            rows={3}
            fullWidth
            placeholder="Provide business justification..."
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDialog}>Cancel</Button>
          <Button variant="contained" onClick={handleCloseDialog}>
            Create Override
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};
