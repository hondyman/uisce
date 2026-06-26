import { Box, Typography, Button, Chip, Card, CardContent, Select, MenuItem } from '@mui/material';
import { RefreshCw, Database, Filter, Link, Wand2 } from 'lucide-react';
import { ProfessionalSearchInput, SearchSuggestion } from '../common/ProfessionalSearchInput';

interface SemanticMapperHeaderProps {
  loading: boolean;
  loadMappings: () => void;
  filteredMappingsCount: number;
  selectedMappingsCount: number;
  highConfidenceCount: number;
  pendingCount: number;
  averageScore: number;
  hasScopeFilter: boolean;
  sortBy: 'confidence' | 'name' | 'none';
  setSortBy: (v: 'confidence' | 'name' | 'none') => void;
  openConfirm?: () => void;
  onOpenWizard?: () => void;
  mappedFilter: Set<string>;
  setMappedFilter: (filter: Set<any>) => void;
  mappingCounts: { all: number; mapped: number; unmapped: number; pending: number };
}

export function SemanticMapperHeader({
  loading,
  loadMappings,
  filteredMappingsCount,
  selectedMappingsCount,
  highConfidenceCount,
  pendingCount,
  averageScore,
  hasScopeFilter,
  sortBy,
  setSortBy,
  openConfirm,
  onOpenWizard,
  mappedFilter,
  setMappedFilter,
  mappingCounts,
}: SemanticMapperHeaderProps) {
  return (
    <Card 
      sx={{ 
        mb: 3, 
        borderRadius: 3,
        background: 'linear-gradient(135deg, rgba(255,255,255,0.9) 0%, rgba(248,250,252,0.8) 100%)',
        backdropFilter: 'blur(10px)',
        border: '1px solid rgba(226, 232, 240, 0.8)',
        boxShadow: '0 4px 20px rgba(0, 0, 0, 0.08), 0 1px 3px rgba(0, 0, 0, 0.1)',
      }} 
      elevation={0}
    >
      <CardContent sx={{ pb: 2.5, pt: 3 }}>
        {/* Header Row */}
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 3 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 2.5 }}>
            <Box 
              sx={{ 
                p: 1.5,
                borderRadius: 2,
                background: 'linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%)',
                boxShadow: '0 4px 12px rgba(59, 130, 246, 0.3)',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center'
              }}
            >
              <Database className="w-6 h-6" style={{ color: '#ffffff' }} />
            </Box>
            <Box>
              <Typography variant="h5" component="h1" sx={{ fontWeight: 700, mb: 0.5, color: '#0f172a' }}>
                Semantic Term Mapper
              </Typography>
              <Typography variant="body2" sx={{ color: '#64748b', fontWeight: 500 }}>
                Map database columns to semantic terms with AI-powered confidence scoring
              </Typography>
            </Box>
          </Box>
          <Box sx={{ display: 'flex', gap: 1.5 }}>
            {/* Contextual actions */}
            <Button 
              onClick={loadMappings} 
              disabled={loading} 
              variant="outlined" 
              size="small"
              startIcon={<RefreshCw className={loading ? 'animate-spin' : ''} width={16} height={16} />} 
              aria-label="Refresh mappings"
              sx={{
                borderRadius: 2,
                textTransform: 'none',
                fontWeight: 600,
                borderColor: 'rgba(59, 130, 246, 0.3)',
                color: '#3b82f6',
                '&:hover': {
                  borderColor: '#3b82f6',
                  backgroundColor: 'rgba(59, 130, 246, 0.05)'
                }
              }}
            >
              Refresh
            </Button>

            {openConfirm && selectedMappingsCount > 0 && (
              <Button 
                onClick={openConfirm} 
                variant="contained" 
                size="small"
                startIcon={<Link width={16} height={16} />}
                aria-label="Create edges for selected mappings"
                sx={{
                  borderRadius: 2,
                  textTransform: 'none',
                  fontWeight: 600,
                  background: 'linear-gradient(135deg, #10b981 0%, #059669 100%)',
                  boxShadow: '0 4px 12px rgba(16, 185, 129, 0.3)',
                  '&:hover': {
                    background: 'linear-gradient(135deg, #059669 0%, #047857 100%)',
                    boxShadow: '0 6px 16px rgba(16, 185, 129, 0.4)',
                  }
                }}
              >
                Create Edges ({selectedMappingsCount})
              </Button>
            )}
          </Box>
        </Box>

        {/* Search removed - logic moved to parent */}

        {/* Stats and Controls Row */}
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: 2 }}>
          {/* Stats Row - Horizontal Layout */}
          <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap', alignItems: 'center' }}>
            {/* Total Columns / All */}
            <Box 
              onClick={() => setMappedFilter(new Set(['all']))}
              sx={{ 
              p: 1.5, 
              borderRadius: 2, 
              border: '1px solid',
              borderColor: mappedFilter.has('all') ? 'primary.main' : 'rgba(59, 130, 246, 0.2)',
              bgcolor: mappedFilter.has('all') ? 'rgba(59, 130, 246, 0.1)' : 'rgba(59, 130, 246, 0.04)',
              display: 'flex',
              flexDirection: 'column',
              minWidth: '130px',
              transition: 'all 0.2s',
              cursor: 'pointer',
              '&:hover': { transform: 'translateY(-2px)', borderColor: 'primary.main' }
            }}>
              <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 600, mb: 0.5 }}>Total Columns</Typography>
              <Typography variant="h6" sx={{ color: '#0f172a', fontWeight: 700 }}>{filteredMappingsCount}</Typography>
            </Box>

            {/* Mapped */}
            <Box 
              onClick={() => setMappedFilter(new Set(['mapped']))}
              sx={{ 
              p: 1.5, 
              borderRadius: 2, 
              border: '1px solid',
              borderColor: mappedFilter.has('mapped') ? 'success.main' : 'rgba(16, 185, 129, 0.2)',
              bgcolor: mappedFilter.has('mapped') ? 'rgba(16, 185, 129, 0.1)' : 'rgba(16, 185, 129, 0.04)',
              display: 'flex',
              flexDirection: 'column',
              minWidth: '120px',
              transition: 'all 0.2s',
              cursor: 'pointer',
              '&:hover': { transform: 'translateY(-2px)', borderColor: 'success.main' }
            }}>
              <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 600, mb: 0.5 }}>Mapped</Typography>
              <Typography variant="h6" sx={{ color: '#059669', fontWeight: 700 }}>{mappingCounts?.mapped || 0}</Typography>
            </Box>

            {/* Unmapped */}
            <Box 
              onClick={() => setMappedFilter(new Set(['unmapped']))}
              sx={{ 
              p: 1.5, 
              borderRadius: 2, 
              border: '1px solid',
              borderColor: mappedFilter.has('unmapped') ? 'warning.main' : 'rgba(245, 158, 11, 0.2)',
              bgcolor: mappedFilter.has('unmapped') ? 'rgba(245, 158, 11, 0.1)' : 'rgba(245, 158, 11, 0.04)',
              display: 'flex',
              flexDirection: 'column',
              minWidth: '120px',
              transition: 'all 0.2s',
              cursor: 'pointer',
              '&:hover': { transform: 'translateY(-2px)', borderColor: 'warning.main' }
            }}>
              <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 600, mb: 0.5 }}>Unmapped</Typography>
              <Typography variant="h6" sx={{ color: '#d97706', fontWeight: 700 }}>{mappingCounts?.unmapped || 0}</Typography>
            </Box>

            {/* Selected */}
            <Box 
              onClick={() => setMappedFilter(new Set(['selected']))}
              sx={{ 
              p: 1.5, 
              borderRadius: 2, 
              border: '1px solid',
              borderColor: mappedFilter.has('selected') ? 'success.main' : 'rgba(16, 185, 129, 0.2)',
              bgcolor: mappedFilter.has('selected') ? 'rgba(16, 185, 129, 0.1)' : selectedMappingsCount > 0 ? 'rgba(16, 185, 129, 0.05)' : 'transparent',
              display: 'flex',
              flexDirection: 'column',
              minWidth: '130px',
              transition: 'all 0.2s',
              cursor: 'pointer',
              '&:hover': { transform: 'translateY(-2px)', borderColor: 'success.main' }
            }}>
              <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 600, mb: 0.5 }}>Selected</Typography>
              <Typography variant="h6" sx={{ color: selectedMappingsCount > 0 ? '#059669' : '#0f172a', fontWeight: 700 }}>
                {selectedMappingsCount}
              </Typography>
            </Box>

            {/* High Confidence */}
            <Box 
              onClick={() => setMappedFilter(new Set(['highConfidence']))}
              sx={{ 
              p: 1.5, 
              borderRadius: 2, 
              border: '1px solid',
              borderColor: mappedFilter.has('highConfidence') ? 'secondary.main' : 'rgba(139, 92, 246, 0.2)',
              bgcolor: mappedFilter.has('highConfidence') ? 'rgba(139, 92, 246, 0.1)' : 'rgba(139, 92, 246, 0.04)',
              display: 'flex',
              flexDirection: 'column',
              minWidth: '130px',
              transition: 'all 0.2s',
              cursor: 'pointer',
              '&:hover': { transform: 'translateY(-2px)', borderColor: 'secondary.main' }
            }}>
              <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 600, mb: 0.5 }}>High Confidence</Typography>
              <Typography variant="h6" sx={{ color: '#7c3aed', fontWeight: 700 }}>{highConfidenceCount}</Typography>
            </Box>

            {/* Pending */}
            <Box 
              onClick={() => setMappedFilter(new Set(['pending']))}
              sx={{ 
              p: 1.5, 
              borderRadius: 2, 
              border: '1px solid',
              borderColor: mappedFilter.has('pending') ? 'warning.main' : 'rgba(245, 158, 11, 0.2)',
              bgcolor: mappedFilter.has('pending') ? 'rgba(245, 158, 11, 0.1)' : 'rgba(245, 158, 11, 0.04)',
              display: 'flex',
              flexDirection: 'column',
              minWidth: '130px',
              transition: 'all 0.2s',
              cursor: 'pointer',
              '&:hover': { transform: 'translateY(-2px)', borderColor: 'warning.main' }
            }}>
              <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 600, mb: 0.5 }}>Pending</Typography>
              <Typography variant="h6" sx={{ color: '#d97706', fontWeight: 700 }}>{pendingCount}</Typography>
            </Box>

            {/* Avg Score (Not a filter logic typically, but keeping style consistent) */}
            <Box sx={{ 
              p: 1.5, 
              borderRadius: 2, 
              border: '1px solid rgba(245, 158, 11, 0.2)',
              bgcolor: 'rgba(245, 158, 11, 0.04)',
              display: 'flex',
              flexDirection: 'column',
              minWidth: '130px',
              transition: 'transform 0.2s',
              '&:hover': { transform: 'translateY(-2px)' }
            }}>
              <Typography variant="caption" sx={{ color: '#64748b', fontWeight: 600, mb: 0.5 }}>Avg. Score</Typography>
              <Typography variant="h6" sx={{ color: '#d97706', fontWeight: 700 }}>{averageScore.toFixed(0)}%</Typography>
            </Box>

            {hasScopeFilter && (
               <Box sx={{ 
                p: 1.5, 
                borderRadius: 2, 
                border: '1px solid rgba(239, 68, 68, 0.2)',
                bgcolor: 'rgba(239, 68, 68, 0.04)',
                display: 'flex',
                alignItems: 'center',
                gap: 1
              }}>
                <Filter className="w-4 h-4 text-red-500" />
                <Typography variant="subtitle2" sx={{ color: '#dc2626', fontWeight: 600 }}>Filters Active</Typography>
              </Box>
            )}
          </Box>
          
          <Box sx={{ 
            display: 'flex', 
            alignItems: 'center', 
            gap: 1.5,
            background: 'rgba(255, 255, 255, 0.7)',
            backdropFilter: 'blur(8px)',
            borderRadius: 2,
            px: 2,
            py: 1,
            border: '1px solid rgba(226, 232, 240, 0.5)'
          }}>
            <Typography variant="caption" sx={{ fontWeight: 600, color: '#475569' }}>Sort by:</Typography>
            <Select 
              value={sortBy} 
              onChange={(e) => setSortBy(e.target.value as any)} 
              size="small" 
              sx={{ 
                minWidth: 140, 
                fontSize: '0.875rem',
                '& .MuiOutlinedInput-root': {
                  borderRadius: 1.5,
                  '& fieldset': {
                    borderColor: 'rgba(226, 232, 240, 0.5)'
                  },
                  '&:hover fieldset': {
                    borderColor: '#3b82f6'
                  }
                }
              }}
            >
              <MenuItem value="confidence">🎯 Confidence</MenuItem>
              <MenuItem value="name">🔤 Column Name</MenuItem>
              <MenuItem value="none">📋 Original</MenuItem>
            </Select>
          </Box>
        </Box>
      </CardContent>
    </Card>
  );
}
