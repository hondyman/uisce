import React from 'react';
import { 
  Box, Typography, Paper, Grid, Card, CardContent, Table, 
  TableBody, TableCell, TableHead, TableRow, Chip 
} from '@mui/material';
import WarningIcon from '@mui/icons-material/Warning';

const ComplianceDashboard = () => {
    // Mock Data
    const summary = {
        piiTerms: 124,
        euTerms: 38,
        highSensitivity: 19,
        openViolations: 7,
        atRiskJobs: 12
    };

    const businessTerms = [
        { name: 'Client Address', pii: true, residency: 'EU', sensitivity: 'HIGH', linked: 14 },
        { name: 'Client SSN', pii: true, residency: 'US', sensitivity: 'HIGH', linked: 6 },
        { name: 'Email Address', pii: true, residency: 'GLOBAL', sensitivity: 'MEDIUM', linked: 22 },
    ];

    const recentEvents = [
        { time: '20:01:00', type: 'ComplianceViolationDetected', object: 'job-positions-preagg' },
        { time: '20:00:01', type: 'SemanticTermComplianceUpdated', object: 'st-client_city' },
        { time: '20:00:00', type: 'BusinessTermComplianceUpdated', object: 'bt-client-address' },
    ];

    const alerts = [
        "EU residency violation for job “Positions Pre-Agg”",
        "New PII exposure via API /positions",
        "Semantic drift introduced PII into 3 jobs"
    ];

    return (
        <Box p={3}>
            <Typography variant="h4" gutterBottom>Compliance Dashboard</Typography>

            {/* 1. Summary Tiles */}
            <Grid container spacing={3} mb={3}>
                <Grid item xs={12} md={3}>
                    <SummaryCard title="PII Terms" value={summary.piiTerms} color="error.main" />
                </Grid>
                <Grid item xs={12} md={3}>
                    <SummaryCard title="EU Terms" value={summary.euTerms} color="primary.main" />
                </Grid>
                <Grid item xs={12} md={3}>
                    <SummaryCard title="High Sensitivity" value={summary.highSensitivity} color="warning.main" />
                </Grid>
                <Grid item xs={12} md={3}>
                    <SummaryCard title="Open Violations" value={summary.openViolations} color="error.dark" />
                </Grid>
            </Grid>

            <Grid container spacing={3}>
                {/* 2. Business Terms List */}
                <Grid item xs={12} lg={8}>
                    <Paper sx={{ p: 2 }}>
                        <Typography variant="h6" gutterBottom>Business Terms with PII</Typography>
                        <Table size="small">
                            <TableHead>
                                <TableRow>
                                    <TableCell>Business Term</TableCell>
                                    <TableCell>PII</TableCell>
                                    <TableCell>Residency</TableCell>
                                    <TableCell>Sensitivity</TableCell>
                                    <TableCell>Linked</TableCell>
                                </TableRow>
                            </TableHead>
                            <TableBody>
                                {businessTerms.map((row, index) => (
                                    <TableRow key={index}>
                                        <TableCell>{row.name}</TableCell>
                                        <TableCell>{row.pii ? "Yes" : "No"}</TableCell>
                                        <TableCell>{row.residency}</TableCell>
                                        <TableCell>
                                            <Chip 
                                                label={row.sensitivity} 
                                                size="small" 
                                                color={row.sensitivity === 'HIGH' ? "error" : "default"} 
                                            />
                                        </TableCell>
                                        <TableCell>{row.linked}</TableCell>
                                    </TableRow>
                                ))}
                            </TableBody>
                        </Table>
                    </Paper>
                </Grid>

                {/* 3. AI Alerts & Events */}
                <Grid item xs={12} lg={4}>
                    <Grid container spacing={3} direction="column">
                        <Grid item>
                             <Paper sx={{ p: 2, bgcolor: '#fff4e5' }}>
                                <Typography variant="h6" gutterBottom color="warning.dark" display="flex" alignItems="center">
                                    <WarningIcon sx={{ mr: 1 }} /> AI Compliance Alerts
                                </Typography>
                                {alerts.map((alert, idx) => (
                                    <Typography key={idx} variant="body2" sx={{ mb: 1, p: 1, bgcolor: 'background.paper', borderRadius: 1 }}>
                                        ⚠️ {alert}
                                    </Typography>
                                ))}
                            </Paper>
                        </Grid>
                        <Grid item>
                            <Paper sx={{ p: 2 }}>
                                <Typography variant="h6" gutterBottom>Recent Events</Typography>
                                <Table size="small">
                                    <TableHead>
                                        <TableRow>
                                            <TableCell>Time</TableCell>
                                            <TableCell>Type</TableCell>
                                        </TableRow>
                                    </TableHead>
                                    <TableBody>
                                        {recentEvents.map((evt, idx) => (
                                            <TableRow key={idx}>
                                                <TableCell>{evt.time}</TableCell>
                                                <TableCell>
                                                    <Typography variant="caption" display="block">{evt.type}</Typography>
                                                    <Typography variant="caption" color="textSecondary">{evt.object}</Typography>
                                                </TableCell>
                                            </TableRow>
                                        ))}
                                    </TableBody>
                                </Table>
                            </Paper>
                        </Grid>
                    </Grid>
                </Grid>
            </Grid>
        </Box>
    );
};

const SummaryCard = ({ title, value, color }) => (
    <Card sx={{ height: '100%' }}>
        <CardContent>
            <Typography color="textSecondary" gutterBottom>{title}</Typography>
            <Typography variant="h3" sx={{ color: color }}>{value}</Typography>
        </CardContent>
    </Card>
);

export default ComplianceDashboard;
