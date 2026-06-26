import React, { useState, useEffect } from 'react';
import {
  Box,
  Card,
  CardContent,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  TextField,
  InputAdornment,
  Button,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  Grid,
  CircularProgress,
  Toolbar,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import RefreshIcon from '@mui/icons-material/Refresh';
import InfoIcon from '@mui/icons-material/Info';
import ErrorIcon from '@mui/icons-material/Error';
import { useTenant } from '../../contexts/TenantContext';
import { devError } from '../../utils/devLogger';

interface APIEndpoint {
  id: string;
  endpoint_name: string;
  endpoint_path: string;
  http_method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';
  description: string;
  category: string;
  subcategory: string;
  request_schema: Record<string, unknown>;
  response_schema: Record<string, unknown>;
  request_examples: Record<string, unknown>[];
  response_examples: Record<string, unknown>[];
  is_active: boolean;
  version: string;
  deprecated: boolean;
  auth_required: boolean;
  rate_limit: number;
  created_at: string;
  updated_at: string;
}

const APIEndpointCatalogPage: React.FC = () => {
  const { tenant, datasource } = useTenant();

  const [endpoints, setEndpoints] = useState<APIEndpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedCategory, setSelectedCategory] = useState<string>('');
  const [selectedEndpoint, setSelectedEndpoint] = useState<APIEndpoint | null>(null);
  const [openDetailsDialog, setOpenDetailsDialog] = useState(false);
  const [categories, setCategories] = useState<string[]>([]);

  // Fetch API endpoints from backend
  useEffect(() => {
    fetchEndpoints();
  }, [tenant, datasource]);

  const fetchEndpoints = async () => {
    setLoading(true);
    setError(null);

    try {
      // Validate tenant scope
      if (!tenant?.id || !datasource?.id) {
        setError('Please select a tenant and datasource from Connections');
        setLoading(false);
        return;
      }

      const params = new URLSearchParams({
        tenant_id: tenant.id,
        tenant_instance_id: datasource.id,
      });

      const response = await fetch(
        `/api/api-endpoints?${params}`,
        {
          headers: {
            'X-Tenant-ID': tenant.id,
            'X-Tenant-Datasource-ID': datasource.id,
          },
        }
      );

      if (!response.ok) {
        throw new Error(`Failed to fetch endpoints: ${response.statusText}`);
      }

      const data = await response.json();
      const endpointList = Array.isArray(data) ? data : data.endpoints || [];

      setEndpoints(endpointList);

      // Extract unique categories
      const uniqueCategories = Array.from(
        new Set(endpointList.map((ep: APIEndpoint) => ep.category))
      ) as string[];
      setCategories(uniqueCategories.sort());

      if (endpointList.length === 0) {
        setError('No API endpoints found in catalog');
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'Failed to load API endpoints';
      setError(errorMessage);
      devError('Error fetching endpoints:', err);
    } finally {
      setLoading(false);
    }
  };

  // Filter endpoints based on search and category
  const filteredEndpoints = endpoints.filter((endpoint) => {
    const matchesSearch =
      endpoint.endpoint_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      endpoint.endpoint_path.toLowerCase().includes(searchTerm.toLowerCase()) ||
      endpoint.description.toLowerCase().includes(searchTerm.toLowerCase());

    const matchesCategory = !selectedCategory || endpoint.category === selectedCategory;

    return matchesSearch && matchesCategory;
  });

  const handleOpenDetails = (endpoint: APIEndpoint) => {
    setSelectedEndpoint(endpoint);
    setOpenDetailsDialog(true);
  };

  const handleCloseDetails = () => {
    setOpenDetailsDialog(false);
    setSelectedEndpoint(null);
  };

  const getMethodColor = (method: string): 'info' | 'success' | 'warning' | 'error' | 'default' => {
    switch (method) {
      case 'GET':
        return 'info';
      case 'POST':
        return 'success';
      case 'PUT':
        return 'warning';
      case 'DELETE':
        return 'error';
      case 'PATCH':
        return 'warning';
      default:
        return 'default';
    }
  };

  const getStatusVariant = (isActive: boolean): 'outlined' | 'filled' => (isActive ? 'filled' : 'outlined');

  return (
    <Box sx={{ p: 3 }}>
      {/* Header */}
      <Box sx={{ mb: 3 }}>
        <Typography variant="h4" gutterBottom>
          API Endpoint Catalog
        </Typography>
        <Typography variant="body2" color="textSecondary">
          Browse and manage all available API endpoints in your organization
        </Typography>
      </Box>

      {/* Tenant Scope Alert */}
      {!tenant?.id && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          <ErrorIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          Please select a tenant and datasource from Connections to view API endpoints.
        </Alert>
      )}

      {/* Error Alert */}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }}>
          <ErrorIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
          {error}
        </Alert>
      )}

      {/* Toolbar with Search and Filters */}
      <Paper sx={{ p: 2, mb: 3 }}>
        <Toolbar disableGutters sx={{ gap: 2, flexWrap: 'wrap' }}>
          {/* Search Field */}
          <TextField
            variant="outlined"
            placeholder="Search endpoints..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            InputProps={{
              startAdornment: (
                <InputAdornment position="start">
                  <SearchIcon />
                </InputAdornment>
              ),
            }}
            sx={{ flex: 1, minWidth: '250px' }}
          />

          {/* Category Filter */}
          {categories.length > 0 && (
            <FormControl sx={{ minWidth: '200px' }}>
              <InputLabel>Category</InputLabel>
              <Select
                value={selectedCategory}
                label="Category"
                onChange={(e) => setSelectedCategory(e.target.value)}
              >
                <MenuItem value="">All Categories</MenuItem>
                {categories.map((cat) => (
                  <MenuItem key={cat} value={cat}>
                    {cat}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
          )}

          {/* Refresh Button */}
          <Button
            startIcon={<RefreshIcon />}
            onClick={fetchEndpoints}
            disabled={loading}
            variant="outlined"
          >
            Refresh
          </Button>
        </Toolbar>
      </Paper>

      {/* Results Summary */}
      <Box sx={{ mb: 2 }}>
        <Typography variant="body2" color="textSecondary">
          Showing {filteredEndpoints.length} of {endpoints.length} endpoints
        </Typography>
      </Box>

      {/* Loading State */}
      {loading && (
        <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
          <CircularProgress />
        </Box>
      )}

      {/* Endpoints Table */}
      {!loading && filteredEndpoints.length > 0 && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
                <TableCell sx={{ fontWeight: 'bold' }}>Method</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Endpoint</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Description</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Category</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Version</TableCell>
                <TableCell sx={{ fontWeight: 'bold' }}>Status</TableCell>
                <TableCell align="right" sx={{ fontWeight: 'bold' }}>
                  Actions
                </TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredEndpoints.map((endpoint) => (
                <TableRow key={endpoint.id} hover>
                  <TableCell>
                    <Chip
                      label={endpoint.http_method}
                      color={getMethodColor(endpoint.http_method)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                      {endpoint.endpoint_path}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2" sx={{ maxWidth: '300px', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                      {endpoint.description}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={endpoint.category}
                      variant="outlined"
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">{endpoint.version}</Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={endpoint.is_active ? 'Active' : 'Inactive'}
                      color={endpoint.is_active ? 'success' : 'default'}
                      variant={getStatusVariant(endpoint.is_active)}
                      size="small"
                    />
                    {endpoint.deprecated && (
                      <Chip
                        label="Deprecated"
                        color="error"
                        variant="outlined"
                        size="small"
                        sx={{ ml: 1 }}
                      />
                    )}
                  </TableCell>
                  <TableCell align="right">
                    <Button
                      size="small"
                      startIcon={<InfoIcon />}
                      onClick={() => handleOpenDetails(endpoint)}
                    >
                      Details
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Empty State */}
      {!loading && filteredEndpoints.length === 0 && endpoints.length === 0 && (
        <Card>
          <CardContent sx={{ textAlign: 'center', p: 4 }}>
            <InfoIcon sx={{ fontSize: 48, color: 'textSecondary', mb: 2 }} />
            <Typography variant="h6" gutterBottom>
              No API Endpoints Found
            </Typography>
            <Typography variant="body2" color="textSecondary">
              The API endpoint catalog is empty. Create endpoints to see them here.
            </Typography>
          </CardContent>
        </Card>
      )}

      {/* No Results State */}
      {!loading && filteredEndpoints.length === 0 && endpoints.length > 0 && (
        <Card>
          <CardContent sx={{ textAlign: 'center', p: 4 }}>
            <SearchIcon sx={{ fontSize: 48, color: 'textSecondary', mb: 2 }} />
            <Typography variant="h6" gutterBottom>
              No Endpoints Match Your Search
            </Typography>
            <Typography variant="body2" color="textSecondary">
              Try adjusting your search terms or category filter.
            </Typography>
          </CardContent>
        </Card>
      )}

      {/* Details Dialog */}
      <Dialog open={openDetailsDialog} onClose={handleCloseDetails} maxWidth="md" fullWidth>
        <DialogTitle>
          {selectedEndpoint && (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Chip
                label={selectedEndpoint.http_method}
                color={getMethodColor(selectedEndpoint.http_method)}
                size="small"
              />
              <Typography variant="h6">{selectedEndpoint.endpoint_name}</Typography>
            </Box>
          )}
        </DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {selectedEndpoint && (
            <Grid container spacing={2}>
              {/* Basic Info */}
              <Grid item xs={12}>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1 }}>
                  Basic Information
                </Typography>
                <Box sx={{ backgroundColor: '#f5f5f5', p: 2, borderRadius: 1 }}>
                  <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="textSecondary">
                        Path
                      </Typography>
                      <Typography variant="body2" sx={{ fontFamily: 'monospace', mt: 0.5 }}>
                        {selectedEndpoint.endpoint_path}
                      </Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="textSecondary">
                        Method
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 0.5 }}>
                        <Chip
                          label={selectedEndpoint.http_method}
                          color={getMethodColor(selectedEndpoint.http_method)}
                          size="small"
                        />
                      </Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="textSecondary">
                        Version
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 0.5 }}>
                        {selectedEndpoint.version}
                      </Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="textSecondary">
                        Category
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 0.5 }}>
                        <Chip label={selectedEndpoint.category} variant="outlined" size="small" />
                      </Typography>
                    </Grid>
                    <Grid item xs={12}>
                      <Typography variant="caption" color="textSecondary">
                        Description
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 0.5 }}>
                        {selectedEndpoint.description}
                      </Typography>
                    </Grid>
                  </Grid>
                </Box>
              </Grid>

              {/* Status Information */}
              <Grid item xs={12}>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1 }}>
                  Status
                </Typography>
                <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
                  <Chip
                    label={selectedEndpoint.is_active ? 'Active' : 'Inactive'}
                    color={selectedEndpoint.is_active ? 'success' : 'default'}
                    variant={getStatusVariant(selectedEndpoint.is_active)}
                  />
                  {selectedEndpoint.deprecated && (
                    <Chip label="Deprecated" color="error" />
                  )}
                  {selectedEndpoint.auth_required && (
                    <Chip label="Auth Required" color="warning" />
                  )}
                </Box>
              </Grid>

              {/* Request/Response Schemas */}
              <Grid item xs={12}>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1 }}>
                  Request & Response
                </Typography>
                <Box sx={{ backgroundColor: '#f5f5f5', p: 2, borderRadius: 1, fontFamily: 'monospace', fontSize: '0.75rem', overflow: 'auto', maxHeight: '300px' }}>
                  <Typography variant="caption" sx={{ fontWeight: 'bold' }}>
                    Request Schema:
                  </Typography>
                  <Box component="pre" sx={{ margin: '8px 0' }}>
                    {JSON.stringify(selectedEndpoint.request_schema, null, 2)}
                  </Box>
                  <Typography variant="caption" sx={{ fontWeight: 'bold' }}>
                    Response Schema:
                  </Typography>
                  <Box component="pre" sx={{ margin: '8px 0' }}>
                    {JSON.stringify(selectedEndpoint.response_schema, null, 2)}
                  </Box>
                </Box>
              </Grid>

              {/* Metadata */}
              <Grid item xs={12}>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold', mb: 1 }}>
                  Metadata
                </Typography>
                <Box sx={{ backgroundColor: '#f5f5f5', p: 2, borderRadius: 1 }}>
                  <Grid container spacing={2}>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="textSecondary">
                        Rate Limit
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 0.5 }}>
                        {selectedEndpoint.rate_limit} requests/min
                      </Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="textSecondary">
                        Subcategory
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 0.5 }}>
                        {selectedEndpoint.subcategory}
                      </Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="textSecondary">
                        Created
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 0.5 }}>
                        {new Date(selectedEndpoint.created_at).toLocaleDateString()}
                      </Typography>
                    </Grid>
                    <Grid item xs={12} sm={6}>
                      <Typography variant="caption" color="textSecondary">
                        Updated
                      </Typography>
                      <Typography variant="body2" sx={{ mt: 0.5 }}>
                        {new Date(selectedEndpoint.updated_at).toLocaleDateString()}
                      </Typography>
                    </Grid>
                  </Grid>
                </Box>
              </Grid>
            </Grid>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleCloseDetails}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default APIEndpointCatalogPage;
