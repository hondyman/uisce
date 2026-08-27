import React, { useState, useEffect } from 'react';
import {
  Box,
  Typography,
  Grid,
  Card,
  CardContent,
  CardActions,
  Button,
  Chip,
  TextField,
  InputAdornment,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Rating,
  Alert,
  Snackbar,
  CircularProgress,
  Stack,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import StorefrontIcon from '@mui/icons-material/Storefront';
import AddCircleOutlineIcon from '@mui/icons-material/AddCircleOutline';
import VerifiedIcon from '@mui/icons-material/Verified';
import DownloadIcon from '@mui/icons-material/Download';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import { apiClient } from '../../../utils/apiClient';

interface MarketplaceItem {
  id: string;
  title: string;
  kind: string;
  category: string;
  publisher_tenant_id: string;
  publisher_name: string;
  description: string;
  price_cents: number;
  billing_period: string;
  rating: number;
  installs_count: number;
  artifact_type?: string;
  artifact_version?: string;
  created_at: string;
}

interface ProductEvolutionData {
  recommendations: Array<{
    cluster_id: string;
    sample_name: string;
    entity_type: string;
    tenant_count: number;
    recommendation: string;
    confidence_score: number;
    detected_at: string;
  }>;
}

const KINDS = ['All', 'rbac_role', 'compliance_rule_pack', 'abac_policy', 'analytics_dashboard', 'integration', 'bundle'];

export const UisceMarketplacePage: React.FC = () => {
  const [items, setItems] = useState<MarketplaceItem[]>([]);
  const [loading, setLoading] = useState<boolean>(true);
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [selectedKind, setSelectedKind] = useState<string>('All');
  const [evolutionData, setEvolutionData] = useState<ProductEvolutionData | null>(null);
  
  // Publish Modal state
  const [publishOpen, setPublishOpen] = useState<boolean>(false);
  const [publishTitle, setPublishTitle] = useState<string>('');
  const [publishCategory, setPublishCategory] = useState<string>('RBAC');
  const [publishDesc, setPublishDesc] = useState<string>('');
  const [publishPrice, setPublishPrice] = useState<number>(0);
  const [publishRoleKey, setPublishRoleKey] = useState<string>('');
  
  // Toast feedback
  const [snackbarMessage, setSnackbarMessage] = useState<string | null>(null);

  useEffect(() => {
    fetchMarketplaceItems();
    fetchProductEvolution();
  }, []);

  const fetchMarketplaceItems = async () => {
    try {
      setLoading(true);
      const res = await apiClient<{ listings: MarketplaceItem[] }>('/api/marketplace/browse?limit=100&offset=0');
      setItems(res?.listings ?? []);
    } catch (err) {
      console.error('Failed to load marketplace listings:', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchProductEvolution = async () => {
    try {
      const res = await apiClient<ProductEvolutionData>('/api/marketplace/product-evolution?min_confidence=0.5');
      setEvolutionData(res);
    } catch (err) {
      console.error('Failed to load product evolution recommendations:', err);
    }
  };

  const handleInstall = async (item: MarketplaceItem) => {
    try {
      await apiClient(`/api/marketplace/${item.id}/install`, { method: 'POST' });
      setSnackbarMessage(`Successfully installed "${item.title}" into your tenant!`);
      fetchMarketplaceItems();
    } catch (err) {
      setSnackbarMessage(`Failed to install ${item.title}: ${String(err)}`);
    }
  };

  const handlePublishSubmit = async () => {
    if (!publishTitle || !publishRoleKey) return;
    try {
      await apiClient('/api/marketplace/publish', {
        method: 'POST',
        body: JSON.stringify({
          title: publishTitle,
          kind: publishCategory.toLowerCase().replace(' ', '_'),
          category: publishCategory,
          description: publishDesc,
          price_cents: publishPrice * 100,
          billing_period: 'monthly',
          role_key: publishRoleKey,
        }),
      });
      setPublishOpen(false);
      setSnackbarMessage(`Successfully published "${publishTitle}" to the Uisce Store!`);
      fetchMarketplaceItems();
    } catch (err) {
      setSnackbarMessage(`Failed to publish item: ${String(err)}`);
    }
  };

  const filteredItems = items.filter((item) => {
    const matchesKind = selectedKind === 'All' || item.kind === selectedKind;
    const matchesSearch = item.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
                          item.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesKind && matchesSearch;
  });

  return (
    <Box sx={{ p: 4, backgroundColor: '#f8fafc', minHeight: '100vh' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <StorefrontIcon sx={{ fontSize: 36, color: '#2563eb' }} />
          <Box>
            <Typography variant="h4" fontWeight={700} color="#0f172a">
              Uisce Marketplace Ecosystem
            </Typography>            <Typography variant="body2" color="#64748b">
              Browse, publish, and install enterprise compliance rule packs, ABAC policies, and custom roles.
            </Typography>
          </Box>
        </Box>
        <Button
          variant="contained"
          startIcon={<AddCircleOutlineIcon />}
          onClick={() => setPublishOpen(true)}
          sx={{ backgroundColor: '#2563eb', textTransform: 'none', px: 3, py: 1.2, fontWeight: 600 }}
        >
          Publish Customization
        </Button>
      </Box>

      {/* AI Product Evolution Recommendation Alert */}
      {evolutionData && evolutionData.recommendations && evolutionData.recommendations.length > 0 && (
        <Alert
          severity="info"
          icon={<AutoAwesomeIcon sx={{ color: '#0284c7' }} />}
          sx={{ mb: 4, backgroundColor: '#e0f2fe', border: '1px solid #bae6fd', borderRadius: 2 }}
        >
          <Typography variant="subtitle2" fontWeight={700} color="#0369a1">
            Customization Intelligence Alert: Gold Copy Candidate Detected
          </Typography>
          <Typography variant="body2" color="#0c4a6e" sx={{ mt: 0.5 }}>
            {evolutionData.recommendations[0].recommendation}{' '}
            (confidence: {(evolutionData.recommendations[0].confidence_score * 100).toFixed(0)}%)
          </Typography>
        </Alert>
      )}

      {/* Search & Filter Bar */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 4, flexWrap: 'wrap', gap: 2 }}>
        <TextField
          placeholder="Search rule packs, roles, ABAC policies..."
          variant="outlined"
          size="small"
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: '#94a3b8' }} />
              </InputAdornment>
            ),
          }}
          sx={{ width: 360, backgroundColor: '#ffffff', borderRadius: 1 }}
        />

        <Stack direction="row" spacing={1}>
          {KINDS.map((kind) => (
            <Chip
              key={kind}
              label={kind === 'All' ? 'All' : kind.replace('_', ' ')}
              clickable
              onClick={() => setSelectedKind(kind)}
              color={selectedKind === kind ? 'primary' : 'default'}
              variant={selectedKind === kind ? 'filled' : 'outlined'}
              sx={{ fontWeight: 600, px: 1 }}
            />
          ))}
        </Stack>
      </Box>

      {/* Marketplace Grid */}
      {loading ? (
        <Box sx={{ display: 'flex', justifyContent: 'center', py: 8 }}>
          <CircularProgress />
        </Box>
      ) : filteredItems.length === 0 ? (
        <Typography variant="body1" color="#64748b" align="center" sx={{ py: 8 }}>
          No listings found matching your filter criteria.
        </Typography>
      ) : (
        <Grid container spacing={3}>
          {filteredItems.map((item) => (
            <Grid    key={item.id} size={{ xs: 12, sm: 6, md: 4 }}>
              <Card
                sx={{
                  height: '100%',
                  display: 'flex',
                  flexDirection: 'column',
                  borderRadius: 3,
                  boxShadow: '0 4px 6px -1px rgba(0, 0, 0, 0.05), 0 2px 4px -1px rgba(0, 0, 0, 0.03)',
                  border: '1px solid #e2e8f0',
                  transition: 'transform 0.2s, box-shadow 0.2s',
                  '&:hover': {
                    transform: 'translateY(-4px)',
                    boxShadow: '0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05)',
                  },
                }}
              >
                <CardContent sx={{ flexGrow: 1, p: 3 }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 1.5 }}>
                    <Box sx={{ display: 'flex', gap: 0.5 }}>
                      <Chip
                        label={item.kind.replace('_', ' ')}
                        size="small"
                        sx={{ backgroundColor: '#eff6ff', color: '#1d4ed8', fontWeight: 700 }}
                      />
                      {item.billing_period && (
                        <Chip
                          label={item.billing_period}
                          size="small"
                          sx={{ backgroundColor: '#f0fdf4', color: '#15803d', fontWeight: 600 }}
                        />
                      )}
                    </Box>
                    <Typography variant="h6" fontWeight={700} color={item.price_cents === 0 ? '#16a34a' : '#0f172a'}>
                      {item.price_cents === 0 ? 'Free' : `$${(item.price_cents / 100).toFixed(2)}`}
                    </Typography>
                  </Box>

                  <Typography variant="h6" fontWeight={700} color="#0f172a" sx={{ mb: 1 }}>
                    {item.title}
                  </Typography>

                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 2 }}>
                    <VerifiedIcon sx={{ fontSize: 16, color: '#2563eb' }} />
                    <Typography variant="caption" color="#64748b" fontWeight={500}>
                      {item.publisher_name}
                    </Typography>
                  </Box>

                  <Typography variant="body2" color="#475569" sx={{ mb: 2, minHeight: 40 }}>
                    {item.description}
                  </Typography>

                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <Rating value={item.rating} precision={0.1} readOnly size="small" />
                    <Typography variant="caption" color="#64748b">
                      {item.installs_count} installs
                    </Typography>
                  </Box>
                </CardContent>

                <CardActions sx={{ p: 2, pt: 0, borderTop: '1px solid #f1f5f9' }}>
                  <Button
                    fullWidth
                    variant="contained"
                    startIcon={<DownloadIcon />}
                    onClick={() => handleInstall(item)}
                    sx={{ backgroundColor: '#0f172a', textTransform: 'none', fontWeight: 600, '&:hover': { backgroundColor: '#1e293b' } }}
                  >
                    1-Click Install
                  </Button>
                </CardActions>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}

      {/* Publish Customization Modal Wizard */}
      <Dialog open={publishOpen} onClose={() => setPublishOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>Publish Customization to Marketplace</DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2.5, pt: 1 }}>
            <TextField
              label="Listing Title"
              fullWidth
              value={publishTitle}
              onChange={(e) => setPublishTitle(e.target.value)}
              placeholder="e.g. SOX Compliance Controller Pack"
            />
            <TextField
              label="Role Key / Artifact Name"
              fullWidth
              value={publishRoleKey}
              onChange={(e) => setPublishRoleKey(e.target.value)}
              placeholder="e.g. sox_compliance_controller"
            />
            <TextField
              label="Kind"
              select
              SelectProps={{ native: true }}
              value={publishCategory}
              onChange={(e) => setPublishCategory(e.target.value)}
            >
              <option value="RBAC">RBAC Role</option>
              <option value="Compliance">Compliance Rule Pack</option>
              <option value="ABAC">ABAC Policy</option>
              <option value="Analytics">Analytics Dashboard</option>
              <option value="Integration">Integration</option>
              <option value="Bundle">Bundle</option>
            </TextField>
            <TextField
              label="Price (USD)"
              type="number"
              fullWidth
              value={publishPrice}
              onChange={(e) => setPublishPrice(Number(e.target.value))}
              helperText="Set to 0 for Free"
            />
            <TextField
              label="Description"
              multiline
              rows={3}
              fullWidth
              value={publishDesc}
              onChange={(e) => setPublishDesc(e.target.value)}
              placeholder="Provide context on what permissions or compliance rules this package enforces..."
            />
          </Box>
        </DialogContent>
        <DialogActions sx={{ p: 2.5 }}>
          <Button onClick={() => setPublishOpen(false)}>Cancel</Button>
          <Button variant="contained" onClick={handlePublishSubmit} sx={{ backgroundColor: '#2563eb' }}>
            Publish Listing
          </Button>
        </DialogActions>
      </Dialog>

      {/* Snackbar Notification Toast */}
      <Snackbar
        open={Boolean(snackbarMessage)}
        autoHideDuration={4000}
        onClose={() => setSnackbarMessage(null)}
        message={snackbarMessage}
      />
    </Box>
  );
};

export default UisceMarketplacePage;
