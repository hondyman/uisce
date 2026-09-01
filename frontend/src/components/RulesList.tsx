/**
 * RulesList Component
 * Displays a filterable and searchable list of validation rules
 */

import React, { useMemo, useCallback, useState } from 'react';
import { useConfirm } from '../components/ConfirmProvider';
import { useNotification } from '../hooks/useNotification';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import RuleCard from './RuleCard';
import { useTheme } from '@mui/material/styles';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import Button from '@mui/material/Button';
import CircularProgress from '@mui/material/CircularProgress';

interface RulesListProps {
  rules: any[];
  loading: boolean;
  onEdit: (rule: any) => void;
  onDelete: (ruleId: string) => void;
  onCreateNew: () => void;
  filterType?: string;
  searchTerm?: string;
  sortBy?: 'name' | 'type' | 'severity' | 'order';
}

/**
 * RulesList displays validation rules with optional filtering/sorting
 */
const RulesList: React.FC<RulesListProps> = ({
  rules,
  loading,
  onEdit,
  onDelete,
  onCreateNew,
  filterType,
  searchTerm = '',
  sortBy = 'name',
}) => {
  const theme = useTheme();
  const [deletingRuleId, setDeletingRuleId] = useState<string | null>(null);
  const isDark = theme.palette.mode === 'dark';

  // Memoize filtered and sorted rules
  const displayedRules = useMemo(() => {
    let filtered = [...rules];

    // Apply search filter
    if (searchTerm) {
      const lowerSearch = searchTerm.toLowerCase();
      filtered = filtered.filter(
        (rule) =>
          rule.name.toLowerCase().includes(lowerSearch) ||
          rule.description?.toLowerCase().includes(lowerSearch) ||
          rule.ruleType.toLowerCase().includes(lowerSearch)
      );
    }

    // Apply type filter
    if (filterType && filterType !== 'ALL') {
      filtered = filtered.filter((rule) => rule.ruleType === filterType);
    }

    // Apply sorting
    filtered.sort((a, b) => {
      switch (sortBy) {
        case 'type':
          return a.ruleType.localeCompare(b.ruleType);
        case 'severity':
          {
            const severityOrder: Record<string, number> = { BLOCK: 0, WARNING: 1, INFO: 2 };
            return (severityOrder[a.severity] || 99) - (severityOrder[b.severity] || 99);
          }
        case 'order':
          return (a.evaluationOrder || 0) - (b.evaluationOrder || 0);
        case 'name':
        default:
          return a.name.localeCompare(b.name);
      }
    });

    return filtered;
  }, [rules, searchTerm, filterType, sortBy]);

  const handleDelete = useCallback(
    async (ruleId: string) => {
      const confirm = useConfirm();
      const notification = useNotification();
      if (!(await confirm({ title: 'Delete rule', description: 'Are you sure you want to delete this rule?' }))) return;
      setDeletingRuleId(ruleId);
      try {
        await onDelete(ruleId);
        notification.success('Rule deleted');
      } finally {
        setDeletingRuleId(null);
      }
    },
    [onDelete]
  );

  if (loading) {
    return (
      <Box sx={{ textAlign: 'center', py: 6 }}>
        <CircularProgress size={32} sx={{ display: 'inline-block' }} />
        <Typography sx={{ color: isDark ? 'grey.400' : 'grey.600', mt: 2 }}>
          Loading rules...
        </Typography>
      </Box>
    );
  }

  if (rules.length === 0) {
    return (
      <Box
        sx={{
          textAlign: 'center',
          py: 6,
          border: '2px dashed',
          borderColor: isDark ? 'grey.600' : 'grey.300',
          borderRadius: 2,
        }}
      >
        <ErrorOutlineIcon sx={{ fontSize: 48, mx: 'auto', mb: 2, color: 'grey.500' }} />
        <Typography sx={{ color: isDark ? 'grey.400' : 'grey.600', mb: 2 }}>
          No validation rules yet
        </Typography>
        <Button
          variant="contained"
          onClick={onCreateNew}
          sx={{
            backgroundColor: '#2563eb',
            '&:hover': { backgroundColor: '#1d4ed8' },
            color: 'white',
            borderRadius: '8px',
          }}
        >
          Create First Rule
        </Button>
      </Box>
    );
  }

  if (displayedRules.length === 0) {
    return (
      <Box
        sx={{
          textAlign: 'center',
          py: 6,
          border: '2px dashed',
          borderColor: isDark ? 'grey.600' : 'grey.300',
          borderRadius: 2,
        }}
      >
        <ErrorOutlineIcon sx={{ fontSize: 48, mx: 'auto', mb: 2, color: 'grey.500' }} />
        <Typography sx={{ color: isDark ? 'grey.400' : 'grey.600', mb: 2 }}>
          No rules match your filters
        </Typography>
        <Typography variant="body2" sx={{ color: isDark ? 'grey.500' : 'grey.500' }}>
          {searchTerm && `Search: "${searchTerm}"`}
          {filterType && filterType !== 'ALL' && ` • Type: ${filterType}`}
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Typography variant="body2" sx={{ color: isDark ? 'grey.400' : 'grey.600' }}>
        Showing {displayedRules.length} of {rules.length} rules
      </Typography>

      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
        {displayedRules.map((rule) => (
          <RuleCard
            key={rule.id}
            rule={rule}
            onEdit={onEdit}
            onDelete={handleDelete}
            isDeleting={deletingRuleId === rule.id}
          />
        ))}
      </Box>
    </Box>
  );
};

export default RulesList;
