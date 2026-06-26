import React, { useEffect } from 'react';
import { Box, Typography, Button, TextField, InputAdornment } from '@mui/material';
import { DataGrid, GridColDef, GridRenderCellParams } from '@mui/x-data-grid';
import { Refresh as RefreshIcon, Search as SearchIcon } from '@mui/icons-material';
import { useSecurityEvents } from '../hooks/useSecurityEvents';
import { format } from 'date-fns';

export const AuditLogsPage: React.FC = () => {
    const { events, loading, error, fetchEvents } = useSecurityEvents();
    const [searchText, setSearchText] = React.useState('');

    useEffect(() => {
        fetchEvents();
    }, [fetchEvents]);

    const filteredEvents = events.filter((e) => 
        JSON.stringify(e).toLowerCase().includes(searchText.toLowerCase())
    );

    const columns: GridColDef[] = [
        { field: 'created_at', headerName: 'Time', width: 180,
            valueFormatter: (params) => {
                if (!params.value) return '';
                return format(new Date(params.value), 'yyyy-MM-dd HH:mm:ss');
            }
        },
        { field: 'event_type', headerName: 'Event Type', width: 180 },
        { field: 'entity_type', headerName: 'Entity Type', width: 120 },
        { field: 'entity_id', headerName: 'Entity ID', width: 150 },
        { field: 'actor_id', headerName: 'Actor ID', width: 150 },
        { 
            field: 'payload', 
            headerName: 'Details', 
            flex: 1,
            renderCell: (params) => (
                <Typography variant="body2" noWrap title={JSON.stringify(params.value, null, 2)}>
                    {JSON.stringify(params.value)}
                </Typography>
            )
        },
    ];

    return (
        <Box sx={{ p: 3, height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
                <Typography variant="h4">Audit Logs</Typography>
                <Box sx={{ display: 'flex', gap: 2 }}>
                    <TextField
                        size="small"
                        placeholder="Search logs..."
                        value={searchText}
                        onChange={(e) => setSearchText(e.target.value)}
                        InputProps={{
                            startAdornment: <InputAdornment position="start"><SearchIcon /></InputAdornment>
                        }}
                    />
                    <Button startIcon={<RefreshIcon />} onClick={() => fetchEvents()}>
                        Refresh
                    </Button>
                </Box>
            </Box>

            {error && (
                <Typography color="error" sx={{ mb: 2 }}>{error}</Typography>
            )}

            <Box sx={{ flexGrow: 1 }}>
                <DataGrid
                    rows={filteredEvents}
                    columns={columns}
                    getRowId={(row) => row.event_id}
                    loading={loading}
                    initialState={{
                        pagination: { paginationModel: { pageSize: 50 } },
                        sorting: { sortModel: [{ field: 'created_at', sort: 'desc' }] },
                    }}
                    pageSizeOptions={[25, 50, 100]}
                />
            </Box>
        </Box>
    );
};
