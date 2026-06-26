import { useState } from 'react';
import {
  Box,
  TextField,
  Typography,
  Stack,
  Card,
  CardContent,
  Chip,
  IconButton,
  Collapse,
  Paper,
  InputAdornment,
  Divider,
} from '@mui/material';
import {
  Search as SearchIcon,
  ExpandMore as ExpandMoreIcon,
  Info as InfoIcon,
} from '@mui/icons-material';

interface SemanticTerm {
  id: string;
  name: string;
  dataType: 'STRING' | 'NUMBER' | 'BOOLEAN' | 'DATE';
  businessDefinition: string;
  sampleValues?: string[];
  governanceStatus: 'APPROVED' | 'DRAFT' | 'DEPRECATED';
  category: string;
}

interface SemanticCatalogProps {
  businessObject?: string;
  terms?: SemanticTerm[];
  onTermDrag?: (term: SemanticTerm) => void;
}

/**
 * SemanticCatalog Component (Material-UI)
 * Left-panel for browsing semantic terms
 */
export const SemanticCatalog = ({
  businessObject = 'calendar',
  terms = [],
  onTermDrag,
}: SemanticCatalogProps) => {
  const [searchQuery, setSearchQuery] = useState('');
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(
    new Set(['IDENTIFICATION', 'CLASSIFICATION'])
  );
  const [hoveredTerm, setHoveredTerm] = useState<string | null>(null);

  // Mock data if none provided
  const mockTerms: SemanticTerm[] = [
    {
      id: 'term-1',
      name: 'CalendarDate',
      dataType: 'DATE',
      businessDefinition: 'The specific date being evaluated for business day status',
      sampleValues: ['2026-12-25', '2026-01-01'],
      governanceStatus: 'APPROVED',
      category: 'IDENTIFICATION',
    },
    {
      id: 'term-2',
      name: 'IsBusinessDay',
      dataType: 'BOOLEAN',
      businessDefinition: 'Whether this date is a business day',
      sampleValues: ['true', 'false'],
      governanceStatus: 'APPROVED',
      category: 'CLASSIFICATION',
    },
  ];

  const displayTerms = terms.length > 0 ? terms : mockTerms;

  const categorizedTerms = displayTerms.reduce(
    (acc, term) => {
      if (!acc[term.category]) acc[term.category] = [];
      acc[term.category].push(term);
      return acc;
    },
    {} as Record<string, SemanticTerm[]>
  );

  const filteredTerms = Object.entries(categorizedTerms).reduce(
    (acc, [category, categoryTerms]) => {
      const filtered = categoryTerms.filter(
        (term) =>
          term.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
          term.businessDefinition.toLowerCase().includes(searchQuery.toLowerCase())
      );
      if (filtered.length > 0) acc[category] = filtered;
      return acc;
    },
    {} as Record<string, SemanticTerm[]>
  );

  const toggleCategory = (category: string) => {
    setExpandedCategories((prev) => {
      const next = new Set(prev);
      next.has(category) ? next.delete(category) : next.add(category);
      return next;
    });
  };

  const getDataTypeIcon = (dataType: string) => {
    const icons: Record<string, string> = {
      STRING: 'Σ',
      NUMBER: '#',
      BOOLEAN: '⊙',
      DATE: '📅',
    };
    return icons[dataType] || '?';
  };

  const getStatusColor = (status: string) => {
    const colors: Record<string, 'success' | 'warning' | 'error'> = {
      APPROVED: 'success',
      DRAFT: 'warning',
      DEPRECATED: 'error',
    };
    return colors[status] || 'default';
  };

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Header */}
      <Paper sx={{ p: 2, borderRadius: 0 }} elevation={0}>
        <Typography variant="subtitle2" fontWeight="600" sx={{ mb: 2 }}>
          Semantic Terms
        </Typography>
        <TextField
          fullWidth
          size="small"
          placeholder="Search terms..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon fontSize="small" />
              </InputAdornment>
            ),
          }}
        />
        <Typography variant="caption" color="textSecondary" sx={{ mt: 1, display: 'block' }}>
          Drag terms → into the rule builder
        </Typography>
      </Paper>

      <Divider />

      {/* Terms */}
      <Box sx={{ flex: 1, overflowY: 'auto' }}>
        {Object.entries(filteredTerms).length > 0 ? (
          Object.entries(filteredTerms).map(([category, categoryTerms]) => (
            <Box key={category}>
              <Box
                onClick={() => toggleCategory(category)}
                sx={{
                  px: 2,
                  py: 1.5,
                  backgroundColor: 'action.hover',
                  cursor: 'pointer',
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  '&:hover': { backgroundColor: 'action.selected' },
                }}
              >
                <Typography variant="caption" fontWeight="700" textTransform="uppercase">
                  {category}
                </Typography>
                <ExpandMoreIcon
                  sx={{
                    fontSize: '1.2rem',
                    transform: expandedCategories.has(category) ? 'rotate(180deg)' : 'rotate(0deg)',
                    transition: 'transform 200ms',
                  }}
                />
              </Box>

              <Collapse in={expandedCategories.has(category)}>
                <Stack spacing={1} sx={{ p: 1 }}>
                  {categoryTerms.map((term) => (
                    <Card
                      key={term.id}
                      draggable
                      onDragStart={() => onTermDrag?.(term)}
                      onMouseEnter={() => setHoveredTerm(term.id)}
                      onMouseLeave={() => setHoveredTerm(null)}
                      sx={{
                        cursor: 'grab',
                        transition: 'all 200ms',
                        '&:hover': {
                          borderColor: 'primary.light',
                          backgroundColor: 'primary.lighter',
                        },
                      }}
                    >
                      <CardContent sx={{ py: 1.5, '&:last-child': { pb: 1.5 } }}>
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                          <Typography variant="subtitle2" fontWeight="600">
                            {term.name}
                          </Typography>
                          <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'center' }}>
                            <Typography variant="h6" sx={{ color: 'action.disabled' }}>
                              {getDataTypeIcon(term.dataType)}
                            </Typography>
                            <Chip
                              label={term.governanceStatus}
                              size="small"
                              color={getStatusColor(term.governanceStatus)}
                              variant="outlined"
                            />
                          </Box>
                        </Box>

                        <Typography variant="caption" color="textSecondary" sx={{ display: 'block', mb: 1 }}>
                          {term.businessDefinition}
                        </Typography>

                        {term.sampleValues && term.sampleValues.length > 0 && (
                          <Stack direction="row" spacing={0.5} sx={{ flexWrap: 'wrap' }}>
                            {term.sampleValues.slice(0, 2).map((value, idx) => (
                              <Chip
                                key={idx}
                                label={value}
                                size="small"
                                variant="outlined"
                                sx={{ font: 'monospace' }}
                              />
                            ))}
                            {term.sampleValues.length > 2 && (
                              <Typography variant="caption" color="textSecondary">
                                +{term.sampleValues.length - 2} more
                              </Typography>
                            )}
                          </Stack>
                        )}

                        {hoveredTerm === term.id && (
                          <Paper
                            sx={{
                              mt: 1.5,
                              p: 1,
                              backgroundColor: 'info.lighter',
                              border: '1px solid',
                              borderColor: 'info.light',
                            }}
                          >
                            <Typography variant="caption" color="info.dark" sx={{ display: 'flex', alignItems: 'center' }}>
                              <InfoIcon sx={{ fontSize: '1rem', mr: 0.5 }} />
                              Drag to add to priority rule
                            </Typography>
                          </Paper>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </Stack>
              </Collapse>
            </Box>
          ))
        ) : (
          <Box sx={{ p: 3, textAlign: 'center' }}>
            <Typography variant="body2" color="textSecondary">
              {searchQuery
                ? 'No semantic terms match your search'
                : 'No semantic terms available for this business object'}
            </Typography>
          </Box>
        )}
      </Box>

      {/* Footer Tip */}
      <Paper sx={{ p: 1.5, borderRadius: 0, backgroundColor: 'background.default' }} elevation={0}>
        <Typography variant="caption" color="textSecondary">
          💡 Semantic terms are business-friendly names for data fields
        </Typography>
      </Paper>
    </Box>
  );
};

export default SemanticCatalog;
