import React, { useState, useEffect } from 'react';
import {
  Box,
  TextField,
  Select,
  MenuItem,
  List,
  ListItem,
  ListItemIcon,
  ListItemText,
  Chip,
  IconButton,
  Typography,
  CircularProgress,
  InputAdornment,
  Paper,
} from '@mui/material';
import {
  Search as SearchIcon,
  DataObject as TermIcon,
  AccountTree as RelatedBOIcon,
  TableChart as TableIcon,
  Functions as CalcIcon,
  OpenInNew as OpenInNewIcon,
} from '@mui/icons-material';
import { useNavigate } from 'react-router-dom';

// ============================================================================
// Types
// ============================================================================

type SearchType = 'all' | 'term' | 'related_bo' | 'table' | 'calc';

interface SearchResult {
  type: string;
  id: string;
  name: string;
  match_type: string;
  matched_term?: string;
  matched_table?: string;
  matched_calculation?: string;
  relationship?: string;
  term_count?: number;
  score: number;
}

interface SearchResponse {
  results: SearchResult[];
  total: number;
}

interface GlobalBOSearchProps {
  onResultClick?: (boId: string) => void;
}

// ============================================================================
// Main Component
// ============================================================================

export const GlobalBOSearch: React.FC<GlobalBOSearchProps> = ({ onResultClick }) => {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [searchType, setSearchType] = useState<SearchType>('all');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    if (query.length >= 2) {
      const debounce = setTimeout(() => {
        handleSearch();
      }, 300);
      return () => clearTimeout(debounce);
    } else {
      setResults([]);
      setTotal(0);
    }
  }, [query, searchType]);

  const handleSearch = async () => {
    setLoading(true);
    try {
      const response = await fetch(
        `/api/bo/search?q=${encodeURIComponent(query)}&type=${searchType}&limit=50`
      );
      const data: SearchResponse = await response.json();
      setResults(data.results);
      setTotal(data.total);
    } catch (err) {
      console.error('Search failed:', err);
      setResults([]);
      setTotal(0);
    } finally {
      setLoading(false);
    }
  };

  const handleResultClick = (result: SearchResult) => {
    if (onResultClick) {
      onResultClick(result.id);
    } else {
      navigate(`/bo/${result.id}`);
    }
  };

  const getMatchTypeIcon = (matchType: string) => {
    switch (matchType) {
      case 'term':
        return <TermIcon color="primary" />;
      case 'related_bo':
        return <RelatedBOIcon color="secondary" />;
      case 'driving_table':
        return <TableIcon color="info" />;
      case 'calculation':
        return <CalcIcon color="success" />;
      default:
        return <SearchIcon />;
    }
  };

  const getMatchTypeLabel = (matchType: string) => {
    switch (matchType) {
      case 'term':
        return 'Term';
      case 'related_bo':
        return 'Related BO';
      case 'driving_table':
        return 'Table';
      case 'calculation':
        return 'Calculation';
      default:
        return 'Match';
    }
  };

  return (
    <Box>
      <TextField
        fullWidth
        placeholder="Search BOs by term, table, calculation..."
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon />
            </InputAdornment>
          ),
          endAdornment: (
            <InputAdornment position="end">
              <Select
                value={searchType}
                onChange={(e) => setSearchType(e.target.value as SearchType)}
                variant="standard"
                sx={{ minWidth: 120 }}
              >
                <MenuItem value="all">All</MenuItem>
                <MenuItem value="term">Term</MenuItem>
                <MenuItem value="related_bo">Related BO</MenuItem>
                <MenuItem value="table">Table</MenuItem>
                <MenuItem value="calc">Calculation</MenuItem>
              </Select>
            </InputAdornment>
          ),
        }}
        sx={{
          '& .MuiOutlinedInput-root': {
            borderRadius: 2,
          },
        }}
      />

      {query.length >= 2 && (
        <Paper elevation={3} sx={{ mt: 1, maxHeight: 400, overflow: 'auto' }}>
          {loading ? (
            <Box sx={{ display: 'flex', justifyContent: 'center', p: 3 }}>
              <CircularProgress />
            </Box>
          ) : results.length > 0 ? (
            <>
              <Typography variant="caption" sx={{ p: 2, display: 'block', color: 'text.secondary' }}>
                {total} result{total !== 1 ? 's' : ''} found
              </Typography>
              <List>
                {results.map((result) => (
                  <ListItem
                    key={result.id}
                    button
                    onClick={() => handleResultClick(result)}
                    sx={{
                      '&:hover': {
                        bgcolor: 'action.hover',
                      },
                    }}
                  >
                    <ListItemIcon>{getMatchTypeIcon(result.match_type)}</ListItemIcon>
                    <ListItemText
                      primary={
                        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                          <Typography variant="body1" fontWeight="medium">
                            {result.name}
                          </Typography>
                          <Chip
                            label={getMatchTypeLabel(result.match_type)}
                            size="small"
                            variant="outlined"
                          />
                          <Chip
                            label={`${Math.round(result.score * 100)}%`}
                            size="small"
                            color="primary"
                          />
                        </Box>
                      }
                      secondary={
                        <>
                          {result.matched_term && `Term: ${result.matched_term}`}
                          {result.matched_table && `Table: ${result.matched_table}`}
                          {result.matched_calculation && `Calculation: ${result.matched_calculation}`}
                          {result.relationship && `Relationship: ${result.relationship}`}
                          {result.term_count && ` • ${result.term_count} terms`}
                        </>
                      }
                    />
                    <IconButton size="small">
                      <OpenInNewIcon fontSize="small" />
                    </IconButton>
                  </ListItem>
                ))}
              </List>
            </>
          ) : (
            <Typography variant="body2" color="text.secondary" sx={{ p: 3, textAlign: 'center' }}>
              No results found for "{query}"
            </Typography>
          )}
        </Paper>
      )}
    </Box>
  );
};

export default GlobalBOSearch;
