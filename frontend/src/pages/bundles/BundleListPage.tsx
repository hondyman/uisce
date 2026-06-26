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
	Chip,
	CircularProgress,
	Alert,
	Box,
	Stack
} from '@mui/material';
import AddIcon from '@mui/icons-material/Add';
import { DataBundle, BundleStatus } from '../../types/bundles';
import useBlockableNavigate from '../../components/RouteBlocker/useBlockableNavigate';
import { useTenant } from '../../contexts/TenantContext';
import { getSelectedRegion } from '../../lib/region';

// Helper to get color for bundle status
const getStatusColor = (status: BundleStatus) => {
    switch (status) {
        case 'Draft':
            return 'primary';
        case 'Certified':
            return 'warning';
        case 'Published':
            return 'success';
        case 'Deprecated':
            return 'default';
        default:
            return 'default';
    }
};

const parseTimestamp = (value?: string) => (value ? new Date(value).getTime() : 0);

const BundleListPage: React.FC = () => {
    const navigate = useBlockableNavigate();
    const [bundles, setBundles] = useState<DataBundle[]>([]);
    const [loading, setLoading] = useState<boolean>(true);
    const [error, setError] = useState<string | null>(null);
    const { tenant, datasource } = useTenant();
    const tenantId = tenant?.id?.trim() ?? '';
    const datasourceId = (datasource?.id ?? datasource?.alpha_tenant_instance_id ?? '').trim();
    const selectionMissing = !tenantId || !datasourceId;

    useEffect(() => {
        let cancelled = false;
        const fetchBundles = async () => {
            if (selectionMissing) {
                setBundles([]);
                setLoading(false);
                setError(null);
                return;
            }

            setLoading(true);
            setError(null);

            try {
                const response = await fetch('/api/bundles', {
                    credentials: 'include',
                    headers: {
                        'X-Tenant-ID': tenantId,
                        'X-Tenant-Datasource-ID': datasourceId,
                        'X-Tenant-Region': getSelectedRegion(),
                    },
                });
                if (!response.ok) {
                    throw new Error(`Failed to fetch bundles: ${response.statusText}`);
                }
                const data: DataBundle[] = await response.json();
                if (cancelled) {
                    return;
                }
                data.sort((a, b) => parseTimestamp(b.updatedAt) - parseTimestamp(a.updatedAt));
                setBundles(data);
            } catch (err: any) {
                if (cancelled) {
                    return;
                }
                setError(err.message);
            } finally {
                if (!cancelled) {
                    setLoading(false);
                }
            }
        };

        fetchBundles();

        return () => {
            cancelled = true;
        };
    }, [selectionMissing, tenantId, datasourceId]);

    const handleNavigateToCreate = () => {
        void navigate('/fabric/bundles/create');
    };

    const handleRowClick = (id: string) => {
        void navigate(`/fabric/bundles/${encodeURIComponent(id)}/edit`);
    };

    return (
        <Container maxWidth="lg" sx={{ mt: 4, mb: 4 }}>
            <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
                <Typography variant="h4" component="h1">
                    Data Bundles
                </Typography>
                <Button
					variant="contained"
					startIcon={<AddIcon />}
					onClick={handleNavigateToCreate}
					disabled={selectionMissing}
				>
					Create Bundle
				</Button>
            </Box>

            {selectionMissing && (
                <Alert severity="warning" sx={{ mt: 2 }}>
                    Select a tenant and tenant datasource to view bundles scoped to that environment.
                </Alert>
            )}

            {loading && !selectionMissing && (
                <Box sx={{ display: 'flex', justifyContent: 'center', mt: 5 }}>
                    <CircularProgress />
                </Box>
            )}

            {error && !selectionMissing && (
                <Alert severity="error" sx={{ mt: 2 }}>
                    {error}
                </Alert>
            )}

            {!loading && !error && !selectionMissing && (
                <TableContainer component={Paper}>
                    <Table sx={{ minWidth: 650 }} aria-label="data bundles table">
                        <TableHead>
                            <TableRow>
                                <TableCell>Name</TableCell>
                                <TableCell>Version</TableCell>
                                <TableCell>Status</TableCell>
                                <TableCell>Owner</TableCell>
                                <TableCell>Last Updated</TableCell>
                            </TableRow>
                        </TableHead>
                        <TableBody>
                            {bundles.map((bundle) => (
                                <TableRow
                                    key={bundle.id}
                                    hover
                                    onClick={() => handleRowClick(bundle.id)}
                                    sx={{ cursor: 'pointer' }}
                                >
                                    <TableCell component="th" scope="row">
                                        {bundle.name}
                                    </TableCell>
                                    <TableCell>{bundle.version}</TableCell>
                                    <TableCell>
                                        <Chip label={bundle.status} color={getStatusColor(bundle.status)} size="small" />
                                    </TableCell>
                                    <TableCell>{bundle.owner}</TableCell>
                                    <TableCell>{bundle.updatedAt ? new Date(bundle.updatedAt).toLocaleString() : '—'}</TableCell>
                                </TableRow>
                            ))}
                            {bundles.length === 0 && (
                                <TableRow>
                                    <TableCell colSpan={5} align="center">
                                        <Stack spacing={1} alignItems="center" sx={{ py: 6 }}>
                                            <Typography variant="subtitle1">No bundles yet</Typography>
                                            <Typography variant="body2" color="text.secondary">
                                                Create a bundle to publish curated semantic datasets.
                                            </Typography>
                                            <Button variant="contained" startIcon={<AddIcon />} onClick={handleNavigateToCreate}>
                                                Create your first bundle
                                            </Button>
                                        </Stack>
                                    </TableCell>
                                </TableRow>
                            )}
                        </TableBody>
                    </Table>
                </TableContainer>
            )}
        </Container>
    );
};

export default BundleListPage;
