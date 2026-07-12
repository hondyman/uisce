import React, { useState } from 'react';
import { 
  Box, TextField, List, ListItem, ListItemText, 
  Paper, InputAdornment, Chip, Stack, CircularProgress, Alert 
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import PageviewIcon from '@mui/icons-material/Pageview';
import HubIcon from '@mui/icons-material/Hub';
import MenuOpenIcon from '@mui/icons-material/MenuOpen';

interface SearchMatch {
  source_type: string;
  entity_key: string;
  display_name: string;
  context_blob: string;
}

export const SemanticDiscoveryOmnibox: React.FC<{ tenantId: string }> = ({ tenantId }) => {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchMatch[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const triggerSearch = async (val: string) => {
    setQuery(val);
    if (val.length < 3) {
      setResults([]);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const response = await fetch('/api/v1/discovery/search', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-Tenant-ID': tenantId
        },
        body: JSON.stringify({ query: val })
      });
      if (!response.ok) throw new Error("Context lookups rejected by authorization filters.");
      const data = await response.json();
      setResults(data.matches || []);
    } catch (err: any) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const renderSourceIcon = (type: string) => {
    switch (type) {
      case 'PAGE': return <PageviewIcon sx={{ color: '#38bdf8' }} />;
      case 'MENU_ITEM': return <MenuOpenIcon sx={{ color: '#fbbf24' }} />;
      default: return <HubIcon sx={{ color: '#34d399' }} />;
    }
  };

  return (
    <Box sx={{ width: '100%', maxWidth: 700, mx: 'auto', p: 2 }}>
      <TextField
        fullWidth
        placeholder="Discover pages, fields, or semantic structures... (e.g. net revenue)"
        value={query}
        onChange={(e) => triggerSearch(e.target.value)}
        size="small"
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              {loading ? <CircularProgress size={20} /> : <SearchIcon />}
            </InputAdornment>
          ),
          sx: { bgcolor: '#1e293b', color: '#f8fafc', borderRadius: 2, '& fieldset': { border: 'none' } }
        }}
      />

      {error && <Alert severity="error" sx={{ mt: 2, borderRadius: 2 }}>{error}</Alert>}

      {results.length > 0 && (
        <Paper sx={{ mt: 1, bgcolor: '#0f172a', border: '1px solid #334155', borderRadius: 2, overflow: 'hidden' }} elevation={4}>
          <List disablePadding>
            {results.map((match, idx) => (
              <ListItem 
                key={idx} 
                sx={{ borderBottom: idx !== results.length - 1 ? '1px solid #334155' : 'none', py: 1.5, px: 3, '&:hover': { bgcolor: 'rgba(255,255,255,0.02)' } }}
              >
                <Stack direction="row" spacing={2} alignItems="center" width="100%">
                  {renderSourceIcon(match.source_type)}
                  <ListItemText 
                    primary={match.display_name} 
                    secondary={match.entity_key}
                    primaryTypographyProps={{ color: '#f1f5f9', fontWeight: 500, fontSize: '14px' }}
                    secondaryTypographyProps={{ color: '#64748b', fontSize: '11px', fontFamily: 'monospace' }}
                  />
                  <Box sx={{ flexGrow: 1 }} />
                  <Chip label={match.source_type} size="small" sx={{ bgcolor: '#334155', color: '#94a3b8', fontSize: '10px', fontWeight: 600 }} />
                </Stack>
              </ListItem>
            ))}
          </List>
        </Paper>
      )}
    </Box>
  );
};
