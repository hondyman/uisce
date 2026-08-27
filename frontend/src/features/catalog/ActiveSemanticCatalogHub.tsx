import React, { useState } from 'react';
import {
  Box,
  Paper,
  Typography,
  Stack,
  Button,
  Chip,
  Grid,
  Card,
  CardActionArea,
  InputAdornment,
  TextField,
  LinearProgress,
  Divider
} from '@mui/material';
import {
  MenuBook as BusinessTermIcon,
  Lightbulb as SemanticTermIcon,
  Storage as BusinessObjectIcon,
  Search as SearchIcon,
  AutoAwesome as AiIcon,
  AltRoute as LineageIcon,
  Shield as GovernanceIcon,
  ArrowForward as ArrowForwardIcon
} from '@mui/icons-material';

interface ActiveSemanticCatalogHubProps {
  tenantId?: string;
  onNavigate?: (view: 'business_terms' | 'semantic_terms' | 'business_objects' | 'lineage_simulator') => void;
}

export const ActiveSemanticCatalogHub: React.FC<ActiveSemanticCatalogHubProps> = ({
  tenantId: _tenantId = '99e99e99-99e9-49e9-89e9-99e99e99e999',
  onNavigate = () => {}
}) => {
  const [searchQuery, setSearchQuery] = useState('');

  return (
    <Box sx={{ p: 4, bgcolor: '#071526', minHeight: '100vh', color: '#F8FAFC' }}>
      <Stack spacing={3} mb={5} alignItems="center" textAlign="center">
        <Chip
          icon={<AiIcon sx={{ fontSize: 14, color: '#00D4FF !important' }} />}
          label="Active Semantic Operating System • 2026.4"
          size="small"
          sx={{ bgcolor: '#0B1E36', color: '#00D4FF', fontWeight: 700, fontSize: 11, border: '1px solid #1E293B' }}
        />
        
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 800, color: '#F8FAFC', letterSpacing: '-0.5px' }}>
            Enterprise Semantic Catalog
          </Typography>
          <Typography variant="body1" sx={{ color: '#94A3B8', mt: 1, maxWidth: 650 }}>
            Unified business ontology, physical data bindings, and self-healing lineage across all institutional backends.
          </Typography>
        </Box>

        <TextField
          fullWidth
          placeholder="Search business terms, physical columns, business objects, or attested calculations..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          sx={{
            maxWidth: 720,
            bgcolor: '#0B1E36',
            borderRadius: 2,
            '& .MuiOutlinedInput-root': {
              color: '#F8FAFC',
              '& fieldset': { borderColor: '#1E293B' },
              '&:hover fieldset': { borderColor: '#0284C7' },
              '&.Mui-focused fieldset': { borderColor: '#00D4FF' }
            }
          }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon sx={{ color: '#64748B' }} />
              </InputAdornment>
            ),
            endAdornment: (
              <InputAdornment position="end">
                <Chip label="⌘K" size="small" sx={{ bgcolor: '#071526', color: '#94A3B8', fontSize: 10, fontWeight: 700 }} />
              </InputAdornment>
            )
          }}
        />
      </Stack>

      <Grid container spacing={3} mb={5}>
        <Grid   size={{ xs: 12, md: 4 }}>
          <Card
            sx={{
              bgcolor: '#0B1E36',
              border: '1px solid #1E293B',
              borderRadius: 2,
              height: '100%',
              transition: 'all 0.2s',
              '&:hover': { borderColor: '#00D4FF', transform: 'translateY(-2px)' }
            }}
          >
            <CardActionArea onClick={() => onNavigate('business_terms')} sx={{ p: 3, height: '100%' }}>
              <Stack spacing={2}>
                <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                  <Box sx={{ p: 1.5, bgcolor: '#071526', borderRadius: 1.5, border: '1px solid #1E293B' }}>
                    <BusinessTermIcon sx={{ color: '#38BDF8', fontSize: 28 }} />
                  </Box>
                  <Chip label="1,420 Terms" size="small" sx={{ bgcolor: '#071526', color: '#38BDF8', fontSize: 11, fontWeight: 700 }} />
                </Box>

                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
                    Business Terms
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#94A3B8', mt: 0.5, minHeight: 40 }}>
                    Defined organizational vocabulary, Level-3 taxonomy, and standardized governance definitions.
                  </Typography>
                </Box>

                <Divider sx={{ borderColor: '#1E293B' }} />

                <Stack direction="row" justifyContent="space-between" alignItems="center">
                  <Typography variant="caption" sx={{ color: '#64748B' }}>100% Taxonomically Bound</Typography>
                  <ArrowForwardIcon sx={{ color: '#38BDF8', fontSize: 16 }} />
                </Stack>
              </Stack>
            </CardActionArea>
          </Card>
        </Grid>

        <Grid   size={{ xs: 12, md: 4 }}>
          <Card
            sx={{
              bgcolor: '#0B1E36',
              border: '1px solid #1E293B',
              borderRadius: 2,
              height: '100%',
              transition: 'all 0.2s',
              '&:hover': { borderColor: '#00D4FF', transform: 'translateY(-2px)' }
            }}
          >
            <CardActionArea onClick={() => onNavigate('semantic_terms')} sx={{ p: 3, height: '100%' }}>
              <Stack spacing={2}>
                <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                  <Box sx={{ p: 1.5, bgcolor: '#071526', borderRadius: 1.5, border: '1px solid #1E293B' }}>
                    <SemanticTermIcon sx={{ color: '#FBBF24', fontSize: 28 }} />
                  </Box>
                  <Chip label="3,890 Bound" size="small" sx={{ bgcolor: '#071526', color: '#FBBF24', fontSize: 11, fontWeight: 700 }} />
                </Box>

                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
                    Semantic Terms
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#94A3B8', mt: 0.5, minHeight: 40 }}>
                    Conceptual execution nodes linking business meaning directly to physical tables and expressions.
                  </Typography>
                </Box>

                <Divider sx={{ borderColor: '#1E293B' }} />

                <Stack direction="row" justifyContent="space-between" alignItems="center">
                  <Typography variant="caption" sx={{ color: '#64748B' }}>MAPS_TO Physical Columns</Typography>
                  <ArrowForwardIcon sx={{ color: '#FBBF24', fontSize: 16 }} />
                </Stack>
              </Stack>
            </CardActionArea>
          </Card>
        </Grid>

        <Grid   size={{ xs: 12, md: 4 }}>
          <Card
            sx={{
              bgcolor: '#0B1E36',
              border: '1px solid #1E293B',
              borderRadius: 2,
              height: '100%',
              transition: 'all 0.2s',
              '&:hover': { borderColor: '#00D4FF', transform: 'translateY(-2px)' }
            }}
          >
            <CardActionArea onClick={() => onNavigate('business_objects')} sx={{ p: 3, height: '100%' }}>
              <Stack spacing={2}>
                <Box display="flex" justifyContent="space-between" alignItems="flex-start">
                  <Box sx={{ p: 1.5, bgcolor: '#071526', borderRadius: 1.5, border: '1px solid #1E293B' }}>
                    <BusinessObjectIcon sx={{ color: '#34D399', fontSize: 28 }} />
                  </Box>
                  <Chip label="48 Models" size="small" sx={{ bgcolor: '#071526', color: '#34D399', fontSize: 11, fontWeight: 700 }} />
                </Box>

                <Box>
                  <Typography variant="h6" sx={{ fontWeight: 700, color: '#F8FAFC' }}>
                    Business Objects
                  </Typography>
                  <Typography variant="body2" sx={{ color: '#94A3B8', mt: 0.5, minHeight: 40 }}>
                    Curated domain contracts, multi-backend driving tables, and federated query generation.
                  </Typography>
                </Box>

                <Divider sx={{ borderColor: '#1E293B' }} />

                <Stack direction="row" justifyContent="space-between" alignItems="center">
                  <Typography variant="caption" sx={{ color: '#64748B' }}>Multi-Backend Bound</Typography>
                  <ArrowForwardIcon sx={{ color: '#34D399', fontSize: 16 }} />
                </Stack>
              </Stack>
            </CardActionArea>
          </Card>
        </Grid>
      </Grid>

      <Paper sx={{ p: 2.5, bgcolor: '#0B1E36', border: '1px solid #1E293B', borderRadius: 2 }}>
        <Grid container spacing={3} alignItems="center">
          <Grid   size={{ xs: 12, md: 4 }}>
            <Stack direction="row" spacing={1.5} alignItems="center">
              <GovernanceIcon sx={{ color: '#00D4FF', fontSize: 22 }} />
              <Box>
                <Typography variant="subtitle2" sx={{ fontWeight: 700, color: '#F8FAFC', fontSize: 13 }}>
                  Catalog Health & Vector Resolution
                </Typography>
                <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                  99.82% of active Business Object fields bound cleanly
                </Typography>
              </Box>
            </Stack>
          </Grid>

          <Grid   size={{ xs: 12, md: 5 }}>
            <Stack spacing={0.5}>
              <Box display="flex" justifyContent="space-between">
                <Typography variant="caption" sx={{ color: '#94A3B8' }}>Overall Contract Coverage</Typography>
                <Typography variant="caption" sx={{ color: '#34D399', fontWeight: 700 }}>99.8%</Typography>
              </Box>
              <LinearProgress
                variant="determinate"
                value={99.8}
                sx={{ height: 6, borderRadius: 1, bgcolor: '#071526', '& .MuiLinearProgress-bar': { bgcolor: '#10B981' } }}
              />
            </Stack>
          </Grid>

          <Grid   sx={{ textAlign: { xs: 'left', md: 'right' } }} size={{ xs: 12, md: 3 }}>
            <Button
              variant="outlined"
              size="small"
              startIcon={<LineageIcon />}
              onClick={() => onNavigate('lineage_simulator')}
              sx={{ borderColor: '#334155', color: '#38BDF8', textTransform: 'none', fontSize: 12, '&:hover': { borderColor: '#38BDF8' } }}
            >
              Open Impact Simulator
            </Button>
          </Grid>
        </Grid>
      </Paper>
    </Box>
  );
};

export default ActiveSemanticCatalogHub;
