import React from 'react';
import {
  Box,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Chip,
  Stack,
  Button,
  InputAdornment,
} from '@mui/material';
import {
  Search as SearchIcon,
  FilterList as FilterIcon,
  Refresh as RefreshIcon,
} from '@mui/icons-material';

interface FilterBarProps {
  timeRange: string;
  onTimeRangeChange: (range: string) => void;
  artifactTypes: string[];
  onArtifactTypesChange: (types: string[]) => void;
  statuses: string[];
  onStatusesChange: (statuses: string[]) => void;
  riskLevels: string[];
  onRiskLevelsChange: (levels: string[]) => void;
  searchTerm: string;
  onSearchTermChange: (term: string) => void;
  userRole: string;
  onRefresh?: () => void;
}

/**
 * FilterBar: Unified filter component for Audit Explorer
 * 
 * Provides:
 * - Time range selector (24h, 7d, 30d, custom)
 * - Multi-select artifact types
 * - Multi-select statuses
 * - Multi-select risk levels
 * - Search/filter input
 * - Role-aware filter visibility
 */
export function FilterBar({
  timeRange,
  onTimeRangeChange,
  artifactTypes,
  onArtifactTypesChange,
  statuses,
  onStatusesChange,
  riskLevels,
  onRiskLevelsChange,
  searchTerm,
  onSearchTermChange,
  userRole,
  onRefresh,
}: FilterBarProps) {
  const artifactTypeOptions = [
    { value: 'job_run', label: 'Job Runs' },
    { value: 'dag_run', label: 'DAG Runs' },
    { value: 'changeset', label: 'ChangeSet' },
    { value: 'semantic_snapshot', label: 'Semantic Snapshot' },
    { value: 'compliance_violation', label: 'Compliance Violation' },
    { value: 'orchestration_event', label: 'Orchestration Event' },
  ];

  const statusOptions = [
    { value: 'success', label: 'Success' },
    { value: 'failed', label: 'Failed' },
    { value: 'pending', label: 'Pending' },
    { value: 'approved', label: 'Approved' },
    { value: 'rejected', label: 'Rejected' },
  ];

  const riskLevelOptions = [
    { value: 'low', label: 'Low', color: 'success' as const },
    { value: 'medium', label: 'Medium', color: 'warning' as const },
    { value: 'high', label: 'High', color: 'error' as const },
  ];

  const showComplianceFilters = ['global_admin', 'global_ops', 'tenant_admin'].includes(userRole);

  return (
    <Stack spacing={2}>
      {/* Primary filters row */}
      <Stack direction="row" spacing={2} alignItems="flex-end" flexWrap="wrap">
        {/* Search Input */}
        <TextField
          placeholder="Search events..."
          variant="outlined"
          size="small"
          value={searchTerm}
          onChange={(e) => onSearchTermChange(e.target.value)}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon fontSize="small" />
              </InputAdornment>
            ),
          }}
          sx={{ minWidth: 250 }}
        />

        {/* Time Range */}
        <FormControl size="small" sx={{ minWidth: 150 }}>
          <InputLabel>Time Range</InputLabel>
          <Select
            value={timeRange}
            onChange={(e) => onTimeRangeChange(e.target.value)}
            label="Time Range"
          >
            <MenuItem value="24h">Last 24 Hours</MenuItem>
            <MenuItem value="7d">Last 7 Days</MenuItem>
            <MenuItem value="30d">Last 30 Days</MenuItem>
            <MenuItem value="custom">Custom Range</MenuItem>
          </Select>
        </FormControl>

        {/* Refresh Button */}
        <Button
          variant="outlined"
          size="small"
          startIcon={<RefreshIcon />}
          onClick={onRefresh}
        >
          Refresh
        </Button>
      </Stack>

      {/* Secondary filters row */}
      <Stack direction="row" spacing={2} flexWrap="wrap">
        {/* Artifact Types */}
        <FormControl size="small" sx={{ minWidth: 200 }}>
          <InputLabel>Artifact Types</InputLabel>
          <Select
            multiple
            value={artifactTypes}
            onChange={(e) =>
              onArtifactTypesChange(
                typeof e.target.value === 'string'
                  ? e.target.value.split(',')
                  : e.target.value
              )
            }
            label="Artifact Types"
            renderValue={(selected) => (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {selected.map((value) => (
                  <Chip
                    key={value}
                    label={
                      artifactTypeOptions.find((o) => o.value === value)?.label ||
                      value
                    }
                    size="small"
                  />
                ))}
              </Box>
            )}
          >
            {artifactTypeOptions.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        {/* Statuses */}
        <FormControl size="small" sx={{ minWidth: 200 }}>
          <InputLabel>Status</InputLabel>
          <Select
            multiple
            value={statuses}
            onChange={(e) =>
              onStatusesChange(
                typeof e.target.value === 'string'
                  ? e.target.value.split(',')
                  : e.target.value
              )
            }
            label="Status"
            renderValue={(selected) => (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {selected.map((value) => (
                  <Chip
                    key={value}
                    label={statusOptions.find((o) => o.value === value)?.label || value}
                    size="small"
                  />
                ))}
              </Box>
            )}
          >
            {statusOptions.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        {/* Risk Levels */}
        <FormControl size="small" sx={{ minWidth: 200 }}>
          <InputLabel>Risk Level</InputLabel>
          <Select
            multiple
            value={riskLevels}
            onChange={(e) =>
              onRiskLevelsChange(
                typeof e.target.value === 'string'
                  ? e.target.value.split(',')
                  : e.target.value
              )
            }
            label="Risk Level"
            renderValue={(selected) => (
              <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                {selected.map((value) => (
                  <Chip
                    key={value}
                    label={riskLevelOptions.find((o) => o.value === value)?.label || value}
                    size="small"
                    color={riskLevelOptions.find((o) => o.value === value)?.color}
                  />
                ))}
              </Box>
            )}
          >
            {riskLevelOptions.map((option) => (
              <MenuItem key={option.value} value={option.value}>
                {option.label}
              </MenuItem>
            ))}
          </Select>
        </FormControl>

        {/* Compliance filter (role-aware) */}
        {showComplianceFilters && (
          <FormControl size="small" sx={{ minWidth: 200 }}>
            <InputLabel>Violation Type</InputLabel>
            <Select label="Violation Type" defaultValue="">
              <MenuItem value="">All Types</MenuItem>
              <MenuItem value="pii_exposure">PII Exposure</MenuItem>
              <MenuItem value="data_classification">Data Classification</MenuItem>
              <MenuItem value="access_control">Access Control</MenuItem>
              <MenuItem value="retention">Retention Policy</MenuItem>
              <MenuItem value="audit_trail">Audit Trail</MenuItem>
            </Select>
          </FormControl>
        )}
      </Stack>

      {/* Active Filters Summary */}
      {(artifactTypes.length > 0 ||
        statuses.length > 0 ||
        riskLevels.length > 0) && (
        <Box sx={{ pt: 1, borderTop: 1, borderColor: 'divider' }}>
          <Stack direction="row" spacing={1} alignItems="center">
            <FilterIcon fontSize="small" sx={{ color: 'text.secondary' }} />
            <Typography variant="body2" sx={{ color: 'text.secondary' }}>
              Active filters: {artifactTypes.length + statuses.length + riskLevels.length}
            </Typography>
            <Button
              size="small"
              variant="text"
              onClick={() => {
                onArtifactTypesChange([]);
                onStatusesChange([]);
                onRiskLevelsChange([]);
              }}
            >
              Clear All
            </Button>
          </Stack>
        </Box>
      )}
    </Stack>
  );
}

import { Typography } from '@mui/material';

export default FilterBar;
