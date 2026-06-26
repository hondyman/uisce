import React, { useState, useMemo } from 'react';
import {
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Alert,
  Chip,
  Typography,
  Divider as _Divider,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
} from '@mui/material';
import ContentCopyIcon from '@mui/icons-material/ContentCopy';
import WarningIcon from '@mui/icons-material/Warning';
import CheckCircleIcon from '@mui/icons-material/CheckCircle';
import ErrorIcon from '@mui/icons-material/Error';
import InfoIcon from '@mui/icons-material/Info';

/**
 * Rule for cloning and comparison
 */
export interface RuleForCloning {
  id: string;
  name: string;
  description: string;
  condition: string | object;
  severity: 'error' | 'warning' | 'info';
  targetEntity: string;
  fieldName: string;
  created_at?: string;
}

/**
 * Conflict detection result
 */
export interface RuleConflict {
  severity: 'error' | 'warning' | 'info';
  message: string;
  conflictingRuleId?: string;
  conflictingRuleName?: string;
  suggestion?: string;
}

interface RuleCloneAndConflictProps {
  onRuleCloned: (baseRule: Partial<RuleForCloning>) => void;
  existingRules: RuleForCloning[];
  newRuleData?: {
    condition?: string;
    targetEntity?: string;
    fieldName?: string;
  };
}

/**
 * Rule Clone & Conflict Detection Component
 * 
 * Features:
 * - Browse and clone existing rules
 * - Detect similar/conflicting rules
 * - Suggest improvements
 * - Performance impact warnings
 */
