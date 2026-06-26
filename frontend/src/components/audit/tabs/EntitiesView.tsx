import React, { useState } from 'react';
import {
  Box,
  TextField,
  Button,
  Stack,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  Typography,
  CircularProgress,
  Alert,
} from '@mui/material';
import {
  Search as SearchIcon,
  Info as InfoIcon,
} from '@mui/icons-material';

interface EntityAudit {
  entityId: string;
  entityType: string;
  firstSeen: string;
  lastSeen: string;
  changeCount: number;
  failureCount: number;
  complianceIssues: number;
  riskScore: number;
  status: string;
}

/**
 * EntitiesView: Entity-centric audit trail
 * 
 * Allows searching for entities (semantic terms, jobs, DAGs) and viewing
 * all audit events related to that entity across time
 */
export function EntitiesView() {
  const [entitySearch, setEntitySearch] = useState('');
  const [selectedEntity, setSelectedEntity] = useState<EntityAudit | null>(null);
  const [loading, setLoading] = useState(false);
  const [searchResults, setSearchResults] = useState<EntityAudit[]>([]);

  const handleSearch = async () => {
    if (!entitySearch.trim()) return;

    setLoading(true);
    try {
      const response = await fetch(
        `/api/audit-explorer/entities/search?q=${encodeURIComponent(
          entitySearch
        )}`,
        {
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      if (response.ok) {
        const data = await response.json();
        setSearchResults(data.entities || []);
      } else {
        setSearchResults([]);
      }
    } catch (err) {
      console.error('Search error:', err);
      setSearchResults([]);
    } finally {
      setLoading(false);
    }
  };

  const handleSelectEntity = async (entity: EntityAudit) => {
    setSelectedEntity(entity);
    setLoading(true);

    try {
      const response = await fetch(
        `/api/audit-explorer/entities/${entity.entityType}/${entity.entityId}`,
        {
          headers: {
            'Content-Type': 'application/json',
          },
        }
      );

      if (response.ok) {
        const data = await response.json();
        setSelectedEntity(data);
      }
    } catch (err) {
      console.error('Entity load error:', err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box>
      <Stack spacing={2}>
        {/* Search Bar */}
        <Paper sx={{ p: 2 }}>
          <Stack direction="row" spacing={1}>
            <TextField
              fullWidth
              placeholder="Search by semantic term, job ID, or DAG ID..."
              value={entitySearch}
              onChange={(e) => setEntitySearch(e.target.value)}
              onKeyPress={(e) => {
                if (e.key === 'Enter') {
                  handleSearch();
                }
              }}
              size="small"
            />
            <Button
              variant="contained"
              onClick={handleSearch}
              disabled={loading}
              startIcon={<SearchIcon />}
            >
              Search
            </Button>
          </Stack>
        </Paper>

        {/* Search Results */}
        {searchResults.length > 0 && !selectedEntity && (
          <TableContainer component={Paper}>
            <Table size="small">
              <TableHead>
                <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
                  <TableCell>Entity ID</TableCell>
                  <TableCell>Type</TableCell>
                  <TableCell>First Seen</TableCell>
                  <TableCell>Last Seen</TableCell>
                  <TableCell align="right">Changes</TableCell>
                  <TableCell align="right">Failures</TableCell>
                  <TableCell align="right">Compliance</TableCell>
                  <TableCell align="right">Risk</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {searchResults.map((entity) => (
                  <TableRow
                    key={`${entity.entityType}-${entity.entityId}`}
                    hover
                    onClick={() => handleSelectEntity(entity)}
                    sx={{ cursor: 'pointer' }}
                  >
                    <TableCell>
                      <Typography variant="body2" sx={{ fontWeight: 500 }}>
                        {entity.entityId}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Chip label={entity.entityType} size="small" />
                    </TableCell>
                    <TableCell>
                      <Typography variant="caption">
                        {new Date(entity.firstSeen).toLocaleDateString()}
                      </Typography>
                    </TableCell>
                    <TableCell>
                      <Typography variant="caption">
                        {new Date(entity.lastSeen).toLocaleDateString()}
                      </Typography>
                    </TableCell>
                    <TableCell align="right">
                      <Chip label={entity.changeCount} size="small" />
                    </TableCell>
                    <TableCell align="right">
                      <Chip
                        label={entity.failureCount}
                        size="small"
                        color={entity.failureCount > 0 ? 'error' : 'default'}
                      />
                    </TableCell>
                    <TableCell align="right">
                      <Chip
                        label={entity.complianceIssues}
                        size="small"
                        color={
                          entity.complianceIssues > 0 ? 'warning' : 'default'
                        }
                      />
                    </TableCell>
                    <TableCell align="right">
                      <Typography
                        variant="body2"
                        sx={{
                          color:
                            entity.riskScore > 0.7
                              ? '#d32f2f'
                              : entity.riskScore > 0.4
                              ? '#f57c00'
                              : '#388e3c',
                        }}
                      >
                        {(entity.riskScore * 100).toFixed(0)}%
                      </Typography>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TableContainer>
        )}

        {/* Entity Details */}
        {selectedEntity && (
          <Box>
            <Stack direction="row" spacing={2} alignItems="center" sx={{ mb: 2 }}>
              <Typography variant="h6">
                {selectedEntity.entityId}
              </Typography>
              <Chip label={selectedEntity.entityType} />
              <Button
                size="small"
                variant="outlined"
                onClick={() => {
                  setSelectedEntity(null);
                  setSearchResults([]);
                }}
              >
                Back to Results
              </Button>
            </Stack>

            <Stack
              direction="row"
              spacing={2}
              sx={{ mb: 2, flexWrap: 'wrap' }}
            >
              <Paper sx={{ p: 2, flex: 1, minWidth: 150 }}>
                <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                  Total Changes
                </Typography>
                <Typography variant="h6">
                  {selectedEntity.changeCount}
                </Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1, minWidth: 150 }}>
                <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                  Failures
                </Typography>
                <Typography
                  variant="h6"
                  sx={{
                    color:
                      selectedEntity.failureCount > 0
                        ? 'error.main'
                        : 'text.primary',
                  }}
                >
                  {selectedEntity.failureCount}
                </Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1, minWidth: 150 }}>
                <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                  Compliance Issues
                </Typography>
                <Typography
                  variant="h6"
                  sx={{
                    color:
                      selectedEntity.complianceIssues > 0
                        ? 'warning.main'
                        : 'text.primary',
                  }}
                >
                  {selectedEntity.complianceIssues}
                </Typography>
              </Paper>
              <Paper sx={{ p: 2, flex: 1, minWidth: 150 }}>
                <Typography variant="caption" sx={{ color: 'text.secondary' }}>
                  Risk Score
                </Typography>
                <Typography
                  variant="h6"
                  sx={{
                    color:
                      selectedEntity.riskScore > 0.7
                        ? 'error.main'
                        : selectedEntity.riskScore > 0.4
                        ? 'warning.main'
                        : 'success.main',
                  }}
                >
                  {(selectedEntity.riskScore * 100).toFixed(0)}%
                </Typography>
              </Paper>
            </Stack>

            {loading && (
              <Box sx={{ display: 'flex', justifyContent: 'center', p: 4 }}>
                <CircularProgress />
              </Box>
            )}

            {selectedEntity && (
              <Alert icon={<InfoIcon />}>
                Entity audit trail showing all changes, failures, and compliance
                issues across the selected time range.
              </Alert>
            )}
          </Box>
        )}

        {!loading && entitySearch && searchResults.length === 0 && (
          <Alert severity="info">
            No entities found matching "{entitySearch}"
          </Alert>
        )}
      </Stack>
    </Box>
  );
}

export default EntitiesView;
