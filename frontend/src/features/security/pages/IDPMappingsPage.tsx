import React, { useState, useEffect } from 'react';
import {
	Box,
	Button,
	Dialog,
	DialogActions,
	DialogContent,
	DialogTitle,
	Paper,
	Table,
	TableBody,
	TableCell,
	TableContainer,
	TableHead,
	TableRow,
	TextField,
	Typography,
	IconButton,
	MenuItem,
	Select,
	FormControl,
	InputLabel,
	FormHelperText,
	CircularProgress,
	Alert,
	Snackbar
} from '@mui/material';
import {
	Delete as DeleteIcon,
	Add as AddIcon,
	Security as SecurityIcon,
	Refresh as RefreshIcon
} from '@mui/icons-material';
import { apiFetch } from '../../../lib/apiClient';

interface IDPMapping {
	mapping_id: string;
	tenant_id: string;
	tenant_name: string;
	idp_client_id: string;
	idp_group_id: string;
	functional_role: string;
	clearance_level: string;
	created_at: string;
}

interface Tenant {
	id: string;
	display_name: string;
	name: string;
}

export const IDPMappingsPage: React.FC = () => {
	const [mappings, setMappings] = useState<IDPMapping[]>([]);
	const [tenants, setTenants] = useState<Tenant[]>([]);
	const [loading, setLoading] = useState(true);
	const [error, setError] = useState<string | null>(null);
	const [isCreating, setIsCreating] = useState(false);
	
	// Form state
	const [formTenantId, setFormTenantId] = useState('');
	const [formClientId, setFormClientId] = useState('semlayer-frontend');
	const [formGroupId, setFormGroupId] = useState('');
	const [formRole, setFormRole] = useState('platform_analyst');
	const [formClearance, setFormClearance] = useState('L1');
	const [validationError, setValidationError] = useState<string | null>(null);

	// Notification state
	const [snackbarOpen, setSnackbarOpen] = useState(false);
	const [snackbarMessage, setSnackbarMessage] = useState('');
	const [snackbarSeverity, setSnackbarSeverity] = useState<'success' | 'error'>('success');

	const fetchMappings = async () => {
		try {
			setLoading(true);
			setError(null);
			// apiFetch injects Authorization: Bearer <jwt> + tenant headers automatically
			const response = await apiFetch('/api/admin/idp-mappings');
			if (!response.ok) {
				throw new Error(`HTTP error ${response.status}`);
			}
			const data = await response.json();
			setMappings(Array.isArray(data) ? data : []);
		} catch (err) {
			console.error('Failed to fetch IDP mappings:', err);
			setError(err instanceof Error ? err.message : 'Failed to fetch mappings');
		} finally {
			setLoading(false);
		}
	};

	const fetchTenants = async () => {
		try {
			const response = await apiFetch('/api/tenants');
			if (response.ok) {
				const data = await response.json();
				setTenants(Array.isArray(data) ? data : []);
			}
		} catch (err) {
			console.error('Failed to fetch tenants:', err);
		}
	};

	useEffect(() => {
		fetchMappings();
		fetchTenants();
	}, []);

	const handleOpenCreateDialog = () => {
		setFormTenantId('');
		setFormClientId('semlayer-frontend');
		setFormGroupId('');
		setFormRole('platform_analyst');
		setFormClearance('L1');
		setValidationError(null);
		setIsCreating(true);
	};

	const handleCloseCreateDialog = () => {
		setIsCreating(false);
	};

	const handleSubmit = async (e: React.FormEvent) => {
		e.preventDefault();
		setValidationError(null);

		// Client-side validations
		if (!formTenantId) {
			setValidationError('Please select a Tenant');
			return;
		}
		if (!formClientId.trim()) {
			setValidationError('Client ID is required');
			return;
		}
		if (!formGroupId.trim()) {
			setValidationError('Group ID is required');
			return;
		}
		if (!formRole) {
			setValidationError('Please select a Functional Role');
			return;
		}

		try {
			const response = await apiFetch('/api/admin/idp-mappings', {
				method: 'POST',
				// apiFetch already sets Content-Type via getTenantHeadersInternal
				body: JSON.stringify({
					tenant_id: formTenantId,
					idp_client_id: formClientId.trim(),
					idp_group_id: formGroupId.trim(),
					functional_role: formRole,
					clearance_level: formClearance
				})
			});

			if (!response.ok) {
				const errData = await response.json().catch(() => ({}));
				throw new Error(errData.error || `HTTP error ${response.status}`);
			}

			setSnackbarMessage('Mapping created successfully');
			setSnackbarSeverity('success');
			setSnackbarOpen(true);
			setIsCreating(false);
			fetchMappings();
		} catch (err) {
			setValidationError(err instanceof Error ? err.message : 'Failed to create mapping');
		}
	};

	const handleDelete = async (id: string) => {
		if (!window.confirm('Are you sure you want to delete this mapping?')) {
			return;
		}

		try {
			const response = await apiFetch(`/api/admin/idp-mappings/${id}`, {
				method: 'DELETE'
			});

			if (!response.ok) {
				const errData = await response.json().catch(() => ({}));
				throw new Error(errData.error || `HTTP error ${response.status}`);
			}

			setSnackbarMessage('Mapping deleted successfully');
			setSnackbarSeverity('success');
			setSnackbarOpen(true);
			fetchMappings();
		} catch (err) {
			setSnackbarMessage(err instanceof Error ? err.message : 'Failed to delete mapping');
			setSnackbarSeverity('error');
			setSnackbarOpen(true);
		}
	};

	return (
		<Box sx={{ p: 3 }}>
			<Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
				<Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
					<SecurityIcon color="primary" />
					<Typography variant="h5" fontWeight="bold">
						IDP Tenant Mappings
					</Typography>
				</Box>
				<Box sx={{ display: 'flex', gap: 1 }}>
					<Button
						variant="outlined"
						startIcon={<RefreshIcon />}
						onClick={fetchMappings}
						disabled={loading}
					>
						Refresh
					</Button>
					<Button
						variant="contained"
						color="primary"
						startIcon={<AddIcon />}
						onClick={handleOpenCreateDialog}
					>
						Create Mapping
					</Button>
				</Box>
			</Box>

			{error && (
				<Alert severity="error" sx={{ mb: 3 }}>
					{error}
				</Alert>
			)}

			<TableContainer component={Paper} elevation={2}>
				{loading ? (
					<Box sx={{ display: 'flex', justifyContent: 'center', p: 5 }}>
						<CircularProgress />
					</Box>
				) : (
					<Table sx={{ minWidth: 650 }}>
						<TableHead sx={{ bgcolor: 'action.hover' }}>
							<TableRow>
								<TableCell sx={{ fontWeight: 'bold' }}>Tenant Name</TableCell>
								<TableCell sx={{ fontWeight: 'bold' }}>IDP Client ID</TableCell>
								<TableCell sx={{ fontWeight: 'bold' }}>IDP Group ID / UUID</TableCell>
								<TableCell sx={{ fontWeight: 'bold' }}>Functional Role</TableCell>
								<TableCell sx={{ fontWeight: 'bold' }}>Clearance Level</TableCell>
								<TableCell sx={{ fontWeight: 'bold' }}>Created At</TableCell>
								<TableCell sx={{ fontWeight: 'bold' }} align="right">Actions</TableCell>
							</TableRow>
						</TableHead>
						<TableBody>
							{mappings.length === 0 ? (
								<TableRow>
									<TableCell colSpan={7} align="center" sx={{ py: 3, color: 'text.secondary' }}>
										No identity provider mappings configured.
									</TableCell>
								</TableRow>
							) : (
								mappings.map((row) => (
									<TableRow key={row.mapping_id} hover>
										<TableCell>{row.tenant_name}</TableCell>
										<TableCell><code>{row.idp_client_id}</code></TableCell>
										<TableCell><code>{row.idp_group_id}</code></TableCell>
										<TableCell>{row.functional_role}</TableCell>
										<TableCell>{row.clearance_level}</TableCell>
										<TableCell>{new Date(row.created_at).toLocaleString()}</TableCell>
										<TableCell align="right">
											<IconButton
												color="error"
												onClick={() => handleDelete(row.mapping_id)}
												title="Delete mapping"
											>
												<DeleteIcon />
											</IconButton>
										</TableCell>
									</TableRow>
								))
							)}
						</TableBody>
					</Table>
				)}
			</TableContainer>

			{/* Create Mapping Dialog */}
			<Dialog open={isCreating} onClose={handleCloseCreateDialog} maxWidth="sm" fullWidth>
				<form onSubmit={handleSubmit}>
					<DialogTitle>Create IDP Tenant Mapping</DialogTitle>
					<DialogContent>
						<Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5, mt: 1 }}>
							{validationError && <Alert severity="error">{validationError}</Alert>}

							<FormControl fullWidth>
								<InputLabel id="tenant-select-label">Tenant</InputLabel>
								<Select
									labelId="tenant-select-label"
									value={formTenantId}
									label="Tenant"
									onChange={(e) => setFormTenantId(e.target.value)}
								>
									{tenants.map((t) => (
										<MenuItem key={t.id} value={t.id}>
											{t.display_name || t.name}
										</MenuItem>
									))}
								</Select>
								<FormHelperText>Select the internal platform tenant to map to</FormHelperText>
							</FormControl>

							<TextField
								label="IDP Client ID (azp)"
								variant="outlined"
								value={formClientId}
								onChange={(e) => setFormClientId(e.target.value)}
								fullWidth
								helperText="OIDC Client ID (e.g. semlayer-frontend)"
							/>

							<TextField
								label="IDP Group ID / UUID"
								variant="outlined"
								value={formGroupId}
								onChange={(e) => setFormGroupId(e.target.value)}
								fullWidth
								helperText="The directory group UUID (e.g. keycloak group ID or Azure Object ID)"
							/>

							<FormControl fullWidth>
								<InputLabel id="role-select-label">Functional Role</InputLabel>
								<Select
									labelId="role-select-label"
									value={formRole}
									label="Functional Role"
									onChange={(e) => setFormRole(e.target.value)}
								>
									<MenuItem value="platform_analyst">Platform Analyst</MenuItem>
									<MenuItem value="platform_trader">Platform Trader</MenuItem>
									<MenuItem value="tenant_admin">Tenant Admin</MenuItem>
									<MenuItem value="global_admin">Global Admin</MenuItem>
								</Select>
								<FormHelperText>Internal ABAC role assigned to matching sessions</FormHelperText>
							</FormControl>

							<FormControl fullWidth>
								<InputLabel id="clearance-select-label">Clearance Level</InputLabel>
								<Select
									labelId="clearance-select-label"
									value={formClearance}
									label="Clearance Level"
									onChange={(e) => setFormClearance(e.target.value)}
								>
									<MenuItem value="L1">L1 - Basic</MenuItem>
									<MenuItem value="L2">L2 - Elevated</MenuItem>
									<MenuItem value="L3">L3 - Highly Restricted</MenuItem>
								</Select>
							</FormControl>
						</Box>
					</DialogContent>
					<DialogActions sx={{ p: 2.5 }}>
						<Button onClick={handleCloseCreateDialog}>Cancel</Button>
						<Button type="submit" variant="contained" color="primary">
							Save Mapping
						</Button>
					</DialogActions>
				</form>
			</Dialog>

			<Snackbar
				open={snackbarOpen}
				autoHideDuration={4000}
				onClose={() => setSnackbarOpen(false)}
			>
				<Alert
					onClose={() => setSnackbarOpen(false)}
					severity={snackbarSeverity}
					sx={{ width: '100%' }}
				>
					{snackbarMessage}
				</Alert>
			</Snackbar>
		</Box>
	);
};
