import React, { useState } from 'react';
import {
    Box,
    Paper,
    Typography,
    Button,
    Chip,
    IconButton,
    Dialog,
    DialogTitle,
    DialogContent,
    DialogActions,
    TextField,
    MenuItem,
    Alert
} from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { Download as DownloadIcon, Refresh as RefreshIcon, Add as AddIcon, Description as DescriptionIcon } from '@mui/icons-material';
import { useComplianceReports } from '../hooks/useComplianceReports';
import { ComplianceReport } from '../types/security';
import { format } from 'date-fns';

export const ComplianceReportsPage: React.FC = () => {
    const { reports, loading, error, fetchReports, generateReport } = useComplianceReports();
    const [openDialog, setOpenDialog] = useState(false);
    const [newReportType, setNewReportType] = useState('SOC2');
    const [newReportTitle, setNewReportTitle] = useState('');

    const handleGenerate = async () => {
        if (!newReportTitle) return;
        await generateReport(newReportType, newReportTitle);
        setOpenDialog(false);
        setNewReportTitle('');
    };

    const columns: GridColDef[] = [
        { 
            field: 'title', 
            headerName: 'Report Title', 
            flex: 1,
            renderCell: (params: GridRenderCellParams<any, string>) => (
                <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                    <DescriptionIcon color="action" fontSize="small" />
                    <Typography variant="body2">{params.value}</Typography>
                </Box>
            )
        },
        { 
            field: 'type', 
            headerName: 'Type', 
            width: 120,
            renderCell: (params: GridRenderCellParams<any, string>) => (
                <Chip label={params.value} size="small" variant="outlined" />
            )
        },
        { 
            field: 'status', 
            headerName: 'Status', 
            width: 130,
            renderCell: (params: GridRenderCellParams<any, string>) => {
                const status = params.value as string;
                let color: 'default' | 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning' = 'default';
                if (status === 'published') color = 'success';
                if (status === 'generated') color = 'info';
                if (status === 'draft') color = 'warning';
                
                return <Chip label={status.toUpperCase()} color={color} size="small" />;
            }
        },
        { 
            field: 'created_at', 
            headerName: 'Created At', 
            width: 180,
            valueFormatter: (params: { value: string }) => {
                if (!params.value) return '';
                return format(new Date(params.value), 'PP pp');
            }
        },
        { field: 'created_by', headerName: 'Created By', width: 150 },
        {
            field: 'actions',
            headerName: 'Actions',
            width: 100,
            sortable: false,
            renderCell: (params: GridRenderCellParams<any, ComplianceReport>) => (
                params.row.download_url ? (
                    <IconButton 
                        size="small" 
                        color="primary"
                        onClick={() => window.open(params.row.download_url, '_blank')}
                        title="Download Report"
                    >
                        <DownloadIcon />
                    </IconButton>
                ) : null
            )
        }
    ];

    return (
        <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 3 }}>
                <div>
                    <Typography variant="h4" gutterBottom>
                        Compliance Reports
                    </Typography>
                    <Typography variant="body1" color="textSecondary">
                        Generate and manage compliance reports for SOC2, GDPR, and internal audits.
                    </Typography>
                </div>
                <Box sx={{ display: 'flex', gap: 2 }}>
                    <Button 
                        startIcon={<RefreshIcon />} 
                        onClick={fetchReports} 
                        variant="outlined"
                    >
                        Refresh
                    </Button>
                    <Button 
                        startIcon={<AddIcon />} 
                        variant="contained" 
                        color="primary"
                        onClick={() => setOpenDialog(true)}
                    >
                        New Report
                    </Button>
                </Box>
            </Box>

            {error && (
                <Alert severity="error" sx={{ mb: 2 }}>
                    {error.message}
                </Alert>
            )}

            <Paper sx={{ flexGrow: 1, width: '100%', p: 1 }}>
                <DataGrid
                    rows={reports}
                    columns={columns}
                    loading={loading}
                    initialState={{
                        pagination: {
                            paginationModel: { pageSize: 10, page: 0 },
                        },
                        sorting: {
                            sortModel: [{ field: 'created_at', sort: 'desc' }],
                        },
                    }}
                    pageSizeOptions={[10, 25, 50]}
                    disableRowSelectionOnClick
                    sx={{ border: 'none' }}
                />
            </Paper>

            <Dialog open={openDialog} onClose={() => setOpenDialog(false)}>
                <DialogTitle>Generate New Compliance Report</DialogTitle>
                <DialogContent sx={{ minWidth: 400, pt: 1 }}>
                    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, mt: 1 }}>
                        <TextField
                            label="Report Title"
                            fullWidth
                            value={newReportTitle}
                            onChange={(e) => setNewReportTitle(e.target.value)}
                            placeholder="e.g. Q1 2026 Audit"
                            autoFocus
                        />
                        <TextField
                            select
                            label="Report Type"
                            fullWidth
                            value={newReportType}
                            onChange={(e) => setNewReportType(e.target.value)}
                        >
                            <MenuItem value="SOC2">SOC2</MenuItem>
                            <MenuItem value="GDPR">GDPR</MenuItem>
                            <MenuItem value="ISO27001">ISO27001</MenuItem>
                            <MenuItem value="Internal">Internal Audit</MenuItem>
                        </TextField>
                    </Box>
                </DialogContent>
                <DialogActions>
                    <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
                    <Button 
                        onClick={handleGenerate} 
                        variant="contained"
                        disabled={!newReportTitle}
                    >
                        Generate
                    </Button>
                </DialogActions>
            </Dialog>
        </Box>
    );
};
