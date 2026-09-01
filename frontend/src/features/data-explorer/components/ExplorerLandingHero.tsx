import React, { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  TextField,
  Button,
  Chip,
  Card,
  CardContent,
  CardActionArea,
  Avatar,
  Stack,
} from '@mui/material';
import { useTheme } from '@mui/material/styles';
import {
  Sparkles,
  Search,
  Database,
  TrendingUp,
  BarChart3,
  Layers,
  ArrowRight,
  Clock,
} from 'lucide-react';
import type { BusinessObjectSummary, SavedExplorerQuery } from '../types/dataExplorerTypes';

interface ExplorerLandingHeroProps {
  businessObjects: BusinessObjectSummary[];
  savedQueries: SavedExplorerQuery[];
  onSelectBo: (bo: BusinessObjectSummary) => void;
  onOpenSavedQuery: (query: SavedExplorerQuery) => void;
  onRunPrompt: (prompt: string, targetBoId?: string) => void;
}

const POPULAR_PROMPTS = [
  {
    title: 'Client Valuation & Holdings',
    prompt: 'Show total valuation by client name for active institutional accounts in 2026',
    boHint: 'Account',
    icon: <TrendingUp size={16} color="#0D9488" />,
  },
  {
    title: 'Trading Activity & Volume',
    prompt: 'Break down total trade execution volume by asset class and currency',
    boHint: 'TradeOrder',
    icon: <BarChart3 size={16} color="#38BDF8" />,
  },
  {
    title: 'Alternative Asset Commitments',
    prompt: 'List private equity commitments grouped by vintage year and strategy',
    boHint: 'AlternativeInvestment',
    icon: <Layers size={16} color="#FB923C" />,
  },
];

