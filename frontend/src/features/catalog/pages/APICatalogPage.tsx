import { useState, useEffect } from 'react';
import { devError, devWarn } from '../../../utils/devLogger';
import {
  Box, Card, CardContent, Typography, Grid, Chip, Button,
  Dialog, DialogContent, DialogActions,
  Table, TableBody, TableCell, TableContainer, TableHead, TableRow,
  Paper, TextField, InputAdornment, Tabs, Tab
} from '@mui/material';
import ModalHeader from '../../../components/ModalHeader';
import SearchIcon from '@mui/icons-material/Search';
import RefreshIcon from '@mui/icons-material/Refresh';
import ApiIcon from '@mui/icons-material/Api';
import LinkIcon from '@mui/icons-material/Link';
import BusinessIcon from '@mui/icons-material/Business';
import { LineageGraph } from '../../../LineageGraph';

interface APIEndpoint {
  id: string;
  path: string;
  method: string;
  description: string;
  category: string;
  service: string;
  version: string;
  status: 'active' | 'deprecated' | 'beta';
  lastUpdated: string;
  businessTerms: string[];
  dependencies: string[];
}

interface BusinessTerm {
  id: string;
  name: string;
  description: string;
  category: string;
  owner: string;
  status: string;
  relatedAPIs: string[];
}

