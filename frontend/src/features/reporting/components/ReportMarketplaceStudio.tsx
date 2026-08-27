import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  Chip,
  Stack,
  TextField,
  InputAdornment,
  Rating,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Divider,
} from '@mui/material';
import StorefrontIcon from '@mui/icons-material/Storefront';
import AutoAwesomeIcon from '@mui/icons-material/AutoAwesome';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import VerifiedIcon from '@mui/icons-material/Verified';
import SearchIcon from '@mui/icons-material/Search';
import DownloadIcon from '@mui/icons-material/Download';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';

export interface MarketplaceItem {
  packageId: string;
  packageCode: string;
  displayName: string;
  publisherName: string;
  category: 'REGULATORY' | 'CLIENT_REPORTING' | 'RISK' | 'PERFORMANCE';
  description: string;
  rating: number;
  installCount: number;
  isVerified: boolean;
  requiredTerms: string[];
}

export const ReportMarketplaceStudio: React.FC<{ tenantId?: string }> = ({ tenantId = 'default' }) => {
  const [activeTab, setActiveTab] = useState<'MARKETPLACE' | 'AI_COPILOT'>('MARKETPLACE');
  const [searchQuery, setSearchQuery] = useState('');
  const [aiPrompt, setAiPrompt] = useState('');
  const [isGenerating, setIsGenerating] = useState(false);
  const [cloneModalOpen, setCloneModalOpen] = useState(false);
  const [selectedTemplateForClone, setSelectedTemplateForClone] = useState<MarketplaceItem | null>(null);

  const mockPackages: MarketplaceItem[] = [
    {
      packageId: 'pkg_1',
      packageCode: 'GIPS_PERF_DECK_V2',
      displayName: 'GIPS Composite Performance Presentation',
      publisherName: 'Uisce Regulatory Core',
      category: 'PERFORMANCE',
      description: 'Institutional composite presentation compliant with GIPS 2026 standards, asset-weighted returns, and dispersion metrics.',
      rating: 4.9,
      installCount: 1420,
      isVerified: true,
      requiredTerms: ['portfolio_nav', 'gross_return', 'net_fund_yield', 'composite_dispersion'],
    },
    {
      packageId: 'pkg_2',
      packageCode: 'UCITS_KIID_FACTSHEET',
      displayName: 'UCITS KIID & Synthetic Risk Indicator (SRI)',
      publisherName: 'European Fund Operations Hub',
      category: 'REGULATORY',
      description: 'Strict 2-page European retail investor fact sheet with SRI risk-reward gauge, benchmark drawdown ribbon, and fee disclosures.',
      rating: 4.8,
      installCount: 890,
      isVerified: true,
      requiredTerms: ['portfolio_nav', 'sri_risk_score', 'benchmark_drawdown', 'ter_expense_ratio'],
    },
    {
      packageId: 'pkg_3',
      packageCode: 'BRINSON_ALPHA_TEARSHEET',
      displayName: 'Executive Alpha Attribution Tear Sheet (1-Page)',
      publisherName: 'Meridian Quantitative Labs',
      category: 'CLIENT_REPORTING',
      description: 'Condensed 1-page institutional tear sheet with Brinson-Fachler sector waterfall, active share, and top 10 contributors.',
      rating: 5.0,
      installCount: 3100,
      isVerified: true,
      requiredTerms: ['brinson_alpha', 'allocation_effect', 'selection_effect', 'active_share'],
    },
  ];

  const handleOpenClone = (pkg: MarketplaceItem) => {
    setSelectedTemplateForClone(pkg);
    setCloneModalOpen(true);
  };

  return (
    <Box sx={{ width: '100%', minHeight: '100vh', bgcolor: '#050D1A', color: '#fff', p: 3, fontFamily: 'sans-serif' }}>
      
      {/* Header Bar */}
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pb: 2, borderBottom: '1px solid #1E293B', mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <StorefrontIcon sx={{ color: '#00D4FF', fontSize: 26 }} />
          <Box>
            <Typography variant="h6" fontWeight="700">
              Report Template Marketplace & AI Studio
            </Typography>
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Install verified institutional templates, clone Gold Copy baselines, or generate reports via AI.
            </Typography>
          </Box>
        </Box>

        <Stack direction="row" spacing={1.5}>
          <Button
            variant={activeTab === 'MARKETPLACE' ? 'contained' : 'outlined'}
            startIcon={<StorefrontIcon />}
            onClick={() => setActiveTab('MARKETPLACE')}
            sx={{ bgcolor: activeTab === 'MARKETPLACE' ? '#00D4FF' : 'transparent', color: activeTab === 'MARKETPLACE' ? '#050D1A' : '#00D4FF', fontWeight: 700, fontSize: '11px', textTransform: 'none' }}
          >
            Template Marketplace
          </Button>
          <Button
            variant={activeTab === 'AI_COPILOT' ? 'contained' : 'outlined'}
            startIcon={<AutoAwesomeIcon />}
            onClick={() => setActiveTab('AI_COPILOT')}
            sx={{ bgcolor: activeTab === 'AI_COPILOT' ? '#A855F7' : 'transparent', color: activeTab === 'AI_COPILOT' ? '#fff' : '#A855F7', borderColor: '#A855F7', fontWeight: 700, fontSize: '11px', textTransform: 'none' }}
          >
            AI Report Copilot
          </Button>
        </Stack>
      </Box>

      {/* Tab 1: Marketplace View */}
      {activeTab === 'MARKETPLACE' && (
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
            <TextField
              size="small"
              placeholder="Search verified templates, regulatory packs, attribution decks..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              sx={{ width: 440, bgcolor: '#071526', input: { color: '#fff', fontSize: '12px' } }}
              InputProps={{
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon sx={{ color: '#64748B', fontSize: 18 }} />
                  </InputAdornment>
                ),
              }}
            />
            <Typography variant="caption" sx={{ color: '#64748B' }}>
              Showing {mockPackages.length} Verified Institutional Packages
            </Typography>
          </Box>

          <Box sx={{ display: 'grid', gridTemplateColumns: { xs: '1fr', md: 'repeat(3, 1fr)' }, gap: 3 }}>
            {mockPackages.map((pkg) => (
              <Paper
                key={pkg.packageId}
                sx={{
                  p: 2.5,
                  bgcolor: '#071526',
                  border: '1px solid #1E293B',
                  borderRadius: 2,
                  display: 'flex',
                  flexDirection: 'column',
                  justifyContent: 'space-between',
                  '&:hover': { borderColor: 'rgba(0, 212, 255, 0.4)', bgcolor: '#0B1E36' },
                  transition: 'all 0.2s ease',
                }}
              >
                <div>
                  <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 1 }}>
                    <Chip
                      size="small"
                      label={pkg.category}
                      sx={{ bgcolor: 'rgba(0, 212, 255, 0.1)', color: '#00D4FF', fontWeight: 700, fontSize: '9px', height: 18 }}
                    />
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <Rating value={pkg.rating} precision={0.1} size="small" readOnly sx={{ fontSize: '13px' }} />
                      <Typography variant="caption" sx={{ color: '#94A3B8', fontSize: '10px' }}>({pkg.installCount})</Typography>
                    </Box>
                  </Box>

                  <Typography variant="subtitle2" fontWeight="700" sx={{ color: '#E2E8F0', mb: 0.5 }}>
                    {pkg.displayName}
                  </Typography>

                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5, mb: 1.5 }}>
                    <VerifiedIcon sx={{ color: '#10B981', fontSize: 14 }} />
                    <Typography variant="caption" sx={{ color: '#64748B', fontSize: '11px' }}>
                      {pkg.publisherName}
                    </Typography>
                  </Box>

                  <Typography variant="body2" sx={{ color: '#94A3B8', fontSize: '11px', mb: 2, lineClamp: 2 }}>
                    {pkg.description}
                  </Typography>

                  <Box sx={{ mb: 2 }}>
                    <Typography variant="caption" sx={{ color: '#64748B', fontSize: '10px', display: 'block', mb: 0.5 }}>
                      Required Semantic Terms:
                    </Typography>
                    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                      {pkg.requiredTerms.map((t) => (
                        <span key={t} style={{ padding: '2px 6px', borderRadius: 4, background: '#050D1A', color: '#94A3B8', fontFamily: 'monospace', fontSize: 9, border: '1px solid #1E293B' }}>
                          {t}
                        </span>
                      ))}
                    </Box>
                  </Box>
                </div>

                <Stack direction="row" spacing={1.5} sx={{ pt: 2, borderTop: '1px solid #1E293B' }}>
                  <Button
                    size="small"
                    variant="outlined"
                    startIcon={<ContentCopyIcon />}
                    onClick={() => handleOpenClone(pkg)}
                    sx={{ flex: 1, color: '#00D4FF', borderColor: 'rgba(0, 212, 255, 0.3)', textTransform: 'none', fontSize: '11px', fontWeight: 600 }}
                  >
                    Clone to My Workspace
                  </Button>
                </Stack>
              </Paper>
            ))}
          </Box>
        </Box>
      )}

      {/* Tab 2: AI Copilot Prompt Studio */}
      {activeTab === 'AI_COPILOT' && (
        <Paper sx={{ p: 4, bgcolor: '#071526', border: '1px solid #1E293B', borderRadius: 2, maxWidth: 800, mx: 'auto' }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 2 }}>
            <AutoAwesomeIcon sx={{ color: '#A855F7', fontSize: 24 }} />
            <Typography variant="subtitle1" fontWeight="700">
              AI Natural Language Report Synthesizer
            </Typography>
          </Box>
          <Typography variant="caption" sx={{ color: '#94A3B8', display: 'block', mb: 3 }}>
            Describe your reporting requirements in plain English. The AI will inspect your active Business Objects, resolve required semantic terms, and synthesize a millimeter-exact report layout.
          </Typography>

          <TextField
            fullWidth
            multiline
            rows={4}
            placeholder="e.g. Build a 1-page Fact Sheet for our Tech Alpha portfolio. Include KPI cards for Ending NAV and Yield, a Brinson attribution waterfall vs S&P 500, and a summary of Top 5 Holdings."
            value={aiPrompt}
            onChange={(e) => setAiPrompt(e.target.value)}
            sx={{ bgcolor: '#050D1A', mb: 3, input: { color: '#fff', fontSize: '12px' } }}
          />

          <Button
            variant="contained"
            startIcon={<AutoAwesomeIcon />}
            disabled={!aiPrompt.trim() || isGenerating}
            onClick={() => {
              setIsGenerating(true);
              setTimeout(() => setIsGenerating(false), 1200);
            }}
            sx={{ bgcolor: '#A855F7', color: '#fff', fontWeight: 700, fontSize: '12px', textTransform: 'none', px: 3, '&:hover': { bgcolor: '#9333EA' } }}
          >
            {isGenerating ? 'Synthesizing Semantic Report...' : 'Generate Report AST'}
          </Button>
        </Paper>
      )}

      {/* Clone Confirmation Modal */}
      <Dialog
        open={cloneModalOpen}
        onClose={() => setCloneModalOpen(false)}
        PaperProps={{ sx: { bgcolor: '#071526', color: '#fff', border: '1px solid #1E293B', minWidth: 440 } }}
      >
        <DialogTitle sx={{ fontWeight: 700, fontSize: '14px' }}>
          Clone Core Template to Custom Workspace
        </DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ color: '#94A3B8', fontSize: '12px', mb: 2 }}>
            You are creating an editable tenant copy of <strong style={{ color: '#fff' }}>{selectedTemplateForClone?.displayName}</strong>. Upstream Gold Copy enhancements can be automatically merged via the 3-Way Rebase Engine.
          </Typography>
          <TextField
            fullWidth
            size="small"
            label="Custom Report Name"
            defaultValue={selectedTemplateForClone ? `Custom - ${selectedTemplateForClone.displayName}` : ''}
            sx={{ bgcolor: '#050D1A', input: { color: '#fff', fontSize: '12px' } }}
          />
        </DialogContent>
        <DialogActions sx={{ p: 2, borderTop: '1px solid #1E293B' }}>
          <Button onClick={() => setCloneModalOpen(false)} sx={{ color: '#94A3B8', textTransform: 'none', fontSize: '11px' }}>
            Cancel
          </Button>
          <Button
            variant="contained"
            onClick={() => setCloneModalOpen(false)}
            sx={{ bgcolor: '#00D4FF', color: '#050D1A', fontWeight: 700, fontSize: '11px', textTransform: 'none' }}
          >
            Confirm & Open in Studio
          </Button>
        </DialogActions>
      </Dialog>

    </Box>
  );
};

export default ReportMarketplaceStudio;
