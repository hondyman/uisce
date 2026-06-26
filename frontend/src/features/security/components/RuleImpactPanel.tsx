import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Stack,
  Typography,
  CircularProgress,
  Alert,
  Chip,
  Divider,
  List,
  ListItem,
  ListItemText,
  ListItemIcon,
  Grid,
  Card,
  CardContent,
} from '@mui/material';
import {
  People as PeopleIcon,
  Api as ApiIcon,
  Assessment as AssessmentIcon,
  Psychology as PsychologyIcon,
  Warning as WarningIcon,
  CheckCircle as CheckIcon,
} from '@mui/icons-material';
import { AccessRuleInput } from '../../../api/accessRules';

interface ImpactData {
  affectedUsers: number;
  affectedApis: string[];
  affectedReports: string[];
  affectedAiQueries: number;
  dataRowsAffected: number;
  fieldsRestricted: number;
  estimatedPerformanceImpact: 'low' | 'medium' | 'high';
  warnings: string[];
}

interface RuleImpactPanelProps {
  rule: AccessRuleInput;
}

export const RuleImpactPanel: React.FC<RuleImpactPanelProps> = ({ rule }) => {
  const [impact, setImpact] = useState<ImpactData | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const analyzeImpact = async () => {
      if (!rule.businessObjectId || !rule.groupDn) {
        setImpact(null);
        return;
      }

      setLoading(true);
      setError(null);

      try {
        // TODO: Replace with actual API call
        const response = await fetch('/api/access-rules/analyze-impact', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(rule),
        });

        if (response.ok) {
          const data = await response.json();
          setImpact(data);
        } else {
          // Mock data for demonstration
          setImpact({
            affectedUsers: 45,
            affectedApis: [
              '/api/portfolios',
              '/api/portfolios/{id}',
              '/api/portfolios/{id}/holdings',
            ],
            affectedReports: [
              'Portfolio Performance Report',
              'Holdings Summary',
              'Asset Allocation Dashboard',
            ],
            affectedAiQueries: 127,
            dataRowsAffected: rule.rowFilterDsl ? 15000 : 50000,
            fieldsRestricted: rule.columnMasks?.length || 0,
            estimatedPerformanceImpact: rule.rowFilterDsl ? 'medium' : 'low',
            warnings: rule.rowFilterDsl
              ? ['Complex row filter may impact query performance']
              : [],
          });
        }
      } catch (err) {
        setError('Failed to analyze impact');
        console.error(err);
      } finally {
        setLoading(false);
      }
    };

    void analyzeImpact();
  }, [rule]);

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return <Alert severity="error">{error}</Alert>;
  }

  if (!impact) {
    return (
      <Alert severity="info">
        Configure the rule to see impact analysis
      </Alert>
    );
  }

  const getPerformanceColor = (level: string) => {
    switch (level) {
      case 'low':
        return 'success';
      case 'medium':
        return 'warning';
      case 'high':
        return 'error';
      default:
        return 'default';
    }
  };

  return (
    <Stack spacing={3}>
      {/* Warnings */}
      {impact.warnings.length > 0 && (
        <Alert severity="warning" icon={<WarningIcon />}>
          <Typography variant="body2" sx={{ fontWeight: 600, mb: 1 }}>
            Potential Issues
          </Typography>
          <List dense>
            {impact.warnings.map((warning, index) => (
              <ListItem key={index}>
                <ListItemText primary={warning} />
              </ListItem>
            ))}
          </List>
        </Alert>
      )}

      {/* Impact Metrics */}
      <Grid container spacing={2}>
        <Grid item xs={12} md={6}>
          <Card elevation={2}>
            <CardContent>
              <Stack direction="row" spacing={2} alignItems="center">
                <PeopleIcon sx={{ fontSize: 40, color: 'primary.main' }} />
                <Box>
                  <Typography variant="h4" sx={{ fontWeight: 700 }}>
                    {impact.affectedUsers}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Affected Users
                  </Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card elevation={2}>
            <CardContent>
              <Stack direction="row" spacing={2} alignItems="center">
                <AssessmentIcon sx={{ fontSize: 40, color: 'info.main' }} />
                <Box>
                  <Typography variant="h4" sx={{ fontWeight: 700 }}>
                    {impact.dataRowsAffected.toLocaleString()}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    Data Rows Affected
                  </Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card elevation={2}>
            <CardContent>
              <Stack direction="row" spacing={2} alignItems="center">
                <ApiIcon sx={{ fontSize: 40, color: 'secondary.main' }} />
                <Box>
                  <Typography variant="h4" sx={{ fontWeight: 700 }}>
                    {impact.affectedApis.length}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    API Endpoints
                  </Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Grid>

        <Grid item xs={12} md={6}>
          <Card elevation={2}>
            <CardContent>
              <Stack direction="row" spacing={2} alignItems="center">
                <PsychologyIcon sx={{ fontSize: 40, color: 'success.main' }} />
                <Box>
                  <Typography variant="h4" sx={{ fontWeight: 700 }}>
                    {impact.affectedAiQueries}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    AI Queries/Month
                  </Typography>
                </Box>
              </Stack>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Performance Impact */}
      <Paper elevation={1} sx={{ p: 2 }}>
        <Stack direction="row" justifyContent="space-between" alignItems="center">
          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>
            Estimated Performance Impact
          </Typography>
          <Chip
            label={impact.estimatedPerformanceImpact.toUpperCase()}
            color={getPerformanceColor(impact.estimatedPerformanceImpact) as any}
            size="small"
          />
        </Stack>
      </Paper>

      {/* Affected APIs */}
      {rule.scope?.appliesToApis && impact.affectedApis.length > 0 && (
        <Paper elevation={1} sx={{ p: 2 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2 }}>
            Affected API Endpoints
          </Typography>
          <Stack spacing={1}>
            {impact.affectedApis.map((api, index) => (
              <Chip
                key={index}
                label={api}
                size="small"
                variant="outlined"
                icon={<ApiIcon />}
              />
            ))}
          </Stack>
        </Paper>
      )}

      {/* Affected Reports */}
      {rule.scope?.appliesToBi && impact.affectedReports.length > 0 && (
        <Paper elevation={1} sx={{ p: 2 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 2 }}>
            Affected Reports & Dashboards
          </Typography>
          <List dense>
            {impact.affectedReports.map((report, index) => (
              <ListItem key={index}>
                <ListItemIcon>
                  <AssessmentIcon color="primary" />
                </ListItemIcon>
                <ListItemText primary={report} />
              </ListItem>
            ))}
          </List>
        </Paper>
      )}

      {/* Field Restrictions */}
      {impact.fieldsRestricted > 0 && (
        <Alert severity="info" icon={<CheckIcon />}>
          <Typography variant="body2">
            <strong>{impact.fieldsRestricted}</strong> field(s) will be masked or hidden for this group
          </Typography>
        </Alert>
      )}
    </Stack>
  );
};

export default RuleImpactPanel;