const RuleCloneAndConflict: React.FC<RuleCloneAndConflictProps> = ({
  onRuleCloned,
  existingRules,
  newRuleData,
}) => {
  const [cloneDialogOpen, setCloneDialogOpen] = useState(false);
  const [conflictDialogOpen, setConflictDialogOpen] = useState(false);
  const [selectedRuleToClone, setSelectedRuleToClone] = useState<RuleForCloning | null>(null);
  const [cloneName, setCloneName] = useState('');
  const [_conflicts, setConflicts] = useState<RuleConflict[]>([]);

  // Helper to safely get string representation of condition
  const getConditionString = (condition: string | object | undefined): string => {
    if (!condition) return '';
    if (typeof condition === 'string') return condition;
    try {
      return JSON.stringify(condition);
    } catch {
      return 'Invalid Condition';
    }
  };

  // Simple string similarity using Levenshtein distance
  const calculateSimilarity = (str1: string, str2: string): number => {
    const len1 = str1.length;
    const len2 = str2.length;
    const maxLen = Math.max(len1, len2);
    if (maxLen === 0) return 1;

    let distance = 0;
    const minLen = Math.min(len1, len2);
    for (let i = 0; i < minLen; i++) {
      if (str1[i] !== str2[i]) distance++;
    }
    distance += Math.abs(len1 - len2);

    return 1 - distance / maxLen;
  };

  // Detect conflicts and similarities
  const detectedConflicts = useMemo(() => {
    if (!newRuleData) return [];

    const conflicts: RuleConflict[] = [];
    const { condition, targetEntity, fieldName } = newRuleData;

    existingRules.forEach(rule => {
      const conditionStr = getConditionString(condition);
      const ruleConditionStr = getConditionString(rule.condition);

      // Same condition on same field
      if (ruleConditionStr === conditionStr && rule.fieldName === fieldName && rule.targetEntity === targetEntity) {
        conflicts.push({
          severity: 'error',
          message: `Exact duplicate rule already exists`,
          conflictingRuleId: rule.id,
          conflictingRuleName: rule.name,
          suggestion: 'Consider cloning the existing rule instead or modifying the condition',
        });
      }

      // Similar condition on same field
      if (rule.fieldName === fieldName && rule.targetEntity === targetEntity && ruleConditionStr !== conditionStr) {
        const similarity = calculateSimilarity(conditionStr || '', ruleConditionStr);
        if (similarity > 0.7) {
          conflicts.push({
            severity: 'warning',
            message: `Similar rule exists (${Math.round(similarity * 100)}% match)`,
            conflictingRuleId: rule.id,
            conflictingRuleName: rule.name,
            suggestion: 'Review the existing rule to avoid redundant validations',
          });
        }
      }

      // Same severity on same field might cause performance issues
      if (rule.fieldName === fieldName && rule.severity === 'error') {
        conflicts.push({
          severity: 'info',
          message: `Already validating this field with error severity`,
          conflictingRuleId: rule.id,
          conflictingRuleName: rule.name,
          suggestion: 'Consider using warning severity or different validation logic',
        });
      }
    });

    // Performance warnings for complex conditions
    const conditionStr = getConditionString(condition);
    if (conditionStr && conditionStr.length > 500) {
      conflicts.push({
        severity: 'warning',
        message: 'Complex condition detected - may impact performance',
        suggestion: 'Consider breaking this into simpler, more focused rules',
      });
    }

    // High number of rules on same entity
    const rulesOnEntity = existingRules.filter(r => r.targetEntity === targetEntity).length;
    if (rulesOnEntity > 10) {
      conflicts.push({
        severity: 'info',
        message: `${rulesOnEntity} rules already exist for ${targetEntity}`,
        suggestion: 'Ensure this rule adds value and is not redundant',
      });
    }

    return conflicts;
  }, [newRuleData, existingRules]);

  // Handle cloning a rule
  const handleCloneRule = (rule: RuleForCloning) => {
    setSelectedRuleToClone(rule);
    setCloneName(`${rule.name} (Copy)`);
  };

  // Confirm clone
  const _handleConfirmClone = () => {
    if (selectedRuleToClone) {
      const clonedRule: Partial<RuleForCloning> = {
        ...selectedRuleToClone,
        name: cloneName,
        id: undefined, // Reset ID for new rule
      };
      onRuleCloned(clonedRule);
      setCloneDialogOpen(false);
      setSelectedRuleToClone(null);
    }
  };

  // Check conflicts
  const handleCheckConflicts = () => {
    setConflicts(detectedConflicts);
    setConflictDialogOpen(true);
  };

  // Get severity icon
  const getSeverityIcon = (severity: string) => {
    switch (severity) {
      case 'error':
        return <ErrorIcon sx={{ color: '#d32f2f' }} />;
      case 'warning':
        return <WarningIcon sx={{ color: '#f57c00' }} />;
      case 'info':
      default:
        return <InfoIcon sx={{ color: '#1976d2' }} />;
    }
  };

  // Get severity color
  const getSeverityColor = (severity: string): 'error' | 'warning' | 'info' | 'success' => {
    switch (severity) {
      case 'error':
        return 'error';
      case 'warning':
        return 'warning';
      case 'info':
      default:
        return 'info';
    }
  };

  return (
    <Box>
      <Grid container spacing={2}>
        {/* Clone Existing Rule */}
        <Grid item xs={12} sm={6}>
          <Card sx={{ height: '100%' }}>
            <CardHeader title="📋 Clone Existing Rule" />
            <CardContent>
              <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                Reuse patterns from {existingRules.length} existing rules
              </Typography>
              <Button
                variant="outlined"
                startIcon={<ContentCopyIcon />}
                onClick={() => setCloneDialogOpen(true)}
                fullWidth
              >
                Select Rule to Clone
              </Button>
            </CardContent>
          </Card>
        </Grid>

        {/* Check for Conflicts */}
        <Grid item xs={12} sm={6}>
          <Card sx={{ height: '100%' }}>
            <CardHeader title="⚠️ Conflict Detection" />
            <CardContent>
              <Typography variant="body2" color="textSecondary" sx={{ mb: 2 }}>
                Check for similar or conflicting rules
              </Typography>
              <Button
                variant="outlined"
                startIcon={<WarningIcon />}
                onClick={handleCheckConflicts}
                fullWidth
              >
                Analyze Conflicts
              </Button>
              {detectedConflicts.length > 0 && (
                <Chip
                  icon={<WarningIcon />}
                  label={`${detectedConflicts.length} issues found`}
                  color="warning"
                  sx={{ mt: 1 }}
                />
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Clone Dialog */}
      <Dialog open={cloneDialogOpen} onClose={() => setCloneDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Select Rule to Clone</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          <List>
            {existingRules.map(rule => {
               const condStr = getConditionString(rule.condition);
               return (
              <ListItem
                key={rule.id}
                secondaryAction={
                  <Chip label={rule.severity} size="small" color={getSeverityColor(rule.severity)} />
                }
                sx={{ mb: 1, border: 1, borderColor: 'divider', borderRadius: 1 }}
                disablePadding
              >
                <ListItemButton onClick={() => handleCloneRule(rule)}>
                  <ListItemIcon>
                    <ContentCopyIcon />
                  </ListItemIcon>
                  <ListItemText
                    primary={rule.name}
                    secondary={
                      <>
                        <Typography component="span" variant="body2" color="textSecondary">
                          {rule.targetEntity}.{rule.fieldName}
                        </Typography>
                        <br />
                        <Typography component="span" variant="caption">
                          {condStr.substring(0, 60)}
                          {condStr.length > 60 ? '...' : ''}
                        </Typography>
                      </>
                    }
                  />
                </ListItemButton>
              </ListItem>
            );
            })}
          </List>

          {existingRules.length === 0 && (
            <Alert severity="info">No existing rules to clone</Alert>
          )}
        </DialogContent>
      </Dialog>

      {/* Conflict Detection Dialog */}
      <Dialog open={conflictDialogOpen} onClose={() => setConflictDialogOpen(false)} maxWidth="md" fullWidth>
        <DialogTitle>Conflict Detection Analysis</DialogTitle>
        <DialogContent sx={{ pt: 2 }}>
          {detectedConflicts.length === 0 ? (
            <Alert severity="success" icon={<CheckCircleIcon />}>
              ✅ No conflicts detected. Rule looks good to deploy!
            </Alert>
          ) : (
            <Box>
              <Typography variant="body2" sx={{ mb: 2 }}>
                Found {detectedConflicts.length} potential issues:
              </Typography>

              {/* Summary Bar */}
              <Box sx={{ mb: 3 }}>
                <LinearProgress
                  variant="determinate"
                  value={
                    (detectedConflicts.filter(c => c.severity !== 'info').length /
                      Math.max(1, detectedConflicts.length)) *
                    100
                  }
                  sx={{
                    mb: 1,
                    height: 6,
                    borderRadius: 3,
                    backgroundColor: '#e0e0e0',
                    '& .MuiLinearProgress-bar': {
                      borderRadius: 3,
                      backgroundColor: detectedConflicts.some(c => c.severity === 'error')
                        ? '#d32f2f'
                        : detectedConflicts.some(c => c.severity === 'warning')
                          ? '#f57c00'
                          : '#1976d2',
                    },
                  }}
                />
                <Box sx={{ display: 'flex', gap: 1 }}>
                  {detectedConflicts.filter(c => c.severity === 'error').length > 0 && (
                    <Chip
                      icon={<ErrorIcon />}
                      label={`${detectedConflicts.filter(c => c.severity === 'error').length} errors`}
                      color="error"
                      size="small"
                    />
                  )}
                  {detectedConflicts.filter(c => c.severity === 'warning').length > 0 && (
                    <Chip
                      icon={<WarningIcon />}
                      label={`${detectedConflicts.filter(c => c.severity === 'warning').length} warnings`}
                      color="warning"
                      size="small"
                    />
                  )}
                  {detectedConflicts.filter(c => c.severity === 'info').length > 0 && (
                    <Chip
                      icon={<InfoIcon />}
                      label={`${detectedConflicts.filter(c => c.severity === 'info').length} info`}
                      color="info"
                      size="small"
                    />
                  )}
                </Box>
              </Box>

              {/* Conflicts Table */}
              <TableContainer>
                <Table size="small">
                  <TableHead>
                    <TableRow sx={{ backgroundColor: '#f5f5f5' }}>
                      <TableCell>Type</TableCell>
                      <TableCell>Issue</TableCell>
                      <TableCell>Suggestion</TableCell>
                      <TableCell>Related Rule</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {detectedConflicts.map((conflict, idx) => (
                      <TableRow key={idx}>
                        <TableCell align="center">
                          <Tooltip title={conflict.severity}>
                            {getSeverityIcon(conflict.severity)}
                          </Tooltip>
                        </TableCell>
                        <TableCell>
                          <Typography variant="body2">{conflict.message}</Typography>
                        </TableCell>
                        <TableCell>
                          <Typography variant="caption" color="textSecondary">
                            {conflict.suggestion}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          {conflict.conflictingRuleName && (
                            <Chip
                              label={conflict.conflictingRuleName}
                              size="small"
                              variant="outlined"
                            />
                          )}
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TableContainer>

              {detectedConflicts.some(c => c.severity === 'error') && (
                <Alert severity="error" sx={{ mt: 2 }}>
                  ⚠️ Critical issues found. Please resolve before deploying.
                </Alert>
              )}
              {detectedConflicts.some(c => c.severity === 'warning') && (
                !detectedConflicts.some(c => c.severity === 'error') && (
                  <Alert severity="warning" sx={{ mt: 2 }}>
                    ⚠️ Warnings detected. Review before deployment.
                  </Alert>
                )
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConflictDialogOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
};

export default RuleCloneAndConflict;
