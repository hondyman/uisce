import React, { useEffect, useState } from 'react';
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
    IconButton,
    Collapse,
    CircularProgress,
    Alert
} from '@mui/material';
import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown';
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp';
import RefreshIcon from '@mui/icons-material/Refresh';
import { ExecutionLog, listExecutionLogs } from '../../../api/execution_logs';
import ReactJson from 'react-json-view';

function Row({ row }: { row: ExecutionLog }) {
    const [open, setOpen] = useState(false);

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'completed': return 'success';
            case 'failed': return 'error';
            case 'started': return 'info';
            default: return 'default';
        }
    };

    return (
        <React.Fragment>
            <TableRow sx={{ '& > *': { borderBottom: 'unset' } }}>
                <TableCell>
                    <IconButton
                        aria-label="expand row"
                        size="small"
                        onClick={() => setOpen(!open)}
                    >
                        {open ? <KeyboardArrowUpIcon /> : <KeyboardArrowDownIcon />}
                    </IconButton>
                </TableCell>
                <TableCell component="th" scope="row">
                    {new Date(row.started_at).toLocaleString()}
                </TableCell>
                <TableCell>{row.event_type}</TableCell>
                <TableCell>{row.engine}</TableCell>
                <TableCell>
                    <Chip label={row.status} color={getStatusColor(row.status) as any} size="small" />
                </TableCell>
                <TableCell align="right">
                    {row.duration_ms ? `${row.duration_ms.toFixed(2)} ms` : '-'}
                </TableCell>
            </TableRow>
            <TableRow>
                <TableCell style={{ paddingBottom: 0, paddingTop: 0 }} colSpan={6}>
                    <Collapse in={open} timeout="auto" unmountOnExit>
                        <Box sx={{ margin: 1 }}>
                            <Typography variant="h6" gutterBottom component="div">
                                Details
                            </Typography>
                            {row.error_message && (
                                <Alert severity="error" sx={{ mb: 2 }}>
                                    {row.error_message}
                                </Alert>
                            )}
                            <Box sx={{ display: 'flex', gap: 2 }}>
                                <Box sx={{ flex: 1 }}>
                                    <Typography variant="subtitle2">Payload</Typography>
                                    <Paper variant="outlined" sx={{ p: 1, maxHeight: 300, overflow: 'auto' }}>
                                        <ReactJson src={row.payload || {}} collapsed={1} name={false} displayDataTypes={false} />
                                    </Paper>
                                </Box>
                                <Box sx={{ flex: 1 }}>
                                    <Typography variant="subtitle2">Result</Typography>
                                    <Paper variant="outlined" sx={{ p: 1, maxHeight: 300, overflow: 'auto' }}>
                                        <ReactJson src={row.result || {}} collapsed={1} name={false} displayDataTypes={false} />
                                    </Paper>
                                </Box>
                            </Box>
                        </Box>
                    </Collapse>
                </TableCell>
            </TableRow>
        </React.Fragment>
    );
}

export default function ExecutionMonitorPage() {
    const [logs, setLogs] = useState<ExecutionLog[]>([]);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);

    const fetchLogs = async () => {
        setLoading(true);
        try {
            const data = await listExecutionLogs();
            setLogs(data);
            setError(null);
        } catch (err: any) {
            setError(err.message || 'Failed to fetch logs');
        } finally {
            setLoading(false);
        }
    };

    useEffect(() => {
        fetchLogs();
        const interval = setInterval(fetchLogs, 5000); // Auto-refresh every 5s
        return () => clearInterval(interval);
    }, []);

    return (
        <Box sx={{ p: 3 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
                <Typography variant="h4">Execution Monitor</Typography>
                <IconButton onClick={fetchLogs} disabled={loading}>
                    <RefreshIcon />
                </IconButton>
            </Box>

            {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

            <TableContainer component={Paper}>
                <Table aria-label="collapsible table">
                    <TableHead>
                        <TableRow>
                            <TableCell />
                            <TableCell>Started At</TableCell>
                            <TableCell>Event Type</TableCell>
                            <TableCell>Engine</TableCell>
                            <TableCell>Status</TableCell>
                            <TableCell align="right">Duration</TableCell>
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {loading && logs.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={6} align="center">
                                    <CircularProgress />
                                </TableCell>
                            </TableRow>
                        ) : (
                            logs.map((row) => (
                                <Row key={row.id} row={row} />
                            ))
                        )}
                    </TableBody>
                </Table>
            </TableContainer>
        </Box>
    );
}
