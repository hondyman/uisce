import React, { useState, useMemo } from 'react';
import {
  Box,
  Typography,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Card,
  CardContent,
  Tooltip,
  IconButton,
  LinearProgress,
  Stack,
  Avatar,
  Tabs,
  Tab,
  Button,
  Divider,
  Grid
} from '@mui/material';
import {
  AccountTree as AccountTreeIcon,
  Security as SecurityIcon,
  VerifiedUser as VerifiedIcon,
  PieChart as PieChartIcon,
  Launch as LaunchIcon,
  TrendingUp as TrendingUpIcon,
  Storage as StorageIcon,
  Hub as HubIcon
} from '@mui/icons-material';
import { usePortfolioAnalytics, SecurityPosition } from '../../hooks/usePortfolioAnalytics';

const PortfolioSecurityView: React.FC = () => {
  const [activeTab, setActiveTab] = useState(0);
  const [selectedSecurity, setSelectedSecurity] = useState<string | null>(null);
  
  // Using a default portfolio ID for demonstration
  const portfolioId = '00000000-0000-0000-0000-000000000001'; 
  const { analytics, lineage, loading, getSecurityLineage } = usePortfolioAnalytics(portfolioId);

  const handleSecurityClick = async (securityId: string) => {
    setSelectedSecurity(securityId);
    await getSecurityLineage(securityId);
    setActiveTab(2); // Switch to Lineage tab
  };

  const getConfidenceColor = (score: number) => {
    if (score > 90) return '#10b981';
    if (score > 80) return '#3b82f6';
    return '#f59e0b';
  };

  const exposureData = useMemo(() => {
    if (!analytics?.sector_exposure) return [];
    return Object.entries(analytics.sector_exposure).map(([name, val]) => ({
      name,
      val,
      color: name === 'Technology' ? '#6366f1' : name === 'Financials' ? '#10b981' : '#f59e0b'
    })).sort((a, b) => b.val - a.val);
  }, [analytics]);

  return (
    <Box sx={{ p: 3, bgcolor: '#f8fafc', minHeight: '100vh' }}>
      <Box sx={{ mb: 4, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 800, color: '#1e293b', letterSpacing: '-0.02em' }}>
            Semantic Execution Fabric
          </Typography>
          <Typography variant="body1" color="textSecondary" sx={{ mt: 0.5 }}>
            Integrated Portfolio-Security Intelligence & Recursive NAV Resolution
          </Typography>
        </Box>
        <Stack direction="row" spacing={2}>
            <Button variant="outlined" startIcon={<HubIcon />} sx={{ borderRadius: 2, textTransform: 'none', fontWeight: 600 }}>
                Graph Explorer
            </Button>
            <Chip 
              icon={<VerifiedIcon />} 
              label="Semantic Spine Active" 
              color="primary" 
              sx={{ borderRadius: 2, fontWeight: 600, px: 1 }}
            />
        </Stack>
      </Box>

      {/* Summary Metrics */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
          <Grid size={{ xs: 12, md: 4 }}>
              <Card elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0' }}>
                  <CardContent sx={{ display: 'flex', alignItems: 'center' }}>
                      <Avatar sx={{ bgcolor: `#6366f115`, color: '#6366f1', mr: 2 }}><TrendingUpIcon /></Avatar>
                      <Box>
                          <Typography variant="caption" color="textSecondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>Total Portfolio Value</Typography>
                          <Typography variant="h5" sx={{ fontWeight: 800 }}>${(analytics?.total_value || 0).toLocaleString()}</Typography>
                      </Box>
                  </CardContent>
              </Card>
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
              <Card elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0' }}>
                  <CardContent sx={{ display: 'flex', alignItems: 'center' }}>
                      <Avatar sx={{ bgcolor: `#10b98115`, color: '#10b981', mr: 2 }}><VerifiedIcon /></Avatar>
                      <Box>
                          <Typography variant="caption" color="textSecondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>Confidence Score</Typography>
                          <Typography variant="h5" sx={{ fontWeight: 800 }}>{(analytics?.confidence_score || 0).toFixed(1)}%</Typography>
                      </Box>
                  </CardContent>
              </Card>
          </Grid>
          <Grid size={{ xs: 12, md: 4 }}>
              <Card elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0' }}>
                  <CardContent sx={{ display: 'flex', alignItems: 'center' }}>
                      <Avatar sx={{ bgcolor: `#3b82f615`, color: '#3b82f6', mr: 2 }}><StorageIcon /></Avatar>
                      <Box>
                          <Typography variant="caption" color="textSecondary" sx={{ fontWeight: 600, textTransform: 'uppercase' }}>Total Positions</Typography>
                          <Typography variant="h5" sx={{ fontWeight: 800 }}>{analytics?.total_positions || 0}</Typography>
                      </Box>
                  </CardContent>
              </Card>
          </Grid>
      </Grid>

      <Paper elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0', overflow: 'hidden' }}>
        <Box sx={{ borderBottom: 1, borderColor: 'divider', px: 2, pt: 1, bgcolor: '#ffffff' }}>
          <Tabs value={activeTab} onChange={(_, v) => setActiveTab(v)} sx={{ '& .MuiTab-root': { textTransform: 'none', fontWeight: 600 } }}>
            <Tab label="Holdings Analysis" />
            <Tab label="Sector & Region Exposure" />
            <Tab label="Semantic Lineage" />
          </Tabs>
        </Box>

        <Box sx={{ p: 0 }}>
          {activeTab === 0 && (
            <TableContainer>
              <Table>
                <TableHead sx={{ bgcolor: '#f8fafc' }}>
                  <TableRow>
                    <TableCell sx={{ fontWeight: 700 }}>Security</TableCell>
                    <TableCell sx={{ fontWeight: 700 }}>Asset Class</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Quantity</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Market Value</TableCell>
                    <TableCell align="right" sx={{ fontWeight: 700 }}>Weight</TableCell>
                    <TableCell align="center" sx={{ fontWeight: 700 }}>DQ Score</TableCell>
                    <TableCell align="right" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {loading ? (
                    <TableRow><TableCell colSpan={7} align="center"><LinearProgress sx={{ my: 4 }} /></TableCell></TableRow>
                  ) : (analytics?.top_holdings || []).map((h: SecurityPosition) => (
                    <TableRow key={h.security_id} hover>
                      <TableCell>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                          <SecurityIcon sx={{ color: '#94a3b8', mr: 2 }} />
                          <Box>
                            <Typography variant="subtitle2" sx={{ fontWeight: 700 }}>{h.security.name}</Typography>
                            <Typography variant="caption" color="textSecondary">{h.security.isin || h.security_id}</Typography>
                          </Box>
                        </Box>
                      </TableCell>
                      <TableCell>
                        <Chip label={h.security.asset_class} size="small" variant="outlined" sx={{ borderRadius: 1 }} />
                      </TableCell>
                      <TableCell align="right">{h.quantity.toLocaleString()}</TableCell>
                      <TableCell align="right">
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>{h.security.currency} {h.market_value.toLocaleString()}</Typography>
                      </TableCell>
                      <TableCell align="right">{(h.weight * 100).toFixed(2)}%</TableCell>
                      <TableCell align="center">
                        <Tooltip title={`Field Coverage: ${(h.confidence * 100).toFixed(0)}%`}>
                          <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                            <Typography variant="caption" sx={{ fontWeight: 800, color: getConfidenceColor(h.confidence * 100) }}>
                              {(h.confidence * 100).toFixed(0)}%
                            </Typography>
                            <Box sx={{ width: 40, mt: 0.5 }}>
                              <LinearProgress 
                                variant="determinate" 
                                value={h.confidence * 100} 
                                sx={{ height: 4, borderRadius: 2, bgcolor: '#f1f5f9', '& .MuiLinearProgress-bar': { bgcolor: getConfidenceColor(h.confidence * 100) } }}
                              />
                            </Box>
                          </Box>
                        </Tooltip>
                      </TableCell>
                      <TableCell align="right">
                        <IconButton size="small" onClick={() => handleSecurityClick(h.security_id)}>
                          <AccountTreeIcon fontSize="small" color="primary" />
                        </IconButton>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          )}

          {activeTab === 1 && (
            <Box sx={{ p: 4 }}>
              <Grid container spacing={4}>
                <Grid size={{ xs: 12, md: 6 }}>
                  <Card elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0' }}>
                    <CardContent>
                      <Typography variant="h6" gutterBottom sx={{ fontWeight: 700, display: 'flex', alignItems: 'center' }}>
                          <PieChartIcon sx={{ mr: 1, color: '#6366f1' }} /> Sector Exposure
                      </Typography>
                      <Box sx={{ mt: 3 }}>
                        {exposureData.map((s, i) => (
                          <Box key={i} sx={{ mb: 3 }}>
                            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                              <Typography variant="body2" sx={{ fontWeight: 600 }}>{s.name}</Typography>
                              <Typography variant="body2" sx={{ fontWeight: 700 }}>{s.val.toFixed(1)}%</Typography>
                            </Box>
                            <LinearProgress 
                              variant="determinate" 
                              value={s.val} 
                              sx={{ height: 10, borderRadius: 5, bgcolor: '#f1f5f9', '& .MuiLinearProgress-bar': { bgcolor: s.color } }}
                            />
                          </Box>
                        ))}
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
                <Grid size={{ xs: 12, md: 6 }}>
                  <Card elevation={0} sx={{ borderRadius: 3, border: '1px solid #e2e8f0' }}>
                    <CardContent>
                      <Typography variant="h6" gutterBottom sx={{ fontWeight: 700 }}>Geographic Breakdown</Typography>
                      <Box sx={{ mt: 3 }}>
                        {Object.entries(analytics?.region_exposure || {}).map(([name, val], i) => (
                           <Box key={i} sx={{ mb: 3 }}>
                              <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                                <Typography variant="body2" sx={{ fontWeight: 600 }}>{name}</Typography>
                                <Typography variant="body2" sx={{ fontWeight: 700 }}>{val.toFixed(1)}%</Typography>
                              </Box>
                              <LinearProgress 
                                variant="determinate" 
                                value={val} 
                                sx={
                                    { height: 10, borderRadius: 5, bgcolor: '#f8fafc', 
                                    '& .MuiLinearProgress-bar': { bgcolor: '#3b82f6' } }
                                }
                              />
                           </Box>
                        ))}
                      </Box>
                    </CardContent>
                  </Card>
                </Grid>
              </Grid>
            </Box>
          )}

          {activeTab === 2 && (
            <Box sx={{ p: 4 }}>
              {!selectedSecurity ? (
                <Box sx={{ textAlign: 'center', py: 8 }}>
                  <AccountTreeIcon sx={{ fontSize: 64, color: '#e2e8f0', mb: 2 }} />
                  <Typography color="textSecondary">Select a security from the holdings tab to view its data lineage.</Typography>
                </Box>
              ) : (
                <Box>
                    <Box sx={{ mb: 4, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                        <Box sx={{ display: 'flex', alignItems: 'center' }}>
                            <Avatar sx={{ bgcolor: '#6366f1', mr: 2 }}><SecurityIcon /></Avatar>
                            <Box>
                                <Typography variant="h6" sx={{ fontWeight: 700 }}>Lineage Trace: {selectedSecurity}</Typography>
                                <Typography variant="caption" color="textSecondary">Visualizing data flow from sources to execution</Typography>
                            </Box>
                        </Box>
                        <Stack direction="row" spacing={1}>
                            <Button variant="outlined" size="small" startIcon={<LaunchIcon />} sx={{ borderRadius: 2, textTransform: 'none' }}>
                                View Raw Source
                            </Button>
                            <Button variant="contained" color="primary" size="small" sx={{ borderRadius: 2, textTransform: 'none' }}>
                                Simulate Impact
                            </Button>
                        </Stack>
                    </Box>
                    
                    <Divider sx={{ mb: 4 }} />

                    <Stack spacing={4} sx={{ position: 'relative' }}>
                        {lineage?.edges && lineage.edges.length > 0 ? (
                            lineage.edges.map((edge, i) => (
                                <Box key={i} sx={{ display: 'flex', position: 'relative', zIndex: 1 }}>
                                    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', mr: 3 }}>
                                        <Avatar sx={{ bgcolor: '#6366f1', width: 40, height: 40, border: '4px solid #fff', boxShadow: '0 2px 4px rgba(0,0,0,0.1)' }}>
                                            <HubIcon />
                                        </Avatar>
                                        {i < lineage.edges.length - 1 && <Box sx={{ width: 2, height: '100%', bgcolor: '#e2e8f0', mt: 1 }} />}
                                    </Box>
                                    <Box sx={{ flex: 1, pb: 4 }}>
                                        <Typography variant="caption" sx={{ fontWeight: 800, color: '#6366f1' }}>{edge.type}</Typography>
                                        <Typography variant="subtitle1" sx={{ fontWeight: 700, mt: 0.5 }}>{edge.label}</Typography>
                                        <Typography variant="body2" color="textSecondary">{edge.source} → {edge.target}</Typography>
                                    </Box>
                                </Box>
                            ))
                        ) : (
                            [
                                { step: 'Source Registry', label: 'Bloomberg PORT', detail: 'Ingested raw position (AAPL_US)', color: '#6366f1', icon: <StorageIcon /> },
                                { step: 'Survivorship Logic', label: 'Golden Record Resolution', detail: 'Resolved quantity: 1,250', color: '#10b981', icon: <VerifiedIcon /> },
                                { step: 'Semantic Spine', label: 'holds_security Edge', detail: 'Linked to ISIN: US0378331005', color: '#f59e0b', icon: <HubIcon /> },
                                { step: 'Execution Engine', label: 'Recursive NAV resolution', detail: 'Valued at market price: $185.92', color: '#ef4444', icon: <TrendingUpIcon /> }
                            ].map((item, i) => (
                                <Box key={i} sx={{ display: 'flex', position: 'relative', zIndex: 1 }}>
                                    <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', mr: 3 }}>
                                        <Avatar sx={{ bgcolor: item.color, width: 40, height: 40, border: '4px solid #fff', boxShadow: '0 2px 4px rgba(0,0,0,0.1)' }}>
                                            {item.icon}
                                        </Avatar>
                                        {i < 3 && <Box sx={{ width: 2, height: '100%', bgcolor: '#e2e8f0', mt: 1 }} />}
                                    </Box>
                                    <Box sx={{ flex: 1, pb: 4 }}>
                                        <Typography variant="caption" sx={{ fontWeight: 800, color: item.color }}>{item.step}</Typography>
                                        <Typography variant="subtitle1" sx={{ fontWeight: 700, mt: 0.5 }}>{item.label}</Typography>
                                        <Typography variant="body2" color="textSecondary">{item.detail}</Typography>
                                    </Box>
                                </Box>
                            ))
                        )}
                    </Stack>
                </Box>
              )}
            </Box>
          )}
        </Box>
      </Paper>
    </Box>
  );
};

export default PortfolioSecurityView;