export const ExplorerLandingHero: React.FC<ExplorerLandingHeroProps> = ({
  businessObjects,
  savedQueries,
  onSelectBo,
  onOpenSavedQuery,
  onRunPrompt,
}) => {
  const theme = useTheme();
  const isDark = theme.palette.mode === 'dark';
  const [nlQuery, setNlQuery] = useState('');
  const [searchBo, setSearchBo] = useState('');

  const filteredBos = businessObjects.filter(
    (bo) =>
      bo.displayName.toLowerCase().includes(searchBo.toLowerCase()) ||
      bo.name.toLowerCase().includes(searchBo.toLowerCase())
  );

  const handleHeroSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (nlQuery.trim()) {
      onRunPrompt(nlQuery.trim());
    }
  };

  return (
    <Box
      sx={{
        flex: 1,
        overflowY: 'auto',
        p: { xs: 2, md: 4 },
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        bgcolor: isDark ? '#040814' : '#F8FAFC',
      }}
    >
      <Box sx={{ width: '100%', maxWidth: 1040, display: 'flex', flexDirection: 'column', gap: 4 }}>
        {/* Hero Banner */}
        <Box sx={{ textAlign: 'center', pt: 2, pb: 1 }}>
          <Stack direction="row" spacing={1} justifyContent="center" alignItems="center" sx={{ mb: 1.5 }}>
            <Chip
              icon={<Sparkles size={14} color="#0D9488" />}
              label="Uuisce Semantic Intelligence"
              size="small"
              sx={{
                bgcolor: isDark ? 'rgba(13, 148, 136, 0.15)' : '#CCFBF1',
                color: isDark ? '#2DD4BF' : '#0F766E',
                fontWeight: 700,
                fontSize: '0.72rem',
                border: '1px solid rgba(45, 212, 191, 0.3)',
              }}
            />
          </Stack>

          <Typography
            variant="h4"
            sx={{
              fontWeight: 900,
              letterSpacing: -0.5,
              color: isDark ? '#F8FAFC' : '#0F172A',
              mb: 1,
            }}
          >
            Conversational Data Explorer
          </Typography>
          <Typography
            variant="body1"
            sx={{
              color: isDark ? '#94A3B8' : '#64748B',
              maxWidth: 620,
              mx: 'auto',
              fontSize: '0.95rem',
            }}
          >
            Explore enterprise semantic models using plain English, drag-and-drop query shelves, or 1-click promotion to SSRS Report Builder datasets.
          </Typography>

          {/* Big Interactive Search Bar */}
          <Paper
            component="form"
            onSubmit={handleHeroSubmit}
            elevation={0}
            sx={{
              mt: 3,
              p: '6px 14px',
              display: 'flex',
              alignItems: 'center',
              maxWidth: 720,
              mx: 'auto',
              borderRadius: 3,
              bgcolor: isDark ? 'rgba(15, 23, 42, 0.9)' : '#FFFFFF',
              border: `1px solid ${isDark ? 'rgba(255, 255, 255, 0.12)' : 'rgba(0, 0, 0, 0.12)'}`,
              boxShadow: '0 8px 24px rgba(0, 0, 0, 0.06)',
            }}
          >
            <Sparkles size={22} color="#0D9488" style={{ marginRight: 12, flexShrink: 0 }} />
            <TextField
              fullWidth
              variant="standard"
              placeholder="Ask a business question in natural language (e.g. 'Show total revenue by region in Q3')..."
              value={nlQuery}
              onChange={(e) => setNlQuery(e.target.value)}
              InputProps={{
                disableUnderline: true,
                sx: { fontSize: '0.92rem', color: isDark ? '#F8FAFC' : '#0F172A' },
              }}
            />
            <Button
              type="submit"
              variant="contained"
              disabled={!nlQuery.trim()}
              endIcon={<ArrowRight size={15} />}
              sx={{
                bgcolor: '#0D9488',
                color: '#FFF',
                textTransform: 'none',
                fontWeight: 700,
                fontSize: '0.8rem',
                borderRadius: 2,
                px: 2.5,
                py: 0.8,
                whiteSpace: 'nowrap',
                '&:hover': { bgcolor: '#0F766E' },
              }}
            >
              Explore
            </Button>
          </Paper>

          {/* Quick Prompts */}
          <Box sx={{ mt: 2.5, display: 'flex', justifyContent: 'center', gap: 1.5, flexWrap: 'wrap' }}>
            {POPULAR_PROMPTS.map((p, i) => (
              <Chip
                key={i}
                icon={p.icon}
                label={p.title}
                onClick={() => onRunPrompt(p.prompt, p.boHint)}
                sx={{
                  cursor: 'pointer',
                  fontSize: '0.75rem',
                  fontWeight: 600,
                  bgcolor: isDark ? 'rgba(255, 255, 255, 0.04)' : '#FFFFFF',
                  border: `1px solid ${isDark ? 'rgba(255, 255, 255, 0.08)' : 'rgba(0, 0, 0, 0.08)'}`,
                  color: isDark ? '#E2E8F0' : '#334155',
                  '&:hover': {
                    bgcolor: isDark ? 'rgba(13, 148, 136, 0.15)' : '#CCFBF1',
                    borderColor: '#0D9488',
                  },
                }}
              />
            ))}
          </Box>
        </Box>

        {/* Business Objects Selection Grid */}
        <Box>
          <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Database size={18} color="#0D9488" />
              <Typography variant="subtitle1" fontWeight={800} sx={{ color: isDark ? '#F8FAFC' : '#0F172A' }}>
                Select a Semantic Business Object
              </Typography>
            </Box>
            <TextField
              size="small"
              placeholder="Filter models..."
              value={searchBo}
              onChange={(e) => setSearchBo(e.target.value)}
              sx={{
                width: 220,
                '& .MuiInputBase-input': { fontSize: '0.78rem', py: 0.6 },
              }}
              InputProps={{
                startAdornment: <Search size={14} color="#94A3B8" style={{ marginRight: 6 }} />,
              }}
            />
          </Box>

          <Box
            sx={{
              display: 'grid',
              gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr', md: '1fr 1fr 1fr' },
              gap: 2,
            }}
          >
            {filteredBos.map((bo) => (
              <Box key={bo.id}>
                <Card
                  elevation={0}
                  sx={{
                    borderRadius: 2.5,
                    border: `1px solid ${isDark ? 'rgba(255, 255, 255, 0.08)' : 'rgba(0, 0, 0, 0.08)'}`,
                    bgcolor: isDark ? 'rgba(11, 19, 43, 0.7)' : '#FFFFFF',
                    transition: 'transform 0.15s, border-color 0.15s',
                    '&:hover': {
                      transform: 'translateY(-2px)',
                      borderColor: '#0D9488',
                      boxShadow: '0 4px 16px rgba(13, 148, 136, 0.15)',
                    },
                  }}
                >
                  <CardActionArea onClick={() => onSelectBo(bo)} sx={{ p: 2 }}>
                    <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
                      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 1 }}>
                        <Avatar
                          sx={{
                            bgcolor: isDark ? 'rgba(13, 148, 136, 0.2)' : '#CCFBF1',
                            color: '#0D9488',
                            width: 36,
                            height: 36,
                            borderRadius: 2,
                          }}
                        >
                          <Database size={18} />
                        </Avatar>
                        <Box sx={{ flexGrow: 1, minWidth: 0 }}>
                          <Typography
                            variant="body2"
                            fontWeight={800}
                            noWrap
                            sx={{ color: isDark ? '#F8FAFC' : '#0F172A' }}
                          >
                            {bo.displayName}
                          </Typography>
                          <Typography variant="caption" sx={{ color: '#94A3B8' }} noWrap>
                            {bo.name}
                          </Typography>
                        </Box>
                      </Box>

                      <Typography
                        variant="body2"
                        sx={{
                          fontSize: '0.78rem',
                          color: isDark ? '#94A3B8' : '#64748B',
                          height: 36,
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          display: '-webkit-box',
                          WebkitLineClamp: 2,
                          WebkitBoxOrient: 'vertical',
                          mb: 1.5,
                        }}
                      >
                        {bo.description || 'Pre-configured semantic core entity with unified metrics, dimensions, and bitemporal isolation.'}
                      </Typography>

                      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', pt: 1, borderTop: `1px solid ${isDark ? 'rgba(255, 255, 255, 0.05)' : 'rgba(0, 0, 0, 0.05)'}` }}>
                        <Chip
                          size="small"
                          label={`${bo.fieldCount || 12} Fields`}
                          sx={{
                            fontSize: '0.68rem',
                            fontWeight: 700,
                            height: 20,
                            bgcolor: isDark ? 'rgba(255, 255, 255, 0.05)' : '#F1F5F9',
                            color: isDark ? '#94A3B8' : '#64748B',
                          }}
                        />
                        <Typography
                          variant="caption"
                          sx={{
                            fontWeight: 700,
                            color: '#0D9488',
                            display: 'flex',
                            alignItems: 'center',
                            gap: 0.5,
                            fontSize: '0.72rem',
                          }}
                        >
                          Explore <ArrowRight size={13} />
                        </Typography>
                      </Box>
                    </CardContent>
                  </CardActionArea>
                </Card>
              </Box>
            ))}
          </Box>
        </Box>

        {/* Recent / Saved Explorations Section */}
        {savedQueries.length > 0 && (
          <Box sx={{ pb: 4 }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1.5 }}>
              <Clock size={16} color="#94A3B8" />
              <Typography variant="subtitle2" fontWeight={800} sx={{ color: isDark ? '#F8FAFC' : '#0F172A' }}>
                Recent & Saved Explorations
              </Typography>
            </Box>
            <Box
              sx={{
                display: 'grid',
                gridTemplateColumns: { xs: '1fr', sm: '1fr 1fr' },
                gap: 1.5,
              }}
            >
              {savedQueries.slice(0, 4).map((sq) => (
                <Box key={sq.id}>
                  <Paper
                    elevation={0}
                    onClick={() => onOpenSavedQuery(sq)}
                    sx={{
                      p: 1.5,
                      cursor: 'pointer',
                      borderRadius: 2,
                      border: `1px solid ${isDark ? 'rgba(255, 255, 255, 0.08)' : 'rgba(0, 0, 0, 0.08)'}`,
                      bgcolor: isDark ? 'rgba(15, 23, 42, 0.6)' : '#FFFFFF',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'space-between',
                      '&:hover': {
                        borderColor: '#0D9488',
                        bgcolor: isDark ? 'rgba(13, 148, 136, 0.08)' : '#F0FDFA',
                      },
                    }}
                  >
                    <Box>
                      <Typography variant="body2" fontWeight={700} sx={{ color: isDark ? '#F8FAFC' : '#0F172A', fontSize: '0.82rem' }}>
                        {sq.name}
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#94A3B8' }}>
                        {sq.queryState.dimensions.length} dims · {sq.queryState.measures.length} measures · {sq.queryState.filters.length} filters
                      </Typography>
                    </Box>
                    <ArrowRight size={14} color="#0D9488" />
                  </Paper>
                </Box>
              ))}
            </Box>
          </Box>
        )}
      </Box>
    </Box>
  );
};

export default ExplorerLandingHero;
