import React, { useState, useEffect } from 'react';
import {
    Container,
    Typography,
    Button,
    Table,
    TableBody,
    TableCell,
    TableContainer,
    TableHead,
    TableRow,
    Paper,
    CircularProgress,
    Alert,
    Box,
    Chip,
    Stack,
    Tooltip
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import ManageAccountsIcon from '@mui/icons-material/ManageAccounts';
import useBlockableNavigate from '../../components/RouteBlocker/useBlockableNavigate';
import { RoleSummary, RoleStatus } from '../../types/roles';
import { apiGet } from '../../utils/api';

const statusColor = (
    status: RoleStatus
): 'default' | 'success' | 'warning' | 'error' | 'primary' | 'info' | 'secondary' => {
    switch (status) {
        case 'Active':
            return 'success';
        case 'Draft':
            return 'primary';
        case 'Suspended':
            return 'warning';
        case 'Retired':
            return 'default';
        default:
            return 'default';
    }
};

const RoleListPage: React.FC = () => {
    const navigate = useBlockableNavigate();
    const [roles, setRoles] = useState<RoleSummary[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        const fetchRoles = async () => {
            try {
                const data: RoleSummary[] = await apiGet('roles');
                data.sort((a, b) => (a.displayName || a.name).localeCompare(b.displayName || b.name));
                setRoles(data);
            } catch (err: any) {
                setError(err.message || 'Unable to load roles.');
            } finally {
                setLoading(false);
            }
        };

        fetchRoles();
    }, []);

    const handleNavigateToCreate = () => {
        void navigate('/fabric/roles/create');
    };

    const handleRowClick = (role: RoleSummary) => {
        void navigate(`/fabric/roles/${encodeURIComponent(role.name)}/edit`);
    };

    return (
        <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
                <Stack direction="row" spacing={1} alignItems="center">
                    <ManageAccountsIcon color="primary" />
                    <Typography variant="h4" component="h1">
                        Role Management
                    </Typography>
                </Stack>
                <Button variant="contained" startIcon={<AddIcon />} onClick={handleNavigateToCreate}>
                    Create Role
                </Button>
            </Box>

            {loading && (
                <Box sx={{ display: 'flex', justifyContent: 'center', mt: 5 }}>
                    <CircularProgress />
                </Box>
            )}

            {error && (
                <Alert severity="error" sx={{ mt: 2 }}>
                    {error}
                </Alert>
            )}

            {!loading && !error && (
                <TableContainer component={Paper}>
                    <Table sx={{ minWidth: 850 }} aria-label="roles table">
                        <TableHead>
                            <TableRow>
                                <TableCell>Name</TableCell>
                                <TableCell>Status</TableCell>
                                <TableCell>Type</TableCell>
                                <TableCell>Owner</TableCell>
                                <TableCell>Bundles</TableCell>
                                <TableCell>Tags</TableCell>
                                <TableCell align="right">Last Updated</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {roles.map((role) => (
                                <TableRow
                                    key={role.id || role.name}
                                    hover
                                    onClick={() => handleRowClick(role)}
                                    sx={{ cursor: 'pointer' }}
                                >
                                    <TableCell component="th" scope="row">
                                        <Stack spacing={0.5}>
                                            <Typography variant="subtitle1" fontWeight={600}>
                                                {role.displayName || role.name}
                                            </Typography>
                                            {role.description && (
                                                <Typography variant="body2" color="text.secondary" noWrap>
                                                    {role.description}
                                                </Typography>
                                            )}
                                        </Stack>
                                    </TableCell>
                                    <TableCell>
                                        <Chip size="small" label={role.status} color={statusColor(role.status)} />
                                    </TableCell>
                                    <TableCell>
                                        <Chip size="small" label={role.type} variant="outlined" />
                                    </TableCell>
                                    <TableCell>{role.owner}</TableCell>
                                    <TableCell>{role.bundleIds?.length ?? 0}</TableCell>
                                    <TableCell>
                                        <Stack direction="row" spacing={0.5} sx={{ maxWidth: 200 }}>
                                            {(role.tags ?? []).slice(0, 3).map((tag) => (
                                                <Chip key={tag} size="small" variant="outlined" label={tag} />
                                            ))}
                                            {role.tags && role.tags.length > 3 && (
                                                <Tooltip title={role.tags.join(', ')}>
                                                    <Chip size="small" label={`+${role.tags.length - 3}`} />
                                                </Tooltip>
                                            )}
                                        </Stack>
                                    </TableCell>
                                    <TableCell align="right">
                                        {role.updatedAt ? new Date(role.updatedAt).toLocaleString() : '—'}
                                    </TableCell>
                                </TableRow>
                            ))}
                        </TableBody>
                    </Table>
                </TableContainer>
            )}
        </Container>
    );
};

export default RoleListPage;
