import React, { useState, useEffect, useCallback } from 'react';
import {
  Box,
  Paper,
  TextField,
  Chip,
  Button,
  CircularProgress,
  Alert,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Card,
  CardContent,
  Grid,
  Typography,
  Checkbox,
  FormGroup,
  FormControlLabel,
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import type { ValidationRule as SharedValidationRule } from './types';
import SearchIcon from '@mui/icons-material/Search';

const useStyles = makeStyles({
  root: {
    padding: '20px',
  },
  facetsPanel: {
    padding: '16px',
    marginBottom: '20px',
    backgroundColor: '#f5f5f5',
    borderRadius: '4px',
  },
  facetGroup: {
    marginBottom: '20px',
  },
  facetTitle: {
    fontWeight: 600,
    marginBottom: '8px',
    fontSize: '0.9rem',
  },
  facetOption: {
    marginBottom: '8px',
  },
  rulesContainer: {
    marginTop: '20px',
  },
  loadMoreButton: {
    marginTop: '16px',
    display: 'flex',
    justifyContent: 'center',
    width: '100%',
  },
  tableContainer: {
    marginTop: '16px',
  },
  statusChip: {
    marginRight: '8px',
  },
  searchBox: {
    marginBottom: '20px',
  },
  loaderContainer: {
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    padding: '40px',
  },
  emptyState: {
    padding: '40px',
    textAlign: 'center',
  },
});

interface FacetOption {
  value: string;
  count: number;
}

interface ApiResponse {
  rules: unknown[]; // backend may include extra fields; we'll normalize below
  total: number;
  page: number;
  limit: number;
  has_more: boolean;
  facets: {
    rule_types?: FacetOption[];
    severities?: FacetOption[];
    entities?: FacetOption[];
  };
  timestamp: string;
}

interface ApiError {
  message: string;
  code: string;
}

const ValidationRulesList: React.FC = () => {
  const classes = useStyles();
  const [rules, setRules] = useState<SharedValidationRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<ApiError | null>(null);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [totalCount, setTotalCount] = useState(0);

  // Filters
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedBPNames, setSelectedBPNames] = useState<string[]>([]);
  const [selectedRuleTypes, setSelectedRuleTypes] = useState<string[]>([]);

  // Facets
  const [bpNameFacets, setBPNameFacets] = useState<FacetOption[]>([]);
  const [ruleTypeFacets, setRuleTypeFacets] = useState<FacetOption[]>([]);

  const getTenantAndDatasource = useCallback(() => {
    const tenantId = localStorage.getItem('selected_tenant')
      ? JSON.parse(localStorage.getItem('selected_tenant') || '{}').id
      : null;
    const datasourceId = localStorage.getItem('selected_datasource')
      ? JSON.parse(localStorage.getItem('selected_datasource') || '{}').id
      : null;

    if (!tenantId || !datasourceId) {
      setError({
        message: 'Please select a tenant and datasource first',
        code: 'SCOPE_REQUIRED',
      });
      return null;
    }

    return { tenantId, datasourceId };
  }, []);

  const buildQueryParams = useCallback(
    (pageNum: number, tenantId: string, datasourceId: string): URLSearchParams => {
      const params = new URLSearchParams();
      params.append('page', pageNum.toString());
      params.append('limit', '20');
      params.append('tenant_id', tenantId);
      params.append('tenant_instance_id', datasourceId);

      if (searchQuery.trim()) {
        params.append('search', searchQuery);
      }

      if (selectedBPNames.length > 0) {
        params.append('target_entity', selectedBPNames.join(','));
      }

      if (selectedRuleTypes.length > 0) {
        params.append('rule_type', selectedRuleTypes.join(','));
      }

      return params;
    },
    [searchQuery, selectedBPNames, selectedRuleTypes]
  );

  const fetchRules = useCallback(
    async (pageNum: number, append: boolean = false) => {
      try {
        const scope = getTenantAndDatasource();
        if (!scope) return;

        setLoading(true);
        setError(null);

          const queryStr = buildQueryParams(pageNum, scope.tenantId, scope.datasourceId).toString();
          const response = await fetch(`/api/validation-rules?${queryStr}`, {
          headers: {
            'X-Tenant-ID': scope.tenantId,
            'X-Tenant-Datasource-ID': scope.datasourceId,
          },
        });

        if (!response.ok) {
          throw new Error(`API error: ${response.statusText}`);
        }

        const data: ApiResponse = await response.json();

        // Normalize backend rule shape into shared ValidationRule
        const normalized: SharedValidationRule[] = (data.rules || []).map((r: unknown) => {
          const rec = (r as Record<string, unknown>) || {}
          return {
            id: typeof rec.id === 'string' ? rec.id : String(rec.id ?? ''),
            name: (typeof rec.rule_name === 'string' ? rec.rule_name : typeof rec.name === 'string' ? rec.name : ''),
            rule_name: typeof rec.rule_name === 'string' ? rec.rule_name : typeof rec.name === 'string' ? rec.name : undefined,
            entity: typeof rec.entity === 'string' ? rec.entity : typeof rec.target_entity === 'string' ? rec.target_entity : undefined,
            target_entity: typeof rec.target_entity === 'string' ? rec.target_entity : typeof rec.entity === 'string' ? rec.entity : undefined,
            target_entities: Array.isArray(rec.target_entities) ? (rec.target_entities as string[]) : undefined,
            rule_type: typeof rec.rule_type === 'string' ? rec.rule_type : undefined,
            description: typeof rec.description === 'string' ? rec.description : undefined,
            severity: typeof rec.severity === 'string' ? rec.severity : undefined,
            is_active: typeof rec.is_active === 'boolean' ? rec.is_active : undefined,
            conditions: typeof rec.condition_json === 'string'
              ? rec.condition_json
              : typeof rec.conditions === 'string'
              ? rec.conditions
              : undefined,
            dependent_rule_ids: Array.isArray(rec.dependent_rule_ids) ? (rec.dependent_rule_ids as string[]) : [],
            created_at: typeof rec.created_at === 'string' ? rec.created_at : undefined,
            updated_at: typeof rec.updated_at === 'string' ? rec.updated_at : undefined,
          } as SharedValidationRule
        });

        // Update rules
        if (append) {
          setRules((prev) => [...prev, ...normalized]);
        } else {
          setRules(normalized);
        }

        // Update facets
        setBPNameFacets(data.facets.entities || []);
        setRuleTypeFacets(data.facets.rule_types || []);

        // Update pagination
        setPage(pageNum);
        setTotalCount(data.total);
        setHasMore(data.has_more);

        setLoading(false);
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unknown error';
        setError({
          message: `Failed to fetch rules: ${message}`,
          code: 'FETCH_ERROR',
        });
        setLoading(false);
      }
    },
    [getTenantAndDatasource, buildQueryParams]
  );

  // Initial load
  useEffect(() => {
    fetchRules(1, false);
  }, []);

  // Refetch when filters change
  const handleFilterChange = useCallback(() => {
    setPage(1);
    setRules([]);
    fetchRules(1, false);
  }, [fetchRules]);

  const handleSearchChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    setSearchQuery(event.target.value);
  };

  const handleSearchSubmit = () => {
    handleFilterChange();
  };

  const handleBPNameToggle = (bpName: string) => {
    setSelectedBPNames((prev) => {
      const newSelected = prev.includes(bpName) ? prev.filter((x) => x !== bpName) : [...prev, bpName];
      // Schedule filter change after state updates
      setTimeout(() => handleFilterChange(), 0);
      return newSelected;
    });
  };

  const handleRuleTypeToggle = (ruleType: string) => {
    setSelectedRuleTypes((prev) => {
      const newSelected = prev.includes(ruleType) ? prev.filter((x) => x !== ruleType) : [...prev, ruleType];
      // Schedule filter change after state updates
      setTimeout(() => handleFilterChange(), 0);
      return newSelected;
    });
  };

  const handleLoadMore = () => {
    fetchRules(page + 1, true);
  };

  const handleClearFilters = () => {
    setSearchQuery('');
    setSelectedBPNames([]);
    setSelectedRuleTypes([]);
    setPage(1);
    setRules([]);
    fetchRules(1, false);
  };

  return (
    <Box className={classes.root}>
      <Typography variant="h5" gutterBottom>
        Validation Rules Library
      </Typography>

      {error && error.code === 'SCOPE_REQUIRED' && (
        <Alert severity="warning" style={{ marginBottom: '20px' }}>
          {error.message}
        </Alert>
      )}

      {error && error.code !== 'SCOPE_REQUIRED' && (
        <Alert severity="error" style={{ marginBottom: '20px' }}>
          {error.message}
        </Alert>
      )}

      {/* Search Bar */}
      <Box className={classes.searchBox}>
        <TextField
          fullWidth
          placeholder="Search by rule name or description..."
          value={searchQuery}
          onChange={handleSearchChange}
          onKeyPress={(e) => e.key === 'Enter' && handleSearchSubmit()}
          InputProps={{
            startAdornment: <SearchIcon style={{ marginRight: '8px' }} />,
          }}
          size="small"
          variant="outlined"
        />
      </Box>

      {/* Facets Panel */}
      {(bpNameFacets.length > 0 || ruleTypeFacets.length > 0) && (
        <Card className={classes.facetsPanel}>
          <CardContent>
            <Box style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <Typography variant="h6">Filters</Typography>
              {(selectedBPNames.length > 0 || selectedRuleTypes.length > 0 || searchQuery) && (
                <Button size="small" onClick={handleClearFilters}>
                  Clear All
                </Button>
              )}
            </Box>

            <Grid container spacing={3}>
              {/* Business Process Names Facet */}
              {bpNameFacets.length > 0 && (
                <Grid item xs={12} md={6}>
                  <Box className={classes.facetGroup}>
                    <Typography className={classes.facetTitle}>Business Process</Typography>
                    <FormGroup>
                      {bpNameFacets.map((facet) => (
                        <FormControlLabel
                          key={facet.value}
                          control={
                            <Checkbox
                              checked={selectedBPNames.includes(facet.value)}
                              onChange={() => handleBPNameToggle(facet.value)}
                            />
                          }
                          label={`${facet.value} (${facet.count})`}
                        />
                      ))}
                    </FormGroup>
                  </Box>
                </Grid>
              )}

              {/* Rule Type Facet */}
              {ruleTypeFacets.length > 0 && (
                <Grid item xs={12} md={6}>
                  <Box className={classes.facetGroup}>
                    <Typography className={classes.facetTitle}>Rule Type</Typography>
                    <FormGroup>
                      {ruleTypeFacets.map((facet) => (
                        <FormControlLabel
                          key={facet.value}
                          control={
                            <Checkbox
                              checked={selectedRuleTypes.includes(facet.value)}
                              onChange={() => handleRuleTypeToggle(facet.value)}
                            />
                          }
                          label={`${facet.value} (${facet.count})`}
                        />
                      ))}
                    </FormGroup>
                  </Box>
                </Grid>
              )}
            </Grid>
          </CardContent>
        </Card>
      )}

      {/* Results Info */}
      {totalCount > 0 && (
        <Typography variant="body2" style={{ marginTop: '16px', marginBottom: '8px', color: '#666' }}>
          Showing {rules.length} of {totalCount} rules
        </Typography>
      )}

      {/* Rules Table */}
      {loading && rules.length === 0 ? (
        <Box className={classes.loaderContainer}>
          <CircularProgress />
        </Box>
      ) : rules.length === 0 ? (
        <Box className={classes.emptyState}>
          <Typography variant="body1" color="textSecondary">
            No validation rules found. Try adjusting your filters.
          </Typography>
        </Box>
      ) : (
        <TableContainer component={Paper} className={classes.tableContainer}>
          <Table>
            <TableHead>
              <TableRow style={{ backgroundColor: '#f5f5f5' }}>
                <TableCell>Rule Name</TableCell>
                <TableCell>Entity</TableCell>
                <TableCell>Type</TableCell>
                <TableCell>Severity</TableCell>
                <TableCell>Status</TableCell>
                <TableCell>Updated</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {rules.map((rule) => (
                <TableRow key={rule.id} hover>
                  <TableCell>
                    <Typography variant="body2" style={{ fontWeight: 500 }}>
                      {rule.rule_name}
                    </Typography>
                    <Typography variant="caption" color="textSecondary">
                      {rule.description}
                    </Typography>
                  </TableCell>
                  <TableCell>{rule.target_entity}</TableCell>
                  <TableCell>
                    <Chip
                      label={rule.rule_type}
                      size="small"
                      variant="outlined"
                      className={classes.statusChip}
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={rule.severity}
                      size="small"
                      color={
                        rule.severity === 'error' ? 'error' : rule.severity === 'warning' ? 'warning' : undefined
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={rule.is_active ? 'Active' : 'Inactive'}
                      size="small"
                      color={rule.is_active ? 'primary' : 'default'}
                    />
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption">
                      {rule.updated_at
                        ? new Date(rule.updated_at).toLocaleDateString()
                        : rule.created_at
                        ? new Date(rule.created_at).toLocaleDateString()
                        : '—'}
                    </Typography>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}

      {/* Load More Button */}
      {hasMore && (
        <Box className={classes.loadMoreButton}>
          {loading && rules.length > 0 ? (
            <CircularProgress size={24} />
          ) : (
            <Button variant="outlined" onClick={handleLoadMore}>
              Load More Rules
            </Button>
          )}
        </Box>
      )}
    </Box>
  );
};

export default ValidationRulesList;
