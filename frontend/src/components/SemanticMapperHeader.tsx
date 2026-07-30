import { Box, Typography, Button, Chip, Card, CardContent, Switch, FormControlLabel } from '@mui/material';
import { Refresh, Storage, Filter } from '@mui/icons-material';

interface SemanticMapperHeaderProps {
  loading: boolean;
  loadMappings: () => void;
  filteredMappingsCount: number;
  selectedMappingsCount: number;
  highConfidenceCount: number;
  averageScore: number;
  hasScopeFilter: boolean;
  compactRows: boolean;
  setCompactRows: (v: boolean) => void;
}

export function SemanticMapperHeader({
  loading,
  loadMappings,
  filteredMappingsCount,
  selectedMappingsCount,
  highConfidenceCount,
  averageScore,
  hasScopeFilter,
  compactRows,
  setCompactRows,
}: SemanticMapperHeaderProps) {
  return (
    <Card sx={{ mb: 3, borderRadius: 2 }} elevation={2}>
      <CardContent>
        <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', mb: 2 }}>
          <Box>
            <Typography variant="h4" component="h1" sx={{ display: 'flex', alignItems: 'center', gap: 1.5, fontWeight: 700 }}>
              <Storage sx={{ width: 32, height: 32, color: '#1976d2' }} />
              Semantic Term Mapper
            </Typography>
            <Typography variant="body2" color="text.secondary" sx={{ mt: 1 }}>
              Map database columns to semantic terms with AI-powered confidence scoring
            </Typography>
          </Box>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <FormControlLabel control={<Switch checked={compactRows} onChange={(e) => setCompactRows(e.target.checked)} size="small" />} label="Compact rows" />
            <Button onClick={loadMappings} disabled={loading} variant="contained" startIcon={<Refresh sx={{ width: 16, height: 16, animation: loading ? 'spin 1s linear infinite' : 'none', '@keyframes spin': { '0%': { transform: 'rotate(0deg)' }, '100%': { transform: 'rotate(360deg)' } } }} />} aria-label="Refresh mappings">
              Refresh
            </Button>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', gap: 2, mt: 2, flexWrap: 'wrap' }}>
          <Chip label={`Total: ${filteredMappingsCount} Mappings`} color="primary" size="small" sx={{ fontWeight: 600 }} />
          <Chip label={`Selected: ${selectedMappingsCount}`} color={selectedMappingsCount > 0 ? 'success' : 'default'} size="small" sx={{ fontWeight: 600 }} />
          <Chip label={`High Confidence: ${highConfidenceCount}`} color="info" size="small" sx={{ fontWeight: 600 }} icon={<span className="emoji-small">🎯</span>} />
          <Chip label={`Avg Score: ${averageScore.toFixed(0)}%`} color="secondary" size="small" sx={{ fontWeight: 600 }} />
          {hasScopeFilter && (
            <Chip label="Filtered" color="warning" size="small" icon={<Filter sx={{ width: 14, height: 14 }} />} sx={{ fontWeight: 600 }} />
          )}
        </Box>
      </CardContent>
    </Card>
  );
}