const APICatalogPage: React.FC = () => {
  const [apis, setApis] = useState<APIEndpoint[]>([]);
  const [businessTerms, setBusinessTerms] = useState<BusinessTerm[]>([]);
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedAPI, setSelectedAPI] = useState<APIEndpoint | null>(null);
  const [selectedBusinessTerm, setSelectedBusinessTerm] = useState<BusinessTerm | null>(null);
  const [activeTab, setActiveTab] = useState(0);
  const [loading, setLoading] = useState(false);

  // Mock data - replace with actual API calls
  useEffect(() => {
    loadAPIs();
    loadBusinessTerms();
  }, []);

  const loadAPIs = async () => {
    setLoading(true);
    try {
      // Replace with actual API call to your backend
      const response = await fetch('/api/catalog/apis');
      const data = await response.json();
      // Defensive normalization: backend may return { apis: [...] } or an object wrapper
      if (Array.isArray(data)) {
        setApis(data);
      } else if (data && Array.isArray(data.apis)) {
        setApis(data.apis);
      } else if (data && Array.isArray(data.items)) {
        setApis(data.items);
      } else {
        // Unexpected shape
        devWarn('Unexpected APIs response shape, expected array. Using empty list.', data);
        setApis([]);
      }
    } catch (error) {
      devError('Failed to load APIs:', error);
      // Mock data for demonstration
      setApis([
        {
          id: '1',
          path: '/api/search/business-terms',
          method: 'POST',
          description: 'Search for business terms in the catalog',
          category: 'Business Terms',
          service: 'API Gateway',
          version: 'v1.0',
          status: 'active',
          lastUpdated: '2024-01-15',
          businessTerms: ['customer_id', 'order_value'],
          dependencies: ['hasura', 'postgres']
        },
        {
          id: '2',
          path: '/api/validate/business-term',
          method: 'POST',
          description: 'Validate business term definitions',
          category: 'Business Terms',
          service: 'API Gateway',
          version: 'v1.0',
          status: 'active',
          lastUpdated: '2024-01-15',
          businessTerms: ['customer_id'],
          dependencies: ['hasura']
        }
      ]);
    } finally {
      setLoading(false);
    }
  };

  const loadBusinessTerms = async () => {
    try {
      // Replace with actual API call
      const response = await fetch('/api/business-terms');
      const data = await response.json();
      if (Array.isArray(data)) {
        setBusinessTerms(data);
      } else if (data && Array.isArray(data.items)) {
        setBusinessTerms(data.items);
      } else if (data && Array.isArray(data.terms)) {
        setBusinessTerms(data.terms);
      } else {
        devWarn('Unexpected business terms response shape, expected array. Using empty list.', data);
        setBusinessTerms([]);
      }
    } catch (error) {
      devError('Failed to load business terms:', error);
      // Mock data
      setBusinessTerms([
        {
          id: 'customer_id',
          name: 'Customer ID',
          description: 'Unique identifier for customers',
          category: 'Customer Data',
          owner: 'Data Team',
          status: 'approved',
          relatedAPIs: ['1', '2']
        }
      ]);
    }
  };

  const safeApis = Array.isArray(apis) ? apis : [];
  const filteredAPIs = safeApis.filter(api =>
    (api.path || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
    (api.description || '').toLowerCase().includes(searchTerm.toLowerCase()) ||
    (api.category || '').toLowerCase().includes(searchTerm.toLowerCase())
  );

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active': return 'success';
      case 'deprecated': return 'error';
      case 'beta': return 'warning';
      default: return 'default';
    }
  };

  const handleTabChange = (_event: React.SyntheticEvent, newValue: number) => {
    setActiveTab(newValue);
  };

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        API Catalog & Business Terms
      </Typography>

      <Box sx={{ mb: 3 }}>
        <TextField
          fullWidth
          variant="outlined"
          placeholder="Search APIs, business terms, or categories..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Box>

      <Box sx={{ mb: 3 }}>
        <Button
          startIcon={<RefreshIcon />}
          onClick={() => {
            loadAPIs();
            loadBusinessTerms();
          }}
          disabled={loading}
        >
          Refresh Catalog
        </Button>
      </Box>

      <Tabs value={activeTab} onChange={handleTabChange} sx={{ mb: 3 }}>
        <Tab icon={<ApiIcon />} label="API Endpoints" />
        <Tab icon={<BusinessIcon />} label="Business Terms" />
        <Tab icon={<LinkIcon />} label="Lineage View" />
      </Tabs>

      {activeTab === 0 && (
        <Grid container spacing={3}>
          {filteredAPIs.map((api) => (
            <Grid item xs={12} md={6} lg={4} key={api.id}>
              <Card sx={{ height: '100%', cursor: 'pointer' }} onClick={() => setSelectedAPI(api)}>
                <CardContent>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Typography variant="h6" component="div">
                      {api.method} {api.path}
                    </Typography>
                    <Chip
                      label={api.status}
                      color={getStatusColor(api.status)}
                      size="small"
                    />
                  </Box>

                  <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                    {api.description}
                  </Typography>

                  <Box sx={{ mb: 1 }}>
                    <Typography variant="caption" color="text.secondary">
                      Service: {api.service} | Version: {api.version}
                    </Typography>
                  </Box>

                  <Box sx={{ mb: 1 }}>
                    <Typography variant="caption" color="text.secondary">
                      Category: {api.category}
                    </Typography>
                  </Box>

                  <Typography variant="caption" color="text.secondary">
                    Updated: {api.lastUpdated}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {activeTab === 1 && (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>Name</TableCell>
                <TableCell>Description</TableCell>
                <TableCell>Category</TableCell>
                <TableCell>Owner</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Related APIs</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {businessTerms.map((term) => (
                <TableRow
                  key={term.id}
                  hover
                  sx={{ cursor: 'pointer' }}
                  onClick={() => setSelectedBusinessTerm(term)}
                >
                  <TableCell>{term.name}</TableCell>
                  <TableCell>{term.description}</TableCell>
                  <TableCell>{term.category}</TableCell>
                  <TableCell>{term.owner}</TableCell>
                  <TableCell>
                    <Chip
                      label={term.status}
                      color={getStatusColor(term.status)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{term.relatedAPIs.length} APIs</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {activeTab === 2 && (
        <Box sx={{ height: '70vh' }}>
          <LineageGraph
            apis={apis}
            businessTerms={businessTerms}
            onNodeClick={(node) => {
              if (node.type === 'api') {
                setSelectedAPI(apis.find(api => api.id === node.id) || null);
              } else {
                setSelectedBusinessTerm(businessTerms.find(term => term.id === node.id) || null);
              }
            }}
          />
        </Box>
      )}

      {/* API Details Dialog */}
      <Dialog
        open={!!selectedAPI}
        onClose={() => setSelectedAPI(null)}
        maxWidth="md"
        fullWidth
      >
        <ModalHeader
          title={<>{selectedAPI?.method} {selectedAPI?.path}</>}
          onClose={() => setSelectedAPI(null)}
        />
        <DialogContent>
          {selectedAPI && (
            <Box>
              <Typography variant="h6" gutterBottom>Description</Typography>
              <Typography paragraph>{selectedAPI.description}</Typography>

              <Typography variant="h6" gutterBottom>Details</Typography>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography><strong>Service:</strong> {selectedAPI.service}</Typography>
                  <Typography><strong>Version:</strong> {selectedAPI.version}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography><strong>Category:</strong> {selectedAPI.category}</Typography>
                  <Typography><strong>Status:</strong> {selectedAPI.status}</Typography>
                </Grid>
              </Grid>

              <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>Business Terms</Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {selectedAPI.businessTerms.map((term) => (
                  <Chip key={term} label={term} size="small" />
                ))}
              </Box>

              <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>Dependencies</Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {selectedAPI.dependencies.map((dep) => (
                  <Chip key={dep} label={dep} size="small" variant="outlined" />
                ))}
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSelectedAPI(null)}>Close</Button>
        </DialogActions>
      </Dialog>

      {/* Business Term Details Dialog */}
      <Dialog
        open={!!selectedBusinessTerm}
        onClose={() => setSelectedBusinessTerm(null)}
        maxWidth="md"
        fullWidth
      >
        <ModalHeader
          title={selectedBusinessTerm?.name}
          onClose={() => setSelectedBusinessTerm(null)}
        />
        <DialogContent>
          {selectedBusinessTerm && (
            <Box>
              <Typography variant="h6" gutterBottom>Description</Typography>
              <Typography paragraph>{selectedBusinessTerm.description}</Typography>

              <Typography variant="h6" gutterBottom>Details</Typography>
              <Grid container spacing={2}>
                <Grid item xs={6}>
                  <Typography><strong>Category:</strong> {selectedBusinessTerm.category}</Typography>
                  <Typography><strong>Owner:</strong> {selectedBusinessTerm.owner}</Typography>
                </Grid>
                <Grid item xs={6}>
                  <Typography><strong>Status:</strong> {selectedBusinessTerm.status}</Typography>
                  <Typography><strong>Related APIs:</strong> {selectedBusinessTerm.relatedAPIs.length}</Typography>
                </Grid>
              </Grid>

              <Typography variant="h6" gutterBottom sx={{ mt: 2 }}>Related API Endpoints</Typography>
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1 }}>
                {selectedBusinessTerm.relatedAPIs.map((apiId) => {
                  const api = apis.find(a => a.id === apiId);
                  return api ? (
                    <Chip
                      key={apiId}
                      label={`${api.method} ${api.path}`}
                      size="small"
                      onClick={() => {
                        setSelectedBusinessTerm(null);
                        setSelectedAPI(api);
                      }}
                      sx={{ cursor: 'pointer' }}
                    />
                  ) : null;
                })}
              </Box>
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setSelectedBusinessTerm(null)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default APICatalogPage;
